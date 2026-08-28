import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'
import Login from '../views/Login.vue'
import Landing from '../views/Landing.vue'
import { useConfigStore } from '../stores/config'

const router = createRouter({
  history: createWebHistory(),
  routes: [
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
    { path: '/settings', name: 'settings', component: () => import('../views/Settings.vue'), meta: { requiresAuth: true } },
    { path: '/telegram', name: 'telegram', component: () => import('../views/TelegramSettings.vue'), meta: { requiresAuth: true } },
    { path: '/account', name: 'account', component: () => import('../views/Account.vue'), meta: { requiresAuth: true } },
    { path: '/notifications', name: 'notifications', component: () => import('../views/Notifications.vue'), meta: { requiresAuth: true } },
    { path: '/admin', name: 'admin', component: () => import('../views/Admin.vue'), meta: { requiresAuth: true, requiresAdmin: true } },
    { path: '/about', name: 'about', component: () => import('../views/About.vue'), meta: { requiresAuth: true } },
  ],
})

router.beforeEach(async (to, _from) => {
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
  if (to.path === '/login' && store.isAuthenticated) {
    return store.landingEnabled ? '/' : '/dashboard'
  }
})

export default router