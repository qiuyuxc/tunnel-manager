package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"tunnel-manager/models"
)

func aiChangeTools(dnsData map[string]any) []aiToolDefinition {
	text := func(description string) map[string]any {
		return map[string]any{"type": "string", "description": description}
	}
	boolean := map[string]any{"type": "boolean"}
	selector := func() map[string]any {
		return map[string]any{"zone_id": text("真实区域 ID"), "record_id": text("可选，真实记录 ID；同名同类型有多条时必填"), "record_name": text("现有记录的完整域名"), "record_type": text("现有记录类型")}
	}
	partialData := map[string]any{}
	for key, value := range dnsData {
		partialData[key] = value
	}
	partialData["description"] = "仅 SRV/CAA，可只提供要修改的数据字段，省略的保持原值；切换记录类型时须给出完整数据。"
	updateDNS := selector()
	updateDNS["changes"] = map[string]any{"type": "object", "additionalProperties": false, "minProperties": 1, "description": "仅提供用户要求修改的字段。省略的值保持现状，data 内也可只改部分字段。切换类型必须提供新的 content 或 data。", "properties": map[string]any{
		"name": text("新的完整域名"), "type": text("新的记录类型"), "content": text("新的记录值"), "data": partialData,
		"ttl": map[string]any{"type": "integer", "description": "1 为自动，或 60 至 86400；省略则保留"}, "proxied": boolean, "priority": map[string]any{"type": "integer"},
	}}
	return []aiToolDefinition{
		{"update_dns_record", "准备修改一条现有 DNS。服务端按名称和类型查找并展示原值，匹配多条时必须提供记录 ID；不会新增记录。", updateDNS, []string{"zone_id", "record_name", "record_type", "changes"}},
		{"delete_dns_record", "准备删除一条 DNS，可能导致域名服务中断。服务端核对真实记录并展示原值，同名同类型多条时须指定 ID。", selector(), []string{"zone_id", "record_name", "record_type"}},
		{"delete_tunnel", "准备删除已有隧道，可能中断服务；不自动删除 DNS，不强制断开活动连接。", map[string]any{"tunnel_id": text("真实隧道 ID")}, []string{"tunnel_id"}},
		{"delete_monitor", "准备删除本人的监控项目及其探测目标和历史记录。", map[string]any{"monitor_id": text("本人的监控项目 ID")}, []string{"monitor_id"}},
		{"delete_monitor_target", "准备删除本人监控项目中的一个探测目标。", map[string]any{"monitor_id": text("本人的监控项目 ID"), "target_id": text("探测目标 ID")}, []string{"monitor_id", "target_id"}},
		{"update_site_settings", "管理员修改站点名称、描述、图标和落地页开关。只提供要改的字段，省略的保持现状。", map[string]any{"name": text("站点名称"), "description": text("站点描述"), "icon": text("图标 URL，空字符串清除"), "landing_enabled": boolean}, []string{}},
		{"update_app_settings", "管理员修改注册开关、邀请码模式或实验功能开关。只提供要改的字段。", map[string]any{"registration_enabled": boolean, "invite_mode": map[string]any{"type": "string", "enum": []string{"off", "optional", "required"}}, "experimental_features_enabled": boolean}, []string{}},
		{"set_preferred_cname", "管理员修改全局默认优选 CNAME。不会自动修改已有 DNS。", map[string]any{"value": text("优选 CNAME 域名")}, []string{"value"}},
		{"set_cname_presets", "管理员替换完整的优选 CNAME 预设列表。确认卡展示原列表，至少保留一项。", map[string]any{"items": map[string]any{"type": "array", "minItems": 1, "maxItems": 20, "items": map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"name": text("预设名称"), "value": text("CNAME 域名")}, "required": []string{"name", "value"}}}}, []string{"items"}},
	}
}

// aiObject validates a partial update without applying creation defaults.
func aiObject(raw string, allowed ...string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := aiDecodeArguments(raw, &fields); err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return nil, errors.New("请指定要修改的字段")
	}
	for key := range fields {
		found := false
		for _, candidate := range allowed {
			if key == candidate {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("不支持字段 %s", key)
		}
	}
	return fields, nil
}

func aiString(fields map[string]json.RawMessage, key string) string {
	var value string
	_ = json.Unmarshal(fields[key], &value)
	return value
}

func aiBuildChange(tool, raw string) (any, string, error) {
	var fields map[string]json.RawMessage
	var err error
	title := ""
	switch tool {
	case "update_dns_record", "delete_dns_record":
		keys := []string{"zone_id", "record_id", "record_name", "record_type"}
		if tool == "update_dns_record" {
			keys = append(keys, "changes")
		}
		fields, err = aiObject(raw, keys...)
		if err != nil {
			return nil, "", err
		}
		name := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(aiString(fields, "record_name"))), ".")
		typ := strings.ToUpper(strings.TrimSpace(aiString(fields, "record_type")))
		if !aiResourceID.MatchString(aiString(fields, "zone_id")) || name == "" || len(name) > 253 || !strings.Contains(name, ".") || !models.EditableDNSRecordType(typ) {
			return nil, "", errors.New("请提供区域 ID、现有完整域名和记录类型")
		}
		if id, exists := fields["record_id"]; exists && !aiResourceID.MatchString(aiString(map[string]json.RawMessage{"id": id}, "id")) {
			return nil, "", errors.New("记录 ID 无效")
		}
		fields["record_name"], _ = json.Marshal(name)
		fields["record_type"], _ = json.Marshal(typ)
		if tool == "update_dns_record" {
			changes, changeErr := aiObject(string(fields["changes"]), "name", "type", "content", "ttl", "proxied", "priority", "data")
			var typed models.DNSRecordRequest
			if changeErr != nil || aiDecodeArguments(string(fields["changes"]), &typed) != nil {
				return nil, "", errors.New("DNS 修改参数无效")
			}
			if data, exists := changes["data"]; exists {
				if _, err := aiObject(string(data), "priority", "weight", "port", "target", "flags", "tag", "value"); err != nil {
					return nil, "", err
				}
			}
			title = "修改 DNS · " + name
		} else {
			title = "删除 DNS · " + name
		}
	case "delete_tunnel", "delete_monitor", "delete_monitor_target":
		keys := []string{"monitor_id"}
		title = "删除监控项目"
		if tool == "delete_tunnel" {
			keys, title = []string{"tunnel_id"}, "删除隧道"
		}
		if tool == "delete_monitor_target" {
			keys, title = append(keys, "target_id"), "删除探测目标"
		}
		fields, err = aiObject(raw, keys...)
		if err == nil {
			for _, key := range keys {
				if !aiResourceID.MatchString(aiString(fields, key)) {
					err = errors.New("资源 ID 无效")
				}
			}
		}
	case "update_site_settings":
		fields, err = aiObject(raw, "name", "description", "icon", "landing_enabled")
		var typed models.SetSiteSettingsRequest
		if err == nil {
			err = aiDecodeArguments(raw, &typed)
		}
		title = "修改站点设置"
	case "update_app_settings":
		fields, err = aiObject(raw, "registration_enabled", "invite_mode", "experimental_features_enabled")
		var typed models.SaveAppSettingsRequest
		if err == nil {
			err = aiDecodeArguments(raw, &typed)
		}
		if _, ok := fields["invite_mode"]; ok && typed.InviteMode != "off" && typed.InviteMode != "optional" && typed.InviteMode != "required" {
			err = errors.New("邀请码模式无效")
		}
		title = "修改后台设置"
	case "set_preferred_cname":
		var typed models.SetValueRequest
		err = aiDecodeArguments(raw, &typed)
		items, invalid := normalizeCNAMEPresets([]models.CNAMEPreset{{Name: "默认", Value: typed.Value}})
		if err != nil || invalid != nil {
			return nil, "", errors.New("优选 CNAME 无效")
		}
		return models.SetValueRequest{Value: items[0].Value}, "修改默认优选 CNAME", nil
	case "set_cname_presets":
		var typed models.SetCNAMEPresetsRequest
		if err := aiDecodeArguments(raw, &typed); err != nil {
			return nil, "", err
		}
		typed.Items, err = normalizeCNAMEPresets(typed.Items)
		return typed, "替换优选 CNAME 预设", err
	default:
		err = errors.New("不支持的任务")
	}
	return fields, title, err
}

func aiFields(value any) map[string]json.RawMessage {
	raw, _ := json.Marshal(value)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	return fields
}

func aiMerge(base, changes map[string]json.RawMessage) {
	for key, value := range changes {
		base[key] = value
	}
}

// DNS snapshots contain editable values only, excluding provider metadata.
func aiDNSFields(record *models.DNSRecord) (map[string]json.RawMessage, error) {
	if record == nil || !models.EditableDNSRecordType(record.Type) {
		return nil, errors.New("DNS 记录不存在或类型只读")
	}
	payload := models.DNSRecordRequest{Name: record.Name, Type: record.Type, Content: record.Content, TTL: record.TTL, Proxied: record.Proxied}
	if record.Type == "MX" {
		payload.Priority = &record.Priority
	}
	if record.Type == "SRV" || record.Type == "CAA" {
		payload.Content = ""
		if err := json.Unmarshal(record.Data, &payload.Data); err != nil {
			return nil, errors.New("无法读取结构化 DNS 数据")
		}
		if payload.Data != nil {
			payload.Data.Service, payload.Data.Proto, payload.Data.Name = "", "", ""
		}
	}
	return aiFields(payload), nil
}

func (handler *AssistantHandler) prepareChange(request *http.Request, user *models.SessionUser, task *models.AITask) error {
	var args map[string]json.RawMessage
	if err := json.Unmarshal(task.Arguments, &args); err != nil {
		return err
	}
	if task.Tool == "update_dns_record" || task.Tool == "delete_dns_record" {
		client := UserCF(request)
		if client == nil {
			return errors.New("请先连接 Cloudflare")
		}
		zone, id, name, typ := aiString(args, "zone_id"), aiString(args, "record_id"), aiString(args, "record_name"), aiString(args, "record_type")
		if id == "" {
			records, err := client.ListDNSRecords(zone, typ, name)
			if err != nil {
				return errors.New("无法读取 DNS 记录，请检查授权")
			}
			matches := []models.DNSRecord{}
			for _, record := range records {
				if strings.EqualFold(strings.TrimSuffix(record.Name, "."), name) && record.Type == typ {
					matches = append(matches, record)
				}
			}
			if len(matches) != 1 {
				return errors.New("DNS 未唯一匹配，请提供业务页面中的记录 ID")
			}
			id = matches[0].ID
		}
		if !aiResourceID.MatchString(id) {
			return errors.New("DNS 记录 ID 无效")
		}
		record, err := client.GetDNSRecord(zone, id)
		if err != nil {
			return errors.New("无法读取 DNS 记录，请检查授权")
		}
		if record.ID != id || !strings.EqualFold(strings.TrimSuffix(record.Name, "."), name) || record.Type != typ {
			return errors.New("记录 ID 与域名或类型不符，请重新核对")
		}
		before, err := aiDNSFields(record)
		if err != nil {
			return err
		}
		task.Before, _ = json.Marshal(before)
		final := aiFields(map[string]string{"zone_id": zone, "record_id": id})
		if task.Tool == "update_dns_record" {
			var changes map[string]json.RawMessage
			_ = json.Unmarshal(args["changes"], &changes)
			after := aiFields(before)
			if newType := strings.ToUpper(strings.TrimSpace(aiString(changes, "type"))); newType != "" && newType != typ {
				delete(after, "data")
				delete(after, "priority")
				delete(after, "content")
			}
			if data, exists := changes["data"]; exists {
				merged := map[string]json.RawMessage{}
				_ = json.Unmarshal(after["data"], &merged)
				var partial map[string]json.RawMessage
				_ = json.Unmarshal(data, &partial)
				aiMerge(merged, partial)
				changes["data"], _ = json.Marshal(merged)
			}
			aiMerge(after, changes)
			raw, _ := json.Marshal(after)
			var payload models.DNSRecordRequest
			if json.Unmarshal(raw, &payload) != nil {
				return errors.New("DNS 修改参数无效")
			}
			if err := models.ValidateDNSRecordRequest(&payload); err != nil {
				return err
			}
			aiMerge(final, aiFields(payload))
		}
		task.Arguments, _ = json.Marshal(final)
		return nil
	}
	if !aiNeedsSnapshot(task.Tool) {
		return nil
	}
	before, err := handler.changeSnapshot(request, user, *task)
	if err != nil {
		return err
	}
	task.Before, _ = json.Marshal(before)
	if task.Tool == "update_site_settings" {
		after := aiFields(before)
		aiMerge(after, args)
		raw, _ := json.Marshal(after)
		var payload models.SetSiteSettingsRequest
		if json.Unmarshal(raw, &payload) != nil {
			return errors.New("站点参数无效")
		}
		payload.Name, payload.Description, payload.Icon = strings.TrimSpace(payload.Name), strings.TrimSpace(payload.Description), strings.TrimSpace(payload.Icon)
		if err := validateSiteSettings(payload); err != nil {
			return err
		}
		// Keep the unrelated panel-host guard out of both the plan and request.
		after = aiFields(payload)
		delete(after, "panel_host")
		task.Arguments, _ = json.Marshal(after)
	}
	if strings.HasPrefix(task.Tool, "delete_") {
		task.Title += " · " + aiString(aiFields(before), "name")
	}
	return nil
}

func aiNeedsSnapshot(tool string) bool {
	return strings.HasPrefix(tool, "delete_") || tool == "update_dns_record" || aiPermission(tool) == "assistant_admin"
}

func aiRouteParams(tool string) map[string]string {
	switch tool {
	case "create_dns_record":
		return map[string]string{"zone_id": "zoneID"}
	case "update_dns_record", "delete_dns_record":
		return map[string]string{"zone_id": "zoneID", "record_id": "recordID"}
	case "delete_tunnel":
		return map[string]string{"tunnel_id": "tunnelID"}
	case "add_monitor_target", "delete_monitor":
		return map[string]string{"monitor_id": "monitorID"}
	case "delete_monitor_target":
		return map[string]string{"monitor_id": "monitorID", "target_id": "targetID"}
	}
	return nil
}

func aiChangeLock(task models.AITask) string {
	var args map[string]json.RawMessage
	_ = json.Unmarshal(task.Arguments, &args)
	if aiPermission(task.Tool) == models.PermDNS {
		return "change:dns:" + aiString(args, "zone_id") + ":" + aiString(args, "record_id")
	}
	if aiPermission(task.Tool) == models.PermMonitors {
		return "change:monitor:" + aiString(args, "monitor_id")
	}
	return "change:" + task.Tool + ":" + aiString(args, "tunnel_id")
}

func aiTargetSnapshot(target models.MonitorTarget) map[string]any {
	return map[string]any{"id": target.ID, "name": target.Name, "url": target.URL, "type": target.Type, "method": target.Method}
}

func (handler *AssistantHandler) changeSnapshot(request *http.Request, user *models.SessionUser, task models.AITask) (any, error) {
	var args map[string]json.RawMessage
	if err := json.Unmarshal(task.Arguments, &args); err != nil {
		return nil, err
	}
	cfg := handler.store.GetConfig()
	switch task.Tool {
	case "update_dns_record", "delete_dns_record":
		if UserCF(request) == nil {
			return nil, errors.New("Cloudflare 不可用")
		}
		record, err := UserCF(request).GetDNSRecord(aiString(args, "zone_id"), aiString(args, "record_id"))
		if err != nil {
			return nil, err
		}
		return aiDNSFields(record)
	case "delete_tunnel":
		if UserCF(request) == nil {
			return nil, errors.New("Cloudflare 不可用")
		}
		tunnels, err := UserCF(request).ListTunnels()
		if err != nil {
			return nil, errors.New("无法读取隧道")
		}
		for _, tunnel := range tunnels {
			if tunnel.ID == aiString(args, "tunnel_id") {
				return map[string]string{"id": tunnel.ID, "name": tunnel.Name}, nil
			}
		}
	case "delete_monitor", "delete_monitor_target":
		for _, monitor := range cfg.Monitors {
			if monitor.ID != aiString(args, "monitor_id") || monitor.UserID != user.ID {
				continue
			}
			targets := []map[string]any{}
			for _, target := range monitor.Targets {
				if task.Tool == "delete_monitor_target" && target.ID == aiString(args, "target_id") {
					return aiTargetSnapshot(target), nil
				}
				targets = append(targets, aiTargetSnapshot(target))
			}
			if task.Tool == "delete_monitor" {
				return map[string]any{"id": monitor.ID, "name": monitor.Name, "targets": targets, "interval_sec": monitor.IntervalSec, "publish_enabled": monitor.PublishEnabled, "alert_enabled": monitor.AlertEnabled}, nil
			}
		}
	case "update_site_settings":
		return map[string]any{"name": cfg.SiteName, "description": cfg.SiteDescription, "icon": cfg.SiteIcon, "landing_enabled": cfg.LandingEnabled}, nil
	case "update_app_settings":
		settings := handler.store.GetAppSettings()
		return map[string]any{"registration_enabled": settings.RegistrationEnabled, "invite_mode": settings.InviteMode, "experimental_features_enabled": settings.ExperimentalFeatures}, nil
	case "set_preferred_cname":
		return map[string]string{"value": cfg.PreferredCNAME}, nil
	case "set_cname_presets":
		return map[string]any{"items": cfg.CNAMEPresets}, nil
	}
	return nil, errors.New("资源不存在或不属于当前用户")
}

func (handler *AssistantHandler) checkChange(request *http.Request, user *models.SessionUser, task models.AITask) error {
	if !aiNeedsSnapshot(task.Tool) {
		return nil
	}
	current, err := handler.changeSnapshot(request, user, task)
	if err != nil {
		return errors.New("无法核对原资源，未执行。请检查资源和权限后重新生成任务")
	}
	var before, actual any
	raw, _ := json.Marshal(current)
	if json.Unmarshal(task.Before, &before) != nil || json.Unmarshal(raw, &actual) != nil || !reflect.DeepEqual(before, actual) {
		return errors.New("资源或设置已变化，未执行。请重新生成任务并确认新旧值")
	}
	return nil
}

// Apply only ordinary settings through the same store validation and audit
// category as the admin form, preserving all unrelated fields verbatim.
func (handler *AssistantHandler) UpdateAppSettings(response http.ResponseWriter, request *http.Request) {
	raw, err := io.ReadAll(io.LimitReader(request.Body, 12001))
	if err != nil {
		writeJSON(response, http.StatusBadRequest, map[string]string{"error": "设置参数无效"})
		return
	}
	changes, _, err := aiBuildChange("update_app_settings", string(raw))
	if err != nil {
		writeJSON(response, http.StatusBadRequest, map[string]string{"error": "设置参数无效"})
		return
	}
	fields := changes.(map[string]json.RawMessage)
	settings := handler.store.GetAppSettings()
	if value, ok := fields["registration_enabled"]; ok {
		_ = json.Unmarshal(value, &settings.RegistrationEnabled)
	}
	if value, ok := fields["invite_mode"]; ok {
		_ = json.Unmarshal(value, &settings.InviteMode)
	}
	if value, ok := fields["experimental_features_enabled"]; ok {
		_ = json.Unmarshal(value, &settings.ExperimentalFeatures)
	}
	if err := handler.store.SetAppSettings(settings); err != nil {
		writeJSON(response, http.StatusServiceUnavailable, map[string]string{"error": "保存设置失败"})
		return
	}
	writeJSON(response, http.StatusOK, map[string]bool{"ok": true})
}
