package handlers

import (
	"log"
	"net/http"
	"os"
	"strings"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

// ManagementHandler implements the single-administrator panel settings: the
// SMTP relay, Cloudflare OAuth client, encryption key and login hardening
// (passkey relying party, Turnstile, rate limits).
type ManagementHandler struct {
	store         *store.Store
	encryptionKey []byte
}

// NewManagementHandler creates the admin backend handler.
func NewManagementHandler(st *store.Store, encryptionKey []byte) *ManagementHandler {
	return &ManagementHandler{store: st, encryptionKey: append([]byte(nil), encryptionKey...)}
}

// ---------------------------------------------------------------------------
// Panel settings

// GetAppSettings handles GET /api/admin/settings.
func (h *ManagementHandler) GetAppSettings(w http.ResponseWriter, r *http.Request) {
	settings := h.store.GetAppSettings()
	// The panel only lets an administrator turn password sign-in off once one of
	// them can still get in with a passkey.
	passkeyAdminReady := false
	if count, err := h.store.CountAdminsWithPasskeys(); err != nil {
		log.Printf("count administrators with passkeys: %v", err)
	} else {
		passkeyAdminReady = count > 0
	}
	writeJSON(w, http.StatusOK, models.AppSettingsView{
		TurnstileEnabled:                    settings.TurnstileEnabled,
		TurnstileSiteKey:                    settings.TurnstileSiteKey,
		TurnstileHasSecret:                  settings.TurnstileSecret != "",
		ExperimentalFeatures:                settings.ExperimentalFeatures,
		PasswordLoginDisabled:               settings.PasswordLoginDisabled,
		PasskeyRPID:                         settings.PasskeyRPID,
		PasskeyOrigins:                      settings.PasskeyOrigins,
		PasskeyAdminReady:                   passkeyAdminReady,
		PasskeyAndroidPackage:               settings.PasskeyAndroidPackage,
		PasskeyAndroidFingerprints:          settings.PasskeyAndroidFingerprints,
		PasskeyAndroidFingerprintsEffective: settings.AndroidFingerprints(),
		RateLimitEnabled:                    !settings.RateLimitDisabled,
		RateLimitPerAccount:                 settings.RateLimitPerAccount,
		RateLimitPerIP:                      settings.RateLimitPerIP,
		RateLimitWindowMinutes:              settings.RateLimitWindowMinutes,
		RateLimitFamiliarMultiplier:         settings.RateLimitFamiliarMultiplier,
		RateLimitNotify:                     settings.RateLimitNotify,
	})
}

// UpdateAppSettings handles PUT /api/admin/settings.
func (h *ManagementHandler) UpdateAppSettings(w http.ResponseWriter, r *http.Request) {
	var req models.SaveAppSettingsRequest
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	stored := h.store.GetAppSettings()
	passwordLoginDisabled := stored.PasswordLoginDisabled
	if req.PasswordLoginDisabled != nil {
		if *req.PasswordLoginDisabled && !stored.PasswordLoginDisabled {
			ready, err := h.store.CountAdminsWithPasskeys()
			if err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "操作失败"})
				return
			}
			if ready == 0 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请先为至少一名启用的管理员绑定通行密钥，再禁用密码登录"})
				return
			}
		}
		passwordLoginDisabled = *req.PasswordLoginDisabled
	}
	// Every rate-limit field arrives as a pointer so a partial save (the
	// Turnstile form, say) leaves the limits alone instead of resetting them to
	// the zero value, which would silently mean "use the default".
	rateLimitDisabled := stored.RateLimitDisabled
	if req.RateLimitEnabled != nil {
		rateLimitDisabled = !*req.RateLimitEnabled
	}
	rateLimitPerAccount := stored.RateLimitPerAccount
	if req.RateLimitPerAccount != nil {
		rateLimitPerAccount = *req.RateLimitPerAccount
	}
	rateLimitPerIP := stored.RateLimitPerIP
	if req.RateLimitPerIP != nil {
		rateLimitPerIP = *req.RateLimitPerIP
	}
	rateLimitWindow := stored.RateLimitWindowMinutes
	if req.RateLimitWindowMinutes != nil {
		rateLimitWindow = *req.RateLimitWindowMinutes
	}
	rateLimitFamiliar := stored.RateLimitFamiliarMultiplier
	if req.RateLimitFamiliarMultiplier != nil {
		rateLimitFamiliar = *req.RateLimitFamiliarMultiplier
	}
	rateLimitNotify := stored.RateLimitNotify
	if req.RateLimitNotify != nil {
		rateLimitNotify = *req.RateLimitNotify
	}

	settings := models.AppSettings{
		TurnstileEnabled:            req.TurnstileEnabled,
		TurnstileSiteKey:            strings.TrimSpace(req.TurnstileSiteKey),
		ExperimentalFeatures:        req.ExperimentalFeatures,
		PasswordLoginDisabled:       passwordLoginDisabled,
		PasskeyRPID:                 strings.TrimSpace(req.PasskeyRPID),
		PasskeyOrigins:              strings.TrimSpace(req.PasskeyOrigins),
		PasskeyAndroidPackage:       strings.TrimSpace(req.PasskeyAndroidPackage),
		PasskeyAndroidFingerprints:  strings.TrimSpace(req.PasskeyAndroidFingerprints),
		RateLimitDisabled:           rateLimitDisabled,
		RateLimitPerAccount:         rateLimitPerAccount,
		RateLimitPerIP:              rateLimitPerIP,
		RateLimitWindowMinutes:      rateLimitWindow,
		RateLimitFamiliarMultiplier: rateLimitFamiliar,
		RateLimitNotify:             rateLimitNotify,
	}
	if settings.TurnstileEnabled && (settings.TurnstileSiteKey == "" || (req.TurnstileSecret == "" && stored.TurnstileSecret == "")) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "开启人机验证需要填写 Site Key 与 Secret Key"})
		return
	}
	// Refuse a relying party that contradicts the stored origins instead of
	// storing it and failing every passkey request later.
	if err := services.ValidatePasskeySettings(settings.PasskeyRPID, settings.PasskeyOrigins); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error() + "；依赖方 ID 与来源需要同时修改"})
		return
	}
	if req.TurnstileSecret != "" {
		encrypted, err := auth.EncryptSecret(h.encryptionKey, turnstileSecretPurpose, []byte(req.TurnstileSecret))
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "加密人机验证密钥失败"})
			return
		}
		settings.TurnstileSecret = encrypted
	} else {
		settings.TurnstileSecret = stored.TurnstileSecret
	}
	if err := h.store.SetAppSettings(settings); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "保存设置失败"})
		return
	}
	h.GetAppSettings(w, r)
}

// ---------------------------------------------------------------------------
// SMTP relay

// GetOAuthConfig handles GET /api/admin/oauth. The client secret is never
// returned, only whether one is stored.
func (h *ManagementHandler) GetOAuthConfig(w http.ResponseWriter, r *http.Request) {
	settings := h.store.GetOAuthSettings()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"client_id":         settings.ClientID,
		"has_client_secret": settings.ClientSecret != "",
		"redirect_uri":      settings.RedirectURI,
		"scopes":            settings.Scopes,
	})
}

// SaveOAuthConfig handles PUT /api/admin/oauth. A blank secret keeps the
// stored one.
func (h *ManagementHandler) SaveOAuthConfig(w http.ResponseWriter, r *http.Request) {
	var req models.SaveOAuthRequest
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	settings := h.store.GetOAuthSettings()
	if strings.TrimSpace(req.ClientID) == "" && strings.TrimSpace(req.ClientSecret) == "" && strings.TrimSpace(req.RedirectURI) == "" && strings.TrimSpace(req.Scopes) == "" {
		// Full reset: every field blank clears the stored client.
		if err := h.store.SetOAuthSettings(models.OAuthSettings{}); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "保存 OAuth 设置失败"})
			return
		}
		h.GetOAuthConfig(w, r)
		return
	}
	settings.ClientID = strings.TrimSpace(req.ClientID)
	if req.ClientSecret != "" {
		settings.ClientSecret = strings.TrimSpace(req.ClientSecret)
	}
	settings.RedirectURI = strings.TrimSpace(req.RedirectURI)
	settings.Scopes = strings.TrimSpace(req.Scopes)
	if err := h.store.SetOAuthSettings(settings); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "保存 OAuth 设置失败"})
		return
	}
	h.GetOAuthConfig(w, r)
}

// GetEncryptionKeyStatus handles GET /api/admin/encryption-key.
func (h *ManagementHandler) GetEncryptionKeyStatus(w http.ResponseWriter, r *http.Request) {
	source := "stored"
	if os.Getenv("APP_ENCRYPTION_KEY") != "" {
		source = "env"
	} else if h.store.GetEncryptionKeyRaw() == "" {
		source = "none"
	}
	writeJSON(w, http.StatusOK, map[string]string{"source": source})
}

// SaveEncryptionKey handles PUT /api/admin/encryption-key. Takes effect after
// a restart; changing the key invalidates previously encrypted payloads.
func (h *ManagementHandler) SaveEncryptionKey(w http.ResponseWriter, r *http.Request) {
	var req models.SaveEncryptionKeyRequest
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	key := strings.TrimSpace(req.Key)
	if _, err := auth.ParseEncryptionKey(key); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "密钥格式无效：需 32 字节随机数据的标准 base64 编码（44 个字符）"})
		return
	}
	if err := h.store.SetEncryptionKeyRaw(key); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "保存失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "已保存，重启服务后生效；更换密钥会使已保存的授权与 2FA 密文失效"})
}

// GetSMTP handles GET /api/admin/smtp.
func (h *ManagementHandler) GetSMTP(w http.ResponseWriter, r *http.Request) {
	settings := h.store.GetSMTPSettings()
	writeJSON(w, http.StatusOK, models.SMTPStatusResponse{
		Configured: settings.Configured(),
		Host:       settings.Host,
		Port:       settings.Port,
		Username:   settings.Username,
		From:       settings.From,
		TLSMode:    settings.TLSMode,
	})
}

// UpdateSMTP handles PUT /api/admin/smtp.
func (h *ManagementHandler) UpdateSMTP(w http.ResponseWriter, r *http.Request) {
	var req models.SaveSMTPRequest
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Port <= 0 || req.Port > 65535 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "端口无效"})
		return
	}
	switch req.TLSMode {
	case "", "ssl", "plain", "starttls":
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tls_mode 仅支持 ssl 或 plain"})
		return
	}
	if req.Host == "" || req.From == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "主机与发件人不能为空"})
		return
	}
	settings := models.SMTPSettings{
		Host:     strings.TrimSpace(req.Host),
		Port:     req.Port,
		Username: strings.TrimSpace(req.Username),
		From:     strings.TrimSpace(req.From),
		TLSMode:  req.TLSMode,
	}
	if req.TLSMode == "" {
		settings.TLSMode = "starttls"
	}
	if req.Password != "" {
		encrypted, err := auth.EncryptSecret(h.encryptionKey, smtpPasswordPurpose, []byte(req.Password))
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "加密 SMTP 密码失败"})
			return
		}
		settings.Password = encrypted
	} else {
		settings.Password = h.store.GetSMTPSettings().Password
	}
	if err := h.store.SetSMTPSettings(settings); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "保存 SMTP 设置失败"})
		return
	}
	h.GetSMTP(w, r)
}

// TestSMTP handles POST /api/admin/smtp/test.
func (h *ManagementHandler) TestSMTP(w http.ResponseWriter, r *http.Request) {
	var req models.SMTPTestRequest
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	to := strings.TrimSpace(req.To)
	if !validEmail(to) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "收件邮箱格式不正确"})
		return
	}
	settings, password := h.smtpSettingsOf()
	if !settings.Configured() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "SMTP 未配置"})
		return
	}
	mailer := services.NewMailer(settings, password)
	plain, htmlBody := services.TestEmail()
	if err := mailer.Send(to, "Tunnel Manager SMTP 测试邮件", plain, htmlBody); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "发送失败: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "测试邮件已发送"})
}

func (h *ManagementHandler) smtpSettingsOf() (models.SMTPSettings, string) {
	settings := h.store.GetSMTPSettings()
	if !settings.Configured() || settings.Password == "" {
		return settings, ""
	}
	plain, err := auth.DecryptSecret(h.encryptionKey, smtpPasswordPurpose, settings.Password)
	if err != nil {
		return settings, ""
	}
	return settings, string(plain)
}
