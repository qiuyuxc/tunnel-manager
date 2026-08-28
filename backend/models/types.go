package models

import (
	"encoding/json"
	"time"
)

// Config represents the application configuration state
type Config struct {
	TunnelID               string        `json:"tunnel_id"`
	TunnelName             string        `json:"tunnel_name"`
	ServiceURL             string        `json:"service_url"`
	PreferredCNAME         string        `json:"preferred_cname"`
	CNAMEPresets           []CNAMEPreset `json:"cname_presets,omitempty"`
	SiteName               string        `json:"site_name"`
	SiteDescription        string        `json:"site_description"`
	SiteIcon               string        `json:"site_icon"`
	LandingEnabled         bool          `json:"landing_enabled"`
	AdminUsername          string        `json:"admin_username"`
	AdminPasswordHash      string        `json:"admin_password_hash"`
	TOTPEnabled            bool          `json:"totp_enabled,omitempty"`
	TOTPSecretEncrypted    string        `json:"totp_secret_encrypted,omitempty"`
	TOTPRecoveryCodeHashes []string      `json:"totp_recovery_code_hashes,omitempty"`
	TOTPLastAcceptedStep   int64         `json:"totp_last_accepted_step,omitempty"`
	CFAccountID            string        `json:"cf_account_id,omitempty"`
	CFAccountName          string        `json:"cf_account_name,omitempty"`
	CFOAuthAccessToken     string        `json:"cf_oauth_access_token,omitempty"`
	CFOAuthRefreshToken    string        `json:"cf_oauth_refresh_token,omitempty"`
	CFOAuthExpiresAt       int64         `json:"cf_oauth_expires_at,omitempty"`
	CFOAuthScope           string        `json:"cf_oauth_scope,omitempty"`
	// Telegram bot settings
	TGBotEnabled    bool   `json:"tg_bot_enabled"`
	TGBotToken      string `json:"tg_bot_token"`
	TGAdminIDs      string `json:"tg_admin_ids"`
	TGMode          string `json:"tg_mode"`
	TGWebhookURL    string `json:"tg_webhook_url"`
	TGWebhookSecret string `json:"tg_webhook_secret"`
	TGApiEndpoint   string `json:"tg_api_endpoint"`

	// Selected zone used by Telegram DNS commands
	SelectedZoneID   string `json:"selected_zone_id,omitempty"`
	SelectedZoneName string `json:"selected_zone_name,omitempty"`

	// Service monitor projects (uptime-style)
	Monitors []Monitor `json:"monitors,omitempty"`
}

// MonitorTarget is one probed endpoint inside a monitor project.
type MonitorTarget struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
	// Type picks the probe method: http (default), tcp or icmp.
	Type        string `json:"type,omitempty"`
	Method      string `json:"method,omitempty"` // http only: GET (default) or POST
	CreatedAt   int64  `json:"created_at,omitempty"`
	LinkEnabled bool   `json:"link_enabled,omitempty"`
	LastState   string `json:"last_state,omitempty"`
}

// Monitor groups targets checked on a schedule and publishable on a
// token-protected public status page.
type Monitor struct {
	ID             string          `json:"id"`
	UserID         string          `json:"user_id,omitempty"`
	Name           string          `json:"name"`
	IntervalSec    int             `json:"interval_sec,omitempty"` // default 60
	PublicToken    string          `json:"public_token,omitempty"`
	PublicSlug     string          `json:"public_slug,omitempty"`
	PublishEnabled bool            `json:"publish_enabled"`
	AlertEnabled   bool            `json:"alert_enabled,omitempty"`
	AlertEmails    string          `json:"alert_emails,omitempty"`
	Targets        []MonitorTarget `json:"targets,omitempty"`
	PublicTitle    string          `json:"public_title,omitempty"` // status page display title
	PublicIcon     string          `json:"public_icon,omitempty"`
	PublicTheme    string          `json:"public_theme,omitempty"`
	Announcement   string          `json:"announcement,omitempty"` // banner text on the status page
	CreatedAt      int64           `json:"created_at,omitempty"`
}

// CNAMEPreset is a reusable preferred CNAME option.
type CNAMEPreset struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SiteSettings contains public-facing site branding.
type SiteSettings struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Icon           string `json:"icon"`
	LandingEnabled bool   `json:"landing_enabled"`
}

// Tunnel represents a Cloudflare Tunnel
type Tunnel struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// CreateTunnelRequest names a new tunnel.
type CreateTunnelRequest struct {
	Name string `json:"name"`
}

// CreateTunnelResponse describes a new tunnel and how to attach cloudflared to it.
type CreateTunnelResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Token      string `json:"token,omitempty"`
	RunCommand string `json:"run_command,omitempty"`
	Warning    string `json:"warning,omitempty"`
}

// DeleteIngressRequest removes a tunnel route and optionally its DNS record.
type DeleteIngressRequest struct {
	Hostname  string `json:"hostname"`
	DeleteDNS bool   `json:"delete_dns"`
}

// Zone represents a Cloudflare Zone
type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Account represents a Cloudflare account available to the current credential.
type Account struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CloudflareOAuthStatus describes the configured Cloudflare credential source.
type CloudflareOAuthStatus struct {
	Configured  bool      `json:"configured"`
	Connected   bool      `json:"connected"`
	Source      string    `json:"source"`
	AccountID   string    `json:"account_id"`
	AccountName string    `json:"account_name"`
	Accounts    []Account `json:"accounts"`
	ExpiresAt   string    `json:"expires_at,omitempty"`
	RedirectURI string    `json:"redirect_uri"`
	Error       string    `json:"error,omitempty"`
	// Multi-account extension (v1.17+).
	Connections        []CFConnectionView `json:"connections"`
	ActiveConnectionID string             `json:"active_connection_id"`
}

// CloudflareOAuthStartResponse contains the URL for the authorization redirect.
type CloudflareOAuthStartResponse struct {
	AuthorizationURL string `json:"authorization_url"`
}

// CloudflareAccountRequest selects an account authorized through OAuth.
type CloudflareAccountRequest struct {
	AccountID string `json:"account_id"`
}

// DNSRecord represents a DNS record
type DNSRecord struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	Proxied   bool   `json:"proxied"`
	Proxiable bool   `json:"proxiable,omitempty"`
	TTL       int    `json:"ttl,omitempty"`
	Priority  int    `json:"priority,omitempty"`
}

// DNSRecordRequest contains the editable fields for a DNS record.
type DNSRecordRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	Proxied  bool   `json:"proxied"`
	TTL      int    `json:"ttl"`
	Priority *int   `json:"priority,omitempty"`
}

// TunnelConfigResponse represents the CF API response for tunnel config
type TunnelConfigResponse struct {
	Success bool `json:"success"`
	Result  struct {
		Config struct {
			Ingress []IngressRule `json:"ingress"`
		} `json:"config"`
	} `json:"result"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// IngressRule represents a tunnel ingress rule
type IngressRule struct {
	Hostname      string                 `json:"hostname,omitempty"`
	Path          string                 `json:"path,omitempty"`
	Service       string                 `json:"service"`
	OriginRequest map[string]interface{} `json:"originRequest,omitempty"`
}

// CFAPIResponse is a generic Cloudflare API response wrapper
type CFAPIResponse struct {
	Success  bool            `json:"success"`
	Result   json.RawMessage `json:"result"`
	Errors   []CFError       `json:"errors"`
	Messages json.RawMessage `json:"messages,omitempty"`
}

// CFError represents a Cloudflare API error
type CFError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// BindRequest is the request body for domain binding
type BindRequest struct {
	Mode           string `json:"mode,omitempty"`
	PreferredCNAME string `json:"preferred_cname"`
	MainDomain     string `json:"main_domain"`
	AuxDomain      string `json:"aux_domain"`
}

// BatchBindItem is a domain binding group with its own origin service.
type BatchBindItem struct {
	Mode           string `json:"mode,omitempty"`
	ServiceURL     string `json:"service_url"`
	PreferredCNAME string `json:"preferred_cname"`
	MainDomain     string `json:"main_domain"`
	AuxDomain      string `json:"aux_domain"`
}

// BatchBindRequest is the request body for batch domain binding.
type BatchBindRequest struct {
	Items []BatchBindItem `json:"items"`
}

// BatchBindResult records the result for a single domain group.
type BatchBindResult struct {
	Mode           string `json:"mode"`
	ServiceURL     string `json:"service_url"`
	PreferredCNAME string `json:"preferred_cname"`
	MainDomain     string `json:"main_domain"`
	AuxDomain      string `json:"aux_domain"`
	Success        bool   `json:"success"`
	Message        string `json:"message"`
}

// BatchBindResponse is the response body for batch domain binding.
type BatchBindResponse struct {
	Results []BatchBindResult `json:"results"`
}

// FallbackRequest is the request body for setting fallback origin
type FallbackRequest struct {
	Domain string `json:"domain"`
}

// CustomHostname represents a Cloudflare SaaS custom hostname
type CustomHostname struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
}

// SetValueRequest is a generic request for setting a single value
type SetValueRequest struct {
	Value string `json:"value"`
}

// SetTunnelRequest selects a tunnel while preserving its display name.
type SetTunnelRequest struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

// SetSiteSettingsRequest updates public-facing site branding.
type SetSiteSettingsRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Icon           string `json:"icon"`
	LandingEnabled bool   `json:"landing_enabled"`
}

// SetCNAMEPresetsRequest replaces the reusable CNAME options.
type SetCNAMEPresetsRequest struct {
	Items []CNAMEPreset `json:"items"`
}

// LoginRequest is the request body for admin login
type LoginRequest struct {
	Account           string `json:"account,omitempty"`
	Username          string `json:"username"`
	Password          string `json:"password"`
	TurnstileResponse string `json:"cf_turnstile_response,omitempty"`
}

// LoginResponse is the response body for admin login
type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Role     string `json:"role,omitempty"`
}

// TwoFactorChallengeResponse requests a second authentication factor.
type TwoFactorChallengeResponse struct {
	TwoFactorRequired bool      `json:"two_factor_required"`
	ChallengeToken    string    `json:"challenge_token"`
	ExpiresAt         time.Time `json:"expires_at"`
}

// TwoFactorLoginRequest completes a challenged login.
type TwoFactorLoginRequest struct {
	ChallengeToken string `json:"challenge_token"`
	Code           string `json:"code"`
}

// TOTPSetupResponse contains a pending authenticator setup.
type TOTPSetupResponse struct {
	SetupToken string    `json:"setup_token"`
	Secret     string    `json:"secret"`
	OTPAuthURI string    `json:"otpauth_uri"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// TOTPConfirmRequest confirms a pending authenticator setup.
type TOTPConfirmRequest struct {
	SetupToken string `json:"setup_token"`
	Code       string `json:"code"`
}

// TOTPConfirmResponse returns one-time recovery codes.
type TOTPConfirmResponse struct {
	Enabled       bool     `json:"enabled"`
	RecoveryCodes []string `json:"recovery_codes"`
}

// TOTPStatusResponse reports two-factor authentication status.
type TOTPStatusResponse struct {
	Enabled                bool `json:"enabled"`
	RecoveryCodesRemaining int  `json:"recovery_codes_remaining"`
	SetupAvailable         bool `json:"setup_available"`
}

// TOTPDisableRequest authorizes disabling two-factor authentication.
type TOTPDisableRequest struct {
	CurrentPassword string `json:"current_password"`
	Code            string `json:"code"`
}

// TOTPDisableResponse reports the disabled state.
type TOTPDisableResponse struct {
	Enabled bool `json:"enabled"`
}

// ChangePasswordRequest is the request body for changing password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangeUsernameRequest is the request body for changing username
type ChangeUsernameRequest struct {
	CurrentPassword string `json:"current_password"`
	NewUsername     string `json:"new_username"`
}

// ChangeEmailRequest binds or replaces the account email. Password
// confirmation is required.
type ChangeEmailRequest struct {
	CurrentPassword string `json:"current_password"`
	NewEmail        string `json:"new_email"`
}

// ChangeProfileRequest updates the account display name and avatar URL.
// Both fields are optional; an empty avatar clears the custom avatar.
type ChangeProfileRequest struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// TelegramSettingsRequest is the request body for saving bot settings
type TelegramSettingsRequest struct {
	Enabled     bool   `json:"enabled"`
	BotToken    string `json:"bot_token"`
	AdminTGIDs  string `json:"admin_tg_ids"`
	Mode        string `json:"mode"`
	WebhookURL  string `json:"webhook_url"`
	ApiEndpoint string `json:"api_endpoint"`
}

// TelegramSettingsResponse is the response body for getting bot settings
type TelegramSettingsResponse struct {
	Enabled      bool   `json:"enabled"`
	BotTokenSet  bool   `json:"bot_token_set"`
	BotTokenHint string `json:"bot_token_hint"`
	AdminTGIDs   string `json:"admin_tg_ids"`
	Mode         string `json:"mode"`
	WebhookURL   string `json:"webhook_url"`
	ApiEndpoint  string `json:"api_endpoint"`
	NotifyBotSet bool   `json:"notify_bot_set"`
}

// TelegramStatusResponse is the response body for bot status
type TelegramStatusResponse struct {
	Enabled      bool   `json:"enabled"`
	Running      bool   `json:"running"`
	Mode         string `json:"mode"`
	BotUsername  string `json:"bot_username"`
	LastError    string `json:"last_error"`
	LastUpdateAt string `json:"last_update_at"`
}
