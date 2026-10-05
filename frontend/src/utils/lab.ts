export function isLabAuthorized(
  verification: { verified_at: number; owner_id: string; domain: string } | undefined,
  ownerId: string,
): boolean {
  return !!verification?.verified_at && !!ownerId && verification.owner_id === ownerId
    && !!verification.domain.trim()
}
export function describeLabFailure(result: { status?: number; error?: string }, statuses?: string): string {
  if (result.error) return `探测失败：${result.error}`
  if (result.status) return `HTTP ${result.status}，不在本次接受状态码${statuses ? ` ${statuses}` : '列表'}中`
  return '未收到有效 HTTP 响应，请检查网络、TLS 和超时设置'
}
export function validateLabLimits(settings: {
  lab_daily_request_limit?: number
  lab_requests_per_second?: number
  lab_max_workers?: number
}): string {
  for (const [value, maximum, label] of [
    [settings.lab_daily_request_limit, 10000000, '每日请求预算'],
    [settings.lab_requests_per_second, 1000, '每秒请求数'],
    [settings.lab_max_workers, 256, '并发上限'],
  ] as const) {
    if (!Number.isInteger(value) || value === undefined || value < 1 || value > maximum) {
      return `${label}须为 1 至 ${maximum} 的整数`
    }
  }
  return ''
}
