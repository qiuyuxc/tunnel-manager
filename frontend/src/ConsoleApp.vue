<template>
  <n-config-provider :theme="naiveTheme" :theme-overrides="currentThemeOverrides" :locale="componentLocale" :date-locale="componentDateLocale">
    <n-message-provider>
      <n-dialog-provider>
        <div class="app-shell" v-if="!route.meta.public">
          <nav-bar v-if="!isLogin" />
          <div class="app-area" :class="{ 'app-area-login': isLogin }">
            <console-header v-if="!isLogin" />
            <main class="app-main" :class="{ 'app-main-login': isLogin }">
              <router-view v-slot="{ Component, route: pageRoute }">
                <component :is="Component" :key="pageRoute.path" />
              </router-view>
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
import { enUS, zhCN, dateEnUS, dateZhCN } from 'naive-ui'
import { language } from './i18n'
import { NConfigProvider, NDialogProvider, NMessageProvider } from 'naive-ui'
import { computed, defineAsyncComponent, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useConfigStore } from './stores/config'
import {
  darkThemeOverrides,
  themeOverrides,
} from './theme'

const route = useRoute()
const NavBar = defineAsyncComponent(() => import('./components/NavBar.vue'))
const ConsoleHeader = defineAsyncComponent(() => import('./components/ConsoleHeader.vue'))
const MobileNav = defineAsyncComponent(() => import('./components/MobileNav.vue'))
const componentLocale = computed(() => language.value === 'zh-CN' ? zhCN : enUS)
const componentDateLocale = computed(() => language.value === 'zh-CN' ? dateZhCN : dateEnUS)
const configStore = useConfigStore()
const isLogin = computed(() => route.path === '/login')
const naiveTheme = computed(() => configStore.darkMode ? darkTheme : null)
const currentThemeOverrides = computed(() => {
  return configStore.darkMode ? darkThemeOverrides : themeOverrides
})

watch(() => route.name, (name) => {
  if (name === 'public-status' || name === 'setup') return
  void configStore.fetchSiteSettings()
  if (route.meta.requiresAuth) void configStore.fetchMe(false)
}, { immediate: true })
</script>

<style>
/* Shell layout (.app-shell / .app-area / .app-main) lives in styles.css so the
   sidebar, top bar and content column share one source of truth. */
</style>
