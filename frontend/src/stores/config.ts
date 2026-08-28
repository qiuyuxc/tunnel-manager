import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'
import { getConfig, getSiteSettings, listTunnels, setTunnelSelection, type Config } from '../api'
import { getMe } from '../api/admin'

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
    landing_enabled: false,
  })
  const darkMode = ref(localStorage.getItem('dark_mode') === 'true')
  const savedVisualTheme = localStorage.getItem('visual_theme')
  const visualTheme = ref<VisualTheme>(savedVisualTheme === 'warm' ? 'warm' : 'enterprise')
  const loading = ref(false)
  const landingEnabled = ref(false)
  const siteSettingsLoaded = ref(false)
  let siteSettingsPromise: Promise<void> | null = null

  // Auth state
  const token = ref(localStorage.getItem('auth_token') || '')
  const username = ref(localStorage.getItem('auth_username') || '')
  const nickname = ref(localStorage.getItem('auth_nickname') || '')
  const avatar = ref(localStorage.getItem('auth_avatar') || '')
  const email = ref(localStorage.getItem('auth_email') || '')
  const role = ref(localStorage.getItem('auth_role') || '')
  const permissions = ref<string[]>([])
  const isAuthenticated = ref(!!token.value)

  const displayName = computed(() => nickname.value || username.value)

  function hasPerm(perm: string) {
    if (role.value === 'admin') return true
    return permissions.value.includes(perm)
  }

  function isAdmin() {
    return role.value === 'admin'
  }

  async function fetchMe() {
    if (!token.value) return
    try {
      const { data } = await getMe()
      role.value = data.role
      permissions.value = data.permissions || []
      username.value = data.username
      nickname.value = data.nickname || ''
      avatar.value = data.avatar || ''
      email.value = data.email || ''
      localStorage.setItem('auth_role', data.role)
      localStorage.setItem('auth_username', data.username)
      localStorage.setItem('auth_nickname', data.nickname || '')
      localStorage.setItem('auth_avatar', data.avatar || '')
      localStorage.setItem('auth_email', data.email || '')
    } catch (_) {
      // 401 is handled by the shared interceptor; keep cached state otherwise.
    }
  }

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

  function fetchSiteSettings() {
    if (siteSettingsPromise) return siteSettingsPromise
    siteSettingsPromise = (async () => {
      try {
        const { data } = await getSiteSettings()
        config.value.site_name = data.name
        config.value.site_description = data.description
        config.value.site_icon = data.icon
        landingEnabled.value = data.landing_enabled
      } catch (_) {
        // Keep local defaults when the public endpoint is unavailable.
      } finally {
        siteSettingsLoaded.value = true
      }
    })()
    return siteSettingsPromise
  }

  async function fetchConfig() {
    loading.value = true
    try {
      const { data } = await getConfig()
      Object.assign(config.value, data)
      landingEnabled.value = !!data.landing_enabled
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

  function setAuth(tokenVal: string, usernameVal: string, roleVal = '') {
    token.value = tokenVal
    username.value = usernameVal
    role.value = roleVal
    isAuthenticated.value = true
    localStorage.setItem('auth_token', tokenVal)
    localStorage.setItem('auth_username', usernameVal)
    localStorage.setItem('auth_role', roleVal)
    fetchMe()
  }

  function clearAuth() {
    token.value = ''
    username.value = ''
    nickname.value = ''
    avatar.value = ''
    email.value = ''
    role.value = ''
    permissions.value = []
    isAuthenticated.value = false
    localStorage.removeItem('auth_token')
    localStorage.removeItem('auth_username')
    localStorage.removeItem('auth_nickname')
    localStorage.removeItem('auth_avatar')
    localStorage.removeItem('auth_email')
    localStorage.removeItem('auth_role')
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
    config, darkMode, visualTheme, loading, landingEnabled, siteSettingsLoaded,
    token, username, nickname, avatar, email, displayName,
    role, permissions, isAuthenticated,
    hasPerm, isAdmin, fetchMe, fetchConfig, fetchSiteSettings, toggleDarkMode, toggleVisualTheme, setAuth, clearAuth,
  }
})
