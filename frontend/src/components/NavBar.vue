<template>
  <!-- Desktop sidebar -->
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
      <button class="footer-btn" :title="collapsed ? '展开侧边栏' : '收起侧边栏'" @click="toggleSidebar">
        <span class="nav-icon" aria-hidden="true" v-html="collapsed ? icons.expand : icons.collapse" />
        <span>{{ collapsed ? '' : '收起侧边栏' }}</span>
      </button>
      <button class="footer-btn" title="切换视觉主题" @click="configStore.toggleVisualTheme()">
        <span class="nav-icon" aria-hidden="true" v-html="icons.palette" />
        <span>{{ configStore.visualTheme === 'warm' ? '切换专业蓝主题' : '切换暖纸主题' }}</span>
      </button>
      <button class="footer-btn" title="切换明暗模式" @click="configStore.toggleDarkMode()">
        <span class="nav-icon" v-html="configStore.darkMode ? icons.sun : icons.moon" />
        <span>{{ configStore.darkMode ? '亮色模式' : '暗色模式' }}</span>
      </button>
      <button class="footer-btn" title="退出登录" @click="handleLogout">
        <span class="nav-icon" v-html="icons.logout" />
        <span>退出登录</span>
      </button>
    </div>
  </aside>

  <!-- Mobile header -->
  <header class="mobile-header">
    <button class="hamburger" @click="mobileOpen = !mobileOpen" aria-label="打开菜单">
      <span></span>
      <span></span>
      <span></span>
    </button>
    <span class="mobile-title">{{ configStore.config.site_name }}</span>
  </header>

  <!-- Mobile drawer -->
  <Transition name="fade">
    <div v-if="mobileOpen" class="mobile-overlay" @click="mobileOpen = false"></div>
  </Transition>
  <Transition name="slide">
    <aside v-if="mobileOpen" class="mobile-drawer">
      <div class="drawer-header">
        <router-link to="/dashboard" class="brand" @click="mobileOpen = false">
          <span class="brand-icon" aria-hidden="true">
            <img v-if="configStore.config.site_icon" :src="configStore.config.site_icon" alt="" />
            <svg v-else width="18" height="18" viewBox="0 0 76 76" fill="none">
              <path d="M49 26H27v24l22-24z" fill="currentColor"/>
              <path d="M38 38L27 50h22L38 38z" fill="currentColor" fill-opacity="0.42"/>
            </svg>
          </span>
          <span class="brand-text">{{ configStore.config.site_name }}</span>
        </router-link>
        <button class="close-btn" @click="mobileOpen = false" aria-label="关闭菜单">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      </div>
      <nav class="drawer-nav">
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="drawer-item"
          :class="{ active: isActive(item.path) }"
          @click="mobileOpen = false"
        >
          <span class="nav-icon" aria-hidden="true" v-html="item.icon" />
          <span>{{ item.label }}</span>
        </router-link>
      </nav>
      <div class="drawer-footer">
        <router-link to="/account" class="footer-btn user-chip" @click="mobileOpen = false">
          <span class="user-avatar" aria-hidden="true">
            <img v-if="configStore.avatar" :src="configStore.avatar" alt="" />
            <span v-else>{{ (configStore.displayName || '?').charAt(0).toUpperCase() }}</span>
          </span>
          <span class="user-name">{{ configStore.displayName }}</span>
        </router-link>
        <button class="footer-btn" @click="configStore.toggleVisualTheme(); mobileOpen = false">
          <span class="nav-icon" aria-hidden="true" v-html="icons.palette" />
          <span>{{ configStore.visualTheme === 'warm' ? '切换专业蓝主题' : '切换暖纸主题' }}</span>
        </button>
        <button class="footer-btn" @click="configStore.toggleDarkMode(); mobileOpen = false">
          <span class="nav-icon" v-html="configStore.darkMode ? icons.sun : icons.moon" />
          <span>{{ configStore.darkMode ? '亮色模式' : '暗色模式' }}</span>
        </button>
        <button class="footer-btn" @click="handleLogout">
          <span class="nav-icon" v-html="icons.logout" />
          <span>退出登录</span>
        </button>
      </div>
    </aside>
  </Transition>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useConfigStore } from '../stores/config'
import { logout as logoutApi } from '../api'

const route = useRoute()
const router = useRouter()
const configStore = useConfigStore()
const mobileOpen = ref(false)

const collapsed = ref(localStorage.getItem('sidebar_collapsed') === 'true')
watch(collapsed, (val) => {
  localStorage.setItem('sidebar_collapsed', String(val))
  document.documentElement.dataset.sidebar = val ? 'collapsed' : ''
}, { immediate: true })

function toggleSidebar() {
  collapsed.value = !collapsed.value
}

const icons = {
  monitor: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>',
  dashboard: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>',
  tunnels: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>',
  domain: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>',
  dns: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>',
  settings: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>',
  telegram: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.198 2.433a2.216 2.216 0 0 0-2.4.557L2.49 14.97a2.216 2.216 0 0 0 1.674 3.716h.003a2.216 2.216 0 0 0 .84-.167l3.59-1.49-2.15-2.15 10.32-6.654-7.66 7.66 6.36 3.84a2.216 2.216 0 0 0 2.15.167l.003-.001a2.216 2.216 0 0 0 1.04-1.36l3.39-15.18a2.217 2.217 0 0 0-.557-2.4z"/></svg>',
  account: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>',
  bell: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>',
  about: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>',
  moon: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>',
  sun: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>',
  palette: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="13.5" cy="6.5" r=".5" fill="currentColor"/><circle cx="17.5" cy="10.5" r=".5" fill="currentColor"/><circle cx="8.5" cy="7.5" r=".5" fill="currentColor"/><circle cx="6.5" cy="12.5" r=".5" fill="currentColor"/><path d="M12 2a10 10 0 0 0 0 20h1.7a1.8 1.8 0 0 0 1.3-3l-.5-.5a1.8 1.8 0 0 1 1.3-3H18a4 4 0 0 0 4-4A10 10 0 0 0 12 2z"/></svg>',
  logout: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>',
  collapse: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="9" y1="3" x2="9" y2="21"/><polyline points="14 9 12 12 14 15"/></svg>',
  expand: '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="9" y1="3" x2="9" y2="21"/><polyline points="11 9 13 12 11 15"/></svg>',
}

const navItems = computed(() => {
  const items = [
    { path: '/dashboard', label: '控制面板', icon: icons.dashboard, perm: '', admin: false },
    { path: '/tunnels', label: '隧道管理', icon: icons.tunnels, perm: 'tunnels', admin: false },
    { path: '/monitors', label: '服务监控', icon: icons.monitor, perm: 'monitors', admin: false },
    { path: '/domain', label: '域名绑定', icon: icons.domain, perm: 'domain_bind', admin: false },
    { path: '/dns', label: 'DNS 管理', icon: icons.dns, perm: 'dns', admin: false },
    { path: '/settings', label: '全局设置', icon: icons.settings, perm: '', admin: true },
    { path: '/telegram', label: 'TG 机器人', icon: icons.telegram, perm: '', admin: false },
    { path: '/admin', label: '管理后台', icon: icons.account, perm: '', admin: true },
    { path: '/account', label: '账户', icon: icons.account, perm: '', admin: false },
    { path: '/notifications', label: '通知', icon: icons.bell, perm: '', admin: false },
    { path: '/about', label: '关于', icon: icons.about, perm: '', admin: false },
  ]
  return items.filter((item) => {
    if (item.admin) return configStore.isAdmin()
    if (item.perm) return configStore.hasPerm(item.perm)
    return true
  })
})

function isActive(path: string) {
  if (path === '/dashboard') return route.path === '/dashboard'
  return route.path === path || route.path.startsWith(path + '/')
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

.brand-icon svg { color: var(--color-sidebar-text-active); }
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
  background: var(--color-sidebar-active);
  color: var(--color-sidebar-text-active);
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
  gap: 2px;
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
  background: var(--color-sidebar-active);
  color: var(--color-sidebar-text-active);
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

/* Mobile header */
.mobile-header {
  display: none;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 56px;
  background: var(--color-header-bg);
  border-bottom: 1px solid var(--color-header-border);
  z-index: 99;
  align-items: center;
  gap: 12px;
  padding: 0 16px;
}

.hamburger {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 4px;
  width: 36px;
  height: 36px;
  padding: 0;
  border: none;
  background: transparent;
  cursor: pointer;
}

.hamburger span {
  display: block;
  width: 18px;
  height: 2px;
  background: var(--color-ink);
  border-radius: 1px;
}

.mobile-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-ink);
}

/* Mobile drawer */
.mobile-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  z-index: 99;
}

.mobile-drawer {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: 260px;
  background: var(--color-sidebar);
  z-index: 100;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}

.drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  border-bottom: 1px solid var(--color-sidebar-divider);
}

.drawer-header .brand { color: var(--color-sidebar-text-active); }

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  color: var(--color-sidebar-text);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.drawer-nav {
  flex: 1;
  padding: 12px 10px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.drawer-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 6px;
  color: var(--color-sidebar-text);
  text-decoration: none;
  font-size: 14px;
  font-weight: 500;
  transition: background-color 120ms ease, color 120ms ease;
}

.drawer-item:hover, .drawer-item.active {
  background: var(--color-sidebar-hover);
  color: var(--color-sidebar-hover-text);
}

.drawer-item.active {
  background: var(--color-sidebar-active);
}

.drawer-footer {
  padding: 12px 10px;
  border-top: 1px solid var(--color-sidebar-divider);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.fade-enter-active, .fade-leave-active { transition: opacity 200ms ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
.slide-enter-active { transition: transform 250ms cubic-bezier(0.16, 1, 0.3, 1); }
.slide-leave-active { transition: transform 200ms ease-in; }
.slide-enter-from { transform: translateX(-100%); }
.slide-leave-to { transform: translateX(-100%); }

@media (max-width: 768px) {
  .sidebar { display: none; }
  .mobile-header { display: flex; }
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
