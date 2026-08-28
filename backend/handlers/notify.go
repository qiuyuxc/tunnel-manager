package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

const notifyTGTokenPurpose = "notify-tg-token"

// NotifyHandler implements per-user notification preferences and delivery.
type NotifyHandler struct {
	store         *store.Store
	encryptionKey []byte
	notifier      *services.Notifier
}

// NewNotifyHandler builds the user notification handler.
func NewNotifyHandler(st *store.Store, encryptionKey []byte, notifier *services.Notifier) *NotifyHandler {
	return &NotifyHandler{store: st, encryptionKey: append([]byte(nil), encryptionKey...), notifier: notifier}
}

// requestUserID resolves the account for a request: the session identity, or
// the administrator account when authenticated with the static API key.
func (h *NotifyHandler) requestUserID(r *http.Request) string {
	if user := SessionUser(r); user != nil && user.ID != "" {
		return user.ID
	}
	return h.store.AdminUserID()
}

// GetSettings handles GET /api/notify/settings.
func (h *NotifyHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	user := SessionUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	view, ok := h.store.GetUserNotifySettings(h.requestUserID(r))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// SaveSettings handles PUT /api/notify/settings.
func (h *NotifyHandler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	var req models.SaveNotifySettingsRequest
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	user := SessionUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	userID := h.requestUserID(r)
	if len(req.TGBotToken) > 200 || len(req.TGNotifyChatID) > 64 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Telegram 配置长度无效"})
		return
	}
	seen := map[string]bool{}
	for _, channel := range req.Channels {
		if channel != models.NotifyChannelEmail && channel != models.NotifyChannelTelegram {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "通知渠道无效"})
			return
		}
		if seen[channel] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "通知渠道重复"})
			return
		}
		seen[channel] = true
	}
	emails := strings.TrimSpace(req.Emails)
	for _, email := range splitNotifyEmailsRaw(emails) {
		if !validEmail(email) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "收件邮箱格式不正确: " + email})
			return
		}
	}
	chatID := strings.TrimSpace(req.TGNotifyChatID)
	if chatID != "" {
		if _, err := strconv.ParseInt(chatID, 10, 64); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Telegram Chat ID 需为数字"})
			return
		}
	}
	events := map[string]bool{}
	for _, event := range models.AllNotifyEvents {
		events[event] = req.Events[event]
	}
	tokenEncrypted := h.store.GetUserPrefs(userID).TGBotTokenEncrypted
	if req.TGBotToken != "" {
		encrypted, err := auth.EncryptSecret(h.encryptionKey, notifyTGTokenPurpose, []byte(req.TGBotToken))
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "加密 Telegram Token 失败"})
			return
		}
		tokenEncrypted = encrypted
	}
	if err := h.store.SetUserNotifySettings(userID, req.Channels, events, emails, tokenEncrypted, chatID); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "保存通知设置失败"})
		return
	}
	h.GetSettings(w, r)
}

// ReuseFromTelegram copies the remote-control bot token into the
// notification slot, so one bot can power both features.
func (h *NotifyHandler) ReuseFromTelegram(w http.ResponseWriter, r *http.Request) {
	user := SessionUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if err := h.store.ReuseTokenForNotify(h.requestUserID(r)); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.GetSettings(w, r)
}

// TestNotify handles POST /api/notify/test: sends a probe message through
// the account's configured channels.
func (h *NotifyHandler) TestNotify(w http.ResponseWriter, r *http.Request) {
	user := SessionUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if h.notifier == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "通知服务未初始化"})
		return
	}
	if err := h.notifier.SendTest(user.ID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "测试通知已发送"})
}

// splitNotifyEmailsRaw splits the multi-recipient field on newlines and
// commas, dropping empty entries.
func splitNotifyEmailsRaw(raw string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == ',' || r == ';' }) {
		if email := strings.TrimSpace(part); email != "" {
			out = append(out, email)
		}
	}
	return out
}
