package handlers

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

const oidcFlowTTL = 10 * time.Minute
const oidcCookieName = "tm_oidc_browser"

type oidcFlow struct {
	provider          models.OIDCProvider
	client            *services.OIDCClient
	verifier          string
	nonce             string
	browserHash       string
	ownerSession      string
	ownerID           string
	ownerPasswordHash string
	action            string
	expires           time.Time
}

type oidcPending struct {
	flow         oidcFlow
	claims       services.OIDCClaims
	mailVerified bool
	attempts     int
	busy         bool
}

// OIDCHandler keeps bounded, short-lived, single-use handoffs. No session or
// provider token is placed in a redirect URL. Handoffs are bound to the browser.
type OIDCHandler struct {
	store   *store.Store
	admin   *AdminHandler
	auth    *AuthHandler
	key     []byte
	mu      sync.Mutex
	flows   map[string]oidcFlow
	pending map[string]*oidcPending
	now     func() time.Time
}

func NewOIDCHandler(st *store.Store, admin *AdminHandler, registration *AuthHandler, key []byte) *OIDCHandler {
	return &OIDCHandler{store: st, admin: admin, auth: registration, key: append([]byte(nil), key...), flows: map[string]oidcFlow{}, pending: map[string]*oidcPending{}, now: time.Now}
}

func oidcSecretPurpose(id string) string { return "oidc-client:" + id }

func (h *OIDCHandler) providerView(provider models.OIDCProvider) models.OIDCProviderView {
	return models.OIDCProviderView{ID: provider.ID, Name: provider.Name, Issuer: provider.Issuer, ClientID: provider.ClientID,
		RedirectURI: provider.RedirectURI, Scopes: provider.Scopes, Enabled: provider.Enabled, Order: provider.Order, HasClientSecret: provider.ClientSecret != ""}
}

func (h *OIDCHandler) Providers(w http.ResponseWriter, r *http.Request) {
	providers := []models.OIDCPublicProvider{}
	for _, provider := range h.store.ListOIDCProviders() {
		if provider.Enabled {
			providers = append(providers, models.OIDCPublicProvider{ID: provider.ID, Name: provider.Name})
		}
	}
	writeJSON(w, http.StatusOK, providers)
}

func (h *OIDCHandler) AdminProviders(w http.ResponseWriter, r *http.Request) {
	providers := []models.OIDCProviderView{}
	for _, provider := range h.store.ListOIDCProviders() {
		providers = append(providers, h.providerView(provider))
	}
	writeJSON(w, http.StatusOK, providers)
}

func (h *OIDCHandler) SaveProvider(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		Issuer       string `json:"issuer"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RedirectURI  string `json:"redirect_uri"`
		Scopes       string `json:"scopes"`
		Enabled      bool   `json:"enabled"`
		Order        int    `json:"order"`
		ClearSecret  bool   `json:"clear_secret"`
	}
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "配置格式无效"})
		return
	}
	provider := models.OIDCProvider{ID: chi.URLParam(r, "id"), Name: strings.TrimSpace(req.Name), Issuer: strings.TrimSpace(req.Issuer),
		ClientID: strings.TrimSpace(req.ClientID), RedirectURI: strings.TrimSpace(req.RedirectURI), Scopes: strings.Join(strings.Fields(req.Scopes), " "), Enabled: req.Enabled, Order: req.Order}
	if provider.Name == "" || len(provider.Name) > 80 || provider.ClientID == "" || len(provider.ClientID) > 512 || len(req.ClientSecret) > 2048 || len(provider.Scopes) > 512 {
		writeJSON(w, 400, map[string]string{"error": "请填写名称和 Client ID，并检查字段长度"})
		return
	}
	if provider.Scopes == "" {
		provider.Scopes = "openid profile email"
	}
	if !containsString(strings.Fields(provider.Scopes), "openid") {
		writeJSON(w, 400, map[string]string{"error": "Scopes 必须包含 openid"})
		return
	}
	for _, value := range []string{provider.Issuer, provider.RedirectURI} {
		if len(value) > 2048 {
			writeJSON(w, 400, map[string]string{"error": "URL 过长"})
			return
		}
		if err := services.ValidateOIDCURL(value); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
	}
	if provider.ID != "" {
		existing, ok := h.store.GetOIDCProvider(provider.ID)
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "接入方不存在"})
			return
		}
		provider.ClientSecret = existing.ClientSecret
		if provider.ClientSecret != "" && !req.ClearSecret && req.ClientSecret == "" && (existing.Issuer != provider.Issuer || existing.ClientID != provider.ClientID) {
			plain, err := auth.DecryptSecret(h.key, oidcSecretPurpose(existing.Issuer+"|"+existing.ClientID), existing.ClientSecret)
			if err != nil {
				writeJSON(w, 503, map[string]string{"error": "原客户端密钥不可用，请重新填写密钥"})
				return
			}
			req.ClientSecret = string(plain)
		}
	}
	if req.ClearSecret {
		provider.ClientSecret = ""
	}
	if req.ClientSecret != "" {
		// The encryption purpose is tied to a stable client configuration rather
		// than a database id, which is allocated during the atomic save below.
		encrypted, err := auth.EncryptSecret(h.key, oidcSecretPurpose(provider.Issuer+"|"+provider.ClientID), []byte(req.ClientSecret))
		if err != nil {
			writeJSON(w, 503, map[string]string{"error": "客户端密钥加密失败"})
			return
		}
		provider.ClientSecret = encrypted
	}
	saved, err := h.store.SaveOIDCProvider(provider)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	h.audit(r, "oidc_provider_update", saved.Name)
	writeJSON(w, http.StatusOK, h.providerView(saved))
}

func (h *OIDCHandler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeleteOIDCProvider(id); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	h.audit(r, "oidc_provider_delete", id)
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (h *OIDCHandler) audit(r *http.Request, action, target string) {
	user := SessionUser(r)
	if user != nil {
		recordAuditEntry(h.store, models.AuditLog{Category: models.AuditCategorySettings, Action: action, Target: target, ActorID: user.ID, ActorName: user.Username, IP: clientIP(r), Success: true})
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (h *OIDCHandler) Start(w http.ResponseWriter, r *http.Request)        { h.start(w, r, false) }
func (h *OIDCHandler) AccountStart(w http.ResponseWriter, r *http.Request) { h.start(w, r, true) }

func (h *OIDCHandler) start(w http.ResponseWriter, r *http.Request, account bool) {
	var req struct {
		ProviderID        string `json:"provider_id"`
		Action            string `json:"action"`
		Password          string `json:"password"`
		TurnstileResponse string `json:"cf_turnstile_response"`
	}
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "请求无效"})
		return
	}
	if !h.auth.guardAuth(w, r, "oidc-start") {
		return
	}
	h.auth.chargeAuth(r, "oidc-start")
	flow := oidcFlow{action: "login", expires: h.now().Add(oidcFlowTTL)}
	if account {
		user := SessionUser(r)
		if user == nil {
			writeJSON(w, 401, map[string]string{"error": "请先登录"})
			return
		}
		if req.Action != "bind" && req.Action != "password" {
			writeJSON(w, 400, map[string]string{"error": "操作无效"})
			return
		}
		flow.action = req.Action
		flow.ownerID = user.ID
		flow.ownerSession = r.Header.Get("X-Auth-Token")
		if _, _, ok := h.admin.userIDFromToken(flow.ownerSession); !ok {
			writeJSON(w, 401, map[string]string{"error": "会话已失效"})
			return
		}
		if flow.action == "bind" {
			account, exists := h.store.GetUserByID(user.ID)
			if !exists || account.PasswordHash == "" || req.Password == "" || len(req.Password) > maxPasswordLength {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "请确认本站密码；尚无密码时先通过已绑定身份重新认证并设置密码"})
				return
			}
			if !h.admin.acquirePasswordVerify() {
				writeJSON(w, 503, map[string]string{"error": "密码验证繁忙，请稍后重试"})
				return
			}
			proof, err := h.store.ConfirmOIDCBindingPassword(user.ID, req.Password)
			h.admin.releasePasswordVerify()
			if err != nil {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "本站密码不正确"})
				return
			}
			flow.ownerPasswordHash = proof
		}
	} else {
		if ok, msg := verifyTurnstile(h.store, h.key, r, req.TurnstileResponse, "login"); !ok {
			writeJSON(w, 400, map[string]string{"error": msg})
			return
		}
	}
	provider, ok := h.store.GetOIDCProvider(req.ProviderID)
	if !ok || !provider.Enabled {
		writeJSON(w, 400, map[string]string{"error": store.ErrOIDCProviderNotFound.Error()})
		return
	}
	secret := ""
	if provider.ClientSecret != "" {
		plain, err := auth.DecryptSecret(h.key, oidcSecretPurpose(provider.Issuer+"|"+provider.ClientID), provider.ClientSecret)
		if err != nil {
			writeJSON(w, 503, map[string]string{"error": "OIDC 密钥不可用，请联系管理员"})
			return
		}
		secret = string(plain)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	client, err := services.NewOIDCClient(ctx, provider, secret)
	if err != nil {
		writeJSON(w, 503, map[string]string{"error": err.Error()})
		return
	}
	state, err := randomURLToken(32)
	if err != nil {
		writeJSON(w, 503, map[string]string{"error": "生成登录请求失败"})
		return
	}
	nonce, err := randomURLToken(32)
	if err != nil {
		writeJSON(w, 503, map[string]string{"error": "生成登录请求失败"})
		return
	}
	verifier, err := randomURLToken(48)
	if err != nil {
		writeJSON(w, 503, map[string]string{"error": "生成登录请求失败"})
		return
	}
	browser, err := randomURLToken(32)
	if err != nil {
		writeJSON(w, 503, map[string]string{"error": "生成登录请求失败"})
		return
	}
	flow.provider = provider
	flow.client = client
	flow.nonce = nonce
	flow.verifier = verifier
	flow.browserHash = hashToken(browser)
	h.mu.Lock()
	h.pruneLocked()
	if len(h.flows)+len(h.pending) >= maxChallenges {
		h.mu.Unlock()
		writeJSON(w, 503, map[string]string{"error": "登录请求过多，请稍后重试"})
		return
	}
	h.flows[state] = flow
	h.mu.Unlock()
	callback, _ := url.Parse(provider.RedirectURI)
	http.SetCookie(w, &http.Cookie{Name: oidcCookieName, Value: browser, Path: "/api/", HttpOnly: true, Secure: callback.Scheme == "https", SameSite: http.SameSiteLaxMode, MaxAge: int(oidcFlowTTL.Seconds())})
	writeJSON(w, 200, map[string]string{"authorization_url": client.AuthorizationURL(state, nonce, verifier, account)})
}

func (h *OIDCHandler) pruneLocked() {
	for state, flow := range h.flows {
		if !h.now().Before(flow.expires) {
			delete(h.flows, state)
		}
	}
	for ticket, pending := range h.pending {
		if !pending.busy && !h.now().Before(pending.flow.expires) {
			delete(h.pending, ticket)
		}
	}
}

func browserMatches(r *http.Request, expected string) bool {
	cookie, err := r.Cookie(oidcCookieName)
	return err == nil && subtle.ConstantTimeCompare([]byte(hashToken(cookie.Value)), []byte(expected)) == 1
}

func (h *OIDCHandler) Callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	state := r.URL.Query().Get("state")
	h.mu.Lock()
	h.pruneLocked()
	flow, ok := h.flows[state]
	if ok && browserMatches(r, flow.browserHash) {
		delete(h.flows, state)
	} else {
		ok = false
	}
	h.mu.Unlock()
	if !ok {
		h.redirectError(w, r, "login", "OIDC 登录请求无效或已过期，请重新登录")
		return
	}
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		h.redirectError(w, r, flow.action, "OIDC 授权未完成，请重试")
		return
	}
	if !h.flowValid(flow) {
		h.redirectError(w, r, flow.action, "接入方配置或本站会话已变化，请重新开始")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	claims, err := flow.client.Exchange(ctx, r.URL.Query().Get("code"), flow.nonce, flow.verifier)
	if err != nil {
		h.redirectError(w, r, flow.action, err.Error())
		return
	}
	if flow.action == "password" && (claims.AuthTime < flow.expires.Add(-oidcFlowTTL-time.Minute).Unix() || claims.AuthTime > h.now().Add(time.Minute).Unix()) {
		h.redirectError(w, r, flow.action, "接入方未确认本次重新认证，请重新登录后重试")
		return
	}
	if flow.action == "login" && !validEmail(claims.Email) {
		h.redirectError(w, r, flow.action, "接入方未提供有效邮箱，请联系管理员检查 email 权限")
		return
	}
	ticket, err := randomURLToken(32)
	if err != nil {
		h.redirectError(w, r, flow.action, "生成登录结果失败")
		return
	}
	flow.client = nil
	h.mu.Lock()
	h.pruneLocked()
	if len(h.pending)+len(h.flows) >= maxChallenges {
		h.mu.Unlock()
		h.redirectError(w, r, flow.action, "登录请求过多，请稍后重试")
		return
	}
	h.pending[ticket] = &oidcPending{flow: flow, claims: claims}
	h.mu.Unlock()
	path := "/login"
	if flow.action != "login" {
		path = "/account"
	}
	http.Redirect(w, r, path+"?oidc_ticket="+url.QueryEscape(ticket), http.StatusSeeOther)
}

func (h *OIDCHandler) redirectError(w http.ResponseWriter, r *http.Request, action, message string) {
	path := "/login"
	if action != "login" {
		path = "/account"
	}
	http.Redirect(w, r, path+"?oidc_error="+url.QueryEscape(message), http.StatusSeeOther)
}

func (h *OIDCHandler) flowValid(flow oidcFlow) bool {
	provider, ok := h.store.GetOIDCProvider(flow.provider.ID)
	if !ok || !provider.Enabled || provider.Revision != flow.provider.Revision {
		return false
	}
	if flow.ownerID != "" {
		id, _, ok := h.admin.userIDFromToken(flow.ownerSession)
		if !ok || id != flow.ownerID {
			return false
		}
		if flow.action == "bind" {
			user, exists := h.store.GetUserByID(id)
			return exists && flow.ownerPasswordHash != "" && user.PasswordHash == flow.ownerPasswordHash
		}
	}
	return true
}

func (h *OIDCHandler) claimPending(r *http.Request, ticket string) (*oidcPending, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pruneLocked()
	pending, ok := h.pending[ticket]
	if !ok || pending.busy || !h.now().Before(pending.flow.expires) || !browserMatches(r, pending.flow.browserHash) {
		return nil, false
	}
	pending.busy = true
	return pending, true
}

func (h *OIDCHandler) releasePending(ticket string, pending *oidcPending, consume bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if consume || pending.attempts >= maxFactorAttempts {
		delete(h.pending, ticket)
	} else {
		pending.busy = false
	}
}

func (h *OIDCHandler) needsEmail(pending *oidcPending) bool {
	_, exists, linked := h.store.OIDCAccount(pending.flow.provider.ID, pending.claims.Issuer, pending.claims.Subject, pending.claims.Email)
	if linked || pending.mailVerified {
		return false
	}
	if exists {
		return !pending.claims.EmailVerified
	}
	return h.auth.mailer() != nil && !h.store.GetAppSettings().EmailVerifyDisabled
}

func (h *OIDCHandler) Complete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var req struct {
		Ticket string `json:"ticket"`
		Code   string `json:"code"`
	}
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "请求无效"})
		return
	}
	pending, ok := h.claimPending(r, req.Ticket)
	if !ok {
		writeJSON(w, 400, map[string]string{"error": "登录结果无效或已过期，请重新登录"})
		return
	}
	consume := false
	defer func() { h.releasePending(req.Ticket, pending, consume) }()
	if !h.flowValid(pending.flow) {
		consume = true
		writeJSON(w, 400, map[string]string{"error": "接入方配置或本站会话已变化，请重新开始"})
		return
	}
	// This endpoint also handles public login handoffs, so the router cannot
	// require a session for every request. Account handoffs must restore their
	// authenticated context from the original session before binding or setup.
	if pending.flow.action != "login" {
		token := r.Header.Get("X-Auth-Token")
		user, valid := h.admin.ValidateSession(token)
		if !valid || token != pending.flow.ownerSession || user.ID != pending.flow.ownerID {
			writeJSON(w, 401, map[string]string{"error": "请使用发起认证时的本站会话重试"})
			return
		}
		r = withUser(r, user)
	}
	if pending.flow.action == "password" {
		user := SessionUser(r)
		if user == nil || user.ID != pending.flow.ownerID {
			writeJSON(w, 401, map[string]string{"error": "请重新登录"})
			return
		}
		writeJSON(w, 200, map[string]any{"password_setup_required": true, "ticket": req.Ticket})
		return
	}
	if pending.flow.action == "login" {
		if !h.auth.guardAuth(w, r, pending.claims.Email) {
			return
		}
		if h.needsEmail(pending) {
			if strings.TrimSpace(req.Code) == "" {
				writeJSON(w, 202, map[string]any{"email_verification_required": true, "email": pending.claims.Email, "ticket": req.Ticket})
				return
			}
			h.auth.chargeAuth(r, pending.claims.Email)
			pending.attempts++
			sum := sha256.Sum256([]byte(strings.TrimSpace(req.Code)))
			if !h.store.ConsumeVerifyCode(pending.claims.Email, "oidc:"+req.Ticket, hex.EncodeToString(sum[:])) {
				writeJSON(w, 400, map[string]string{"error": "验证码错误或已过期"})
				return
			}
			pending.mailVerified = true
		}
	} else {
		user := SessionUser(r)
		if user == nil || user.ID != pending.flow.ownerID {
			writeJSON(w, 401, map[string]string{"error": "请重新登录"})
			return
		}
	}
	identity := models.OIDCIdentity{ProviderID: pending.flow.provider.ID, Issuer: pending.claims.Issuer, Subject: pending.claims.Subject, Email: pending.claims.Email}
	requireNewEmail := h.auth.mailer() != nil && !h.store.GetAppSettings().EmailVerifyDisabled
	_, exists, _ := h.store.OIDCAccount(identity.ProviderID, identity.Issuer, identity.Subject, identity.Email)
	emailProven := pending.mailVerified || (pending.claims.EmailVerified && (exists || !requireNewEmail))
	user, err := h.store.ResolveOIDCAccount(identity, pending.flow.provider.Revision, emailProven, requireNewEmail, pending.flow.ownerID, pending.flow.ownerPasswordHash)
	if errors.Is(err, store.ErrOIDCEmailProofRequired) {
		writeJSON(w, 202, map[string]any{"email_verification_required": true, "email": pending.claims.Email, "ticket": req.Ticket})
		return
	}
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	consume = true
	if pending.flow.action == "bind" {
		h.audit(r, "oidc_bind", pending.flow.provider.Name)
		writeJSON(w, 200, map[string]string{"status": "ok"})
		return
	}
	h.admin.beginOIDCLogin(w, r, user)
}

func (h *OIDCHandler) SendCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Ticket string `json:"ticket"`
	}
	if err := readAdminJSON(w, r, &req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "请求无效"})
		return
	}
	pending, ok := h.claimPending(r, req.Ticket)
	if !ok {
		writeJSON(w, 400, map[string]string{"error": "登录结果无效或已过期"})
		return
	}
	defer h.releasePending(req.Ticket, pending, false)
	if pending.flow.action != "login" || !h.flowValid(pending.flow) || !h.needsEmail(pending) {
		writeJSON(w, 400, map[string]string{"error": "此登录请求不需要邮箱验证"})
		return
	}
	if !h.auth.guardAuth(w, r, pending.claims.Email) {
		return
	}
	h.auth.chargeAuth(r, pending.claims.Email)
	mailer := h.auth.mailer()
	if mailer == nil {
		writeJSON(w, 400, map[string]string{"error": "接入方未确认邮箱归属，且本站未配置邮件服务，请联系管理员"})
		return
	}
	// Rate-limit by email across handoffs, while each code is usable only by
	// the precise browser-bound OIDC login that requested it.
	if h.store.LastCodeSentWithin(pending.claims.Email, "oidc-mail", verifyCodeResendWait) {
		writeJSON(w, 429, map[string]string{"error": "发送过于频繁，请稍后重试"})
		return
	}
	code, err := generateNumericCode()
	if err != nil {
		writeJSON(w, 503, map[string]string{"error": "生成验证码失败"})
		return
	}
	sum := sha256.Sum256([]byte(code))
	h.store.PutVerifyCode(pending.claims.Email, "oidc:"+req.Ticket, hex.EncodeToString(sum[:]), verifyCodeTTL)
	h.store.PutVerifyCode(pending.claims.Email, "oidc-mail", "", verifyCodeTTL)
	plain, html := services.VerificationCodeEmail("OIDC 登录", code, int(verifyCodeTTL/time.Minute))
	if err := mailer.Send(pending.claims.Email, "Tunnel Manager 邮箱验证码", plain, html); err != nil {
		writeJSON(w, 503, map[string]string{"error": "邮件发送失败，请稍后重试"})
		return
	}
	writeJSON(w, 200, map[string]string{"message": "验证码已发送"})
}

func (h *AdminHandler) beginOIDCLogin(w http.ResponseWriter, r *http.Request, user models.User) {
	if user.TOTPEnabled {
		if _, err := auth.DecryptTOTPSecret(h.encryptionKey, user.TOTPSecretEncrypted); err != nil {
			writeJSON(w, 503, map[string]string{"error": "二步验证暂不可用"})
			return
		}
		token, expires, err := h.newChallenge(user.ID, h.currentAuthEpoch())
		if err != nil {
			writeJSON(w, 503, map[string]string{"error": "登录暂不可用"})
			return
		}
		count, _ := h.store.CountPasskeys(user.ID)
		writeJSON(w, 202, models.TwoFactorChallengeResponse{TwoFactorRequired: true, ChallengeToken: token, ExpiresAt: expires, PasskeysAvailable: count > 0})
		return
	}
	h.completeLogin(w, r, user, "oidc_login")
}

func (h *OIDCHandler) Account(w http.ResponseWriter, r *http.Request) {
	user, ok := h.store.GetUserByID(sessionUID(r))
	if !ok {
		writeJSON(w, 401, map[string]string{"error": "请重新登录"})
		return
	}
	identities := h.store.ListOIDCIdentities(user.ID)
	bindings := []map[string]any{}
	for _, identity := range identities {
		provider, ok := h.store.GetOIDCProvider(identity.ProviderID)
		if ok {
			bindings = append(bindings, map[string]any{"provider_id": provider.ID, "name": provider.Name, "enabled": provider.Enabled, "email": identity.Email, "created_at": identity.CreatedAt})
		}
	}
	writeJSON(w, 200, map[string]any{"has_password": user.PasswordHash != "", "bindings": bindings})
}

func (h *OIDCHandler) Unlink(w http.ResponseWriter, r *http.Request) {
	if err := h.store.UnlinkOIDC(sessionUID(r), chi.URLParam(r, "id")); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	h.audit(r, "oidc_unbind", chi.URLParam(r, "id"))
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// SetPassword enables existing account settings for accounts provisioned
// without a password. A fresh OIDC login must prove the exact bound identity;
// matching only the email is insufficient for this sensitive operation.
func (h *OIDCHandler) SetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Ticket   string `json:"ticket"`
		Password string `json:"password"`
	}
	if err := readAdminJSON(w, r, &req); err != nil || len(req.Password) < 6 || len(req.Password) > maxPasswordLength {
		writeJSON(w, 400, map[string]string{"error": "密码长度需在 6-1024 位之间"})
		return
	}
	pending, ok := h.claimPending(r, req.Ticket)
	if !ok {
		writeJSON(w, 400, map[string]string{"error": "重新认证已过期，请重试"})
		return
	}
	consume := false
	defer func() { h.releasePending(req.Ticket, pending, consume) }()
	userID := sessionUID(r)
	if pending.flow.action != "password" || pending.flow.ownerID != userID || r.Header.Get("X-Auth-Token") != pending.flow.ownerSession || !h.flowValid(pending.flow) {
		writeJSON(w, 400, map[string]string{"error": "重新认证无效"})
		return
	}
	user, exists, linked := h.store.OIDCAccount(pending.flow.provider.ID, pending.claims.Issuer, pending.claims.Subject, "")
	if !exists || !linked || user.ID != userID || user.PasswordHash != "" || user.Status != models.UserActive {
		writeJSON(w, 400, map[string]string{"error": "需要使用当前账号已关联的 OIDC 身份，已有密码请使用修改密码"})
		return
	}
	if !h.admin.acquirePasswordVerify() {
		writeJSON(w, 503, map[string]string{"error": "密码保存繁忙，请稍后重试"})
		return
	}
	defer h.admin.releasePasswordVerify()
	identity := models.OIDCIdentity{ProviderID: pending.flow.provider.ID, Issuer: pending.claims.Issuer, Subject: pending.claims.Subject, UserID: userID}
	if err := h.store.SetOIDCPassword(identity, pending.flow.provider.Revision, store.HashPassword(req.Password)); err != nil {
		writeJSON(w, 503, map[string]string{"error": "密码保存失败"})
		return
	}
	consume = true
	h.audit(r, "oidc_password_setup", user.Username)
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
