// Session identity and global settings APIs.
// Single-user panel: registration, user/group/invite management and the audit
// log were removed. The remaining endpoints under /admin/* are the panel's own
// session/auth and the global system settings.
import { api } from './index'

export interface AuthConfig {
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

export function getMe() {
  return api.get<MeResponse>('/auth/me')
}

export interface AppSettings {
  turnstile_enabled?: boolean
  turnstile_site_key?: string
  turnstile_has_secret?: boolean
  experimental_features_enabled?: boolean
  /** 面板全局禁用密码登录：必须使用通行密钥 */
  password_login_disabled?: boolean
  /** 通行密钥依赖方 ID；留空按访问域名自动推导 */
  passkey_rp_id?: string
  /** 通行密钥允许的来源，逗号分隔；留空按访问域名自动推导 */
  passkey_origins?: string
  /** 已绑定至少一个通行密钥，才允许开启全局开关 */
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
  /** 触发锁定时是否通知 */
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
