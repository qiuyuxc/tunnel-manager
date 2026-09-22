package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

const (
	verifyCodeTTL        = 10 * time.Minute
	verifyCodeResendWait = 60 * time.Second
	smtpPasswordPurpose  = "smtp-password"
)

// AuthHandler serves the public registration flows.
type AuthHandler struct {
	store         *store.Store
	encryptionKey []byte
	// throttle rate-limits the unauthenticated endpoints below. Nil disables
	// the limit, which is what tests unrelated to throttling want.
	throttle *Throttle
}

// SetThrottle wires the sign-in rate limiter.
func (h *AuthHandler) SetThrottle(t *Throttle) {
	h.throttle = t
}

// guardAuth applies the rate limit to one unauthenticated request.
//
// key identifies what the request is aimed at — an email address, usually —
// so one mailbox cannot be bombed, while the address budget covers a caller
// working through a list of them. Unlike sign-in there is no account to
// relax for: nothing here proves who the caller is.
func (h *AuthHandler) guardAuth(w http.ResponseWriter, r *http.Request, key string) bool {
	if h.throttle == nil {
		return true
	}
	if ok, wait := h.throttle.Guard(r, key, ""); !ok {
		writeTooManyAttempts(w, wait)
		return false
	}
	return true
}

// chargeAuth books one request against the same budgets. Every request that
// gets past the guard is charged: for these endpoints the request itself is
// the cost, whether it ends in a sent code, a new account or an error.
func (h *AuthHandler) chargeAuth(r *http.Request, key string) {
	if h.throttle != nil {
		h.throttle.Fail(r, key, "")
	}
}

// NewAuthHandler creates the registration handler.
func NewAuthHandler(st *store.Store, encryptionKey []byte) *AuthHandler {
	return &AuthHandler{store: st, encryptionKey: append([]byte(nil), encryptionKey...)}
}

// smtpSettings decrypts the stored relay configuration.
func (h *AuthHandler) smtpSettings() (models.SMTPSettings, string) {
	settings := h.store.GetSMTPSettings()
	if !settings.Configured() {
		return settings, ""
	}
	plain, err := auth.DecryptSecret(h.encryptionKey, smtpPasswordPurpose, settings.Password)
	if err != nil {
		return settings, ""
	}
	return settings, string(plain)
}

// mailer builds an SMTP client from stored settings, or nil when unconfigured.
func (h *AuthHandler) mailer() *services.Mailer {
	settings, password := h.smtpSettings()
	if !settings.Configured() {
		return nil
	}
	return services.NewMailer(settings, password)
}

// AuthConfig handles GET /api/auth/config: Turnstile settings for the login
// and password-recovery forms.
func (h *AuthHandler) AuthConfig(w http.ResponseWriter, r *http.Request) {
	settings := h.store.GetAppSettings()
	writeJSON(w, http.StatusOK, models.AuthConfigResponse{
		TurnstileEnabled: settings.TurnstileEnabled,
		TurnstileSiteKey: settings.TurnstileSiteKey,
	})
}

// Me handles GET /api/auth/me.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := SessionUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, models.MeResponse{
		ID:          user.ID,
		Username:    user.Username,
		Nickname:    user.Nickname,
		Avatar:      user.Avatar,
		Email:       user.Email,
		Role:        user.Role,
		Permissions: user.Permissions,
	})
}

func validEmail(email string) bool {
	at := strings.Index(email, "@")
	if at <= 0 || at == len(email)-1 || strings.Count(email, "@") != 1 {
		return false
	}
	domain := email[at+1:]
	if !strings.Contains(domain, ".") || strings.Contains(email, " ") {
		return false
	}
	return len(email) <= 254
}

func generateNumericCode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	num := (int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])) % 1000000
	return fmt.Sprintf("%06d", num), nil
}

// ForgotPassword handles POST /api/auth/forgot-password: emails a reset code
// to the account owner when SMTP is available.
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req models.SendCodeRequest
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if ok, msg := verifyTurnstile(h.store, h.encryptionKey, r, req.TurnstileResponse, "forgot-password"); !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if !validEmail(email) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邮箱格式不正确"})
		return
	}
	if !h.guardAuth(w, r, email) {
		return
	}
	h.chargeAuth(r, email)
	mailer := h.mailer()
	if mailer == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邮件服务未配置，请联系管理员通过后台重置密码"})
		return
	}
	if _, exists := h.store.GetUserByEmail(email); !exists {
		// Do not reveal whether the account exists.
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "如果该邮箱已注册，将收到重置验证码"})
		return
	}
	if h.store.LastCodeSentWithin(email, "reset", verifyCodeResendWait) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "发送过于频繁，请稍后再试"})
		return
	}
	code, err := generateNumericCode()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "生成验证码失败"})
		return
	}
	sum := sha256.Sum256([]byte(code))
	h.store.PutVerifyCode(email, "reset", hex.EncodeToString(sum[:]), verifyCodeTTL)
	subject := "Tunnel Manager 密码重置验证码"
	plain, htmlBody := services.VerificationCodeEmail("密码重置", code, int(verifyCodeTTL/time.Minute))
	if err := mailer.Send(email, subject, plain, htmlBody); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "邮件发送失败: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "重置验证码已发送"})
}

// ResetPassword handles POST /api/auth/reset-password: sets a new password
// with the emailed reset code and revokes the account's sessions.
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req models.ResetPasswordRequest
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if ok, msg := verifyTurnstile(h.store, h.encryptionKey, r, req.TurnstileResponse, "reset-password"); !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if !validEmail(email) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邮箱格式不正确"})
		return
	}
	if !h.guardAuth(w, r, email) {
		return
	}
	h.chargeAuth(r, email)
	if len(req.NewPassword) < 6 || len(req.NewPassword) > maxPasswordLength {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "密码长度需在 6-1024 位之间"})
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "重置验证码不能为空"})
		return
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(req.Code)))
	if !h.store.ConsumeVerifyCode(email, "reset", hex.EncodeToString(sum[:])) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "验证码错误或已过期"})
		return
	}
	user, exists := h.store.GetUserByEmail(email)
	if !exists || user.Status != models.UserActive {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "验证码错误或已过期"})
		return
	}
	if err := h.store.SetUserPasswordHash(user.ID, store.HashPassword(req.NewPassword)); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "重置失败，请稍后再试"})
		return
	}
	_ = h.store.DeleteUserSessions(user.ID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "密码已重置，请使用新密码登录"})
}
