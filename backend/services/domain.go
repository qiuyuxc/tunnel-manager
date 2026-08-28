package services

import (
	"fmt"
	"strings"

	"tunnel-manager/models"
	"tunnel-manager/store"
)

const (
	BindingModeSimple    = "simple"
	BindingModePreferred = "preferred"
)

type DomainService struct {
	cf    *CloudflareClient
	store *store.Store
}

func NewDomainService(cf *CloudflareClient, st *store.Store) *DomainService {
	return &DomainService{cf: cf, store: st}
}

// ForUser returns a domain service bound to one account's connection.
func (s *DomainService) ForUser(userID string) *DomainService {
	clone := *s
	clone.cf = s.cf.ForUser(userID)
	return &clone
}

func NormalizeBindingMode(mode string) (string, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		return BindingModePreferred, nil
	}
	if mode != BindingModeSimple && mode != BindingModePreferred {
		return "", fmt.Errorf("unsupported binding mode: %s", mode)
	}
	return mode, nil
}

func (d *DomainService) BindDomain(mainDomain, auxDomain string) (string, error) {
	return d.BindDomainWithPreferredCNAME(mainDomain, auxDomain, "")
}
func (d *DomainService) BindDomainWithPreferredCNAME(mainDomain, auxDomain, preferredCNAME string) (string, error) {
	cfg := d.store.GetConfig()
	if cfg.ServiceURL == "" {
		return "", fmt.Errorf("service_url 未配置，请先在面板中设置")
	}
	_, preferred, err := d.BindDomainWithMode(BindingModePreferred, mainDomain, auxDomain, cfg.ServiceURL, preferredCNAME)
	return preferred, err
}
func (d *DomainService) BindDomainWithService(mainDomain, auxDomain, serviceURL, preferredCNAME string) (string, error) {
	_, preferred, err := d.BindDomainWithMode(BindingModePreferred, mainDomain, auxDomain, serviceURL, preferredCNAME)
	return preferred, err
}
func (d *DomainService) BindDomainWithConfiguredService(mode, mainDomain, auxDomain, preferredCNAME string) (string, string, error) {
	cfg := d.store.GetConfig()
	if cfg.ServiceURL == "" {
		return mode, "", fmt.Errorf("service_url 未配置，请先在面板中设置")
	}
	return d.BindDomainWithMode(mode, mainDomain, auxDomain, cfg.ServiceURL, preferredCNAME)
}

// BindDomainWithMode dispatches to simple direct binding or the existing preferred SaaS flow.
func (d *DomainService) BindDomainWithMode(mode, mainDomain, auxDomain, serviceURL, preferredCNAME string) (string, string, error) {
	actual, err := NormalizeBindingMode(mode)
	if err != nil {
		return "", "", err
	}
	cfg := d.store.GetConfig()
	if cfg.TunnelID == "" || strings.TrimSpace(serviceURL) == "" {
		return actual, "", fmt.Errorf("tunnel_id 或 service_url 未配置，请先在面板中设置")
	}
	mainDomain = strings.TrimSuffix(strings.TrimSpace(mainDomain), ".")
	auxDomain = strings.TrimSuffix(strings.TrimSpace(auxDomain), ".")
	if mainDomain == "" {
		return actual, "", fmt.Errorf("main_domain is required")
	}
	if actual == BindingModeSimple {
		return actual, "", d.bindSimple(cfg, mainDomain, serviceURL)
	}
	if auxDomain == "" {
		return actual, "", fmt.Errorf("aux_domain is required for preferred mode")
	}
	if preferredCNAME == "" {
		preferredCNAME = cfg.PreferredCNAME
	}
	if preferredCNAME == "" {
		return actual, "", fmt.Errorf("preferred_cname 未配置")
	}
	return actual, preferredCNAME, d.bindPreferred(cfg, mainDomain, auxDomain, serviceURL, preferredCNAME)
}

func (d *DomainService) bindSimple(cfg models.Config, mainDomain, serviceURL string) error {
	zoneID, err := d.cf.GetZoneIDByHostname(mainDomain)
	if err != nil {
		return fmt.Errorf("主域名 zone 查询失败: %w", err)
	}
	if err = d.upsertIngress(cfg.TunnelID, []models.IngressRule{{Hostname: mainDomain, Service: serviceURL}}, map[string]bool{mainDomain: true}); err != nil {
		return err
	}
	tunnelCNAME := fmt.Sprintf("%s.cfargotunnel.com", cfg.TunnelID)
	if err = d.cf.UpsertDNSRecord(zoneID, mainDomain, "CNAME", tunnelCNAME, true); err != nil {
		return fmt.Errorf("隧道路由已更新，但主域名 DNS 创建失败: %w", err)
	}
	return nil
}

func (d *DomainService) bindPreferred(cfg models.Config, mainDomain, auxDomain, serviceURL, preferredCNAME string) error {
	mainZoneID, err := d.cf.GetZoneIDByHostname(mainDomain)
	if err != nil {
		return fmt.Errorf("主域名 zone 查询失败: %w", err)
	}
	auxZoneID, err := d.cf.GetZoneIDByHostname(auxDomain)
	if err != nil {
		return fmt.Errorf("辅助域名 zone 查询失败: %w", err)
	}
	replace := map[string]bool{mainDomain: true, auxDomain: true}
	rules := []models.IngressRule{{Hostname: mainDomain, Service: serviceURL}, {Hostname: auxDomain, Service: serviceURL}}
	if err = d.upsertIngress(cfg.TunnelID, rules, replace); err != nil {
		return err
	}
	tunnelCNAME := fmt.Sprintf("%s.cfargotunnel.com", cfg.TunnelID)
	if err = d.cf.UpsertDNSRecord(auxZoneID, auxDomain, "CNAME", tunnelCNAME, true); err != nil {
		return fmt.Errorf("隧道路由已更新，但辅助域名 DNS 设置失败: %w", err)
	}
	if err = d.cf.UpsertDNSRecord(mainZoneID, mainDomain, "CNAME", preferredCNAME, false); err != nil {
		return fmt.Errorf("隧道路由已更新，但主域名 DNS 设置失败: %w", err)
	}
	if err = d.cf.UpsertCustomHostname(auxZoneID, mainDomain, auxDomain); err != nil {
		return fmt.Errorf("DNS 已更新，但 SaaS 主机名设置失败: %w", err)
	}
	return nil
}

func (d *DomainService) upsertIngress(tunnelID string, newRules []models.IngressRule, replace map[string]bool) error {
	tunnelCfg, err := d.cf.GetTunnelConfig(tunnelID)
	if err != nil {
		return fmt.Errorf("获取隧道配置失败: %w", err)
	}
	tunnelCfg.Result.Config.Ingress = mergeIngressRules(tunnelCfg.Result.Config.Ingress, newRules, replace)
	if err = d.cf.UpdateTunnelConfig(tunnelID, map[string]interface{}{"config": tunnelCfg.Result.Config}); err != nil {
		return fmt.Errorf("更新隧道配置失败: %w", err)
	}
	return nil
}

// mergeIngressRules preserves every existing rule and inserts managed hostname rules
// immediately before the terminal catch-all. Path-only rules are not fallbacks.
func mergeIngressRules(existing, newRules []models.IngressRule, replace map[string]bool) []models.IngressRule {
	filtered := make([]models.IngressRule, 0, len(existing)+len(newRules)+1)
	for _, rule := range existing {
		if rule.Hostname != "" && replace[rule.Hostname] {
			continue
		}
		filtered = append(filtered, rule)
	}

	fallbackIndex := -1
	if len(filtered) > 0 {
		last := filtered[len(filtered)-1]
		if last.Hostname == "" && last.Path == "" {
			fallbackIndex = len(filtered) - 1
		}
	}
	if fallbackIndex == -1 {
		filtered = append(filtered, newRules...)
		return append(filtered, models.IngressRule{Service: "http_status:404"})
	}

	result := make([]models.IngressRule, 0, len(filtered)+len(newRules))
	result = append(result, filtered[:fallbackIndex]...)
	result = append(result, newRules...)
	result = append(result, filtered[fallbackIndex])
	return result
}

func (d *DomainService) SetFallbackOrigin(domain string) error {
	zoneID, err := d.cf.GetZoneIDByHostname(domain)
	if err != nil {
		return fmt.Errorf("未找到域名对应的 zone: %w", err)
	}
	return d.cf.SetFallbackOrigin(zoneID, domain)
}
