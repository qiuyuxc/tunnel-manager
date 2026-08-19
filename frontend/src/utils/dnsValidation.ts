export type ValidatableDNSRecordType = 'A' | 'AAAA' | 'CNAME' | 'TXT' | 'MX'

function isIPv4(value: string): boolean {
  const parts = value.split('.')
  return parts.length === 4 && parts.every(part => /^\d{1,3}$/.test(part) && Number(part) <= 255)
}

function isIPv6(value: string): boolean {
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
      return isHostname(content) ? null : 'MX 记录必须填写邮件服务器域名'
    case 'TXT':
      return null
  }
}
