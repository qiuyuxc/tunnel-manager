<template>
  <!-- Phone chrome: a slim header that only carries identity, and the tab bar
       that does the navigating. The sidebar and its drawer are desktop-only.
       The header deliberately shows the site name rather than the page title:
       every console page already renders its own heading, and the tab bar
       already says where you are. -->
  <header class="mobile-header">
    <span class="mobile-title">{{ configStore.config.site_name }}</span>
    <button
      type="button"
      class="avatar-btn"
      :aria-label="`账户菜单（${configStore.displayName}）`"
      :aria-expanded="menuOpen"
      @click="menuOpen = !menuOpen"
    >
      <span class="user-avatar">
        <img v-if="configStore.avatar" :src="configStore.avatar" alt="" />
        <span v-else>{{ initial }}</span>
      </span>
    </button>
  </header>

  <Transition name="menu-fade">
    <div v-if="menuOpen" class="menu-scrim" @click="menuOpen = false" />
  </Transition>
  <Transition name="menu-pop">
    <div v-if="menuOpen" class="avatar-menu" role="menu">
      <div class="menu-head">
        <span class="user-avatar lg">
          <img v-if="configStore.avatar" :src="configStore.avatar" alt="" />
          <span v-else>{{ initial }}</span>
        </span>
        <span class="menu-ident">
          <span class="menu-name">{{ configStore.displayName }}</span>
          <span class="menu-sub">{{ roleLabel }}</span>
        </span>
      </div>
      <div class="menu-sep" />
      <router-link to="/account" class="menu-row" role="menuitem" @click="menuOpen = false">
        <span class="menu-icon" v-html="icons.account" />
        <span class="menu-label">账户信息</span>
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
      <div class="menu-sep" />
      <button type="button" class="menu-row danger" role="menuitem" @click="handleLogout">
        <span class="menu-icon" v-html="icons.logout" />
        <span class="menu-label">退出登录</span>
      </button>
    </div>
  </Transition>

  <nav class="tabbar" :style="{ gridTemplateColumns: `repeat(${tabEntries.length}, 1fr)` }">
    <template v-for="(entry, index) in tabEntries" :key="index">
      <router-link
        v-if="entry.kind === 'link'"
        :to="entry.item.path"
        class="tab"
        :class="{ active: isActive(entry.item.path) }"
      >
        <span class="tab-icon" v-html="entry.item.icon" />
        <span class="tab-label">{{ entry.item.tabLabel || entry.item.label }}</span>
      </router-link>
      <button
        v-else-if="entry.kind === 'fab'"
        type="button"
        class="tab tab-fab"
        aria-label="新建"
        @click="createOpen = true"
      >
        <span class="fab" v-html="icons.plus" />
      </button>
      <button v-else type="button" class="tab" @click="moreOpen = true">
        <span class="tab-icon" v-html="icons.more" />
        <span class="tab-label">更多</span>
      </button>
    </template>
  </nav>

  <SheetPanel :open="createOpen" title="新建" @close="createOpen = false">
    <router-link
      v-for="action in CREATE_ACTIONS"
      :key="action.path"
      :to="action.path"
      class="action-row"
      @click="createOpen = false"
    >
      <span class="action-icon" v-html="action.icon" />
      <span class="action-text">
        <span class="action-title">{{ action.title }}</span>
        <span class="action-desc">{{ action.desc }}</span>
      </span>
      <span class="action-chevron" v-html="icons.chevron" />
    </router-link>
  </SheetPanel>

  <SheetPanel :open="moreOpen" title="更多" @close="moreOpen = false">
    <template v-for="group in moreGroups" :key="group.label">
      <div class="group-label">{{ group.label }}</div>
      <div class="group-list">
        <router-link
          v-for="item in group.items"
          :key="item.path"
          :to="item.path"
          class="group-row"
          @click="moreOpen = false"
        >
          <span class="row-icon" v-html="item.icon" />
          <span class="row-label">{{ item.label }}</span>
          <span v-if="item.badge" class="badge">{{ item.badge }}</span>
          <span class="row-chevron" v-html="icons.chevron" />
        </router-link>
      </div>
    </template>
  </SheetPanel>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { logout as logoutApi } from '../api'
import {
  CREATE_ACTIONS,
  GROUP_LABELS,
  TAB_PATHS,
  icons,
  isActivePath,
  useNavItems,
  type NavGroup,
  type NavItem,
} from '../navigation'
import { useConfigStore } from '../stores/config'
import SheetPanel from './SheetPanel.vue'

type TabEntry =
  | { kind: 'link'; item: NavItem }
  | { kind: 'fab' }
  | { kind: 'more' }

const route = useRoute()
const router = useRouter()
const configStore = useConfigStore()
const navItems = useNavItems()

const menuOpen = ref(false)
const createOpen = ref(false)
const moreOpen = ref(false)

const initial = computed(() => (configStore.displayName || '?').charAt(0).toUpperCase())
const themeLabel = computed(() => (configStore.visualTheme === 'warm' ? 'Claude' : 'Vercel'))
const roleLabel = computed(() => (configStore.isAdmin() ? '管理员' : '用户'))

// 概览 · 监控 · [+] · DNS 管理 · 更多, with any entry the account cannot see
// dropped rather than left as a dead slot.
const tabEntries = computed<TabEntry[]>(() => {
  const find = (path: string) => navItems.value.find((item) => item.path === path)
  const entries: TabEntry[] = []
  for (const path of TAB_PATHS) {
    const item = find(path)
    if (item) entries.push({ kind: 'link', item })
    if (path === '/monitors') entries.push({ kind: 'fab' })
  }
  entries.push({ kind: 'more' })
  return entries
})

const moreGroups = computed(() =>
  (['network', 'system', 'personal'] as NavGroup[])
    .map((group) => ({
      label: GROUP_LABELS[group],
      items: navItems.value.filter((item) => item.group === group && !TAB_PATHS.includes(item.path)),
    }))
    .filter((group) => group.items.length > 0),
)

function isActive(path: string) {
  return isActivePath(route.path, path)
}

// Any navigation closes whatever was open, so a sheet never outlives its page.
watch(() => route.path, () => {
  menuOpen.value = false
  createOpen.value = false
  moreOpen.value = false
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
.mobile-header,
.tabbar {
  display: none;
}

.user-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  flex: 0 0 auto;
  overflow: hidden;
  border-radius: 50%;
  background: var(--color-canvas-soft-2);
  color: var(--color-ink);
  font-size: 13px;
  font-weight: 600;
  box-shadow: 0 0 0 1px var(--color-hairline);
}

.user-avatar img { width: 100%; height: 100%; object-fit: cover; }
.user-avatar.lg { width: 38px; height: 38px; font-size: 15px; }

.avatar-btn {
  padding: 0;
  border: none;
  background: transparent;
  cursor: pointer;
  display: inline-flex;
}

/* ---- tab bar ---- */
.tabbar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 100;
  background: var(--color-canvas);
  border-top: 1px solid var(--color-hairline);
  padding-bottom: env(safe-area-inset-bottom);
}

.tab {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
  padding: 9px 2px 10px;
  border: none;
  background: transparent;
  color: var(--color-mute);
  font-family: inherit;
  font-size: 11px;
  font-weight: 500;
  line-height: 1.2;
  text-decoration: none;
  cursor: pointer;
  white-space: nowrap;
}

.tab-icon { display: inline-flex; }
.tab-icon :deep(svg) { width: 22px; height: 22px; }

.tab.active { color: var(--color-ink); font-weight: 600; }

/* Transposed from the sidebar's ink left-edge indicator: the same mark, turned
   into a top edge because the bar is horizontal. */
.tab.active::before {
  content: "";
  position: absolute;
  top: 0;
  left: 50%;
  width: 22px;
  height: 2px;
  margin-left: -11px;
  border-radius: 2px;
  background: var(--color-ink);
}

.tab-fab { padding-top: 0; justify-content: flex-start; }

.fab {
  display: grid;
  place-items: center;
  width: 46px;
  height: 46px;
  margin-top: -18px;
  border-radius: 50%;
  background: var(--color-btn-primary-bg);
  color: var(--color-btn-primary-text);
  box-shadow: 0 0 0 4px var(--color-canvas), 0 6px 16px rgba(0, 0, 0, 0.2);
}

.fab :deep(svg) { width: 24px; height: 24px; }

/* ---- avatar menu ---- */
.menu-scrim { position: fixed; inset: 0; z-index: 109; }

.avatar-menu {
  position: fixed;
  top: 60px;
  right: 12px;
  z-index: 110;
  width: 224px;
  overflow: hidden;
  background: var(--color-canvas-raised);
  border: 1px solid var(--color-hairline);
  border-radius: 10px;
  box-shadow: 0 10px 28px rgba(0, 0, 0, 0.22);
}

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
  font-family: inherit;
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

.menu-fade-enter-active,
.menu-fade-leave-active { transition: opacity 140ms ease; }
.menu-fade-enter-from,
.menu-fade-leave-to { opacity: 0; }

.menu-pop-enter-active,
.menu-pop-leave-active { transition: opacity 140ms ease, transform 140ms ease; }
.menu-pop-enter-from,
.menu-pop-leave-to { opacity: 0; transform: translateY(-6px) scale(0.98); }

/* ---- sheet contents ---- */
.action-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 10px;
  border: 1px solid var(--color-hairline);
  border-radius: 10px;
  color: inherit;
  text-decoration: none;
}

.action-row + .action-row { margin-top: 8px; }

.action-icon {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  flex: 0 0 auto;
  border-radius: 10px;
  background: var(--color-canvas-soft-2);
  color: var(--color-ink);
}

.action-icon :deep(svg) { width: 20px; height: 20px; }
.action-text { flex: 1; min-width: 0; }
.action-title { display: block; font-size: 14px; font-weight: 600; color: var(--color-ink); }
.action-desc { display: block; margin-top: 1px; font-size: 12px; color: var(--color-mute); }
.action-chevron { display: inline-flex; flex: 0 0 auto; color: var(--color-mute); }

.group-label {
  padding: 14px 10px 5px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--color-mute);
}

.group-list {
  border-radius: 10px;
  overflow: hidden;
  background: var(--color-canvas-raised);
}

.group-row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 44px;
  padding: 8px 10px;
  color: inherit;
  text-decoration: none;
}

.group-row + .group-row { border-top: 1px solid var(--color-hairline); }

.row-icon { display: inline-flex; flex: 0 0 auto; color: var(--color-body); }
.row-icon :deep(svg) { width: 19px; height: 19px; }
.row-label { flex: 1; min-width: 0; font-size: 14px; color: var(--color-ink); }
.row-chevron { display: inline-flex; flex: 0 0 auto; color: var(--color-mute); }

.badge {
  flex: 0 0 auto;
  padding: 2px 6px;
  border: 1px solid var(--color-status-degraded-border);
  border-radius: 4px;
  background: var(--color-status-degraded-bg);
  color: var(--color-status-degraded-text);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.04em;
}

@media (prefers-reduced-motion: reduce) {
  .menu-fade-enter-active,
  .menu-fade-leave-active,
  .menu-pop-enter-active,
  .menu-pop-leave-active { transition: none; }
}

@media (max-width: 768px) {
  .mobile-header {
    display: flex;
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    z-index: 99;
    height: 56px;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 0 16px;
    background: var(--color-header-bg);
    border-bottom: 1px solid var(--color-header-border);
  }

  .mobile-title {
    overflow: hidden;
    font-size: 15px;
    font-weight: 600;
    color: var(--color-ink);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .tabbar { display: grid; }
}
</style>
