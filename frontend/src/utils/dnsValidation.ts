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
  if (!content) return '解析值不能为空'

  switch (type) {
    case 'A':
      return isIPv4(content) ? null : 'A 记录必须填写有效的 IPv4 地址'
    case 'AAAA':
      return isIPv6(content) ? null : 'AAAA 记录必须填写有效的 IPv6 地址'
    case 'CNAME':
      return isHostname(content) ? null : 'CNAME 记录必须填写域名目标，不能填写 IP 地址'
    case 'MX':
      return content === '.' || isHostname(content) ? null : 'MX 记录必须填写邮件服务器域名'
    case 'NS':
    case 'PTR':
      return isHostname(content) ? null : `${type} 记录必须填写域名目标`
    case 'TXT':
      return null
    case 'SRV':
    case 'CAA':
      return `${type} 记录请使用结构化字段`
  }
}

function integerInRange(value: unknown, maximum: number): boolean {
  return typeof value === 'number' && Number.isInteger(value) && value >= 0 && value <= maximum
}

export function validateDNSRecord(input: DNSRecordInput): string | null {
  if (!editableType(input.type)) return '不支持的记录类型，只读保护'
  const name = input.name.trim().replace(/\.$/, '')
  if (name !== '@' && name !== '*' && !isHostname(name.replace(/^\*\./, ''))) return '请输入有效的记录名称（国际域名请使用 Punycode）'
  if (input.ttl !== 1 && (!integerInRange(input.ttl, 86400) || input.ttl < 60)) return 'TTL 必须为自动或 60–86400 秒'
  if (structuredType(input.type)) {
    const data = input.data
    if (!data) return `${input.type} 缺少结构化字段`
    const keys = input.type === 'SRV' ? ['priority', 'weight', 'port', 'target', 'service', 'proto', 'name'] : ['flags', 'tag', 'value']
    if (Object.keys(data).some(key => !keys.includes(key))) return '含有不支持的 data 字段，只读保护'
    if (input.type === 'SRV') {
      if (!/^_[^.]+\._[^.]+\..+/.test(name)) return 'SRV 名称应为 _服务._协议.域名，例如 _sip._tcp.example.com'
      if (![data.priority, data.weight, data.port].every(value => integerInRange(value, 65535))) return 'SRV 优先级、权重和端口必须为 0–65535 的整数'
      if (typeof data.target !== 'string' || (data.target.trim() !== '.' && !isHostname(data.target.trim()))) return 'SRV 目标必须为域名或 .'
    } else {
      if (!integerInRange(data.flags, 255)) return 'CAA flags 必须为 0–255 的整数'
      if (!/^[A-Za-z0-9]{1,15}$/.test(data.tag ?? '')) return 'CAA tag 必须为 1–15 个字母或数字'
      if (typeof data.value !== 'string' || /[\0\r\n]/.test(data.value) || new TextEncoder().encode(data.value).length > 4096) return 'CAA value 必须为单行文本，最多 4096 字节'
    }
    return null
  }
  if (new TextEncoder().encode(input.content).length > 4096 || input.content.includes('\0')) return '解析值最多 4096 字节，不能包含空字符'
  const error = validateDNSContent(input.type, input.content)
  if (error) return error
  if (input.type === 'MX' && (!integerInRange(input.priority, 65535) || (input.content.trim() === '.' && input.priority !== 0))) return 'MX 优先级必须为 0–65535 的整数（空 MX 为 0）'
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
  if (!DNS_BATCH_TYPES.includes(type)) return { rows: [], error: '批量新增仅支持 A / AAAA / TXT / MX / NS / PTR；CNAME、SRV、CAA 请单条添加' }
  if (text.length > 512000) return { rows: [], error: '粘贴内容过大，请拆分批次' }
  const rows: DNSBatchRow[] = []
  const seen = new Map<string, number>()
  const existingKeys = new Set(existing.map(value => dnsValueKey(type, value)))
  text.split(/\r?\n/).forEach((raw, index) => {
    if (!raw.trim()) return
    const content = type === 'TXT' ? raw : raw.trim()
    const error = validateDNSContent(type, content) || (new TextEncoder().encode(content).length > 4096 || content.includes('\0') ? '解析值最多 4096 字节，不能包含空字符' : undefined)
    const key = dnsValueKey(type, content)
    const skipped = existingKeys.has(key) ? '已存在，跳过' : seen.has(key) ? `与第 ${seen.get(key)} 行重复，跳过` : undefined
    rows.push({ line: index + 1, content, error, skipped })
    if (!error && !skipped) seen.set(key, index + 1)
  })
  return { rows, error: rows.length > DNS_BATCH_LIMIT ? `每批最多 ${DNS_BATCH_LIMIT} 条非空行（含重复行）` : null }
}

export class DNSBatchStoppedError extends Error {}

export async function runDNSBatch(rows: DNSBatchRow[], send: (row: DNSBatchRow) => Promise<unknown>, onResult: (result: DNSBatchResult) => void = () => {}): Promise<DNSBatchResult[]> {
  if (rows.length > DNS_BATCH_LIMIT || rows.some(row => row.error)) throw new Error('请先修正预览中的校验错误')
  const results: DNSBatchResult[] = []
  let stopReason = ''
  for (const row of rows.filter(row => !row.skipped)) {
    let result: DNSBatchResult
    if (stopReason) {
      result = { ...row, success: false, notSent: true, message: `未发送：${stopReason}` }
      results.push(result); onResult(result); continue
    }
    try {
      await send(row)
      result = { ...row, success: true, message: '已新增' }
    } catch (error: any) {
      if (error instanceof DNSBatchStoppedError) {
        stopReason = error.message
        result = { ...row, success: false, notSent: true, message: `未发送：${stopReason}` }
      } else {
        const uncertain = dnsFailureUncertain(error)
        const detail = error.response?.data?.error || error.message || '请求失败'
        result = { ...row, success: false, uncertain, message: uncertain ? `结果未知，请核对后再重试：${detail}` : detail }
        if (uncertain) stopReason = '前一请求结果未知，队列已停止'
      }
    }
    results.push(result)
    onResult(result)
  }
  return results
}
