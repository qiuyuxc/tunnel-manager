<template>
  <n-config-provider :theme="naiveTheme" :theme-overrides="currentThemeOverrides">
    <n-message-provider>
      <n-dialog-provider>
        <div class="app-shell">
          <nav-bar v-if="$route.path !== '/login'" />
          <main class="app-main" :class="{ 'app-main-login': route.path === '/login' }">
            <router-view />
          </main>
        </div>
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
import { useConfigStore } from './stores/config'
import {
  darkThemeOverrides,
  themeOverrides,
  warmDarkThemeOverrides,
  warmThemeOverrides,
} from './theme'

const route = useRoute()
const configStore = useConfigStore()
const naiveTheme = computed(() => configStore.darkMode ? darkTheme : null)
const currentThemeOverrides = computed(() => {
  if (configStore.visualTheme === 'warm') {
    return configStore.darkMode ? warmDarkThemeOverrides : warmThemeOverrides
  }
  return configStore.darkMode ? darkThemeOverrides : themeOverrides
})

configStore.fetchSiteSettings()
</script>

<style>
.app-shell {
  display: flex;
  min-height: 100vh;
}

.app-main {
  flex: 1;
  min-width: 0;
  margin-left: var(--sidebar-width);
  background: var(--color-canvas-soft);
  min-height: 100vh;
}

.app-main-login {
  margin-left: 0;
}

@media (max-width: 768px) {
  .app-main {
    margin-left: 0;
    padding-top: 56px;
  }

  .app-main-login {
    padding-top: 0;
  }
}
</style>
