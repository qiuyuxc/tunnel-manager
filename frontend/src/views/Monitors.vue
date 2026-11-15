<template>
  <div class="page-container monitors-page">
    <div class="head-actions"><RefreshButton :busy="loading" @refresh="load" /><button class="btn btn-primary" @click="openCreate"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M12 5v14M5 12h14"/></svg>{{ tr('m_bf2247cd1864') }}</button></div>

    <div class="monitor-metrics">
      <div><span>{{ tr('monitors.projects') }}</span><strong>{{ loaded ? monitors.length : '—' }}</strong></div>
      <div><span>{{ tr('monitors.targets') }}</span><strong>{{ loaded ? allTargets.length : '—' }}</strong><small>{{ tr('monitors.targetsHint') }}</small></div>
      <div><span>{{ tr('monitors.issues') }}</span><strong :class="{ 'has-issues': totals.counts.warn + totals.counts.down > 0 }">{{ loaded ? totals.counts.warn + totals.counts.down : '—' }}</strong><small>{{ tr('monitors.issuesHint') }}</small></div>
      <div><span>{{ tr('monitors.published') }}</span><strong>{{ loaded ? monitors.filter(m => m.publish_enabled).length : '—' }}</strong></div>
    </div>

    <div class="monitor-toolbar">
      <label class="monitor-search"><svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"><circle cx="10.5" cy="10.5" r="6.5"/><path d="m16 16 4 4"/></svg><input v-model="query" type="search" :placeholder="tr('monitors.search')" :aria-label="tr('monitors.search')" /></label>
      <div class="monitor-filters" role="group" :aria-label="tr('monitors.filter')"><button v-for="option in filters" :key="option" :class="{ selected: filter === option }" :aria-pressed="filter === option" @click="filter = option">{{ tr('monitors.filter.' + option) }}</button></div>
    </div>

    <div v-if="error && loaded" class="monitor-error-banner" role="alert"><span>{{ tr('monitors.refreshFailed') }} {{ error }}</span><button class="btn btn-secondary btn-sm" :disabled="loading" @click="load">{{ tr('status.retry') }}</button></div>
    <div v-if="!loaded && loading" class="monitor-loading" aria-busy="true" :aria-label="tr('m_9dc0825fba54')"><div v-for="n in 3" :key="n" class="loading-card"></div><p role="status">{{ tr('m_9dc0825fba54') }}</p></div>
    <div v-else-if="!loaded && error" class="empty-card" role="alert"><div class="empty-icon"><svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"><circle cx="12" cy="12" r="9"/><path d="M12 7v6m0 4h.01"/></svg></div><h3>{{ tr('monitors.loadFailed') }}</h3><p>{{ error }}</p><button class="btn btn-secondary" @click="load">{{ tr('status.retry') }}</button></div>
    <div v-else-if="monitors.length === 0" class="empty-card">
      <div class="empty-icon" aria-hidden="true"><svg width="30" height="30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="16" rx="5"/><path d="M6 12h3l2-4 3 8 2-4h2"/></svg></div>
      <h3>{{ tr('m_de10338046d1') }}</h3><p>{{ tr('m_fd4bea55e1ad') }}</p><button class="btn btn-primary" @click="openCreate">{{ tr('m_b466199f99b9') }}</button>
    </div>
    <div v-else-if="visibleMonitors.length === 0" class="empty-card"><h3>{{ tr('monitors.noMatches') }}</h3><p>{{ tr('monitors.noMatchesHint') }}</p><button class="btn btn-secondary" @click="query = ''; filter = 'all'">{{ tr('monitors.resetFilters') }}</button></div>
    <template v-else>
      <div class="list-heading"><span>{{ tr('monitors.resultCount', { count: visibleMonitors.length }) }}</span><span>{{ tr('monitors.manageHint') }}</span></div>
      <div class="monitor-list">
        <article v-for="{ monitor: m, summary } in visibleMonitors" :key="m.id" class="card monitor-item">
          <div class="mi-heading">
            <router-link class="mi-title" :to="'/monitors/' + encodeURIComponent(m.id)">
              <span class="monitor-symbol" :class="summary.state"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="16" rx="5"/><path d="M6 12h3l2-4 3 8 2-4h2"/></svg></span>
              <div><h3>{{ m.name }}</h3><span>{{ tr('monitors.targetCount', { count: m.targets.length }) }} · {{ tr('monitors.interval', { seconds: m.interval_sec }) }}</span></div>
            </router-link>
            <span class="monitor-state" :class="summary.state"><i></i>{{ tr('status.state.' + summary.state) }}</span>
          </div>
          <div class="mi-targets"><span v-for="target in m.targets.slice(0, 3)" :key="target.id" :title="target.name"><i :class="publicState(target.state)"></i>{{ target.name }}</span><span v-if="m.targets.length > 3" class="more-targets">+{{ m.targets.length - 3 }}</span><span v-if="!m.targets.length" class="target-empty">{{ tr('monitors.noTargets') }}</span></div>
          <div class="mi-states"><span><i class="ok"></i>{{ tr('status.state.ok') }} {{ summary.counts.ok }}</span><span><i class="warn"></i>{{ tr('status.state.warn') }} {{ summary.counts.warn }}</span><span><i class="down"></i>{{ tr('status.state.down') }} {{ summary.counts.down }}</span><span v-if="summary.counts.unknown"><i class="unknown"></i>{{ tr('status.state.unknown') }} {{ summary.counts.unknown }}</span></div>
          <footer class="mi-footer">
            <span class="public-badge" :class="{ published: m.publish_enabled }">{{ m.publish_enabled ? tr('m_9f1e25609b1a') : tr('m_abcd7f608563') }}</span>
            <div class="mi-actions"><router-link class="detail-link" :to="'/monitors/' + encodeURIComponent(m.id)">{{ tr('monitors.details') }}<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="m9 5 7 7-7 7"/></svg></router-link><button class="copy-public" :disabled="!m.publish_enabled || !(m.public_slug || m.public_token)" :title="m.publish_enabled ? tr('m_2596d7de6c7b') : tr('monitors.enablePublicFirst')" @click="copyPublic(m)"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="m10 13 4-4m-6 6-1 1a4 4 0 0 1-6-6l4-4a4 4 0 0 1 6 0m2 3 1-1a4 4 0 0 1 6 6l-4 4a4 4 0 0 1-6 0"/></svg><span>{{ tr('m_2596d7de6c7b') }}</span></button><button class="row-delete" :aria-label="tr('monitors.deleteNamed', { name: m.name })" :title="tr('m_b2fc749edfa3')" @click="deleting = m"><svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18M9 6V4h6v2M5 6l1 14h12l1-14M10 10v6m4-6v6"/></svg></button></div>
          </footer>
        </article>
      </div>
    </template>

    <SheetPanel size="medium" :open="createOpen" :title="tr('m_f9313095ee64')" @close="closeCreate">
      <form class="monitor-form" @submit.prevent="doCreate"><p class="form-intro">{{ tr('monitors.createHint') }}</p><label class="fld"><span>{{ tr('m_867698fb7b0c') }}</span><input v-model.trim="newName" maxlength="60" required autofocus :disabled="creating" :placeholder="tr('m_f34513a8e6f4')" /></label><p v-if="createError" class="form-error" role="alert">{{ createError }}</p><div class="dlg-actions"><button type="button" class="btn btn-secondary" :disabled="creating" @click="closeCreate">{{ tr('m_2cd0f3be8738') }}</button><button type="submit" class="btn btn-primary" :disabled="creating || !newName">{{ creating ? tr('m_56b63def0e29') : tr('m_253dbbfd9d0f') }}</button></div></form>
    </SheetPanel>
    <SheetPanel size="compact" :open="!!deleting" :title="tr('m_b2fc749edfa3')" @close="closeDelete"><template v-if="deleting"><p class="dlg-desc">{{ tr('m_fbc9178f86b7') }}<strong>{{ deleting.name }}</strong>{{ tr('m_1b54e6f260d4') }}</p><p v-if="deleteError" class="form-error" role="alert">{{ deleteError }}</p><div class="dlg-actions"><button class="btn btn-secondary" :disabled="busy" @click="closeDelete">{{ tr('m_2cd0f3be8738') }}</button><button class="btn btn-danger" :disabled="busy" @click="confirmDelete">{{ busy ? tr('m_5e8e7af57abb') : tr('m_a3ea3c17b401') }}</button></div></template></SheetPanel>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { t as tr } from '../i18n'
import { createMonitor, deleteMonitor, listMonitors, publicStatusPath, type MonitorView } from '../api'
import RefreshButton from '../components/RefreshButton.vue'
import SheetPanel from '../components/SheetPanel.vue'
import { filterMonitors, type MonitorFilter } from '../utils/monitorFilters'
import { publicState, statusSummary } from '../utils/publicStatus'

const router = useRouter()
const message = useMessage()
const monitors = ref<MonitorView[]>([])
const loaded = ref(false)
const loading = ref(false)
const error = ref('')
const query = ref('')
const filter = ref<MonitorFilter>('all')
const filters = ['all', 'issues', 'public'] as const
const allTargets = computed(() => monitors.value.flatMap(m => m.targets))
const totals = computed(() => statusSummary(allTargets.value))
const visibleMonitors = computed(() => filterMonitors(monitors.value, query.value, filter.value).map(monitor => ({ monitor, summary: statusSummary(monitor.targets) })))
const createOpen = ref(false)
const newName = ref('')
const creating = ref(false)
const createError = ref('')
const deleting = ref<MonitorView | null>(null)
const deleteError = ref('')
const busy = ref(false)
let alive = true
let controller: AbortController | undefined

function apiError(cause: unknown, fallback: string) {
  return (cause as { response?: { data?: { error?: string } } })?.response?.data?.error || fallback
}
async function load() {
  if (loading.value || !alive) return
  loading.value = true
  const request = new AbortController()
  controller = request
  try {
    const { data } = await listMonitors(request.signal)
    if (!alive || controller !== request) return
    monitors.value = data || []
    loaded.value = true
    error.value = ''
  } catch (cause) {
    if (alive && controller === request && !request.signal.aborted) error.value = apiError(cause, tr('monitors.connectionError'))
  } finally {
    if (controller === request) {
      if (alive) loading.value = false
      controller = undefined
    }
  }
}
function openCreate() { createError.value = ''; createOpen.value = true }
function closeCreate() { if (!creating.value) createOpen.value = false }
function closeDelete() { if (!busy.value) { deleting.value = null; deleteError.value = '' } }
async function doCreate() {
  if (!newName.value || creating.value) return
  creating.value = true
  createError.value = ''
  try {
    const { data } = await createMonitor(newName.value)
    if (!alive) return
    createOpen.value = false
    newName.value = ''
    message.success(tr('m_95a3e8f09bfc'))
    void router.push('/monitors/' + encodeURIComponent(data.id))
  } catch (cause) {
    if (alive) createError.value = apiError(cause, tr('m_7e6a71efbf63'))
  } finally { creating.value = false }
}
async function copyPublic(monitor: MonitorView) {
  const segment = monitor.public_slug || monitor.public_token
  if (!segment || !monitor.publish_enabled) return
  try {
    await navigator.clipboard.writeText(location.origin + publicStatusPath(segment))
    message.success(tr('m_ce710b698fbc'))
  } catch { message.error(tr('m_753d8bb0da99')) }
}
async function confirmDelete() {
  if (!deleting.value || busy.value) return
  busy.value = true
  deleteError.value = ''
  const id = deleting.value.id
  try {
    await deleteMonitor(id)
    if (!alive) return
    monitors.value = monitors.value.filter(m => m.id !== id)
    deleting.value = null
    message.success(tr('m_077a6d37719a'))
    // A list request started before deletion may still contain the removed
    // project. Cancel it before refreshing so it cannot restore a stale row.
    controller?.abort()
    controller = undefined
    loading.value = false
    void load()
  } catch (cause) {
    if (alive) deleteError.value = apiError(cause, tr('m_c228558cf257'))
  } finally { busy.value = false }
}
onMounted(load)
onBeforeUnmount(() => { alive = false; controller?.abort() })
</script>

<style scoped>
.head-actions { display: flex; justify-content: flex-end; align-items: center; flex-wrap: wrap; gap: 10px; margin-bottom: 20px; }
.monitor-metrics { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 14px; margin-bottom: 28px; }.monitor-metrics > div { background: var(--color-canvas-raised); border: 1px solid var(--color-hairline); border-radius: 20px; padding: 20px 22px; }.monitor-metrics span { display: block; color: var(--color-body); font-size: 12px; margin-bottom: 13px; }.monitor-metrics strong { display: block; font-size: 32px; font-weight: 500; line-height: 1.15; letter-spacing: -.03em; font-variant-numeric: tabular-nums; margin-bottom: 10px; }.monitor-metrics small { color: var(--color-mute); font-size: 11px; }.monitor-metrics .has-issues { color: var(--color-error); }
.monitor-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 15px 0 21px; border-bottom: 1px solid var(--color-hairline); margin-bottom: 23px; }.monitor-search { display: flex; align-items: center; gap: 9px; color: var(--color-mute); padding: 0 13px; height: 43px; max-width: 420px; flex: 1; border: 1px solid var(--color-hairline); border-radius: 12px; background: var(--color-canvas-raised); }.monitor-search:focus-within { border-color: var(--color-focus); }.monitor-search input { flex: 1; min-width: 0; width: 100%; border: 0; outline: 0; padding: 8px 0; background: transparent; color: var(--color-ink); font-size: 12px; }.monitor-search input::placeholder { color: var(--color-mute); }.monitor-filters { display: flex; gap: 4px; padding: 4px; background: var(--color-canvas-soft-2); border: 1px solid var(--color-hairline); border-radius: 12px; }.monitor-filters button { background: transparent; border: 1px solid transparent; border-radius: 8px; padding: 7px 14px; color: var(--color-mute); cursor: pointer; font-size: 12px; white-space: nowrap; }.monitor-filters button.selected { color: var(--color-ink); background: var(--color-canvas-raised); border-color: var(--color-hairline); }
.list-heading { display: flex; justify-content: space-between; gap: 14px; color: var(--color-mute); font-size: 11px; margin: 0 2px 14px; }.monitor-list { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px; }.monitor-item { padding: 23px 23px 0; min-width: 0; }.mi-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; margin-bottom: 20px; }.mi-title { display: flex; align-items: center; gap: 13px; min-width: 0; color: var(--color-ink); text-decoration: none; }.mi-title > div { min-width: 0; }.mi-title h3 { font-size: 17px; font-weight: 600; line-height: 1.4; margin: 0 0 6px; overflow-wrap: anywhere; }.mi-title:hover h3 { color: var(--color-focus); }.mi-title > div > span { color: var(--color-mute); font-size: 11px; }
.monitor-symbol { display: grid; place-items: center; width: 47px; height: 47px; border-radius: 15px; flex-shrink: 0; color: var(--color-mute); background: var(--color-canvas-soft-2); }.monitor-symbol.ok { color: var(--color-ink); background: var(--color-canvas-soft-2); }.monitor-symbol.warn { color: var(--color-warning); background: var(--color-status-degraded-bg); }.monitor-symbol.down { color: var(--color-error); background: var(--color-status-down-bg); }.monitor-state { display: inline-flex; align-items: center; gap: 6px; white-space: nowrap; color: var(--color-mute); font-size: 11px; padding-top: 4px; }.monitor-state i, .mi-targets i, .mi-states i { display: inline-block; width: 6px; height: 6px; border-radius: 50%; background: var(--color-mute); flex-shrink: 0; }.monitor-state.ok i, i.ok { background: var(--color-ink); }.monitor-state.warn i, i.warn { background: var(--color-warning); }.monitor-state.down i, i.down { background: var(--color-error); }
.mi-targets { display: flex; align-items: center; flex-wrap: wrap; gap: 7px; min-height: 29px; margin-bottom: 17px; }.mi-targets > span { display: inline-flex; align-items: center; gap: 6px; background: var(--color-canvas-soft); border: 1px solid var(--color-hairline); border-radius: 8px; padding: 5px 9px; color: var(--color-body); font-size: 11px; min-width: 0; max-width: 100%; overflow-wrap: anywhere; }.mi-targets .more-targets, .mi-targets .target-empty { color: var(--color-mute); background: transparent; border-color: transparent; padding-left: 0; }
.mi-states { display: flex; flex-wrap: wrap; gap: 14px; color: var(--color-mute); font-size: 11px; padding-bottom: 20px; }.mi-states > span { display: inline-flex; align-items: center; gap: 5px; }.mi-footer { border-top: 1px solid var(--color-hairline); margin: 0 -23px; padding: 12px 16px 12px 23px; display: flex; align-items: center; justify-content: space-between; gap: 10px; }.public-badge { color: var(--color-mute); font-size: 10px; white-space: nowrap; }.public-badge.published { color: var(--color-ink); }.mi-actions { display: flex; align-items: center; gap: 6px; }.detail-link { display: inline-flex; align-items: center; gap: 3px; color: var(--color-body); font-size: 11px; min-height: 38px; padding: 0 5px; text-decoration: none; }.detail-link:hover { color: var(--color-focus); }.copy-public { display: inline-flex; align-items: center; justify-content: center; gap: 6px; min-height: 38px; border: 1px solid var(--color-hairline); border-radius: 10px; color: var(--color-body); background: transparent; font-size: 11px; padding: 0 10px; cursor: pointer; }.copy-public:disabled { color: var(--color-mute); opacity: .55; cursor: default; }.copy-public:hover:not(:disabled) { background: var(--color-canvas-soft-2); }.row-delete { display: grid; place-items: center; width: 40px; height: 40px; border: 1px solid transparent; border-radius: 11px; background: transparent; color: var(--color-mute); cursor: pointer; }.row-delete:hover { color: var(--color-error); background: var(--color-status-down-bg); }
.empty-card { border: 1px dashed var(--color-hairline-strong); border-radius: 22px; background: var(--color-canvas-raised); padding: 58px 24px; text-align: center; display: flex; flex-direction: column; align-items: center; gap: 12px; }.empty-icon { width: 60px; height: 60px; display: grid; place-items: center; color: var(--color-mute); border-radius: 19px; background: var(--color-canvas-soft-2); margin-bottom: 7px; }.empty-card h3 { font-size: 17px; margin: 0; font-weight: 600; }.empty-card p { font-size: 13px; margin: 0 0 10px; max-width: 430px; color: var(--color-mute); line-height: 1.7; }.monitor-error-banner { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 16px 20px; border: 1px solid var(--color-status-degraded-border); border-radius: 14px; color: var(--color-status-degraded-text); background: var(--color-status-degraded-bg); font-size: 12px; margin-bottom: 18px; }.monitor-loading { display: grid; gap: 14px; }.loading-card { height: 150px; border: 1px solid var(--color-hairline); background: var(--color-canvas-soft-2); border-radius: 20px; }.monitor-loading p { text-align: center; color: var(--color-mute); font-size: 12px; }
.monitor-form { padding-top: 3px; }.form-intro, .dlg-desc { margin: 2px 0 22px; color: var(--color-body); font-size: 13px; line-height: 1.8; overflow-wrap: anywhere; }.fld { display: flex; flex-direction: column; gap: 9px; font-size: 12px; color: var(--color-body); margin-bottom: 22px; }.fld input { min-height: 43px; border-radius: 12px; border: 1px solid var(--color-hairline-strong); background: var(--color-canvas-soft); color: var(--color-ink); padding: 0 13px; font-size: 13px; }.form-error { color: var(--color-error); font-size: 12px; line-height: 1.7; margin: -5px 0 16px; }.dlg-actions { display: flex; justify-content: flex-end; gap: 10px; }
@media (max-width: 1180px) { .monitor-list { grid-template-columns: minmax(0, 1fr); }.monitor-metrics > div { padding: 18px 16px; } }
@media (max-width: 640px) { .head-actions { width: 100%; justify-content: flex-end; }.head-actions > .btn { flex: 1; }.monitor-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; margin-bottom: 20px; }.monitor-metrics > div { padding: 17px; border-radius: 17px; }.monitor-metrics strong { font-size: 28px; }.monitor-toolbar { flex-direction: column; align-items: stretch; gap: 12px; padding-top: 0; }.monitor-search { max-width: none; flex: none; }.monitor-filters { align-self: flex-start; }.monitor-filters button { padding: 7px 16px; }.list-heading > span:last-child { display: none; }.monitor-item { padding: 19px 17px 0; }.mi-title { gap: 10px; }.mi-title h3 { font-size: 16px; }.monitor-symbol { width: 40px; height: 40px; border-radius: 13px; }.mi-footer { margin: 0 -17px; padding: 10px 10px 10px 17px; }.copy-public span { display: none; }.copy-public { min-width: 38px; padding: 0 10px; }.mi-states { gap: 10px; }.monitor-error-banner { align-items: flex-start; padding: 14px; }.empty-card { padding: 44px 18px; } }
</style>
