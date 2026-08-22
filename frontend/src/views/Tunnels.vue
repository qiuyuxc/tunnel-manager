<template>
  <div class="page-container">
    <div class="page-header">
      <router-link to="/" class="back-link">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
        返回控制面板
      </router-link>
      <h2>隧道管理</h2>
      <p>浏览并选择您的 Cloudflare Tunnel</p>
    </div>

    <div class="selection-card section">
      <div class="selection-header">
        <span class="caption-mono selection-label">当前已选隧道</span>
      </div>
      <div v-if="config.tunnel_id" class="selection-body">
        <div class="selected-tunnel">
          <strong>{{ config.tunnel_name || '当前隧道' }}</strong>
          <code class="tunnel-id">{{ config.tunnel_id }}</code>
        </div>
        <button class="btn-ghost-sm" @click="clearTunnel">清除</button>
      </div>
      <div v-else class="selection-empty">
        <span>未选择隧道</span>
      </div>
    </div>

    <div class="toolbar section">
      <button class="btn btn-primary" :disabled="loading" @click="loadTunnels">
        <svg v-if="loading" class="spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
        <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
        刷新列表
      </button>
      <button class="btn btn-secondary" @click="startCreate">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
        新建隧道
      </button>
    </div>

    <div v-if="tunnels.length > 0" class="tunnel-list section">
      <div v-for="(tunnel, idx) in tunnels" :key="tunnel.id" class="tunnel-card card-transition" :class="{ 'stagger-item': listVisible }" :style="{ animationDelay: `${0.05 * idx}s` }">
        <div class="tunnel-card-left">
          <div class="tunnel-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
          </div>
          <div class="tunnel-info">
            <div class="tunnel-name">{{ tunnel.name }}</div>
            <code class="tunnel-id">{{ tunnel.id }}</code>
          </div>
        </div>
        <div class="tunnel-card-right">
          <span class="status-tag" :class="tunnel.status">{{ tunnel.status }}</span>
          <router-link :to="`/tunnels/${tunnel.id}`" class="btn-sm btn-select">详情</router-link>
          <button
            class="btn-sm"
            :class="config.tunnel_id === tunnel.id ? 'btn-active' : 'btn-select'"
            :disabled="config.tunnel_id === tunnel.id"
            @click="selectTunnel(tunnel)"
          >
            {{ config.tunnel_id === tunnel.id ? '已选' : '选择' }}
          </button>
          <button class="btn-sm btn-delete" title="删除隧道" @click="confirmDelete(tunnel)">删除</button>
        </div>
      </div>
    </div>

    <div v-else-if="!loading" class="empty-state">
      <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="var(--color-mute)" stroke-width="1" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
      <span class="empty-text">暂无隧道数据</span>
      <span class="empty-hint">请检查 Cloudflare API Token 是否正确配置</span>
    </div>

    <!-- Create Tunnel -->
    <n-modal v-model:show="showCreate" preset="card" title="新建隧道" class="tunnel-modal" :bordered="false" :segmented="{ content: true, footer: true }" :auto-focus="false" :mask-closable="!creating">
      <div class="modal-form">
        <div class="form-field">
          <label class="form-label">隧道名称</label>
          <input v-model="createName" placeholder="例如 home-server" class="vercel-input" @keyup.enter="submitCreate" />
        </div>
        <p class="modal-hint">创建为「远程管理」模式，之后可直接在本面板编辑应用程序路由。</p>
      </div>
      <template #footer>
        <div class="modal-footer">
          <button class="btn btn-ghost" :disabled="creating" @click="showCreate = false">取消</button>
          <button class="btn btn-primary" :disabled="creating || !createName.trim()" @click="submitCreate">
            {{ creating ? '创建中...' : '创建' }}
          </button>
        </div>
      </template>
    </n-modal>

    <!-- Connector Token -->
    <n-modal v-model:show="showToken" preset="card" :title="`隧道「${created?.name}」已创建`" class="tunnel-modal" :bordered="false" :segmented="{ content: true, footer: true }" :auto-focus="false">
      <div class="modal-form">
        <p class="modal-hint">在需要接入的机器上运行下面的命令即可连上隧道。连接令牌等同于凭据，请妥善保存、不要外传。</p>
        <div v-if="created?.warning" class="token-warning">{{ created.warning }}</div>

        <div class="install-guide">
          <button class="install-toggle" @click="showInstall = !showInstall">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" :class="{ open: showInstall }"><polyline points="9 18 15 12 9 6"/></svg>
            还没装 cloudflared？查看 Debian / Ubuntu 安装命令
          </button>
          <div v-if="showInstall" class="copy-row">
            <code class="token-box install-script">{{ installScript }}</code>
            <button class="btn-sm btn-select" @click="copy(installScript)">复制</button>
          </div>
        </div>
        <template v-if="created?.run_command">
          <div class="form-field">
            <label class="form-label">运行命令</label>
            <div class="copy-row">
              <code class="token-box">{{ created.run_command }}</code>
              <button class="btn-sm btn-select" @click="copy(created!.run_command!)">复制</button>
            </div>
          </div>
          <div class="form-field">
            <label class="form-label">连接令牌</label>
            <div class="copy-row">
              <code class="token-box">{{ created.token }}</code>
              <button class="btn-sm btn-select" @click="copy(created!.token!)">复制</button>
            </div>
          </div>
        </template>
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
.page-header { margin-bottom: var(--spacing-lg); }
.section { margin-bottom: var(--spacing-md); }

.selection-card {
  background: var(--color-canvas);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
  box-shadow: 0 1px 2px rgba(58, 47, 34, 0.05);
}
.selection-label { color: var(--color-mute); }
.selection-header { margin-bottom: 10px; }
.selection-body { display: flex; align-items: center; gap: 10px; }
.selection-body { justify-content: space-between; }
.selected-tunnel { display: flex; min-width: 0; flex-direction: column; gap: 3px; }
.selected-tunnel strong { color: var(--color-ink); font-size: 15px; }
.selection-empty { color: var(--color-mute); font-size: 14px; }
.toolbar { display: flex; flex-wrap: wrap; gap: 10px; }

.tunnel-list {
  display: flex;
  flex-direction: column;
  gap: 1px;
  background: var(--color-hairline);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  box-shadow: 0 1px 2px rgba(58, 47, 34, 0.05);
  overflow: hidden;
}
.tunnel-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-md) var(--spacing-lg);
  background: var(--color-canvas);
  gap: var(--spacing-md);
  transition: background-color 160ms ease-out, transform 180ms cubic-bezier(0.16, 1, 0.3, 1);
}
.tunnel-card:hover {
  background: var(--color-canvas-soft);
  transform: translateX(2px);
}
.tunnel-card-left {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  min-width: 0;
}
.tunnel-icon {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  background: var(--color-canvas-soft-2);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--color-body);
}
.tunnel-info { min-width: 0; }
.tunnel-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-ink);
  overflow-wrap: anywhere;
}
.tunnel-id {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--color-mute);
  overflow-wrap: anywhere;
}
.tunnel-card-right {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  flex-shrink: 0;
  flex-wrap: wrap;
}

.status-tag {
  font-family: var(--font-mono);
  font-size: 12px;
  padding: 0 8px;
  height: 22px;
  line-height: 20px;
  border-radius: 999px;
  text-transform: uppercase;
}
.status-tag.healthy {
  background: var(--color-status-healthy-bg);
  color: var(--color-status-healthy-text);
  border: 1px solid var(--color-status-healthy-border);
}
.status-tag.degraded {
  background: var(--color-status-degraded-bg);
  color: var(--color-status-degraded-text);
  border: 1px solid var(--color-status-degraded-border);
}
.status-tag.down,
.status-tag.inactive {
  background: var(--color-status-down-bg);
  color: var(--color-status-down-text);
  border: 1px solid var(--color-status-down-border);
}

.btn-sm {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 10px;
  height: 28px;
  border-radius: var(--radius-md);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: transform 120ms ease-out, background-color 160ms ease-out, border-color 160ms ease-out, color 160ms ease-out, opacity 160ms ease-out;
  border: none;
  text-decoration: none;
  box-sizing: border-box;
}
.btn-sm:active:not(:disabled) { transform: scale(0.97); }
.btn-select { background: transparent; color: var(--color-ink); border: 1px solid var(--color-hairline); }
.btn-select:hover { border-color: var(--color-hairline-strong); background: var(--color-canvas-soft); }
.btn-active { background: var(--color-canvas-soft-2); color: var(--color-mute); cursor: default; }
.btn-delete { background: transparent; color: var(--color-mute); border: 1px solid var(--color-hairline); }
.btn-delete:hover { color: var(--color-error); border-color: var(--color-error); }
.btn-ghost-sm { background: transparent; color: var(--color-ink); border: none; cursor: pointer; }
.btn-ghost-sm:hover { opacity: 0.6; }

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-3xl) var(--spacing-lg);
  background: var(--color-canvas);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  box-shadow: 0 1px 2px rgba(58, 47, 34, 0.05);
  gap: var(--spacing-sm);
}
.empty-text { color: var(--color-body); font-size: 16px; font-weight: 600; }
.empty-hint { color: var(--color-mute); font-size: 14px; text-align: center; }

@keyframes spin { to { transform: rotate(360deg); } }
.spin { animation: spin 1s linear infinite; }

@media (max-width: 768px) {
  .tunnel-card { padding: var(--spacing-sm) var(--spacing-md); }
}
@media (max-width: 480px) {
  .selection-body { align-items: flex-start; flex-direction: column; }
  .tunnel-card { align-items: flex-start; flex-direction: column; gap: var(--spacing-sm); }
  .tunnel-card-left,
  .tunnel-card-right { width: 100%; }
  .tunnel-card-right { justify-content: flex-end; }
}

.tunnel-modal {
  max-width: 520px;
  width: 100%;
}
@media (max-width: 768px) {
  .tunnel-modal { max-width: calc(100vw - 2 * var(--spacing-md)); }
}
:deep(.n-modal) { --n-duration: 0.15s; }
:deep(.n-mask) { --n-duration: 0.15s; }

.modal-form {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}
.form-label {
  display: block;
  margin-bottom: 6px;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--color-mute);
  text-transform: uppercase;
  letter-spacing: 0.4px;
}
.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
.modal-hint {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--color-mute);
}
.copy-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.token-box {
  flex: 1;
  min-width: 0;
  max-height: 120px;
  overflow-y: auto;
  padding: 10px 12px;
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.5;
  color: var(--color-ink);
  background: var(--color-canvas-soft-2);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  word-break: break-all;
}
.copy-row .btn-sm { height: 34px; flex-shrink: 0; }
.token-warning {
  padding: 10px 12px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--color-error);
  background: var(--color-canvas-soft);
  border: 1px solid var(--color-error);
  border-radius: var(--radius-md);
}

.install-guide {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.install-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0;
  font-size: 13px;
  color: var(--color-link);
  background: none;
  border: none;
  cursor: pointer;
  text-align: left;
}
.install-toggle:hover { opacity: 0.75; }
.install-toggle svg {
  flex-shrink: 0;
  transition: transform 160ms ease-out;
}
.install-toggle svg.open { transform: rotate(90deg); }
.install-script {
  max-height: 260px;
  white-space: pre-wrap;
  word-break: normal;
  overflow-wrap: anywhere;
}
</style>
