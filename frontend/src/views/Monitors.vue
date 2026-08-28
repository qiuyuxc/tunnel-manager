<template>
  <div class="page-container">
    <div class="page-header monitors-head">
      <div>
        <h2>服务监控</h2>
        <p>创建监控项目，持续探测服务可用性</p>
      </div>
      <button class="btn btn-primary btn-sm" @click="createOpen = true">创建监控</button>
    </div>

    <div v-if="loading" class="empty-note">加载中...</div>
    <div v-else-if="monitors.length === 0" class="empty-card">
      <div class="empty-icon" aria-hidden="true">
        <svg width="30" height="30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
      </div>
      <h3>还没有监控项目</h3>
      <p>创建一个监控项目，添加需要盯住的服务地址，系统会按固定间隔自动检测并在公开页展示状态。</p>
      <button class="btn btn-primary btn-sm" @click="createOpen = true">创建第一个监控</button>
    </div>

    <div v-else class="monitor-list">
      <div
        v-for="m in monitors"
        :key="m.id"
        class="card monitor-item"
        role="button"
        tabindex="0"
        @click="goDetail(m.id)"
        @keydown.enter="goDetail(m.id)"
      >
        <div class="mi-main">
          <div class="mi-name-row">
            <strong class="mi-name" :title="m.name">{{ m.name }}</strong>
            <span class="mi-count">{{ m.targets.length }} 个服务</span>
          </div>
          <div class="mi-meta text-muted">
            每 {{ m.interval_sec }}s 检测 · {{ m.publish_enabled ? '公开页已开启' : '未公开' }}
            <span v-if="m.targets.length" class="sum-dots">
              <span class="sum-dot ok"></span>{{ countState(m, 'ok') }}
              <span class="sum-dot warn"></span>{{ countState(m, 'warn') }}
              <span class="sum-dot down"></span>{{ countState(m, 'down') }}
            </span>
          </div>
        </div>
        <div class="mi-actions" @click.stop>
          <button class="btn btn-secondary btn-sm" @click="copyPublic(m)">复制公开链接</button>
          <button class="row-delete" title="删除监控项目" @click="askDelete(m)">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
          </button>
        </div>
      </div>
    </div>

    <!-- 创建弹窗 -->
    <Teleport to="body">
      <div v-if="createOpen" class="dlg-mask" @click.self="createOpen = false">
        <div class="dlg">
          <h3>创建监控项目</h3>
          <label class="fld">
            <span>项目名称</span>
            <input v-model.trim="newName" maxlength="60" placeholder="例如：生产环境服务" />
          </label>
          <div class="dlg-actions">
            <button class="btn btn-secondary btn-sm" @click="createOpen = false">取消</button>
            <button class="btn btn-primary btn-sm" :disabled="creating || !newName" @click="doCreate">
              {{ creating ? '创建中…' : '创建并配置服务' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 删除确认 -->
    <Teleport to="body">
      <div v-if="deleting" class="dlg-mask" @click.self="deleting = null">
        <div class="dlg">
          <h3>删除监控项目</h3>
          <p class="dlg-desc">将删除「{{ deleting.name }}」及其全部检测历史，公开链接同步失效。</p>
          <div class="dlg-actions">
            <button class="btn btn-secondary btn-sm" :disabled="busy" @click="deleting = null">取消</button>
            <button class="btn btn-danger btn-sm" :disabled="busy" @click="confirmDelete">{{ busy ? '删除中…' : '确认删除' }}</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import {
  createMonitor,
  deleteMonitor,
  listMonitors,
  publicStatusPath,
  type MonitorView,
} from '../api'

const router = useRouter()
const message = useMessage()

const monitors = ref<MonitorView[]>([])
const loading = ref(true)

const createOpen = ref(false)
const newName = ref('')
const creating = ref(false)
const deleting = ref<MonitorView | null>(null)
const busy = ref(false)

async function load() {
  try {
    const { data } = await listMonitors()
    monitors.value = data || []
  } finally {
    loading.value = false
  }
}

function countState(m: MonitorView, state: string) {
  return m.targets.filter((t) => t.state === state).length
}

function goDetail(id: string) {
  void router.push('/monitors/' + id)
}

async function doCreate() {
  if (!newName.value || creating.value) return
  creating.value = true
  try {
    const { data } = await createMonitor(newName.value)
    createOpen.value = false
    newName.value = ''
    message.success('已创建，开始添加需要监控的服务')
    void router.push('/monitors/' + data.id)
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    message.error(msg || '创建失败')
  } finally {
    creating.value = false
  }
}

function copyPublic(m: MonitorView) {
  if (!m.public_token) return
  if (!m.publish_enabled) return
  const seg = m.public_slug || m.public_token
  if (!seg) return
  navigator.clipboard.writeText(location.origin + publicStatusPath(seg))
    .then(() => message.success('公开链接已复制'))
    .catch(() => message.error('复制失败'))
}

function askDelete(m: MonitorView) {
  deleting.value = m
}

async function confirmDelete() {
  if (!deleting.value || busy.value) return
  busy.value = true
  try {
    await deleteMonitor(deleting.value.id)
    message.success('已删除')
    await load()
    deleting.value = null
  } catch (_) {
    message.error('删除失败')
  } finally {
    busy.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.monitors-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 16px; }
.monitor-list { display: flex; flex-direction: column; gap: 12px; }
.monitor-item {
  display: flex; align-items: center; justify-content: space-between; gap: 16px;
  padding: 16px 20px; cursor: pointer; transition: border-color 120ms ease, transform 120ms ease;
}
.monitor-item:hover { border-color: var(--color-hairline-strong); }
.mi-main { min-width: 0; }
.mi-name-row { display: flex; align-items: baseline; gap: 10px; min-width: 0; }
.mi-name { font-size: 15px; font-weight: 600; color: var(--color-ink); min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mi-count { font-size: 12px; color: var(--color-mute); flex-shrink: 0; }
.mi-meta { margin-top: 5px; font-size: 12.5px; display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.sum-dots { display: inline-flex; align-items: center; gap: 5px; margin-left: 8px; color: var(--color-body); font-weight: 500; }
.sum-dot { width: 7px; height: 7px; border-radius: 50%; display: inline-block; margin-left: 4px; }
.sum-dot:first-child { margin-left: 0; }
.sum-dot.ok { background: var(--color-success); }
.sum-dot.warn { background: var(--color-warning); }
.sum-dot.down { background: var(--color-error); }
.mi-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
.empty-card {
  margin-top: 8px; background: var(--color-canvas-raised); border: 1px dashed var(--color-hairline-strong);
  border-radius: var(--radius-lg); padding: 48px 24px; text-align: center;
  display: flex; flex-direction: column; align-items: center; gap: 10px;
}
.empty-icon { width: 56px; height: 56px; border-radius: 14px; display: grid; place-items: center;
  background: var(--color-canvas-soft); color: var(--color-body); border: 1px solid var(--color-hairline); }
.empty-card h3 { margin: 0; font-size: 15px; color: var(--color-ink); }
.empty-card p { margin: 0 0 8px; max-width: 420px; font-size: 13px; line-height: 1.7; color: var(--color-mute); }
.empty-note { padding: 40px; text-align: center; color: var(--color-mute); }
.fld { display: flex; flex-direction: column; gap: 6px; font-size: 12.5px; color: var(--color-body); margin-bottom: 16px; }
.fld input {
  height: 36px; border-radius: 6px; border: 1px solid var(--color-hairline-strong);
  background: var(--color-canvas-raised); color: var(--color-ink); padding: 0 10px; outline: none; font-size: 13.5px;
}
.fld input:focus { border-color: var(--color-focus); }
.dlg-mask { position: fixed; inset: 0; z-index: 2000; display: grid; place-items: center; padding: 16px; background: rgba(8, 10, 18, .45); }
.dlg { width: min(420px, 100%); background: var(--color-canvas-raised); border: 1px solid var(--color-hairline);
  border-radius: 10px; padding: 20px; box-shadow: 0 16px 48px rgba(0, 0, 0, .22); }
.dlg h3 { margin: 0 0 8px; font-size: 15px; color: var(--color-ink); }
.dlg-desc { margin: 0 0 16px; font-size: 13px; line-height: 1.6; color: var(--color-body); }
.dlg-actions { display: flex; justify-content: flex-end; gap: 8px; }
@media (max-width: 640px) {
  .monitor-item { flex-direction: column; align-items: stretch; }
  .mi-actions { justify-content: flex-end; }
}

.row-delete {
	display: inline-flex; align-items: center; justify-content: center; width: 26px; height: 26px; flex-shrink: 0;
	border-radius: 6px; border: 1px solid transparent; background: transparent; color: var(--color-mute); cursor: pointer;
	transition: color 120ms ease, background-color 120ms ease, border-color 120ms ease;
}
.row-delete:hover:not(:disabled) { color: var(--color-error); background: var(--color-status-down-bg); border-color: var(--color-status-down-border); }
</style>
