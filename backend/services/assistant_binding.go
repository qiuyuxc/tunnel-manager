package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tunnel-manager/models"
)

var assistantBindingSlot = make(chan struct{}, 1)

func (client *CloudflareClient) assistantRequest(ctx context.Context, method, path string, payload, result any) error {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}
	request, err := client.newRequest(method, path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	return client.do(request.WithContext(ctx), result)
}

func (client *CloudflareClient) CreateAssistantBinding(ctx context.Context, binding models.AIBindingRequest) error {
	ctx, cancel := context.WithTimeout(ctx, 150*time.Second)
	defer cancel()
	select {
	case assistantBindingSlot <- struct{}{}:
		defer func() { <-assistantBindingSlot }()
	case <-ctx.Done():
		return errors.New("等待其他绑定任务超时，未开始本次绑定")
	}
	accountID, err := client.currentAccountID()
	if err != nil {
		return err
	}
	var zone models.Zone
	if err := client.assistantRequest(ctx, http.MethodGet, "/zones/"+binding.ZoneID, nil, &zone); err != nil {
		return err
	}
	zoneName := strings.TrimSuffix(strings.ToLower(zone.Name), ".")
	if zoneName == "" || (binding.Hostname != zoneName && !strings.HasSuffix(binding.Hostname, "."+zoneName)) {
		return errors.New("域名不属于指定区域")
	}
	var records []models.DNSRecord
	dnsPath := "/zones/" + binding.ZoneID + "/dns_records"
	if err := client.assistantRequest(ctx, http.MethodGet, dnsPath+"?name="+url.QueryEscape(binding.Hostname), nil, &records); err != nil {
		return err
	}
	if len(records) > 0 {
		return errors.New("域名已有 DNS 记录，助手不会覆盖")
	}
	configPath := fmt.Sprintf("/accounts/%s/cfd_tunnel/%s/configurations", accountID, binding.TunnelID)
	var result struct {
		Config map[string]json.RawMessage `json:"config"`
	}
	if err := client.assistantRequest(ctx, http.MethodGet, configPath, nil, &result); err != nil {
		return err
	}
	if result.Config == nil {
		return errors.New("无法读取完整隧道配置，未执行绑定")
	}
	before, _ := json.Marshal(result.Config)
	var ingress []json.RawMessage
	if err := json.Unmarshal(result.Config["ingress"], &ingress); err != nil {
		return errors.New("隧道路由格式无效，未执行绑定")
	}
	fallbackIndex := len(ingress)
	for index, raw := range ingress {
		var rule models.IngressRule
		if json.Unmarshal(raw, &rule) != nil {
			return errors.New("隧道路由格式无效")
		}
		hostname := strings.TrimSuffix(strings.ToLower(rule.Hostname), ".")
		if hostname == binding.Hostname || (strings.HasPrefix(hostname, "*.") && strings.HasSuffix(binding.Hostname, hostname[1:])) {
			return errors.New("域名已有精确或通配路由，助手不会覆盖")
		}
		if hostname == "" && rule.Path == "" {
			if index != len(ingress)-1 {
				return errors.New("隧道包含非末尾兜底规则，请先手动核对")
			}
			fallbackIndex = index
		}
	}
	createdRule, _ := json.Marshal(models.IngressRule{Hostname: binding.Hostname, Service: binding.ServiceURL})
	rules := append([]json.RawMessage(nil), ingress[:fallbackIndex]...)
	rules = append(rules, createdRule)
	if fallbackIndex < len(ingress) {
		rules = append(rules, ingress[fallbackIndex:]...)
	} else {
		rules = append(rules, json.RawMessage(`{"service":"http_status:404"}`))
	}
	record := models.DNSRecordRequest{Type: "CNAME", Name: binding.Hostname, Content: binding.TunnelID + ".cfargotunnel.com", TTL: 1, Proxied: true}
	if err := client.assistantRequest(ctx, http.MethodPost, dnsPath, record, nil); err != nil {
		return err
	}
	var latest struct {
		Config map[string]json.RawMessage `json:"config"`
	}
	if err := client.assistantRequest(ctx, http.MethodGet, configPath, nil, &latest); err != nil {
		return errors.New("DNS 可能已创建，无法重新核对隧道配置，请手动检查")
	}
	after, _ := json.Marshal(latest.Config)
	if !bytes.Equal(before, after) {
		return errors.New("DNS 已创建，但隧道配置被其他操作更改，未覆盖路由，请手动检查")
	}
	result.Config["ingress"], _ = json.Marshal(rules)
	if err := client.assistantRequest(ctx, http.MethodPut, configPath, result, nil); err != nil {
		return errors.New("DNS 已创建，隧道路由结果不确定，请手动检查")
	}
	return nil
}
