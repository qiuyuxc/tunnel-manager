package services

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"tunnel-manager/models"
	"tunnel-manager/store"
)

// Errors reported by the passkey ceremonies.
var (
	// ErrPasskeyCeremonyExpired reports an unknown or expired ceremony token;
	// the browser has to start the ceremony over.
	ErrPasskeyCeremonyExpired = errors.New("通行密钥验证已过期，请重试")
	// ErrNoPasskeys reports an account that has no credential to assert with.
	ErrNoPasskeys = errors.New("该账户尚未绑定通行密钥")
)

// passkeyCeremonyTTL bounds how long a begun ceremony stays valid.
const passkeyCeremonyTTL = 5 * time.Minute

// maxPasskeyCeremonies caps the in-flight ceremonies so a flood of begin calls
// cannot grow the map without limit.
const maxPasskeyCeremonies = 256

// passkeyCeremony is the server-side state of one in-flight ceremony. The
// library hands it back on begin and expects it on finish.
type passkeyCeremony struct {
	session   webauthn.SessionData
	userID    string
	expiresAt time.Time
}

// RelyingParty is the WebAuthn scope resolved for one request.
type RelyingParty struct {
	ID      string
	Origins []string
}

// PasskeyService runs WebAuthn ceremonies: it resolves the relying party for
// each request and keeps the short-lived ceremony state between the browser's
// begin and finish calls.
type PasskeyService struct {
	store      *store.Store
	mu         sync.Mutex
	ceremonies map[string]passkeyCeremony
	now        func() time.Time
}

// NewPasskeyService creates the passkey service.
func NewPasskeyService(st *store.Store) *PasskeyService {
	return &PasskeyService{store: st, ceremonies: map[string]passkeyCeremony{}, now: time.Now}
}

// passkeyUser adapts an account record to the library's User interface. The
// user handle is the account id, which the authenticator stores alongside the
// credential so a usernameless login can resolve the account again.
type passkeyUser struct {
	user        models.User
	credentials []webauthn.Credential
}

func (u *passkeyUser) WebAuthnID() []byte { return []byte(u.user.ID) }

func (u *passkeyUser) WebAuthnName() string { return u.user.Username }

func (u *passkeyUser) WebAuthnDisplayName() string {
	if u.user.Nickname != "" {
		return u.user.Nickname
	}
	return u.user.Username
}

func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// DecodePasskeyCredential parses a stored credential.
func DecodePasskeyCredential(raw string) (*webauthn.Credential, error) {
	var credential webauthn.Credential
	if err := json.Unmarshal([]byte(raw), &credential); err != nil {
		return nil, err
	}
	return &credential, nil
}

// EncodePasskeyCredential serializes a credential for storage.
func EncodePasskeyCredential(credential *webauthn.Credential) (string, error) {
	data, err := json.Marshal(credential)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// PasskeyCredentialID is the stable row id of a credential: the base64url form
// of the WebAuthn credential id the browser reports.
func PasskeyCredentialID(credential *webauthn.Credential) string {
	return base64.RawURLEncoding.EncodeToString(credential.ID)
}

// ValidatePasskeySettings rejects a relying party and origin list that can never
// work together, at the moment they are saved.
//
// The pair is only consumed when a ceremony starts, and an explicit origin list
// overrides the one derived from the request, so a relying party that no longer
// matches the stored origins fails every passkey request with a mismatch. That
// is a bad place to discover a typo: nothing about the settings page suggests
// the two fields have to agree.
func ValidatePasskeySettings(rpID, origins string) error {
	rpID = passkeyHost(rpID)
	if rpID == "" {
		return nil
	}
	for _, origin := range splitOrigins(origins) {
		if err := validatePasskeyOrigin(origin, rpID); err != nil {
			return err
		}
	}
	return nil
}

// ResolveRelyingParty derives the relying party for a request: administrator
// overrides win, otherwise the panel host (seeded from the first administrator
// login) and the request's own host.
func (s *PasskeyService) ResolveRelyingParty(r *http.Request) (RelyingParty, error) {
	settings := s.store.GetAppSettings()
	rpID := strings.TrimSpace(settings.PasskeyRPID)
	if rpID == "" {
		rpID = strings.TrimSpace(s.store.GetConfig().PanelHost)
	}
	if rpID == "" {
		rpID = r.Host
	}
	rpID = passkeyHost(rpID)
	if rpID == "" {
		return RelyingParty{}, errors.New("无法确定通行密钥域名，请在全局设置中填写依赖方 ID")
	}

	origins := splitOrigins(settings.PasskeyOrigins)
	if len(origins) == 0 {
		scheme := requestScheme(r)
		host := strings.ToLower(strings.TrimSpace(r.Host))
		origins = append(origins, scheme+"://"+host)
		// A TLS-terminating proxy that forgets X-Forwarded-Proto would
		// otherwise make the panel unusable behind HTTPS: accept both schemes
		// for a real domain (the browser only ever reports the secure one).
		if scheme == "http" && !isLocalhost(rpID) {
			origins = append(origins, "https://"+host)
		}
	}
	for _, origin := range origins {
		if err := validatePasskeyOrigin(origin, rpID); err != nil {
			return RelyingParty{}, err
		}
	}
	return RelyingParty{ID: rpID, Origins: origins}, nil
}

// webauthnFor builds the library instance for one request.
func (s *PasskeyService) webauthnFor(r *http.Request) (*webauthn.WebAuthn, RelyingParty, error) {
	rp, err := s.ResolveRelyingParty(r)
	if err != nil {
		return nil, rp, err
	}
	displayName := strings.TrimSpace(s.store.GetConfig().SiteName)
	if displayName == "" {
		displayName = "Tunnel Manager"
	}
	instance, err := webauthn.New(&webauthn.Config{
		RPID:          rp.ID,
		RPDisplayName: displayName,
		RPOrigins:     rp.Origins,
	})
	if err != nil {
		return nil, rp, fmt.Errorf("初始化通行密钥依赖方失败: %w", err)
	}
	return instance, rp, nil
}

// loadPasskeyUser loads an account together with its parsed credentials.
func (s *PasskeyService) loadPasskeyUser(userID string) (*passkeyUser, error) {
	user, ok := s.store.GetUserByID(userID)
	if !ok {
		return nil, store.ErrUserNotFound
	}
	entries, err := s.store.ListPasskeys(userID)
	if err != nil {
		return nil, err
	}
	credentials := make([]webauthn.Credential, 0, len(entries))
	for _, entry := range entries {
		credential, err := DecodePasskeyCredential(entry.Credential)
		if err != nil {
			// A credential we cannot parse cannot authenticate anyone, so skip
			// it instead of failing every ceremony for this account.
			continue
		}
		credentials = append(credentials, *credential)
	}
	return &passkeyUser{user: user, credentials: credentials}, nil
}

// BeginRegistration starts a registration ceremony for an account.
func (s *PasskeyService) BeginRegistration(r *http.Request, userID string) (*protocol.CredentialCreation, string, RelyingParty, error) {
	instance, rp, err := s.webauthnFor(r)
	if err != nil {
		return nil, "", rp, err
	}
	user, err := s.loadPasskeyUser(userID)
	if err != nil {
		return nil, "", rp, err
	}
	options, session, err := instance.BeginRegistration(user,
		// Prefer a discoverable credential so the account can later sign in
		// without typing a username, and keep already bound authenticators out
		// of the picker.
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementPreferred,
			UserVerification: protocol.VerificationPreferred,
		}),
		webauthn.WithExclusions(webauthn.Credentials(user.WebAuthnCredentials()).CredentialDescriptors()),
	)
	if err != nil {
		return nil, "", rp, fmt.Errorf("开始通行密钥注册失败: %w", err)
	}
	token, err := s.storeCeremony(session, userID)
	if err != nil {
		return nil, "", rp, err
	}
	return options, token, rp, nil
}

// FinishRegistration validates the browser's attestation and returns the
// credential to store.
func (s *PasskeyService) FinishRegistration(r *http.Request, token, userID string, raw []byte) (*webauthn.Credential, RelyingParty, error) {
	instance, rp, err := s.webauthnFor(r)
	if err != nil {
		return nil, rp, err
	}
	ceremony, ok := s.claimCeremony(token)
	if !ok || ceremony.userID != userID {
		return nil, rp, ErrPasskeyCeremonyExpired
	}
	user, err := s.loadPasskeyUser(userID)
	if err != nil {
		return nil, rp, err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(raw)
	if err != nil {
		return nil, rp, fmt.Errorf("通行密钥数据无效: %w", err)
	}
	credential, err := instance.CreateCredential(user, ceremony.session, parsed)
	if err != nil {
		return nil, rp, fmt.Errorf("通行密钥注册验证失败: %w", err)
	}
	return credential, rp, nil
}

// BeginLogin starts an assertion for a known account.
func (s *PasskeyService) BeginLogin(r *http.Request, userID string) (*protocol.CredentialAssertion, string, RelyingParty, error) {
	instance, rp, err := s.webauthnFor(r)
	if err != nil {
		return nil, "", rp, err
	}
	user, err := s.loadPasskeyUser(userID)
	if err != nil {
		return nil, "", rp, err
	}
	if len(user.credentials) == 0 {
		return nil, "", rp, ErrNoPasskeys
	}
	options, session, err := instance.BeginLogin(user, webauthn.WithUserVerification(protocol.VerificationPreferred))
	if err != nil {
		return nil, "", rp, fmt.Errorf("开始通行密钥登录失败: %w", err)
	}
	token, err := s.storeCeremony(session, userID)
	if err != nil {
		return nil, "", rp, err
	}
	return options, token, rp, nil
}

// BeginDiscoverableLogin starts a usernameless assertion: the browser picks one
// of its passkeys and reports which account it belongs to.
func (s *PasskeyService) BeginDiscoverableLogin(r *http.Request) (*protocol.CredentialAssertion, string, RelyingParty, error) {
	instance, rp, err := s.webauthnFor(r)
	if err != nil {
		return nil, "", rp, err
	}
	options, session, err := instance.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationPreferred))
	if err != nil {
		return nil, "", rp, fmt.Errorf("开始通行密钥登录失败: %w", err)
	}
	token, err := s.storeCeremony(session, "")
	if err != nil {
		return nil, "", rp, err
	}
	return options, token, rp, nil
}

// FinishLogin validates an assertion and reports which account it belongs to.
//
// Which validator runs follows the ceremony, not the caller: BeginLogin names
// the account up front and BeginDiscoverableLogin does not, and the library
// refuses a session that is validated the other way round. expectedUserID is
// therefore only a cross-check for flows that already know who they expect —
// passing it as the switch instead made every named-account sign-in fail with
// "Session was not initiated as a client-side discoverable login".
func (s *PasskeyService) FinishLogin(r *http.Request, token, expectedUserID string, raw []byte) (string, *webauthn.Credential, RelyingParty, error) {
	instance, rp, err := s.webauthnFor(r)
	if err != nil {
		return "", nil, rp, err
	}
	ceremony, ok := s.claimCeremony(token)
	if !ok {
		return "", nil, rp, ErrPasskeyCeremonyExpired
	}
	if expectedUserID != "" && ceremony.userID != "" && ceremony.userID != expectedUserID {
		return "", nil, rp, ErrPasskeyCeremonyExpired
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(raw)
	if err != nil {
		return "", nil, rp, fmt.Errorf("通行密钥数据无效: %w", err)
	}

	if ceremony.userID != "" {
		user, err := s.loadPasskeyUser(ceremony.userID)
		if err != nil {
			return "", nil, rp, err
		}
		credential, err := instance.ValidateLogin(user, ceremony.session, parsed)
		if err != nil {
			return "", nil, rp, fmt.Errorf("通行密钥验证失败: %w", err)
		}
		return ceremony.userID, credential, rp, nil
	}

	handler := func(_ []byte, userHandle []byte) (webauthn.User, error) {
		if len(userHandle) == 0 {
			return nil, errors.New("通行密钥缺少账户标识")
		}
		return s.loadPasskeyUser(string(userHandle))
	}
	user, credential, err := instance.ValidatePasskeyLogin(handler, ceremony.session, parsed)
	if err != nil {
		return "", nil, rp, fmt.Errorf("通行密钥验证失败: %w", err)
	}
	adapter, ok := user.(*passkeyUser)
	if !ok {
		return "", nil, rp, errors.New("通行密钥账户解析失败")
	}
	return adapter.user.ID, credential, rp, nil
}

// storeCeremony keeps one ceremony and returns the single-use token the browser
// sends back on finish.
func (s *PasskeyService) storeCeremony(session *webauthn.SessionData, userID string) (string, error) {
	token, err := randomPasskeyToken()
	if err != nil {
		return "", fmt.Errorf("生成通行密钥会话失败: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked()
	if len(s.ceremonies) >= maxPasskeyCeremonies {
		return "", errors.New("通行密钥请求过多，请稍后重试")
	}
	s.ceremonies[token] = passkeyCeremony{session: *session, userID: userID, expiresAt: s.now().Add(passkeyCeremonyTTL)}
	return token, nil
}

// claimCeremony consumes one ceremony: a token is single use.
func (s *PasskeyService) claimCeremony(token string) (passkeyCeremony, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ceremony, ok := s.ceremonies[token]
	if !ok {
		return passkeyCeremony{}, false
	}
	delete(s.ceremonies, token)
	if s.now().After(ceremony.expiresAt) {
		return passkeyCeremony{}, false
	}
	return ceremony, true
}

func (s *PasskeyService) pruneLocked() {
	now := s.now()
	for token, ceremony := range s.ceremonies {
		if now.After(ceremony.expiresAt) {
			delete(s.ceremonies, token)
		}
	}
}

// randomPasskeyToken returns a 32 byte URL-safe random token.
func randomPasskeyToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// splitOrigins parses the administrator's origin list (comma or whitespace
// separated) into normalized origins.
// splitOrigins reads the origin list a person typed. Besides the ASCII comma the
// field documents it accepts the full-width one a Chinese input method produces,
// and either width of space. A separator that is silently taken for part of a
// value collapses the whole list into one unparseable origin, which then fails
// with a message about paths and fragments rather than about separators.
func splitOrigins(raw string) []string {
	out := []string{}
	for _, candidate := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == ' ' || r == '\u3000' ||
			r == '\t' || r == '\n' || r == '\r'
	}) {
		trimmed := strings.TrimSuffix(strings.TrimSpace(candidate), "/")
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// validatePasskeyOrigin enforces the WebAuthn rule that an origin's host equals
// the relying party id or is a subdomain of it.
func validatePasskeyOrigin(origin, rpID string) error {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return fmt.Errorf("通行密钥来源 %q 无效：需形如 https://panel.example.com", origin)
	}
	if (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("通行密钥来源 %q 不能带路径、参数或片段", origin)
	}
	host := strings.ToLower(parsed.Hostname())
	if host != rpID && !strings.HasSuffix(host, "."+rpID) {
		return fmt.Errorf("通行密钥来源 %q 与依赖方 ID %s 不匹配", origin, rpID)
	}
	return nil
}

// requestScheme reports the scheme the browser used, honouring the usual
// reverse proxy header.
func requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		if idx := strings.Index(proto, ","); idx >= 0 {
			proto = proto[:idx]
		}
		if scheme := strings.ToLower(strings.TrimSpace(proto)); scheme != "" {
			return scheme
		}
	}
	return "http"
}

// passkeyHost normalizes a hostname for use as a relying party id: no port, no
// trailing dot, lower case.
func passkeyHost(host string) string {
	host = strings.TrimSpace(host)
	if idx := strings.LastIndex(host, ":"); idx >= 0 && !strings.Contains(host[idx:], "]") {
		host = host[:idx]
	}
	return strings.ToLower(strings.TrimSuffix(strings.Trim(host, "[]"), "."))
}

// isLocalhost reports whether a relying party id is a local development host,
// where WebAuthn also accepts an insecure origin.
func isLocalhost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasSuffix(host, ".localhost")
}
