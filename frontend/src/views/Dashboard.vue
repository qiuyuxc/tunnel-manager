<template>
  <div class="page-container">
    <div class="page-header">
      <h2>控制面板</h2>
      <p>系统概览与快捷操作</p>
    </div>
    <!-- Status Cards -->
    <div class="card-grid card-grid-4 section">
      <div class="metric-card">
        <span class="metric-label">当前隧道</span>
        <div class="metric-value">
          <template v-if="config.tunnel_id">{{ config.tunnel_name || '已选隧道' }}</template>
          <span v-else class="text-muted">未配置</span>
        </div>
        <div class="metric-foot">
          <router-link to="/tunnels" class="link">管理隧道</router-link>
        </div>
      </div>
      <div class="metric-card">
        <span class="metric-label">转发地址</span>
        <div class="metric-value">
          <code v-if="config.service_url" class="inline-code">{{ config.service_url }}</code>
          <span v-else class="text-muted">未配置</span>
        </div>
        <div class="metric-foot">
          <router-link to="/domain" class="link">设置地址</router-link>
        </div>
      </div>
      <div class="metric-card">
        <span class="metric-label">默认优选 CNAME</span>
        <div class="metric-value">
          <code class="inline-code">{{ config.preferred_cname }}</code>
        </div>
        <div class="metric-foot">
          <router-link to="/settings" class="link">修改默认</router-link>
        </div>
      </div>
      <div class="metric-card">
        <span class="metric-label">运行状态</span>
        <div class="metric-value">
          <span class="status-tag" :class="isReady ? 'healthy' : 'down'">
            {{ isReady ? '配置就绪' : '配置未就绪' }}
          </span>
        </div>
        <div class="metric-foot text-muted">
          {{ isReady ? '可以进行域名绑定' : '缺少必要配置' }}
        </div>
      </div>
    </div>
    <!-- Quick Actions -->
        <div class="card section">
      <div class="card-header">
        <span class="caption-mono">快捷操作</span>
      </div>
      <div class="quick-actions">
        <router-link to="/tunnels" class="action-tile">
          <div class="action-icon" v-html="icons.tunnel" />
          <div class="action-body">
            <strong>隧道管理</strong>
            <span>查看、创建和选择 Cloudflare Tunnel</span>
          </div>
        </router-link>
        <router-link to="/domain" class="action-tile" :class="{ disabled: !isReady }">
          <div class="action-icon" v-html="icons.domain" />
          <div class="action-body">
            <strong>域名绑定</strong>
            <span>将域名绑定到已选隧道</span>
          </div>
        </router-link>
        <router-link to="/dns" class="action-tile">
          <div class="action-icon" v-html="icons.dns" />
          <div class="action-body">
            <strong>DNS 管理</strong>
            <span>管理 Cloudflare DNS 记录</span>
          </div>
        </router-link>
        <router-link to="/settings" class="action-tile">
          <div class="action-icon" v-html="icons.settings" />
          <div class="action-body">
            <strong>全局设置</strong>
            <span>站点品牌、CNAME 预设、回退源</span>
          </div>
        </router-link>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useConfigStore } from '../stores/config'

const icons = {
  tunnel: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>',
  domain: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>',
  dns: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4.89 15.14C3.63 13.75 3 12 3 12s.63-1.75 1.89-3.14C6.4 7.33 8.4 6 10.8 6h.41c.49 0 .68.64.34 1.05l-.6.7c-.23.27-.28.64-.13.96.15.32.47.53.82.53h1.36c.35 0 .64-.29.64-.64v-1.3c0-.35-.29-.64-.64-.64h-.09c-.55 0-.82-.66-.44-1.05l.6-.7c.23-.27.28-.64.13-.96a.92.92 0 0 0-.82-.53H10.8C8.4 3 6.4 4.33 4.89 6.14 3.63 7.53 3 9.25 3 12s.63 4.47 1.89 5.86C6.4 19.67 8.4 21 10.8 21h.41c.49 0 .68-.64.34-1.05l-.6-.7a.95.95 0 0 1-.13-.96c.15-.32.47-.53.82-.53h1.36c.35 0 .64.29.64.64v1.3c0 .35-.29.64-.64.64h-.09c-.55 0-.82.66-.44 1.05l.6.7c.23.27.28.64.13.96a.92.92 0 0 1-.82.53H10.8C8.4 21 6.4 19.67 4.89 17.86z"/></svg>',
  settings: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>'
}
const configStore = useConfigStore()
const config = configStore.config
const isReady = computed(() => !!(config.tunnel_id && config.service_url))
onMounted(() => {
  configStore.fetchConfig()
})
</script>
<style scoped>
.section { margin-bottom: var(--spacing-xl); }

.metric-card {
  background: var(--color-canvas-raised);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.metric-label { font-size: 12px; color: var(--color-mute); font-weight: 500; }
.metric-value { min-height: 24px; font-size: 15px; font-weight: 600; color: var(--color-ink); word-break: break-all; }
.metric-foot { margin-top: auto; padding-top: 8px; border-top: 1px solid var(--color-hairline); font-size: 12px; }
.link { color: var(--color-link); text-decoration: none; font-weight: 500; }
.link:hover { color: var(--color-link-hover); text-decoration: underline; }
.text-muted { color: var(--color-mute); font-weight: 400; }

.card {
  background: var(--color-canvas-raised);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  overflow: hidden;
}
.card-header { padding: var(--spacing-md) var(--spacing-lg); border-bottom: 1px solid var(--color-hairline); background: var(--color-canvas-soft); }

.quick-actions {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1px;
  background: var(--color-hairline);
}
.action-tile {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: var(--spacing-lg);
  background: var(--color-canvas-raised);
  text-decoration: none;
  color: var(--color-ink);
  transition: background-color 120ms ease;
}
.action-tile:hover { background: var(--color-canvas-soft); }
.action-tile.disabled { opacity: 0.5; pointer-events: none; }
.action-icon { width: 40px; height: 40px; border-radius: var(--radius-md); background: var(--color-canvas-soft); color: var(--color-link); display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; }
.action-body { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.action-body strong { font-size: 14px; font-weight: 600; }
.action-body span { font-size: 12px; color: var(--color-body); line-height: 1.5; }

@media (max-width: 1024px) { .card-grid-4 { grid-template-columns: repeat(2, minmax(0, 1fr)); } .quick-actions { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 640px) { .card-grid-4 { grid-template-columns: 1fr; } .quick-actions { grid-template-columns: 1fr; } }
</style>
