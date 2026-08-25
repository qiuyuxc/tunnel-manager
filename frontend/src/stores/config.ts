import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import { getConfig, getSiteSettings, listTunnels, setTunnelSelection, type Config } from '../api'

export type VisualTheme = 'enterprise' | 'warm'

export const useConfigStore = defineStore('config', () => {
  const config = ref<Config>({
    tunnel_id: '',
    tunnel_name: '',
    service_url: '',
    preferred_cname: 'cf.090227.xyz',
    cname_presets: [{ name: '默认优选', value: 'cf.090227.xyz' }],
    site_name: 'Tunnel Manager',
    site_description: 'Cloudflare 隧道管理中心',
    site_icon: '',
  })
  const darkMode = ref(localStorage.getItem('dark_mode') === 'true')
  const savedVisualTheme = localStorage.getItem('visual_theme')
  const visualTheme = ref<VisualTheme>(savedVisualTheme === 'warm' ? 'warm' : 'enterprise')
  const loading = ref(false)

  // Auth state
  const token = ref(localStorage.getItem('auth_token') || '')
  const username = ref(localStorage.getItem('auth_username') || '')
  const isAuthenticated = ref(!!token.value)

  // Persist darkMode
  watch(darkMode, (val) => {
    localStorage.setItem('dark_mode', String(val))
    document.documentElement.setAttribute('data-theme', val ? 'dark' : '')
  }, { immediate: true })

  watch(visualTheme, (val) => {
    localStorage.setItem('visual_theme', val)
    document.documentElement.setAttribute('data-visual-theme', val)
  }, { immediate: true })

  watch(() => [config.value.site_name, config.value.site_icon], applySiteBranding, { immediate: true })

  async function fetchSiteSettings() {
    try {
      const { data } = await getSiteSettings()
      config.value.site_name = data.name
      config.value.site_description = data.description
      config.value.site_icon = data.icon
    } catch (_) {
      // Keep local defaults when the public endpoint is unavailable.
    }
  }

  async function fetchConfig() {
    loading.value = true
    try {
      const { data } = await getConfig()
      Object.assign(config.value, data)
      if (data.tunnel_id && !data.tunnel_name) {
        await resolveLegacyTunnelName(data.tunnel_id)
      }
    } catch (e) {
      // Config might not be loaded yet
    } finally {
      loading.value = false
    }
  }

  function toggleDarkMode() {
    darkMode.value = !darkMode.value
  }

  function toggleVisualTheme() {
    visualTheme.value = visualTheme.value === 'warm' ? 'enterprise' : 'warm'
  }

  function setAuth(tokenVal: string, usernameVal: string) {
    token.value = tokenVal
    username.value = usernameVal
    isAuthenticated.value = true
    localStorage.setItem('auth_token', tokenVal)
    localStorage.setItem('auth_username', usernameVal)
  }

  function clearAuth() {
    token.value = ''
    username.value = ''
    isAuthenticated.value = false
    localStorage.removeItem('auth_token')
    localStorage.removeItem('auth_username')
  }

  async function resolveLegacyTunnelName(tunnelID: string) {
    try {
      const { data } = await listTunnels()
      const tunnel = data.find((item) => item.id === tunnelID)
      if (!tunnel) return
      await setTunnelSelection(tunnel.id, tunnel.name)
      config.value.tunnel_name = tunnel.name
    } catch (_) {
      // The ID remains usable even if Cloudflare is temporarily unavailable.
    }
  }

  function applySiteBranding() {
    document.title = config.value.site_name || 'Tunnel Manager'
    const existing = document.querySelector<HTMLLinkElement>('link[data-site-icon]')
    if (!config.value.site_icon) {
      existing?.remove()
      return
    }
    const icon = existing || document.createElement('link')
    icon.rel = 'icon'
    icon.dataset.siteIcon = 'true'
    icon.href = config.value.site_icon
    if (!existing) document.head.appendChild(icon)
  }

  return {
    config, darkMode, visualTheme, loading, token, username, isAuthenticated,
    fetchConfig, fetchSiteSettings, toggleDarkMode, toggleVisualTheme, setAuth, clearAuth,
  }
})
