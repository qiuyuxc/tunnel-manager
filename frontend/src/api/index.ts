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
  tunnel_name: string
  service_url: string
  preferred_cname: string
  cname_presets: CNAMEPreset[]
  site_name: string
  site_description: string
  site_icon: string
}

export interface CNAMEPreset {
  name: string
  value: string
}

export interface SiteSettings {
  name: string
  description: string
  icon: string
}

export interface CloudflareAccount {
  id: string
  name: string
}

export interface CloudflareOAuthStatus {
  configured: boolean
  connected: boolean
  source: 'oauth' | 'api_token' | 'none'
  account_id: string
  account_name: string
  accounts: CloudflareAccount[] | null
  expires_at?: string
  redirect_uri: string
  error?: string
}

export type BindMode = 'simple' | 'preferred'

export interface BindRequest {
  mode: BindMode
  preferred_cname: string
  main_domain: string
  aux_domain: string
}

export interface BatchBindItem {
  mode: BindMode
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

export function getSiteSettings() {
  return api.get<SiteSettings>('/site')
}

export function setTunnelSelection(id: string, name: string) {
  return api.post('/config/tunnel', { id, name })
}

export function setServiceURL(value: string) {
  return api.post('/config/service', { value })
}

export function setPreferredCNAME(value: string) {
  return api.post('/config/preferred-cname', { value })
}

export function setSiteSettings(data: SiteSettings) {
  return api.put<SiteSettings>('/config/site', data)
}

export function setCNAMEPresets(items: CNAMEPreset[]) {
  return api.put<{ status: string; cname_presets: CNAMEPreset[] }>('/config/cname-presets', { items })
}

// Cloudflare OAuth
export function getCloudflareOAuthStatus() {
  return api.get<CloudflareOAuthStatus>('/cloudflare/oauth/status')
}

export function startCloudflareOAuth() {
  return api.post<{ authorization_url: string }>('/cloudflare/oauth/start')
}

export function selectCloudflareAccount(accountID: string) {
  return api.put<CloudflareAccount>('/cloudflare/oauth/account', { account_id: accountID })
}

export function disconnectCloudflareOAuth() {
  return api.delete<{ status: string; warning?: string }>('/cloudflare/oauth')
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

export type DNSRecordType = 'A' | 'AAAA' | 'CNAME' | 'TXT' | 'MX'

export interface DNSRecord {
  id: string
  type: string
  name: string
  content: string
  ttl: number
  proxied?: boolean
  priority?: number
  created_on?: string
  modified_on?: string
}

export interface DNSRecordInput {
  type: DNSRecordType
  name: string
  content: string
  ttl: number
  proxied?: boolean
  priority?: number
}

export function listDNSRecords(zoneID: string) {
  return api.get<DNSRecord[]>(`/zones/${zoneID}/dns-records`)
}

export function createDNSRecord(zoneID: string, data: DNSRecordInput) {
  return api.post<DNSRecord>(`/zones/${zoneID}/dns-records`, data)
}

export function updateDNSRecord(zoneID: string, recordID: string, data: DNSRecordInput) {
  return api.put<DNSRecord>(`/zones/${zoneID}/dns-records/${recordID}`, data)
}

export function deleteDNSRecord(zoneID: string, recordID: string) {
  return api.delete<ApiResponse>(`/zones/${zoneID}/dns-records/${recordID}`)
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
