import { api } from './index'

export interface LabIPSelectorSettings {
  host: string
  sni: string
  path: string
  statuses: string
  timeout: number
  workers: number
  top: number
  ip_targets: string
  schedule: boolean
  interval_minutes: number
  update_dns: boolean
  zone: string
  zone_id: string
  record: string
  ttl: number
  endpoint: string
  access_key: string
  has_secret_key: boolean
}

export interface SaveLabIPSelectorSettings extends Omit<LabIPSelectorSettings, 'has_secret_key'> {
  secret_key?: string
}

export interface LabIPScanResult {
  ip: string
  status?: number
  latency_ms: number
  error?: string
}

export interface LabIPSegmentResult {
  target: string
  scanned: number
  matched: number
}

export interface LabIPSelectorRun {
  id: string
  started_at: number
  finished_at: number
  success: boolean
  error?: string
  scanned: number
  matched: number
  selected_ips?: string[]
  results?: LabIPScanResult[]
  diagnostics?: LabIPScanResult[]
  rejected?: number
  probe?: { host: string; sni: string; path: string; statuses: string }
  segments?: LabIPSegmentResult[]
  dns_updated: boolean
}

export interface LabIPSelectorProgress {
  scanned: number
  matched: number
  total: number
  percent: number
}

export interface LabIPSelectorStatusResponse {
  status: {
    running: boolean
    next_run?: string
    last_run?: LabIPSelectorRun
    progress: LabIPSelectorProgress
    phase: string
    verification: LabDomainVerification
    suggested_domain: string
    budget_day: string
    budget_used: number
    budget_limit: number
    requests_per_second: number
    max_workers: number
    next_allowed_at: number
    last_error: string
  }
  runs: LabIPSelectorRun[]
}

export interface LabDomainVerification {
  domain: string
  host?: string
  sni?: string
  record_name: string
  token: string
  owner_id: string
  issued_at: number
  verified_at: number
  checked_at: number
  last_error: string
}

export interface LabDNSAvailability {
  available: boolean
  account_name?: string
  message: string
}

export interface LabDNSProvisionResult {
  verification: LabDomainVerification
  record_created: boolean
  verified: boolean
  message: string
}

export const getLabIPSelectorSettings = () => api.get<LabIPSelectorSettings>('/lab/ip-selector')
export const saveLabIPSelectorSettings = (payload: SaveLabIPSelectorSettings) =>
  api.put<LabIPSelectorSettings>('/lab/ip-selector', payload)
export const getLabIPSelectorStatus = () => api.get<LabIPSelectorStatusResponse>('/lab/ip-selector/status')
export const runLabIPSelector = () => api.post<{ started: boolean }>('/lab/ip-selector/run', {})
export const stopLabIPSelector = () => api.post('/lab/ip-selector/stop', {})
export const createLabChallenge = (domain: string) => api.post<LabDomainVerification>('/lab/ip-selector/ownership/challenge', { domain })
export const verifyLabOwnership = () => api.post<LabDomainVerification>('/lab/ip-selector/ownership/verify', {})
export const getLabDNSAvailability = () => api.get<LabDNSAvailability>('/lab/ip-selector/ownership/dns')
export const provisionLabDNS = (domain: string) => api.post<LabDNSProvisionResult>('/lab/ip-selector/ownership/dns', { domain })
