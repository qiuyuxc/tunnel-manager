package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/store"
)

const (
	cloudflareAuthorizationEndpoint = "https://dash.cloudflare.com/oauth2/auth"
	cloudflareTokenEndpoint         = "https://dash.cloudflare.com/oauth2/token"
	cloudflareRevokeEndpoint        = "https://dash.cloudflare.com/oauth2/revoke"
	cloudflareAccessTokenPurpose    = "cloudflare-oauth-access-token"
	cloudflareRefreshTokenPurpose   = "cloudflare-oauth-refresh-token"
)

var ErrCloudflareOAuthNotConnected = errors.New("Cloudflare OAuth is not connected")

// CloudflareOAuthConfig contains server-side OAuth client settings.
type CloudflareOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       string
}

// CloudflareOAuth manages authorization code exchange and access token refresh.
type CloudflareOAuth struct {
	store          *store.Store
	encryptionKey  []byte
	config         CloudflareOAuthConfig
	httpClient     *http.Client
	now            func() time.Time
	mu             sync.Mutex
	tokenEndpoint  string
	revokeEndpoint string
}

type cloudflareOAuthToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int64  `json:"expires_in"`
	Error        string `json:"error"`
	Description  string `json:"error_description"`
}

// NewCloudflareOAuth creates the OAuth credential manager.
func NewCloudflareOAuth(st *store.Store, encryptionKey []byte, config CloudflareOAuthConfig) *CloudflareOAuth {
	return &CloudflareOAuth{
		store:          st,
		encryptionKey:  append([]byte(nil), encryptionKey...),
		config:         config,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		now:            time.Now,
		tokenEndpoint:  cloudflareTokenEndpoint,
		revokeEndpoint: cloudflareRevokeEndpoint,
	}
}

// Configured reports whether the server has all settings needed for OAuth.
func (o *CloudflareOAuth) Configured() bool {
	return o.config.ClientID != "" && o.config.ClientSecret != "" && len(o.encryptionKey) == 32
}

// ConfigurationError explains why OAuth cannot currently be started.
func (o *CloudflareOAuth) ConfigurationError() string {
	if o.config.ClientID == "" || o.config.ClientSecret == "" {
		return "CF_OAUTH_CLIENT_ID 和 CF_OAUTH_CLIENT_SECRET 未配置"
	}
	if len(o.encryptionKey) != 32 {
		return "APP_ENCRYPTION_KEY 未配置或无效"
	}
	return ""
}

// RedirectURI returns the configured callback URI, if one was explicitly set.
func (o *CloudflareOAuth) RedirectURI() string {
	return o.config.RedirectURI
}

// Connected reports whether encrypted OAuth credentials exist.
func (o *CloudflareOAuth) Connected() bool {
	return o.store.GetConfig().CFOAuthAccessToken != ""
}

// AccountID returns the account selected for OAuth API calls.
func (o *CloudflareOAuth) AccountID() string {
	return o.store.GetConfig().CFAccountID
}

// AuthorizationURL builds Cloudflare's authorization URL with PKCE.
func (o *CloudflareOAuth) AuthorizationURL(redirectURI, state, codeChallenge string) (string, error) {
	if !o.Configured() {
		return "", errors.New(o.ConfigurationError())
	}
	if redirectURI == "" || state == "" || codeChallenge == "" {
		return "", errors.New("OAuth redirect URI, state and code challenge are required")
	}
	query := url.Values{
		"response_type":         {"code"},
		"client_id":             {o.config.ClientID},
		"redirect_uri":          {redirectURI},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}
	if scopes := strings.TrimSpace(o.config.Scopes); scopes != "" {
		query.Set("scope", scopes)
	}
	return cloudflareAuthorizationEndpoint + "?" + query.Encode(), nil
}

// ExchangeCode exchanges an authorization code and persists encrypted credentials.
func (o *CloudflareOAuth) ExchangeCode(code, redirectURI, codeVerifier string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.Configured() {
		return errors.New(o.ConfigurationError())
	}
	values := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {codeVerifier},
	}
	token, err := o.requestToken(values)
	if err != nil {
		return err
	}
	return o.persistToken(token, "")
}

// AccessToken returns a usable access token, refreshing it when needed.
func (o *CloudflareOAuth) AccessToken() (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	config := o.store.GetConfig()
	if config.CFOAuthAccessToken == "" {
		return "", ErrCloudflareOAuthNotConnected
	}
	accessToken, err := auth.DecryptSecret(o.encryptionKey, cloudflareAccessTokenPurpose, config.CFOAuthAccessToken)
	if err != nil {
		return "", fmt.Errorf("decrypt Cloudflare OAuth access token: %w", err)
	}
	if config.CFOAuthExpiresAt == 0 || o.now().Add(time.Minute).Unix() < config.CFOAuthExpiresAt {
		return string(accessToken), nil
	}
	if config.CFOAuthRefreshToken == "" {
		if o.now().Unix() < config.CFOAuthExpiresAt {
			return string(accessToken), nil
		}
		return "", errors.New("Cloudflare OAuth access token expired and no refresh token is available")
	}
	refreshToken, err := auth.DecryptSecret(o.encryptionKey, cloudflareRefreshTokenPurpose, config.CFOAuthRefreshToken)
	if err != nil {
		return "", fmt.Errorf("decrypt Cloudflare OAuth refresh token: %w", err)
	}
	token, err := o.requestToken(url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {string(refreshToken)},
	})
	if err != nil {
		return "", fmt.Errorf("refresh Cloudflare OAuth token: %w", err)
	}
	if err := o.persistToken(token, config.CFOAuthRefreshToken); err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

// RevokeAndClear revokes the best available token and always clears local OAuth state.
func (o *CloudflareOAuth) RevokeAndClear() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	config := o.store.GetConfig()
	var revokeErr error
	encrypted := config.CFOAuthRefreshToken
	purpose := cloudflareRefreshTokenPurpose
	if encrypted == "" {
		encrypted = config.CFOAuthAccessToken
		purpose = cloudflareAccessTokenPurpose
	}
	if encrypted != "" && o.Configured() {
		plain, err := auth.DecryptSecret(o.encryptionKey, purpose, encrypted)
		if err != nil {
			revokeErr = err
		} else {
			revokeErr = o.revoke(string(plain))
		}
	}
	if err := o.store.ClearCloudflareOAuth(); err != nil {
		return err
	}
	return revokeErr
}

func (o *CloudflareOAuth) requestToken(values url.Values) (cloudflareOAuthToken, error) {
	req, err := http.NewRequest(http.MethodPost, o.tokenEndpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return cloudflareOAuthToken{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(o.config.ClientID, o.config.ClientSecret)
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return cloudflareOAuthToken{}, fmt.Errorf("OAuth token request failed: %w", err)
	}
	defer resp.Body.Close()
	var token cloudflareOAuthToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return cloudflareOAuthToken{}, fmt.Errorf("decode OAuth token response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || token.Error != "" {
		message := token.Description
		if message == "" {
			message = token.Error
		}
		if message == "" {
			message = resp.Status
		}
		return cloudflareOAuthToken{}, fmt.Errorf("Cloudflare OAuth rejected token request: %s", message)
	}
	if token.AccessToken == "" {
		return cloudflareOAuthToken{}, errors.New("Cloudflare OAuth response did not include an access token")
	}
	return token, nil
}

func (o *CloudflareOAuth) persistToken(token cloudflareOAuthToken, existingRefreshToken string) error {
	accessToken, err := auth.EncryptSecret(o.encryptionKey, cloudflareAccessTokenPurpose, []byte(token.AccessToken))
	if err != nil {
		return fmt.Errorf("encrypt Cloudflare OAuth access token: %w", err)
	}
	refreshToken := existingRefreshToken
	if token.RefreshToken != "" {
		refreshToken, err = auth.EncryptSecret(o.encryptionKey, cloudflareRefreshTokenPurpose, []byte(token.RefreshToken))
		if err != nil {
			return fmt.Errorf("encrypt Cloudflare OAuth refresh token: %w", err)
		}
	}
	var expiresAt time.Time
	if token.ExpiresIn > 0 {
		expiresAt = o.now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	if err := o.store.SetCloudflareOAuth(accessToken, refreshToken, expiresAt, token.Scope); err != nil {
		return fmt.Errorf("save Cloudflare OAuth token: %w", err)
	}
	return nil
}

func (o *CloudflareOAuth) revoke(token string) error {
	values := url.Values{"token": {token}}
	req, err := http.NewRequest(http.MethodPost, o.revokeEndpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(o.config.ClientID, o.config.ClientSecret)
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("revoke Cloudflare OAuth token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("revoke Cloudflare OAuth token: %s", resp.Status)
	}
	return nil
}

// AccessTokenFor returns a usable access token for one connection, refreshing
// and rotating the stored tokens when they are about to expire.
func (o *CloudflareOAuth) AccessTokenFor(conn models.CFConnection) (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if conn.AccessToken == "" {
		return "", ErrCloudflareOAuthNotConnected
	}
	accessToken, err := auth.DecryptSecret(o.encryptionKey, cloudflareAccessTokenPurpose, conn.AccessToken)
	if err != nil {
		return "", fmt.Errorf("decrypt Cloudflare OAuth access token: %w", err)
	}
	if conn.ExpiresAt == 0 || o.now().Add(time.Minute).Unix() < conn.ExpiresAt {
		return string(accessToken), nil
	}
	if conn.RefreshToken == "" {
		return "", errors.New("Cloudflare OAuth access token expired and no refresh token is available")
	}
	refreshToken, err := auth.DecryptSecret(o.encryptionKey, cloudflareRefreshTokenPurpose, conn.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("decrypt Cloudflare OAuth refresh token: %w", err)
	}
	token, err := o.requestToken(url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {string(refreshToken)},
	})
	if err != nil {
		return "", fmt.Errorf("refresh Cloudflare OAuth token: %w", err)
	}
	encAccess, err := auth.EncryptSecret(o.encryptionKey, cloudflareAccessTokenPurpose, []byte(token.AccessToken))
	if err != nil {
		return "", fmt.Errorf("encrypt Cloudflare OAuth access token: %w", err)
	}
	encRefresh := conn.RefreshToken
	if token.RefreshToken != "" {
		encRefresh, err = auth.EncryptSecret(o.encryptionKey, cloudflareRefreshTokenPurpose, []byte(token.RefreshToken))
		if err != nil {
			return "", fmt.Errorf("encrypt Cloudflare OAuth refresh token: %w", err)
		}
	}
	var expiresAt int64
	if token.ExpiresIn > 0 {
		expiresAt = o.now().Add(time.Duration(token.ExpiresIn) * time.Second).Unix()
	}
	if err := o.store.UpdateCFConnectionTokens(conn.ID, encAccess, encRefresh, expiresAt, token.Scope); err != nil {
		return "", fmt.Errorf("save Cloudflare OAuth token: %w", err)
	}
	return token.AccessToken, nil
}

// ExchangeCodeForUser exchanges an authorization code into a new connection
// owned by userID and returns it.
func (o *CloudflareOAuth) ExchangeCodeForUser(userID, code, redirectURI, codeVerifier string) (models.CFConnection, error) {
	o.mu.Lock()
	if !o.Configured() {
		o.mu.Unlock()
		return models.CFConnection{}, errors.New(o.ConfigurationError())
	}
	values := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {codeVerifier},
	}
	token, err := o.requestToken(values)
	o.mu.Unlock()
	if err != nil {
		return models.CFConnection{}, err
	}

	encAccess, err := auth.EncryptSecret(o.encryptionKey, cloudflareAccessTokenPurpose, []byte(token.AccessToken))
	if err != nil {
		return models.CFConnection{}, fmt.Errorf("encrypt Cloudflare OAuth access token: %w", err)
	}
	encRefresh := ""
	if token.RefreshToken != "" {
		encRefresh, err = auth.EncryptSecret(o.encryptionKey, cloudflareRefreshTokenPurpose, []byte(token.RefreshToken))
		if err != nil {
			return models.CFConnection{}, fmt.Errorf("encrypt Cloudflare OAuth refresh token: %w", err)
		}
	}
	var expiresAt int64
	if token.ExpiresIn > 0 {
		expiresAt = o.now().Add(time.Duration(token.ExpiresIn) * time.Second).Unix()
	}
	conn := models.CFConnection{
		UserID:       userID,
		Label:        "Cloudflare 账户",
		AccessToken:  encAccess,
		RefreshToken: encRefresh,
		ExpiresAt:    expiresAt,
		Scope:        token.Scope,
	}
	connID, err := o.store.CreateCFConnection(conn)
	if err != nil {
		return models.CFConnection{}, fmt.Errorf("save Cloudflare connection: %w", err)
	}
	conn.ID = connID
	return conn, nil
}

// RevokeConnection revokes the grant behind a connection and deletes it.
func (o *CloudflareOAuth) RevokeConnection(conn models.CFConnection) error {
	var revokeErr error
	if o.Configured() && (conn.RefreshToken != "" || conn.AccessToken != "") {
		encrypted := conn.RefreshToken
		purpose := cloudflareRefreshTokenPurpose
		if encrypted == "" {
			encrypted = conn.AccessToken
			purpose = cloudflareAccessTokenPurpose
		}
		plain, err := auth.DecryptSecret(o.encryptionKey, purpose, encrypted)
		if err != nil {
			revokeErr = err
		} else {
			revokeErr = o.revoke(string(plain))
		}
	}
	if err := o.store.DeleteCFConnection(conn.UserID, conn.ID); err != nil {
		return err
	}
	return revokeErr
}
