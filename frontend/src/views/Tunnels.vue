<template>
  <div class="page-container">
    <div class="toolbar section">
      <RefreshButton :busy="loading" @refresh="loadTunnels" />
      <button class="btn btn-primary" @click="startCreate">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
        <span>{{ tr('m_ee6527b22aaf') }}</span>
      </button>
    </div>
    <div class="tunnel-summary section">
      <div><span>{{ tr('console.tunnels_total') }}</span><strong>{{ tunnels.length }}</strong></div>
      <div><span>{{ tr('console.tunnels_active') }}</span><strong>{{ tunnels.filter(t => t.status === 'healthy').length }}</strong></div>
      <div><span>{{ tr('m_b903d203f70a') }}</span><strong class="selection-name">{{ config.tunnel_name || tr('m_80a57e03f071') }}</strong></div>
    </div>
    <div v-if="loadError" class="load-notice" role="status">{{ loadError }}</div>
    <div class="tunnel-filter section">
      <label class="search-field"><span>{{ tr('console.tunnel_search') }}</span><input v-model="search" class="vercel-input" :placeholder="tr('console.tunnel_search_placeholder')" /></label>
      <span class="filter-count">{{ filteredTunnels.length }} / {{ tunnels.length }}</span>
    </div>
    <div class="card section tunnel-table-card" :aria-busy="loading">
      <div v-if="loading && !loaded" class="empty-state">
        <svg class="spin" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="var(--color-mute)" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
        <span class="empty-text">{{ tr('m_9dc0825fba54') }}</span>
      </div>
      <div v-else-if="filteredTunnels.length > 0" class="table-scroll">
        <table class="data-table" :aria-label="tr('m_caebfe1e68a0')">
          <colgroup>
            <col class="col-name" />
            <col class="col-id" />
            <col class="col-status" />
            <col class="col-selection" />
            <col class="col-actions" />
          </colgroup>
          <thead>
            <tr>
              <th>{{ tr('m_d44e9b3d3b31') }}</th>
              <th>{{ tr('m_17f261a9385c') }}</th>
              <th>{{ tr('m_6320b4a8722a') }}</th>
              <th>{{ tr('m_8d30e0eb426f') }}</th>
              <th class="action-column">{{ tr('m_ed31fbb483ee') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="tunnel in filteredTunnels" :key="tunnel.id" :class="{ 'row-selected': config.tunnel_id === tunnel.id }">
              <td class="cell-name" :data-label="tr('m_d44e9b3d3b31')">{{ tunnel.name }}</td>
              <td :data-label="tr('m_17f261a9385c')"><code class="tunnel-id-code">{{ tunnel.id }}</code></td>
              <td :data-label="tr('m_6320b4a8722a')"><span class="status-tag" :class="tunnel.status">{{ tunnel.status }}</span></td>
              <td :data-label="tr('m_8d30e0eb426f')">
                <span v-if="config.tunnel_id === tunnel.id" class="selected-mark"><span class="selected-dot"></span>{{ tr('m_a737fd72b016') }}</span>
                <span v-else class="text-muted">{{ tr('m_1182c5454f11') }}</span>
              </td>
              <td class="action-column" :data-label="tr('m_ed31fbb483ee')">
                <div class="row-actions">
                  <router-link :to="`/tunnels/${tunnel.id}`" class="btn btn-secondary btn-sm">{{ tr('m_979a332955c8') }}</router-link>
                  <button class="btn btn-primary btn-sm" :disabled="config.tunnel_id === tunnel.id" @click="selectTunnel(tunnel)">
                    {{ config.tunnel_id === tunnel.id ? tr('m_a737fd72b016') : tr('m_c11330b85234') }}
                  </button>
                  <button class="btn btn-ghost btn-sm btn-text-danger" @click="confirmDelete(tunnel)">{{ tr('m_2f9daa828907') }}</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="empty-state">
        <span class="empty-text">{{ loadError ? tr('console.load_failed') : search ? tr('console.no_matches') : tr('m_6413e61b6bc7') }}</span>
        <span class="text-muted">{{ tr('m_2efdc680cddf') }}</span>
      </div>
    </div>
    <n-modal v-model:show="showCreate" preset="card" :title="tr('m_ee6527b22aaf')" class="tunnel-modal" :bordered="false">
      <div class="modal-form">
        <label class="form-label">{{ tr('m_b679939ed787') }}</label>
        <input v-model="createName" :placeholder="tr('m_d03538e23717')" class="vercel-input" @keyup.enter="submitCreate" />
      </div>
      <template #footer>
        <div class="modal-footer">
          <button class="btn btn-ghost" @click="showCreate = false">{{ tr('m_2cd0f3be8738') }}</button>
          <button class="btn btn-primary" :disabled="creating || !createName.trim()" @click="submitCreate">
            {{ creating ? tr('m_687d8f3ba99e') : tr('m_cde2cd071d25') }}
          </button>
        </div>
      </template>
    </n-modal>
    <n-modal v-model:show="showToken" preset="card" :title="tr('m_93d9bdf907c6')" class="tunnel-modal" :bordered="false">
      <div class="modal-form">
        <p class="modal-hint">{{ tr('m_77e218f54885') }}</p>
        <div class="copy-row">
          <code class="token-box">{{ created?.run_command || created?.token }}</code>
          <button class="btn btn-secondary btn-sm" @click="copyText(created?.run_command || created?.token || '')">{{ tr('m_63d90d977348') }}</button>
        </div>
        <div v-if="created?.warning" class="token-warning">{{ created.warning }}</div>
        <div class="install-guide">
          <button class="install-toggle" @click="showInstall = !showInstall">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>
            {{ tr('m_9c685c211f33') }}
          </button>
          <pre v-if="showInstall" class="token-box install-script">{{ installScript }}</pre>
        </div>
      </div>
      <template #footer>
        <div class="modal-footer">
          <button class="btn btn-primary" @click="showToken = false">{{ tr('m_c0b3fbff51cc') }}</button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
<script setup lang="ts">
import { t as tr } from '../i18n'

import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import RefreshButton from '../components/RefreshButton.vue'
import { createRefreshTask } from '../utils/refreshTask'
import { useMessage, useDialog, NModal } from 'naive-ui'
import { listTunnels, setTunnelSelection, createTunnel, deleteTunnel, type Tunnel, type CreateTunnelResponse } from '../api'
import { useConfigStore } from '../stores/config'
const message = useMessage()
const dialog = useDialog()
const configStore = useConfigStore()
const config = configStore.config
const tunnels = ref<Tunnel[]>([])
const loading = ref(false)
const search = ref('')
const loaded = ref(false)
const loadError = ref('')
let disposed = false
const filteredTunnels = computed(() => {
  const query = search.value.trim().toLowerCase()
  return tunnels.value.filter(tunnel => !query || [tunnel.name, tunnel.id, tunnel.status].some(value => value.toLowerCase().includes(query)))
})
// Create / token state
const showCreate = ref(false)
const creating = ref(false)
const createName = ref('')
const showToken = ref(false)
const created = ref<CreateTunnelResponse | null>(null)
const showInstall = ref(false)
const installScript = computed(() => tr('m_7a3234e91a76'))
async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    message.success(tr('m_8f6f8d979c98'))
  } catch {
    message.warning(tr('m_03ae6cf8db7c'))
  }
}
const loadTunnels = createRefreshTask(async (isCurrent, signal) => {
  loading.value = true
  try {
    const { data } = await listTunnels(signal)
    if (!isCurrent()) return
    tunnels.value = data
    loaded.value = true
    loadError.value = ''
  } catch (e: any) {
    if (isCurrent()) loadError.value = tr('m_6779e3e1d2f2') + (e.response?.data?.error || e.message)
  } finally {
    loading.value = false
  }
})
async function selectTunnel(tunnel: Tunnel) {
  if (disposed) return
  const session = configStore.token
  try {
    await setTunnelSelection(tunnel.id, tunnel.name)
    if (disposed || configStore.token !== session) return
    configStore.invalidateConfig()
    config.tunnel_id = tunnel.id
    config.tunnel_name = tunnel.name
    message.success(tr('m_6e2b7249dc7c'))
  } catch (e: any) {
    message.error(tr('m_a6c42e79dfd2') + (e.response?.data?.error || e.message))
  }
}
async function clearTunnel() {
  if (disposed) return
  const session = configStore.token
  try {
    await setTunnelSelection('', '')
    if (disposed || configStore.token !== session) return
    configStore.invalidateConfig()
    config.tunnel_id = ''
    config.tunnel_name = ''
    message.success(tr('m_58be968d5567'))
  } catch (e: any) {
    message.error(tr('m_2952f4853185'))
  }
}
function startCreate() {
  createName.value = ''
  showCreate.value = true
}
async function submitCreate() {
  if (disposed) return
  const session = configStore.token
  const name = createName.value.trim()
  if (!name || creating.value) return
  creating.value = true
  try {
    const { data } = await createTunnel(name)
    if (disposed || configStore.token !== session) return
    created.value = data
    showCreate.value = false
    showInstall.value = false
    showToken.value = true
    await loadTunnels(true)
  } catch (e: any) {
    message.error(tr('m_cdcc0d62ab79') + (e.response?.data?.error || e.message))
  } finally {
    creating.value = false
  }
}
function confirmDelete(tunnel: Tunnel) {
  dialog.warning({
    title: tr('m_cf5e530b1ad3'),
    content: tr('m_f6412c0cf213', { p0: tunnel.name }),
    positiveText: tr('m_a3ea3c17b401'),
    negativeText: tr('m_2cd0f3be8738'),
    async onPositiveClick() {
      if (disposed) return false
      const session = configStore.token
      try {
        await deleteTunnel(tunnel.id)
        if (disposed || configStore.token !== session) return
        configStore.invalidateConfig()
        if (config.tunnel_id === tunnel.id) {
          config.tunnel_id = ''
          config.tunnel_name = ''
        }
        message.success(tr('m_72d83d119709'))
        await loadTunnels(true)
      } catch (e: any) {
        message.error(tr('m_48477dd59e05') + (e.response?.data?.error || e.message))
        return false
      }
    },
  })
}
async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    message.success(tr('m_8f6f8d979c98'))
  } catch {
    message.warning(tr('m_03ae6cf8db7c'))
  }
}
onMounted(() => { void configStore.fetchConfig(); void loadTunnels() })
onBeforeUnmount(() => { disposed = true; loadTunnels.dispose() })
</script>
<style scoped>
.tunnel-summary { display: grid; grid-template-columns: 1fr 1fr 2fr; overflow: hidden; border: 1px solid var(--color-hairline); border-radius: 18px; background: var(--color-canvas-raised); }
.tunnel-summary > div { display: flex; flex-direction: column; gap: 12px; min-width: 0; padding: 20px 24px; }
.tunnel-summary > div + div { border-left: 1px solid var(--color-hairline); }
.tunnel-summary span { color: var(--color-mute); font-size: 12px; }
.tunnel-summary strong { color: var(--color-ink); font-size: 24px; font-weight: 600; font-variant-numeric: tabular-nums; }
.tunnel-summary .selection-name { font-size: 18px; overflow-wrap: anywhere; }
.tunnel-filter { display: flex; align-items: flex-end; justify-content: space-between; gap: 16px; }
.search-field { display: flex; flex-direction: column; gap: 8px; width: min(100%, 380px); color: var(--color-mute); font-size: 12px; }
.filter-count { flex-shrink: 0; padding-bottom: 12px; color: var(--color-mute); font: 12px var(--font-mono); }
@media (max-width: 640px) { .tunnel-summary { grid-template-columns: 1fr 1fr; } .tunnel-summary > div:last-child { grid-column: 1 / -1; border-left: 0; border-top: 1px solid var(--color-hairline); } .tunnel-summary > div { padding: 18px; } }

.toolbar { display: flex; align-items: center; justify-content: flex-end; flex-wrap: wrap; gap: var(--spacing-sm); }
.tunnel-table-card { overflow: hidden; box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08); }
.table-scroll { overflow-x: auto; }
.data-table { width: 100%; min-width: 920px; border-collapse: collapse; table-layout: fixed; font-size: 13px; }
.col-name { width: 18%; }
.col-id { width: auto; }
.col-status { width: 112px; }
.col-selection { width: 126px; }
.col-actions { width: 214px; }
.data-table th, .data-table td { padding: 13px 16px; text-align: left; vertical-align: middle; border-bottom: 1px solid var(--color-hairline); }
.data-table th { background: var(--color-canvas-soft); color: var(--color-mute); font-size: 12px; font-weight: 600; letter-spacing: 0.02em; }
.data-table tbody tr { background: var(--color-canvas-raised); transition: background-color 140ms ease-out; }
.data-table tbody tr:hover { background: var(--color-canvas-soft); }
.data-table tbody tr.row-selected { background: color-mix(in srgb, var(--color-link) 5%, var(--color-canvas-raised)); }
.data-table tbody tr:last-child td { border-bottom: 0; }
.cell-name { color: var(--color-ink); font-size: 14px; font-weight: 600; overflow-wrap: anywhere; }
.tunnel-id-code { display: inline-block; max-width: 100%; overflow: hidden; color: var(--color-body); font-family: var(--font-mono); font-size: 12px; line-height: 1.4; text-overflow: ellipsis; white-space: nowrap; }
.selected-mark { display: inline-flex; align-items: center; gap: 6px; color: var(--color-ink); font-size: 12px; font-weight: 600; white-space: nowrap; }
.selected-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--color-ink);  }
.text-muted { color: var(--color-mute); font-size: 12px; }
.action-column { text-align: right !important; }
.row-actions { display: flex; align-items: center; justify-content: flex-end; gap: 6px; white-space: nowrap; }
.btn-text-danger:hover:not(:disabled) { color: var(--color-error); background: var(--color-status-down-bg); }
.modal-form { display: flex; flex-direction: column; gap: var(--spacing-md); }
.form-label { font-size: 12px; color: var(--color-mute); font-weight: 500; }
.modal-footer { display: flex; justify-content: flex-end; gap: 10px; }
.modal-hint { margin: 0; font-size: 13px; color: var(--color-body); line-height: 1.7; }
.copy-row { display: flex; align-items: flex-start; gap: 8px; }
.token-box { flex: 1; min-width: 0; max-height: 160px; overflow-y: auto; padding: 10px 12px; color: var(--color-ink); background: var(--color-canvas-soft); border: 1px solid var(--color-hairline); border-radius: var(--radius-md); font-family: var(--font-mono); font-size: 12px; line-height: 1.5; word-break: break-all; }
.token-warning { padding: 10px 12px; color: var(--color-error); background: var(--color-status-down-bg); border: 1px solid var(--color-status-down-border); border-radius: var(--radius-md); font-size: 13px; }
.install-toggle { display: inline-flex; align-items: center; gap: 6px; padding: 0; color: var(--color-link); background: none; border: 0; font-size: 13px; cursor: pointer; }
.install-script { max-height: 260px; white-space: pre-wrap; }
.tunnel-modal { width: 100%; max-width: 520px; }
@media (max-width: 768px) {
.table-scroll { overflow: visible; }
.tunnel-table-card { overflow: visible; border: 0; background: transparent; box-shadow: none; }
.data-table { display: block; min-width: 0; }
.data-table colgroup, .data-table thead { display: none; }
.data-table tbody { display: grid; gap: var(--spacing-md); }
.data-table tr { display: grid; overflow: hidden; border: 1px solid var(--color-hairline); border-radius: var(--radius-lg); box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08); }
.data-table td { display: grid; grid-template-columns: 88px minmax(0, 1fr); gap: var(--spacing-md); padding: 11px 14px; border-bottom: 1px solid var(--color-hairline); }
.data-table td::before { content: attr(data-label); color: var(--color-mute); font-size: 12px; font-weight: 500; }
.data-table .action-column { display: block; padding: 12px 14px; text-align: left !important; }
.data-table .action-column::before { display: none; }
.tunnel-id-code { white-space: normal; overflow-wrap: anywhere; }
.row-actions { flex-wrap: wrap; }
.copy-row { flex-direction: column; }
.copy-row .btn { align-self: flex-end; }
}
</style>
