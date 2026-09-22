package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

// maxPasskeyNameLength bounds the user supplied credential label.
const maxPasskeyNameLength = 60

// PasskeyHandler serves passkey registration, passkey sign-in and the two
// password-login switches (per account and panel-wide).
type PasskeyHandler struct {
	store    *store.Store
	passkeys *services.PasskeyService
	admin    *AdminHandler
}

// NewPasskeyHandler creates the passkey handler. admin supplies the session and
// login-challenge machinery shared with the password flow.
func NewPasskeyHandler(st *store.Store, passkeys *services.PasskeyService, admin *AdminHandler) *PasskeyHandler {
	return &PasskeyHandler{store: st, passkeys: passkeys, admin: admin}
}

// guardPasskey applies the rate limit to one passkey request.
//
// The ceremony does not name an account until it finishes, so the address
// budget is the one that applies: it is enough to stop a caller minting
// ceremony state or replaying credentials in a loop.
func (h *PasskeyHandler) guardPasskey(w http.ResponseWriter, r *http.Request) bool {
	if h.admin == nil || h.admin.throttle == nil {
		return true
	}
	if ok, wait := h.admin.throttle.Guard(r, "", ""); !ok {
		writeTooManyAttempts(w, wait)
		return false
	}
	return true
}

// chargePasskey books one passkey request. userID is empty when the ceremony
// has not resolved an account, which leaves only the address budget.
func (h *PasskeyHandler) chargePasskey(r *http.Request, userID string) {
	if h.admin != nil && h.admin.throttle != nil {
		h.admin.throttle.Fail(r, "", userID)
	}
}

// ---------------------------------------------------------------------------
// Account management (session required)

// Settings handles GET /api/account/passkeys.
func (h *PasskeyHandler) Settings(w http.ResponseWriter, r *http.Request) {
	user := SessionUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, h.settingsView(r, user.ID))
}

// BeginRegistration handles POST /api/account/passkeys/begin. The current
// password is required so a stolen session cannot quietly add a credential.
func (h *PasskeyHandler) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	user := SessionUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req models.PasskeyBeginRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if !h.confirmPassword(w, r, user.ID, req.Password) {
		return
	}
	options, token, _, err := h.passkeys.BeginRegistration(r, user.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	// The library wraps the options in a "publicKey" envelope; clients expect
	// the bare WebAuthn dictionary.
	writeJSON(w, http.StatusOK, map[string]interface{}{"ceremony_token": token, "public_key": options.Response})
}

// FinishRegistration handles POST /api/account/passkeys/finish.
func (h *PasskeyHandler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	user := SessionUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req models.PasskeyFinishRequest
	if err := readJSON(r, &req); err != nil || len(req.Credential) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	credential, _, err := h.passkeys.FinishRegistration(r, strings.TrimSpace(req.CeremonyToken), user.ID, req.Credential)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	encoded, err := services.EncodePasskeyCredential(credential)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "保存通行密钥失败"})
		return
	}
	name := trimPasskeyName(req.Name)
	if name == "" {
		existing, _ := h.store.ListPasskeys(user.ID)
		name = fmt.Sprintf("通行密钥 %d", len(existing)+1)
	}
	entry := models.Passkey{
		ID:         services.PasskeyCredentialID(credential),
		UserID:     user.ID,
		Name:       name,
		Credential: encoded,
		CreatedAt:  time.Now().Unix(),
	}
	if err := h.store.AddPasskey(entry); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "保存通行密钥失败"})
		return
	}
	writeJSON(w, http.StatusCreated, passkeyView(entry))
}

// Rename handles PUT /api/account/passkeys/{id}.
func (h *PasskeyHandler) Rename(w http.ResponseWriter, r *http.Request) {
	user := SessionUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req models.PasskeyRenameRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	name := trimPasskeyName(req.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "名称不能为空"})
		return
	}
	if err := h.store.RenamePasskey(chi.URLParam(r, "id"), user.ID, name); err != nil {
		writePasskeyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Delete handles DELETE /api/account/passkeys/{id}. Removing the last
// credential of an account that disabled password sign-in would lock it out, so
// that combination is refused.
func (h *PasskeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := SessionUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req models.PasskeyDeleteRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if !h.confirmPassword(w, r, user.ID, req.Password) {
		return
	}
	if account, ok := h.store.GetUserByID(user.ID); ok && account.PasswordLoginDisabled {
		if count, err := h.store.CountPasskeys(user.ID); err == nil && count <= 1 {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "该账户已禁用密码登录，请先恢复密码登录再删除最后一个通行密钥"})
			return
		}
	}
	id := chi.URLParam(r, "id")
	if err := h.store.DeletePasskey(id, user.ID); err != nil {
		writePasskeyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// UpdatePasswordLogin handles PUT /api/account/password-login.
func (h *PasskeyHandler) UpdatePasswordLogin(w http.ResponseWriter, r *http.Request) {
	user := SessionUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req models.PasswordLoginRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if !h.confirmPassword(w, r, user.ID, req.Password) {
		return
	}
	if req.Disabled {
		count, err := h.store.CountPasskeys(user.ID)
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "操作失败"})
			return
		}
		if count == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请先绑定至少一个通行密钥，再禁用密码登录"})
			return
		}
	}
	if err := h.store.SetUserPasswordLoginDisabled(user.ID, req.Disabled); err != nil {
		writePasskeyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, h.settingsView(r, user.ID))
}

// AdminUpdatePasswordLogin handles PUT /api/admin/users/{id}/password-login, so
// an administrator can restore password sign-in for a locked-out account.
func (h *PasskeyHandler) AdminUpdatePasswordLogin(w http.ResponseWriter, r *http.Request) {
	var req models.AdminPasswordLoginRequest
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	id := chi.URLParam(r, "id")
	if _, ok := h.store.GetUserByID(id); !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "账户不存在"})
		return
	}
	if req.Disabled {
		count, err := h.store.CountPasskeys(id)
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "操作失败"})
			return
		}
		if count == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "该账户尚未绑定通行密钥，无法禁用密码登录"})
			return
		}
	}
	if err := h.store.SetUserPasswordLoginDisabled(id, req.Disabled); err != nil {
		writePasskeyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Android Digital Asset Links

// AssetLinks handles GET /.well-known/assetlinks.json. Android fetches this
// document before Credential Manager will create or use a passkey for the
// panel's domain, so without it the native app cannot sign in with a passkey.
func (h *PasskeyHandler) AssetLinks(w http.ResponseWriter, r *http.Request) {
	settings := h.store.GetAppSettings()
	fingerprints := settings.AndroidFingerprints()
	if len(fingerprints) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "未配置有效的 Android 签名指纹，请在「系统设置 → 通行密钥」填写"})
		return
	}
	document := []map[string]interface{}{{
		// handle_all_urls is what Credential Manager checks for passkeys;
		// get_login_creds is kept for the older Smart Lock flow.
		"relation": []string{
			"delegate_permission/common.handle_all_urls",
			"delegate_permission/common.get_login_creds",
		},
		"target": map[string]interface{}{
			"namespace":                "android_app",
			"package_name":             settings.AndroidPackage(),
			"sha256_cert_fingerprints": fingerprints,
		},
	}}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	if err := json.NewEncoder(w).Encode(document); err != nil {
		log.Printf("write asset links: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Sign-in (no session yet)

// LoginBegin handles POST /api/auth/passkey/login/begin.
func (h *PasskeyHandler) LoginBegin(w http.ResponseWriter, r *http.Request) {
	if !h.guardPasskey(w, r) {
		return
	}
	var req models.PasskeyLoginBeginRequest
	if err := readJSON(r, &req); err != nil {
		// An empty body simply means "offer every passkey".
		req = models.PasskeyLoginBeginRequest{}
	}

	// Naming an account only narrows the list the browser offers. An unknown,
	// disabled or credential-less account falls back to the usernameless flow
	// so the response never reveals whether an account exists.
	userID := ""
	if account := strings.TrimSpace(req.Account); account != "" {
		if user, ok := h.admin.lookupUser(account); ok && user.Status == models.UserActive {
			if count, countErr := h.store.CountPasskeys(user.ID); countErr == nil && count > 0 {
				userID = user.ID
			}
		}
	}

	var (
		options *protocol.CredentialAssertion
		token   string
		err     error
	)
	if userID != "" {
		options, token, _, err = h.passkeys.BeginLogin(r, userID)
	} else {
		options, token, _, err = h.passkeys.BeginDiscoverableLogin(r)
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ceremony_token": token, "public_key": options.Response})
}

// LoginFinish handles POST /api/auth/passkey/login/finish. A passkey assertion
// is itself a strong factor, so it completes the sign-in without a second step.
func (h *PasskeyHandler) LoginFinish(w http.ResponseWriter, r *http.Request) {
	if !h.guardPasskey(w, r) {
		return
	}
	var req models.PasskeyLoginFinishRequest
	if err := readJSON(r, &req); err != nil || len(req.Credential) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	userID, credential, _, err := h.passkeys.FinishLogin(r, strings.TrimSpace(req.CeremonyToken), "", req.Credential)
	if err != nil {
		h.failPasskeyLogin(r, "", err)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	user, ok := h.store.GetUserByID(userID)
	if !ok || user.Status != models.UserActive {
		h.failPasskeyLogin(r, userID, errors.New("account unavailable"))
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "account is disabled"})
		return
	}
	h.refreshCredential(credential)
	h.admin.completeLogin(w, r, user)
}

// TwoFactorBegin handles POST /api/admin/login/2fa/passkey/begin: the password
// step already succeeded, so this only proves possession of a passkey.
func (h *PasskeyHandler) TwoFactorBegin(w http.ResponseWriter, r *http.Request) {
	if !h.guardPasskey(w, r) {
		return
	}
	var req models.PasskeyTwoFactorBeginRequest
	if err := readJSON(r, &req); err != nil || !validOpaqueToken(req.ChallengeToken) {
		writeInvalidFactor(w)
		return
	}
	challenge, ok := h.admin.claimChallenge(req.ChallengeToken)
	if !ok {
		writeInvalidFactor(w)
		return
	}
	options, token, _, err := h.passkeys.BeginLogin(r, challenge.userID)
	if err != nil {
		h.admin.releaseChallenge(req.ChallengeToken, challenge)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ceremony_token": token, "public_key": options.Response})
}

// TwoFactorFinish handles POST /api/admin/login/2fa/passkey/finish.
func (h *PasskeyHandler) TwoFactorFinish(w http.ResponseWriter, r *http.Request) {
	if !h.guardPasskey(w, r) {
		return
	}
	var req models.PasskeyTwoFactorFinishRequest
	if err := readJSON(r, &req); err != nil || !validOpaqueToken(req.ChallengeToken) || len(req.Credential) == 0 {
		writeInvalidFactor(w)
		return
	}
	challenge, ok := h.admin.claimChallenge(req.ChallengeToken)
	if !ok {
		writeInvalidFactor(w)
		return
	}
	userID, credential, _, err := h.passkeys.FinishLogin(r, strings.TrimSpace(req.CeremonyToken), challenge.userID, req.Credential)
	if err != nil {
		h.admin.failChallenge(req.ChallengeToken, challenge)
		h.failPasskeyLogin(r, challenge.userID, err)
		writeInvalidFactor(w)
		return
	}
	sessionToken, err := generateToken()
	if err != nil {
		h.admin.releaseChallenge(req.ChallengeToken, challenge)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "authentication temporarily unavailable"})
		return
	}
	valid, internalErr := h.admin.finishChallenge(req.ChallengeToken, challenge, sessionToken, nil)
	if internalErr != nil {
		h.admin.releaseChallenge(req.ChallengeToken, challenge)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor authentication unavailable"})
		return
	}
	if !valid {
		writeInvalidFactor(w)
		return
	}
	user, ok := h.store.GetUserByID(userID)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "two-factor authentication unavailable"})
		return
	}
	h.refreshCredential(credential)
	h.admin.finishLogin(w, r, user, sessionToken)
}

// ---------------------------------------------------------------------------
// Helpers

// settingsView assembles one account's passwordless state.
func (h *PasskeyHandler) settingsView(r *http.Request, userID string) models.PasskeySettingsView {
	view := models.PasskeySettingsView{Passkeys: []models.PasskeyView{}}
	entries, err := h.store.ListPasskeys(userID)
	if err != nil {
		log.Printf("list passkeys: %v", err)
	}
	for _, entry := range entries {
		view.Passkeys = append(view.Passkeys, passkeyView(entry))
	}
	if account, ok := h.store.GetUserByID(userID); ok {
		view.PasswordLoginDisabled = account.PasswordLoginDisabled
	}
	view.PasswordLoginDisabledGlobally = h.store.GetAppSettings().PasswordLoginDisabled
	if rp, rpErr := h.passkeys.ResolveRelyingParty(r); rpErr == nil {
		view.RPID = rp.ID
		view.Available = true
		if len(rp.Origins) > 0 {
			view.Origin = rp.Origins[0]
		}
	} else {
		log.Printf("resolve passkey relying party: %v", rpErr)
	}
	return view
}

// confirmPassword re-authenticates the session owner before a credential or a
// password switch changes. It writes the error response itself.
func (h *PasskeyHandler) confirmPassword(w http.ResponseWriter, r *http.Request, userID, password string) bool {
	account, ok := h.store.GetUserByID(userID)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return false
	}
	if password == "" || !h.store.ValidateUserPassword(userID, password, account.PasswordHash) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "密码不正确"})
		return false
	}
	return true
}

// refreshCredential persists the signature counter the authenticator reported,
// which is what makes clone detection work on the next assertion.
func (h *PasskeyHandler) refreshCredential(credential *webauthn.Credential) {
	if credential == nil {
		return
	}
	encoded, err := services.EncodePasskeyCredential(credential)
	if err != nil {
		log.Printf("encode passkey credential: %v", err)
		return
	}
	if err := h.store.UpdatePasskeyCredential(services.PasskeyCredentialID(credential), encoded, time.Now().Unix()); err != nil {
		log.Printf("update passkey credential: %v", err)
	}
}

// failPasskeyLogin charges a rejected assertion against the rate limit. The
// account may be unknown (for example a credential bound to a deleted account).
func (h *PasskeyHandler) failPasskeyLogin(r *http.Request, userID string, cause error) {
	log.Printf("passkey sign-in rejected: %v", cause)
	h.chargePasskey(r, userID)
}

// passkeyView projects a stored credential for the API, decoding the backup
// flags so the page can mark a synced passkey.
func passkeyView(entry models.Passkey) models.PasskeyView {
	view := models.PasskeyView{
		ID:         entry.ID,
		Name:       entry.Name,
		CreatedAt:  entry.CreatedAt,
		LastUsedAt: entry.LastUsedAt,
	}
	if credential, err := services.DecodePasskeyCredential(entry.Credential); err == nil {
		view.BackupEligible = credential.Flags.BackupEligible
		view.BackupState = credential.Flags.BackupState
	}
	return view
}

// trimPasskeyName caps a user supplied label without splitting a rune.
func trimPasskeyName(name string) string {
	runes := []rune(strings.TrimSpace(name))
	if len(runes) > maxPasskeyNameLength {
		return string(runes[:maxPasskeyNameLength])
	}
	return string(runes)
}

// writePasskeyError maps store errors onto HTTP status codes.
func writePasskeyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrPasskeyNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "通行密钥不存在"})
	case errors.Is(err, store.ErrUserNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "账户不存在"})
	default:
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "操作失败"})
	}
}
