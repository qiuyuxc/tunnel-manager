import axios, { type InternalAxiosRequestConfig } from 'axios'

declare module 'axios' {
  interface AxiosRequestConfig {
    skipAuthInvalidation?: boolean
  }

  interface InternalAxiosRequestConfig {
    skipAuthInvalidation?: boolean
  }
}

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

// Attach auth token to all requests
api.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = localStorage.getItem('auth_token')
  if (token) {
    config.headers['X-Auth-Token'] = token
  }
  return config
})

// Auto-logout on 401 (e.g. backend restarted, session lost)
api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401 && !err.config?.skipAuthInvalidation && window.location.pathname !== '/login') {
      localStorage.removeItem('auth_token')
      localStorage.removeItem('auth_username')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export interface Tunnel {
  id: string
  name: string
  status: string
}

export interface IngressRule {
  hostname?: string
  service: string
}

export interface TunnelDetail {
  id: string
  name: string
  status: string
  ingress: IngressRule[]
}

export interface Zone {
  id: string
  name: string
}

export interface Config {
  tunnel_id: string
  service_url: string
  preferred_cname: string
}

export interface BindRequest {
  main_domain: string
  aux_domain: string
}

export interface BatchBindItem {
  service_url: string
  preferred_cname: string
  main_domain: string
  aux_domain: string
}

export interface BatchBindResult extends BatchBindItem {
  success: boolean
  message: string
}

export interface BatchBindResponse {
  results: BatchBindResult[]
}

export interface ApiResponse {
  status: string
  error?: string
  message?: string
}

// Auth
export interface LoginResponse {
  token: string
  username: string
}

export interface TwoFactorChallengeResponse {
  two_factor_required: true
  challenge_token: string
  expires_at: string
}

export type LoginResult = LoginResponse | TwoFactorChallengeResponse

export interface TOTPStatusResponse {
  enabled: boolean
  recovery_codes_remaining: number
  setup_available: boolean
}

export interface TOTPSetupResponse {
  setup_token: string
  secret: string
  otpauth_uri: string
  expires_at: string
}

export interface TOTPConfirmResponse {
  enabled: boolean
  recovery_codes: string[]
}

export function login(username: string, password: string) {
  return api.post<LoginResult>('/admin/login', { username, password })
}

export function completeTwoFactorLogin(challengeToken: string, code: string) {
  return api.post<LoginResponse>('/admin/login/2fa', { challenge_token: challengeToken, code })
}

export function getTwoFactorStatus() {
  return api.get<TOTPStatusResponse>('/admin/2fa/status')
}

export function setupTwoFactor() {
  return api.post<TOTPSetupResponse>('/admin/2fa/setup', {})
}

export function confirmTwoFactor(setupToken: string, code: string) {
  return api.post<TOTPConfirmResponse>('/admin/2fa/confirm', { setup_token: setupToken, code }, { skipAuthInvalidation: true })
}

export function disableTwoFactor(currentPassword: string, code: string) {
  return api.post<{ enabled: false }>('/admin/2fa/disable', { current_password: currentPassword, code }, { skipAuthInvalidation: true })
}

export function logout() {
  return api.post('/admin/logout')
}

export function checkAuthStatus() {
  return api.get<{ authenticated: boolean; username?: string }>('/admin/status')
}

export function changePassword(currentPassword: string, newPassword: string) {
  return api.put('/admin/password', { current_password: currentPassword, new_password: newPassword })
}

export function changeUsername(currentPassword: string, newUsername: string) {
  return api.put('/admin/username', { current_password: currentPassword, new_username: newUsername })
}

// Config
export function getConfig() {
  return api.get<Config>('/config')
}

export function setTunnelID(value: string) {
  return api.post('/config/tunnel', { value })
}

export function setServiceURL(value: string) {
  return api.post('/config/service', { value })
}

export function setPreferredCNAME(value: string) {
  return api.post('/config/preferred-cname', { value })
}

// Tunnels
export function listTunnels() {
  return api.get<Tunnel[]>('/tunnels')
}

export function getTunnelDetail(tunnelID: string) {
  return api.get<TunnelDetail>(`/tunnels/${tunnelID}`)
}

export function addIngressRule(tunnelID: string, hostname: string, service: string) {
  return api.post<ApiResponse>(`/tunnels/${tunnelID}/ingress`, { hostname, service })
}

export function updateIngressRule(tunnelID: string, old_hostname: string, hostname: string, service: string) {
  return api.put<ApiResponse>(`/tunnels/${tunnelID}/ingress`, { old_hostname, hostname, service })
}

export function listZones() {
  return api.get<Zone[]>('/zones')
}

// Domain
export function bindDomain(data: BindRequest) {
  return api.post<ApiResponse>('/domain/bind', data)
}

export function bindDomainsBatch(items: BatchBindItem[]) {
  return api.post<BatchBindResponse>('/domain/bind-batch', { items })
}

export function setFallbackOrigin(domain: string) {
  return api.post<ApiResponse>('/domain/fallback', { domain })
}

// Telegram Bot
export interface TelegramSettings {
  enabled: boolean
  bot_token_set: boolean
  bot_token_hint: string
  admin_tg_ids: string
  mode: string
  webhook_url: string
  api_endpoint: string
}

export interface TelegramStatus {
  enabled: boolean
  running: boolean
  mode: string
  bot_username: string
  last_error: string
  last_update_at: string
}

export interface TelegramSettingsRequest {
  enabled: boolean
  bot_token: string
  admin_tg_ids: string
  mode: string
  webhook_url: string
  api_endpoint: string
}

export function getTelegramSettings() {
  return api.get<TelegramSettings>('/telegram/settings')
}

export function saveTelegramSettings(data: TelegramSettingsRequest) {
  return api.put<{ status: string; running: boolean; error?: string }>('/telegram/settings', data)
}

export function getTelegramStatus() {
  return api.get<TelegramStatus>('/telegram/status')
}

export function testTelegram() {
  return api.post<ApiResponse>('/telegram/test')
}