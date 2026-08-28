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
