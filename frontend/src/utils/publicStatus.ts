export type PublicState = 'ok' | 'warn' | 'down' | 'unknown'

export function publicState(value?: string): PublicState {
  return value === 'ok' || value === 'warn' || value === 'down' ? value : 'unknown'
}

export function statusSummary(targets: ReadonlyArray<{ state?: string }>) {
  const counts = { ok: 0, warn: 0, down: 0, unknown: 0 }
  for (const target of targets) counts[publicState(target.state)]++
  const state = counts.down > 0 ? 'down' : counts.warn > 0 ? 'warn' : counts.unknown > 0 || targets.length === 0 ? 'unknown' : 'ok'
  return { counts, state: state as PublicState }
}

// Defense in depth for existing servers: public content must not supply an
// executable link, even if a stored monitor predates URL validation.
export function safePublicLink(value?: string): string {
  if (!value) return ''
  try {
    const url = new URL(value)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') return ''
    url.username = ''
    url.password = ''
    return url.href
  } catch {
    return ''
  }
}

export function safePublicIcon(value?: string): string {
  const icon = value?.trim() || ''
  if (/^\/uploads\/[^?#\\]+(?:\?[^#]*)?$/.test(icon) && !icon.split('/').includes('..')) return icon
  if (/^data:image\/(?:png|jpeg|gif|webp);base64,[a-z0-9+/=\s]+$/i.test(icon)) return icon
  return safePublicLink(icon)
}
