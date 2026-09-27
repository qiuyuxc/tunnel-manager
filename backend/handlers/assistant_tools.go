package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"tunnel-manager/models"
	"tunnel-manager/services"
)

func aiPermission(tool string) string {
	switch tool {
	case "create_tunnel":
		return models.PermTunnels
	case "create_dns_record":
		return models.PermDNS
	case "bind_domain":
		return models.PermDomainBind
	case "create_monitor", "add_monitor_target":
		return models.PermMonitors
	}
	return ""
}

func aiTools(user *models.SessionUser) []map[string]any {
	text := func(description string) map[string]any {
		return map[string]any{"type": "string", "description": description}
	}
	definitions := []struct {
		name, description string
		properties        map[string]any
		required          []string
	}{
		{"bind_domain", "准备将已有隧道上的 HTTP/HTTPS 服务通过新域名公开访问。仅新增直连 CNAME（代理开启、TTL自动）及路由，不覆盖已有 DNS 或路由；不支持优选/SaaS模式。执行前必须让用户确认将服务公开到互联网。", map[string]any{"tunnel_id": text("已存在隧道的真实 ID"), "zone_id": text("域名区域真实 ID"), "hostname": text("要公开的新域名"), "service_url": text("源站 HTTP/HTTPS URL，例如 http://192.168.1.8:5000")}, []string{"tunnel_id", "zone_id", "hostname", "service_url"}},
		{"create_tunnel", "准备创建隧道，不会立即执行。不含部署主机或启动连接器。", map[string]any{"name": text("隧道名称")}, []string{"name"}},
		{"create_dns_record", "准备新增 DNS 记录，不修改已有记录。必须使用上下文或用户提供的真实 zone_id。", map[string]any{"zone_id": text("区域 ID"), "name": text("完整域名"), "type": map[string]any{"type": "string", "enum": []string{"A", "AAAA", "CNAME", "TXT", "MX"}}, "content": text("记录值"), "ttl": map[string]any{"type": "integer", "description": "1 表示自动，否则 60 至 86400"}, "proxied": map[string]any{"type": "boolean"}, "priority": map[string]any{"type": "integer", "description": "仅 MX 必填，0 至 65535"}}, []string{"zone_id", "name", "type", "content", "ttl", "proxied"}},
		{"create_monitor", "准备创建空的私有监控项目，默认不公开、不发告警。执行成功后才能添加目标。", map[string]any{"name": text("项目名称"), "interval_sec": map[string]any{"type": "integer", "description": "探测间隔秒数，60 至 3600"}}, []string{"name", "interval_sec"}},
		{"add_monitor_target", "为已有的本人监控项目准备添加 HTTP GET 探测目标，添加后会周期访问该 URL。", map[string]any{"monitor_id": text("已存在项目的真实 ID"), "name": text("目标名称"), "url": text("目标 HTTP 或 HTTPS URL，不可含密码")}, []string{"monitor_id", "name", "url"}},
	}
	tools := []map[string]any{}
	for _, definition := range definitions {
		if user.HasPerm(aiPermission(definition.name)) {
			tools = append(tools, map[string]any{"type": "function", "function": map[string]any{"name": definition.name, "description": definition.description, "parameters": map[string]any{"type": "object", "properties": definition.properties, "required": definition.required, "additionalProperties": false}}})
		}
	}
	return tools
}

var aiResourceID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
var aiHostnameLabel = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

func aiDecodeArguments(raw string, target any) error {
	if len(raw) > 12000 {
		return errors.New("任务参数过大")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("任务参数格式无效")
	}
	return nil
}

func aiBuildTask(user *models.SessionUser, call services.AIToolCall) (models.AITask, error) {
	task := models.AITask{ID: newAPIToken(), Tool: call.Function.Name, Status: "pending"}
	permission := aiPermission(task.Tool)
	if call.Type != "function" || permission == "" || !user.HasPerm(permission) {
		return task, errors.New("任务无权限")
	}
	var normalized any
	switch task.Tool {
	case "bind_domain":
		var payload models.AIBindingRequest
		if aiDecodeArguments(call.Function.Arguments, &payload) != nil || !aiResourceID.MatchString(payload.TunnelID) || !aiResourceID.MatchString(payload.ZoneID) {
			return task, errors.New("域名绑定参数无效")
		}
		payload.Hostname = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(payload.Hostname)), ".")
		if len(payload.Hostname) > 253 || !strings.Contains(payload.Hostname, ".") {
			return task, errors.New("请输入完整域名")
		}
		for _, label := range strings.Split(payload.Hostname, ".") {
			if len(label) > 63 || !aiHostnameLabel.MatchString(label) {
				return task, errors.New("域名格式无效")
			}
		}
		payload.ServiceURL = strings.TrimSpace(payload.ServiceURL)
		parsed, err := url.Parse(payload.ServiceURL)
		if err != nil || len(payload.ServiceURL) > 2048 || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
			return task, errors.New("源站地址无效")
		}
		task.Title, normalized = "公开服务（直连绑定） · "+payload.Hostname, payload
	case "create_tunnel":
		var payload models.CreateTunnelRequest
		if aiDecodeArguments(call.Function.Arguments, &payload) != nil || strings.TrimSpace(payload.Name) == "" || len(payload.Name) > 200 {
			return task, errors.New("隧道名称无效")
		}
		payload.Name = strings.TrimSpace(payload.Name)
		task.Title, normalized = "新建隧道 · "+payload.Name, payload
	case "create_dns_record":
		var payload struct {
			ZoneID string `json:"zone_id"`
			models.DNSRecordRequest
		}
		if aiDecodeArguments(call.Function.Arguments, &payload) != nil || !aiResourceID.MatchString(payload.ZoneID) || len(payload.Name) > 253 || len(payload.Content) > 4096 {
			return task, errors.New("DNS 参数无效")
		}
		body, _ := json.Marshal(payload.DNSRecordRequest)
		request, _ := http.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		capture := &aiCapture{header: http.Header{}}
		record, valid := readDNSRecordRequest(capture, request, payload.ZoneID)
		if !valid {
			return task, errors.New("DNS 参数无效")
		}
		payload.DNSRecordRequest = record
		task.Title, normalized = "新增 DNS · "+record.Name, payload
	case "create_monitor":
		var payload struct {
			Name     string `json:"name"`
			Interval int    `json:"interval_sec"`
		}
		if aiDecodeArguments(call.Function.Arguments, &payload) != nil || strings.TrimSpace(payload.Name) == "" || len(payload.Name) > 200 || payload.Interval < 60 || payload.Interval > 3600 {
			return task, errors.New("监控项目参数无效")
		}
		payload.Name = strings.TrimSpace(payload.Name)
		task.Title, normalized = "新建私有监控项目 · "+payload.Name, payload
	case "add_monitor_target":
		var payload struct {
			MonitorID string `json:"monitor_id"`
			Name      string `json:"name"`
			URL       string `json:"url"`
		}
		if aiDecodeArguments(call.Function.Arguments, &payload) != nil || !aiResourceID.MatchString(payload.MonitorID) || len(payload.Name) > 200 || len(payload.URL) > 2048 {
			return task, errors.New("监控目标参数无效")
		}
		name, targetURL, _, _, invalid, _ := probeSpec(payload.Name, payload.URL, "http", "GET")
		parsed, parseErr := url.Parse(targetURL)
		if invalid || parseErr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil {
			return task, errors.New("监控地址无效")
		}
		payload.Name, payload.URL = name, targetURL
		task.Title, normalized = "添加 HTTP GET 探测 · "+name, payload
	}
	arguments, err := json.Marshal(normalized)
	task.Arguments = arguments
	return task, err
}

func (handler *AssistantHandler) BindDomain(response http.ResponseWriter, request *http.Request) {
	var payload models.AIBindingRequest
	if readAdminJSON(response, request, &payload) != nil || UserCF(request) == nil {
		writeJSON(response, http.StatusBadRequest, map[string]string{"error": "绑定参数或 Cloudflare 连接无效"})
		return
	}
	if err := UserCF(request).CreateAssistantBinding(request.Context(), payload); err != nil {
		writeJSON(response, http.StatusBadRequest, map[string]string{"error": "绑定未完成，请核对 DNS 和隧道路由"})
		return
	}
	SetAuditTarget(request, payload.Hostname)
	writeJSON(response, http.StatusOK, map[string]string{"id": payload.Hostname})
}
