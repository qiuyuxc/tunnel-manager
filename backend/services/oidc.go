package services

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"tunnel-manager/models"
)

// OIDCClient uses discovery and the library's signature, issuer, audience and
// expiry checks. The caller owns browser state, PKCE and nonce lifecycle.
type OIDCClient struct {
	provider *oidc.Provider
	config   oauth2.Config
	verifier *oidc.IDTokenVerifier
	http     *http.Client
}

type OIDCClaims struct {
	Issuer        string
	Subject       string
	Email         string
	EmailVerified bool
	AuthTime      int64
}

func ValidateOIDCURL(value string) error {
	u, err := url.Parse(value)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" {
		return errors.New("地址必须是无查询参数的完整 HTTPS URL")
	}
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1") {
		return nil
	}
	return errors.New("OIDC 必须使用 HTTPS，本机调试可使用 localhost HTTP")
}

func NewOIDCClient(ctx context.Context, settings models.OIDCProvider, secret string) (*OIDCClient, error) {
	if err := ValidateOIDCURL(settings.Issuer); err != nil {
		return nil, err
	}
	if err := ValidateOIDCURL(settings.RedirectURI); err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	ctx = oidc.ClientContext(ctx, client)
	provider, err := oidc.NewProvider(ctx, settings.Issuer)
	if err != nil {
		return nil, errors.New("读取 OIDC Discovery 失败，请检查 Issuer 和网络连接")
	}
	endpoint := provider.Endpoint()
	for _, endpointURL := range []string{endpoint.AuthURL, endpoint.TokenURL} {
		if err := ValidateOIDCURL(endpointURL); err != nil {
			return nil, errors.New("OIDC Discovery 返回了不安全的认证端点")
		}
	}
	scopes := strings.Fields(settings.Scopes)
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail}
	}
	return &OIDCClient{provider: provider, http: client, verifier: provider.Verifier(&oidc.Config{ClientID: settings.ClientID}),
		config: oauth2.Config{ClientID: settings.ClientID, ClientSecret: secret, RedirectURL: settings.RedirectURI, Endpoint: endpoint, Scopes: scopes}}, nil
}

func (c *OIDCClient) AuthorizationURL(state, nonce, verifier string, reauthenticate bool) string {
	options := []oauth2.AuthCodeOption{oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)}
	if reauthenticate {
		options = append(options, oauth2.SetAuthURLParam("prompt", "login"), oauth2.SetAuthURLParam("max_age", "0"))
	}
	return c.config.AuthCodeURL(state, options...)
}

func (c *OIDCClient) Exchange(ctx context.Context, code, nonce, verifier string) (OIDCClaims, error) {
	ctx = oidc.ClientContext(ctx, c.http)
	token, err := c.config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return OIDCClaims{}, errors.New("OIDC 授权码交换失败")
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return OIDCClaims{}, errors.New("OIDC 未返回 ID Token")
	}
	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return OIDCClaims{}, errors.New("OIDC 身份凭据校验失败")
	}
	if idToken.Nonce != nonce || idToken.Subject == "" {
		return OIDCClaims{}, errors.New("OIDC nonce 或用户标识无效")
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		AuthTime      int64  `json:"auth_time"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return OIDCClaims{}, errors.New("OIDC 用户信息无效")
	}
	authTime := claims.AuthTime
	// Some providers return email only from UserInfo. Never accept UserInfo
	// belonging to a different subject than the already-verified ID Token.
	if claims.Email == "" {
		info, err := c.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
		if err != nil || info.Subject != idToken.Subject {
			return OIDCClaims{}, errors.New("OIDC UserInfo 校验失败")
		}
		if err := info.Claims(&claims); err != nil {
			return OIDCClaims{}, errors.New("OIDC 邮箱信息无效")
		}
	}
	return OIDCClaims{Issuer: idToken.Issuer, Subject: idToken.Subject, Email: strings.ToLower(strings.TrimSpace(claims.Email)), EmailVerified: claims.EmailVerified, AuthTime: authTime}, nil
}
