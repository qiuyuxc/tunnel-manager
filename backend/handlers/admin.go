package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

const (
	defaultSessionTTL   = 12 * time.Hour
	twoFactorTTL        = 5 * time.Minute
	maxRequestBodyBytes = 4 << 10
	maxUsernameLength   = 128
	maxNicknameLength   = 64
	maxAvatarLength     = 2048
	maxPasswordLength   = 1024
	maxCodeLength       = 32
	maxChallenges       = 1024
	maxPendingSetups    = 128
	maxSessions         = 1024
	maxFactorAttempts   = 5
	maxPasswordVerifies = 4
)

type loginChallenge struct {
	userID    string
	expiresAt time.Time
	epoch     uint64
	attempts  int
	inUse     bool
}

type pendingTOTPSetup struct {
	ownerUserID      string
	ownerSessionHash string
	secret           []byte
	expiresAt        time.Time
	epoch            uint64
	attempts         int
	inUse            bool
}

// AdminHandler handles authentication and account management for every user.
type AdminHandler struct {
	store            *store.Store
	encryptionKey    []byte
	notifier         *services.Notifier
	challenges       map[string]*loginChallenge
	setups           map[string]*pendingTOTPSetup
	mu               sync.RWMutex
	authEpoch        uint64
	passwordVerifies chan struct{}
	tokenTTL         time.Duration
	now              func() time.Time
}

// SetNotifier wires per-user notification delivery (e.g. login events).
func (h *AdminHandler) SetNotifier(notifier *services.Notifier) {
	h.notifier = notifier
}

// NewAdminHandler creates a new AdminHandler. The optional encryption key keeps
// construction compatible with callers that do not configure two-factor auth.
func NewAdminHandler(st *store.Store, encryptionKey ...[]byte) *AdminHandler {
	var key []byte
	if len(encryptionKey) > 0 {
		key = append([]byte(nil), encryptionKey[0]...)
	}
	return &AdminHandler{
		store:            st,
		encryptionKey:    key,
		challenges:       make(map[string]*loginChallenge),
		setups:           make(map[string]*pendingTOTPSetup),
		passwordVerifies: make(chan struct{}, maxPasswordVerifies),
		tokenTTL:         defaultSessionTTL,
		now:              time.Now,
	}
}

// hashToken derives the stored form of a session token.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// userIDFromToken resolves the authenticated user behind a session token.
func (h *AdminHandler) userIDFromToken(token string) (string, models.SessionUser, bool) {
	su, ok := h.ValidateSession(token)
	if !ok {
		return "", models.SessionUser{}, false
	}
	return su.ID, su, true
}

// Login handles POST /api/admin/login and authenticates any account by
// username or email.
func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	account := req.Account
	if account == "" {
		account = req.Username
	}
	if account == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password are required"})
		return
	}
	if len(account) > maxUsernameLength || len(req.Password) > maxPasswordLength {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if ok, msg := verifyTurnstile(h.store, h.encryptionKey, r, req.TurnstileResponse, "login"); !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}

	user, found := h.lookupUser(account)
	if !found {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}
	if user.Status != models.UserActive {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "account is disabled"})
		return
	}

	epoch := h.currentAuthEpoch()
	if !h.acquirePasswordVerify() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication temporarily unavailable"})
		return
	}
	passwordValid := h.store.ValidateUserPassword(user.ID, req.Password, user.PasswordHash)
	h.releasePasswordVerify()
	if !passwordValid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	enabled, encryptedSecret, _, _ := h.store.GetTOTPState(user.ID)
	if enabled {
		if _, err := auth.DecryptTOTPSecret(h.encryptionKey, encryptedSecret); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor authentication unavailable"})
			return
		}
		token, expiresAt, err := h.newChallenge(user.ID, epoch)
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication temporarily unavailable"})
			return
		}
		writeJSON(w, http.StatusAccepted, models.TwoFactorChallengeResponse{
			TwoFactorRequired: true,
			ChallengeToken:    token,
			ExpiresAt:         expiresAt,
		})
		return
	}

	token, err := h.createSession(user.ID, epoch)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication temporarily unavailable"})
		return
	}
	h.store.UpdateUserLogin(user.ID)
	h.notifyLogin(user.ID, user.Username, r)
	writeJSON(w, http.StatusOK, models.LoginResponse{Token: token, Username: user.Username, Role: user.Role})
}

// notifyLogin sends a per-user login notification (if enabled) without
// blocking the login response.
func (h *AdminHandler) notifyLogin(userID, username string, r *http.Request) {
	if h.notifier != nil {
		h.notifier.NotifyLogin(userID, username, clientIP(r))
	}
}

// lookupUser finds an account by email (when the input contains @) or name.
func (h *AdminHandler) lookupUser(account string) (models.User, bool) {
	if strings.Contains(account, "@") {
		if user, ok := h.store.GetUserByEmail(account); ok {
			return user, true
		}
	}
	if user, ok := h.store.GetUserByUsername(account); ok {
		return user, true
	}
	if !strings.Contains(account, "@") {
		if user, ok := h.store.GetUserByEmail(account); ok {
			return user, true
		}
	}
	return models.User{}, false
}

// LoginTwoFactor handles POST /api/admin/login/2fa.
func (h *AdminHandler) LoginTwoFactor(w http.ResponseWriter, r *http.Request) {
	var req models.TwoFactorLoginRequest
	if err := readAdminJSON(w, r, &req); err != nil || !validOpaqueToken(req.ChallengeToken) || req.Code == "" || len(req.Code) > maxCodeLength {
		writeInvalidFactor(w)
		return
	}
	challenge, ok := h.claimChallenge(req.ChallengeToken)
	if !ok {
		writeInvalidFactor(w)
		return
	}

	token, err := generateToken()
	if err != nil {
		h.releaseChallenge(req.ChallengeToken, challenge)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication temporarily unavailable"})
		return
	}
	prepared, internalErr := h.prepareFactor(challenge.userID, req.Code)
	if internalErr != nil {
		h.releaseChallenge(req.ChallengeToken, challenge)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor authentication unavailable"})
		return
	}
	if prepared == nil {
		h.failChallenge(req.ChallengeToken, challenge)
		writeInvalidFactor(w)
		return
	}

	valid, internalErr := h.finishChallenge(req.ChallengeToken, challenge, token, prepared)
	if internalErr != nil {
		h.releaseChallenge(req.ChallengeToken, challenge)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor authentication unavailable"})
		return
	}
	if !valid {
		writeInvalidFactor(w)
		return
	}
	username := ""
	role := ""
	if user, ok := h.store.GetUserByID(challenge.userID); ok {
		username, role = user.Username, user.Role
		h.store.UpdateUserLogin(user.ID)
		h.notifyLogin(user.ID, user.Username, r)
	}
	writeJSON(w, http.StatusOK, models.LoginResponse{Token: token, Username: username, Role: role})
}

// SetupTOTP handles POST /api/admin/2fa/setup.
func (h *AdminHandler) SetupTOTP(w http.ResponseWriter, r *http.Request) {
	if err := consumeLimitedBody(w, r); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	owner, _, ok := h.userIDFromToken(r.Header.Get("X-Auth-Token"))
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if len(h.encryptionKey) != 32 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor setup unavailable"})
		return
	}
	enabled, _, _, _ := h.store.GetTOTPState(owner)
	if enabled {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "two-factor authentication is already enabled"})
		return
	}
	secret, encodedSecret, err := auth.GenerateTOTPSecret()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor setup unavailable"})
		return
	}
	username, _ := h.store.GetAdminCredentials()
	_ = username
	displayName := owner
	if user, found := h.store.GetUserByID(owner); found {
		displayName = user.Username
	}
	uri, err := auth.BuildOTPAuthURI(displayName, encodedSecret)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor setup unavailable"})
		return
	}
	token, expiresAt, err := h.addSetup(owner, hashToken(r.Header.Get("X-Auth-Token")), secret)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor setup unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, models.TOTPSetupResponse{SetupToken: token, Secret: encodedSecret, OTPAuthURI: uri, ExpiresAt: expiresAt})
}

// ConfirmTOTP handles POST /api/admin/2fa/confirm.
func (h *AdminHandler) ConfirmTOTP(w http.ResponseWriter, r *http.Request) {
	var req models.TOTPConfirmRequest
	if err := readAdminJSON(w, r, &req); err != nil || !validOpaqueToken(req.SetupToken) || req.Code == "" || len(req.Code) > maxCodeLength {
		writeInvalidFactor(w)
		return
	}
	owner, _, ownerOK := h.userIDFromToken(r.Header.Get("X-Auth-Token"))
	if !ownerOK {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	secret, ok := h.claimSetup(req.SetupToken, owner)
	if !ok {
		writeInvalidFactor(w)
		return
	}
	step, valid := auth.MatchTOTPCode(secret, req.Code, h.now())
	if !valid {
		h.failSetup(req.SetupToken)
		writeInvalidFactor(w)
		return
	}
	encrypted, err := auth.EncryptTOTPSecret(h.encryptionKey, secret)
	if err != nil {
		h.releaseSetup(req.SetupToken)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor setup unavailable"})
		return
	}
	codes, hashes, err := auth.GenerateRecoveryCodes(10)
	if err != nil {
		h.releaseSetup(req.SetupToken)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor setup unavailable"})
		return
	}
	if err := h.finalizeSetup(req.SetupToken, owner, encrypted, hashes, step); err != nil {
		if errors.Is(err, store.ErrTOTPAlreadyEnabled) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "two-factor authentication is already enabled"})
		} else if errors.Is(err, errInvalidAuthState) {
			writeInvalidFactor(w)
		} else {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor setup unavailable"})
		}
		return
	}
	writeJSON(w, http.StatusOK, models.TOTPConfirmResponse{Enabled: true, RecoveryCodes: codes})
}

// TOTPStatus handles GET /api/admin/2fa/status.
func (h *AdminHandler) TOTPStatus(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := h.userIDFromToken(r.Header.Get("X-Auth-Token"))
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	enabled, _, _, count := h.store.GetTOTPState(userID)
	writeJSON(w, http.StatusOK, models.TOTPStatusResponse{
		Enabled:                enabled,
		RecoveryCodesRemaining: count,
		SetupAvailable:         !enabled && len(h.encryptionKey) == 32,
	})
}

// DisableTOTP handles POST /api/admin/2fa/disable.
func (h *AdminHandler) DisableTOTP(w http.ResponseWriter, r *http.Request) {
	var req models.TOTPDisableRequest
	if err := readAdminJSON(w, r, &req); err != nil || req.CurrentPassword == "" || len(req.CurrentPassword) > maxPasswordLength || req.Code == "" || len(req.Code) > maxCodeLength {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	userID, _, tokenOK := h.userIDFromToken(r.Header.Get("X-Auth-Token"))
	if !tokenOK {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !h.acquirePasswordVerify() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication temporarily unavailable"})
		return
	}
	stored, _ := h.store.GetUserByID(userID)
	passwordValid := h.store.ValidateUserPassword(userID, req.CurrentPassword, stored.PasswordHash)
	h.releasePasswordVerify()
	if !passwordValid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}
	enabled, encryptedSecret, _, _ := h.store.GetTOTPState(userID)
	if !enabled {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "two-factor authentication is not enabled"})
		return
	}
	secret, err := auth.DecryptTOTPSecret(h.encryptionKey, encryptedSecret)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor authentication unavailable"})
		return
	}

	if step, ok := auth.MatchTOTPCode(secret, req.Code, h.now()); ok {
		err = h.store.DisableTOTPWithStep(userID, step)
	} else if normalized, normalizeErr := auth.NormalizeRecoveryCode(req.Code); normalizeErr == nil {
		err = h.store.DisableTOTPWithRecovery(userID, auth.HashRecoveryCode(normalized))
	} else {
		writeInvalidFactor(w)
		return
	}
	if err != nil {
		if errors.Is(err, store.ErrTOTPReplay) || errors.Is(err, store.ErrRecoveryCodeNotFound) {
			writeInvalidFactor(w)
		} else {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor authentication unavailable"})
		}
		return
	}
	// Revoking sessions after a 2FA change matches the password-change policy.
	_ = h.store.DeleteUserSessions(userID)
	h.revokeAllAuthState()
	writeJSON(w, http.StatusOK, models.TOTPDisableResponse{Enabled: false})
}

// Status handles GET /api/admin/status.
func (h *AdminHandler) Status(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Auth-Token")
	su, ok := h.ValidateSession(token)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{"authenticated": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"authenticated": true, "username": su.Username, "role": su.Role})
}

// ChangePassword handles PUT /api/admin/password.
func (h *AdminHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req models.ChangePasswordRequest
	if err := readAdminJSON(w, r, &req); err != nil || len(req.CurrentPassword) > maxPasswordLength || len(req.NewPassword) > maxPasswordLength {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	userID, _, ok := h.userIDFromToken(r.Header.Get("X-Auth-Token"))
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !h.acquirePasswordVerify() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication temporarily unavailable"})
		return
	}
	stored, _ := h.store.GetUserByID(userID)
	passwordValid := h.store.ValidateUserPassword(userID, req.CurrentPassword, stored.PasswordHash)
	h.releasePasswordVerify()
	if !passwordValid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "current password is incorrect"})
		return
	}
	if len(req.NewPassword) < 6 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "new password must be at least 6 characters"})
		return
	}
	if err := h.store.SetUserPasswordHash(userID, store.HashPassword(req.NewPassword)); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unable to update password"})
		return
	}
	// Every session of this account is revoked after a password change.
	_ = h.store.DeleteUserSessions(userID)
	h.revokeAllAuthState()
	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated successfully"})
}

// ChangeUsername handles PUT /api/admin/username.
func (h *AdminHandler) ChangeUsername(w http.ResponseWriter, r *http.Request) {
	var req models.ChangeUsernameRequest
	if err := readAdminJSON(w, r, &req); err != nil || len(req.CurrentPassword) > maxPasswordLength || len(req.NewUsername) > maxUsernameLength {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	userID, _, ok := h.userIDFromToken(r.Header.Get("X-Auth-Token"))
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !h.acquirePasswordVerify() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication temporarily unavailable"})
		return
	}
	stored, _ := h.store.GetUserByID(userID)
	passwordValid := h.store.ValidateUserPassword(userID, req.CurrentPassword, stored.PasswordHash)
	h.releasePasswordVerify()
	if !passwordValid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "password is incorrect"})
		return
	}
	if req.NewUsername == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "new username cannot be empty"})
		return
	}
	if err := h.store.SetUsername(userID, req.NewUsername); err != nil {
		if errors.Is(err, store.ErrUsernameTaken) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "username already taken"})
		} else {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unable to update username"})
		}
		return
	}
	_ = h.store.DeleteUserSessions(userID)
	h.revokeAllAuthState()
	writeJSON(w, http.StatusOK, map[string]string{"message": "username updated successfully"})
}

// ChangeEmail handles PUT /api/admin/email: binds or replaces the account
// email (password confirmation required).
func (h *AdminHandler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	var req models.ChangeEmailRequest
	if err := readAdminJSON(w, r, &req); err != nil || len(req.CurrentPassword) > maxPasswordLength || len(req.NewEmail) > maxUsernameLength {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	userID, _, ok := h.userIDFromToken(r.Header.Get("X-Auth-Token"))
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !h.acquirePasswordVerify() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication temporarily unavailable"})
		return
	}
	stored, _ := h.store.GetUserByID(userID)
	passwordValid := h.store.ValidateUserPassword(userID, req.CurrentPassword, stored.PasswordHash)
	h.releasePasswordVerify()
	if !passwordValid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "password is incorrect"})
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.NewEmail))
	if email != "" && !validEmail(email) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邮箱格式不正确"})
		return
	}
	if err := h.store.SetUserEmail(userID, email, email != ""); err != nil {
		if errors.Is(err, store.ErrEmailTaken) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "邮箱已被其他账号使用"})
		} else {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unable to update email"})
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "邮箱已更新"})
}

// ChangeProfile handles PUT /api/admin/profile: updates the account display
// nickname and custom avatar URL. Password confirmation is not required for
// these low-risk fields.
func (h *AdminHandler) ChangeProfile(w http.ResponseWriter, r *http.Request) {
	var req models.ChangeProfileRequest
	if err := readAdminJSON(w, r, &req); err != nil || len(req.Nickname) > maxNicknameLength || len(req.Avatar) > maxAvatarLength {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	userID, _, ok := h.userIDFromToken(r.Header.Get("X-Auth-Token"))
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	nickname := strings.TrimSpace(req.Nickname)
	if strings.ContainsAny(nickname, "\r\n") || len([]rune(nickname)) > maxNicknameLength {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "昵称不能包含换行且长度不能超过 64 个字符"})
		return
	}
	avatar := strings.TrimSpace(req.Avatar)
	if !validAvatarURL(avatar) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "头像地址无效，仅支持 /uploads/ 路径或 http(s) 链接"})
		return
	}
	if err := h.store.SetUserProfile(userID, nickname, avatar); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unable to update profile"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "个人资料已更新"})
}

// validAvatarURL accepts an empty value (clears the avatar), a stored
// /uploads/ path or an absolute http(s) URL.
func validAvatarURL(value string) bool {
	if value == "" {
		return true
	}
	if strings.HasPrefix(value, "/uploads/") {
		return !strings.Contains(value, "..") && !strings.ContainsAny(value, " \t\r\n")
	}
	lower := strings.ToLower(value)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func (h *AdminHandler) ValidateSession(token string) (models.SessionUser, bool) {
	if !validOpaqueToken(token) {
		return models.SessionUser{}, false
	}
	return h.store.GetSessionUser(hashToken(token), h.now().Unix())
}

// ValidateToken reports whether a session token is valid.
func (h *AdminHandler) ValidateToken(token string) bool {
	_, ok := h.ValidateSession(token)
	return ok
}

// Logout handles POST /api/admin/logout.
func (h *AdminHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Auth-Token")
	if token != "" {
		tokenHash := hashToken(token)
		su, _ := h.store.GetSessionUser(tokenHash, h.now().Unix())
		_ = h.store.DeleteSession(tokenHash)
		h.mu.Lock()
		for setupToken, setup := range h.setups {
			if su.ID != "" && setup.ownerUserID == su.ID {
				delete(h.setups, setupToken)
			}
		}
		h.mu.Unlock()
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

type preparedFactor struct {
	step         *int64
	recoveryHash string
}

var errInvalidAuthState = errors.New("authentication state changed")

func (h *AdminHandler) prepareFactor(userID, code string) (*preparedFactor, error) {
	enabled, encryptedSecret, _, _ := h.store.GetTOTPState(userID)
	if !enabled {
		return nil, nil
	}
	secret, err := auth.DecryptTOTPSecret(h.encryptionKey, encryptedSecret)
	if err != nil {
		return nil, err
	}
	if step, ok := auth.MatchTOTPCode(secret, code, h.now()); ok {
		return &preparedFactor{step: &step}, nil
	}
	normalized, err := auth.NormalizeRecoveryCode(code)
	if err != nil {
		return nil, nil
	}
	return &preparedFactor{recoveryHash: auth.HashRecoveryCode(normalized)}, nil
}

func (h *AdminHandler) consumeFactor(userID string, prepared *preparedFactor) error {
	if prepared.step != nil {
		return h.store.AdvanceTOTPStep(userID, *prepared.step)
	}
	return h.store.ConsumeRecoveryCode(userID, prepared.recoveryHash)
}

// createSession persists a database-backed session for the account.
func (h *AdminHandler) createSession(userID string, epoch uint64) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}
	h.mu.Lock()
	if h.authEpoch != epoch {
		h.mu.Unlock()
		return "", errInvalidAuthState
	}
	h.mu.Unlock()
	expiresAt := h.now().Add(h.tokenTTL)
	if err := h.store.CreateSession(hashToken(token), userID, expiresAt.Unix()); err != nil {
		return "", err
	}
	return token, nil
}

func (h *AdminHandler) newChallenge(userID string, epoch uint64) (string, time.Time, error) {
	token, err := generateToken()
	if err != nil {
		return "", time.Time{}, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.now()
	h.pruneLocked(now)
	if h.authEpoch != epoch {
		return "", time.Time{}, errInvalidAuthState
	}
	if len(h.challenges) >= maxChallenges {
		return "", time.Time{}, errors.New("challenge capacity reached")
	}
	expiresAt := now.Add(twoFactorTTL)
	h.challenges[token] = &loginChallenge{userID: userID, expiresAt: expiresAt, epoch: epoch}
	return token, expiresAt, nil
}

func (h *AdminHandler) addSetup(ownerUserID, ownerSessionHash string, secret []byte) (string, time.Time, error) {
	token, err := generateToken()
	if err != nil {
		return "", time.Time{}, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.now()
	h.pruneLocked(now)
	for setupToken, setup := range h.setups {
		if setup.ownerUserID == ownerUserID {
			delete(h.setups, setupToken)
		}
	}
	if len(h.setups) >= maxPendingSetups {
		return "", time.Time{}, errors.New("setup capacity reached")
	}
	expiresAt := now.Add(twoFactorTTL)
	h.setups[token] = &pendingTOTPSetup{
		ownerUserID:      ownerUserID,
		ownerSessionHash: ownerSessionHash,
		secret:           append([]byte(nil), secret...),
		expiresAt:        expiresAt,
		epoch:            h.authEpoch,
	}
	return token, expiresAt, nil
}

func (h *AdminHandler) claimChallenge(token string) (*loginChallenge, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.now()
	h.pruneLocked(now)
	challenge, ok := h.challenges[token]
	if !ok || challenge.inUse || challenge.attempts >= maxFactorAttempts || challenge.epoch != h.authEpoch || !now.Before(challenge.expiresAt) {
		return nil, false
	}
	challenge.inUse = true
	return challenge, true
}

func (h *AdminHandler) releaseChallenge(token string, claimed *loginChallenge) {
	h.mu.Lock()
	if challenge := h.challenges[token]; challenge == claimed && challenge.inUse {
		challenge.inUse = false
	}
	h.mu.Unlock()
}

func (h *AdminHandler) failChallenge(token string, claimed *loginChallenge) {
	h.mu.Lock()
	if challenge := h.challenges[token]; challenge == claimed && challenge.inUse {
		challenge.attempts++
		challenge.inUse = false
		if challenge.attempts >= maxFactorAttempts {
			delete(h.challenges, token)
		}
	}
	h.mu.Unlock()
}

func (h *AdminHandler) finishChallenge(token string, claimed *loginChallenge, sessionToken string, factor *preparedFactor) (bool, error) {
	h.mu.Lock()
	challenge := h.challenges[token]
	now := h.now()
	if challenge != claimed || !challenge.inUse || challenge.epoch != h.authEpoch || !now.Before(challenge.expiresAt) {
		h.mu.Unlock()
		return false, nil
	}
	userID := challenge.userID
	h.mu.Unlock()

	if err := h.consumeFactor(userID, factor); err != nil {
		if errors.Is(err, store.ErrTOTPReplay) || errors.Is(err, store.ErrRecoveryCodeNotFound) || errors.Is(err, store.ErrTOTPDisabled) {
			return false, nil
		}
		return false, err
	}
	if err := h.store.CreateSession(hashToken(sessionToken), userID, h.now().Add(h.tokenTTL).Unix()); err != nil {
		return false, err
	}
	h.mu.Lock()
	delete(h.challenges, token)
	h.mu.Unlock()
	return true, nil
}

func (h *AdminHandler) claimSetup(token, owner string) ([]byte, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.now()
	h.pruneLocked(now)
	setup, ok := h.setups[token]
	if !ok || setup.inUse || setup.attempts >= maxFactorAttempts || setup.epoch != h.authEpoch || !now.Before(setup.expiresAt) || setup.ownerUserID != owner {
		return nil, false
	}
	setup.inUse = true
	return append([]byte(nil), setup.secret...), true
}

func (h *AdminHandler) finalizeSetup(token, owner, encrypted string, hashes []string, step int64) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	setup := h.setups[token]
	now := h.now()
	if setup == nil || !setup.inUse || setup.epoch != h.authEpoch || !now.Before(setup.expiresAt) || setup.ownerUserID != owner {
		return errInvalidAuthState
	}
	// The originating session must still be alive.
	if _, alive := h.store.GetSessionUser(setup.ownerSessionHash, now.Unix()); !alive {
		delete(h.setups, token)
		return errInvalidAuthState
	}
	if err := h.store.EnableTOTP(owner, encrypted, hashes, step); err != nil {
		setup.inUse = false
		return err
	}
	// Enabling a second factor revokes existing sessions, like the original
	// single-admin flow did.
	_ = h.store.DeleteUserSessions(owner)
	h.revokeAllAuthStateLocked()
	return nil
}

func (h *AdminHandler) releaseSetup(token string) {
	h.mu.Lock()
	if setup := h.setups[token]; setup != nil {
		setup.inUse = false
	}
	h.mu.Unlock()
}

func (h *AdminHandler) failSetup(token string) {
	h.mu.Lock()
	if setup := h.setups[token]; setup != nil {
		setup.attempts++
		setup.inUse = false
		if setup.attempts >= maxFactorAttempts {
			delete(h.setups, token)
		}
	}
	h.mu.Unlock()
}

func (h *AdminHandler) pruneLocked(now time.Time) {
	for token, challenge := range h.challenges {
		if (!now.Before(challenge.expiresAt) || challenge.epoch != h.authEpoch) && !challenge.inUse {
			delete(h.challenges, token)
		}
	}
	for token, setup := range h.setups {
		if (!now.Before(setup.expiresAt) || setup.epoch != h.authEpoch) && !setup.inUse {
			delete(h.setups, token)
		}
	}
}

func (h *AdminHandler) currentAuthEpoch() uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.authEpoch
}

func (h *AdminHandler) acquirePasswordVerify() bool {
	select {
	case h.passwordVerifies <- struct{}{}:
		return true
	default:
		return false
	}
}

func (h *AdminHandler) releasePasswordVerify() {
	<-h.passwordVerifies
}

func (h *AdminHandler) revokeAllAuthState() {
	h.mu.Lock()
	h.revokeAllAuthStateLocked()
	h.mu.Unlock()
}

func (h *AdminHandler) revokeAllAuthStateLocked() {
	h.authEpoch++
	h.challenges = make(map[string]*loginChallenge)
	h.setups = make(map[string]*pendingTOTPSetup)
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func validOpaqueToken(token string) bool {
	if len(token) != 64 {
		return false
	}
	for i := range token {
		if (token[i] < '0' || token[i] > '9') && (token[i] < 'a' || token[i] > 'f') {
			return false
		}
	}
	return true
}

func readAdminJSON(w http.ResponseWriter, r *http.Request, target interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra interface{}
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func consumeLimitedBody(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	defer r.Body.Close()
	_, err := io.Copy(io.Discard, r.Body)
	return err
}

func writeInvalidFactor(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid authentication code"})
}
