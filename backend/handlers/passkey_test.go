package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

func newTestPasskeyHandler(t *testing.T) (*PasskeyHandler, *AdminHandler, *store.Store) {
	t.Helper()
	admin := newTestAdminHandler(t)
	return NewPasskeyHandler(admin.store, services.NewPasskeyService(admin.store), admin), admin, admin.store
}

// sessionHandler wraps a passkey route the way the router does.
func sessionHandler(admin *AdminHandler, next http.HandlerFunc) http.HandlerFunc {
	return (&Middleware{AdminHandler: admin}).SessionOnly(next)
}

// serveSession sends one request through a router so path parameters resolve.
func serveSession(t *testing.T, router http.Handler, method, path, body, session string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	if session != "" {
		req.Header.Set("X-Auth-Token", session)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func TestPasswordLoginBlockedWhenAccountSwitchOff(t *testing.T) {
	h, admin, st := newTestPasskeyHandler(t)
	_ = h
	adminID := st.AdminUserID()
	if err := st.AddPasskey(models.Passkey{ID: "cred-1", UserID: adminID, Name: "Phone"}); err != nil {
		t.Fatalf("AddPasskey() error = %v", err)
	}
	if err := st.SetUserPasswordLoginDisabled(adminID, true); err != nil {
		t.Fatalf("SetUserPasswordLoginDisabled() error = %v", err)
	}

	resp := performJSON(t, admin.Login, http.MethodPost, "/api/admin/login", `{"username":"admin","password":"password"}`, "")
	if resp.Code != http.StatusForbidden {
		t.Fatalf("Login() code = %d: %s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "禁用密码登录") {
		t.Fatalf("Login() body = %s, want the passkey hint", resp.Body.String())
	}

	page, err := st.QueryAuditLogs(models.AuditQuery{Action: models.AuditActionLoginFailed})
	if err != nil {
		t.Fatalf("QueryAuditLogs() error = %v", err)
	}
	if page.Total != 1 || page.Logs[0].ActorName != "admin" {
		t.Fatalf("audit entries = %#v, want one rejected password sign-in", page.Logs)
	}
}

func TestPasswordLoginBlockedWhenPanelSwitchOff(t *testing.T) {
	_, admin, st := newTestPasskeyHandler(t)
	settings := st.GetAppSettings()
	settings.PasswordLoginDisabled = true
	if err := st.SetAppSettings(settings); err != nil {
		t.Fatalf("SetAppSettings() error = %v", err)
	}

	resp := performJSON(t, admin.Login, http.MethodPost, "/api/admin/login", `{"username":"admin","password":"password"}`, "")
	if resp.Code != http.StatusForbidden {
		t.Fatalf("Login() code = %d: %s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "面板已禁用密码登录") {
		t.Fatalf("Login() body = %s, want the panel-wide hint", resp.Body.String())
	}
}

func TestUpdatePasswordLoginNeedsPasswordAndPasskey(t *testing.T) {
	h, admin, st := newTestPasskeyHandler(t)
	adminID := st.AdminUserID()
	session := login(t, admin, "admin", "password")
	router := chi.NewRouter()
	router.Put("/api/account/password-login", sessionHandler(admin, h.UpdatePasswordLogin))

	// Wrong password first: the switch must not move.
	resp := serveSession(t, router, http.MethodPut, "/api/account/password-login", `{"disabled":true,"password":"nope"}`, session)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("wrong password code = %d: %s", resp.Code, resp.Body.String())
	}
	// No passkey bound yet: disabling would lock the account out.
	resp = serveSession(t, router, http.MethodPut, "/api/account/password-login", `{"disabled":true,"password":"password"}`, session)
	if resp.Code != http.StatusBadRequest || !strings.Contains(resp.Body.String(), "通行密钥") {
		t.Fatalf("without passkey code = %d: %s", resp.Code, resp.Body.String())
	}
	if user, _ := st.GetUserByID(adminID); user.PasswordLoginDisabled {
		t.Fatal("password switch moved without a passkey")
	}

	if err := st.AddPasskey(models.Passkey{ID: "cred-1", UserID: adminID, Name: "Phone"}); err != nil {
		t.Fatalf("AddPasskey() error = %v", err)
	}
	resp = serveSession(t, router, http.MethodPut, "/api/account/password-login", `{"disabled":true,"password":"password"}`, session)
	if resp.Code != http.StatusOK {
		t.Fatalf("disable code = %d: %s", resp.Code, resp.Body.String())
	}
	var view models.PasskeySettingsView
	decodeResponse(t, resp, &view)
	if !view.PasswordLoginDisabled || len(view.Passkeys) != 1 {
		t.Fatalf("response = %#v, want the switch on with one credential", view)
	}
	if user, _ := st.GetUserByID(adminID); !user.PasswordLoginDisabled {
		t.Fatal("password switch did not persist")
	}

	// Re-enabling needs no passkey.
	resp = serveSession(t, router, http.MethodPut, "/api/account/password-login", `{"disabled":false,"password":"password"}`, session)
	if resp.Code != http.StatusOK {
		t.Fatalf("enable code = %d: %s", resp.Code, resp.Body.String())
	}
	if user, _ := st.GetUserByID(adminID); user.PasswordLoginDisabled {
		t.Fatal("password switch stayed on")
	}
}

func TestDeleteLastPasskeyRefusedWhilePasswordLoginDisabled(t *testing.T) {
	h, admin, st := newTestPasskeyHandler(t)
	adminID := st.AdminUserID()
	// Sign in before the switch closes the password door.
	session := login(t, admin, "admin", "password")
	if err := st.AddPasskey(models.Passkey{ID: "cred-1", UserID: adminID, Name: "Phone"}); err != nil {
		t.Fatalf("AddPasskey() error = %v", err)
	}
	if err := st.SetUserPasswordLoginDisabled(adminID, true); err != nil {
		t.Fatalf("SetUserPasswordLoginDisabled() error = %v", err)
	}

	router := chi.NewRouter()
	router.Delete("/api/account/passkeys/{id}", sessionHandler(admin, h.Delete))
	resp := serveSession(t, router, http.MethodDelete, "/api/account/passkeys/cred-1", `{"password":"password"}`, session)
	if resp.Code != http.StatusConflict {
		t.Fatalf("delete last credential code = %d: %s", resp.Code, resp.Body.String())
	}
	if count, _ := st.CountPasskeys(adminID); count != 1 {
		t.Fatalf("credential count = %d, want it kept", count)
	}

	// With a second credential, and with password sign-in restored, removal works.
	if err := st.AddPasskey(models.Passkey{ID: "cred-2", UserID: adminID, Name: "Laptop"}); err != nil {
		t.Fatalf("AddPasskey() error = %v", err)
	}
	resp = serveSession(t, router, http.MethodDelete, "/api/account/passkeys/cred-1", `{"password":"password"}`, session)
	if resp.Code != http.StatusOK {
		t.Fatalf("delete code = %d: %s", resp.Code, resp.Body.String())
	}
	if count, _ := st.CountPasskeys(adminID); count != 1 {
		t.Fatalf("credential count = %d, want 1 left", count)
	}
	if err := st.SetUserPasswordLoginDisabled(adminID, false); err != nil {
		t.Fatalf("SetUserPasswordLoginDisabled() error = %v", err)
	}
	resp = serveSession(t, router, http.MethodDelete, "/api/account/passkeys/cred-2", `{"password":"password"}`, session)
	if resp.Code != http.StatusOK {
		t.Fatalf("delete last credential code = %d: %s", resp.Code, resp.Body.String())
	}
	if count, _ := st.CountPasskeys(adminID); count != 0 {
		t.Fatalf("credential count = %d, want 0", count)
	}

	// Missing credential reports 404 rather than succeeding silently.
	resp = serveSession(t, router, http.MethodDelete, "/api/account/passkeys/cred-9", `{"password":"password"}`, session)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("delete unknown code = %d: %s", resp.Code, resp.Body.String())
	}
}

func TestAdminCanRestorePasswordLogin(t *testing.T) {
	h, admin, st := newTestPasskeyHandler(t)
	if err := st.CreateUser(models.User{Username: "bob", PasswordHash: store.HashPassword("password")}); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	bob, _ := st.GetUserByUsername("bob")
	if err := st.AddPasskey(models.Passkey{ID: "cred-bob", UserID: bob.ID, Name: "Phone"}); err != nil {
		t.Fatalf("AddPasskey() error = %v", err)
	}
	if err := st.SetUserPasswordLoginDisabled(bob.ID, true); err != nil {
		t.Fatalf("SetUserPasswordLoginDisabled() error = %v", err)
	}
	session := login(t, admin, "admin", "password")

	router := chi.NewRouter()
	router.Put("/api/admin/users/{id}/password-login", sessionHandler(admin, h.AdminUpdatePasswordLogin))
	resp := serveSession(t, router, http.MethodPut, "/api/admin/users/"+bob.ID+"/password-login", `{"disabled":false}`, session)
	if resp.Code != http.StatusOK {
		t.Fatalf("restore code = %d: %s", resp.Code, resp.Body.String())
	}
	if user, _ := st.GetUserByID(bob.ID); user.PasswordLoginDisabled {
		t.Fatal("administrator could not restore password sign-in")
	}
	// Bob can sign in again with his password.
	resp = performJSON(t, admin.Login, http.MethodPost, "/api/admin/login", `{"username":"bob","password":"password"}`, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("bob login code = %d: %s", resp.Code, resp.Body.String())
	}

	// Turning it back on for an account without credentials is refused.
	if err := st.DeletePasskey("cred-bob", bob.ID); err != nil {
		t.Fatalf("DeletePasskey() error = %v", err)
	}
	resp = serveSession(t, router, http.MethodPut, "/api/admin/users/"+bob.ID+"/password-login", `{"disabled":true}`, session)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("disable without credential code = %d: %s", resp.Code, resp.Body.String())
	}
}

func TestPasskeySignInBeginAndRejectedFinish(t *testing.T) {
	h, _, st := newTestPasskeyHandler(t)

	resp := performJSON(t, h.LoginBegin, http.MethodPost, "/api/auth/passkey/login/begin", `{}`, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("LoginBegin() code = %d: %s", resp.Code, resp.Body.String())
	}
	var begin struct {
		CeremonyToken string          `json:"ceremony_token"`
		PublicKey     json.RawMessage `json:"public_key"`
	}
	decodeResponse(t, resp, &begin)
	if begin.CeremonyToken == "" || len(begin.PublicKey) == 0 {
		t.Fatalf("LoginBegin() = %#v, want options and a ceremony token", begin)
	}

	// A credential the panel cannot parse is rejected, and the attempt lands in
	// the audit trail.
	body, _ := json.Marshal(models.PasskeyLoginFinishRequest{
		CeremonyToken: begin.CeremonyToken,
		Credential:    json.RawMessage(`{"id":"AQ"}`),
	})
	resp = performJSON(t, h.LoginFinish, http.MethodPost, "/api/auth/passkey/login/finish", string(body), "")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("LoginFinish() code = %d: %s", resp.Code, resp.Body.String())
	}
	page, err := st.QueryAuditLogs(models.AuditQuery{Action: models.AuditActionPasskeyLoginFailed})
	if err != nil {
		t.Fatalf("QueryAuditLogs() error = %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("audit entries = %#v, want one rejected passkey sign-in", page.Logs)
	}

	// The same ceremony token cannot be replayed.
	resp = performJSON(t, h.LoginFinish, http.MethodPost, "/api/auth/passkey/login/finish", string(body), "")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("replayed LoginFinish() code = %d: %s", resp.Code, resp.Body.String())
	}
}

func TestPasskeySettingsReportsRelyingParty(t *testing.T) {
	h, admin, st := newTestPasskeyHandler(t)
	if err := st.AddPasskey(models.Passkey{ID: "cred-1", UserID: st.AdminUserID(), Name: "Phone"}); err != nil {
		t.Fatalf("AddPasskey() error = %v", err)
	}
	session := login(t, admin, "admin", "password")

	resp := performJSON(t, sessionHandler(admin, h.Settings), http.MethodGet, "/api/account/passkeys", "", session)
	if resp.Code != http.StatusOK {
		t.Fatalf("Settings() code = %d: %s", resp.Code, resp.Body.String())
	}
	var view models.PasskeySettingsView
	decodeResponse(t, resp, &view)
	if !view.Available || view.RPID != "example.com" {
		t.Fatalf("Settings() = %#v, want the request host as relying party", view)
	}
	if len(view.Passkeys) != 1 || view.Passkeys[0].Name != "Phone" {
		t.Fatalf("passkeys = %#v, want the bound credential", view.Passkeys)
	}

	// API keys have no account to bind credentials to.
	resp = performJSON(t, sessionHandler(admin, h.Settings), http.MethodGet, "/api/account/passkeys", "", "")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("Settings() without session code = %d, want 401", resp.Code)
	}
}

func TestAssetLinksDocument(t *testing.T) {
	_, admin, st := newTestPasskeyHandler(t)
	h := NewPasskeyHandler(st, services.NewPasskeyService(st), admin)

	// Defaults: this repository's app id and the shared debug keystore.
	resp := performJSON(t, h.AssetLinks, http.MethodGet, "/.well-known/assetlinks.json", "", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("AssetLinks() code = %d: %s", resp.Code, resp.Body.String())
	}
	var document []map[string]interface{}
	decodeResponse(t, resp, &document)
	if len(document) != 1 {
		t.Fatalf("AssetLinks() document = %#v, want one entry", document)
	}
	relations, _ := document[0]["relation"].([]interface{})
	if len(relations) != 2 || relations[0] != "delegate_permission/common.handle_all_urls" {
		t.Fatalf("relations = %#v, want handle_all_urls first", relations)
	}
	target, _ := document[0]["target"].(map[string]interface{})
	if target["namespace"] != "android_app" || target["package_name"] != models.DefaultAndroidPackage {
		t.Fatalf("target = %#v, want the default android app", target)
	}
	fingerprints, _ := target["sha256_cert_fingerprints"].([]interface{})
	if len(fingerprints) != 1 || fingerprints[0] != models.DefaultAndroidFingerprint {
		t.Fatalf("fingerprints = %#v, want the shared debug keystore hash", fingerprints)
	}

	// Configured values win, and a pasted hash is normalized.
	settings := st.GetAppSettings()
	settings.PasskeyAndroidPackage = "com.example.custom"
	settings.PasskeyAndroidFingerprints = "fd6cc9aa64ff59aa3eb9b0b6063ea378d89d8da52e90aa3c8902d5e4cf2a4a81 AA:BB:CC"
	if err := st.SetAppSettings(settings); err != nil {
		t.Fatalf("SetAppSettings() error = %v", err)
	}
	resp = performJSON(t, h.AssetLinks, http.MethodGet, "/.well-known/assetlinks.json", "", "")
	decodeResponse(t, resp, &document)
	target, _ = document[0]["target"].(map[string]interface{})
	if target["package_name"] != "com.example.custom" {
		t.Fatalf("package = %v, want the configured one", target["package_name"])
	}
	fingerprints, _ = target["sha256_cert_fingerprints"].([]interface{})
	if len(fingerprints) != 1 || fingerprints[0] != models.DefaultAndroidFingerprint {
		t.Fatalf("fingerprints = %#v, want the normalized hash with the short entry dropped", fingerprints)
	}

	// A configuration without a single valid hash reports the problem instead of
	// serving the debug fingerprint.
	settings.PasskeyAndroidFingerprints = "oops"
	if err := st.SetAppSettings(settings); err != nil {
		t.Fatalf("SetAppSettings() error = %v", err)
	}
	resp = performJSON(t, h.AssetLinks, http.MethodGet, "/.well-known/assetlinks.json", "", "")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("AssetLinks() code = %d, want 404 for an unusable configuration", resp.Code)
	}
}

func TestNormalizeAndroidFingerprints(t *testing.T) {
	if got := models.NormalizeAndroidFingerprints(""); len(got) != 0 {
		t.Fatalf("NormalizeAndroidFingerprints() = %#v, want empty", got)
	}
	got := models.NormalizeAndroidFingerprints("fd6cc9aa64ff59aa3eb9b0b6063ea378d89d8da52e90aa3c8902d5e4cf2a4a81\nAA:BB")
	if len(got) != 1 || got[0] != models.DefaultAndroidFingerprint {
		t.Fatalf("NormalizeAndroidFingerprints() = %#v, want the uppercase colon form only", got)
	}
	settings := models.AppSettings{}
	if settings.AndroidPackage() != models.DefaultAndroidPackage || len(settings.AndroidFingerprints()) != 1 {
		t.Fatalf("blank settings = %q/%#v, want the defaults", settings.AndroidPackage(), settings.AndroidFingerprints())
	}
}

func TestGlobalPasswordSwitchRequiresAdministratorPasskey(t *testing.T) {
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
	h := NewManagementHandler(st, []byte("0123456789abcdef0123456789abcdef"))

	resp := performJSON(t, h.UpdateAppSettings, http.MethodPut, "/api/admin/settings",
		`{"registration_enabled":true,"invite_mode":"off","password_login_disabled":true}`, "")
	if resp.Code != http.StatusBadRequest || !strings.Contains(resp.Body.String(), "管理员") {
		t.Fatalf("UpdateAppSettings() code = %d: %s", resp.Code, resp.Body.String())
	}
	if st.GetAppSettings().PasswordLoginDisabled {
		t.Fatal("panel switch moved without an administrator passkey")
	}

	if err := st.AddPasskey(models.Passkey{ID: "cred-1", UserID: st.AdminUserID(), Name: "Phone"}); err != nil {
		t.Fatalf("AddPasskey() error = %v", err)
	}
	resp = performJSON(t, h.UpdateAppSettings, http.MethodPut, "/api/admin/settings",
		`{"registration_enabled":true,"invite_mode":"off","password_login_disabled":true}`, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("UpdateAppSettings() code = %d: %s", resp.Code, resp.Body.String())
	}
	var view models.AppSettingsView
	decodeResponse(t, resp, &view)
	if !view.PasswordLoginDisabled || !view.PasskeyAdminReady {
		t.Fatalf("settings = %#v, want the switch on with a ready administrator", view)
	}
	page, err := st.QueryAuditLogs(models.AuditQuery{Action: models.AuditActionPasswordLoginGlobalOff})
	if err != nil {
		t.Fatalf("QueryAuditLogs() error = %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("audit entries = %#v, want one panel-wide switch entry", page.Logs)
	}

	// A partial save that omits the field keeps the switch on.
	resp = performJSON(t, h.UpdateAppSettings, http.MethodPut, "/api/admin/settings",
		`{"registration_enabled":true,"invite_mode":"off","turnstile_enabled":false,"turnstile_site_key":""}`, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("UpdateAppSettings() code = %d: %s", resp.Code, resp.Body.String())
	}
	if !st.GetAppSettings().PasswordLoginDisabled {
		t.Fatal("partial save reset the panel password switch")
	}
}
