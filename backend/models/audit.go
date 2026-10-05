package models

// Audit trail categories. Each category groups the actions recorded for one
// area of the panel.
const (
	AuditCategoryAuth     = "auth"
	AuditCategoryUser     = "user"
	AuditCategoryGroup    = "group"
	AuditCategoryInvite   = "invite"
	AuditCategorySettings = "settings"
	AuditCategoryTunnel   = "tunnel"
	AuditCategoryDomain   = "domain"
	AuditCategoryDNS      = "dns"
	AuditCategoryMonitor  = "monitor"
	AuditCategoryLab      = "lab"
	AuditCategoryTelegram = "telegram"
	AuditCategoryPasskey  = "passkey"
)

// Audit actions. The values are persisted in the database, so they are never
// renamed; new operations get new keys.
const (
	AuditActionLogin          = "login"
	AuditActionLoginFailed    = "login_failed"
	AuditActionLoginThrottled = "login_throttled"
	AuditActionLogout         = "logout"

	AuditActionUserCreate   = "user_create"
	AuditActionUserStatus   = "user_status"
	AuditActionUserGroup    = "user_group"
	AuditActionUserPassword = "user_password"
	AuditActionUserDelete   = "user_delete"

	AuditActionGroupCreate = "group_create"
	AuditActionGroupUpdate = "group_update"
	AuditActionGroupDelete = "group_delete"

	AuditActionInviteCreate = "invite_create"
	AuditActionInviteUpdate = "invite_update"
	AuditActionInviteDelete = "invite_delete"

	AuditActionSettingsUpdate      = "settings_update"
	AuditActionSiteUpdate          = "site_update"
	AuditActionCNAMEPresetsUpdate  = "cname_presets_update"
	AuditActionPreferredCNAME      = "preferred_cname_update"
	AuditActionSMTPUpdate          = "smtp_update"
	AuditActionSMTPTest            = "smtp_test"
	AuditActionOAuthUpdate         = "oauth_update"
	AuditActionEncryptionKeyUpdate = "encryption_key_update"

	AuditActionTunnelCreate  = "tunnel_create"
	AuditActionTunnelDelete  = "tunnel_delete"
	AuditActionIngressAdd    = "ingress_add"
	AuditActionIngressUpdate = "ingress_update"
	AuditActionIngressDelete = "ingress_delete"

	AuditActionDomainBind      = "domain_bind"
	AuditActionDomainBindBatch = "domain_bind_batch"
	AuditActionDomainFallback  = "domain_fallback"

	AuditActionDNSCreate = "dns_create"
	AuditActionDNSUpdate = "dns_update"
	AuditActionDNSDelete = "dns_delete"

	AuditActionMonitorCreate = "monitor_create"
	AuditActionMonitorUpdate = "monitor_update"
	AuditActionMonitorDelete = "monitor_delete"
	AuditActionMonitorCheck  = "monitor_check"
	AuditActionTargetAdd     = "target_add"
	AuditActionTargetUpdate  = "target_update"
	AuditActionTargetDelete  = "target_delete"

	AuditActionLabSettingsUpdate = "lab_settings_update"
	AuditActionLabRun            = "lab_run"

	AuditActionTelegramEndpointUpdate = "telegram_endpoint_update"

	AuditActionPasskeyAdd             = "passkey_add"
	AuditActionPasskeyRemove          = "passkey_remove"
	AuditActionPasskeyRename          = "passkey_rename"
	AuditActionPasskeyLogin           = "passkey_login"
	AuditActionPasskeyLoginFailed     = "passkey_login_failed"
	AuditActionPasswordLoginDisable   = "password_login_disable"
	AuditActionPasswordLoginEnable    = "password_login_enable"
	AuditActionPasswordLoginGlobalOff = "password_login_global_disable"
	AuditActionPasswordLoginGlobalOn  = "password_login_global_enable"
)

// AuditLog is one recorded operation. ActorName is stored alongside the actor
// id so the trail stays readable after an account is renamed or deleted.
type AuditLog struct {
	ID        int64  `json:"id"`
	CreatedAt int64  `json:"created_at"`
	ActorID   string `json:"actor_id"`
	ActorName string `json:"actor_name"`
	Category  string `json:"category"`
	Action    string `json:"action"`
	Target    string `json:"target"`
	IP        string `json:"ip"`
	Success   bool   `json:"success"`
}

// AuditQuery filters and paginates the audit trail. Zero values mean "no
// filter"; Page and PageSize are normalized by the store.
type AuditQuery struct {
	Actor    string
	Category string
	Action   string
	From     int64
	To       int64
	Page     int
	PageSize int
}

// AuditPage is one page of the audit trail, newest entry first.
type AuditPage struct {
	Logs     []AuditLog `json:"logs"`
	Total    int        `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

// AuditStats summarizes the recent audit trail for the admin overview cards.
type AuditStats struct {
	Total  int `json:"total"`
	Failed int `json:"failed"`
	Today  int `json:"today"`
	Actors int `json:"actors"`
}

// DefaultAuditRetentionDays is the retention window used when the setting is
// left at its zero value.
const DefaultAuditRetentionDays = 90

// EffectiveAuditRetentionDays resolves the configured retention window: zero
// means the default, a negative value keeps entries forever.
func (s AppSettings) EffectiveAuditRetentionDays() int {
	if s.AuditRetentionDays == 0 {
		return DefaultAuditRetentionDays
	}
	return s.AuditRetentionDays
}
