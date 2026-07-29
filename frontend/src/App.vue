<template>
  <n-message-provider>
    <n-config-provider :theme="naiveTheme" :theme-overrides="themeOverrides">
      <n-layout class="app-layout">
        <nav-bar v-if="$route.path !== '/login'" />
        <main class="main-content">
          <router-view v-slot="{ Component }">
            <transition name="page">
              <component :is="Component" :key="$route.fullPath" />
            </transition>
          </router-view>
        </main>
      </n-layout>
    </n-config-provider>
  </n-message-provider>
</template>

<script setup lang="ts">
import { darkTheme } from 'naive-ui'
import { NMessageProvider, NConfigProvider, NLayout } from 'naive-ui'
import { computed } from 'vue'
import NavBar from './components/NavBar.vue'
import { useConfigStore } from './stores/config'
import { warmDarkThemeOverrides, warmThemeOverrides } from './theme'

const configStore = useConfigStore()
const naiveTheme = computed(() => configStore.darkMode ? darkTheme : null)
const themeOverrides = computed(() => configStore.darkMode ? warmDarkThemeOverrides : warmThemeOverrides)

configStore.fetchSiteSettings()

// Sync dark mode to data-theme attribute on mount
if (configStore.darkMode) {
  document.documentElement.setAttribute('data-theme', 'dark')
}
</script>

<style scoped>
.app-layout {
  min-height: 100vh;
  background-color: var(--color-canvas-soft);
}

.main-content {
  position: relative;
  width: 100%;
}

/* Page route transitions — only the leaving page overlays, enter stays in flow */
.page-enter-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.page-leave-active {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  z-index: 1;
  pointer-events: none;
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.page-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.page-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
</style>
