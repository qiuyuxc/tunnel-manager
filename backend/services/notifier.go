package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/store"
)

const (
	smtpPasswordPurpose = "smtp-password"
	notifyTGTokenPurpose = "notify-tg-token"
)

// Notifier delivers per-user notifications (login events, test messages)
// over the channels each account configured: SMTP email and/or the user's
// own Telegram bot. It is independent of the global admin Telegram bot.
type Notifier struct {
	store         *store.Store
	encryptionKey []byte
}

// NewNotifier builds the user notification delivery service.
func NewNotifier(st *store.Store, encryptionKey []byte) *Notifier {
	return &Notifier{store: st, encryptionKey: append([]byte(nil), encryptionKey...)}
}

// NotifyLogin sends a login notification when the account enabled it.
// Delivery happens asynchronously so it never slows the login flow.
func (n *Notifier) NotifyLogin(userID, username, remoteIP string) {
	if n == nil {
		return
	}
	prefs := n.store.GetUserPrefs(userID)
	if len(prefs.NotifyChannels) == 0 || !prefs.NotifyEvents[models.NotifyEventLogin] {
		return
	}
	text, htmlBody := LoginNotifyEmail(username, time.Now().Format("2006-01-02 15:04:05"), remoteIP)
	go n.deliver(prefs, "Tunnel Manager 登录通知", text, htmlBody)
}

// SendTest delivers a test message through the account's current channels.
func (n *Notifier) SendTest(userID string) error {
	if n == nil {
		return fmt.Errorf("通知服务未初始化")
	}
	prefs := n.store.GetUserPrefs(userID)
	if len(prefs.NotifyChannels) == 0 {
		return fmt.Errorf("尚未启用任何通知渠道")
	}
	text, htmlBody := NotifyTestEmail()
	n.deliver(prefs, "Tunnel Manager 测试通知", text, htmlBody)
	return nil
}

func (n *Notifier) deliver(prefs models.UserPrefs, subject, plainText, htmlText string) {
	for _, channel := range prefs.NotifyChannels {
		switch channel {
		case models.NotifyChannelEmail:
			n.sendEmails(prefs, subject, plainText, htmlText)
		case models.NotifyChannelTelegram:
			n.sendTelegram(prefs, plainText)
		}
	}
}

func (n *Notifier) sendEmails(prefs models.UserPrefs, subject, plainText, htmlText string) {
	recipients := splitNotifyEmails(prefs.NotifyEmails)
	if len(recipients) == 0 {
		return
	}
	settings := n.store.GetSMTPSettings()
	if !settings.Configured() {
		log.Printf("[notify] SMTP 未配置，跳过邮件通知")
		return
	}
	plain, err := auth.DecryptSecret(n.encryptionKey, smtpPasswordPurpose, settings.Password)
	if err != nil || plain == nil {
		log.Printf("[notify] 解密 SMTP 密码失败: %v", err)
		return
	}
	mailer := NewMailer(settings, string(plain))
	for _, email := range recipients {
		if err := mailer.Send(email, subject, plainText, htmlText); err != nil {
			log.Printf("[notify] 邮件通知发送失败 %s: %v", email, err)
		}
	}
}

func (n *Notifier) sendTelegram(prefs models.UserPrefs, text string) {
	chatID := strings.TrimSpace(prefs.TGNotifyChatID)
	if prefs.TGBotTokenEncrypted == "" || chatID == "" {
		return
	}
	token, err := auth.DecryptSecret(n.encryptionKey, notifyTGTokenPurpose, prefs.TGBotTokenEncrypted)
	if err != nil || token == nil {
		log.Printf("[notify] 解密 Telegram token 失败: %v", err)
		return
	}
	base := strings.TrimRight(n.store.GetConfig().TGApiEndpoint, "/")
	if base == "" {
		base = "https://api.telegram.org"
	}
	url := base + "/bot" + string(token) + "/sendMessage"
	body, err := json.Marshal(map[string]interface{}{"chat_id": chatID, "text": text})
	if err != nil {
		log.Printf("[notify] 编码 Telegram 消息失败: %v", err)
		return
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[notify] Telegram 通知发送失败: %v", err)
		return
	}
	defer resp.Body.Close()
	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("[notify] Telegram 通知响应解析失败: %v", err)
		return
	}
	if !result.OK {
		log.Printf("[notify] Telegram 通知发送失败: %s", result.Description)
	}
}

// splitNotifyEmails splits the multi-recipient field on newlines and commas.
func splitNotifyEmails(raw string) []string {
	seen := map[string]bool{}
	var out []string
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == ',' || r == ';' }) {
		email := strings.TrimSpace(part)
		if email == "" || seen[email] {
			continue
		}
		seen[email] = true
		out = append(out, email)
	}
	return out
}
