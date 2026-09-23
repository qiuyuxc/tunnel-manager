<template>
  <n-config-provider :theme="naiveTheme" :theme-overrides="currentThemeOverrides">
    <n-message-provider>
      <n-dialog-provider>
        <div class="app-shell" v-if="!route.meta.public">
          <nav-bar v-if="!isLogin" />
          <div class="app-area" :class="{ 'app-area-login': isLogin }">
            <console-header v-if="!isLogin" />
            <main class="app-main" :class="{ 'app-main-login': isLogin }">
              <router-view />
            </main>
          </div>
          <mobile-nav v-if="!isLogin" />
        </div>
        <router-view v-else />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { darkTheme } from 'naive-ui'
import { NConfigProvider, NDialogProvider, NMessageProvider } from 'naive-ui'
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import NavBar from './components/NavBar.vue'
import ConsoleHeader from './components/ConsoleHeader.vue'
import MobileNav from './components/MobileNav.vue'
import { useConfigStore } from './stores/config'
import {
  darkThemeOverrides,
  themeOverrides,
  warmDarkThemeOverrides,
  warmThemeOverrides,
} from './theme'

const route = useRoute()
const configStore = useConfigStore()
const isLogin = computed(() => route.path === '/login')
const naiveTheme = computed(() => configStore.darkMode ? darkTheme : null)
const currentThemeOverrides = computed(() => {
  if (configStore.visualTheme === 'warm') {
    return configStore.darkMode ? warmDarkThemeOverrides : warmThemeOverrides
  }
  return configStore.darkMode ? darkThemeOverrides : themeOverrides
})

configStore.fetchSiteSettings()
configStore.fetchMe()
</script>

<style>
/* Shell layout (.app-shell / .app-area / .app-main) lives in styles.css so the
   sidebar, top bar and content column share one source of truth. */
</style>
