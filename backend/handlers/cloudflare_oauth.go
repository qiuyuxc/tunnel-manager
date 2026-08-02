package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
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
	redirectURI string
	verifier    string
	expiresAt   time.Time
}

// CloudflareOAuthHandler exposes the administrator OAuth connection flow.
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

// Status returns OAuth readiness, credential source and available accounts.
func (h *CloudflareOAuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	redirectURI := h.oauth.RedirectURI()
	if redirectURI == "" {
		redirectURI = cloudflareOAuthRedirectURI(r)
	}
	status := models.CloudflareOAuthStatus{
		Configured:  h.oauth.Configured(),
		Connected:   h.oauth.Connected() || h.cf.HasStaticCredentials(),
		Source:      "none",
		Accounts:    []models.Account{},
		RedirectURI: redirectURI,
	}
	if h.oauth.Connected() {
		status.Source = "oauth"
		accounts, err := h.cf.ListAccounts()
		if err != nil {
			status.Error = err.Error()
		} else {
			status.Accounts = accounts
		}
	} else if h.cf.HasStaticCredentials() {
		status.Source = "api_token"
	} else if !status.Configured {
		status.Error = h.oauth.ConfigurationError()
	}
	config := h.store.GetConfig()
	status.AccountID = config.CFAccountID
	status.AccountName = config.CFAccountName
	if status.Source == "api_token" && status.AccountID == "" {
		status.AccountID = h.cf.DefaultAccountID()
	}
	if config.CFOAuthExpiresAt > 0 {
		status.ExpiresAt = time.Unix(config.CFOAuthExpiresAt, 0).Format(time.RFC3339)
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
		redirectURI: redirectURI,
		verifier:    verifier,
		expiresAt:   h.now().Add(cloudflareOAuthFlowTTL),
	}
	h.mu.Unlock()
	writeJSON(w, http.StatusOK, models.CloudflareOAuthStartResponse{AuthorizationURL: authorizationURL})
}

// Callback validates state, exchanges the code and selects an available account.
func (h *CloudflareOAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	flow, ok := h.claimFlow(state)
	if !ok {
		h.redirectResult(w, r, "error", "OAuth state 无效或已过期")
		return
	}
	if !h.admin.ValidateToken(flow.ownerToken) {
		h.redirectResult(w, r, "error", "管理员会话已失效，请重新登录")
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
	if err := h.oauth.ExchangeCode(code, flow.redirectURI, flow.verifier); err != nil {
		h.redirectResult(w, r, "error", err.Error())
		return
	}
	accounts, err := h.cf.ListAccounts()
	if err != nil {
		h.redirectResult(w, r, "error", "授权成功，但读取账户失败: "+err.Error())
		return
	}
	if len(accounts) == 0 {
		h.redirectResult(w, r, "error", "授权成功，但没有可用的 Cloudflare 账户")
		return
	}
	selected := selectCloudflareAccount(accounts, h.store.GetConfig().CFAccountID, h.cf.DefaultAccountID())
	if err := h.store.SetCloudflareAccount(selected.ID, selected.Name); err != nil {
		h.redirectResult(w, r, "error", "保存 Cloudflare 账户失败")
		return
	}
	h.redirectResult(w, r, "success", "")
}

// SelectAccount changes the account used by account-scoped API calls.
func (h *CloudflareOAuthHandler) SelectAccount(w http.ResponseWriter, r *http.Request) {
	var request models.CloudflareAccountRequest
	if err := readJSON(r, &request); err != nil || strings.TrimSpace(request.AccountID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "account_id is required"})
		return
	}
	accounts, err := h.cf.ListAccounts()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	for _, account := range accounts {
		if account.ID == request.AccountID {
			if err := h.store.SetCloudflareAccount(account.ID, account.Name); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save account selection"})
				return
			}
			writeJSON(w, http.StatusOK, account)
			return
		}
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": "account is not available to this authorization"})
}

// Disconnect revokes OAuth and clears its local credentials.
func (h *CloudflareOAuthHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	err := h.oauth.RevokeAndClear()
	response := map[string]string{"status": "ok"}
	if err != nil {
		response["warning"] = "本地凭据已清除，但 Cloudflare 撤销请求失败: " + err.Error()
	}
	writeJSON(w, http.StatusOK, response)
}

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
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := firstForwardedValue(r.Header.Get("X-Forwarded-Proto")); forwarded == "http" || forwarded == "https" {
		scheme = forwarded
	}
	host := firstForwardedValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	return fmt.Sprintf("%s://%s/api/cloudflare/oauth/callback", scheme, host)
}

func firstForwardedValue(value string) string {
	value, _, _ = strings.Cut(value, ",")
	return strings.TrimSpace(value)
}

func randomURLToken(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func selectCloudflareAccount(accounts []models.Account, preferredIDs ...string) models.Account {
	for _, preferredID := range preferredIDs {
		for _, account := range accounts {
			if preferredID != "" && account.ID == preferredID {
				return account
			}
		}
	}
	return accounts[0]
}
