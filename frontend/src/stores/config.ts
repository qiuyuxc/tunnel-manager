import { t as tr } from '../i18n.ts'
import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'
import { getConfig, getSiteSettings, listTunnels, type Config } from '../api'
import { getMe } from '../api/admin'

export type VisualTheme = 'enterprise'

export const useConfigStore = defineStore('config', () => {
  const config = ref<Config>({
    tunnel_id: '',
    tunnel_name: '',
    service_url: '',
    preferred_cname: 'cf.090227.xyz',
    cname_presets: [{ name: tr('m_a45d83b644b9'), value: 'cf.090227.xyz' }],
    site_name: 'Tunnel Manager',
    site_description: tr('m_c2c8982623de'),
    site_icon: '',
    landing_enabled: false,
    experimental_features_enabled: false,
  })
  const darkMode = ref(localStorage.getItem('dark_mode') !== 'false')
  const lightEffects = ref(localStorage.getItem('light_effects') === 'true')
  const visualTheme = ref<VisualTheme>('enterprise')
  const loading = ref(false)
  const landingEnabled = ref(false)
  const experimentalFeatures = ref(false)
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
  let mePromise: { token: string; promise: Promise<void> } | null = null
  let meLoadedToken = ''
  let meLoadedAt = 0
  let configPromise: { token: string; promise: Promise<void>; controller: AbortController } | null = null
  let configGeneration = 0

  const displayName = computed(() => nickname.value || username.value)

  function hasPerm(_perm: string) {
    return true
  }

  function isAdmin() {
    return true
  }

  function fetchMe(refresh = true): Promise<void> {
    const requestToken = token.value
    if (!requestToken || (!refresh && meLoadedToken === requestToken && Date.now() - meLoadedAt < 60000)) return Promise.resolve()
    if (mePromise?.token === requestToken) return mePromise.promise
    const promise = (async () => {
      try {
        const { data } = await getMe()
        if (token.value !== requestToken) return
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
        meLoadedToken = requestToken
        meLoadedAt = Date.now()
      } catch (_) {
        // 401 is handled by the shared interceptor; keep cached state otherwise.
      } finally {
        if (mePromise?.token === requestToken) mePromise = null
      }
    })()
    mePromise = { token: requestToken, promise }
    return promise
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

  watch(lightEffects, (value) => {
    localStorage.setItem('light_effects', String(value))
    document.documentElement.dataset.effects = value ? 'light' : 'normal'
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
        experimentalFeatures.value = !!data.experimental_features_enabled
      } catch (_) {
        // Keep local defaults when the public endpoint is unavailable.
        siteSettingsPromise = null
      } finally {
        siteSettingsLoaded.value = true
      }
    })()
    return siteSettingsPromise
  }

  function fetchConfig(): Promise<void> {
    const requestToken = token.value
    if (!requestToken) return Promise.resolve()
    if (configPromise?.token === requestToken) return configPromise.promise
    const generation = ++configGeneration
    const controller = new AbortController()
    loading.value = true
    const promise = (async () => {
      try {
        const { data } = await getConfig(controller.signal)
        if (token.value !== requestToken || generation !== configGeneration) return
        Object.assign(config.value, data)
        landingEnabled.value = !!data.landing_enabled
        experimentalFeatures.value = !!data.experimental_features_enabled
        if (data.tunnel_id && !data.tunnel_name) {
          // The selected ID and other settings are usable immediately. Legacy
          // name resolution may need a slow Cloudflare request.
          void resolveLegacyTunnelName(data.tunnel_id, requestToken)
        }
      } catch (e) {
        // Config might not be loaded yet
      } finally {
        if (configPromise?.controller === controller) {
          configPromise = null
          loading.value = false
        }
      }
    })()
    configPromise = { token: requestToken, promise, controller }
    return promise
  }

  function invalidateConfig() {
    configGeneration++
    configPromise?.controller.abort()
    configPromise = null
    loading.value = false
  }

  function setExperimentalFeatures(enabled: boolean) {
    experimentalFeatures.value = enabled
  }

  function toggleDarkMode() {
    darkMode.value = !darkMode.value
  }

  function toggleLightEffects() {
    lightEffects.value = !lightEffects.value
  }

  function setAuth(tokenVal: string, usernameVal: string, roleVal = '') {
    invalidateConfig()
    meLoadedToken = ''
    permissions.value = []
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
    invalidateConfig()
    meLoadedToken = ''
    mePromise = null
    configPromise = null
    loading.value = false
    config.value.tunnel_id = ''
    config.value.tunnel_name = ''
    config.value.service_url = ''
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

  async function resolveLegacyTunnelName(tunnelID: string, requestToken: string) {
    try {
      const { data } = await listTunnels()
      const tunnel = data.find((item) => item.id === tunnelID)
      if (!tunnel || token.value !== requestToken || config.value.tunnel_id !== tunnelID) return
      // A display-name lookup is read-only; it must not rewrite a selection
      // that another page or session changed while Cloudflare was responding.
      config.value.tunnel_name = tunnel.name
    } catch (_) {
      // The ID remains usable even if Cloudflare is temporarily unavailable.
    }
  }

  function applySiteBranding() {
    if (document.documentElement.dataset.entry === 'status') return
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
    config, darkMode, lightEffects, visualTheme, loading, landingEnabled, experimentalFeatures, siteSettingsLoaded,
    token, username, nickname, avatar, email, displayName,
    role, permissions, isAuthenticated,
    hasPerm, isAdmin, fetchMe, fetchConfig, invalidateConfig, fetchSiteSettings, setExperimentalFeatures, toggleDarkMode, toggleLightEffects, setAuth, clearAuth,
  }
})
