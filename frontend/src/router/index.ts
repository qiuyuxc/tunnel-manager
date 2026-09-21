import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'
import Login from '../views/Login.vue'
import Landing from '../views/Landing.vue'
import { useConfigStore } from '../stores/config'
import { setupStatus } from '../api/setup'
import LabIPSelector from '../views/LabIPSelector.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    // The wizard stands alone: no sidebar, no sign-in, nothing else to reach.
    { path: '/setup', name: 'setup', component: () => import('../views/Setup.vue'), meta: { setup: true, public: true } },
    { path: '/login', name: 'login', component: Login },
    { path: '/', name: 'landing', component: Landing, meta: { public: true } },
    { path: '/home', redirect: '/' },
    { path: '/dashboard', name: 'dashboard', component: Dashboard, meta: { requiresAuth: true } },
    { path: '/tunnels', name: 'tunnels', component: () => import('../views/Tunnels.vue'), meta: { requiresAuth: true } },
    { path: '/tunnels/:id', name: 'tunnel-detail', component: () => import('../views/TunnelDetail.vue'), meta: { requiresAuth: true } },
    { path: '/monitors', name: 'monitors', component: () => import('../views/Monitors.vue'), meta: { requiresAuth: true } },
    { path: '/monitors/:id', name: 'monitor-detail', component: () => import('../views/MonitorDetail.vue'), meta: { requiresAuth: true } },
    { path: '/status/:token', name: 'public-status', component: () => import('../views/PublicStatus.vue'), meta: { public: true } },
    { path: '/domain', name: 'domain', component: () => import('../views/DomainBinding.vue'), meta: { requiresAuth: true } },
    { path: '/domain/batch', name: 'domain-batch', component: () => import('../views/BatchDomainBinding.vue'), meta: { requiresAuth: true } },
    { path: '/dns', name: 'dns', component: () => import('../views/DNSManagement.vue'), meta: { requiresAuth: true } },
    { path: '/lab/ip-selector', name: 'lab-ip-selector', component: LabIPSelector, meta: { requiresAuth: true, requiresAdmin: true, requiresExperimental: true } },
    { path: '/settings', name: 'settings', component: () => import('../views/Settings.vue'), meta: { requiresAuth: true } },
    { path: '/telegram', name: 'telegram', component: () => import('../views/TelegramSettings.vue'), meta: { requiresAuth: true } },
    { path: '/account', name: 'account', component: () => import('../views/Account.vue'), meta: { requiresAuth: true } },
    { path: '/notifications', name: 'notifications', component: () => import('../views/Notifications.vue'), meta: { requiresAuth: true } },
    { path: '/admin', name: 'admin', component: () => import('../views/Admin.vue'), meta: { requiresAuth: true, requiresAdmin: true } },
    { path: '/about', name: 'about', component: () => import('../views/About.vue'), meta: { requiresAuth: true } },
  ],
})

router.beforeEach(async (to, from) => {
  // An instance without an administrator serves the install wizard and nothing
  // else; a panel that is already installed answers 404 and never sees /setup.
  let wizard = await setupStatus()
  if (wizard && !to.meta.setup) {
    // Leaving the wizard means an install may have just finished: re-read once,
    // because a stale "needs setup" would send the sign-in page back to the
    // route we are on, which vue-router quietly turns into a no-op navigation.
    if (from.meta.setup) wizard = await setupStatus(true)
    if (wizard) return '/setup'
  }
  if (!wizard && to.meta.setup) return '/login'
  if (wizard) return true
  const store = useConfigStore()
  if (!store.siteSettingsLoaded) {
    await store.fetchSiteSettings()
  }
  if (to.path === '/') {
    // Landing page is the front door for everyone when enabled.
    if (store.landingEnabled) return true
    return store.isAuthenticated ? '/dashboard' : '/login'
  }
  if (to.meta.requiresAuth && !store.isAuthenticated) {
    return store.landingEnabled ? '/' : '/login'
  }
  if (to.meta.requiresAdmin && !store.isAdmin()) {
    return '/dashboard'
  }
  if (to.meta.requiresExperimental && !store.experimentalFeatures) {
    return '/dashboard'
  }
  if (to.path === '/login' && store.isAuthenticated) {
    return store.landingEnabled ? '/' : '/dashboard'
  }
})

export default router