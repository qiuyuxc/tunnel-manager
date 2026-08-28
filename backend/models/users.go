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
}

// UserView is the API-safe projection of a user.
type UserView struct {
	ID            string   `json:"id"`
	Username      string   `json:"username"`
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
}

// SessionUser is the authenticated identity attached to requests.
type SessionUser struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
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
	Username   string `json:"username"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	Invite     string `json:"invite"`
	VerifyCode string `json:"verify_code"`
}

// AuthConfigResponse tells the frontend how to render the register form.
type AuthConfigResponse struct {
	RegistrationEnabled bool   `json:"registration_enabled"`
	InviteMode          string `json:"invite_mode"`
	EmailVerifyEnabled  bool   `json:"email_verify_enabled"`
}

// SendCodeRequest is the body of POST /api/auth/send-code.
type SendCodeRequest struct {
	Email string `json:"email"`
}

// MeResponse describes the authenticated user for the frontend.
type MeResponse struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
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
