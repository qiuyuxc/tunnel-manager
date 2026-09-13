<template>
  <!-- Desktop sidebar. Phone widths get MobileNav instead; the two read the same
       menu list from navigation.ts so they can never drift apart. -->
  <aside class="sidebar" :class="{ collapsed }">
    <div class="sidebar-header">
      <router-link to="/dashboard" class="brand">
        <span class="brand-icon" aria-hidden="true">
          <img v-if="configStore.config.site_icon" :src="configStore.config.site_icon" alt="" />
          <svg v-else width="18" height="18" viewBox="0 0 76 76" fill="none">
            <path d="M49 26H27v24l22-24z" fill="currentColor"/>
            <path d="M38 38L27 50h22L38 38z" fill="currentColor" fill-opacity="0.42"/>
          </svg>
        </span>
        <span class="brand-text">{{ configStore.config.site_name }}</span>
      </router-link>
    </div>

    <nav class="sidebar-nav">
      <router-link
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="nav-item"
        :class="{ active: isActive(item.path) }"
        :title="item.label"
      >
        <span class="nav-icon" aria-hidden="true" v-html="item.icon" />
        <span class="nav-label">{{ item.label }}</span>
      </router-link>
    </nav>

    <div class="sidebar-footer">
      <router-link to="/account" class="footer-btn user-chip" :title="configStore.displayName">
        <span class="user-avatar" aria-hidden="true">
          <img v-if="configStore.avatar" :src="configStore.avatar" alt="" />
          <span v-else>{{ (configStore.displayName || '?').charAt(0).toUpperCase() }}</span>
        </span>
        <span class="user-name">{{ configStore.displayName }}</span>
      </router-link>
      <div class="footer-actions">
        <button class="icon-btn" type="button"
          :title="collapsed ? '展开侧边栏' : '收起侧边栏'"
          :aria-label="collapsed ? '展开侧边栏' : '收起侧边栏'"
          @click="toggleSidebar">
          <span class="nav-icon" aria-hidden="true" v-html="collapsed ? icons.expand : icons.collapse" />
        </button>
        <button class="icon-btn" type="button" title="切换视觉主题" aria-label="切换视觉主题"
          @click="configStore.toggleVisualTheme()">
          <span class="nav-icon" aria-hidden="true" v-html="icons.palette" />
        </button>
        <button class="icon-btn" type="button"
          :title="configStore.darkMode ? '切换亮色模式' : '切换暗色模式'"
          :aria-label="configStore.darkMode ? '切换亮色模式' : '切换暗色模式'"
          @click="configStore.toggleDarkMode()">
          <span class="nav-icon" aria-hidden="true" v-html="configStore.darkMode ? icons.sun : icons.moon" />
        </button>
        <button class="icon-btn" type="button" title="退出登录" aria-label="退出登录" @click="handleLogout">
          <span class="nav-icon" aria-hidden="true" v-html="icons.logout" />
        </button>
      </div>
    </div>
  </aside>

  <MobileNav />
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { logout as logoutApi } from '../api'
import { icons, isActivePath, useNavItems } from '../navigation'
import { useConfigStore } from '../stores/config'
import MobileNav from './MobileNav.vue'

const route = useRoute()
const router = useRouter()
const configStore = useConfigStore()
const navItems = useNavItems()

const collapsed = ref(localStorage.getItem('sidebar_collapsed') === 'true')
watch(collapsed, (val) => {
  localStorage.setItem('sidebar_collapsed', String(val))
  document.documentElement.dataset.sidebar = val ? 'collapsed' : ''
}, { immediate: true })

function toggleSidebar() {
  collapsed.value = !collapsed.value
}

function isActive(path: string) {
  return isActivePath(route.path, path)
}

async function handleLogout() {
  try { await logoutApi() } catch (_) { /* ignore */ }
  configStore.clearAuth()
  router.push('/')
}
</script>

<style scoped>
.sidebar {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: var(--sidebar-width);
  background: var(--color-sidebar);
  color: var(--color-sidebar-text);
  display: flex;
  flex-direction: column;
  z-index: 100;
  border-right: 1px solid var(--color-hairline);
  transition: width 180ms ease;
  overflow: hidden;
}

.sidebar-header {
  padding: 16px 18px;
  border-bottom: 1px solid var(--color-sidebar-divider);
  white-space: nowrap;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--color-sidebar-text-active);
  text-decoration: none;
  font-weight: 600;
  font-size: 15px;
}

.brand-icon {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  background: var(--color-brand-icon);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.brand-icon svg { color: var(--color-canvas); }
.brand-icon img { width: 100%; height: 100%; object-fit: cover; border-radius: inherit; }

.sidebar-nav {
  flex: 1;
  padding: 12px 10px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow-y: auto;
}

.nav-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border-radius: 6px;
  color: var(--color-sidebar-text);
  text-decoration: none;
  font-size: 13px;
  font-weight: 500;
  transition: background-color 120ms ease, color 120ms ease;
}

.nav-item:hover {
  background: var(--color-sidebar-hover);
  color: var(--color-sidebar-hover-text);
}

.nav-item.active {
  background: var(--color-sidebar-active-bg);
  color: var(--color-sidebar-text-active);
}

/* vercel.md ex-app-shell-row: the active row is marked by an ink left-edge
   indicator rather than a filled accent block. The phone tab bar transposes the
   same mark onto its top edge. */
.nav-item.active::before {
  content: "";
  position: absolute;
  left: 0;
  top: 7px;
  bottom: 7px;
  width: 2px;
  border-radius: 2px;
  background: var(--color-sidebar-active);
}

.nav-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.sidebar-footer {
  padding: 12px 10px;
  border-top: 1px solid var(--color-sidebar-divider);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.footer-btn {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 9px 12px;
  border-radius: 6px;
  background: transparent;
  border: none;
  color: var(--color-sidebar-text);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  text-align: left;
  transition: background-color 120ms ease, color 120ms ease;
}

.footer-btn:hover {
  background: var(--color-sidebar-hover);
  color: var(--color-sidebar-hover-text);
}

/* Console actions stay icon-only (Vercel's compact icon button: 6px radius,
   the label lives in the tooltip + aria-label instead of visible text). */
.footer-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
  padding: 0;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--color-sidebar-text);
  cursor: pointer;
  transition: background-color 120ms ease, color 120ms ease;
}

.icon-btn:hover {
  background: var(--color-sidebar-hover);
  color: var(--color-sidebar-hover-text);
}

.user-chip {
  text-decoration: none;
  border-bottom: 1px solid var(--color-sidebar-divider);
  padding-bottom: 12px;
  margin-bottom: 4px;
}

.user-avatar {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: var(--color-ink);
  color: var(--color-canvas);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 700;
  overflow: hidden;
  flex: 0 0 auto;
}

.user-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.user-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar.collapsed .user-name { display: none; }
.sidebar.collapsed .user-chip { justify-content: center; padding-left: 6px; padding-right: 6px; }
.sidebar.collapsed .footer-actions { flex-direction: column; align-items: center; }

@media (max-width: 768px) {
  .sidebar { display: none; }
}

/* Collapsed rail (desktop only) */
.sidebar.collapsed .sidebar-header { padding: 16px 10px; display: flex; justify-content: center; }
.sidebar.collapsed .brand { gap: 0; }
.sidebar.collapsed .brand-text,
.sidebar.collapsed .nav-label,
.sidebar.collapsed .footer-btn > span:not(.nav-icon) { display: none; }
.sidebar.collapsed .nav-item,
.sidebar.collapsed .footer-btn { justify-content: center; padding: 9px 0; gap: 0; width: auto; margin: 0 auto; }
</style>
