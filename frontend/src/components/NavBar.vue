<template>
  <!-- Desktop console sidebar. Phone widths get MobileNav instead; both read the
       same permission-filtered menu from navigation.ts so they cannot drift.
       Nav is grouped with captions (工作空间 / 网络 / 管理); 个人 is pinned to
       the footer. Theme, effects and sign-out live in the top bar's avatar menu
       (ConsoleHeader), not here. -->
  <aside class="sidebar" :class="{ collapsed }" aria-label="主导航">
    <router-link to="/dashboard" class="brand" :aria-label="`${configStore.config.site_name} 控制面板`">
      <BrandMark :size="36" />
      <span class="brand-text">
        <span class="brand-name">{{ configStore.config.site_name }}</span>
        <span class="brand-sub">CONNECTION HUB</span>
      </span>
    </router-link>

    <nav class="nav-scroll">
      <div v-for="group in groups" :key="group.key" class="nav-group">
        <span class="nav-caption">{{ group.label }}</span>
        <router-link
          v-for="item in group.items"
          :key="item.path"
          :to="item.path"
          class="nav-link"
          :class="{ active: isActive(item.path) }"
          :aria-current="isActive(item.path) ? 'page' : undefined"
          :title="collapsed ? item.label : undefined"
        >
          <span class="nav-icon" aria-hidden="true" v-html="item.icon" />
          <span class="nav-label">{{ item.label }}</span>
          <span v-if="item.badge" class="nav-count">{{ item.badge }}</span>
        </router-link>
      </div>
    </nav>

    <div class="sidebar-bottom">
      <router-link
        v-for="item in personalItems"
        :key="item.path"
        :to="item.path"
        class="nav-link"
        :class="{ active: isActive(item.path) }"
        :aria-current="isActive(item.path) ? 'page' : undefined"
        :title="collapsed ? item.label : undefined"
      >
        <span class="nav-icon" aria-hidden="true" v-html="item.icon" />
        <span class="nav-label">{{ item.label }}</span>
      </router-link>
      <button
        type="button"
        class="collapse-btn"
        :aria-label="collapsed ? '展开侧边栏' : '收起侧边栏'"
        :title="collapsed ? '展开侧边栏' : '收起侧边栏'"
        @click="collapsed = !collapsed"
      >
        <span class="nav-icon" aria-hidden="true" v-html="collapsed ? icons.expand : icons.collapse" />
        <span class="nav-label">收起侧边栏</span>
      </button>
      <div class="sidebar-foot">
        <span class="foot-status"><span class="dot" aria-hidden="true" />{{ configStore.displayName || '当前实例' }}</span>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { GROUP_LABELS, SIDEBAR_GROUPS, icons, isActivePath, useNavItems } from '../navigation'
import { useConfigStore } from '../stores/config'
import BrandMark from './BrandMark.vue'

const route = useRoute()
const configStore = useConfigStore()
const navItems = useNavItems()

// Rail collapse (desktop only). One data attribute on <html> feeds the
// --sidebar-width variable in styles.css, so the content column re-flows with
// the rail. Persisted so the choice survives reloads.
const collapsed = ref(localStorage.getItem('sidebar_collapsed') === 'true')
watch(collapsed, (val) => {
  localStorage.setItem('sidebar_collapsed', String(val))
  document.documentElement.dataset.sidebar = val ? 'collapsed' : ''
}, { immediate: true })

const groups = computed(() =>
  SIDEBAR_GROUPS.map((key) => ({
    key,
    label: GROUP_LABELS[key],
    items: navItems.value.filter((item) => item.group === key),
  })).filter((group) => group.items.length > 0),
)

const personalItems = computed(() => navItems.value.filter((item) => item.group === 'personal'))

function isActive(path: string) {
  return isActivePath(route.path, path)
}
</script>

<style scoped>
.sidebar {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: var(--sidebar-width);
  display: flex;
  flex-direction: column;
  padding: 22px 16px 16px;
  background: var(--color-sidebar);
  border-right: 1px solid var(--color-hairline);
  z-index: 100;
}

.brand {
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 0 8px;
  color: var(--color-ink);
  text-decoration: none;
}

.brand-text { min-width: 0; line-height: 1.25; }
.brand-name { display: block; font-size: 15px; font-weight: 600; letter-spacing: -0.3px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.brand-sub { display: block; font-size: 10px; letter-spacing: 1.5px; color: var(--color-mute); }

.nav-scroll {
  flex: 1;
  overflow-y: auto;
  margin: 22px -4px 12px;
  padding: 0 4px;
  scrollbar-width: thin;
}

.nav-group + .nav-group { margin-top: 20px; }

.nav-caption {
  display: block;
  margin: 0 14px 6px;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 1px;
  text-transform: uppercase;
  color: var(--color-mute);
}

.nav-link {
  display: flex;
  align-items: center;
  gap: 11px;
  min-height: 42px;
  margin: 3px 0;
  padding: 9px 14px;
  border-radius: var(--radius-pill);
  color: var(--color-sidebar-text);
  font-size: 13px;
  font-weight: 500;
  text-decoration: none;
  transition: background-color 120ms ease, color 120ms ease;
}

.nav-link:hover { background: var(--color-sidebar-hover); color: var(--color-sidebar-hover-text); }

.nav-link.active {
  background: var(--color-sidebar-active-bg);
  color: var(--color-sidebar-text-active);
  box-shadow: inset 0 1px 0 rgb(255 255 255 / 4%);
}

.nav-icon { display: inline-flex; align-items: center; justify-content: center; width: 18px; height: 18px; flex-shrink: 0; }
.nav-icon :deep(svg) { width: 17px; height: 17px; }

.nav-count {
  margin-left: auto;
  padding: 1px 7px;
  border-radius: 6px;
  background: var(--color-canvas-soft-2);
  color: var(--color-mute);
  font-size: 10px;
  font-weight: 600;
}

.nav-link.active .nav-count { background: var(--color-status-degraded-bg); color: var(--color-status-degraded-text); }

.sidebar-bottom {
  margin-top: auto;
  padding-top: 12px;
  border-top: 1px solid var(--color-sidebar-divider);
}

.sidebar-foot {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 12px 14px 0;
  font-size: 10px;
  color: var(--color-mute);
}

.foot-status { display: inline-flex; align-items: center; gap: 6px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dot { width: 5px; height: 5px; border-radius: 50%; background: var(--color-success); flex-shrink: 0; }

.collapse-btn {
  display: flex;
  align-items: center;
  gap: 11px;
  width: 100%;
  min-height: 42px;
  margin: 3px 0;
  padding: 9px 14px;
  border: none;
  border-radius: var(--radius-pill);
  background: transparent;
  color: var(--color-sidebar-text);
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  text-align: left;
  cursor: pointer;
}
.collapse-btn:hover { background: var(--color-sidebar-hover); color: var(--color-sidebar-hover-text); }

/* Collapsed rail: labels, captions and the foot line fold away; icons centre.
   Width itself comes from --sidebar-width (styles.css, data-sidebar). */
.sidebar.collapsed { padding-left: 12px; padding-right: 12px; }
.sidebar.collapsed .brand { justify-content: center; gap: 0; }
.sidebar.collapsed .brand-text,
.sidebar.collapsed .nav-caption,
.sidebar.collapsed .nav-label,
.sidebar.collapsed .nav-count,
.sidebar.collapsed .sidebar-foot { display: none; }
.sidebar.collapsed .nav-link,
.sidebar.collapsed .collapse-btn { justify-content: center; gap: 0; padding-left: 0; padding-right: 0; }

@media (max-width: 768px) {
  .sidebar { display: none; }
}
</style>
