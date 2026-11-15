import type { Heartbeat } from './index'

/** 公开状态页的页面路由（给浏览器地址栏 / 复制分享用） */
export function publicStatusPath(token: string) {
  return `/status/${encodeURIComponent(token)}`
}

/** 公开状态数据的接口地址（仅供页面内部拉取） */
export function publicStatusApiUrl(token: string) {
  return `/api/public/status/${encodeURIComponent(token)}`
}

export async function fetchPublicStatus(token: string, signal?: AbortSignal): Promise<PublicStatusData> {
  const res = await fetch(publicStatusApiUrl(token), { signal, cache: 'no-store', credentials: 'omit' })
  if (!res.ok) throw Object.assign(new Error('status page unavailable'), { status: res.status })
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
  /** Exact rolling 24-hour range represented by the history bars, in Unix milliseconds. */
  window_start: number
  window_end: number
  /** Each public history bar is a 30-minute window, not an individual check. */
  bucket_sec: number
  targets: Array<{
    name: string
    state: string
    latency_ms: number
    http_code?: number
    error?: string
    uptime_24h: number
    observed_count: number
    link?: string
    bars: Array<Omit<Heartbeat, 's'> & { s: Heartbeat['s'] | 'unknown'; samples: number; ok: number; warn: number; down: number }>
  }>
}
