package services

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"tunnel-manager/auth"
	"tunnel-manager/store"
)

// userBotMeta records the settings a running per-user bot was started with,
// so Reconcile can detect changes and restart.
type userBotMeta struct {
	token         string
	operatorIDs   string
	apiEndpoint   string
	mode          string
	webhookURL    string
	webhookSecret string
}

// UserTelegramManager runs one isolated Telegram bot per account. Every bot
// operates only on its owner's Cloudflare connection, tunnel selection and
// DNS permissions.
type UserTelegramManager struct {
	store         *store.Store
	cf            *CloudflareClient
	ds            *DomainService
	encryptionKey []byte

	mu          sync.Mutex
	bots        map[string]*TelegramBot
	meta        map[string]userBotMeta
	startErrors map[string]string
}

// NewUserTelegramManager builds the per-user bot manager.
func NewUserTelegramManager(st *store.Store, cf *CloudflareClient, ds *DomainService, encryptionKey []byte) *UserTelegramManager {
	return &UserTelegramManager{
		store:         st,
		cf:            cf,
		ds:            ds,
		encryptionKey: append([]byte(nil), encryptionKey...),
		bots:          map[string]*TelegramBot{},
		meta:          map[string]userBotMeta{},
		startErrors:   map[string]string{},
	}
}

// MigrateLegacyAdminBot copies the legacy global Telegram bot configuration
// into the administrator's own per-user preferences, so the previously
// configured bot keeps running without reconfiguration. It also carries the
// legacy global webhook configuration over for deployments that already
// migrated the token (e.g. v2.2.0-test.2), so existing webhook users are not
// silently downgraded to long polling.
func (m *UserTelegramManager) MigrateLegacyAdminBot() {
	adminID := m.store.AdminUserID()
	prefs := m.store.GetUserPrefs(adminID)
	cfg := m.store.GetConfig()
	if cfg.TGBotToken == "" {
		return
	}

	// First boot after the multi-user split: copy everything.
	if prefs.TGRemoteTokenEncrypted == "" {
		enc, err := auth.EncryptSecret(m.encryptionKey, notifyTGTokenPurpose, []byte(cfg.TGBotToken))
		if err != nil {
			log.Printf("[user-telegram] migrate admin bot token failed: %v", err)
			return
		}
		mode := cfg.TGMode
		if mode == "" {
			mode = "polling"
		}
		if err := m.store.SetUserRemoteSettings(adminID, cfg.TGBotEnabled, cfg.TGAdminIDs, enc, mode, cfg.TGWebhookURL, cfg.TGWebhookSecret); err != nil {
			log.Printf("[user-telegram] migrate admin bot preferences failed: %v", err)
			return
		}
		log.Printf("[user-telegram] migrated legacy admin bot to per-user preferences (mode=%s)", mode)
		return
	}

	// Token already migrated: only fill still-empty per-user webhook settings
	// from the legacy global configuration, without touching the rest.
	if prefs.TGRemoteMode == "polling" && prefs.TGRemoteWebhookURL == "" && prefs.TGRemoteWebhookSecret == "" && cfg.TGMode == "webhook" && cfg.TGWebhookURL != "" {
		if err := m.store.SetUserRemoteSettings(adminID, prefs.TGRemoteEnabled, prefs.TGOperatorIDs, prefs.TGRemoteTokenEncrypted, "webhook", cfg.TGWebhookURL, cfg.TGWebhookSecret); err != nil {
			log.Printf("[user-telegram] migrate admin webhook preferences failed: %v", err)
			return
		}
		log.Printf("[user-telegram] migrated legacy admin webhook settings to per-user preferences")
	}
}

// Reconcile starts bots for accounts that enabled remote control and stops
// bots whose settings were disabled or changed.
func (m *UserTelegramManager) Reconcile() {
	m.mu.Lock()
	defer m.mu.Unlock()

	desired := map[string]userBotMeta{}
	apiEndpoint := m.store.GetConfig().TGApiEndpoint
	for _, u := range m.store.ListUsers() {
		prefs := m.store.GetUserPrefs(u.ID)
		if !prefs.TGRemoteEnabled || strings.TrimSpace(prefs.TGOperatorIDs) == "" || prefs.TGRemoteTokenEncrypted == "" {
			continue
		}
		token, err := auth.DecryptSecret(m.encryptionKey, notifyTGTokenPurpose, prefs.TGRemoteTokenEncrypted)
		if err != nil {
			log.Printf("[user-telegram] decrypt token for %s failed: %v", u.Username, err)
			continue
		}
		mode := prefs.TGRemoteMode
		if mode == "" {
			mode = "polling"
		}
		if mode == "webhook" && strings.TrimSpace(prefs.TGRemoteWebhookURL) == "" {
			log.Printf("[user-telegram] skip user %s: webhook mode without public HTTPS base URL", u.Username)
			continue
		}
		desired[u.ID] = userBotMeta{
			token:         string(token),
			operatorIDs:   prefs.TGOperatorIDs,
			apiEndpoint:   apiEndpoint,
			mode:          mode,
			webhookURL:    strings.TrimRight(strings.TrimSpace(prefs.TGRemoteWebhookURL), "/"),
			webhookSecret: prefs.TGRemoteWebhookSecret,
		}
	}

	for uid, meta := range desired {
		if bot, ok := m.bots[uid]; ok {
			if m.meta[uid] == meta {
				continue
			}
			bot.Stop()
			delete(m.bots, uid)
			log.Printf("[user-telegram] restarting bot for user %s", uid)
		}
		bot := NewUserTelegramBot(m.store, m.cf, m.ds, uid, meta.token, meta.operatorIDs, meta.mode, meta.webhookURL, meta.webhookSecret)
		if err := bot.Start(); err != nil {
			m.startErrors[uid] = err.Error()
			log.Printf("[user-telegram] start bot for user %s failed: %v", uid, err)
			continue
		}
		delete(m.startErrors, uid)
		m.bots[uid] = bot
		// Re-read the webhook secret: Start() may have just generated and
		// persisted one, so the next reconcile won't see a diff and restart.
		if prefs := m.store.GetUserPrefs(uid); prefs.TGRemoteWebhookSecret != "" {
			meta.webhookSecret = prefs.TGRemoteWebhookSecret
		}
		m.meta[uid] = meta
		log.Printf("[user-telegram] bot started for user %s (%s mode), bot @%s", uid, meta.mode, bot.botUsername)
	}

	for uid, bot := range m.bots {
		if _, ok := desired[uid]; !ok {
			bot.Stop()
			delete(m.bots, uid)
			delete(m.meta, uid)
			delete(m.startErrors, uid)
			log.Printf("[user-telegram] stopped bot for user %s", uid)
		}
	}
}

// Status returns the running status of one account's bot.
func (m *UserTelegramManager) Status(userID string) BotStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	if bot, ok := m.bots[userID]; ok {
		return bot.Status()
	}
	prefs := m.store.GetUserPrefs(userID)
	mode := prefs.TGRemoteMode
	if mode == "" {
		mode = "polling"
	}
	lastError := ""
	if prefs.TGRemoteEnabled {
		lastError = m.startErrors[userID]
	}
	return BotStatus{
		Enabled:   prefs.TGRemoteEnabled,
		Running:   false,
		Mode:      mode,
		LastError: lastError,
	}
}

// HandleWebhook verifies the secret token and dispatches a webhook update to
// the account's own running bot. Unknown bots and wrong secrets are rejected.
func (m *UserTelegramManager) HandleWebhook(userID, secret string, body []byte) error {
	m.mu.Lock()
	bot, ok := m.bots[userID]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("bot not found")
	}
	if !bot.VerifyWebhookSecret(secret) {
		return fmt.Errorf("invalid secret token")
	}
	bot.HandleWebhookUpdate(body)
	return nil
}

// SendTest sends a test message to one account's authorized Telegram IDs.
func (m *UserTelegramManager) SendTest(userID string) error {
	m.mu.Lock()
	bot, ok := m.bots[userID]
	m.mu.Unlock()
	if !ok {
		prefs := m.store.GetUserPrefs(userID)
		if !prefs.TGRemoteEnabled {
			return fmt.Errorf("请先启用远程控制")
		}
		if prefs.TGRemoteTokenEncrypted == "" || strings.TrimSpace(prefs.TGOperatorIDs) == "" {
			return fmt.Errorf("请先配置 Bot Token 和授权 TG ID")
		}
		return fmt.Errorf("Bot 未运行，请保存设置后重试")
	}
	return bot.SendTestMessage()
}

// StopAll stops every per-user bot.
func (m *UserTelegramManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for uid, bot := range m.bots {
		bot.Stop()
		delete(m.bots, uid)
		delete(m.meta, uid)
		delete(m.startErrors, uid)
	}
}
