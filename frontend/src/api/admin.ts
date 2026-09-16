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

export const PERMISSION_LABELS: Record<string, string> = {
  tunnels: '隧道管理',
  domain_bind: '域名绑定',
  dns: 'DNS 记录',
  monitors: '服务监控',
  oauth_connect: 'Cloudflare 授权',
}

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

export const AUDIT_CATEGORY_LABELS: Record<string, string> = {
  auth: '登录认证',
  user: '用户管理',
  group: '用户组',
  invite: '邀请码',
  settings: '系统设置',
  tunnel: '隧道管理',
  domain: '域名绑定',
  dns: 'DNS 记录',
  monitor: '服务监控',
  lab: 'IP 优选实验室',
  telegram: 'TG 机器人',
  passkey: '通行密钥',
}

/** 操作类型选项，按分类分组，供审计日志筛选使用。 */
export const AUDIT_ACTIONS: { category: string; action: string; label: string }[] = [
  { category: 'auth', action: 'login', label: '登录成功' },
  { category: 'auth', action: 'login_failed', label: '登录失败' },
  { category: 'auth', action: 'logout', label: '退出登录' },
  { category: 'user', action: 'user_create', label: '创建用户' },
  { category: 'user', action: 'user_status', label: '启用 / 禁用用户' },
  { category: 'user', action: 'user_group', label: '调整用户组' },
  { category: 'user', action: 'user_password', label: '重置用户密码' },
  { category: 'user', action: 'user_delete', label: '删除用户' },
  { category: 'group', action: 'group_create', label: '创建用户组' },
  { category: 'group', action: 'group_update', label: '修改用户组' },
  { category: 'group', action: 'group_delete', label: '删除用户组' },
  { category: 'invite', action: 'invite_create', label: '生成邀请码' },
  { category: 'invite', action: 'invite_update', label: '启用 / 停用邀请码' },
  { category: 'invite', action: 'invite_delete', label: '删除邀请码' },
  { category: 'settings', action: 'settings_update', label: '修改注册策略' },
  { category: 'settings', action: 'site_update', label: '修改站点设置' },
  { category: 'settings', action: 'cname_presets_update', label: '修改 CNAME 预设' },
  { category: 'settings', action: 'preferred_cname_update', label: '修改优选 CNAME' },
  { category: 'settings', action: 'smtp_update', label: '修改邮件服务' },
  { category: 'settings', action: 'smtp_test', label: '发送测试邮件' },
  { category: 'settings', action: 'oauth_update', label: '修改 OAuth 客户端' },
  { category: 'settings', action: 'encryption_key_update', label: '修改加密密钥' },
  { category: 'tunnel', action: 'tunnel_create', label: '创建隧道' },
  { category: 'tunnel', action: 'tunnel_delete', label: '删除隧道' },
  { category: 'tunnel', action: 'ingress_add', label: '新增隧道规则' },
  { category: 'tunnel', action: 'ingress_update', label: '修改隧道规则' },
  { category: 'tunnel', action: 'ingress_delete', label: '删除隧道规则' },
  { category: 'domain', action: 'domain_bind', label: '绑定域名' },
  { category: 'domain', action: 'domain_bind_batch', label: '批量绑定域名' },
  { category: 'domain', action: 'domain_fallback', label: '设置回退源' },
  { category: 'dns', action: 'dns_create', label: '新增 DNS 记录' },
  { category: 'dns', action: 'dns_update', label: '修改 DNS 记录' },
  { category: 'dns', action: 'dns_delete', label: '删除 DNS 记录' },
  { category: 'monitor', action: 'monitor_create', label: '创建监控项目' },
  { category: 'monitor', action: 'monitor_update', label: '修改监控项目' },
  { category: 'monitor', action: 'monitor_delete', label: '删除监控项目' },
  { category: 'monitor', action: 'monitor_check', label: '手动检测监控' },
  { category: 'monitor', action: 'target_add', label: '新增监控目标' },
  { category: 'monitor', action: 'target_update', label: '修改监控目标' },
  { category: 'monitor', action: 'target_delete', label: '删除监控目标' },
  { category: 'lab', action: 'lab_settings_update', label: '修改实验室设置' },
  { category: 'lab', action: 'lab_run', label: '执行 IP 优选' },
  { category: 'telegram', action: 'telegram_endpoint_update', label: '修改 TG API 端点' },
  { category: 'passkey', action: 'passkey_add', label: '绑定通行密钥' },
  { category: 'passkey', action: 'passkey_remove', label: '删除通行密钥' },
  { category: 'passkey', action: 'passkey_rename', label: '重命名通行密钥' },
  { category: 'passkey', action: 'passkey_login', label: '通行密钥登录' },
  { category: 'passkey', action: 'passkey_login_failed', label: '通行密钥登录失败' },
  { category: 'passkey', action: 'password_login_disable', label: '禁用密码登录' },
  { category: 'passkey', action: 'password_login_enable', label: '恢复密码登录' },
  { category: 'passkey', action: 'password_login_global_disable', label: '全局禁用密码登录' },
  { category: 'passkey', action: 'password_login_global_enable', label: '全局恢复密码登录' },
]

const AUDIT_ACTION_LABELS: Record<string, string> = Object.fromEntries(AUDIT_ACTIONS.map((item) => [item.action, item.label]))

/** 操作类型的中文名；未知操作回退到分类名 + 原始键值。 */
export function auditActionLabel(log: AuditLog) {
  const label = AUDIT_ACTION_LABELS[log.action]
  if (label) return label
  const category = AUDIT_CATEGORY_LABELS[log.category] || log.category
  return log.action ? category + ' · ' + log.action : category
}

export const listAuditLogs = (params: AuditQuery) => api.get<AuditPage>('/admin/audit-logs', { params })
export const getAuditStats = (days = 7) => api.get<AuditStats>('/admin/audit-logs/stats', { params: { days } })

