package models

// Roles and account statuses.
const (
	RoleAdmin = "admin"
	RoleUser  = "user"

	UserActive   = "active"
	UserDisabled = "disabled"
)

// Permission keys granted through user groups. Administrators bypass them.
const (
	PermTunnels      = "tunnels"
	PermDomainBind   = "domain_bind"
	PermDNS          = "dns"
	PermMonitors     = "monitors"
	PermOAuthConnect = "oauth_connect"
)

// AllPermissions lists every permission key in display order.
var AllPermissions = []string{PermTunnels, PermDomainBind, PermDNS, PermMonitors, PermOAuthConnect}

// Invite modes for the registration policy.
const (
	InviteModeOff      = "off"
	InviteModeOptional = "optional"
	InviteModeRequired = "required"
)

// User is the internal account record, including secrets. API responses use
// UserView instead.
type User struct {
	ID                     string
	Username               string
	Nickname               string
	Avatar                 string
	Email                  string
	PasswordHash           string
	Role                   string
	GroupID                string
	Status                 string
	EmailVerified          bool
	TOTPEnabled            bool
	TOTPSecretEncrypted    string
	TOTPLastAcceptedStep   int64
	TOTPRecoveryCodeHashes []string
	CreatedAt              int64
	LastLoginAt            int64
	ActiveCFConnectionID   string
	// PasswordLoginDisabled blocks password sign-in for this account; it can
	// only be set while at least one passkey is bound.
	PasswordLoginDisabled bool
}

// UserView is the API-safe projection of a user.
type UserView struct {
	ID            string   `json:"id"`
	Username      string   `json:"username"`
	Nickname      string   `json:"nickname"`
	Avatar        string   `json:"avatar"`
	Email         string   `json:"email"`
	Role          string   `json:"role"`
	GroupID       string   `json:"group_id"`
	GroupName     string   `json:"group_name,omitempty"`
	Status        string   `json:"status"`
	EmailVerified bool     `json:"email_verified"`
	TOTPEnabled   bool     `json:"totp_enabled"`
	CreatedAt     int64    `json:"created_at"`
	LastLoginAt   int64    `json:"last_login_at"`
	Permissions   []string `json:"permissions,omitempty"`
	// Passkeys counts bound passkeys; PasswordLoginDisabled reports whether
	// the account can still sign in with a password.
	Passkeys              int  `json:"passkeys"`
	PasswordLoginDisabled bool `json:"password_login_disabled"`
}

// UserGroup bundles a set of permissions granted to invited users.
type UserGroup struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	Builtin     bool     `json:"builtin"`
	CreatedAt   int64    `json:"created_at"`
}

// Invite grants registration under a specific user group.
type Invite struct {
	Code      string `json:"code"`
	GroupID   string `json:"group_id"`
	MaxUses   int    `json:"max_uses"`
	UsedCount int    `json:"used_count"`
	ExpiresAt int64  `json:"expires_at"`
	Enabled   bool   `json:"enabled"`
	CreatedAt int64  `json:"created_at"`
}

// AppSettings holds the registration policy knobs managed in the admin panel.
type AppSettings struct {
	RegistrationEnabled bool   `json:"registration_enabled"`
	InviteMode          string `json:"invite_mode"`
	DefaultGroupID      string `json:"default_group_id,omitempty"`
	// When true, registration never asks for an email verification code even
	// though SMTP is configured (SMTP stays active for alerts).
	EmailVerifyDisabled bool `json:"email_verify_disabled"`
	// Optional Cloudflare Turnstile human verification. Secret is stored
	// encrypted at rest.
	TurnstileEnabled bool   `json:"turnstile_enabled"`
	TurnstileSiteKey string `json:"turnstile_site_key"`
	TurnstileSecret  string `json:"turnstile_secret,omitempty"`
	// ExperimentalFeatures controls whether lab navigation is exposed.
	ExperimentalFeatures bool `json:"experimental_features_enabled"`
	// AuditRetentionDays is how long audit entries are kept. Zero means the
	// default window (models.DefaultAuditRetentionDays); a negative value
	// keeps entries forever.
	AuditRetentionDays int `json:"audit_retention_days"`
	// PasswordLoginDisabled turns off password sign-in for the whole panel:
	// every account must use a passkey. Only settable while an active
	// administrator has one bound, so the panel cannot lock itself out.
	PasswordLoginDisabled bool `json:"password_login_disabled"`
	// PasskeyRPID overrides the WebAuthn relying party id; empty derives it
	// from the panel host (or the request host). PasskeyOrigins lists extra
	// accepted origins, comma or space separated; empty derives them from the
	// request scheme and host.
	PasskeyRPID    string `json:"passkey_rp_id,omitempty"`
	PasskeyOrigins string `json:"passkey_origins,omitempty"`
	// PasskeyAndroidPackage and PasskeyAndroidFingerprints feed
	// /.well-known/assetlinks.json, which the native Android app needs before
	// Credential Manager will create or use a passkey for this domain. The
	// fingerprints are SHA-256 certificate hashes, comma or newline separated;
	// empty means the shared debug keystore this project builds with.
	PasskeyAndroidPackage      string `json:"passkey_android_package,omitempty"`
	PasskeyAndroidFingerprints string `json:"passkey_android_fingerprints,omitempty"`
	// Sign-in rate limiting. The account budget is the hard stop against
	// guessing one password; the address budget stops a single machine walking
	// a list of accounts. A zero threshold means "never configured" and
	// resolves to the default; a negative one disables that dimension.
	//
	// The switch is stored inverted so an upgraded panel is protected by
	// default: a settings document written before this feature existed has no
	// field at all, and the zero value has to mean "on".
	RateLimitDisabled           bool `json:"rate_limit_disabled"`
	RateLimitPerAccount         int  `json:"rate_limit_per_account"`
	RateLimitPerIP              int  `json:"rate_limit_per_ip"`
	RateLimitWindowMinutes      int  `json:"rate_limit_window_minutes"`
	RateLimitFamiliarMultiplier int  `json:"rate_limit_familiar_multiplier"`
	RateLimitNotify             bool `json:"rate_limit_notify"`
}

// Sign-in rate limit defaults and bounds. The bounds exist so a hand-edited
// settings document cannot turn a typo into a week-long lockout, or a
// multiplier into an unlimited budget.
const (
	DefaultRateLimitPerAccount   = 5
	DefaultRateLimitPerIP        = 20
	DefaultRateLimitWindowMin    = 15
	DefaultRateLimitFamiliarMult = 3

	MaxRateLimitPerAccount   = 1000
	MaxRateLimitPerIP        = 100000
	MaxRateLimitWindowMin    = 24 * 60
	MaxRateLimitFamiliarMult = 50
)

// RateLimits is the resolved sign-in rate limit: stored values with defaults
// filled in and ranges clamped. A threshold of zero reaches the limiter as
// zero, which is how that dimension is switched off.
type RateLimits struct {
	Enabled            bool
	PerAccount         int
	PerIP              int
	WindowMinutes      int
	FamiliarMultiplier int
	Notify             bool
}

// RateLimits resolves the stored sign-in rate limit. Nothing here is a
// security boundary on its own — it only decides what the limiter is told.
func (a AppSettings) RateLimits() RateLimits {
	return RateLimits{
		Enabled:            !a.RateLimitDisabled,
		PerAccount:         resolveLimit(a.RateLimitPerAccount, DefaultRateLimitPerAccount, MaxRateLimitPerAccount),
		PerIP:              resolveLimit(a.RateLimitPerIP, DefaultRateLimitPerIP, MaxRateLimitPerIP),
		WindowMinutes:      resolveWindow(a.RateLimitWindowMinutes),
		FamiliarMultiplier: resolveMultiplier(a.RateLimitFamiliarMultiplier),
		Notify:             a.RateLimitNotify,
	}
}

// resolveLimit turns one stored threshold into an effective one: zero means
// the default, negative means off, anything else is clamped to the bound.
func resolveLimit(stored, fallback, max int) int {
	switch {
	case stored == 0:
		return fallback
	case stored < 0:
		return 0
	case stored > max:
		return max
	default:
		return stored
	}
}

func resolveWindow(stored int) int {
	if stored <= 0 {
		return DefaultRateLimitWindowMin
	}
	if stored > MaxRateLimitWindowMin {
		return MaxRateLimitWindowMin
	}
	return stored
}

// A multiplier below one would tighten the limit for familiar networks,
// which is the opposite of what it is for; one means no relaxation.
func resolveMultiplier(stored int) int {
	switch {
	case stored <= 0:
		return DefaultRateLimitFamiliarMult
	case stored > MaxRateLimitFamiliarMult:
		return MaxRateLimitFamiliarMult
	default:
		return stored
	}
}

// AppSettingsView is the admin-facing projection of AppSettings; it never
// exposes the stored (encrypted) Turnstile secret.
type AppSettingsView struct {
	RegistrationEnabled   bool   `json:"registration_enabled"`
	InviteMode            string `json:"invite_mode"`
	DefaultGroupID        string `json:"default_group_id,omitempty"`
	EmailVerifyDisabled   bool   `json:"email_verify_disabled"`
	TurnstileEnabled      bool   `json:"turnstile_enabled"`
	TurnstileSiteKey      string `json:"turnstile_site_key"`
	TurnstileHasSecret    bool   `json:"turnstile_has_secret"`
	ExperimentalFeatures  bool   `json:"experimental_features_enabled"`
	AuditRetentionDays    int    `json:"audit_retention_days"`
	PasswordLoginDisabled bool   `json:"password_login_disabled"`
	PasskeyRPID           string `json:"passkey_rp_id"`
	PasskeyOrigins        string `json:"passkey_origins"`
	// PasskeyAdminReady reports whether at least one active administrator has a
	// passkey bound, i.e. whether the global password switch may be enabled.
	PasskeyAdminReady bool `json:"passkey_admin_ready"`
	// Android asset links: the package name and the SHA-256 certificate
	// fingerprints served to the native app, plus the effective list the
	// endpoint really serves after defaults and normalization.
	PasskeyAndroidPackage               string   `json:"passkey_android_package"`
	PasskeyAndroidFingerprints          string   `json:"passkey_android_fingerprints"`
	PasskeyAndroidFingerprintsEffective []string `json:"passkey_android_fingerprints_effective"`
	// Sign-in rate limiting, exactly as stored. The admin form edits these;
	// the limiter reads the resolved values from AppSettings.RateLimits.
	RateLimitEnabled            bool `json:"rate_limit_enabled"`
	RateLimitPerAccount         int  `json:"rate_limit_per_account"`
	RateLimitPerIP              int  `json:"rate_limit_per_ip"`
	RateLimitWindowMinutes      int  `json:"rate_limit_window_minutes"`
	RateLimitFamiliarMultiplier int  `json:"rate_limit_familiar_multiplier"`
	RateLimitNotify             bool `json:"rate_limit_notify"`
}

// OAuthSettings holds the Cloudflare OAuth client configured in the admin
// panel. Non-empty fields override the environment variables.
type OAuthSettings struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
	Scopes       string `json:"scopes"`
}

// SMTPSettings describes the outgoing mail relay. Password is stored
// encrypted at rest.
type SMTPSettings struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	From     string `json:"from"`
	TLSMode  string `json:"tls_mode"`
}

// Configured reports whether the relay has the minimum required settings.
func (s *SMTPSettings) Configured() bool {
	return s.Host != "" && s.Port > 0 && s.From != ""
}

// UserPrefs are per-user selections previously stored on the global config.
type UserPrefs struct {
	TunnelID         string `json:"tunnel_id"`
	TunnelName       string `json:"tunnel_name"`
	ServiceURL       string `json:"service_url"`
	SelectedZoneID   string `json:"selected_zone_id"`
	SelectedZoneName string `json:"selected_zone_name"`

	// Per-user notification preferences. Channels and events are persisted
	// as JSON in the user_prefs table.
	NotifyChannels         []string        `json:"notify_channels,omitempty"`
	NotifyEvents           map[string]bool `json:"notify_events,omitempty"`
	NotifyEmails           string          `json:"notify_emails,omitempty"`
	TGBotTokenEncrypted    string          `json:"tg_bot_token_encrypted,omitempty"`
	TGNotifyChatID         string          `json:"tg_notify_chat_id,omitempty"`
	TGRemoteTokenEncrypted string          `json:"tg_remote_token_encrypted,omitempty"`

	// Per-user Telegram remote-control bot. Notification and remote control
	// keep separate tokens: each may be configured independently, and either
	// side can be one-click reused from the other. TGOperatorIDs lists the
	// Telegram user IDs allowed to send commands. TGRemoteMode is "polling"
	// (default) or "webhook"; the webhook URL is the panel's public HTTPS
	// base address and the generated secret never leaves the backend.
	TGRemoteEnabled       bool   `json:"tg_remote_enabled,omitempty"`
	TGOperatorIDs         string `json:"tg_operator_ids,omitempty"`
	TGRemoteMode          string `json:"tg_remote_mode,omitempty"`
	TGRemoteWebhookURL    string `json:"tg_webhook_url,omitempty"`
	TGRemoteWebhookSecret string `json:"tg_webhook_secret,omitempty"`
	PreferredCNAME        string `json:"preferred_cname,omitempty"`
}

// Per-user notification channels and events.
const (
	NotifyChannelEmail    = "email"
	NotifyChannelTelegram = "telegram"
	NotifyEventLogin      = "login"
)

// AllNotifyEvents lists every supported notification event in display order.
var AllNotifyEvents = []string{NotifyEventLogin}

// NotifySettingsView is the API projection of one account's notification
// preferences; it never exposes the stored (encrypted) Telegram token.
type NotifySettingsView struct {
	Channels       []string        `json:"channels"`
	Events         map[string]bool `json:"events"`
	Emails         string          `json:"emails"`
	TGBotTokenSet  bool            `json:"tg_bot_token_set"`
	TGNotifyChatID string          `json:"tg_notify_chat_id"`
	TGRemoteBotSet bool            `json:"tg_remote_bot_set"`
}

// SaveNotifySettingsRequest is the body of PUT /api/notify/settings.
type SaveNotifySettingsRequest struct {
	Channels       []string        `json:"channels"`
	Events         map[string]bool `json:"events"`
	Emails         string          `json:"emails"`
	TGBotToken     string          `json:"tg_bot_token,omitempty"` // blank keeps the stored token
	TGNotifyChatID string          `json:"tg_notify_chat_id"`
}

// SessionUser is the authenticated identity attached to requests.
type SessionUser struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Nickname    string   `json:"nickname"`
	Avatar      string   `json:"avatar"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions,omitempty"`
}

// IsAdmin reports whether the session belongs to an administrator.
func (u *SessionUser) IsAdmin() bool {
	return u != nil && u.Role == RoleAdmin
}

// HasPerm reports whether the session may use a permission-gated area.
func (u *SessionUser) HasPerm(perm string) bool {
	if u == nil {
		return false
	}
	if u.IsAdmin() {
		return true
	}
	for _, p := range u.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// IsAPIKey reports whether the identity is the synthetic API-key administrator.
func (u *SessionUser) IsAPIKey() bool {
	return u != nil && u.Username == "api-key" && u.Role == RoleAdmin
}

// RegisterRequest is the body of POST /api/auth/register.
type RegisterRequest struct {
	Username          string `json:"username"`
	Email             string `json:"email"`
	Password          string `json:"password"`
	Invite            string `json:"invite"`
	VerifyCode        string `json:"verify_code"`
	TurnstileResponse string `json:"cf_turnstile_response,omitempty"`
}

// AuthConfigResponse tells the frontend how to render the register form.
type AuthConfigResponse struct {
	RegistrationEnabled bool   `json:"registration_enabled"`
	InviteMode          string `json:"invite_mode"`
	EmailVerifyEnabled  bool   `json:"email_verify_enabled"`
	// Turnstile human verification (optional). Site key is public.
	TurnstileEnabled bool   `json:"turnstile_enabled"`
	TurnstileSiteKey string `json:"turnstile_site_key"`
}

// SendCodeRequest is the body of POST /api/auth/send-code.
type SendCodeRequest struct {
	Email             string `json:"email"`
	TurnstileResponse string `json:"cf_turnstile_response,omitempty"`
}

// ResetPasswordRequest is the body of POST /api/auth/reset-password.
type ResetPasswordRequest struct {
	Email             string `json:"email"`
	Code              string `json:"code"`
	NewPassword       string `json:"new_password"`
	TurnstileResponse string `json:"cf_turnstile_response,omitempty"`
}

// MeResponse describes the authenticated user for the frontend.
type MeResponse struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Nickname    string   `json:"nickname"`
	Avatar      string   `json:"avatar"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

// CreateUserRequest is the body of POST /api/admin/users.
type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"`
	GroupID  string `json:"group_id,omitempty"`
}

// UpdateUserStatusRequest toggles an account between active and disabled.
type UpdateUserStatusRequest struct {
	Status string `json:"status"`
}

// UpdateUserGroupRequest reassigns an account group.
type UpdateUserGroupRequest struct {
	GroupID string `json:"group_id"`
}

// AdminResetPasswordRequest sets a new password without the old one.
type AdminResetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

// SaveGroupRequest is the body for creating and updating user groups.
type SaveGroupRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

// SaveInviteRequest is the body for creating invite codes.
type SaveInviteRequest struct {
	GroupID   string `json:"group_id"`
	MaxUses   int    `json:"max_uses"`
	ExpiresAt int64  `json:"expires_at"`
}

// UpdateInviteRequest toggles an invite code.
type UpdateInviteRequest struct {
	Enabled bool `json:"enabled"`
}

// SaveAppSettingsRequest is the body of PUT /api/admin/settings.
type SaveAppSettingsRequest struct {
	RegistrationEnabled bool   `json:"registration_enabled"`
	InviteMode          string `json:"invite_mode"`
	DefaultGroupID      string `json:"default_group_id"`
	EmailVerifyDisabled bool   `json:"email_verify_disabled"`
	// Optional Cloudflare Turnstile verification. A blank secret keeps the
	// stored one.
	TurnstileEnabled bool   `json:"turnstile_enabled"`
	TurnstileSiteKey string `json:"turnstile_site_key"`
	TurnstileSecret  string `json:"turnstile_secret,omitempty"`
	// TurnstileHasSecret is accepted for round-tripping the GET response;
	// it carries no save semantics.
	TurnstileHasSecret   bool `json:"turnstile_has_secret,omitempty"`
	ExperimentalFeatures bool `json:"experimental_features_enabled"`
	// AuditRetentionDays is how long the audit trail is kept. Zero means the
	// default window, a negative value keeps entries forever. Omitted keeps
	// the stored value, so partial saves do not reset the window.
	AuditRetentionDays *int `json:"audit_retention_days"`
	// PasswordLoginDisabled is a pointer so a partial save (for example the
	// Turnstile form) leaves the stored switch alone.
	PasswordLoginDisabled      *bool  `json:"password_login_disabled"`
	PasskeyRPID                string `json:"passkey_rp_id"`
	PasskeyOrigins             string `json:"passkey_origins"`
	PasskeyAndroidPackage      string `json:"passkey_android_package"`
	PasskeyAndroidFingerprints string `json:"passkey_android_fingerprints"`
	// PasskeyAdminReady and PasskeyAndroidFingerprintsEffective only ever come
	// out of the GET response. They are accepted here so a client can send the
	// document back unchanged — the decoder rejects unknown fields, so a view
	// field with no counterpart here turns every save into a 400.
	PasskeyAdminReady                   bool     `json:"passkey_admin_ready,omitempty"`
	PasskeyAndroidFingerprintsEffective []string `json:"passkey_android_fingerprints_effective,omitempty"`
	// Rate limiting uses pointers for the same reason as the other switches:
	// a partial save (the Turnstile form, say) must leave the limits alone.
	RateLimitEnabled            *bool `json:"rate_limit_enabled"`
	RateLimitPerAccount         *int  `json:"rate_limit_per_account"`
	RateLimitPerIP              *int  `json:"rate_limit_per_ip"`
	RateLimitWindowMinutes      *int  `json:"rate_limit_window_minutes"`
	RateLimitFamiliarMultiplier *int  `json:"rate_limit_familiar_multiplier"`
	RateLimitNotify             *bool `json:"rate_limit_notify"`
}

// SaveOAuthRequest is the body of PUT /api/admin/oauth. A blank secret keeps
// the stored one.
type SaveOAuthRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
	Scopes       string `json:"scopes"`
}

// SaveEncryptionKeyRequest is the body of PUT /api/admin/encryption-key.
type SaveEncryptionKeyRequest struct {
	Key string `json:"key"`
}

// SaveSMTPRequest is the body of PUT /api/admin/smtp. An empty password keeps
// the stored one.
type SaveSMTPRequest struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	TLSMode  string `json:"tls_mode"`
}

// SMTPStatusResponse reports the relay state without exposing the password.
type SMTPStatusResponse struct {
	Configured bool   `json:"configured"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	From       string `json:"from"`
	TLSMode    string `json:"tls_mode"`
}

// SMTPTestRequest is the body of POST /api/admin/smtp/test.
type SMTPTestRequest struct {
	To string `json:"to"`
}
