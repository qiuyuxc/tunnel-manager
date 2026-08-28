package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

const cloudflareOAuthFlowTTL = 10 * time.Minute

type cloudflareOAuthFlow struct {
	ownerToken  string
	userID      string
	redirectURI string
	verifier    string
	expiresAt   time.Time
}

// CloudflareOAuthHandler manages per-user Cloudflare OAuth connections.
type CloudflareOAuthHandler struct {
	store *store.Store
	oauth *services.CloudflareOAuth
	cf    *services.CloudflareClient
	admin *AdminHandler
	now   func() time.Time
	mu    sync.Mutex
	flows map[string]cloudflareOAuthFlow
}

// NewCloudflareOAuthHandler creates a Cloudflare OAuth handler.
func NewCloudflareOAuthHandler(st *store.Store, oauth *services.CloudflareOAuth, cf *services.CloudflareClient, admin *AdminHandler) *CloudflareOAuthHandler {
	return &CloudflareOAuthHandler{
		store: st,
		oauth: oauth,
		cf:    cf,
		admin: admin,
		now:   time.Now,
		flows: make(map[string]cloudflareOAuthFlow),
	}
}

// Status returns OAuth readiness, the account connections owned by the
// requesting user and the accounts available through the active connection.
func (h *CloudflareOAuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	uid := sessionUID(r)
	redirectURI := h.oauth.RedirectURI()
	if redirectURI == "" {
		redirectURI = cloudflareOAuthRedirectURI(r)
	}
	connections := h.store.ListCFConnectionViews(uid)
	activeConn, hasActive := h.store.ActiveCFConnection(uid)
	isAdmin := uid != "" && uid == h.store.AdminUserID()
	status := models.CloudflareOAuthStatus{
		Configured:         h.oauth.Configured(),
		Connected:          (hasActive && activeConn.HasToken()) || (isAdmin && h.cf.HasStaticCredentials()),
		Source:             "none",
		Accounts:           []models.Account{},
		RedirectURI:        redirectURI,
		Connections:        connections,
		ActiveConnectionID: activeConn.ID,
	}
	if hasActive && activeConn.HasToken() {
		status.Source = "oauth"
		status.AccountID = activeConn.AccountID
		status.AccountName = activeConn.AccountName
		if activeConn.ExpiresAt > 0 {
			status.ExpiresAt = time.Unix(activeConn.ExpiresAt, 0).Format(time.RFC3339)
		}
		if client := UserCF(r); client != nil {
			accounts, err := client.ListAccounts()
			if err != nil {
				status.Error = err.Error()
			} else {
				status.Accounts = accounts
			}
			// ListAccounts may have rotated the token; re-read the connection.
			if fresh, ok := h.store.ActiveCFConnection(uid); ok {
				activeConn = fresh
				status.AccountID = fresh.AccountID
				status.AccountName = fresh.AccountName
				status.ExpiresAt = ""
				if fresh.ExpiresAt > 0 {
					status.ExpiresAt = time.Unix(fresh.ExpiresAt, 0).Format(time.RFC3339)
				}
			}
		}
	} else if isAdmin && h.cf.HasStaticCredentials() {
		status.Source = "api_token"
		status.AccountID = h.cf.DefaultAccountID()
	} else if !status.Configured {
		status.Error = h.oauth.ConfigurationError()
	} else {
		status.Error = "尚未授权 Cloudflare 账户，请先完成授权"
	}
	writeJSON(w, http.StatusOK, status)
}

// Start creates a single-use state and returns Cloudflare's authorization URL.
func (h *CloudflareOAuthHandler) Start(w http.ResponseWriter, r *http.Request) {
	redirectURI := h.oauth.RedirectURI()
	if redirectURI == "" {
		redirectURI = cloudflareOAuthRedirectURI(r)
	}
	state, err := randomURLToken(32)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to start OAuth flow"})
		return
	}
	verifier, err := randomURLToken(48)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to start OAuth flow"})
		return
	}
	challengeBytes := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeBytes[:])
	authorizationURL, err := h.oauth.AuthorizationURL(redirectURI, state, challenge)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	ownerToken := r.Header.Get("X-Auth-Token")
	h.mu.Lock()
	h.pruneFlowsLocked()
	for existingState, flow := range h.flows {
		if flow.ownerToken == ownerToken {
			delete(h.flows, existingState)
		}
	}
	h.flows[state] = cloudflareOAuthFlow{
		ownerToken:  ownerToken,
		userID:      sessionUID(r),
		redirectURI: redirectURI,
		verifier:    verifier,
		expiresAt:   h.now().Add(cloudflareOAuthFlowTTL),
	}
	h.mu.Unlock()
	writeJSON(w, http.StatusOK, models.CloudflareOAuthStartResponse{AuthorizationURL: authorizationURL})
}

// Callback validates state, exchanges the code into a NEW connection owned
// by the initiating user and selects an available account.
func (h *CloudflareOAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	flow, ok := h.claimFlow(state)
	if !ok {
		h.redirectResult(w, r, "error", "OAuth state 无效或已过期")
		return
	}
	if !h.admin.ValidateToken(flow.ownerToken) {
		h.redirectResult(w, r, "error", "登录会话已失效，请重新登录后再授权")
		return
	}
	if oauthError := r.URL.Query().Get("error"); oauthError != "" {
		description := r.URL.Query().Get("error_description")
		if description == "" {
			description = oauthError
		}
		h.redirectResult(w, r, "error", description)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		h.redirectResult(w, r, "error", "Cloudflare 未返回授权码")
		return
	}
	conn, err := h.oauth.ExchangeCodeForUser(flow.userID, code, flow.redirectURI, flow.verifier)
	if err != nil {
		h.redirectResult(w, r, "error", err.Error())
		return
	}
	client := h.cf.WithConnection(conn)
	accounts, err := client.ListAccounts()
	if err != nil {
		h.redirectResult(w, r, "error", "授权成功，但读取账户失败: "+err.Error())
		return
	}
	if len(accounts) == 0 {
		h.redirectResult(w, r, "error", "授权成功，但没有可用的 Cloudflare 账户")
		return
	}
	preferred := ""
	if previous, ok := h.store.ActiveCFConnection(flow.userID); ok {
		preferred = previous.AccountID
	}
	selected := selectCloudflareAccount(accounts, preferred, h.cf.DefaultAccountID())
	if err := h.store.UpdateCFConnectionAccount(conn.ID, selected.ID, selected.Name); err != nil {
		h.redirectResult(w, r, "error", "保存 Cloudflare 账户失败")
		return
	}
	if err := h.store.SetActiveCFConnection(flow.userID, conn.ID); err != nil {
		h.redirectResult(w, r, "error", "保存连接选择失败")
		return
	}
	h.redirectResult(w, r, "success", "")
}

// SelectAccount changes the Cloudflare account used by the active connection.
func (h *CloudflareOAuthHandler) SelectAccount(w http.ResponseWriter, r *http.Request) {
	var request models.CloudflareAccountRequest
	if err := readJSON(r, &request); err != nil || strings.TrimSpace(request.AccountID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "account_id is required"})
		return
	}
	uid := sessionUID(r)
	conn, hasConn := h.store.ActiveCFConnection(uid)
	if !hasConn || !conn.HasToken() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "尚未授权 Cloudflare 账户"})
		return
	}
	client := h.cf.ForUser(uid)
	accounts, err := client.ListAccounts()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	for _, account := range accounts {
		if account.ID == request.AccountID {
			if err := h.store.UpdateCFConnectionAccount(conn.ID, account.ID, account.Name); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save account selection"})
				return
			}
			writeJSON(w, http.StatusOK, account)
			return
		}
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": "account is not available to this authorization"})
}

// ActivateConnection switches the Cloudflare connection the account operates on.
func (h *CloudflareOAuthHandler) ActivateConnection(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ConnectionID string `json:"connection_id"`
	}
	if err := readJSON(r, &request); err != nil || strings.TrimSpace(request.ConnectionID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "connection_id is required"})
		return
	}
	uid := sessionUID(r)
	if err := h.store.SetActiveCFConnection(uid, request.ConnectionID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "connection not found"})
		return
	}
	conn, _ := h.store.ActiveCFConnection(uid)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "account_id": conn.AccountID, "account_name": conn.AccountName})
}

// Disconnect revokes and deletes one of the user's connections. Without an
// explicit connection_id the active connection is disconnected.
func (h *CloudflareOAuthHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	uid := sessionUID(r)
	connID := r.URL.Query().Get("connection_id")
	target, ok := h.store.ActiveCFConnection(uid)
	if connID != "" {
		target, ok = h.store.ActiveCFConnection(uid)
		for _, conn := range h.store.ListCFConnections(uid) {
			if conn.ID == connID {
				target, ok = conn, true
			}
		}
	} else if !ok {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "connection not found"})
		return
	}
	err := h.oauth.RevokeConnection(target)
	response := map[string]string{"status": "ok"}
	if err != nil {
		response["warning"] = "本地连接已删除，但 Cloudflare 撤销请求失败: " + err.Error()
	}
	writeJSON(w, http.StatusOK, response)
}

// claimFlow atomically consumes a pending OAuth flow by state.
func (h *CloudflareOAuthHandler) claimFlow(state string) (cloudflareOAuthFlow, bool) {
	if state == "" {
		return cloudflareOAuthFlow{}, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pruneFlowsLocked()
	flow, ok := h.flows[state]
	if ok {
		delete(h.flows, state)
	}
	return flow, ok
}

func (h *CloudflareOAuthHandler) pruneFlowsLocked() {
	now := h.now()
	for state, flow := range h.flows {
		if !now.Before(flow.expiresAt) {
			delete(h.flows, state)
		}
	}
}

func (h *CloudflareOAuthHandler) redirectResult(w http.ResponseWriter, r *http.Request, result, message string) {
	query := url.Values{"cloudflare_oauth": {result}}
	if message != "" {
		query.Set("message", message)
	}
	http.Redirect(w, r, "/account?"+query.Encode(), http.StatusFound)
}

func cloudflareOAuthRedirectURI(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	if forwarded := firstForwardedValue(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = forwarded
	}
	host := r.Host
	if forwarded := firstForwardedValue(r.Header.Get("X-Forwarded-Host")); forwarded != "" {
		host = forwarded
	}
	return scheme + "://" + host + "/api/cloudflare/oauth/callback"
}

func firstForwardedValue(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ",")
	return strings.TrimSpace(parts[0])
}

func randomURLToken(size int) (string, error) {
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	// RawURLEncoding of `size` bytes is URL-safe and long enough; no slicing
	// (a +1 slice bound here panicked for 48-byte verifiers).
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func selectCloudflareAccount(accounts []models.Account, preferredIDs ...string) models.Account {
	for _, preferred := range preferredIDs {
		if preferred == "" {
			continue
		}
		for _, account := range accounts {
			if account.ID == preferred {
				return account
			}
		}
	}
	if len(accounts) > 0 {
		return accounts[0]
	}
	return models.Account{}
}
