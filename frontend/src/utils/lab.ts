import { t as tr } from '../i18n.ts'
export function isLabAuthorized(
  verification: { verified_at: number; owner_id: string; domain: string } | undefined,
  ownerId: string,
): boolean {
  return !!verification?.verified_at && !!ownerId && verification.owner_id === ownerId
    && !!verification.domain.trim()
}
export function describeLabFailure(result: { status?: number; error?: string }, statuses?: string): string {
  if (result.error) return tr('m_a73beb267311', { p0: result.error })
  if (result.status) return tr('m_608eb97e72a2', { p0: result.status, p1: statuses ? ` ${statuses}` : '列表' })
  return tr('m_68dba47286c3')
}
export function validateLabLimits(settings: {
  lab_daily_request_limit?: number
  lab_requests_per_second?: number
  lab_max_workers?: number
}): string {
  for (const [value, maximum, label] of [
    [settings.lab_daily_request_limit, 10000000, tr('m_823ba31c4299')],
    [settings.lab_requests_per_second, 1000, tr('m_b8f8cb247e4d')],
    [settings.lab_max_workers, 256, tr('m_652a6f861b6d')],
  ] as const) {
    if (!Number.isInteger(value) || value === undefined || value < 1 || value > maximum) {
      return tr('m_32da6b5b2c56', { p0: label, p1: maximum })
    }
  }
  return ''
}
