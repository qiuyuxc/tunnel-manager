package services

import (
	"fmt"

	"tunnel-manager/models"
	"tunnel-manager/store"
)

// DomainService handles domain binding orchestration shared by REST and bot
type DomainService struct {
	cf    *CloudflareClient
	store *store.Store
}

// NewDomainService creates a new DomainService
func NewDomainService(cf *CloudflareClient, st *store.Store) *DomainService {
	return &DomainService{cf: cf, store: st}
}

// BindDomain runs the full flow using the configured default service URL.
func (d *DomainService) BindDomain(mainDomain, auxDomain string) (string, error) {
	return d.BindDomainWithPreferredCNAME(mainDomain, auxDomain, "")
}

// BindDomainWithPreferredCNAME runs the flow with the configured service URL and an optional CNAME override.
func (d *DomainService) BindDomainWithPreferredCNAME(mainDomain, auxDomain, preferredCNAME string) (string, error) {
	cfg := d.store.GetConfig()
	if cfg.ServiceURL == "" {
		return "", fmt.Errorf("service_url 未配置，请先在面板中设置")
	}
	return d.BindDomainWithService(mainDomain, auxDomain, cfg.ServiceURL, preferredCNAME)
}

// BindDomainWithService runs the full flow using the supplied service URL and optional preferred CNAME.
func (d *DomainService) BindDomainWithService(mainDomain, auxDomain, serviceURL, preferredCNAME string) (string, error) {
	cfg := d.store.GetConfig()
	if cfg.TunnelID == "" || serviceURL == "" {
		return "", fmt.Errorf("tunnel_id 或 service_url 未配置，请先在面板中设置")
	}
	if preferredCNAME == "" {
		preferredCNAME = cfg.PreferredCNAME
	}

	mainZoneID, err := d.cf.GetZoneIDByHostname(mainDomain)
	if err != nil {
		return "", fmt.Errorf("主域名 zone 查询失败: %w", err)
	}

	auxZoneID, err := d.cf.GetZoneIDByHostname(auxDomain)
	if err != nil {
		return "", fmt.Errorf("辅助域名 zone 查询失败: %w", err)
	}

	tunnelCfg, err := d.cf.GetTunnelConfig(cfg.TunnelID)
	if err != nil {
		return "", fmt.Errorf("获取隧道配置失败: %w", err)
	}

	newRules := []models.IngressRule{
		{Hostname: mainDomain, Service: serviceURL},
		{Hostname: auxDomain, Service: serviceURL},
	}

	ingress := tunnelCfg.Result.Config.Ingress
	filtered := make([]models.IngressRule, 0, len(ingress))
	for _, rule := range ingress {
		if rule.Hostname != mainDomain && rule.Hostname != auxDomain {
			filtered = append(filtered, rule)
		}
	}
	if len(filtered) > 0 {
		lastRule := filtered[len(filtered)-1]
		filtered = filtered[:len(filtered)-1]
		filtered = append(filtered, newRules...)
		filtered = append(filtered, lastRule)
	} else {
		filtered = append(newRules, models.IngressRule{Service: "http_status:404"})
	}
	tunnelCfg.Result.Config.Ingress = filtered

	if err := d.cf.UpdateTunnelConfig(cfg.TunnelID, map[string]interface{}{"config": tunnelCfg.Result.Config}); err != nil {
		return "", fmt.Errorf("更新隧道配置失败: %w", err)
	}

	tunnelCNAME := fmt.Sprintf("%s.cfargotunnel.com", cfg.TunnelID)
	if err := d.cf.UpsertDNSRecord(auxZoneID, auxDomain, "CNAME", tunnelCNAME, true); err != nil {
		return "", fmt.Errorf("设置辅助域名 DNS 失败: %w", err)
	}

	if err := d.cf.UpsertDNSRecord(mainZoneID, mainDomain, "CNAME", preferredCNAME, false); err != nil {
		return "", fmt.Errorf("设置主域名 DNS 失败: %w", err)
	}

	if err := d.cf.UpsertCustomHostname(auxZoneID, mainDomain, auxDomain); err != nil {
		return "", fmt.Errorf("设置 SaaS 主机名失败: %w", err)
	}

	return preferredCNAME, nil
}

// SetFallbackOrigin resolves the zone and sets the SaaS fallback origin
func (d *DomainService) SetFallbackOrigin(domain string) error {
	zoneID, err := d.cf.GetZoneIDByHostname(domain)
	if err != nil {
		return fmt.Errorf("未找到域名对应的 zone: %w", err)
	}
	return d.cf.SetFallbackOrigin(zoneID, domain)
}
