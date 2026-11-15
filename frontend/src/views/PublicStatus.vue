<template>
  <main class="pub-page">
    <div class="pub-wrap">
      <header class="pub-topbar">
        <span class="pub-eyebrow"><span class="live-dot"></span>{{ tr('status.public') }}</span>
        <div class="pub-actions">
          <RefreshButton :busy="loading" @refresh="load" />
          <button class="pub-button icon-button" :title="isDark ? tr('m_8a9add32ce6a') : tr('m_3defe73838c1')" :aria-label="isDark ? tr('m_8a9add32ce6a') : tr('m_3defe73838c1')" @click="toggleTheme">
            <svg v-if="isDark" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.4 1.4m11.2 11.2L19 19M5 19l1.4-1.4M17.6 6.4L19 5"/></svg>
            <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M20.5 13.2A8.5 8.5 0 0 1 10.8 3.5 8.5 8.5 0 1 0 20.5 13.2Z"/></svg>
          </button>
        </div>
      </header>

      <template v-if="data">
        <section class="pub-hero">
          <div class="pub-brand" aria-hidden="true">
            <img v-if="iconUrl" :src="iconUrl" alt="" referrerpolicy="no-referrer" />
            <span v-else-if="data.public_icon && !looksLikeIconUrl" class="brand-emoji">{{ data.public_icon }}</span>
            <svg v-else width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="16" rx="5"/><path d="M6 12h3l2-4 3 8 2-4h2"/></svg>
          </div>
          <div class="pub-heading">
            <h1>{{ data.public_title || data.name }}</h1>
            <p class="pub-sub" v-if="data.updated_at">{{ tr('status.lastCheck') }} <time :datetime="new Date(data.updated_at).toISOString()">{{ formatTime(data.updated_at) }}</time></p>
            <p class="pub-sub" v-else>{{ tr('status.waitingCheck') }}</p>
          </div>
          <span class="pub-range">{{ tr('status.last24h') }}</span>
        </section>

        <section class="pub-overall" :class="summary.state" role="status">
          <span class="overall-icon" aria-hidden="true">
            <svg v-if="summary.state === 'ok'" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="m6 12 4 4 8-8"/></svg>
            <svg v-else width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><circle cx="12" cy="12" r="8"/><path d="M12 7v6m0 4h.01"/></svg>
          </span>
          <div><h2>{{ overallText }}</h2><p>{{ tr('status.checkInterval', { seconds: data.interval_sec }) }}</p></div>
          <span class="overall-state"><span class="st-dot" :class="'d-' + summary.state"></span>{{ stateText(summary.state) }}</span>
        </section>

        <div v-if="error" class="pub-notice error-notice" role="alert"><span>{{ error }}</span><button class="text-button" :disabled="loading" @click="load">{{ tr('status.retry') }}</button></div>
        <aside v-if="data.announcement" class="pub-notice"><span class="notice-label">{{ tr('status.announcement') }}</span><p>{{ data.announcement }}</p></aside>

        <div class="pub-metrics">
          <div><span>{{ tr('status.services') }}</span><strong>{{ data.targets.length }}</strong></div>
          <div><span>{{ tr('status.operational') }}</span><strong class="metric-ok">{{ summary.counts.ok }}</strong></div>
          <div><span>{{ tr('status.issues') }}</span><strong :class="{ 'metric-issue': summary.counts.warn + summary.counts.down > 0 }">{{ summary.counts.warn + summary.counts.down }}</strong></div>
          <div><span>{{ tr('status.unobserved') }}</span><strong>{{ summary.counts.unknown }}</strong></div>
        </div>

        <section class="pub-services" :aria-label="tr('status.services')">
          <div class="section-heading"><h2>{{ tr('status.serviceHistory') }}</h2><span>{{ tr('status.historyScale') }}</span></div>
          <div v-if="!data.targets.length" class="pub-empty">{{ tr('status.noServices') }}</div>
          <div class="pub-list">
            <article v-for="(target, ti) in data.targets" :key="ti" class="svc">
              <header class="svc-head">
                <div class="svc-title"><span class="st-dot" :class="'d-' + publicState(target.state)"></span><h3>{{ target.name }}</h3>
                  <a v-if="safePublicLink(target.link)" class="svc-link" :href="safePublicLink(target.link)" target="_blank" rel="noopener noreferrer" :aria-label="tr('m_84041df0ce44') + target.name"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M14 4h6v6m0-6-10 10M10 4H6a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-4"/></svg></a>
                </div>
                <span class="svc-current" :class="'current-' + publicState(target.state)">{{ stateText(target.state) }}<span v-if="target.state === 'ok' || target.state === 'warn'" class="lat">{{ target.latency_ms }} ms</span></span>
              </header>
              <div class="svc-uptime"><span>{{ tr('status.observedUptime') }}</span><strong>{{ observedCount(target) ? fmtPct(target.uptime_24h) : '—' }}</strong><span>{{ tr('status.checkCount', { count: observedCount(target) }) }}</span></div>
              <div class="bars-lg" role="group" :aria-label="tr('status.historyGroup', { name: target.name })">
                <button v-for="(bar, bi) in target.bars" :key="bi" type="button" :class="['bar', 'b-' + publicState(bar.s)]" :tabindex="bi === (selected[ti] ?? target.bars.length - 1) ? 0 : -1" :title="barDescription(bar)" :aria-label="barDescription(bar)" @focus="selected[ti] = bi" @click="selected[ti] = bi" @keydown="moveBar($event, ti, bi, target.bars.length)"></button>
              </div>
              <footer class="svc-foot"><time v-if="data.window_start" :datetime="new Date(data.window_start).toISOString()">{{ formatTime(data.window_start) }}</time><span v-else>{{ tr('status.last24h') }}</span><time v-if="data.window_end" :datetime="new Date(data.window_end).toISOString()">{{ formatTime(data.window_end) }}</time><span v-else>{{ tr('status.now') }}</span></footer>
              <p v-if="selectedDescription(target, ti)" class="bar-detail" aria-live="polite">{{ selectedDescription(target, ti) }}</p>
            </article>
          </div>
        </section>

        <div class="pub-legend"><span v-for="state in legendStates" :key="state"><i :class="'d-' + state"></i>{{ stateText(state) }}</span></div>
        <p class="history-note">{{ tr('status.historyNote') }}</p>
      </template>

      <section v-else-if="error" class="pub-error" role="alert">
        <div class="pub-brand"><svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"><circle cx="12" cy="12" r="9"/><path d="M12 7v6m0 4h.01"/></svg></div>
        <h1>{{ tr('status.unavailable') }}</h1><p>{{ error }}</p>
        <div class="error-actions"><button class="pub-button" :disabled="loading" @click="load">{{ tr('status.retry') }}</button><router-link to="/" class="pub-button">{{ tr('m_d66f4359dae3') }}</router-link></div>
      </section>
      <section v-else class="pub-loading" aria-busy="true" :aria-label="tr('status.loading')">
        <div class="loading-heading"><div class="skeleton skeleton-icon"></div><div><div class="skeleton skeleton-title"></div><div class="skeleton skeleton-sub"></div></div></div>
        <div class="skeleton skeleton-banner"></div><div class="loading-cards"><div v-for="i in 3" :key="i" class="skeleton skeleton-card"></div></div><p role="status">{{ tr('status.loading') }}</p>
      </section>
      <footer class="pub-footer"><span>Tunnel Manager</span><span>{{ tr('status.autoRefresh') }}</span></footer>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { fetchPublicStatus, type PublicStatusData } from '../api/publicStatus'
import { locale, t as tr } from '../i18n'
import RefreshButton from '../components/RefreshButton.vue'
import { publicState, safePublicIcon, safePublicLink, statusSummary } from '../utils/publicStatus'

const route = useRoute()
const data = ref<PublicStatusData | null>(null)
const error = ref('')
const loading = ref(false)
const selected = ref<Record<number, number>>({})
const legendStates = ['ok', 'warn', 'down', 'unknown'] as const
let savedVisual: string | null = null
let savedDark: string | null = null
let savedTitle = ''
let publicTitle = ''
let mounted = false
let controller: AbortController | undefined
let timer: ReturnType<typeof setTimeout> | undefined

function storedDark() {
  try {
    const value = localStorage.getItem('pub_dark_mode')
    return value === 'true' || (value === null && matchMedia('(prefers-color-scheme: dark)').matches)
  } catch { return matchMedia('(prefers-color-scheme: dark)').matches }
}
const isDark = ref(storedDark())
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.setAttribute('data-theme', isDark.value ? 'dark' : '')
  try { localStorage.setItem('pub_dark_mode', String(isDark.value)) } catch {}
}
const iconUrl = computed(() => safePublicIcon(data.value?.public_icon))
const looksLikeIconUrl = computed(() => /^(?:[a-z][a-z\d+.-]*:|\/)/i.test(data.value?.public_icon || ''))
const summary = computed(() => statusSummary(data.value?.targets || []))
const overallText = computed(() => {
  const { state, counts } = summary.value
  if (state === 'down') return tr('status.downCount', { count: counts.down })
  if (state === 'warn') return tr('status.warnCount', { count: counts.warn })
  if (!data.value?.targets.length) return tr('status.noServices')
  if (state === 'unknown') return tr('status.unknownSummary')
  return tr('m_7a5db8b2e966')
})
function stateText(state: string) { return tr(`status.state.${publicState(state)}`) }
const timeFormatter = computed(() => new Intl.DateTimeFormat(locale.value, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', timeZoneName: 'short' }))
function formatTime(value: number) { return Number.isFinite(value) ? timeFormatter.value.format(value) : '—' }
function fmtPct(value: number) { return Number.isFinite(value) ? `${value.toFixed(2).replace(/\.00$/, '')}%` : '—' }
type Target = PublicStatusData['targets'][number]
type Bar = Target['bars'][number]
function observedCount(target: Target) {
  return target.observed_count ?? target.bars.reduce((sum, bar) => sum + (bar.samples ?? 1), 0)
}
function barDescription(bar: Bar) {
  const end = bar.t + (data.value?.bucket_sec || 1800) * 1000
  const range = `${formatTime(bar.t)} – ${formatTime(end)}`
  if (!bar.samples && publicState(bar.s) === 'unknown') return `${range} · ${tr('status.noChecks')}`
  if (bar.samples === undefined) return `${formatTime(bar.t)} · ${stateText(bar.s)}`
  return `${range} · ${tr('status.bucketDetail', { count: bar.samples, ok: bar.ok, warn: bar.warn, down: bar.down })}`
}
function selectedDescription(target: Target, targetIndex: number) {
  const index = selected.value[targetIndex]
  return index !== undefined && target.bars[index] ? barDescription(target.bars[index]) : ''
}
function moveBar(event: KeyboardEvent, targetIndex: number, index: number, length: number) {
  let next = index
  if (event.key === 'ArrowRight') next = Math.min(index + 1, length - 1)
  else if (event.key === 'ArrowLeft') next = Math.max(index - 1, 0)
  else if (event.key === 'Home') next = 0
  else if (event.key === 'End') next = length - 1
  else return
  event.preventDefault()
  selected.value[targetIndex] = next
  const group = (event.currentTarget as HTMLElement).parentElement
  ;(group?.children[next] as HTMLButtonElement | undefined)?.focus()
}
function schedule() {
  if (timer) clearTimeout(timer)
  if (mounted && !document.hidden) timer = setTimeout(load, 30000)
}
async function load() {
  if (loading.value || !mounted) return
  if (timer) clearTimeout(timer)
  loading.value = true
  const request = new AbortController()
  controller = request
  const timeout = setTimeout(() => request.abort(), 12000)
  try {
    const response = await fetchPublicStatus(String(route.params.token || ''), request.signal)
    if (!mounted || controller !== request) return
    data.value = response
    error.value = ''
    const root = document.documentElement
    root.setAttribute('data-visual-theme', 'enterprise')
    publicTitle = `${response.public_title || response.name || tr('status.public')} · Tunnel Manager`
    document.title = publicTitle
  } catch (cause) {
    if (!mounted || controller !== request) return
    const status = (cause as { status?: number }).status
    if (status === 404) data.value = null
    error.value = data.value ? tr('status.refreshFailed') : status === 404 ? tr('m_2f8ac8eb2be1') : tr('status.loadFailed')
  } finally {
    clearTimeout(timeout)
    if (controller === request) {
      loading.value = false
      controller = undefined
      schedule()
    }
  }
}
function onVisibilityChange() {
  if (document.hidden) { if (timer) clearTimeout(timer) }
  else void load()
}
watch(() => route.params.token, () => {
  controller?.abort()
  controller = undefined
  loading.value = false
  data.value = null
  error.value = ''
  selected.value = {}
  void load()
})
onMounted(() => {
  mounted = true
  const root = document.documentElement
  savedVisual = root.getAttribute('data-visual-theme')
  savedDark = root.getAttribute('data-theme')
  savedTitle = document.title
  root.setAttribute('data-theme', isDark.value ? 'dark' : '')
  root.removeAttribute('data-visual-theme')
  document.addEventListener('visibilitychange', onVisibilityChange)
  void load()
})
onBeforeUnmount(() => {
  mounted = false
  controller?.abort()
  if (timer) clearTimeout(timer)
  document.removeEventListener('visibilitychange', onVisibilityChange)
  const root = document.documentElement
  try {
    root.setAttribute('data-visual-theme', 'enterprise')
    root.setAttribute('data-theme', localStorage.getItem('dark_mode') !== 'false' ? 'dark' : '')
  } catch {
    if (savedVisual === null) root.removeAttribute('data-visual-theme')
    else root.setAttribute('data-visual-theme', savedVisual)
    if (savedDark === null) root.removeAttribute('data-theme')
    else root.setAttribute('data-theme', savedDark)
  }
  if (document.title === publicTitle) document.title = savedTitle
})
</script>

<style scoped>
.pub-page { min-height: 100dvh; box-sizing: border-box; padding: 28px 24px 40px; color: var(--color-ink); background: var(--color-canvas-soft); }
.pub-wrap { max-width: 920px; margin: 0 auto; }
.pub-topbar, .pub-actions, .pub-hero, .svc-title, .svc-current, .error-actions { display: flex; align-items: center; }
.pub-topbar { justify-content: space-between; gap: 16px; margin-bottom: 52px; }
.pub-eyebrow { display: flex; align-items: center; gap: 9px; color: var(--color-mute); font-size: 12px; letter-spacing: .04em; }
.live-dot, .st-dot { width: 7px; height: 7px; flex-shrink: 0; border-radius: 50%; background: var(--color-mute); }
.live-dot { background: var(--color-ink); }
.pub-actions { gap: 8px; }
.pub-button { display: inline-flex; align-items: center; justify-content: center; gap: 8px; min-height: 42px; padding: 0 16px; border-radius: 13px; border: 1px solid var(--color-hairline); background: var(--color-canvas-raised); color: var(--color-body); cursor: pointer; font-size: 12px; text-decoration: none; }
.pub-button:hover:not(:disabled) { border-color: var(--color-hairline-strong); color: var(--color-ink); background: var(--color-canvas-soft-2); }
.pub-button:disabled { cursor: wait; opacity: .65; }
.icon-button { width: 42px; padding: 0; }
.pub-hero { gap: 18px; margin-bottom: 28px; }
.pub-brand { width: 60px; height: 60px; flex-shrink: 0; display: grid; place-items: center; border: 1px solid var(--color-hairline); border-radius: 19px; background: var(--color-canvas-raised); color: var(--color-focus); overflow: hidden; }
.pub-brand img { display: block; width: 100%; height: 100%; object-fit: cover; }
.brand-emoji { font-size: 34px; line-height: 1; }
.pub-heading { min-width: 0; flex: 1; }
.pub-heading h1 { font-size: clamp(24px, 4vw, 32px); font-weight: 600; line-height: 1.3; letter-spacing: -.035em; margin: 0 0 9px; overflow-wrap: anywhere; }
.pub-sub { margin: 0; font-size: 12px; color: var(--color-mute); }
.pub-range { border: 1px solid var(--color-hairline); color: var(--color-body); border-radius: 999px; padding: 6px 12px; font-size: 11px; white-space: nowrap; }
.pub-overall { display: flex; align-items: center; gap: 18px; padding: 26px; border-radius: 22px; border: 1px solid var(--color-hairline); background: var(--color-canvas-raised); margin-bottom: 20px; }
.overall-icon { display: grid; place-items: center; width: 52px; height: 52px; border-radius: 17px; flex-shrink: 0; color: var(--color-mute); background: var(--color-canvas-soft-2); }
.pub-overall.ok .overall-icon { color: var(--color-ink); background: var(--color-canvas-soft-2); }
.pub-overall.warn .overall-icon { color: var(--color-warning); background: var(--color-status-degraded-bg); }
.pub-overall.down .overall-icon { color: var(--color-error); background: var(--color-status-down-bg); }
.pub-overall > div { flex: 1; min-width: 0; }
.pub-overall h2 { margin: 0 0 6px; font-size: 18px; font-weight: 600; overflow-wrap: anywhere; }
.pub-overall p { margin: 0; color: var(--color-mute); font-size: 12px; }
.overall-state { display: flex; align-items: center; gap: 7px; font-size: 12px; color: var(--color-body); white-space: nowrap; }
.pub-notice { display: flex; align-items: flex-start; gap: 16px; padding: 18px 22px; margin-bottom: 20px; border: 1px solid var(--color-hairline); border-radius: 17px; background: var(--color-canvas-raised); font-size: 13px; color: var(--color-body); line-height: 1.7; }
.pub-notice p { margin: 0; white-space: pre-wrap; overflow-wrap: anywhere; min-width: 0; }
.notice-label { font-size: 11px; color: var(--color-mute); white-space: nowrap; padding-top: 2px; }
.error-notice { align-items: center; background: var(--color-status-degraded-bg); color: var(--color-status-degraded-text); }
.error-notice > span { flex: 1; }
.text-button { border: 0; background: transparent; color: inherit; text-decoration: underline; cursor: pointer; padding: 6px; font-size: 12px; white-space: nowrap; }
.pub-metrics { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); border: 1px solid var(--color-hairline); background: var(--color-canvas-raised); border-radius: 20px; margin-bottom: 38px; padding: 22px 0; }
.pub-metrics > div { padding: 0 24px; border-left: 1px solid var(--color-hairline); }
.pub-metrics > div:first-child { border-left: 0; }
.pub-metrics span { display: block; color: var(--color-mute); font-size: 12px; margin-bottom: 10px; }
.pub-metrics strong { font-size: 29px; line-height: 1.1; font-weight: 500; font-variant-numeric: tabular-nums; }
.metric-ok { color: var(--color-ink); }.metric-issue { color: var(--color-error); }
.section-heading { display: flex; justify-content: space-between; align-items: center; gap: 12px; margin: 0 2px 16px; }
.section-heading h2 { font-size: 16px; font-weight: 600; margin: 0; }.section-heading > span { font-size: 11px; color: var(--color-mute); }
.pub-list { display: grid; gap: 14px; }
.svc { background: var(--color-canvas-raised); border: 1px solid var(--color-hairline); border-radius: 20px; padding: 22px 24px 18px; min-width: 0; }
.svc-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; margin-bottom: 16px; }.svc-title { gap: 9px; min-width: 0; }
.svc-title h3 { margin: 0; font-size: 15px; font-weight: 600; overflow-wrap: anywhere; }
.svc-link { display: grid; place-items: center; width: 28px; height: 28px; flex-shrink: 0; margin: -3px 0; border-radius: 8px; color: var(--color-mute); }.svc-link:hover { background: var(--color-canvas-soft-2); color: var(--color-focus); }
.svc-current { gap: 12px; font-size: 11px; color: var(--color-mute); white-space: nowrap; padding-top: 3px; }.current-ok { color: var(--color-ink); }.current-warn { color: var(--color-warning); }.current-down { color: var(--color-error); }
.lat { color: var(--color-mute); font-family: var(--font-mono); font-size: 11px; }
.svc-uptime { display: flex; align-items: baseline; flex-wrap: wrap; gap: 8px; margin-bottom: 17px; font-size: 11px; color: var(--color-mute); }.svc-uptime strong { color: var(--color-ink); font-size: 17px; font-weight: 500; font-variant-numeric: tabular-nums; }.svc-uptime > span:last-child { margin-left: auto; }
.bars-lg { display: flex; gap: 4px; height: 34px; margin-bottom: 9px; }.bar { min-width: 0; flex: 1; border-radius: 4px; border: 0; padding: 0; background: var(--color-canvas-soft-2); cursor: pointer; }.bar:hover, .bar:focus-visible { outline: 2px solid var(--color-ink); outline-offset: 2px; }
.b-ok, .d-ok { background: var(--color-ink); }.b-warn, .d-warn { background: var(--color-warning); }.b-down, .d-down { background: var(--color-error); }
.svc-foot { display: flex; align-items: center; justify-content: space-between; gap: 8px; color: var(--color-mute); font-size: 10px; font-variant-numeric: tabular-nums; }
.bar-detail { font-size: 11px; line-height: 1.7; margin: 13px 0 0; padding-top: 12px; border-top: 1px solid var(--color-hairline); color: var(--color-body); overflow-wrap: anywhere; }
.pub-legend { display: flex; justify-content: center; flex-wrap: wrap; gap: 18px; margin: 24px 0 10px; }.pub-legend > span { display: flex; align-items: center; gap: 6px; color: var(--color-mute); font-size: 11px; }.pub-legend i { width: 7px; height: 7px; border-radius: 2px; background: var(--color-canvas-soft-2); }.pub-legend i.d-ok { background: var(--color-ink); }.pub-legend i.d-warn { background: var(--color-warning); }.pub-legend i.d-down { background: var(--color-error); }
.history-note { font-size: 11px; color: var(--color-mute); text-align: center; line-height: 1.8; margin: 0; }
.pub-footer { display: flex; justify-content: space-between; flex-wrap: wrap; gap: 10px; color: var(--color-mute); border-top: 1px solid var(--color-hairline); font-size: 11px; padding-top: 22px; margin-top: 38px; }
.pub-empty { text-align: center; padding: 48px 24px; border: 1px dashed var(--color-hairline-strong); border-radius: 20px; color: var(--color-mute); }
.pub-error { text-align: center; padding: 70px 20px 100px; }.pub-error .pub-brand { margin: 0 auto 22px; color: var(--color-mute); }.pub-error h1 { font-size: 24px; font-weight: 600; margin: 0 0 12px; }.pub-error p { margin: 0 0 26px; color: var(--color-mute); }.error-actions { justify-content: center; gap: 10px; }
.pub-loading { min-height: 500px; }.pub-loading > p { text-align: center; color: var(--color-mute); font-size: 12px; }.loading-heading { display: flex; gap: 18px; align-items: center; margin-bottom: 28px; }.skeleton { background: var(--color-canvas-soft-2); border: 1px solid var(--color-hairline); border-radius: 20px; }.skeleton-icon { width: 60px; height: 60px; }.skeleton-title { width: min(260px, 55vw); height: 22px; margin-bottom: 12px; border-radius: 7px; }.skeleton-sub { width: min(180px, 40vw); height: 12px; border-radius: 5px; }.skeleton-banner { height: 106px; margin-bottom: 32px; }.loading-cards { display: grid; gap: 14px; }.skeleton-card { height: 146px; }
@media (max-width: 640px) {
  .pub-page { padding: 20px 16px 28px; }.pub-topbar { margin-bottom: 32px; }.pub-eyebrow { font-size: 11px; }.pub-button { padding: 0 12px; }.icon-button { padding: 0; }
  .pub-hero { gap: 13px; flex-wrap: wrap; margin-bottom: 22px; }.pub-brand { width: 50px; height: 50px; border-radius: 16px; }.pub-range { margin-left: 63px; margin-top: -8px; }.pub-sub { font-size: 11px; }
  .pub-overall { padding: 20px 18px; gap: 13px; }.overall-icon { width: 40px; height: 40px; border-radius: 13px; }.pub-overall h2 { font-size: 16px; }.pub-overall p { font-size: 11px; }.overall-state { display: none; }
  .pub-metrics { padding: 18px 0; margin-bottom: 30px; }.pub-metrics > div { padding: 0 12px; }.pub-metrics span { font-size: 10px; margin-bottom: 9px; }.pub-metrics strong { font-size: 24px; }
  .section-heading { align-items: flex-start; flex-direction: column; gap: 5px; }.svc { padding: 19px 16px 16px; }.svc-head { gap: 9px; }.svc-title h3 { font-size: 14px; }.svc-current { flex-direction: column; align-items: flex-end; gap: 3px; }.svc-uptime > span:last-child { font-size: 10px; }.bars-lg { gap: 2px; height: 30px; }.bar { border-radius: 2px; }.svc-foot { font-size: 9px; }
  .pub-notice { padding: 15px 16px; gap: 10px; }.pub-legend { gap: 13px; }.history-note { text-align: left; }.pub-footer { margin-top: 28px; }.pub-error { padding: 50px 0 70px; }
}
</style>
