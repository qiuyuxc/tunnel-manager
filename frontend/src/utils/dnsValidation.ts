import { t as tr } from '../i18n.ts'
import type { DNSRecord, DNSRecordData, DNSRecordInput, DNSRecordType } from '../api/index.ts'

export type ValidatableDNSRecordType = DNSRecordType
export const DNS_TYPES: DNSRecordType[] = ['A', 'AAAA', 'CNAME', 'TXT', 'MX', 'NS', 'SRV', 'CAA', 'PTR']
export const DNS_BATCH_TYPES: DNSRecordType[] = ['A', 'AAAA', 'TXT', 'MX', 'NS', 'PTR']
export const DNS_BATCH_LIMIT = 100

export function proxyEligible(type: string): boolean { return ['A', 'AAAA', 'CNAME'].includes(type) }
export function editableType(type: string): type is DNSRecordType { return DNS_TYPES.includes(type as DNSRecordType) }
export function structuredType(type: string): boolean { return type === 'SRV' || type === 'CAA' }

export function editableRecord(record: DNSRecord): boolean {
  if (!editableType(record.type)) return false
  if (!structuredType(record.type)) return !record.data || Object.keys(record.data).length === 0
  return validateDNSRecord(recordInput(record)) === null
}

export function recordInput(record: DNSRecord): DNSRecordInput {
  return {
    type: record.type as DNSRecordType, name: record.name, content: structuredType(record.type) ? '' : record.content,
    ttl: record.ttl, proxied: proxyEligible(record.type) && !!record.proxied,
    ...(record.type === 'MX' ? { priority: record.priority ?? 0 } : {}),
    ...(structuredType(record.type) && record.data ? { data: { ...record.data } } : {}),
  }
}

export function defaultDNSData(type: string): DNSRecordData | undefined {
  if (type === 'SRV') return { priority: 0, weight: 0, port: 443, target: '' }
  if (type === 'CAA') return { flags: 0, tag: 'issue', value: '' }
}

export function normalizeDNSInput(input: DNSRecordInput): DNSRecordInput {
  const result = { ...input, name: input.name.trim().replace(/\.$/, ''), proxied: proxyEligible(input.type) && !!input.proxied }
  if (input.type !== 'TXT') result.content = input.content.trim()
  if (input.type !== 'MX') delete result.priority
  if (structuredType(input.type)) {
    result.content = ''
    result.data = { ...input.data }
    if (input.type === 'SRV') {
      const [service, proto, ...labels] = result.name.split('.')
      if (result.data.service) result.data.service = service
      if (result.data.proto) result.data.proto = proto
      if (result.data.name) result.data.name = labels.join('.')
      result.data.target = result.data.target?.trim()
    }
  } else delete result.data
  return result
}

function isIPv4(value: string): boolean {
  const parts = value.split('.')
  return parts.length === 4 && parts.every(part => /^(0|[1-9]\d{0,2})$/.test(part) && Number(part) <= 255)
}

function isIPv6(value: string): boolean {
  if (!/^[\da-fA-F:.]+$/.test(value)) return false
  try {
    new URL(`http://[${value}]/`)
    return value.includes(':')
  } catch {
    return false
  }
}

function isHostname(value: string): boolean {
  const hostname = value.replace(/\.$/, '')
  if (!hostname || hostname.length > 253 || isIPv4(hostname) || isIPv6(hostname)) return false
  return hostname.split('.').every(label =>
    label.length > 0 &&
    label.length <= 63 &&
    /^[A-Za-z0-9_](?:[A-Za-z0-9_-]*[A-Za-z0-9_])?$/.test(label),
  )
}

export function validateDNSContent(type: ValidatableDNSRecordType, rawContent: string): string | null {
  const content = rawContent.trim()
  if (!content) return tr('m_95ad513207d6')

  switch (type) {
    case 'A':
      return isIPv4(content) ? null : tr('m_658bc75675bb')
    case 'AAAA':
      return isIPv6(content) ? null : tr('m_27651148a1a7')
    case 'CNAME':
      return isHostname(content) ? null : tr('m_57a3d93cff1e')
    case 'MX':
      return content === '.' || isHostname(content) ? null : tr('m_941a243f03d1')
    case 'NS':
    case 'PTR':
      return isHostname(content) ? null : tr('m_e66800c3f076', { p0: type })
    case 'TXT':
      return null
    case 'SRV':
    case 'CAA':
      return tr('m_6a551fd95987', { p0: type })
  }
}

function integerInRange(value: unknown, maximum: number): boolean {
  return typeof value === 'number' && Number.isInteger(value) && value >= 0 && value <= maximum
}

export function validateDNSRecord(input: DNSRecordInput): string | null {
  if (!editableType(input.type)) return tr('m_7afbb9648b38')
  const name = input.name.trim().replace(/\.$/, '')
  if (name !== '@' && name !== '*' && !isHostname(name.replace(/^\*\./, ''))) return tr('m_571da4a006fb')
  if (input.ttl !== 1 && (!integerInRange(input.ttl, 86400) || input.ttl < 60)) return tr('m_c640ca457de3')
  if (structuredType(input.type)) {
    const data = input.data
    if (!data) return tr('m_4ce6fab5acfd', { p0: input.type })
    const keys = input.type === 'SRV' ? ['priority', 'weight', 'port', 'target', 'service', 'proto', 'name'] : ['flags', 'tag', 'value']
    if (Object.keys(data).some(key => !keys.includes(key))) return tr('m_fad41203d080')
    if (input.type === 'SRV') {
      if (!/^_[^.]+\._[^.]+\..+/.test(name)) return tr('m_df83b72b2b58')
      if (![data.priority, data.weight, data.port].every(value => integerInRange(value, 65535))) return tr('m_a6dc72c28ce8')
      if (typeof data.target !== 'string' || (data.target.trim() !== '.' && !isHostname(data.target.trim()))) return tr('m_a137ec3f71b5')
    } else {
      if (!integerInRange(data.flags, 255)) return tr('m_0ece41cc39ac')
      if (!/^[A-Za-z0-9]{1,15}$/.test(data.tag ?? '')) return tr('m_1c49e7ee8c85')
      if (typeof data.value !== 'string' || /[\0\r\n]/.test(data.value) || new TextEncoder().encode(data.value).length > 4096) return tr('m_8f5aab64a61a')
    }
    return null
  }
  if (new TextEncoder().encode(input.content).length > 4096 || input.content.includes('\0')) return tr('m_0d8414fecd29')
  const error = validateDNSContent(input.type, input.content)
  if (error) return error
  if (input.type === 'MX' && (!integerInRange(input.priority, 65535) || (input.content.trim() === '.' && input.priority !== 0))) return tr('m_b19238527e95')
  return null
}

export interface DNSBatchRow { line: number; content: string; error?: string; skipped?: string }
export interface DNSBatchResult extends DNSBatchRow { success: boolean; message: string; uncertain?: boolean; notSent?: boolean }

export function dnsFailureUncertain(error: any): boolean {
  const status = error.response?.status
  const message = error.response?.data?.error || error.message || ''
  return !status || status === 408 || status >= 500 || /request failed|read response failed|parse response failed|timeout|deadline|connection|network|EOF/i.test(message)
}

export function dnsValueKey(type: DNSRecordType, content: string): string {
  if (type === 'TXT') return content
  if (type === 'AAAA' && isIPv6(content.trim())) return new URL(`http://[${content.trim()}]/`).hostname
  return content.trim().toLowerCase().replace(/\.$/, '')
}

export function parseDNSBatch(type: DNSRecordType, text: string, existing: string[] = []): { rows: DNSBatchRow[]; error: string | null } {
  if (!DNS_BATCH_TYPES.includes(type)) return { rows: [], error: tr('m_d62c04c43b36') }
  if (text.length > 512000) return { rows: [], error: tr('m_7ba14fc9e64c') }
  const rows: DNSBatchRow[] = []
  const seen = new Map<string, number>()
  const existingKeys = new Set(existing.map(value => dnsValueKey(type, value)))
  text.split(/\r?\n/).forEach((raw, index) => {
    if (!raw.trim()) return
    const content = type === 'TXT' ? raw : raw.trim()
    const error = validateDNSContent(type, content) || (new TextEncoder().encode(content).length > 4096 || content.includes('\0') ? tr('m_0d8414fecd29') : undefined)
    const key = dnsValueKey(type, content)
    const skipped = existingKeys.has(key) ? tr('m_24a0a2486445') : seen.has(key) ? tr('m_71e59f81e400', { p0: seen.get(key) }) : undefined
    rows.push({ line: index + 1, content, error, skipped })
    if (!error && !skipped) seen.set(key, index + 1)
  })
  return { rows, error: rows.length > DNS_BATCH_LIMIT ? tr('m_717c8af97010', { p0: DNS_BATCH_LIMIT }) : null }
}

export class DNSBatchStoppedError extends Error {}

export async function runDNSBatch(rows: DNSBatchRow[], send: (row: DNSBatchRow) => Promise<unknown>, onResult: (result: DNSBatchResult) => void = () => {}): Promise<DNSBatchResult[]> {
  if (rows.length > DNS_BATCH_LIMIT || rows.some(row => row.error)) throw new Error(tr('m_231cf4730a4b'))
  const results: DNSBatchResult[] = []
  let stopReason = ''
  for (const row of rows.filter(row => !row.skipped)) {
    let result: DNSBatchResult
    if (stopReason) {
      result = { ...row, success: false, notSent: true, message: tr('m_fbc9162dd316', { p0: stopReason }) }
      results.push(result); onResult(result); continue
    }
    try {
      await send(row)
      result = { ...row, success: true, message: tr('m_1bcd31e0174a') }
    } catch (error: any) {
      if (error instanceof DNSBatchStoppedError) {
        stopReason = error.message
        result = { ...row, success: false, notSent: true, message: tr('m_fbc9162dd316', { p0: stopReason }) }
      } else {
        const uncertain = dnsFailureUncertain(error)
        const detail = error.response?.data?.error || error.message || tr('m_126fbf59cf04')
        result = { ...row, success: false, uncertain, message: uncertain ? tr('m_faa8fc8f6294', { p0: detail }) : detail }
        if (uncertain) stopReason = tr('m_f3e63077b975')
      }
    }
    results.push(result)
    onResult(result)
  }
  return results
}
