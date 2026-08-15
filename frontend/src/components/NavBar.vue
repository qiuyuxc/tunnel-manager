<template>
  <div class="nav-bar">
    <div class="nav-inner">
      <div class="nav-brand">
        <router-link to="/" class="logo">
          <span class="logo-mark" aria-hidden="true">
            <img v-if="configStore.config.site_icon" :src="configStore.config.site_icon" alt="" class="brand-icon" />
            <svg v-else width="18" height="18" viewBox="0 0 76 76" fill="none">
              <path d="M49 26H27v24l22-24z" fill="currentColor"/>
              <path d="M38 38L27 50h22L38 38z" fill="currentColor" fill-opacity="0.42"/>
            </svg>
          </span>
          <span class="logo-text">{{ configStore.config.site_name }}</span>
        </router-link>
        <span class="nav-context">{{ configStore.config.site_description }}</span>
      </div>

      <div class="nav-center">
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="nav-link"
          :class="{ active: route.path === item.path }"
        >
          {{ item.label }}
        </router-link>
      </div>

      <div class="nav-right">
        <button class="hamburger" @click="mobileOpen = !mobileOpen" :aria-label="mobileOpen ? '关闭菜单' : '打开菜单'">
          <span class="hamburger-line" :class="{ open: mobileOpen }"></span>
          <span class="hamburger-line" :class="{ open: mobileOpen }"></span>
          <span class="hamburger-line" :class="{ open: mobileOpen }"></span>
        </button>
        <button class="icon-button" @click="configStore.toggleDarkMode()" :title="configStore.darkMode ? '亮色模式' : '暗色模式'">
          <svg v-if="configStore.darkMode" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="5"/>
            <line x1="12" y1="1" x2="12" y2="3"/>
            <line x1="12" y1="21" x2="12" y2="23"/>
            <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/>
            <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/>
            <line x1="1" y1="12" x2="3" y2="12"/>
            <line x1="21" y1="12" x2="23" y2="12"/>
            <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/>
            <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
          </svg>
          <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
          </svg>
        </button>
        <button v-if="configStore.isAuthenticated" class="icon-button logout-btn" @click="handleLogout" title="退出登录">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
            <polyline points="16 17 21 12 16 7"/>
            <line x1="21" y1="12" x2="9" y2="12"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- Mobile menu overlay -->
    <transition name="fade">
      <div v-if="mobileOpen" class="mobile-overlay" @click="mobileOpen = false"></div>
    </transition>

    <!-- Mobile menu panel -->
    <transition name="slide">
      <div v-if="mobileOpen" class="mobile-menu">
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="mobile-nav-link"
          :class="{ active: route.path === item.path }"
          @click="mobileOpen = false"
        >
          {{ item.label }}
        </router-link>
        <div class="mobile-menu-divider"></div>
        <button class="mobile-logout-btn" @click="handleLogout">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
            <polyline points="16 17 21 12 16 7"/>
            <line x1="21" y1="12" x2="9" y2="12"/>
          </svg>
          退出登录
        </button>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useConfigStore } from '../stores/config'
import { logout as logoutApi } from '../api'

const route = useRoute()
const router = useRouter()
const configStore = useConfigStore()
const mobileOpen = ref(false)

const navItems = [
  { path: '/', label: '控制面板' },
  { path: '/tunnels', label: '隧道管理' },
  { path: '/domain', label: '域名绑定' },
  { path: '/dns', label: 'DNS 管理' },
  { path: '/settings', label: '全局设置' },
  { path: '/telegram', label: 'TG 机器人' },
  { path: '/account', label: '账户' },
]

async function handleLogout() {
  try { await logoutApi() } catch (_) { /* ignore */ }
  configStore.clearAuth()
  mobileOpen.value = false
  router.push('/login')
}
</script>

<style scoped>
.nav-bar {
  position: sticky;
  top: 0;
  z-index: 100;
  min-height: var(--header-height);
  background: color-mix(in srgb, var(--color-canvas) 92%, transparent);
  border-bottom: 1px solid var(--color-hairline);
  backdrop-filter: blur(18px);
}

.brand-icon {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: inherit;
}

.nav-inner {
  max-width: var(--max-width);
  min-height: var(--header-height);
  margin: 0 auto;
  padding: 0 var(--spacing-lg);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-lg);
}

.nav-brand {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 220px;
}

.logo {
  display: flex;
  align-items: center;
  gap: 10px;
  text-decoration: none;
  color: var(--color-ink);
}

.logo-mark {
  width: 30px;
  height: 30px;
  border-radius: var(--radius-lg);
  background: var(--color-ink);
  color: var(--color-canvas);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--color-canvas) 10%, transparent);
}

.logo-text {
  font-family: var(--font-display);
  font-size: 17px;
  font-weight: 600;
  line-height: 1;
}

.nav-context {
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: var(--color-mute);
  padding-left: 12px;
  border-left: 1px solid var(--color-hairline);
}

.nav-center {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 4px;
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  background: var(--color-canvas-soft);
}

.nav-link {
  position: relative;
  padding: 7px 11px;
  border-radius: var(--radius-md);
  font-size: 13px;
  font-weight: 600;
  line-height: 18px;
  color: var(--color-body);
  text-decoration: none;
  transition: color 160ms ease-out, background-color 160ms ease-out;
}

.nav-link:hover {
  color: var(--color-ink);
  background: color-mix(in srgb, var(--color-canvas) 76%, transparent);
}

.nav-link.active {
  color: var(--color-ink);
  background: var(--color-canvas);
  box-shadow: 0 1px 2px rgba(38, 31, 22, 0.08);
}

.nav-right {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
}

.icon-button {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-hairline);
  background: var(--color-canvas);
  color: var(--color-body);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 120ms ease-out, border-color 160ms ease-out, color 160ms ease-out, background-color 160ms ease-out;
}

.icon-button:hover {
  color: var(--color-ink);
  border-color: var(--color-hairline-strong);
  background: var(--color-canvas-soft);
}

.icon-button:active { transform: scale(0.96); }

.logout-btn:hover {
  color: var(--color-error);
  border-color: var(--color-error);
}

.hamburger {
  display: none;
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-hairline);
  background: var(--color-canvas);
  cursor: pointer;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 0;
}

.hamburger-line {
  display: block;
  width: 16px;
  height: 2px;
  background: var(--color-ink);
  border-radius: 1px;
  transition: transform 180ms cubic-bezier(0.16, 1, 0.3, 1), opacity 140ms ease-out;
}

.hamburger-line.open:nth-child(1) { transform: translateY(6px) rotate(45deg); }
.hamburger-line.open:nth-child(2) { opacity: 0; }
.hamburger-line.open:nth-child(3) { transform: translateY(-6px) rotate(-45deg); }

.mobile-overlay {
  position: fixed;
  inset: 0;
  background: rgba(29, 24, 18, 0.38);
  z-index: 99;
}

.mobile-menu {
  position: fixed;
  top: var(--header-height);
  left: 0;
  right: 0;
  background: var(--color-canvas);
  border-bottom: 1px solid var(--color-hairline);
  padding: var(--spacing-sm);
  z-index: 100;
  display: flex;
  flex-direction: column;
  gap: 2px;
  box-shadow: 0 16px 36px rgba(38, 31, 22, 0.14);
}

.mobile-nav-link,
.mobile-logout-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px var(--spacing-md);
  border-radius: var(--radius-md);
  font-size: 15px;
  font-weight: 600;
  color: var(--color-body);
  text-decoration: none;
  transition: color 160ms ease-out, background-color 160ms ease-out;
}

.mobile-nav-link:hover,
.mobile-nav-link.active {
  color: var(--color-ink);
  background: var(--color-canvas-soft);
}

.mobile-menu-divider {
  height: 1px;
  background: var(--color-hairline);
  margin: var(--spacing-xs) 0;
}

.mobile-logout-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  width: 100%;
  text-align: left;
}

.mobile-logout-btn:hover {
  color: var(--color-error);
  background: var(--color-canvas-soft);
}

.fade-enter-active { transition: opacity 180ms ease-out; }
.fade-leave-active { transition: opacity 140ms ease-in; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

.slide-enter-active { transition: opacity 240ms ease-out, transform 240ms cubic-bezier(0.16, 1, 0.3, 1); }
.slide-leave-active { transition: opacity 160ms ease-in, transform 160ms ease-in; }
.slide-enter-from { opacity: 0; transform: translateY(-8px); }
.slide-leave-to { opacity: 0; transform: translateY(-4px); }

@media (max-width: 1120px) {
  .nav-context { display: none; }
  .nav-brand { min-width: auto; }
}

@media (max-width: 900px) {
  .nav-center { display: none; }
  .hamburger { display: flex; }
  .nav-inner { padding: 0 var(--spacing-md); }
  .logo-text { font-size: 16px; }
}
</style>
