import { t as tr, localized } from '../i18n.ts'
// Registration, session identity and admin backend APIs.
import { api } from './index'
import type { LoginResponse } from './index'

export interface AuthConfig {
  registration_enabled: boolean
  invite_mode: 'off' | 'optional' | 'required'
  email_verify_enabled: boolean
  turnstile_enabled: boolean
  turnstile_site_key: string
}

export interface MeResponse {
  id: string
  username: string
  nickname: string
  avatar: string
  email: string
  role: string
  permissions: string[]
}

export function getAuthConfig() {
  return api.get<AuthConfig>('/auth/config')
}

export interface RegisterPayload {
  username: string
  email: string
  password: string
  invite?: string
  verify_code?: string
  cf_turnstile_response?: string
}

export function register(payload: RegisterPayload) {
  return api.post<LoginResponse>('/auth/register', payload)
}

export function sendRegisterCode(email: string, turnstileToken = '') {
  return api.post<{ message: string }>('/auth/send-code', { email, cf_turnstile_response: turnstileToken })
}

export function getMe() {
  return api.get<MeResponse>('/auth/me')
}

export interface UserView {
  id: string
  username: string
  nickname: string
  avatar: string
  email: string
  role: string
  group_id: string
  group_name?: string
  status: string
  email_verified: boolean
  totp_enabled: boolean
  created_at: number
  last_login_at: number
  permissions?: string[]
  /** 已绑定的通行密钥数量 */
  passkeys: number
  /** 该账户已禁用密码登录 */
  password_login_disabled: boolean
  has_password?: boolean
  oidc_identities?: number
}

export interface UserGroup {
  id: string
  name: string
  permissions: string[]
  builtin: boolean
  created_at: number
}

export interface Invite {
  code: string
  group_id: string
  max_uses: number
  used_count: number
  expires_at: number
  enabled: boolean
  created_at: number
}

export interface AppSettings {
  registration_enabled: boolean
  invite_mode: string
  default_group_id?: string
  email_verify_disabled?: boolean
  turnstile_enabled?: boolean
  turnstile_site_key?: string
  turnstile_has_secret?: boolean
  experimental_features_enabled?: boolean
  lab_daily_request_limit?: number
  lab_requests_per_second?: number
  lab_max_workers?: number
  /** 审计日志保留天数；负数表示永久保留，0 表示使用默认值（90 天）。 */
  audit_retention_days?: number
  /** 面板全局禁用密码登录：所有账户都必须使用通行密钥 */
  password_login_disabled?: boolean
  /** 通行密钥依赖方 ID；留空按访问域名自动推导 */
  passkey_rp_id?: string
  /** 通行密钥允许的来源，逗号分隔；留空按访问域名自动推导 */
  passkey_origins?: string
  /** 至少一名启用的管理员已绑定通行密钥，才允许开启全局开关 */
  passkey_admin_ready?: boolean
  /** 登录限流总开关 */
  rate_limit_enabled?: boolean
  /** 每账号失败次数上限；0 = 默认 5 次，负数 = 不限 */
  rate_limit_per_account?: number
  /** 每 IP 失败次数上限；0 = 默认 20 次，负数 = 不限 */
  rate_limit_per_ip?: number
  /** 统计窗口（分钟）；0 = 默认 15 分钟 */
  rate_limit_window_minutes?: number
  /** 熟悉来源的额度倍数；0 = 默认 3 倍，1 = 不放宽 */
  rate_limit_familiar_multiplier?: number
  /** 触发锁定时是否通知管理员 */
  rate_limit_notify?: boolean
}

export interface OAuthConfig {
  client_id: string
  has_client_secret: boolean
  redirect_uri: string
  scopes: string
}

export interface SMTPStatus {
  configured: boolean
  host: string
  port: number
  username: string
  from: string
  tls_mode: string
}

export const ALL_PERMISSIONS = ['tunnels', 'domain_bind', 'dns', 'monitors', 'oauth_connect'] as const

export const PERMISSION_LABELS: Record<string, string> = localized(() => ({
  tunnels: tr('m_e920f47d0c05'),
  domain_bind: tr('m_4bf8b66d7390'),
  dns: tr('m_72af91239cad'),
  monitors: tr('m_7fab82054a8f'),
  oauth_connect: tr('m_1d81613aa5b5'),
}))

export const listUsers = () => api.get<{ users: UserView[] }>('/admin/users')
export const createUser = (payload: { username: string; email?: string; password: string; role?: string; group_id?: string }) =>
  api.post<UserView>('/admin/users', payload)
export const setUserStatus = (id: string, status: string) => api.put('/admin/users/' + id + '/status', { status })
export const setUserGroup = (id: string, group_id: string) => api.put('/admin/users/' + id + '/group', { group_id })
export const resetUserPassword = (id: string, new_password: string) => api.put('/admin/users/' + id + '/password', { new_password })
export const deleteUser = (id: string) => api.delete('/admin/users/' + id)
export const setUserPasswordLogin = (id: string, disabled: boolean) =>
  api.put('/admin/users/' + id + '/password-login', { disabled })
export const listGroups = () => api.get<{ groups: UserGroup[] }>('/admin/groups')
export const createGroup = (name: string, permissions: string[]) => api.post<UserGroup>('/admin/groups', { name, permissions })
export const updateGroup = (id: string, name: string, permissions: string[]) => api.put('/admin/groups/' + id, { name, permissions })
export const deleteGroup = (id: string) => api.delete('/admin/groups/' + id)
export const listInvites = () => api.get<{ invites: Invite[] }>('/admin/invites')
export const createInvite = (payload: { group_id?: string; max_uses?: number; expires_at?: number }) =>
  api.post<Invite>('/admin/invites', payload)
export const updateInvite = (code: string, enabled: boolean) => api.put('/admin/invites/' + code, { enabled })
export const deleteInvite = (code: string) => api.delete('/admin/invites/' + code)
export const getAppSettings = () => api.get<AppSettings>('/admin/settings')
export const updateAppSettings = (payload: Partial<AppSettings> & { turnstile_secret?: string }) =>
  api.put<AppSettings>('/admin/settings', payload)
export const getOAuthConfig = () => api.get<OAuthConfig>('/admin/oauth')
export const updateOAuthConfig = (payload: { client_id: string; client_secret?: string; redirect_uri: string; scopes: string }) =>
  api.put<OAuthConfig>('/admin/oauth', payload)
export const getEncryptionKeyStatus = () => api.get<{ source: string }>('/admin/encryption-key')
export const saveEncryptionKey = (key: string) => api.put<{ message: string }>('/admin/encryption-key', { key })
export function forgotPassword(email: string, turnstileToken = '') {
  return api.post<{ message: string }>('/auth/forgot-password', { email, cf_turnstile_response: turnstileToken })
}
export function resetPassword(email: string, code: string, newPassword: string, turnstileToken = '') {
  return api.post<{ message: string }>('/auth/reset-password', { email, code, new_password: newPassword, cf_turnstile_response: turnstileToken })
}
export const getSMTP = () => api.get<SMTPStatus>('/admin/smtp')
export const updateSMTP = (payload: { host: string; port: number; username: string; password?: string; from: string; tls_mode: string }) =>
  api.put<SMTPStatus>('/admin/smtp', payload)
export const testSMTP = (to: string) => api.post<{ message: string }>('/admin/smtp/test', { to })

// ---------------------------------------------------------------------------
// 审计日志

export interface AuditLog {
  id: number
  created_at: number
  actor_id: string
  actor_name: string
  category: string
  action: string
  target: string
  ip: string
  success: boolean
}

export interface AuditPage {
  logs: AuditLog[]
  total: number
  page: number
  page_size: number
}

export interface AuditStats {
  total: number
  failed: number
  today: number
  actors: number
}

export interface AuditQuery {
  actor?: string
  category?: string
  action?: string
  from?: number
  to?: number
  page?: number
  page_size?: number
}

export const AUDIT_CATEGORY_LABELS: Record<string, string> = localized(() => ({
  auth: tr('m_bbf58983ccf5'),
  user: tr('m_fbf413d429bd'),
  group: tr('m_963ead08d78d'),
  invite: tr('m_c8b56dca7aed'),
  settings: tr('m_68ea5dd4d7af'),
  tunnel: tr('m_e920f47d0c05'),
  domain: tr('m_4bf8b66d7390'),
  dns: tr('m_72af91239cad'),
  monitor: tr('m_7fab82054a8f'),
  lab: tr('m_3db29474a184'),
  telegram: tr('m_805dcced3f07'),
  passkey: tr('m_102824e51a9d'),
}))

/** 操作类型选项，按分类分组，供审计日志筛选使用。 */
export const AUDIT_ACTIONS: { category: string; action: string; label: string }[] = localized(() => ([
  { category: 'auth', action: 'login', label: tr('m_645b934deb86') },
  { category: 'auth', action: 'login_failed', label: tr('m_e4343921c928') },
  { category: 'auth', action: 'logout', label: tr('m_3ab8cc15939f') },
  { category: 'user', action: 'user_create', label: tr('m_37c825ff40e7') },
  { category: 'user', action: 'user_status', label: tr('m_088829f635e5') },
  { category: 'user', action: 'user_group', label: tr('m_3f18c306c522') },
  { category: 'user', action: 'user_password', label: tr('m_69d844482cde') },
  { category: 'user', action: 'user_delete', label: tr('m_b75872bba6f1') },
  { category: 'group', action: 'group_create', label: tr('m_f5c0c8360a8a') },
  { category: 'group', action: 'group_update', label: tr('m_880dc01d1427') },
  { category: 'group', action: 'group_delete', label: tr('m_2aacf3a5dbf0') },
  { category: 'invite', action: 'invite_create', label: tr('m_5f6a55e484cc') },
  { category: 'invite', action: 'invite_update', label: tr('m_0f69f3cff13d') },
  { category: 'invite', action: 'invite_delete', label: tr('m_6f2555f607af') },
  { category: 'settings', action: 'settings_update', label: tr('m_e04f5a9c92cb') },
  { category: 'settings', action: 'site_update', label: tr('m_9c4e82ea3a5e') },
  { category: 'settings', action: 'cname_presets_update', label: tr('m_e3a19ed99df8') },
  { category: 'settings', action: 'preferred_cname_update', label: tr('m_33a47993a6eb') },
  { category: 'settings', action: 'smtp_update', label: tr('m_b6d496d548f8') },
  { category: 'settings', action: 'smtp_test', label: tr('m_f9ae1a31638d') },
  { category: 'settings', action: 'oauth_update', label: tr('m_30ebe9296b01') },
  { category: 'settings', action: 'encryption_key_update', label: tr('m_026bad67a38d') },
  { category: 'tunnel', action: 'tunnel_create', label: tr('m_e33080d615cb') },
  { category: 'tunnel', action: 'tunnel_delete', label: tr('m_cf5e530b1ad3') },
  { category: 'tunnel', action: 'ingress_add', label: tr('m_b6c70706eb7d') },
  { category: 'tunnel', action: 'ingress_update', label: tr('m_9c66533b5540') },
  { category: 'tunnel', action: 'ingress_delete', label: tr('m_7d67e8e89593') },
  { category: 'domain', action: 'domain_bind', label: tr('m_dd799f925f94') },
  { category: 'domain', action: 'domain_bind_batch', label: tr('m_d7220322e884') },
  { category: 'domain', action: 'domain_fallback', label: tr('m_0bb58707f90c') },
  { category: 'dns', action: 'dns_create', label: tr('m_d9099a5e05d2') },
  { category: 'dns', action: 'dns_update', label: tr('m_ce46f070601d') },
  { category: 'dns', action: 'dns_delete', label: tr('m_ab190f1faa64') },
  { category: 'monitor', action: 'monitor_create', label: tr('m_f9313095ee64') },
  { category: 'monitor', action: 'monitor_update', label: tr('m_40b947bb3c5e') },
  { category: 'monitor', action: 'monitor_delete', label: tr('m_b2fc749edfa3') },
  { category: 'monitor', action: 'monitor_check', label: tr('m_579e2b6a5204') },
  { category: 'monitor', action: 'target_add', label: tr('m_b8c4303aff45') },
  { category: 'monitor', action: 'target_update', label: tr('m_c21bd0ef300e') },
  { category: 'monitor', action: 'target_delete', label: tr('m_bf7241072328') },
  { category: 'lab', action: 'lab_settings_update', label: tr('m_247f5a5d0147') },
  { category: 'lab', action: 'lab_run', label: tr('m_15cce67fe3ee') },
  { category: 'lab', action: 'lab_stop', label: tr('m_5dc85ec6744a') },
  { category: 'lab', action: 'lab_ownership_challenge', label: tr('m_b824edacb9d7') },
  { category: 'lab', action: 'lab_ownership_verify', label: tr('m_9ee6089139dd') },
  { category: 'lab', action: 'lab_ownership_dns', label: tr('m_eb83a1ede9ac') },
  { category: 'telegram', action: 'telegram_endpoint_update', label: tr('m_68dc91ef78e7') },
  { category: 'passkey', action: 'passkey_add', label: tr('m_54255856f81f') },
  { category: 'passkey', action: 'passkey_remove', label: tr('m_fc3f78b499e1') },
  { category: 'passkey', action: 'passkey_rename', label: tr('m_c384f03c2393') },
  { category: 'passkey', action: 'passkey_login', label: tr('m_5903fbc149e8') },
  { category: 'passkey', action: 'passkey_login_failed', label: tr('m_9b4340ac3382') },
  { category: 'passkey', action: 'password_login_disable', label: tr('m_3a027241650b') },
  { category: 'passkey', action: 'password_login_enable', label: tr('m_466c8603a96a') },
  { category: 'passkey', action: 'password_login_global_disable', label: tr('m_aa2fb06da444') },
  { category: 'passkey', action: 'password_login_global_enable', label: tr('m_7022fc61d84f') },
]))

const AUDIT_ACTION_LABELS: Record<string, string> = localized(() => (Object.fromEntries(AUDIT_ACTIONS.map((item) => [item.action, item.label]))))

/** 操作类型的中文名；未知操作回退到分类名 + 原始键值。 */
export function auditActionLabel(log: AuditLog) {
  const label = AUDIT_ACTION_LABELS[log.action]
  if (label) return label
  const category = AUDIT_CATEGORY_LABELS[log.category] || log.category
  return log.action ? category + ' · ' + log.action : category
}

export const listAuditLogs = (params: AuditQuery) => api.get<AuditPage>('/admin/audit-logs', { params })
export const getAuditStats = (days = 7) => api.get<AuditStats>('/admin/audit-logs/stats', { params: { days } })
