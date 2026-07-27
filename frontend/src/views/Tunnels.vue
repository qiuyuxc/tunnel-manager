<template>
  <div class="page-container" style="padding-top: 0;">
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
        <code class="inline-code">{{ config.tunnel_id }}</code>
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
            @click="selectTunnel(tunnel.id)"
          >
            {{ config.tunnel_id === tunnel.id ? '已选' : '选择' }}
          </button>
        </div>
      </div>
    </div>

    <div v-else-if="!loading" class="empty-state">
      <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="var(--color-mute)" stroke-width="1" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
      <span class="empty-text">暂无隧道数据</span>
      <span class="empty-hint">请检查 Cloudflare API Token 是否正确配置</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { listTunnels, setTunnelID, type Tunnel } from '../api'
import { useConfigStore } from '../stores/config'

const message = useMessage()
const configStore = useConfigStore()
const config = configStore.config

const tunnels = ref<Tunnel[]>([])
const loading = ref(false)
const listVisible = ref(false)

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

async function selectTunnel(id: string) {
  try {
    await setTunnelID(id)
    config.tunnel_id = id
    message.success('隧道已锁定')
  } catch (e: any) {
    message.error('选择失败: ' + (e.response?.data?.error || e.message))
  }
}

async function clearTunnel() {
  try {
    await setTunnelID('')
    config.tunnel_id = ''
    message.success('已清除隧道选择')
  } catch (e: any) {
    message.error('清除失败')
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
.selection-empty { color: var(--color-mute); font-size: 14px; }
.toolbar { display: flex; }

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
</style>