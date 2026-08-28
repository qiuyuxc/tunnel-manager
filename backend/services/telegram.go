package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"tunnel-manager/models"
	"tunnel-manager/store"
)

// ---- wire types ----

type tgUpdate struct {
	UpdateID int64      `json:"update_id"`
	Message  *tgMessage `json:"message"`
}

type tgMessage struct {
	Text string `json:"text"`
	Chat struct {
		ID int64 `json:"id"`
	} `json:"chat"`
	From struct {
		ID int64 `json:"id"`
	} `json:"from"`
}

type tgGetMeResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		Username string `json:"username"`
	} `json:"result"`
	Description string `json:"description"`
}

type tgGenericResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

// ---- status ----

// BotStatus is the public status snapshot
type BotStatus struct {
	Enabled      bool   `json:"enabled"`
	Running      bool   `json:"running"`
	Mode         string `json:"mode"`
	BotUsername  string `json:"bot_username"`
	LastError    string `json:"last_error"`
	LastUpdateAt string `json:"last_update_at"`
}

// ---- bot ----

// TelegramBot runs the Telegram bot logic
type dnsDeleteConfirmation struct {
	ChatID  int64
	UserID  int64
	ZoneID  string
	Record  models.DNSRecord
	Label   string
	Expires time.Time
}

type TelegramBot struct {
	store  *store.Store
	cf     *CloudflareClient
	domain *DomainService

	mu           sync.Mutex
	cancel       context.CancelFunc
	running      bool
	mode         string
	botUsername  string
	lastError    string
	lastUpdateAt time.Time
	lastUpdateID int64
	httpClient   *http.Client

	confirmMu     sync.Mutex
	confirmations map[string]dnsDeleteConfirmation
}

// NewTelegramBot creates a new TelegramBot
func NewTelegramBot(st *store.Store, cf *CloudflareClient, ds *DomainService) *TelegramBot {
	return &TelegramBot{
		store:         st,
		cf:            cf,
		domain:        ds,
		httpClient:    &http.Client{Timeout: 40 * time.Second},
		confirmations: make(map[string]dnsDeleteConfirmation),
	}
}

// Start validates config and starts the bot
func (b *TelegramBot) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	cfg := b.store.GetConfig()
	if cfg.TGBotToken == "" {
		return fmt.Errorf("Bot Token 未设置")
	}
	if cfg.TGAdminIDs == "" {
		return fmt.Errorf("管理员 TG ID 未设置")
	}

	username, err := b.getMe(cfg)
	if err != nil {
		return fmt.Errorf("验证 Token 失败: %w", err)
	}
	b.botUsername = username
	b.mode = cfg.TGMode
	if b.mode == "" {
		b.mode = "polling"
	}

	if b.botUsername == "" {
		return fmt.Errorf("无法获取 Bot 信息，请检查 Token")
	}

	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	b.running = true

	if err := b.setCommands(cfg); err != nil {
		log.Printf("[telegram] set commands failed: %v", err)
	}

	if b.mode == "webhook" {
		if cfg.TGWebhookSecret == "" {
			secret := generateRandomHex(32)
			b.store.SetTelegramWebhookSecret(secret)
			cfg.TGWebhookSecret = secret
		}
		if err := b.setWebhook(cfg); err != nil {
			b.running = false
			cancel()
			return fmt.Errorf("注册 Webhook 失败: %w", err)
		}
		log.Printf("[telegram] webhook registered, bot @%s", b.botUsername)
	} else {
		b.deleteWebhook(cfg)
		go b.pollLoop(ctx, cfg)
		log.Printf("[telegram] polling started, bot @%s", b.botUsername)
	}

	b.lastError = ""
	return nil
}

// Stop stops the bot
func (b *TelegramBot) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.cancel != nil {
		b.cancel()
		b.cancel = nil
	}
	if b.mode == "webhook" {
		cfg := b.store.GetConfig()
		b.deleteWebhook(cfg)
	}
	b.running = false
	b.botUsername = ""
	log.Printf("[telegram] bot stopped")
}

// Status returns current bot status
func (b *TelegramBot) Status() BotStatus {
	b.mu.Lock()
	defer b.mu.Unlock()

	cfg := b.store.GetConfig()
	lastAt := ""
	if !b.lastUpdateAt.IsZero() {
		lastAt = b.lastUpdateAt.Format(time.RFC3339)
	}
	return BotStatus{
		Enabled:      cfg.TGBotEnabled,
		Running:      b.running,
		Mode:         b.mode,
		BotUsername:  b.botUsername,
		LastError:    b.lastError,
		LastUpdateAt: lastAt,
	}
}

// ---- polling ----

func (b *TelegramBot) pollLoop(ctx context.Context, cfg models.Config) {
	defer func() { log.Printf("[telegram] poll loop exited") }()

	for ctx.Err() == nil {
		updates, err := b.getUpdates(ctx, cfg, b.lastUpdateID+1)
		if err != nil {
			b.mu.Lock()
			b.lastError = err.Error()
			b.mu.Unlock()
			log.Printf("[telegram] poll error: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		b.mu.Lock()
		b.lastError = ""
		b.mu.Unlock()

		for _, u := range updates {
			b.lastUpdateID = u.UpdateID
			b.lastUpdateAt = time.Now()
			b.handleUpdate(u)
		}
	}
}

func (b *TelegramBot) getUpdates(ctx context.Context, cfg models.Config, offset int64) ([]tgUpdate, error) {
	baseURL := b.apiBase(cfg)
	url := fmt.Sprintf("%s/bot%s/getUpdates?timeout=10&offset=%d",
		baseURL, cfg.TGBotToken, offset)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OK     bool       `json:"ok"`
		Result []tgUpdate `json:"result"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		bodyStr := string(raw)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500]
		}
		log.Printf("[telegram] getUpdates decode error: %v, status=%d, body=%q", err, resp.StatusCode, bodyStr)
		return nil, err
	}
	if !result.OK {
		return nil, fmt.Errorf("getUpdates returned not ok")
	}
	return result.Result, nil
}

// ---- command dispatch ----

func (b *TelegramBot) handleUpdate(u tgUpdate) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[telegram] panic in handleUpdate: %v", r)
		}
	}()

	if u.Message == nil || u.Message.Chat.ID == 0 || u.Message.Text == "" {
		return
	}

	chatID := u.Message.Chat.ID
	userID := u.Message.From.ID
	cfg := b.store.GetConfig()

	// Auth check
	adminIDs := strings.Split(cfg.TGAdminIDs, ",")
	authorized := false
	for _, id := range adminIDs {
		if strings.TrimSpace(id) == strconv.FormatInt(userID, 10) {
			authorized = true
			break
		}
	}
	if !authorized {
		b.sendMessage(cfg, chatID, fmt.Sprintf("⛔ 拒绝访问\n身份校验失败：您的 TG ID [%d] 未授权。", userID))
		return
	}

	text := strings.TrimSpace(u.Message.Text)
	args, parseErr := splitCommandArgs(text)
	if parseErr != nil {
		b.sendMessage(cfg, chatID, "❌ 命令解析失败: "+parseErr.Error())
		return
	}
	if len(args) == 0 {
		return
	}
	command, addressedToThisBot := telegramCommandForBot(args[0], b.botUsername)
	if !addressedToThisBot {
		return
	}

	switch command {
	case "/start", "/help":
		b.handleHelp(cfg, chatID)
	case "/当前配置", "/config":
		b.handleCurrentConfig(cfg, chatID)
	case "/列出隧道", "/tunnels":
		b.handleListTunnels(cfg, chatID)
	case "/选择隧道":
		if len(args) < 2 {
			b.sendMessage(cfg, chatID, "❌ 请输入隧道名称或ID。例如: /选择隧道 haxvps-2")
		} else {
			b.handleSelectTunnel(chatID, args[1])
		}
	case "/转发":
		if len(args) < 2 {
			b.sendMessage(cfg, chatID, "❌ 请输入完整的本地服务URL。例如: /转发 http://localhost:3000")
		} else {
			b.handleSetService(chatID, args[1])
		}
	case "/全局优选":
		if len(args) < 2 {
			b.sendMessage(cfg, chatID, "❌ 请指定你的自定义优选 CNAME。例如: /全局优选 cdn.kukie.cn")
		} else {
			b.handleSetPreferredCNAME(chatID, args[1])
		}
	case "/设置回退源":
		if len(args) < 2 {
			b.sendMessage(cfg, chatID, "❌ 请指定回退源域名。例如: /设置回退源 fallback.169977.xyz")
		} else {
			b.handleSetFallback(cfg, chatID, args[1])
		}
	case "/绑定域名":
		if len(args) < 3 {
			b.sendMessage(cfg, chatID, "❌ 参数不足。请按顺序输入: /绑定域名 [访问域名] [辅助域名]")
		} else {
			b.handleDomainBinding(cfg, chatID, args[1], args[2])
		}

	case "/直连域名", "/bind_direct":
		if len(args) < 2 {
			b.sendMessage(cfg, chatID, "❌ 参数不足。用法: /直连域名 [主域名]")
		} else {
			b.handleDomainBindingMode(cfg, chatID, BindingModeSimple, args[1], "", "")
		}
	case "/优选绑定", "/bind_preferred":
		if len(args) < 3 {
			b.sendMessage(cfg, chatID, "❌ 参数不足。用法: /优选绑定 [主域名] [辅助域名] [可选优选CNAME]")
		} else {
			preferred := ""
			if len(args) > 3 {
				preferred = args[3]
			}
			b.handleDomainBindingMode(cfg, chatID, BindingModePreferred, args[1], args[2], preferred)
		}
	case "/列出区域", "/zones":
		b.handleListZones(cfg, chatID)
	case "/选择区域", "/select_zone":
		if len(args) < 2 {
			b.sendMessage(cfg, chatID, "❌ 用法: /选择区域 [域名或ZoneID]\n例如: /选择区域 kukie.cn")
		} else {
			b.handleSelectZone(cfg, chatID, args[1])
		}
	case "/DNS列表", "/dns_list":
		if len(args) < 2 {
			if cfg.SelectedZoneID != "" {
				b.handleListDNS(cfg, chatID, cfg.SelectedZoneID, "", "")
			} else {
				b.sendMessage(cfg, chatID, "❌ 用法: /DNS列表 [区域可选] [类型可选] [名称可选]\n未选择区域时需带区域，例如: /DNS列表 kukie.cn A")
			}
		} else if isRecordType(args[1]) {
			if cfg.SelectedZoneID == "" {
				b.sendMessage(cfg, chatID, "❌ 未选择区域，请先 /选择区域 [域名]\n或带区域使用: /DNS列表 kukie.cn A")
			} else {
				name := ""
				if len(args) > 2 {
					name = args[2]
				}
				b.handleListDNS(cfg, chatID, cfg.SelectedZoneID, args[1], name)
			}
		} else {
			recordType, name := "", ""
			if len(args) > 2 {
				recordType = args[2]
			}
			if len(args) > 3 {
				name = args[3]
			}
			b.handleListDNS(cfg, chatID, args[1], recordType, name)
		}
	case "/DNS详情", "/dns_detail":
		switch len(args) {
		case 1:
			b.sendMessage(cfg, chatID, "❌ 用法: /DNS详情 [完整主机名]\n例如: /DNS详情 bbs.kukie.cn\n或选定区域后: /DNS详情 bbs")
		case 2:
			b.handleDNSDetail(cfg, chatID, args[1], args[1])
		default:
			b.handleDNSDetail(cfg, chatID, args[1], args[2])
		}
	case "/DNS添加", "/dns_add":
		if len(args) < 2 {
			b.sendMessage(cfg, chatID, dnsWriteUsage(false))
		} else if isRecordType(args[1]) {
			if cfg.SelectedZoneID == "" {
				b.sendMessage(cfg, chatID, "❌ 未选择区域，请先 /选择区域 [域名]\n或带区域使用: /DNS添加 kukie.cn A www 1.2.3.4")
			} else {
				b.handleDNSWriteCommand(cfg, chatID, "", append([]string{cfg.SelectedZoneID}, args[1:]...))
			}
		} else {
			b.handleDNSWriteCommand(cfg, chatID, "", args[1:])
		}
	case "/DNS修改", "/dns_update":
		if len(args) < 3 {
			b.sendMessage(cfg, chatID, dnsWriteUsage(true))
		} else if isRecordType(args[2]) {
			// 无区域: /DNS修改 [记录名或ID] [类型] [新内容] [TTL可选] [代理可选]
			if cfg.SelectedZoneID == "" {
				b.sendMessage(cfg, chatID, "❌ 未选择区域，请先 /选择区域 [域名]\n或带区域使用: /DNS修改 kukie.cn bbs CNAME saas.com")
			} else {
				zoneForParse := cfg.SelectedZoneID
				if strings.Contains(args[1], ".") {
					zoneForParse = args[1]
				}
				updateArgs := append([]string{zoneForParse, args[2], args[1]}, args[3:]...)
				b.handleDNSWriteCommand(cfg, chatID, args[1], updateArgs)
			}
		} else if len(args) >= 4 && isRecordType(args[3]) {
			// 带区域: /DNS修改 [区域] [记录名或ID] [类型] [新内容] [TTL可选] [代理可选]
			updateArgs := append([]string{args[1], args[3], args[2]}, args[4:]...)
			b.handleDNSWriteCommand(cfg, chatID, args[2], updateArgs)
		} else {
			b.sendMessage(cfg, chatID, dnsWriteUsage(true))
		}
	case "/DNS删除", "/dns_delete":
		switch len(args) {
		case 1:
			b.sendMessage(cfg, chatID, "❌ 用法: /DNS删除 [区域可选] [记录名或RecordID]\n例如: /DNS删除 kukie.cn bbs 或 /DNS删除 bbs")
		case 2:
			if strings.Contains(args[1], ".") {
				b.handleDNSDeleteRequest(cfg, chatID, userID, args[1], args[1])
			} else if cfg.SelectedZoneID != "" {
				b.handleDNSDeleteRequest(cfg, chatID, userID, cfg.SelectedZoneID, args[1])
			} else {
				b.sendMessage(cfg, chatID, "❌ 未选择区域，请先 /选择区域 [域名]\n或带区域使用: /DNS删除 kukie.cn bbs")
			}
		default:
			b.handleDNSDeleteRequest(cfg, chatID, userID, args[1], args[2])
		}
	case "/确认删除", "/dns_confirm":
		if len(args) < 2 {
			b.sendMessage(cfg, chatID, "❌ 用法: /确认删除 [确认码]")
		} else {
			b.handleDNSDeleteConfirm(cfg, chatID, userID, strings.ToLower(args[1]))
		}
	}
}

// ---- command implementations ----

func (b *TelegramBot) handleHelp(cfg models.Config, chatID int64) {
	msg1 := strings.Join([]string{
		"🤖 SaaS 模块化管理助手",
		"",
		"📖 基础配置指令",
		"• /全局优选 [域名]",
		"• /设置回退源 [辅助域名]",
		"• /选择区域 [域名或ZoneID]（DNS 命令可省略区域）",
		"",
		"⚙️ 隧道与转发指令",
		"• /列出隧道",
		"• /选择隧道 [名称或ID]",
		"• /转发 [本地地址]",
	}, "\n")

	msg2 := strings.Join([]string{
		"🌐 域名绑定",
		"• /直连域名 [主域名]",
		"• /优选绑定 [主域名] [辅助域名] [可选优选CNAME]",
		"• /绑定域名 [主域名] [辅助域名]（兼容旧命令）",
		"",
		"🧭 DNS 管理（先 /选择区域 可省略区域）",
		"• /列出区域",
		"• /选择区域 [域名或ZoneID]",
		"• /DNS列表 [区域可选] [类型可选] [名称可选]",
		"• /DNS详情 [区域可选] [记录名]（或直接传完整主机名）",
		"• /DNS添加 [区域可选] [类型] [名称] [内容] [TTL可选] [代理可选]",
		"• /DNS修改 [区域可选] [记录名或ID] [类型] [新内容] [TTL可选] [代理可选]",
		"• /DNS删除 [区域可选] [记录名或ID]",
		"（TTL 缺省 auto，代理缺省 on；TXT/MX 自动关代理）",
		"",
		"🔍 状态查询",
		"• /当前配置",
		"",
		"💡 提示：本 Bot 与面板共享同一配置。",
	}, "\n")

	b.sendMessage(cfg, chatID, msg1)
	b.sendMessage(cfg, chatID, msg2)
}

func (b *TelegramBot) handleCurrentConfig(cfg models.Config, chatID int64) {
	cfg = b.store.GetConfig()

	globalPref := cfg.PreferredCNAME
	if globalPref == "" {
		globalPref = "cf.090227.xyz"
	}

	var out strings.Builder
	out.WriteString("⚙️ 当前配置\n\n")
	if cfg.TunnelID != "" {
		fmt.Fprintf(&out, "隧道: %s\n", cfg.TunnelName)
	} else {
		out.WriteString("隧道: 未锁定\n")
	}
	if cfg.ServiceURL != "" {
		fmt.Fprintf(&out, "转发: %s\n", cfg.ServiceURL)
	} else {
		out.WriteString("转发: 未设置\n")
	}
	fmt.Fprintf(&out, "优选: %s\n", globalPref)
	if cfg.SelectedZoneID != "" {
		fmt.Fprintf(&out, "区域: %s\n", cfg.SelectedZoneName)
	}
	out.WriteString("\n下一步：\n")
	switch {
	case cfg.TunnelID == "":
		out.WriteString("• /列出隧道 查看可用隧道\n• /选择隧道 [名称或ID] 锁定")
	case cfg.ServiceURL == "":
		out.WriteString("• /转发 [本地服务地址] 设置转发目标")
	default:
		out.WriteString("配置已就绪，可直接绑定域名：\n• /直连域名 [主域名]\n• /优选绑定 [主域名] [辅助域名]")
	}
	b.sendMessage(cfg, chatID, out.String())
}

func (b *TelegramBot) handleListTunnels(cfg models.Config, chatID int64) {
	tunnels, err := b.cf.ListTunnels()
	if err != nil {
		b.sendMessage(cfg, chatID, fmt.Sprintf("❌ 获取隧道列表失败: %s", err.Error()))
		return
	}

	if len(tunnels) == 0 {
		b.sendMessage(cfg, chatID, "账户下没有隧道。")
		return
	}
	limit := len(tunnels)
	if limit > 15 {
		limit = 15
	}
	var out strings.Builder
	out.WriteString("🚇 可用隧道\n\n")
	for i, t := range tunnels[:limit] {
		fmt.Fprintf(&out, "%d. %s · %s\n", i+1, t.Name, t.Status)
	}
	if len(tunnels) > limit {
		fmt.Fprintf(&out, "\n共 %d 个，仅显示前 %d 个。", len(tunnels), limit)
	}
	out.WriteString("\n\n选择: /选择隧道 [名称或ID]")
	b.sendMessage(cfg, chatID, out.String())
}

func (b *TelegramBot) handleSelectTunnel(chatID int64, arg string) {
	t, err := b.resolveTunnelArg(arg)
	if err != nil {
		cfg := b.store.GetConfig()
		b.sendMessage(cfg, chatID, "❌ "+err.Error())
		return
	}
	if err := b.store.SetUserTunnelSelection(b.store.AdminUserID(), t.ID, t.Name); err != nil {
		cfg := b.store.GetConfig()
		b.sendMessage(cfg, chatID, fmt.Sprintf("❌ 保存隧道选择失败: %s", err.Error()))
		return
	}
	cfg := b.store.GetConfig()
	b.sendMessage(cfg, chatID, fmt.Sprintf("✅ 已锁定隧道: %s\n\n下一步：设置本地转发地址。\n例如: /转发 http://localhost:3000", t.Name))
}

func (b *TelegramBot) handleSetService(chatID int64, url string) {
	b.store.SetServiceURL(url)
	cfg := b.store.GetConfig()
	b.sendMessage(cfg, chatID, fmt.Sprintf("✅ 转发源站已锁定: %s\n\n下一步：绑定域名。\n直连: /直连域名 [主域名]\n优选: /优选绑定 [主域名] [辅助域名]", url))
}

func (b *TelegramBot) handleSetPreferredCNAME(chatID int64, cname string) {
	b.store.SetPreferredCNAME(cname)
	cfg := b.store.GetConfig()
	b.sendMessage(cfg, chatID, fmt.Sprintf("🎯 全局优选 CNAME 已变更为: %s", cname))
}

func (b *TelegramBot) handleSetFallback(cfg models.Config, chatID int64, domain string) {
	if err := b.domain.SetFallbackOrigin(domain); err != nil {
		b.sendMessage(cfg, chatID, fmt.Sprintf("❌ 设置回退源失败: %s", err.Error()))
		return
	}
	b.sendMessage(cfg, chatID, fmt.Sprintf("✅ 已将 %s 设置为回退源", domain))
}

func (b *TelegramBot) handleDomainBinding(cfg models.Config, chatID int64, mainDomain, auxDomain string) {
	b.sendMessage(cfg, chatID, "⏳ 正在拉取隧道配置并全自动下发核心路由，请稍候...")

	_, preferredCNAME, err := b.domain.BindDomainWithConfiguredService(BindingModePreferred, mainDomain, auxDomain, "")
	if err != nil {
		b.sendMessage(cfg, chatID, fmt.Sprintf("❌ 绑定失败: %s", err.Error()))
		return
	}

	msg := fmt.Sprintf(
		"🎉 优选绑定配置成功！\n\n访问入口: %s\n内部回源: %s\n优选指向: %s\n\n请等待 1-2 分钟后尝试 HTTPS 访问。",
		mainDomain, auxDomain, preferredCNAME,
	)
	b.sendMessage(cfg, chatID, msg)
}

// ---- message sending ----

func (b *TelegramBot) sendMessage(cfg models.Config, chatID int64, text string) error {
	for _, chunk := range splitTelegramText(text, 3900) {
		if err := b.sendMessageChunk(cfg, chatID, chunk); err != nil {
			log.Printf("[telegram] sendMessage error: %v", err)
			return err
		}
	}
	return nil
}

func (b *TelegramBot) sendMessageChunk(cfg models.Config, chatID int64, text string) error {
	url := fmt.Sprintf("%s/bot%s/sendMessage", b.apiBase(cfg), cfg.TGBotToken)
	body, err := json.Marshal(map[string]interface{}{"chat_id": chatID, "text": text})
	if err != nil {
		return err
	}
	resp, err := b.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var result tgGenericResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("sendMessage failed: %s", result.Description)
	}
	return nil
}

// ---- webhook ----

// HandleWebhookUpdate processes a webhook update in a goroutine
func (b *TelegramBot) HandleWebhookUpdate(raw []byte) {
	var u tgUpdate
	if err := json.Unmarshal(raw, &u); err != nil {
		log.Printf("[telegram] webhook parse error: %v", err)
		return
	}
	go b.handleUpdate(u)
}

// VerifyWebhookSecret checks the incoming secret token header
func (b *TelegramBot) VerifyWebhookSecret(header string) bool {
	if header == "" {
		return false
	}
	cfg := b.store.GetConfig()
	return cfg.TGWebhookSecret != "" && header == cfg.TGWebhookSecret
}

// ---- test ----

// SendTestMessage sends a test message to all admin IDs
func (b *TelegramBot) SendTestMessage() error {
	cfg := b.store.GetConfig()
	if cfg.TGBotToken == "" {
		return fmt.Errorf("Bot Token 未设置")
	}
	if cfg.TGAdminIDs == "" {
		return fmt.Errorf("管理员 TG ID 未设置")
	}

	adminIDs := strings.Split(cfg.TGAdminIDs, ",")
	var lastErr error
	for _, idStr := range adminIDs {
		idStr = strings.TrimSpace(idStr)
		if idStr == "" {
			continue
		}
		chatID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			lastErr = fmt.Errorf("无效的 TG ID: %s", idStr)
			continue
		}
		if err := b.sendMessage(cfg, chatID, "✅ 面板连接测试成功！\n\n我是 Tunnel Manager Bot，配置已生效。发送 /help 查看可用指令。"); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// ---- TG API helpers ----

func (b *TelegramBot) apiBase(cfg models.Config) string {
	if cfg.TGApiEndpoint != "" {
		return strings.TrimRight(cfg.TGApiEndpoint, "/")
	}
	return "https://api.telegram.org"
}

func (b *TelegramBot) getMe(cfg models.Config) (string, error) {
	url := fmt.Sprintf("%s/bot%s/getMe", b.apiBase(cfg), cfg.TGBotToken)
	resp, err := b.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result tgGetMeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.OK {
		return "", fmt.Errorf("getMe failed: %s", result.Description)
	}
	return result.Result.Username, nil
}

func (b *TelegramBot) setCommands(cfg models.Config) error {
	commands := []map[string]string{
		{"command": "help", "description": "查看可用命令"},
		{"command": "config", "description": "查看当前 Tunnel 配置"},
		{"command": "tunnels", "description": "列出 Cloudflare Tunnel"},
		{"command": "bind_direct", "description": "简化模式绑定主域名"},
		{"command": "bind_preferred", "description": "优选模式绑定域名"},
		{"command": "zones", "description": "列出 Cloudflare Zone"},
		{"command": "select_zone", "description": "选定 DNS 区域"},
		{"command": "dns_list", "description": "查询 DNS 记录"},
		{"command": "dns_detail", "description": "查看 DNS 记录详情"},
		{"command": "dns_add", "description": "新增 DNS 记录"},
		{"command": "dns_update", "description": "修改 DNS 记录"},
		{"command": "dns_delete", "description": "申请删除 DNS 记录"},
	}
	body, err := json.Marshal(map[string]interface{}{"commands": commands})
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/bot%s/setMyCommands", b.apiBase(cfg), cfg.TGBotToken)
	resp, err := b.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var result tgGenericResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("setMyCommands failed: %s", result.Description)
	}
	return nil
}

func (b *TelegramBot) setWebhook(cfg models.Config) error {
	webhookURL := strings.TrimRight(cfg.TGWebhookURL, "/") + "/api/telegram/webhook"
	url := fmt.Sprintf("%s/bot%s/setWebhook", b.apiBase(cfg), cfg.TGBotToken)
	payload := map[string]interface{}{
		"url":             webhookURL,
		"secret_token":    cfg.TGWebhookSecret,
		"allowed_updates": []string{"message"},
	}
	body, _ := json.Marshal(payload)
	resp, err := b.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result tgGenericResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if !result.OK {
		return fmt.Errorf("setWebhook failed: %s", result.Description)
	}
	return nil
}

func (b *TelegramBot) deleteWebhook(cfg models.Config) {
	url := fmt.Sprintf("%s/bot%s/deleteWebhook?drop_pending_updates=true", b.apiBase(cfg), cfg.TGBotToken)
	resp, err := b.httpClient.Get(url)
	if err != nil {
		log.Printf("[telegram] deleteWebhook error: %v", err)
		return
	}
	resp.Body.Close()
}

// ---- util ----

func telegramCommandForBot(raw, username string) (string, bool) {
	parts := strings.SplitN(raw, "@", 2)
	if len(parts) == 2 && !strings.EqualFold(parts[1], username) {
		return "", false
	}
	return parts[0], true
}

func splitTelegramText(text string, limit int) []string {
	if limit <= 0 || len([]rune(text)) <= limit {
		return []string{text}
	}
	var chunks []string
	remaining := []rune(text)
	for len(remaining) > 0 {
		n := limit
		if len(remaining) < n {
			n = len(remaining)
		}
		cut := n
		for i := n - 1; i > n/2; i-- {
			if remaining[i] == '\n' {
				cut = i + 1
				break
			}
		}
		chunks = append(chunks, string(remaining[:cut]))
		remaining = remaining[cut:]
	}
	return chunks
}

func splitCommandArgs(text string) ([]string, error) {
	var args []string
	var current strings.Builder
	var quote rune
	started := false
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if quote != 0 {
			if r == quote {
				quote = 0
				started = true
				continue
			}
			if r == '\\' && i+1 < len(runes) && (runes[i+1] == quote || runes[i+1] == '\\') {
				i++
				current.WriteRune(runes[i])
				started = true
				continue
			}
			current.WriteRune(r)
			started = true
			continue
		}
		if r == '"' || r == '\'' {
			quote = r
			started = true
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' {
			if started {
				args = append(args, current.String())
				current.Reset()
				started = false
			}
			continue
		}
		current.WriteRune(r)
		started = true
	}
	if quote != 0 {
		return nil, fmt.Errorf("引号未闭合")
	}
	if started {
		args = append(args, current.String())
	}
	return args, nil
}

func generateRandomHex(length int) string {
	b := make([]byte, length/2)
	rand.Read(b)
	return hex.EncodeToString(b)
}
