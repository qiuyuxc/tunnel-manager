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
	token       string
	operatorIDs string
	apiEndpoint string
}

// UserTelegramManager runs one isolated Telegram bot per account. Every bot
// operates only on its owner's Cloudflare connection, tunnel selection and
// DNS permissions.
type UserTelegramManager struct {
	store         *store.Store
	cf            *CloudflareClient
	ds            *DomainService
	encryptionKey []byte

	mu   sync.Mutex
	bots map[string]*TelegramBot
	meta map[string]userBotMeta
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
	}
}

// MigrateLegacyAdminBot copies the legacy global Telegram bot configuration
// into the administrator's own per-user preferences on first run, so the
// previously configured bot keeps running without reconfiguration.
func (m *UserTelegramManager) MigrateLegacyAdminBot() {
	adminID := m.store.AdminUserID()
	prefs := m.store.GetUserPrefs(adminID)
	if prefs.TGRemoteTokenEncrypted != "" {
		return
	}
	cfg := m.store.GetConfig()
	if cfg.TGBotToken == "" {
		return
	}
	enc, err := auth.EncryptSecret(m.encryptionKey, notifyTGTokenPurpose, []byte(cfg.TGBotToken))
	if err != nil {
		log.Printf("[user-telegram] migrate admin bot token failed: %v", err)
		return
	}
	if err := m.store.SetUserRemoteSettings(adminID, cfg.TGBotEnabled, cfg.TGAdminIDs, enc); err != nil {
		log.Printf("[user-telegram] migrate admin bot preferences failed: %v", err)
		return
	}
	log.Printf("[user-telegram] migrated legacy admin bot to per-user preferences")
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
		desired[u.ID] = userBotMeta{token: string(token), operatorIDs: prefs.TGOperatorIDs, apiEndpoint: apiEndpoint}
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
		bot := NewUserTelegramBot(m.store, m.cf, m.ds, uid, meta.token, meta.operatorIDs)
		if err := bot.Start(); err != nil {
			log.Printf("[user-telegram] start bot for user %s failed: %v", uid, err)
			continue
		}
		m.bots[uid] = bot
		m.meta[uid] = meta
		log.Printf("[user-telegram] polling started for user %s, bot @%s", uid, bot.botUsername)
	}

	for uid, bot := range m.bots {
		if _, ok := desired[uid]; !ok {
			bot.Stop()
			delete(m.bots, uid)
			delete(m.meta, uid)
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
	return BotStatus{
		Enabled: prefs.TGRemoteEnabled,
		Running: false,
		Mode:    "polling",
	}
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
	}
}
