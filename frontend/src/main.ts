import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import App from './App.vue'
import './styles.css'
import { installDesignTokens } from './design-tokens'

installDesignTokens()
const app = createApp(App)
app.use(createPinia())
app.use(router)
// Do not render the console against vue-router's empty initial route. This
// also prevents its sidebar and mobile dock flashing on public status URLs.
router.isReady().then(() => app.mount('#app')).catch(() => {
  const startup = document.querySelector<HTMLElement>('[data-startup-message]')
  if (startup) startup.textContent = document.documentElement.lang === 'zh-CN'
    ? '页面未能加载，请刷新重试。'
    : 'Unable to load this page. Please refresh to retry.'
  document.querySelector('.startup-indicator')?.remove()
})
