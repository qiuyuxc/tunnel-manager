<template>
  <div class="page-container">
    <div class="page-heading-row section">
      <div class="page-header">
        <h2>{{ tr('m_e920f47d0c05') }}</h2>
        <p>{{ tr('m_c7a5c63e5953') }}</p>
      </div>
      <div class="toolbar">
        <button class="btn btn-secondary" :disabled="loading" @click="loadTunnels">
          <svg :class="{ spin: loading }" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
          <span>{{ loading ? tr('m_3af7e5489e61') : tr('m_aee887434131') }}</span>
        </button>
        <button class="btn btn-primary" @click="startCreate">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          <span>{{ tr('m_ee6527b22aaf') }}</span>
        </button>
      </div>
    </div>
    <div class="card section tunnel-table-card">
      <div v-if="loading" class="empty-state">
        <svg class="spin" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="var(--color-mute)" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
        <span class="empty-text">{{ tr('m_9dc0825fba54') }}</span>
      </div>
      <div v-else-if="tunnels.length > 0" class="table-scroll">
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
            <tr v-for="tunnel in tunnels" :key="tunnel.id" :class="{ 'row-selected': config.tunnel_id === tunnel.id }">
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
        <span class="empty-text">{{ tr('m_6413e61b6bc7') }}</span>
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

import { computed, ref, onMounted } from 'vue'
import { useMessage, useDialog, NModal } from 'naive-ui'
import { listTunnels, setTunnelSelection, createTunnel, deleteTunnel, type Tunnel, type CreateTunnelResponse } from '../api'
import { useConfigStore } from '../stores/config'
const message = useMessage()
const dialog = useDialog()
const configStore = useConfigStore()
const config = configStore.config
const tunnels = ref<Tunnel[]>([])
const loading = ref(false)
const listVisible = ref(false)
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
async function loadTunnels() {
  loading.value = true
  listVisible.value = false
  try {
    const { data } = await listTunnels()
    tunnels.value = data
    requestAnimationFrame(() => {
      requestAnimationFrame(() => { listVisible.value = true })
    })
  } catch (e: any) {
    message.error(tr('m_6779e3e1d2f2') + (e.response?.data?.error || e.message))
  } finally {
    loading.value = false
  }
}
async function selectTunnel(tunnel: Tunnel) {
  try {
    await setTunnelSelection(tunnel.id, tunnel.name)
    config.tunnel_id = tunnel.id
    config.tunnel_name = tunnel.name
    message.success(tr('m_6e2b7249dc7c'))
  } catch (e: any) {
    message.error(tr('m_a6c42e79dfd2') + (e.response?.data?.error || e.message))
  }
}
async function clearTunnel() {
  try {
    await setTunnelSelection('', '')
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
  const name = createName.value.trim()
  if (!name || creating.value) return
  creating.value = true
  try {
    const { data } = await createTunnel(name)
    created.value = data
    showCreate.value = false
    showInstall.value = false
    showToken.value = true
    await loadTunnels()
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
      try {
        await deleteTunnel(tunnel.id)
        if (config.tunnel_id === tunnel.id) {
          config.tunnel_id = ''
          config.tunnel_name = ''
        }
        message.success(tr('m_72d83d119709'))
        await loadTunnels()
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
onMounted(() => { loadTunnels() })
</script>
<style scoped>
.page-heading-row { display: flex; align-items: flex-end; justify-content: space-between; gap: var(--spacing-xl); }
/* The heading sits in a row with the toolbar, so it drops the shared bottom margin. */
.page-header { margin-bottom: 0; }
.toolbar { display: flex; gap: var(--spacing-sm); flex-shrink: 0; }
.tunnel-table-card { overflow: hidden; box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08); }
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
.selected-mark { display: inline-flex; align-items: center; gap: 6px; color: var(--color-status-healthy-text); font-size: 12px; font-weight: 600; white-space: nowrap; }
.selected-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--color-success); box-shadow: 0 0 0 3px var(--color-status-healthy-bg); }
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
.page-heading-row { align-items: stretch; flex-direction: column; gap: var(--spacing-lg); }
.toolbar { justify-content: flex-end; }
.table-scroll { overflow: visible; }
.tunnel-table-card { overflow: visible; border: 0; background: transparent; box-shadow: none; }
.data-table { display: block; min-width: 0; }
.data-table colgroup, .data-table thead { display: none; }
.data-table tbody { display: grid; gap: var(--spacing-md); }
.data-table tr { display: grid; overflow: hidden; border: 1px solid var(--color-hairline); border-radius: var(--radius-lg); box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08); }
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
