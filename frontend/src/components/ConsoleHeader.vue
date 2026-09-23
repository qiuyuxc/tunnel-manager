<template>
  <!-- Desktop console top bar: breadcrumb on the left, quick-jump + theme + a
       notifications shortcut + the account menu on the right. Hidden on phones,
       where MobileNav carries identity and navigation instead. Every control
       here maps to a real route or a real preference — nothing decorative. -->
  <header class="topbar">
    <nav class="breadcrumb" aria-label="面包屑">
      <span class="crumb-root">{{ configStore.config.site_name }}</span>
      <template v-for="(crumb, i) in crumbs" :key="i">
        <span class="sep" aria-hidden="true">/</span>
        <router-link v-if="crumb.path" :to="crumb.path" class="crumb-link">{{ crumb.label }}</router-link>
        <span v-else class="crumb-current">{{ crumb.label }}</span>
      </template>
    </nav>

    <div class="topbar-actions">
      <button type="button" class="search-trigger" @click="openJump">
        <span class="nav-icon" aria-hidden="true" v-html="icons.search" />
        快速前往
        <kbd>{{ shortcutLabel }}</kbd>
      </button>
      <button
        type="button"
        class="icon-btn"
        :aria-label="configStore.darkMode ? '切换亮色模式' : '切换暗色模式'"
        @click="configStore.toggleDarkMode()"
      >
        <span class="nav-icon" aria-hidden="true" v-html="configStore.darkMode ? icons.sun : icons.moon" />
      </button>
      <router-link class="icon-btn" to="/notifications" aria-label="通知设置">
        <span class="nav-icon" aria-hidden="true" v-html="icons.bell" />
      </router-link>
      <button
        type="button"
        class="avatar"
        :aria-label="`账户菜单（${configStore.displayName}）`"
        :aria-expanded="menuOpen"
        @click="menuOpen = !menuOpen"
      >
        <img v-if="configStore.avatar" :src="configStore.avatar" alt="" />
        <span v-else>{{ initial }}</span>
      </button>
    </div>

    <Transition name="menu-fade">
      <div v-if="menuOpen" class="menu-scrim" @click="menuOpen = false" />
    </Transition>
    <Transition name="menu-pop">
      <div v-if="menuOpen" class="account-menu" role="menu">
        <div class="menu-head">
          <span class="avatar lg">
            <img v-if="configStore.avatar" :src="configStore.avatar" alt="" />
            <span v-else>{{ initial }}</span>
          </span>
          <span class="menu-ident">
            <span class="menu-name">{{ configStore.displayName }}</span>
            <span class="menu-sub">{{ roleLabel }}</span>
          </span>
        </div>
        <div class="menu-sep" />
        <router-link to="/account" class="menu-row" role="menuitem">
          <span class="menu-icon" v-html="icons.account" />
          <span class="menu-label">账户设置</span>
          <span class="menu-chevron" v-html="icons.chevron" />
        </router-link>
        <button type="button" class="menu-row" role="menuitem" @click="configStore.toggleVisualTheme()">
          <span class="menu-icon" v-html="icons.palette" />
          <span class="menu-label">视觉主题</span>
          <span class="menu-tail">{{ themeLabel }}</span>
        </button>
        <button type="button" class="menu-row" role="menuitem" @click="configStore.toggleDarkMode()">
          <span class="menu-icon" v-html="configStore.darkMode ? icons.sun : icons.moon" />
          <span class="menu-label">暗色模式</span>
          <span class="switch" :class="{ on: configStore.darkMode }" />
        </button>
        <button type="button" class="menu-row" role="menuitem" @click="configStore.toggleLightEffects()">
          <span class="menu-icon" v-html="icons.palette" />
          <span class="menu-label">轻量效果</span>
          <span class="switch" :class="{ on: configStore.lightEffects }" />
        </button>
        <div class="menu-sep" />
        <button type="button" class="menu-row danger" role="menuitem" @click="handleLogout">
          <span class="menu-icon" v-html="icons.logout" />
          <span class="menu-label">退出登录</span>
        </button>
      </div>
    </Transition>

    <JumpPalette :open="jumpOpen" @close="jumpOpen = false" />
  </header>
</template>
<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { logout as logoutApi } from '../api'
import { breadcrumbFor, icons } from '../navigation'
import { useConfigStore } from '../stores/config'
import JumpPalette from './JumpPalette.vue'

const route = useRoute()
const router = useRouter()
const configStore = useConfigStore()

const menuOpen = ref(false)
const jumpOpen = ref(false)

const crumbs = computed(() => breadcrumbFor(route.path))
const initial = computed(() => (configStore.displayName || '?').charAt(0).toUpperCase())
const roleLabel = computed(() => (configStore.isAdmin() ? '管理员' : '用户'))
const themeLabel = computed(() => (configStore.visualTheme === 'warm' ? '暖色' : '默认'))

// ⌘K on macOS, Ctrl+K elsewhere — the label follows the platform.
const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform)
const shortcutLabel = isMac ? '⌘ K' : 'Ctrl K'

function openJump() {
  menuOpen.value = false
  jumpOpen.value = true
}

function onKeydown(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault()
    jumpOpen.value = true
  }
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => window.removeEventListener('keydown', onKeydown))

// Any navigation closes the account menu so it never outlives its page.
watch(() => route.path, () => {
  menuOpen.value = false
})

async function handleLogout() {
  menuOpen.value = false
  try {
    await logoutApi()
  } catch (_) {
    /* offline or already invalid — clearing the local session is enough */
  }
  configStore.clearAuth()
  router.push('/')
}
</script>
<style scoped>
.topbar {
  position: sticky;
  top: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  height: var(--header-height);
  padding: 0 36px;
  background: var(--color-header-bg);
  border-bottom: 1px solid var(--color-header-border);
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  color: var(--color-mute);
  font-size: 12px;
}

.breadcrumb .sep { color: var(--color-hairline-strong); }
.crumb-root { color: var(--color-mute); }
.crumb-link { color: var(--color-body); text-decoration: none; }
.crumb-link:hover { color: var(--color-ink); }
.crumb-current { color: var(--color-ink); font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.topbar-actions { display: flex; align-items: center; gap: 9px; flex-shrink: 0; }

.search-trigger {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 36px;
  padding: 7px 13px;
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-pill);
  background: transparent;
  color: var(--color-mute);
  font-size: 12px;
  cursor: pointer;
}
.search-trigger:hover { color: var(--color-body); border-color: var(--color-hairline-strong); }
.search-trigger .nav-icon :deep(svg) { width: 15px; height: 15px; }

kbd {
  padding: 0 5px;
  border: 1px solid var(--color-hairline);
  border-radius: 4px;
  font: 10px inherit;
  color: var(--color-mute);
}
/* icon buttons, avatar, account menu */
.icon-btn {
  display: inline-grid;
  place-items: center;
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--color-body);
  cursor: pointer;
}
.icon-btn:hover { background: var(--color-canvas-soft-2); color: var(--color-ink); }
.icon-btn .nav-icon :deep(svg) { width: 18px; height: 18px; }

.avatar {
  display: inline-grid;
  place-items: center;
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  padding: 0;
  border: 1px solid var(--color-hairline);
  border-radius: 50%;
  background: var(--color-canvas-soft-2);
  color: var(--color-link);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  overflow: hidden;
}
.avatar img { width: 100%; height: 100%; object-fit: cover; }
.avatar.lg { width: 38px; height: 38px; font-size: 15px; }

.nav-icon { display: inline-flex; align-items: center; justify-content: center; }

/* ---- account menu ---- */
.menu-scrim { position: fixed; inset: 0; z-index: 109; }

.account-menu {
  position: fixed;
  top: 60px;
  right: 24px;
  z-index: 110;
  width: 236px;
  overflow: hidden;
  background: var(--color-canvas-raised);
  border: 1px solid var(--color-hairline);
  border-radius: 14px;
  box-shadow: 0 18px 48px rgb(0 0 0 / 22%);
}
/* account menu rows */
.menu-head { display: flex; align-items: center; gap: 10px; padding: 12px; }
.menu-ident { min-width: 0; }
.menu-name { display: block; font-size: 14px; font-weight: 600; color: var(--color-ink); }
.menu-sub { font-size: 12px; color: var(--color-mute); }
.menu-sep { height: 1px; background: var(--color-hairline); }

.menu-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 9px 12px;
  border: none;
  background: transparent;
  color: var(--color-ink);
  font: inherit;
  font-size: 14px;
  text-align: left;
  text-decoration: none;
  cursor: pointer;
}
.menu-row:hover { background: var(--color-btn-ghost-hover); }
.menu-icon { display: inline-flex; flex: 0 0 auto; color: var(--color-body); }
.menu-icon :deep(svg) { width: 17px; height: 17px; }
.menu-label { flex: 1; min-width: 0; }
.menu-tail { font-size: 12px; color: var(--color-mute); }
.menu-chevron { display: inline-flex; flex: 0 0 auto; color: var(--color-mute); }
.menu-chevron :deep(svg) { width: 15px; height: 15px; }
.menu-row.danger, .menu-row.danger .menu-icon { color: var(--color-error); }

.switch {
  position: relative;
  width: 34px;
  height: 20px;
  flex: 0 0 auto;
  border-radius: 10px;
  background: var(--color-hairline-strong);
  transition: background-color 140ms ease;
}
.switch.on { background: var(--color-btn-primary-bg); }
.switch::after {
  content: "";
  position: absolute;
  top: 2px;
  left: 2px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--color-btn-primary-text);
  transition: transform 140ms ease;
}
.switch.on::after { transform: translateX(14px); }
/* transitions + responsive */
.menu-fade-enter-active, .menu-fade-leave-active { transition: opacity 140ms ease; }
.menu-fade-enter-from, .menu-fade-leave-to { opacity: 0; }
.menu-pop-enter-active, .menu-pop-leave-active { transition: opacity 140ms ease, transform 140ms ease; }
.menu-pop-enter-from, .menu-pop-leave-to { opacity: 0; transform: translateY(-6px) scale(0.98); }

@media (prefers-reduced-motion: reduce) {
  .menu-fade-enter-active, .menu-fade-leave-active,
  .menu-pop-enter-active, .menu-pop-leave-active { transition: none; }
}

@media (max-width: 1060px) {
  .search-trigger { display: none; }
}

@media (max-width: 768px) {
  .topbar { display: none; }
}
</style>
