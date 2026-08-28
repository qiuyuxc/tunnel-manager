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
	limit := len(zones)
	if limit > 20 {
		limit = 20
	}
	var out strings.Builder
	out.WriteString("🌐 可用 Cloudflare 区域\n\n")
	for i, z := range zones[:limit] {
		fmt.Fprintf(&out, "%d. %s\n", i+1, z.Name)
	}
	if len(zones) > limit {
		fmt.Fprintf(&out, "\n共 %d 个区域，仅显示前 %d 个。", len(zones), limit)
	}
	out.WriteString("\n\nDNS 命令可直接使用区域域名，例如：\n/DNS列表 example.com\n选定后省略区域: /选择区域 example.com")
	b.sendMessage(cfg, chatID, out.String())
}

func (b *TelegramBot) handleSelectZone(cfg models.Config, chatID int64, arg string) {
	zoneID, zoneName, err := b.resolveZoneArg(arg)
	if err != nil {
		b.sendMessage(cfg, chatID, "❌ "+err.Error())
		return
	}
	if b.isPerUser() {
		if err := b.store.SetUserZoneSelection(b.userID, zoneID, zoneName); err != nil {
			b.sendMessage(cfg, chatID, fmt.Sprintf("❌ 保存区域选择失败: %s", err))
			return
		}
	} else if err := b.store.SetZoneSelection(zoneID, zoneName); err != nil {
		b.sendMessage(cfg, chatID, fmt.Sprintf("❌ 保存区域选择失败: %s", err))
		return
	}
	b.sendMessage(cfg, chatID, fmt.Sprintf("✅ 已选定区域: %s\n\n后续 DNS 命令可省略区域参数。\n例如: /DNS添加 A www 1.2.3.4\n切换: /选择区域 [域名或ZoneID]", zoneName))
}

func (b *TelegramBot) handleListDNS(cfg models.Config, chatID int64, zoneArg, typ, name string) {
	zoneID, zoneName, err := b.resolveZoneArg(zoneArg)
	if err != nil {
		b.sendMessage(cfg, chatID, "❌ "+err.Error())
		return
	}
	records, err := b.cf.ListDNSRecords(zoneID, typ, name)
	if err != nil {
		b.sendMessage(cfg, chatID, fmt.Sprintf("❌ 获取 DNS 失败: %s", err))
		return
	}
	if len(records) == 0 {
		b.sendMessage(cfg, chatID, fmt.Sprintf("未找到匹配的 DNS 记录（%s）。", zoneName))
		return
	}
	limit := len(records)
	if limit > 15 {
		limit = 15
	}
	var out strings.Builder
	fmt.Fprintf(&out, "📋 %s · %d 条记录\n\n", zoneName, len(records))
	for i, r := range records[:limit] {
		fmt.Fprintf(&out, "%d. %-5s %s → %s\n", i+1, r.Type, r.Name, truncateRunes(r.Content, 60))
	}
	if len(records) > limit {
		fmt.Fprintf(&out, "\n共 %d 条，仅显示前 %d 条。\n", len(records), limit)
	}
	out.WriteString("\n过滤: /DNS列表 example.com A www\n详情: /DNS详情 www.example.com")
	b.sendMessage(cfg, chatID, out.String())
}

func (b *TelegramBot) handleDNSDetail(cfg models.Config, chatID int64, zoneArg, nameArg string) {
	zoneID, zoneName, err := b.resolveZoneArg(zoneArg)
	if err != nil {
		if cfg.SelectedZoneID != "" && zoneArg == nameArg && !strings.Contains(zoneArg, ".") {
			// 单参数且不含点时，视为选定区域下的记录名
			zoneID, zoneName = cfg.SelectedZoneID, cfg.SelectedZoneName
		} else {
			b.sendMessage(cfg, chatID, "❌ "+err.Error())
			return
		}
	}
	records, err := b.cf.ListDNSRecords(zoneID, "", "")
	if err != nil {
		b.sendMessage(cfg, chatID, fmt.Sprintf("❌ 获取 DNS 失败: %s", err))
		return
	}
	var matched []models.DNSRecord
	for _, r := range records {
		if recordNameMatches(r.Name, nameArg) {
			matched = append(matched, r)
		}
	}
	if len(matched) == 0 {
		b.sendMessage(cfg, chatID, fmt.Sprintf("❌ 在 %s 未找到名称为 %q 的记录。", zoneName, nameArg))
		return
	}
	var out strings.Builder
	for _, r := range matched {
		fmt.Fprintf(&out, "📋 %s · %s\n\n", zoneName, r.Name)
		fmt.Fprintf(&out, "类型: %s\n", r.Type)
		fmt.Fprintf(&out, "名称: %s\n", r.Name)
		fmt.Fprintf(&out, "内容: %s\n", r.Content)
		ttl := "自动"
		if r.TTL > 1 {
			ttl = fmt.Sprintf("%d 秒", r.TTL)
		}
		fmt.Fprintf(&out, "TTL:  %s\n", ttl)
		proxied := "关"
		if r.Proxied {
			proxied = "开（橙云）"
		}
		fmt.Fprintf(&out, "代理: %s\n", proxied)
		if r.Priority > 0 {
			fmt.Fprintf(&out, "优先级: %d\n", r.Priority)
		}
		fmt.Fprintf(&out, "ID: %s\n\n", r.ID)
	}
	out.WriteString("修改: /DNS修改 [域名] [记录名] 类型 新内容\n删除: /DNS删除 [域名] [记录名]")
	b.sendMessage(cfg, chatID, out.String())
}

func recordNameMatches(recordName, arg string) bool {
	a := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(arg), "."))
	n := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(recordName), "."))
	return n == a || strings.HasPrefix(n, a+".")
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// resolveTunnelArg accepts a tunnel ID or a unique tunnel name and returns
// the matching tunnel.
func (b *TelegramBot) resolveTunnelArg(arg string) (models.Tunnel, error) {
	tunnels, err := b.cf.ListTunnels()
	if err != nil {
		return models.Tunnel{}, err
	}
	arg = strings.TrimSpace(arg)
	var best *models.Tunnel
	for i := range tunnels {
		t := &tunnels[i]
		if strings.EqualFold(arg, t.ID) {
			return *t, nil
		}
		if strings.EqualFold(arg, t.Name) {
			if best != nil {
				return models.Tunnel{}, fmt.Errorf("隧道名称 %q 有多个，请改用 ID 选择", arg)
			}
			best = t
		}
	}
	if best != nil {
		return *best, nil
	}
	return models.Tunnel{}, fmt.Errorf("未找到隧道 %q。可用 /列出隧道 查看", arg)
}

// resolveZoneArg accepts either a Cloudflare zone ID or a hostname and
// returns the matching zone ID and name. Hostnames are matched by longest
// suffix against active zone names (app.example.com → example.com).
func (b *TelegramBot) resolveZoneArg(arg string) (string, string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", "", fmt.Errorf("缺少区域参数")
	}
	zones, err := b.cf.ListZones()
	if err != nil {
		return "", "", err
	}
	host := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(arg), "."))
	var best *models.Zone
	for i := range zones {
		z := &zones[i]
		if strings.EqualFold(arg, z.ID) {
			return z.ID, z.Name, nil
		}
		zoneName := strings.ToLower(strings.TrimSpace(z.Name))
		if host == zoneName || strings.HasSuffix(host, "."+zoneName) {
			if best == nil || len(zoneName) > len(best.Name) {
				best = z
			}
		}
	}
	if best != nil {
		return best.ID, best.Name, nil
	}
	if len(zones) == 0 {
		return "", "", fmt.Errorf("Cloudflare 账户下没有可用区域")
	}
	names := make([]string, 0, len(zones))
	for _, z := range zones {
		names = append(names, z.Name)
	}
	return "", "", fmt.Errorf("未找到与 %q 匹配的区域。可用区域: %s", arg, strings.Join(names, ", "))
}

func dnsWriteUsage(update bool) string {
	if update {
		return "❌ 用法: /DNS修改 [区域可选] [记录名或ID] [类型] [新内容] [TTL可选] [代理可选]\n例如: /DNS修改 bbs CNAME saas.com（区域用选定的）\nTTL 缺省 auto，代理缺省 on（TXT/MX 自动关闭代理）"
	}
	return "❌ 用法: /DNS添加 [区域可选] [类型] [名称] [内容] [TTL可选] [代理可选] [MX优先级]\n例如: /DNS添加 A www 1.2.3.4（区域用选定的）\nTTL 缺省 auto，代理缺省 on（TXT/MX 自动关闭代理）"
}

// looksLikeRecordID reports whether s looks like a Cloudflare DNS record ID
// (32 hexadecimal characters).
func looksLikeRecordID(s string) bool {
	if len(s) != 32 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

// resolveRecordArg maps a record argument (a 32-hex Cloudflare record ID or
// a record name) to the matching record. Ambiguous names return an error
// listing the candidates.
func (b *TelegramBot) resolveRecordArg(zoneID, arg string) (models.DNSRecord, error) {
	records, err := b.cf.ListDNSRecords(zoneID, "", "")
	if err != nil {
		return models.DNSRecord{}, err
	}
	if looksLikeRecordID(arg) {
		for _, r := range records {
			if r.ID == arg {
				return r, nil
			}
		}
		return models.DNSRecord{}, fmt.Errorf("未在该区域找到记录 ID")
	}
	var matches []models.DNSRecord
	for _, r := range records {
		if recordNameMatches(r.Name, arg) {
			matches = append(matches, r)
		}
	}
	switch len(matches) {
	case 0:
		return models.DNSRecord{}, fmt.Errorf("未找到名称为 %q 的记录", arg)
	case 1:
		return matches[0], nil
	default:
		var sb strings.Builder
		fmt.Fprintf(&sb, "名称 %q 有多条记录，请用 /DNS详情 获取 RecordID 后重试：\n", arg)
		for _, r := range matches {
			fmt.Fprintf(&sb, "• %s %s → %s\n  ID: %s\n", r.Type, r.Name, r.Content, r.ID)
		}
		return models.DNSRecord{}, fmt.Errorf("%s", strings.TrimSuffix(sb.String(), "\n"))
	}
}

func isRecordType(s string) bool {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "A", "AAAA", "CNAME", "TXT", "MX":
		return true
	}
	return false
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// parseDNSWrite parses DNS write arguments: [区域可选] [类型] [名称] [内容]
// [TTL可选] [代理可选] [MX优先级]. The zone may be omitted (empty string is
// returned and the caller falls back to the selected zone). TTL defaults to
// auto and proxy defaults to on; TXT/MX never proxy and explicit "on" for
// them is rejected.
func parseDNSWrite(args []string) (string, models.DNSRecordRequest, error) {
	if len(args) < 3 {
		return "", models.DNSRecordRequest{}, fmt.Errorf("参数不足")
	}
	i := 0
	zone := ""
	if !isRecordType(args[0]) {
		zone = args[0]
		i = 1
	}
	rest := args[i:]
	if len(rest) < 3 {
		return "", models.DNSRecordRequest{}, fmt.Errorf("参数不足")
	}
	typ := strings.ToUpper(rest[0])
	if !isRecordType(typ) {
		return "", models.DNSRecordRequest{}, fmt.Errorf("仅支持 A/AAAA/CNAME/TXT/MX")
	}
	p := models.DNSRecordRequest{Type: typ, Name: rest[1], Content: rest[2], TTL: 1, Proxied: true}
	extra := rest[3:]
	// MX 优先级：类型为 MX 且末尾为纯数字时，末位数字为优先级
	if typ == "MX" && len(extra) > 0 && isNumeric(extra[len(extra)-1]) {
		v, e := strconv.Atoi(extra[len(extra)-1])
		if e != nil || v < 0 || v > 65535 {
			return "", p, fmt.Errorf("MX 优先级必须为 0-65535")
		}
		p.Priority = &v
		extra = extra[:len(extra)-1]
	}
	// 代理：末尾 on/off，缺省 on
	proxyExplicit := false
	if len(extra) > 0 && (strings.EqualFold(extra[len(extra)-1], "on") || strings.EqualFold(extra[len(extra)-1], "off")) {
		p.Proxied = strings.EqualFold(extra[len(extra)-1], "on")
		proxyExplicit = true
		extra = extra[:len(extra)-1]
	}
	// TTL：末尾 auto 或数字，缺省 auto
	if len(extra) > 0 {
		last := extra[len(extra)-1]
		if strings.EqualFold(last, "auto") {
			p.TTL = 1
		} else if isNumeric(last) {
			v, e := strconv.Atoi(last)
			if e != nil || v < 60 || v > 86400 {
				return "", p, fmt.Errorf("TTL 必须为 auto 或 60-86400")
			}
			p.TTL = v
		} else {
			return "", p, fmt.Errorf("无法识别的参数: %s", last)
		}
		extra = extra[:len(extra)-1]
	}
	if len(extra) > 0 {
		return "", p, fmt.Errorf("参数过多")
	}
	if typ == "TXT" || typ == "MX" {
		if proxyExplicit && p.Proxied {
			return "", p, fmt.Errorf("%s 记录不支持 Cloudflare 代理", typ)
		}
		p.Proxied = false
	}
	return zone, p, nil
}

func (b *TelegramBot) handleDNSWriteCommand(cfg models.Config, chatID int64, recordArg string, args []string) {
	zoneArg, p, err := parseDNSWrite(args)
	if err != nil {
		b.sendMessage(cfg, chatID, fmt.Sprintf("%s\n%s", err, dnsWriteUsage(recordArg != "")))
		return
	}
	if zoneArg == "" {
		if cfg.SelectedZoneID == "" {
			b.sendMessage(cfg, chatID, "❌ 未选择区域，请先 /选择区域 [域名或ZoneID]\n或带区域使用: /DNS添加 kukie.cn A www 1.2.3.4")
			return
		}
		zoneArg = cfg.SelectedZoneID
	}
	zoneID, zoneName, err := b.resolveZoneArg(zoneArg)
	if err != nil {
		b.sendMessage(cfg, chatID, "❌ "+err.Error())
		return
	}
	if recordArg == "" {
		_, err = b.cf.CreateDNSRecord(zoneID, p)
	} else {
		rec, rerr := b.resolveRecordArg(zoneID, recordArg)
		if rerr != nil {
			b.sendMessage(cfg, chatID, "❌ "+rerr.Error())
			return
		}
		_, err = b.cf.UpdateDNSRecord(zoneID, rec.ID, p)
	}
	if err != nil {
		b.sendMessage(cfg, chatID, fmt.Sprintf("❌ DNS 操作失败: %s", err))
		return
	}
	action := "已添加"
	if recordArg != "" {
		action = "已修改"
	}
	b.sendMessage(cfg, chatID, fmt.Sprintf("✅ %s: %s %s → %s（%s）", action, p.Type, p.Name, p.Content, zoneName))
}

func (b *TelegramBot) handleDNSDeleteRequest(cfg models.Config, chatID, userID int64, zoneArg, recordArg string) {
	zone, zoneName, err := b.resolveZoneArg(zoneArg)
	if err != nil {
		b.sendMessage(cfg, chatID, "❌ "+err.Error())
		return
	}
	target, err := b.resolveRecordArg(zone, recordArg)
	if err != nil {
		b.sendMessage(cfg, chatID, "❌ "+err.Error())
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
	b.sendMessage(cfg, chatID, fmt.Sprintf("⚠️ 将删除: %s（%s）\n\n5 分钟内发送: /确认删除 %s", label, zoneName, code))
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
