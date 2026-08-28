package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/store"
)

func TestLegacyLoginReturnsExactSessionResponse(t *testing.T) {
	h := newTestAdminHandler(t)
	resp := performJSON(t, h.Login, http.MethodPost, "/api/admin/login", `{"username":"admin","password":"password"}`, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("Login() code = %d: %s", resp.Code, resp.Body.String())
	}
	var result map[string]interface{}
	decodeResponse(t, resp, &result)
	if len(result) != 3 || result["username"] != "admin" || result["role"] != "admin" || !validOpaqueToken(result["token"].(string)) {
		t.Fatalf("Login() response = %#v, want exact LoginResponse", result)
	}
}

func TestTOTPLoginChallengeAndSingleUse(t *testing.T) {
	h, secret, _ := newEnabledAdminHandler(t)
	challenge := beginChallenge(t, h)

	wrong := performJSON(t, h.LoginTwoFactor, http.MethodPost, "/api/admin/login/2fa", `{"challenge_token":"`+challenge+`","code":"000000"}`, "")
	if wrong.Code != http.StatusUnauthorized {
		t.Fatalf("wrong code = %d: %s", wrong.Code, wrong.Body.String())
	}
	code := auth.GenerateTOTPCode(secret, h.now().Unix()/30)
	good := performJSON(t, h.LoginTwoFactor, http.MethodPost, "/api/admin/login/2fa", `{"challenge_token":"`+challenge+`","code":"`+code+`"}`, "")
	if good.Code != http.StatusOK {
		t.Fatalf("correct code = %d: %s", good.Code, good.Body.String())
	}
	reused := performJSON(t, h.LoginTwoFactor, http.MethodPost, "/api/admin/login/2fa", `{"challenge_token":"`+challenge+`","code":"`+code+`"}`, "")
	if reused.Code != http.StatusUnauthorized {
		t.Fatalf("reused challenge = %d", reused.Code)
	}

	secondChallenge := beginChallenge(t, h)
	replay := performJSON(t, h.LoginTwoFactor, http.MethodPost, "/api/admin/login/2fa", `{"challenge_token":"`+secondChallenge+`","code":"`+code+`"}`, "")
	if replay.Code != http.StatusUnauthorized || !strings.Contains(replay.Body.String(), "invalid authentication code") {
		t.Fatalf("replayed TOTP = %d: %s", replay.Code, replay.Body.String())
	}
}

func TestTOTPChallengeExpiresAndStopsAfterFiveAttempts(t *testing.T) {
	h, _, _ := newEnabledAdminHandler(t)
	challenge := beginChallenge(t, h)
	h.now = func() time.Time { return time.Date(2026, 7, 29, 12, 5, 0, 0, time.UTC) }
	expired := performJSON(t, h.LoginTwoFactor, http.MethodPost, "", `{"challenge_token":"`+challenge+`","code":"000000"}`, "")
	if expired.Code != http.StatusUnauthorized {
		t.Fatalf("expired challenge = %d", expired.Code)
	}

	h.now = func() time.Time { return time.Date(2026, 7, 29, 13, 0, 0, 0, time.UTC) }
	challenge = beginChallenge(t, h)
	for i := 0; i < maxFactorAttempts; i++ {
		resp := performJSON(t, h.LoginTwoFactor, http.MethodPost, "", `{"challenge_token":"`+challenge+`","code":"000000"}`, "")
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d = %d", i+1, resp.Code)
		}
	}
	if _, ok := h.challenges[challenge]; ok {
		t.Fatal("challenge survived five wrong attempts")
	}
}

func TestRecoveryCodeWorksOnce(t *testing.T) {
	h, _, recovery := newEnabledAdminHandler(t)
	challenge := beginChallenge(t, h)
	resp := performJSON(t, h.LoginTwoFactor, http.MethodPost, "", `{"challenge_token":"`+challenge+`","code":"`+recovery+`"}`, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("recovery login = %d: %s", resp.Code, resp.Body.String())
	}
	challenge = beginChallenge(t, h)
	resp = performJSON(t, h.LoginTwoFactor, http.MethodPost, "", `{"challenge_token":"`+challenge+`","code":"`+recovery+`"}`, "")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("reused recovery code = %d", resp.Code)
	}
}

func TestLoginTwoFactorDoesNotIssueSessionAfterConcurrentRevocation(t *testing.T) {
	h := newTestAdminHandler(t)
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	h.now = func() time.Time { return now }
	challengeToken := strings.Repeat("a", 64)
	claimed := &loginChallenge{expiresAt: now.Add(time.Minute), inUse: true}
	h.challenges[challengeToken] = claimed

	revoked := make(chan struct{})
	go func() {
		h.revokeAllAuthState()
		close(revoked)
	}()
	<-revoked

	sessionToken := strings.Repeat("b", 64)
	if valid, err := h.finishChallenge(challengeToken, claimed, sessionToken, &preparedFactor{}); err != nil || valid {
		t.Fatalf("finishChallenge after revocation = (%v, %v), want false, nil", valid, err)
	}
	if h.ValidateToken(sessionToken) || len(h.challenges) != 0 {
		t.Fatalf("post-revocation auth state: %d challenges", len(h.challenges))
	}
}

func TestEnabledLoginMissingOrWrongEncryptionKeyFailsClosed(t *testing.T) {
	good, _, _ := newEnabledAdminHandler(t)
	for _, key := range [][]byte{nil, bytes.Repeat([]byte{9}, 32)} {
		h := NewAdminHandler(good.store, key)
		resp := performJSON(t, h.Login, http.MethodPost, "", `{"username":"admin","password":"password"}`, "")
		if resp.Code != http.StatusServiceUnavailable {
			t.Fatalf("key length %d login = %d: %s", len(key), resp.Code, resp.Body.String())
		}
		if len(h.challenges) != 0 {
			t.Fatal("unusable key created challenge")
		}
	}
}

func TestSetupConfirmStatusAndDisable(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	h := newTestAdminHandler(t, key)
	session := login(t, h, "admin", "password")

	setupResp := performJSON(t, h.SetupTOTP, http.MethodPost, "", `{}`, session)
	if setupResp.Code != http.StatusOK {
		t.Fatalf("setup = %d: %s", setupResp.Code, setupResp.Body.String())
	}
	var setup models.TOTPSetupResponse
	decodeResponse(t, setupResp, &setup)
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(setup.Secret)
	if err != nil || len(secret) != 20 {
		t.Fatalf("setup secret = %q, err %v", setup.Secret, err)
	}

	// Another user must not be able to confirm someone else's setup; a
	// second session of the same user still may (asserted below).
	if err := h.store.CreateUser(models.User{
		Username:       "member",
		PasswordHash:   store.HashPassword("password"),
		Role:           models.RoleUser,
		Status:         models.UserActive,
		EmailVerified:  true,
	}); err != nil {
		t.Fatal(err)
	}
	memberSession := login(t, h, "member", "password")
	code := auth.GenerateTOTPCode(secret, h.now().Unix()/30)
	wrongOwner := performJSON(t, h.ConfirmTOTP, http.MethodPost, "", `{"setup_token":"`+setup.SetupToken+`","code":"`+code+`"}`, memberSession)
	if wrongOwner.Code != http.StatusUnauthorized {
		t.Fatalf("wrong owner confirm = %d", wrongOwner.Code)
	}
	confirm := performJSON(t, h.ConfirmTOTP, http.MethodPost, "", `{"setup_token":"`+setup.SetupToken+`","code":"`+code+`"}`, session)
	if confirm.Code != http.StatusOK {
		t.Fatalf("confirm = %d: %s", confirm.Code, confirm.Body.String())
	}
	var confirmed models.TOTPConfirmResponse
	decodeResponse(t, confirm, &confirmed)
	if !confirmed.Enabled || len(confirmed.RecoveryCodes) != 10 {
		t.Fatalf("confirm response = %#v", confirmed)
	}
	// The owner's sessions are revoked; other users stay logged in.
	if h.ValidateToken(session) || !h.ValidateToken(memberSession) || len(h.setups) != 0 || len(h.challenges) != 0 {
		t.Fatal("confirmation revoked sessions incorrectly")
	}

	session = completeTOTPLogin(t, h, secret)
	status := performJSON(t, h.TOTPStatus, http.MethodGet, "", "", session)
	var state models.TOTPStatusResponse
	decodeResponse(t, status, &state)
	if !state.Enabled || state.RecoveryCodesRemaining != 10 || state.SetupAvailable {
		t.Fatalf("status = %#v", state)
	}

	disable := performJSON(t, h.DisableTOTP, http.MethodPost, "", `{"current_password":"password","code":"`+confirmed.RecoveryCodes[0]+`"}`, session)
	if disable.Code != http.StatusOK {
		t.Fatalf("disable = %d: %s", disable.Code, disable.Body.String())
	}
	if h.ValidateToken(session) {
		t.Fatal("disable did not revoke session")
	}
	enabled, secretText, step, count := h.store.GetTOTPState(h.store.AdminUserID())
	if enabled || secretText != "" || step != 0 || count != 0 {
		t.Fatalf("disabled state = %v %q %d %d", enabled, secretText, step, count)
	}
}

func TestDisableRecoveryStillRequiresUsableEncryptionKey(t *testing.T) {
	h, secret, recovery := newEnabledAdminHandler(t)
	session := completeTOTPLogin(t, h, secret)
	h.encryptionKey = nil
	resp := performJSON(t, h.DisableTOTP, http.MethodPost, "", `{"current_password":"password","code":"`+recovery+`"}`, session)
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("disable without key = %d: %s", resp.Code, resp.Body.String())
	}
}

func TestLogoutDeletesPendingSetupsOwnedByUser(t *testing.T) {
	h := newTestAdminHandler(t, bytes.Repeat([]byte{7}, 32))
	// A second regular account so ownership boundaries are exercised.
	if err := h.store.CreateUser(models.User{
		Username:       "member",
		PasswordHash:   store.HashPassword("password"),
		Role:           models.RoleUser,
		Status:         models.UserActive,
		EmailVerified:  true,
	}); err != nil {
		t.Fatal(err)
	}
	adminToken := login(t, h, "admin", "password")
	memberToken := login(t, h, "member", "password")
	adminSetup, _, err := h.addSetup(h.store.AdminUserID(), hashToken(adminToken), bytes.Repeat([]byte{1}, 20))
	if err != nil {
		t.Fatal(err)
	}
	member, _ := h.store.GetUserByUsername("member")
	memberSetup, _, err := h.addSetup(member.ID, hashToken(memberToken), bytes.Repeat([]byte{2}, 20))
	if err != nil {
		t.Fatal(err)
	}

	resp := performJSON(t, h.Logout, http.MethodPost, "", "", adminToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("logout = %d: %s", resp.Code, resp.Body.String())
	}
	if h.ValidateToken(adminToken) {
		t.Fatal("logged-out session remains valid")
	}
	if _, ok := h.setups[adminSetup]; ok {
		t.Fatal("logout preserved setup owned by logged-out user")
	}
	if _, ok := h.setups[memberSetup]; !ok {
		t.Fatal("logout deleted setup owned by another user")
	}
}

func TestAddSetupReplacesOwnersPreviousSetupBeforeCapacityCheck(t *testing.T) {
	h := newTestAdminHandler(t)
	owner := strings.Repeat("a", 64)
	previous, _, err := h.addSetup(owner, hashToken(strings.Repeat("c", 64)), bytes.Repeat([]byte{1}, 20))
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < maxPendingSetups; i++ {
		h.setups[strings.Repeat("x", i)] = &pendingTOTPSetup{
			ownerUserID: strings.Repeat("b", 63) + string(rune('A'+i)),
			secret:      bytes.Repeat([]byte{2}, 20),
			expiresAt:   h.now().Add(time.Minute),
		}
	}

	replacement, _, err := h.addSetup(owner, hashToken(strings.Repeat("c", 64)), bytes.Repeat([]byte{3}, 20))
	if err != nil {
		t.Fatalf("replacement at capacity: %v", err)
	}
	if len(h.setups) != maxPendingSetups {
		t.Fatalf("setup count = %d, want %d", len(h.setups), maxPendingSetups)
	}
	if _, ok := h.setups[previous]; ok {
		t.Fatal("previous setup remains after replacement")
	}
	if setup := h.setups[replacement]; setup == nil || setup.ownerUserID != owner {
		t.Fatal("replacement setup was not added for owner")
	}
}

func TestInputAndCapacityLimits(t *testing.T) {
	h := newTestAdminHandler(t)
	oversized := `{"username":"admin","password":"` + strings.Repeat("x", maxRequestBodyBytes) + `"}`
	resp := performJSON(t, h.Login, http.MethodPost, "", oversized, "")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("oversized login = %d", resp.Code)
	}
	resp = performJSON(t, h.Login, http.MethodPost, "", `{"username":"`+strings.Repeat("a", maxUsernameLength+1)+`","password":"password"}`, "")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("long username = %d", resp.Code)
	}

	h, _, _ = newEnabledAdminHandler(t)
	h.challenges = make(map[string]*loginChallenge, maxChallenges)
	for i := 0; i < maxChallenges; i++ {
		h.challenges[strings.Repeat("x", i+1)] = &loginChallenge{expiresAt: h.now().Add(time.Minute)}
	}
	resp = performJSON(t, h.Login, http.MethodPost, "", `{"username":"admin","password":"password"}`, "")
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("full challenge capacity = %d", resp.Code)
	}
}

func TestChangeUsernamePreservesMigratedHash(t *testing.T) {
	legacy := sha256.Sum256([]byte("password"))
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
	if err := st.SetAdminCredentials("admin", hexDigest(legacy[:])); err != nil {
		t.Fatal(err)
	}
	h := NewAdminHandler(st)
	session := login(t, h, "admin", "password")
	resp := performJSON(t, h.ChangeUsername, http.MethodPut, "", `{"current_password":"password","new_username":"new-admin"}`, session)
	if resp.Code != http.StatusOK {
		t.Fatalf("ChangeUsername() = %d: %s", resp.Code, resp.Body.String())
	}
	username, hash := st.GetAdminCredentials()
	if username != "new-admin" || !strings.HasPrefix(hash, "$argon2id$") || !st.ValidatePassword("password", hash) {
		t.Fatalf("credentials after username change = %q %q", username, hash)
	}
}

func TestSessionExpiresAndCredentialChangesRevokeAllState(t *testing.T) {
	h := newTestAdminHandler(t)
	// Anchor on the real clock: the store prunes sessions against it.
	base := time.Now()
	h.now = func() time.Time { return base }
	h.tokenTTL = time.Hour
	token := login(t, h, "admin", "password")
	h.now = func() time.Time { return base.Add(time.Hour) }
	if h.ValidateToken(token) {
		t.Fatal("expired session remains valid")
	}

	first := login(t, h, "admin", "password")
	second := login(t, h, "admin", "password")
	h.challenges["pending"] = &loginChallenge{expiresAt: base.Add(2 * time.Hour)}
	resp := performJSON(t, h.ChangePassword, http.MethodPut, "", `{"current_password":"password","new_password":"new-password"}`, first)
	if resp.Code != http.StatusOK || h.ValidateToken(first) || h.ValidateToken(second) || len(h.challenges) != 0 {
		t.Fatalf("password change did not revoke state: %d", resp.Code)
	}
}

func TestLoginEpochPreventsSessionAfterRevocation(t *testing.T) {
	h := newTestAdminHandler(t)
	epoch := h.currentAuthEpoch()
	h.revokeAllAuthState()
	if token, err := h.createSession(h.store.AdminUserID(), epoch); !errors.Is(err, errInvalidAuthState) || token != "" {
		t.Fatalf("createSession with stale epoch = (%q, %v)", token, err)
	}
}

func TestConfirmCannotEnableAfterLogout(t *testing.T) {
	h := newTestAdminHandler(t, bytes.Repeat([]byte{7}, 32))
	owner := login(t, h, "admin", "password")
	uid := h.store.AdminUserID()
	secret := bytes.Repeat([]byte{2}, 20)
	setupToken, _, err := h.addSetup(uid, hashToken(owner), secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := h.claimSetup(setupToken, uid); !ok {
		t.Fatal("claimSetup failed")
	}
	performJSON(t, h.Logout, http.MethodPost, "", "", owner)
	encrypted, err := auth.EncryptTOTPSecret(h.encryptionKey, secret)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.finalizeSetup(setupToken, uid, encrypted, []string{"hash"}, 1); !errors.Is(err, errInvalidAuthState) {
		t.Fatalf("finalizeSetup after logout error = %v", err)
	}
	if enabled, _, _, _ := h.store.GetTOTPState(uid); enabled {
		t.Fatal("TOTP enabled after setup owner logout")
	}
}

func TestLogoutDoesNotInvalidateOtherLoginChallenges(t *testing.T) {
	h, secret, _ := newEnabledAdminHandler(t)
	challenge := beginChallenge(t, h)
	otherSession := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if err := h.store.CreateSession(hashToken(otherSession), h.store.AdminUserID(), h.now().Add(time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}

	performJSON(t, h.Logout, http.MethodPost, "", "", otherSession)
	code := auth.GenerateTOTPCode(secret, h.now().Unix()/30)
	resp := performJSON(t, h.LoginTwoFactor, http.MethodPost, "", `{"challenge_token":"`+challenge+`","code":"`+code+`"}`, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("challenge after unrelated logout = %d: %s", resp.Code, resp.Body.String())
	}
}

func TestPasswordVerificationSaturationReturnsServiceUnavailable(t *testing.T) {
	h := newTestAdminHandler(t)
	for i := 0; i < maxPasswordVerifies; i++ {
		if !h.acquirePasswordVerify() {
			t.Fatal("failed to fill verification semaphore")
		}
	}
	defer func() {
		for i := 0; i < maxPasswordVerifies; i++ {
			h.releasePasswordVerify()
		}
	}()
	resp := performJSON(t, h.Login, http.MethodPost, "", `{"username":"admin","password":"password"}`, "")
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("saturated Login() = %d: %s", resp.Code, resp.Body.String())
	}
}

func TestChangePasswordSaveFailureDoesNotRevokeOrReportSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	st := store.NewStore(path)
	if err := st.SetAdminCredentials("admin", store.HashPassword("password")); err != nil {
		t.Fatal(err)
	}
	h := NewAdminHandler(st)
	token := login(t, h, "admin", "password")
	// Removing the whole directory takes the SQLite database with it, so the
	// next save fails exactly like the old JSON store losing its file.
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	resp := performJSON(t, h.ChangePassword, http.MethodPut, "", `{"current_password":"password","new_password":"new-password"}`, token)
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("ChangePassword() = %d: %s", resp.Code, resp.Body.String())
	}
	if !h.ValidateToken(token) {
		t.Fatal("failed password save revoked session")
	}
	_, hash := h.store.GetAdminCredentials()
	if !h.store.ValidatePassword("password", hash) || h.store.ValidatePassword("new-password", hash) {
		t.Fatal("failed password save changed stored password")
	}
}

func newTestAdminHandler(t *testing.T, keys ...[]byte) *AdminHandler {
	t.Helper()
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
	if err := st.SetAdminCredentials("admin", store.HashPassword("password")); err != nil {
		t.Fatal(err)
	}
	return NewAdminHandler(st, keys...)
}

func newEnabledAdminHandler(t *testing.T) (*AdminHandler, []byte, string) {
	t.Helper()
	key := bytes.Repeat([]byte{3}, 32)
	h := newTestAdminHandler(t, key)
	h.now = func() time.Time { return time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC) }
	secret := bytes.Repeat([]byte{4}, 20)
	encrypted, err := auth.EncryptTOTPSecret(key, secret)
	if err != nil {
		t.Fatal(err)
	}
	codes, hashes, err := auth.GenerateRecoveryCodes(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.store.EnableTOTP(h.store.AdminUserID(), encrypted, hashes, h.now().Unix()/30-2); err != nil {
		t.Fatal(err)
	}
	return h, secret, codes[0]
}

func beginChallenge(t *testing.T, h *AdminHandler) string {
	t.Helper()
	resp := performJSON(t, h.Login, http.MethodPost, "", `{"username":"admin","password":"password"}`, "")
	if resp.Code != http.StatusAccepted {
		t.Fatalf("challenge login = %d: %s", resp.Code, resp.Body.String())
	}
	var result models.TwoFactorChallengeResponse
	decodeResponse(t, resp, &result)
	if !result.TwoFactorRequired || !validOpaqueToken(result.ChallengeToken) {
		t.Fatalf("challenge response = %#v", result)
	}
	return result.ChallengeToken
}

func completeTOTPLogin(t *testing.T, h *AdminHandler, secret []byte) string {
	t.Helper()
	challenge := beginChallenge(t, h)
	code := auth.GenerateTOTPCode(secret, h.now().Unix()/30+1)
	resp := performJSON(t, h.LoginTwoFactor, http.MethodPost, "", `{"challenge_token":"`+challenge+`","code":"`+code+`"}`, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("complete login = %d: %s", resp.Code, resp.Body.String())
	}
	var result models.LoginResponse
	decodeResponse(t, resp, &result)
	return result.Token
}

func login(t *testing.T, h *AdminHandler, username, password string) string {
	t.Helper()
	body, _ := json.Marshal(models.LoginRequest{Username: username, Password: password})
	resp := performJSON(t, h.Login, http.MethodPost, "", string(body), "")
	if resp.Code != http.StatusOK {
		t.Fatalf("Login() code = %d: %s", resp.Code, resp.Body.String())
	}
	var result models.LoginResponse
	decodeResponse(t, resp, &result)
	return result.Token
}

func performJSON(t *testing.T, handler http.HandlerFunc, method, path, body, session string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	if path == "" {
		path = "/"
	}
	req := httptest.NewRequest(method, path, reader)
	if session != "" {
		req.Header.Set("X-Auth-Token", session)
	}
	resp := httptest.NewRecorder()
	handler(resp, req)
	return resp
}

func decodeResponse(t *testing.T, resp *httptest.ResponseRecorder, target interface{}) {
	t.Helper()
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatal(err)
	}
}

func hexDigest(data []byte) string {
	const chars = "0123456789abcdef"
	out := make([]byte, len(data)*2)
	for i, value := range data {
		out[i*2] = chars[value>>4]
		out[i*2+1] = chars[value&15]
	}
	return string(out)
}
