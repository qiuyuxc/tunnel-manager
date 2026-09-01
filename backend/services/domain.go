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
	cf     *CloudflareClient
	store  *store.Store
	userID string
}

func NewDomainService(cf *CloudflareClient, st *store.Store) *DomainService {
	return &DomainService{cf: cf, store: st}
}

// ForUser returns a domain service bound to one account's connection.
func (s *DomainService) ForUser(userID string) *DomainService {
	clone := *s
	clone.cf = s.cf.ForUser(userID)
	clone.userID = userID
	return &clone
}

// bindSelection resolves the tunnel and origin service for a domain bind.
// The requesting user's saved selections take precedence; the global
// configuration is the fallback so legacy single-user setups keep working.
func (d *DomainService) bindSelection() (tunnelID, serviceURL string) {
	cfg := d.store.GetConfig()
	tunnelID, serviceURL = cfg.TunnelID, cfg.ServiceURL
	if d.userID != "" {
		prefs := d.store.GetUserPrefs(d.userID)
		if prefs.TunnelID != "" {
			tunnelID = prefs.TunnelID
		}
		if prefs.ServiceURL != "" {
			serviceURL = prefs.ServiceURL
		}
	}
	return tunnelID, serviceURL
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
	_, serviceURL := d.bindSelection()
	if serviceURL == "" {
		return "", fmt.Errorf("service_url 未配置，请先在面板中设置")
	}
	_, preferred, err := d.BindDomainWithMode(BindingModePreferred, mainDomain, auxDomain, serviceURL, preferredCNAME)
	return preferred, err
}
func (d *DomainService) BindDomainWithService(mainDomain, auxDomain, serviceURL, preferredCNAME string) (string, error) {
	_, preferred, err := d.BindDomainWithMode(BindingModePreferred, mainDomain, auxDomain, serviceURL, preferredCNAME)
	return preferred, err
}
func (d *DomainService) BindDomainWithConfiguredService(mode, mainDomain, auxDomain, preferredCNAME string) (string, string, error) {
	_, serviceURL := d.bindSelection()
	if serviceURL == "" {
		return mode, "", fmt.Errorf("service_url 未配置，请先在面板中设置")
	}
	return d.BindDomainWithMode(mode, mainDomain, auxDomain, serviceURL, preferredCNAME)
}

// BindDomainWithMode dispatches to simple direct binding or the existing preferred SaaS flow.
func (d *DomainService) BindDomainWithMode(mode, mainDomain, auxDomain, serviceURL, preferredCNAME string) (string, string, error) {
	actual, err := NormalizeBindingMode(mode)
	if err != nil {
		return "", "", err
	}
	cfg := d.store.GetConfig()
	tunnelID, _ := d.bindSelection()
	if tunnelID == "" || strings.TrimSpace(serviceURL) == "" {
		return actual, "", fmt.Errorf("tunnel_id 或 service_url 未配置，请先在面板中设置")
	}
	cfg.TunnelID = tunnelID
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

// ProvisionStatusDomain exposes one public status hostname through the
// monitor owner's selected tunnel using direct or preferred routing.
func (d *DomainService) ProvisionStatusDomain(userID, panelHost, hostname, mode, auxDomain, preferredCNAME string) error {
	userID = strings.TrimSpace(userID)
	panelHost = normalizeDomainName(panelHost)
	hostname = normalizeDomainName(hostname)
	auxDomain = normalizeDomainName(auxDomain)
	preferredCNAME = normalizeDomainName(preferredCNAME)
	actual, err := NormalizeBindingMode(mode)
	if err != nil {
		return err
	}
	if userID == "" {
		return fmt.Errorf("监控项目缺少所有者，无法选择 Cloudflare 连接")
	}
	if panelHost == "" {
		return fmt.Errorf("面板域名未配置，请管理员先在全局设置中填写")
	}
	if hostname == "" {
		return fmt.Errorf("自定义域名不能为空")
	}
	if hostname == panelHost {
		return fmt.Errorf("自定义域名不能与面板域名相同")
	}

	cfg := d.store.GetConfig()
	if actual == BindingModePreferred {
		if auxDomain == "" {
			return fmt.Errorf("优选模式需要填写辅助回源域名")
		}
		if auxDomain == hostname {
			return fmt.Errorf("访问域名和辅助回源域名不能相同")
		}
		if preferredCNAME == "" {
			preferredCNAME = normalizeDomainName(cfg.PreferredCNAME)
		}
		if preferredCNAME == "" {
			return fmt.Errorf("优选 CNAME 未配置，请填写本次使用的 CNAME 或在全局设置中配置默认值")
		}
	}
	prefs := d.store.GetUserPrefs(userID)
	if prefs.TunnelID == "" {
		return fmt.Errorf("未选择面板所在隧道，请先在全局设置中选择")
	}
	userService := d.ForUser(userID)
	tunnelCfg, err := userService.cf.GetTunnelConfig(prefs.TunnelID)
	if err != nil {
		return fmt.Errorf("读取面板隧道配置失败: %w", err)
	}
	panelRule, ok := findPanelIngressRule(tunnelCfg.Result.Config.Ingress, panelHost)
	if !ok {
		return fmt.Errorf("隧道配置中未找到面板域名 %s 对应的 ingress 规则", panelHost)
	}
	originRequest := cloneStatusOriginRequest(panelRule.OriginRequest)

	statusZoneID, err := userService.cf.GetZoneIDByHostname(hostname)
	if err != nil {
		return fmt.Errorf("访问域名不在当前 Cloudflare 连接的 zone 中: %w", err)
	}
	if actual == BindingModeSimple {
		cleanupDomain := auxDomain
		if cleanupDomain == "" {
			cleanupDomain = panelHost
		}
		cleanupZoneID, err := userService.cf.GetZoneIDByHostname(cleanupDomain)
		if err != nil {
			return fmt.Errorf("无法查询原 Custom Hostname 所在 zone: %w", err)
		}
		statusRule := models.IngressRule{
			Hostname:      hostname,
			Service:       panelRule.Service,
			OriginRequest: originRequest,
		}
		tunnelCfg.Result.Config.Ingress = mergeIngressRules(
			tunnelCfg.Result.Config.Ingress,
			[]models.IngressRule{statusRule},
			map[string]bool{hostname: true},
		)
		if err := userService.cf.UpdateTunnelConfig(prefs.TunnelID, map[string]interface{}{"config": tunnelCfg.Result.Config}); err != nil {
			return fmt.Errorf("更新隧道 ingress 失败: %w", err)
		}
		tunnelCNAME := fmt.Sprintf("%s.cfargotunnel.com", prefs.TunnelID)
		if err := userService.cf.UpsertDNSRecord(statusZoneID, hostname, "CNAME", tunnelCNAME, true); err != nil {
			return fmt.Errorf("ingress 已更新，但直连 Tunnel CNAME 创建失败: %w", err)
		}
		if err := userService.cf.DeleteCustomHostname(cleanupZoneID, hostname); err != nil {
			return fmt.Errorf("ingress 与直连 DNS 已更新，但清理 SaaS 自定义主机名失败: %w", err)
		}
		return nil
	}

	auxZoneID, err := userService.cf.GetZoneIDByHostname(auxDomain)
	if err != nil {
		return fmt.Errorf("辅助回源域名不在当前 Cloudflare 连接的 zone 中: %w", err)
	}
	if panelZoneID, panelZoneErr := userService.cf.GetZoneIDByHostname(panelHost); panelZoneErr == nil && panelZoneID != auxZoneID {
		if err := userService.cf.DeleteCustomHostname(panelZoneID, hostname); err != nil {
			return fmt.Errorf("清理旧版面板 zone 中的 SaaS 自定义主机名失败: %w", err)
		}
	}
	return userService.configurePreferredDomain(
		prefs.TunnelID,
		statusZoneID,
		auxZoneID,
		hostname,
		auxDomain,
		panelRule.Service,
		preferredCNAME,
		originRequest,
	)
}

func findPanelIngressRule(rules []models.IngressRule, panelHost string) (models.IngressRule, bool) {
	panelHost = normalizeDomainName(panelHost)
	for _, rule := range rules {
		if rule.Path == "" && normalizeDomainName(rule.Hostname) == panelHost && strings.TrimSpace(rule.Service) != "" {
			return rule, true
		}
	}
	return models.IngressRule{}, false
}

func normalizeDomainName(hostname string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(hostname), "."))
}

func cloneStatusOriginRequest(input map[string]interface{}) map[string]interface{} {
	if len(input) == 0 {
		return nil
	}
	cloned := make(map[string]interface{}, len(input))
	for key, value := range input {
		if strings.EqualFold(key, "httpHostHeader") {
			continue
		}
		cloned[key] = value
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
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
	return d.configurePreferredDomain(
		cfg.TunnelID,
		mainZoneID,
		auxZoneID,
		mainDomain,
		auxDomain,
		serviceURL,
		preferredCNAME,
		nil,
	)
}

func (d *DomainService) configurePreferredDomain(
	tunnelID,
	mainZoneID,
	auxZoneID,
	mainDomain,
	auxDomain,
	serviceURL,
	preferredCNAME string,
	originRequest map[string]interface{},
) error {
	replace := map[string]bool{mainDomain: true, auxDomain: true}
	rules := []models.IngressRule{
		{Hostname: mainDomain, Service: serviceURL, OriginRequest: cloneStatusOriginRequest(originRequest)},
		{Hostname: auxDomain, Service: serviceURL, OriginRequest: cloneStatusOriginRequest(originRequest)},
	}
	if err := d.upsertIngress(tunnelID, rules, replace); err != nil {
		return err
	}
	tunnelCNAME := fmt.Sprintf("%s.cfargotunnel.com", tunnelID)
	if err := d.cf.UpsertDNSRecord(auxZoneID, auxDomain, "CNAME", tunnelCNAME, true); err != nil {
		return fmt.Errorf("隧道路由已更新，但辅助域名 DNS 设置失败: %w", err)
	}
	if err := d.cf.UpsertDNSRecord(mainZoneID, mainDomain, "CNAME", preferredCNAME, false); err != nil {
		return fmt.Errorf("隧道路由已更新，但主域名 DNS 设置失败: %w", err)
	}
	if err := d.cf.UpsertCustomHostname(auxZoneID, mainDomain, auxDomain); err != nil {
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
