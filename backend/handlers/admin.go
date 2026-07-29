package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/store"
)

const (
	defaultSessionTTL   = 12 * time.Hour
	twoFactorTTL        = 5 * time.Minute
	maxRequestBodyBytes = 4 << 10
	maxUsernameLength   = 128
	maxPasswordLength   = 1024
	maxCodeLength       = 32
	maxChallenges       = 1024
	maxPendingSetups    = 128
	maxSessions         = 1024
	maxFactorAttempts   = 5
	maxPasswordVerifies = 4
)

type loginChallenge struct {
	expiresAt time.Time
	epoch     uint64
	attempts  int
	inUse     bool
}

type pendingTOTPSetup struct {
	ownerToken string
	secret     []byte
	expiresAt  time.Time
	epoch      uint64
	attempts   int
	inUse      bool
}

// AdminHandler handles admin authentication and account management.
type AdminHandler struct {
	store            *store.Store
	encryptionKey    []byte
	sessions         map[string]time.Time
	challenges       map[string]*loginChallenge
	setups           map[string]*pendingTOTPSetup
	mu               sync.RWMutex
	authEpoch        uint64
	passwordVerifies chan struct{}
	tokenTTL         time.Duration
	now              func() time.Time
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
		sessions:         make(map[string]time.Time),
		challenges:       make(map[string]*loginChallenge),
		setups:           make(map[string]*pendingTOTPSetup),
		passwordVerifies: make(chan struct{}, maxPasswordVerifies),
		tokenTTL:         defaultSessionTTL,
		now:              time.Now,
	}
}

// Login handles POST /api/admin/login.
func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password are required"})
		return
	}
	if len(req.Username) > maxUsernameLength || len(req.Password) > maxPasswordLength {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	epoch := h.currentAuthEpoch()
	if !h.acquirePasswordVerify() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication temporarily unavailable"})
		return
	}
	username, passwordHash := h.store.GetAdminCredentials()
	passwordValid := h.store.ValidatePassword(req.Password, passwordHash)
	h.releasePasswordVerify()
	usernameValid := subtle.ConstantTimeCompare([]byte(req.Username), []byte(username)) == 1
	if !passwordValid || !usernameValid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	enabled, encryptedSecret, _, _ := h.store.GetTOTPState()
	if enabled {
		if _, err := auth.DecryptTOTPSecret(h.encryptionKey, encryptedSecret); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor authentication unavailable"})
			return
		}
		token, expiresAt, err := h.newChallenge(epoch)
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

	token, err := h.createSession(epoch)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication temporarily unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, models.LoginResponse{Token: token, Username: username})
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
	prepared, internalErr := h.prepareFactor(req.Code)
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

	username, _ := h.store.GetAdminCredentials()
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
	writeJSON(w, http.StatusOK, models.LoginResponse{Token: token, Username: username})
}

// SetupTOTP handles POST /api/admin/2fa/setup.
func (h *AdminHandler) SetupTOTP(w http.ResponseWriter, r *http.Request) {
	if err := consumeLimitedBody(w, r); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	owner := r.Header.Get("X-Auth-Token")
	if len(h.encryptionKey) != 32 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor setup unavailable"})
		return
	}
	enabled, _, _, _ := h.store.GetTOTPState()
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
	uri, err := auth.BuildOTPAuthURI(username, encodedSecret)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor setup unavailable"})
		return
	}
	token, expiresAt, err := h.addSetup(owner, secret)
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
	owner := r.Header.Get("X-Auth-Token")
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
	enabled, _, _, count := h.store.GetTOTPState()
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
	if !h.acquirePasswordVerify() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication temporarily unavailable"})
		return
	}
	_, passwordHash := h.store.GetAdminCredentials()
	passwordValid := h.store.ValidatePassword(req.CurrentPassword, passwordHash)
	h.releasePasswordVerify()
	if !passwordValid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}
	enabled, encryptedSecret, _, _ := h.store.GetTOTPState()
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
		err = h.store.DisableTOTPWithStep(step)
	} else if normalized, normalizeErr := auth.NormalizeRecoveryCode(req.Code); normalizeErr == nil {
		err = h.store.DisableTOTPWithRecovery(auth.HashRecoveryCode(normalized))
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
	h.revokeAllAuthState()
	writeJSON(w, http.StatusOK, models.TOTPDisableResponse{Enabled: false})
}

// Status handles GET /api/admin/status.
func (h *AdminHandler) Status(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Auth-Token")
	if !h.ValidateToken(token) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"authenticated": "false"})
		return
	}
	username, _ := h.store.GetAdminCredentials()
	writeJSON(w, http.StatusOK, map[string]interface{}{"authenticated": true, "username": username})
}

// ChangePassword handles PUT /api/admin/password.
func (h *AdminHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req models.ChangePasswordRequest
	if err := readAdminJSON(w, r, &req); err != nil || len(req.CurrentPassword) > maxPasswordLength || len(req.NewPassword) > maxPasswordLength {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if !h.acquirePasswordVerify() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication temporarily unavailable"})
		return
	}
	_, passwordHash := h.store.GetAdminCredentials()
	passwordValid := h.store.ValidatePassword(req.CurrentPassword, passwordHash)
	h.releasePasswordVerify()
	if !passwordValid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "current password is incorrect"})
		return
	}
	if len(req.NewPassword) < 6 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "new password must be at least 6 characters"})
		return
	}
	if err := h.store.SetAdminPasswordHash(store.HashPassword(req.NewPassword)); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unable to update password"})
		return
	}
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
	if !h.acquirePasswordVerify() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication temporarily unavailable"})
		return
	}
	_, passwordHash := h.store.GetAdminCredentials()
	passwordValid := h.store.ValidatePassword(req.CurrentPassword, passwordHash)
	h.releasePasswordVerify()
	if !passwordValid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "password is incorrect"})
		return
	}
	if req.NewUsername == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "new username cannot be empty"})
		return
	}
	if err := h.store.SetAdminUsername(req.NewUsername); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unable to update username"})
		return
	}
	h.revokeAllAuthState()
	writeJSON(w, http.StatusOK, map[string]string{"message": "username updated successfully"})
}

// ValidateToken checks whether a well-formed, unexpired session token is valid.
func (h *AdminHandler) ValidateToken(token string) bool {
	if !validOpaqueToken(token) {
		return false
	}
	now := h.now()
	h.mu.RLock()
	expiresAt, ok := h.sessions[token]
	h.mu.RUnlock()
	if ok && now.Before(expiresAt) {
		return true
	}
	if ok {
		h.mu.Lock()
		if expiresAt, exists := h.sessions[token]; exists && !h.now().Before(expiresAt) {
			delete(h.sessions, token)
		}
		h.mu.Unlock()
	}
	return false
}

// Logout handles POST /api/admin/logout.
func (h *AdminHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Auth-Token")
	if token != "" {
		h.mu.Lock()
		delete(h.sessions, token)
		for setupToken, setup := range h.setups {
			if subtle.ConstantTimeCompare([]byte(setup.ownerToken), []byte(token)) == 1 {
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

func (h *AdminHandler) prepareFactor(code string) (*preparedFactor, error) {
	enabled, encryptedSecret, _, _ := h.store.GetTOTPState()
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

func (h *AdminHandler) consumeFactor(prepared *preparedFactor) error {
	if prepared.step != nil {
		return h.store.AdvanceTOTPStep(*prepared.step)
	}
	return h.store.ConsumeRecoveryCode(prepared.recoveryHash)
}

func (h *AdminHandler) createSession(epoch uint64) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.authEpoch != epoch {
		return "", errInvalidAuthState
	}
	if !h.issueSessionLocked(token, h.now()) {
		return "", errors.New("session capacity reached")
	}
	return token, nil
}

func (h *AdminHandler) issueSessionLocked(token string, now time.Time) bool {
	h.pruneSessionsLocked(now)
	if len(h.sessions) >= maxSessions {
		return false
	}
	h.sessions[token] = now.Add(h.tokenTTL)
	return true
}

func (h *AdminHandler) newChallenge(epoch uint64) (string, time.Time, error) {
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
	h.challenges[token] = &loginChallenge{expiresAt: expiresAt, epoch: epoch}
	return token, expiresAt, nil
}

func (h *AdminHandler) addSetup(owner string, secret []byte) (string, time.Time, error) {
	token, err := generateToken()
	if err != nil {
		return "", time.Time{}, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.now()
	h.pruneLocked(now)
	for setupToken, setup := range h.setups {
		if subtle.ConstantTimeCompare([]byte(setup.ownerToken), []byte(owner)) == 1 {
			delete(h.setups, setupToken)
		}
	}
	if len(h.setups) >= maxPendingSetups {
		return "", time.Time{}, errors.New("setup capacity reached")
	}
	expiresAt := now.Add(twoFactorTTL)
	h.setups[token] = &pendingTOTPSetup{
		ownerToken: owner,
		secret:     append([]byte(nil), secret...),
		expiresAt:  expiresAt,
		epoch:      h.authEpoch,
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
	defer h.mu.Unlock()
	challenge := h.challenges[token]
	now := h.now()
	if challenge != claimed || !challenge.inUse || challenge.epoch != h.authEpoch || !now.Before(challenge.expiresAt) {
		return false, nil
	}
	h.pruneSessionsLocked(now)
	if len(h.sessions) >= maxSessions {
		return false, errors.New("session capacity reached")
	}
	if err := h.consumeFactor(factor); err != nil {
		if errors.Is(err, store.ErrTOTPReplay) || errors.Is(err, store.ErrRecoveryCodeNotFound) || errors.Is(err, store.ErrTOTPDisabled) {
			return false, nil
		}
		return false, err
	}
	delete(h.challenges, token)
	if !h.issueSessionLocked(sessionToken, now) {
		return false, errors.New("session capacity reached after factor consumption")
	}
	return true, nil
}

func (h *AdminHandler) claimSetup(token, owner string) ([]byte, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.now()
	h.pruneLocked(now)
	setup, ok := h.setups[token]
	if !ok || setup.inUse || setup.attempts >= maxFactorAttempts || setup.epoch != h.authEpoch || !now.Before(setup.expiresAt) || subtle.ConstantTimeCompare([]byte(setup.ownerToken), []byte(owner)) != 1 {
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
	ownerExpiry, ownerValid := h.sessions[owner]
	if setup == nil || !setup.inUse || setup.epoch != h.authEpoch || !now.Before(setup.expiresAt) || !ownerValid || !now.Before(ownerExpiry) || subtle.ConstantTimeCompare([]byte(setup.ownerToken), []byte(owner)) != 1 {
		return errInvalidAuthState
	}
	if err := h.store.EnableTOTP(encrypted, hashes, step); err != nil {
		setup.inUse = false
		return err
	}
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

func (h *AdminHandler) pruneSessionsLocked(now time.Time) {
	for token, expiresAt := range h.sessions {
		if !now.Before(expiresAt) {
			delete(h.sessions, token)
		}
	}
}

func (h *AdminHandler) pruneLocked(now time.Time) {
	h.pruneSessionsLocked(now)
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
	h.sessions = make(map[string]time.Time)
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
