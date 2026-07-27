<template>
  <div class="page-container" style="padding-top: 0;">
    <div class="page-header">
      <h2>控制面板</h2>
      <p>当前配置概览与快速操作</p>
    </div>

    <!-- Status Summary -->
    <div class="status-banner" :class="[isReady ? 'ready' : 'pending', { 'stagger-item': visible }]" style="animation-delay: 0s;">
      <div class="status-banner-dot" :class="isReady ? 'ready' : 'pending'" />
      <span>{{ isReady ? '所有参数已就绪，可以绑定域名' : '部分参数未配置，请完成以下步骤' }}</span>
    </div>

    <!-- Config Cards Grid -->
    <div class="card-grid card-grid-3 section">
      <div class="config-card card-transition card-hover-lift" :class="{ 'stagger-item': visible }" style="animation-delay: 0.08s;">
        <div class="config-card-top">
          <span class="config-label caption-mono">隧道</span>
          <span class="config-badge" :class="config.tunnel_id ? 'active' : 'inactive'">
            {{ config.tunnel_id ? '已锁定' : '未配置' }}
          </span>
        </div>
        <div class="config-value">
          <code v-if="config.tunnel_id" class="inline-code">{{ config.tunnel_id }}</code>
          <span v-else class="config-empty">尚未锁定隧道</span>
        </div>
        <div class="config-action">
          <router-link to="/tunnels" class="action-link">{{ config.tunnel_id ? '切换' : '选择隧道' }}</router-link>
        </div>
      </div>

      <div class="config-card card-transition card-hover-lift" :class="{ 'stagger-item': visible }" style="animation-delay: 0.16s;">
        <div class="config-card-top">
          <span class="config-label caption-mono">转发地址</span>
          <span class="config-badge" :class="config.service_url ? 'active' : 'inactive'">
            {{ config.service_url ? '已锁定' : '未配置' }}
          </span>
        </div>
        <div class="config-value">
          <code v-if="config.service_url" class="inline-code">{{ config.service_url }}</code>
          <span v-else class="config-empty">尚未设置转发地址</span>
        </div>
        <div class="config-action">
          <router-link to="/domain" class="action-link">{{ config.service_url ? '修改' : '设置地址' }}</router-link>
        </div>
      </div>

      <div class="config-card card-transition card-hover-lift" :class="{ 'stagger-item': visible }" style="animation-delay: 0.24s;">
        <div class="config-card-top">
          <span class="config-label caption-mono">优选 CNAME</span>
          <span class="config-badge active">已配置</span>
        </div>
        <div class="config-value">
          <code class="inline-code">{{ config.preferred_cname }}</code>
        </div>
        <div class="config-action">
          <router-link to="/settings" class="action-link">修改</router-link>
        </div>
      </div>
    </div>

    <!-- Quick Actions -->
    <div class="section">
      <div class="actions-card card-transition" :class="{ 'stagger-item': visible }" style="animation-delay: 0.32s;">
        <span class="actions-label caption-mono">快速操作</span>
        <div class="actions-row">
          <router-link to="/tunnels" class="btn btn-primary">
            管理隧道
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>
          </router-link>
          <router-link to="/domain" class="btn btn-secondary" :class="{ disabled: !isReady }">
            绑定域名
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>
          </router-link>
          <router-link to="/settings" class="btn btn-ghost">
            全局设置
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useConfigStore } from '../stores/config'

const configStore = useConfigStore()
const config = computed(() => configStore.config)
const isReady = computed(() => !!(config.value.tunnel_id && config.value.service_url))

const visible = ref(false)

onMounted(() => {
  configStore.fetchConfig()
  requestAnimationFrame(() => { visible.value = true })
})
</script>

<style scoped>
.status-banner {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 13px 16px;
  border-radius: var(--radius-md);
  font-size: 14px;
  font-weight: 600;
  margin-bottom: var(--spacing-lg);
  border: 1px solid transparent;
}
.status-banner.ready {
  background: var(--color-banner-info-bg);
  border-color: var(--color-banner-info-border);
  color: var(--color-banner-info-text);
}
.status-banner.pending {
  background: var(--color-banner-warning-bg);
  border-color: var(--color-banner-warning-border);
  color: var(--color-banner-warning-text);
}
.status-banner-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  transition: background-color 180ms ease-out;
}
.status-banner-dot.ready { background: var(--color-success); }
.status-banner-dot.pending {
  background: var(--color-warning);
  animation: pulse-subtle 2.5s ease-in-out infinite;
}

@keyframes pulse-subtle {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.45; }
}

.config-card,
.actions-card {
  background: var(--color-canvas);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
  box-shadow: 0 1px 2px rgba(58, 47, 34, 0.05);
}
.config-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-height: 164px;
}
.config-card:hover,
.actions-card:hover {
  border-color: var(--color-hairline-strong);
  box-shadow: 0 12px 28px rgba(58, 47, 34, 0.08);
}
.config-card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-md);
}
.config-label { color: var(--color-mute); }
.config-badge {
  font-size: 12px;
  padding: 0 8px;
  border-radius: 999px;
  min-height: 22px;
  line-height: 20px;
  font-weight: 600;
  font-family: var(--font-sans);
  border: 1px solid transparent;
}
.config-badge.active {
  background: var(--color-status-healthy-bg);
  color: var(--color-status-healthy-text);
  border-color: var(--color-status-healthy-border);
}
.config-badge.inactive {
  background: var(--color-banner-warning-bg);
  color: var(--color-banner-warning-text);
  border-color: var(--color-banner-warning-border);
}
.config-value {
  min-height: 32px;
  display: flex;
  align-items: center;
}
.config-empty { color: var(--color-mute); font-size: 14px; }
.config-action {
  margin-top: auto;
  padding-top: 12px;
  border-top: 1px solid var(--color-hairline);
}
.action-link {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-link);
  text-decoration: none;
}
.action-link:hover { color: var(--color-ink); }

.actions-card { overflow: hidden; }
.actions-label {
  display: block;
  color: var(--color-mute);
  margin-bottom: var(--spacing-md);
}
.actions-row {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.btn-secondary.disabled {
  opacity: 0.45;
  pointer-events: none;
}

@media (max-width: 768px) {
  .actions-row {
    flex-direction: column;
  }
  .actions-row .btn,
  .actions-row .btn-primary,
  .actions-row .btn-secondary,
  .actions-row .btn-ghost {
    width: 100%;
    max-width: 100%;
    justify-content: center;
    box-sizing: border-box;
  }
}
</style>