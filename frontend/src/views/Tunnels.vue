<template>
  <div class="page-container">
    <div class="page-heading-row section">
      <div class="page-header">
        <h2>隧道管理</h2>
        <p>浏览 Cloudflare Tunnel，并锁定当前要管理的隧道</p>
      </div>
      <div class="toolbar">
        <button class="btn btn-secondary" :disabled="loading" @click="loadTunnels">
          <svg :class="{ spin: loading }" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>
          <span>{{ loading ? '刷新中' : '刷新' }}</span>
        </button>
        <button class="btn btn-primary" @click="startCreate">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          <span>新建隧道</span>
        </button>
      </div>
    </div>
    <div class="card section tunnel-table-card">
      <div v-if="loading" class="empty-state">
        <svg class="spin" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="var(--color-mute)" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
        <span class="empty-text">加载中...</span>
      </div>
      <div v-else-if="tunnels.length > 0" class="table-scroll">
        <table class="data-table" aria-label="Cloudflare 隧道列表">
          <colgroup>
            <col class="col-name" />
            <col class="col-id" />
            <col class="col-status" />
            <col class="col-selection" />
            <col class="col-actions" />
          </colgroup>
          <thead>
            <tr>
              <th>名称</th>
              <th>隧道 ID</th>
              <th>状态</th>
              <th>当前选择</th>
              <th class="action-column">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="tunnel in tunnels" :key="tunnel.id" :class="{ 'row-selected': config.tunnel_id === tunnel.id }">
              <td class="cell-name" data-label="名称">{{ tunnel.name }}</td>
              <td data-label="隧道 ID"><code class="tunnel-id-code">{{ tunnel.id }}</code></td>
              <td data-label="状态"><span class="status-tag" :class="tunnel.status">{{ tunnel.status }}</span></td>
              <td data-label="当前选择">
                <span v-if="config.tunnel_id === tunnel.id" class="selected-mark"><span class="selected-dot"></span>已选中</span>
                <span v-else class="text-muted">未选择</span>
              </td>
              <td class="action-column" data-label="操作">
                <div class="row-actions">
                  <router-link :to="`/tunnels/${tunnel.id}`" class="btn btn-secondary btn-sm">详情</router-link>
                  <button class="btn btn-primary btn-sm" :disabled="config.tunnel_id === tunnel.id" @click="selectTunnel(tunnel)">
                    {{ config.tunnel_id === tunnel.id ? '已选中' : '选择' }}
                  </button>
                  <button class="btn btn-ghost btn-sm btn-text-danger" @click="confirmDelete(tunnel)">删除</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="empty-state">
        <span class="empty-text">暂无隧道</span>
        <span class="text-muted">点击"新建隧道"创建第一条隧道</span>
      </div>
    </div>
    <n-modal v-model:show="showCreate" preset="card" title="新建隧道" class="tunnel-modal" :bordered="false">
      <div class="modal-form">
        <label class="form-label">隧道名称</label>
        <input v-model="createName" placeholder="例如：prod-tunnel" class="vercel-input" @keyup.enter="submitCreate" />
      </div>
      <template #footer>
        <div class="modal-footer">
          <button class="btn btn-ghost" @click="showCreate = false">取消</button>
          <button class="btn btn-primary" :disabled="creating || !createName.trim()" @click="submitCreate">
            {{ creating ? '创建中...' : '创建' }}
          </button>
        </div>
      </template>
    </n-modal>
    <n-modal v-model:show="showToken" preset="card" title="隧道创建成功" class="tunnel-modal" :bordered="false">
      <div class="modal-form">
        <p class="modal-hint">保存以下连接令牌，或复制运行命令到安装了 cloudflared 的服务器执行。</p>
        <div class="copy-row">
          <code class="token-box">{{ created?.run_command || created?.token }}</code>
          <button class="btn btn-secondary btn-sm" @click="copyText(created?.run_command || created?.token || '')">复制</button>
        </div>
        <div v-if="created?.warning" class="token-warning">{{ created.warning }}</div>
        <div class="install-guide">
          <button class="install-toggle" @click="showInstall = !showInstall">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>
            查看 cloudflared 安装脚本
          </button>
          <pre v-if="showInstall" class="token-box install-script">{{ installScript }}</pre>
        </div>
      </div>
      <template #footer>
        <div class="modal-footer">
          <button class="btn btn-primary" @click="showToken = false">完成</button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
<script setup lang="ts">
import { ref, onMounted } from 'vue'
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
const installScript = `# 添加 Cloudflare GPG 密钥
sudo mkdir -p --mode=0755 /usr/share/keyrings
curl -fsSL https://pkg.cloudflare.com/cloudflare-public-v2.gpg | sudo tee /usr/share/keyrings/cloudflare-public-v2.gpg >/dev/null
# 添加 apt 软件源
echo 'deb [signed-by=/usr/share/keyrings/cloudflare-public-v2.gpg] https://pkg.cloudflare.com/cloudflared any main' | sudo tee /etc/apt/sources.list.d/cloudflared.list
# 安装 cloudflared
sudo apt-get update && sudo apt-get install cloudflared`
async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    message.success('已复制')
  } catch {
    message.warning('复制失败，请手动选择文本')
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
    message.error('获取隧道列表失败: ' + (e.response?.data?.error || e.message))
  } finally {
    loading.value = false
  }
}
async function selectTunnel(tunnel: Tunnel) {
  try {
    await setTunnelSelection(tunnel.id, tunnel.name)
    config.tunnel_id = tunnel.id
    config.tunnel_name = tunnel.name
    message.success('隧道已锁定')
  } catch (e: any) {
    message.error('选择失败: ' + (e.response?.data?.error || e.message))
  }
}
async function clearTunnel() {
  try {
    await setTunnelSelection('', '')
    config.tunnel_id = ''
    config.tunnel_name = ''
    message.success('已清除隧道选择')
  } catch (e: any) {
    message.error('清除失败')
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
    message.error('创建失败: ' + (e.response?.data?.error || e.message))
  } finally {
    creating.value = false
  }
}
function confirmDelete(tunnel: Tunnel) {
  dialog.warning({
    title: '删除隧道',
    content: `确定删除隧道「${tunnel.name}」吗？此操作无法撤销。若该隧道仍有活动连接，Cloudflare 会拒绝删除。`,
    positiveText: '确认删除',
    negativeText: '取消',
    async onPositiveClick() {
      try {
        await deleteTunnel(tunnel.id)
        if (config.tunnel_id === tunnel.id) {
          config.tunnel_id = ''
          config.tunnel_name = ''
        }
        message.success('隧道已删除')
        await loadTunnels()
      } catch (e: any) {
        message.error('删除失败: ' + (e.response?.data?.error || e.message))
        return false
      }
    },
  })
}
async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    message.success('已复制')
  } catch {
    message.warning('复制失败，请手动选择文本')
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