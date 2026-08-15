package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"tunnel-manager/models"
)

func (b *TelegramBot) handleDomainBindingMode(cfg models.Config, chatID int64, mode, mainDomain, auxDomain, preferred string) {
	b.sendMessage(cfg, chatID, "⏳ 正在配置隧道路由与 DNS，请稍候...")
	actual, cname, err := b.domain.BindDomainWithConfiguredService(mode, mainDomain, auxDomain, preferred)
	if err != nil {
		b.sendMessage(cfg, chatID, fmt.Sprintf("❌ 绑定失败: %s", err))
		return
	}
	if actual == BindingModeSimple {
		b.sendMessage(cfg, chatID, fmt.Sprintf("✅ 直连绑定成功！\n\n主域名: %s", mainDomain))
		return
	}
	b.sendMessage(cfg, chatID, fmt.Sprintf("✅ 优选绑定成功！\n\n主域名: %s\n辅助域名: %s\n优选 CNAME: %s", mainDomain, auxDomain, cname))
}

func (b *TelegramBot) handleListZones(cfg models.Config, chatID int64) {
	zones, err := b.cf.ListZones()
	if err != nil {
		b.sendMessage(cfg, chatID, fmt.Sprintf("❌ 获取区域失败: %s", err))
		return
	}
	if len(zones) == 0 {
		b.sendMessage(cfg, chatID, "未找到可用区域。")
		return
	}
	var out strings.Builder
	out.WriteString("🌐 Cloudflare 区域\n\n")
	for _, z := range zones {
		fmt.Fprintf(&out, "• %s\n  %s\n", z.Name, z.ID)
	}
	out.WriteString("\n使用 /DNS列表 [ZoneID] 查看记录。")
	b.sendMessage(cfg, chatID, out.String())
}

func (b *TelegramBot) handleListDNS(cfg models.Config, chatID int64, zoneID, typ, name string) {
	records, err := b.cf.ListDNSRecords(zoneID, typ, name)
	if err != nil {
		b.sendMessage(cfg, chatID, fmt.Sprintf("❌ 获取 DNS 失败: %s", err))
		return
	}
	if len(records) == 0 {
		b.sendMessage(cfg, chatID, "未找到匹配的 DNS 记录。")
		return
	}
	limit := len(records)
	if limit > 40 {
		limit = 40
	}
	var out strings.Builder
	fmt.Fprintf(&out, "📋 DNS 记录（%d 条）\n\n", len(records))
	for _, r := range records[:limit] {
		fmt.Fprintf(&out, "• %s %s → %s\n  ID: %s\n", r.Type, r.Name, r.Content, r.ID)
	}
	if len(records) > limit {
		fmt.Fprintf(&out, "\n仅显示前 %d 条，请添加过滤。", limit)
	}
	b.sendMessage(cfg, chatID, out.String())
}

func dnsWriteUsage(update bool) string {
	if update {
		return "❌ 用法: /DNS修改 [ZoneID] [RecordID] [类型] [名称] [内容] [TTL/auto] [代理on/off] [MX优先级]"
	}
	return "❌ 用法: /DNS添加 [ZoneID] [类型] [名称] [内容] [TTL/auto] [代理on/off] [MX优先级]"
}

func parseDNSWrite(args []string) (string, models.DNSRecordRequest, error) {
	if len(args) < 4 {
		return "", models.DNSRecordRequest{}, fmt.Errorf("参数不足")
	}
	zone, typ := args[0], strings.ToUpper(args[1])
	allowed := map[string]bool{"A": true, "AAAA": true, "CNAME": true, "TXT": true, "MX": true}
	if !allowed[typ] {
		return "", models.DNSRecordRequest{}, fmt.Errorf("仅支持 A/AAAA/CNAME/TXT/MX")
	}
	p := models.DNSRecordRequest{Type: typ, Name: args[2], Content: args[3], TTL: 1}
	if len(args) > 4 && strings.ToLower(args[4]) != "auto" {
		v, e := strconv.Atoi(args[4])
		if e != nil || v < 60 || v > 86400 {
			return "", p, fmt.Errorf("TTL 必须为 auto 或 60-86400")
		}
		p.TTL = v
	}
	expectedMax := 6
	if typ == "MX" {
		expectedMax = 7
	}
	if len(args) > expectedMax {
		return "", p, fmt.Errorf("参数过多")
	}
	if len(args) > 5 {
		switch strings.ToLower(args[5]) {
		case "on":
			p.Proxied = true
		case "off":
			p.Proxied = false
		default:
			return "", p, fmt.Errorf("代理状态只能是 on 或 off")
		}
	}
	if typ == "TXT" || typ == "MX" {
		if p.Proxied {
			return "", p, fmt.Errorf("%s 记录不支持 Cloudflare 代理", typ)
		}
		p.Proxied = false
	}
	if typ == "MX" {
		if len(args) < 7 {
			return "", p, fmt.Errorf("MX 需要优先级")
		}
		v, e := strconv.Atoi(args[6])
		if e != nil || v < 0 || v > 65535 {
			return "", p, fmt.Errorf("MX 优先级无效")
		}
		p.Priority = &v
	}
	return zone, p, nil
}

func (b *TelegramBot) handleDNSWriteCommand(cfg models.Config, chatID int64, recordID string, args []string) {
	zone, p, err := parseDNSWrite(args)
	if err != nil {
		b.sendMessage(cfg, chatID, fmt.Sprintf("%s\n%s", err, dnsWriteUsage(recordID != "")))
		return
	}
	if recordID == "" {
		_, err = b.cf.CreateDNSRecord(zone, p)
	} else {
		_, err = b.cf.UpdateDNSRecord(zone, recordID, p)
	}
	if err != nil {
		b.sendMessage(cfg, chatID, fmt.Sprintf("❌ DNS 操作失败: %s", err))
		return
	}
	b.sendMessage(cfg, chatID, "✅ DNS 记录已保存。")
}

func (b *TelegramBot) handleDNSDeleteRequest(cfg models.Config, chatID, userID int64, zone, recordID string) {
	records, err := b.cf.ListDNSRecords(zone, "", "")
	if err != nil {
		b.sendMessage(cfg, chatID, fmt.Sprintf("❌ 查询记录失败: %s", err))
		return
	}
	var target models.DNSRecord
	found := false
	for _, r := range records {
		if r.ID == recordID {
			target = r
			found = true
			break
		}
	}
	if !found {
		b.sendMessage(cfg, chatID, "❌ 未在该区域找到记录 ID。")
		return
	}
	label := fmt.Sprintf("%s %s → %s", target.Type, target.Name, target.Content)
	code, now := generateRandomHex(8), time.Now()
	b.confirmMu.Lock()
	for k, v := range b.confirmations {
		if now.After(v.Expires) {
			delete(b.confirmations, k)
		}
	}
	b.confirmations[code] = dnsDeleteConfirmation{ChatID: chatID, UserID: userID, ZoneID: zone, Record: target, Label: label, Expires: now.Add(5 * time.Minute)}
	b.confirmMu.Unlock()
	b.sendMessage(cfg, chatID, fmt.Sprintf("⚠️ 将删除: %s\n\n5 分钟内发送: /确认删除 %s", label, code))
}

func (b *TelegramBot) handleDNSDeleteConfirm(cfg models.Config, chatID, userID int64, code string) {
	b.confirmMu.Lock()
	c, ok := b.confirmations[code]
	if ok {
		delete(b.confirmations, code)
	}
	b.confirmMu.Unlock()
	if !ok || c.ChatID != chatID || c.UserID != userID || time.Now().After(c.Expires) {
		b.sendMessage(cfg, chatID, "❌ 确认码无效或已过期。")
		return
	}
	records, err := b.cf.ListDNSRecords(c.ZoneID, "", "")
	if err != nil {
		b.sendMessage(cfg, chatID, fmt.Sprintf("❌ 删除前校验失败: %s", err))
		return
	}
	var current *models.DNSRecord
	for i := range records {
		if records[i].ID == c.Record.ID {
			current = &records[i]
			break
		}
	}
	if current == nil || !sameDNSRecord(c.Record, *current) {
		b.sendMessage(cfg, chatID, "❌ DNS 记录在确认期间已变化，请重新发起删除。")
		return
	}
	if err := b.cf.DeleteDNSRecord(c.ZoneID, c.Record.ID); err != nil {
		b.sendMessage(cfg, chatID, fmt.Sprintf("❌ 删除失败: %s", err))
		return
	}
	b.sendMessage(cfg, chatID, fmt.Sprintf("✅ 已删除 DNS 记录: %s", c.Label))
}

func sameDNSRecord(a, b models.DNSRecord) bool {
	return a.ID == b.ID && a.Type == b.Type && a.Name == b.Name && a.Content == b.Content && a.TTL == b.TTL && a.Proxied == b.Proxied && a.Priority == b.Priority
}
