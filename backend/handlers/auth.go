package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	userSessionTTL       = 7 * 24 * time.Hour
	verifyCodeTTL        = 10 * time.Minute
	verifyCodeResendWait = 60 * time.Second
	smtpPasswordPurpose  = "smtp-password"
)

// AuthHandler serves the public registration flows.
type AuthHandler struct {
	store         *store.Store
	encryptionKey []byte
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

// AuthConfig handles GET /api/auth/config: how the register form renders.
func (h *AuthHandler) AuthConfig(w http.ResponseWriter, r *http.Request) {
	settings := h.store.GetAppSettings()
	writeJSON(w, http.StatusOK, models.AuthConfigResponse{
		RegistrationEnabled: settings.RegistrationEnabled,
		InviteMode:          settings.InviteMode,
		EmailVerifyEnabled:  h.mailer() != nil,
	})
}

// SendCode handles POST /api/auth/send-code.
func (h *AuthHandler) SendCode(w http.ResponseWriter, r *http.Request) {
	var req models.SendCodeRequest
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if !validEmail(email) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邮箱格式不正确"})
		return
	}
	mailer := h.mailer()
	if mailer == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邮件服务未配置，请联系管理员"})
		return
	}
	if h.store.LastCodeSentWithin(email, "register", verifyCodeResendWait) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "发送过于频繁，请稍后再试"})
		return
	}

	code, err := generateNumericCode()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "生成验证码失败"})
		return
	}
	sum := sha256.Sum256([]byte(code))
	h.store.PutVerifyCode(email, "register", hex.EncodeToString(sum[:]), verifyCodeTTL)

	subject := "注册验证码"
	body := fmt.Sprintf("您的注册验证码是：%s\n\n%v 分钟内有效。若非本人操作请忽略本邮件。", code, verifyCodeTTL/time.Minute)
	if err := mailer.Send(email, subject, body); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "邮件发送失败: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "验证码已发送"})
}

// Register handles POST /api/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	settings := h.store.GetAppSettings()
	if !settings.RegistrationEnabled {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "注册未开放"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !validUsername(req.Username) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "用户名需 2-32 位字母、数字、下划线或短横线"})
		return
	}
	if !validEmail(req.Email) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邮箱格式不正确"})
		return
	}
	if len(req.Password) < 6 || len(req.Password) > maxPasswordLength {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "密码长度需在 6-1024 位之间"})
		return
	}

	// Invite policy.
	groupID := settings.DefaultGroupID
	switch settings.InviteMode {
	case models.InviteModeRequired:
		if strings.TrimSpace(req.Invite) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邀请码不能为空"})
			return
		}
		granted, err := h.store.ConsumeInvite(strings.TrimSpace(req.Invite))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邀请码无效、已过期或已用尽"})
			return
		}
		if granted != "" {
			groupID = granted
		}
	case models.InviteModeOptional:
		if strings.TrimSpace(req.Invite) != "" {
			granted, err := h.store.ConsumeInvite(strings.TrimSpace(req.Invite))
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邀请码无效、已过期或已用尽"})
				return
			}
			if granted != "" {
				groupID = granted
			}
		}
	default: // off
	}

	// Email verification policy: enforced once SMTP is configured.
	emailVerified := true
	if h.mailer() != nil {
		emailVerified = false
		if strings.TrimSpace(req.VerifyCode) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邮箱验证码不能为空"})
			return
		}
		sum := sha256.Sum256([]byte(strings.TrimSpace(req.VerifyCode)))
		if !h.store.ConsumeVerifyCode(req.Email, "register", hex.EncodeToString(sum[:])) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "验证码错误或已过期"})
			return
		}
		emailVerified = true
	}

	if _, exists := h.store.GetUserByUsername(req.Username); exists {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "用户名已被占用"})
		return
	}
	if _, exists := h.store.GetUserByEmail(req.Email); exists {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "邮箱已被注册"})
		return
	}

	user := models.User{
		Username:      req.Username,
		Email:         req.Email,
		PasswordHash:  store.HashPassword(req.Password),
		Role:          models.RoleUser,
		GroupID:       groupID,
		Status:        models.UserActive,
		EmailVerified: emailVerified,
		CreatedAt:     time.Now().Unix(),
	}
	if err := h.store.CreateUser(user); err != nil {
		switch {
		case errors.Is(err, store.ErrUsernameTaken):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "用户名已被占用"})
		case errors.Is(err, store.ErrEmailTaken):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "邮箱已被注册"})
		default:
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "创建账户失败"})
		}
		return
	}

	created, _ := h.store.GetUserByUsername(req.Username)
	token, err := generateToken()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication temporarily unavailable"})
		return
	}
	if err := h.store.CreateSession(hashToken(token), created.ID, time.Now().Add(userSessionTTL).Unix()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication temporarily unavailable"})
		return
	}
	h.store.UpdateUserLogin(created.ID)
	writeJSON(w, http.StatusCreated, models.LoginResponse{Token: token, Username: created.Username, Role: created.Role})
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
		Email:       user.Email,
		Role:        user.Role,
		Permissions: user.Permissions,
	})
}

func validUsername(name string) bool {
	if len(name) < 2 || len(name) > 32 {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
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
