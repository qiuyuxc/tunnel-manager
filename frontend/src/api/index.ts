import axios, { type InternalAxiosRequestConfig } from 'axios'

declare module 'axios' {
  interface AxiosRequestConfig {
    skipAuthInvalidation?: boolean
  }

  interface InternalAxiosRequestConfig {
    skipAuthInvalidation?: boolean
  }
}

export const api = axios.create({
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
      localStorage.removeItem('auth_role')
      localStorage.removeItem('auth_nickname')
      localStorage.removeItem('auth_avatar')
      localStorage.removeItem('auth_email')
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
  panel_host?: string
  landing_enabled: boolean
}

export interface CNAMEPreset {
  name: string
  value: string
}

export interface SiteSettings {
  name: string
  description: string
  icon: string
  landing_enabled: boolean
}

export interface CloudflareAccount {
  id: string
  name: string
}

export interface CFConnectionView {
  id: string
  label: string
  account_id: string
  account_name: string
  active: boolean
  expires_at: number
  created_at: number
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
  connections?: CFConnectionView[]
  active_connection_id?: string
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
  role?: string
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

export function login(account: string, password: string, turnstileToken = '') {
  return api.post<LoginResult>('/admin/login', { account, password, cf_turnstile_response: turnstileToken })
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

export function changeEmail(currentPassword: string, newEmail: string) {
  return api.put('/admin/email', { current_password: currentPassword, new_email: newEmail })
}

export function changeProfile(nickname: string, avatar: string) {
  return api.put('/admin/profile', { nickname, avatar })
}

export function uploadAvatar(file: File) {
  const form = new FormData()
  form.append('file', file)
  form.append('filename', file.name)
  return api.post<{ url: string }>('/account/avatar', form)
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

export function activateCloudflareConnection(connectionID: string) {
  return api.put<{ status: string; account_id: string; account_name: string }>('/cloudflare/oauth/connection', { connection_id: connectionID })
}

export function disconnectCloudflareOAuth(connectionID?: string) {
  return api.delete<{ status: string; warning?: string }>('/cloudflare/oauth', {
    params: connectionID ? { connection_id: connectionID } : undefined,
  })
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

export interface CreateTunnelResponse {
  id: string
  name: string
  token?: string
  run_command?: string
  warning?: string
}

export interface DeleteIngressResponse extends ApiResponse {
  dns_deleted?: number
  dns_warning?: string
}

export function createTunnel(name: string) {
  return api.post<CreateTunnelResponse>('/tunnels', { name })
}

export function deleteTunnel(tunnelID: string) {
  return api.delete<ApiResponse>(`/tunnels/${tunnelID}`)
}

export function deleteIngressRule(tunnelID: string, hostname: string, deleteDNS: boolean) {
  return api.delete<DeleteIngressResponse>(`/tunnels/${tunnelID}/ingress`, {
    data: { hostname, delete_dns: deleteDNS },
  })
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
  notify_bot_set: boolean
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

// Copies the notification bot token into the remote-control slot.
export function reuseTelegramFromNotify() {
  return api.post<ApiResponse>('/telegram/reuse')
}

// Updates the panel-wide Telegram Bot API endpoint (admin only).
export function saveTelegramAPIEndpoint(endpoint: string) {
  return api.put<{ status: string; api_endpoint: string }>('/telegram/endpoint', { api_endpoint: endpoint })
}

// Service monitor
export interface ServiceProbe {
  hostname: string
  service: string
  state: 'ok' | 'warn' | 'down'
  http_code?: number
  latency_ms: number
  error?: string
}

export interface ServicesHealth {
  tunnel_id?: string
  tunnel_name?: string
  checked_at: string
  services: ServiceProbe[]
}

export function getServicesHealth() {
  return api.get<ServicesHealth>('/monitor/services', { timeout: 20000 })
}

// ---- Monitor projects (uptime-style) ----
export interface Heartbeat {
  t: number
  s: 'ok' | 'warn' | 'down'
  ms?: number
  c?: number
  e?: string
}

export interface TargetStatus {
  id: string
  name: string
  url: string
  type?: 'http' | 'tcp' | 'icmp'
  link_enabled?: boolean
  method?: '' | 'GET' | 'POST'
  state?: '' | 'ok' | 'warn' | 'down'
  latency_ms?: number
  http_code?: number
  error?: string
  uptime_24h: number
  bars?: Heartbeat[]
}

export interface MonitorView {
  id: string
  name: string
  interval_sec: number
  publish_enabled: boolean
  public_token?: string
  public_slug?: string
  public_domain?: string
  domain_warning?: string
  public_title?: string
  public_icon?: string
  public_theme?: '' | 'blue' | 'warm'
  announcement?: string
  alert_enabled?: boolean
  alert_emails?: string
  created_at?: number
  targets: TargetStatus[]
}

export interface AlertLog {
  id: number
  monitor_id: string
  target_id: string
  target_name: string
  state: string
  http_code: number
  error: string
  notified: boolean
  detail: string
  created_at: number
}

export function listMonitorAlerts(id: string) {
  return api.get<AlertLog[]>(`/monitors/${id}/alerts`)
}

export function listMonitors() {
  return api.get<MonitorView[]>('/monitors')
}

export function createMonitor(name: string) {
  return api.post<MonitorView>('/monitors', { name })
}

export function getMonitor(id: string) {
  return api.get<MonitorView>(`/monitors/${id}`)
}

export interface MonitorUpdate {
  name?: string
  interval_sec?: number
  publish_enabled?: boolean
  regenerate_token?: boolean
  public_title?: string
  public_slug?: string
  public_domain?: string
  public_icon?: string
  public_theme?: '' | 'blue' | 'warm'
  announcement?: string
  alert_enabled?: boolean
  alert_emails?: string
}

export function updateMonitor(id: string, patch: MonitorUpdate) {
  return api.put<MonitorView>(`/monitors/${id}`, patch)
}

export function deleteMonitor(id: string) {
  return api.delete<ApiResponse>(`/monitors/${id}`)
}

export interface ProbeOutcome {
  target_id: string
  name: string
  url: string
  state: string
  http_code?: number
  latency_ms: number
  error?: string
}

export function checkMonitorNow(id: string) {
  return api.post<{ outcomes: ProbeOutcome[] }>(`/monitors/${id}/check`)
}

export function addMonitorTarget(id: string, name: string, url: string, probeType?: string, method?: string, linkEnabled?: boolean) {
  return api.post<MonitorView>(`/monitors/${id}/targets`, { name, url, type: probeType, method, link_enabled: linkEnabled })
}

export function editMonitorTarget(
  id: string,
  targetId: string,
  payload: { name: string; url: string; type?: string; method?: string; link_enabled?: boolean }
) {
  return api.put<MonitorView>(`/monitors/${id}/targets/${targetId}`, payload)
}

export function uploadImage(file: File) {
  const fd = new FormData()
  fd.append('file', file)
  fd.append('filename', file.name)
  return api.post<{ url: string }>('/uploads', fd)
}

export function removeMonitorTarget(id: string, targetId: string) {
  return api.delete<MonitorView>(`/monitors/${id}/targets/${targetId}`)
}

export interface BucketStat {
  hour: number
  avg_ms: number
  peak_ms: number
  total: number
  warn: number
  down: number
}

export interface OverviewResp {
  targets: number
  ok: number
  warn: number
  down: number
  uptime_24h: number
  avg_latency_ms: number
  peak_latency_ms: number
  buckets: BucketStat[]
}

export function getMonitorOverview() {
  return api.get<OverviewResp>('/monitors/overview', { timeout: 20000 })
}

/** 公开状态页的页面路由（给浏览器地址栏 / 复制分享用） */
export function publicStatusPath(token: string) {
  return `/status/${token}`
}

/** 公开状态数据的接口地址（仅供页面内部拉取） */
export function publicStatusApiUrl(token: string) {
  return `/api/public/status/${token}`
}

export async function fetchPublicStatus(token: string): Promise<PublicStatusData> {
  const res = await fetch(publicStatusApiUrl(token))
  if (!res.ok) throw new Error('status page unavailable')
  return res.json() as Promise<PublicStatusData>
}

export interface PublicStatusData {
  name: string
  public_title?: string
  public_icon?: string
  public_theme?: 'blue' | 'warm'
  announcement?: string
  updated_at: number
  interval_sec: number
  targets: Array<{
    name: string
    state: string
    latency_ms: number
    http_code?: number
    error?: string
    uptime_24h: number
    link?: string
    bars: Heartbeat[]
  }>
}
