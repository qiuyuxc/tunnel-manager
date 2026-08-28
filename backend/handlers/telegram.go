package handlers

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

// TelegramHandler implements per-user Telegram remote-control endpoints.
type TelegramHandler struct {
	store         *store.Store
	bot           *services.TelegramBot // legacy global bot (webhook route only)
	manager       *services.UserTelegramManager
	encryptionKey []byte
}

// NewTelegramHandler creates the per-user Telegram handler.
func NewTelegramHandler(st *store.Store, bot *services.TelegramBot, manager *services.UserTelegramManager, encryptionKey []byte) *TelegramHandler {
	return &TelegramHandler{
		store:         st,
		bot:           bot,
		manager:       manager,
		encryptionKey: append([]byte(nil), encryptionKey...),
	}
}

// requestUserID resolves the account for a request: the session identity, or
// the administrator account when authenticated with the static API key.
func (h *TelegramHandler) requestUserID(r *http.Request) string {
	if user := SessionUser(r); user != nil && user.ID != "" {
		return user.ID
	}
	return h.store.AdminUserID()
}

// GetSettings returns the current user's bot settings (token masked).
func (h *TelegramHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID := h.requestUserID(r)
	prefs := h.store.GetUserPrefs(userID)
	cfg := h.store.GetConfig()

	hint := ""
	if prefs.TGRemoteTokenEncrypted != "" {
		if plain, err := auth.DecryptSecret(h.encryptionKey, notifyTGTokenPurpose, prefs.TGRemoteTokenEncrypted); err == nil {
			hint = maskToken(string(plain))
		}
	}

	resp := models.TelegramSettingsResponse{
		Enabled:      prefs.TGRemoteEnabled,
		BotTokenSet:  prefs.TGRemoteTokenEncrypted != "",
		BotTokenHint: hint,
		AdminTGIDs:   prefs.TGOperatorIDs,
		Mode:         "polling",
		ApiEndpoint:  cfg.TGApiEndpoint,
		NotifyBotSet: prefs.TGBotTokenEncrypted != "",
	}
	writeJSON(w, http.StatusOK, resp)
}

// ReuseFromNotify copies the notification bot token into the remote-control
// slot, so one bot can power both features.
func (h *TelegramHandler) ReuseFromNotify(w http.ResponseWriter, r *http.Request) {
	userID := h.requestUserID(r)
	if err := h.store.ReuseTokenForRemote(userID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.manager.Reconcile()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "已复用通知的 Bot Token"})
}

// SaveSettings saves the current user's bot settings and restarts their bot.
func (h *TelegramHandler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	var req models.TelegramSettingsRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	userID := h.requestUserID(r)
	prefs := h.store.GetUserPrefs(userID)
	token := strings.TrimSpace(req.BotToken)

	if req.Enabled {
		if token == "" && prefs.TGRemoteTokenEncrypted == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "启用 Bot 需要先设置 Token"})
			return
		}
		if strings.TrimSpace(req.AdminTGIDs) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "授权 TG ID 不能为空"})
			return
		}
	}
	if msg := validateTGIDs(req.AdminTGIDs); msg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}

	encrypted := prefs.TGRemoteTokenEncrypted
	if token != "" {
		enc, err := auth.EncryptSecret(h.encryptionKey, notifyTGTokenPurpose, []byte(token))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "加密 Token 失败"})
			return
		}
		encrypted = enc
	}

	if err := h.store.SetUserRemoteSettings(userID, req.Enabled, strings.TrimSpace(req.AdminTGIDs), encrypted); err != nil {
		if err == store.ErrUserNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	h.manager.Reconcile()
	status := h.manager.Status(userID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"running": status.Running,
	})
}

// SaveAPIEndpoint updates the panel-wide Telegram Bot API endpoint used by
// every per-user bot. Administrator only; all bots restart to pick it up.
func (h *TelegramHandler) SaveAPIEndpoint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIEndpoint string `json:"api_endpoint"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	endpoint := strings.TrimRight(strings.TrimSpace(req.APIEndpoint), "/")
	if endpoint == "" || !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "API 端点需为 http(s):// 地址"})
		return
	}
	h.store.SetTelegramAPIEndpoint(endpoint)
	h.manager.Reconcile()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "api_endpoint": endpoint})
}

// GetStatus returns the current user's bot running status.
func (h *TelegramHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.manager.Status(h.requestUserID(r)))
}

// SendTest sends a test message to the current user's authorized TG IDs.
func (h *TelegramHandler) SendTest(w http.ResponseWriter, r *http.Request) {
	if err := h.manager.SendTest(h.requestUserID(r)); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "测试消息已发送"})
}

// Webhook handles incoming Telegram webhook updates for the legacy global bot
// (no auth middleware; verified via secret token).
func (h *TelegramHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	cfg := h.store.GetConfig()
	if !cfg.TGBotEnabled || cfg.TGMode != "webhook" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "bot not in webhook mode"})
		return
	}
	if !h.bot.Status().Running {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "legacy bot not running"})
		return
	}

	secret := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
	if !h.bot.VerifyWebhookSecret(secret) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid secret token"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "read body failed"})
		return
	}

	h.bot.HandleWebhookUpdate(body)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// validateTGIDs accepts a comma-separated list of numeric Telegram IDs.
func validateTGIDs(raw string) string {
	for _, part := range strings.Split(raw, ",") {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			return "授权 TG ID 格式不正确: " + id
		}
	}
	return ""
}

// maskToken returns a masked version of the bot token
func maskToken(token string) string {
	if token == "" {
		return ""
	}
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 || len(parts[1]) <= 4 {
		return parts[0] + ":****"
	}
	return parts[0] + ":" + strings.Repeat("*", len(parts[1])-4) + parts[1][len(parts[1])-4:]
}
