import assert from 'node:assert/strict'
import { resolve } from 'node:path'
import { createServer } from 'vite'

// Execute the real router, Pinia store and Vue component setup through Vite.
// The small document fixture supplies browser state, without rendering pictures
// or contacting an actual panel. Business methods and Axios interceptors remain real.
const root = resolve(import.meta.dirname, '..')
const stored = new Map([['auth_token', 'first-session'], ['auth_username', 'first-user']])
const attributes = new Map()
const documentElement = {
  dataset: {}, lang: 'zh-CN',
  setAttribute: (name, value) => attributes.set(name, value),
  getAttribute: name => attributes.get(name) ?? null,
  removeAttribute: name => attributes.delete(name),
}
const documentFixture = {
  documentElement, title: '', hidden: false,
  querySelector: () => null,
  createElement: () => ({ dataset: {}, setAttribute() {}, remove() {}, content: {} }),
  head: { appendChild() {} },
  addEventListener() {}, removeEventListener() {},
}
const locationFixture = new URL('http://runtime.test/status/test-service')
const historyFixture = {
  state: null, length: 1,
  replaceState(state, _title, url) { this.state = state; if (url) locationFixture.href = new URL(url, locationFixture).href },
  pushState(state, _title, url) { this.state = state; this.length++; if (url) locationFixture.href = new URL(url, locationFixture).href },
  go() {},
}
const windowFixture = {
  location: locationFixture, history: historyFixture,
  document: documentFixture, pageXOffset: 0, pageYOffset: 0,
  addEventListener() {}, removeEventListener() {}, dispatchEvent() {},
  matchMedia: () => ({ matches: false, addListener() {}, removeListener() {}, addEventListener() {}, removeEventListener() {} }),
}
const globals = new Map()
for (const [name, value] of Object.entries({
  document: documentFixture, window: windowFixture, history: historyFixture, location: locationFixture,
  matchMedia: windowFixture.matchMedia,
  localStorage: { getItem: key => stored.get(key) ?? null, setItem: (key, value) => stored.set(key, String(value)), removeItem: key => stored.delete(key) },
  navigator: { languages: ['zh-CN'], language: 'zh-CN', userAgent: 'RuntimeCheck' },
})) {
  globals.set(name, Object.getOwnPropertyDescriptor(globalThis, name))
  Object.defineProperty(globalThis, name, { configurable: true, value })
}

const server = await createServer({ root, server: { middlewareMode: true, watch: null }, appType: 'custom', logLevel: 'error' })
let api
let oldAdapter
try {
  const vue = await import('vue')
  const pinia = await import('pinia')
  const routing = await import('vue-router')
  const { renderToString } = await import('@vue/server-renderer')
  const requests = []
  ;({ api } = await server.ssrLoadModule('/src/api/index.ts'))
  oldAdapter = api.defaults.adapter
  api.defaults.adapter = config => new Promise((resolveResponse, rejectResponse) => {
    requests.push({ config, url: config.url, token: config.headers['X-Auth-Token'], resolveResponse, rejectResponse, done: false })
  })
  const flush = async () => { for (let step = 0; step < 12; step++) await Promise.resolve() }
  const pending = (url, token, method = 'get') => requests.find(request => !request.done && request.url === url && request.config.method === method && (token === undefined || request.token === token))
  const reply = (url, data, token, method = 'get') => {
    const request = pending(url, token, method)
    assert.ok(request, `expected pending ${url} (${token ?? 'any session'})`)
    request.done = true
    request.resolveResponse({ status: 200, statusText: 'OK', headers: {}, config: request.config, data })
  }
  const count = url => requests.filter(request => request.url === url).length

  const { default: router } = await server.ssrLoadModule('/src/router/index.ts')
  await router.push('/status/test-service')
  await router.isReady()
  assert.equal(router.currentRoute.value.name, 'public-status')
  assert.equal(documentElement.dataset.entry, 'status')
  assert.deepEqual(requests, [], 'public route must not wait for setup, site branding or account APIs')
  const { default: RootApp } = await server.ssrLoadModule('/src/App.vue')
  const publicApp = vue.createSSRApp(RootApp)
  publicApp.use(pinia.createPinia())
  publicApp.use(router)
  const publicMarkup = await renderToString(publicApp)
  assert.match(publicMarkup, /class="pub-page"/, 'loading status renders its independent page')
  assert.doesNotMatch(publicMarkup, /class="(?:app-shell|sidebar|mobile-header)"/, 'status loading must not render console chrome')
  assert.deepEqual(requests, [], 'rendering public status also avoids panel bootstrap')
  const consoleModule = server.moduleGraph.getModuleById(resolve(root, 'src/ConsoleApp.vue'))
  assert.ok(!consoleModule?.ssrModule, 'public entry must not import console providers and navigation')
  console.log('PASS: public route bypasses panel bootstrap and console module loading')

  const { useConfigStore } = await server.ssrLoadModule('/src/stores/config.ts')
  const instance = pinia.createPinia()
  pinia.setActivePinia(instance)
  const store = useConfigStore(instance)
  const firstMe = store.fetchMe()
  const sameMe = store.fetchMe(false)
  const firstConfig = store.fetchConfig()
  const sameConfig = store.fetchConfig()
  await flush()
  assert.equal(count('/auth/me'), 1, 'concurrent identity requests are deduplicated')
  assert.equal(count('/config'), 1, 'concurrent config requests are deduplicated')

  store.token = 'next-session'
  store.username = 'next-user'
  localStorage.setItem('auth_token', 'next-session')
  const nextMe = store.fetchMe()
  const nextConfig = store.fetchConfig()
  await flush()
  reply('/auth/me', { username: 'next-user', role: 'user', permissions: ['monitors'], nickname: 'Current account' }, 'next-session')
  reply('/config', { tunnel_id: 'next-tunnel', tunnel_name: 'Current tunnel', site_name: 'Current panel', landing_enabled: false }, 'next-session')
  await Promise.all([nextMe, nextConfig])
  reply('/auth/me', { username: 'first-user', role: 'admin', permissions: ['admin'] }, 'first-session')
  reply('/config', { tunnel_id: 'first-tunnel', tunnel_name: 'Old tunnel', site_name: 'Old panel' }, 'first-session')
  await Promise.all([firstMe, sameMe, firstConfig, sameConfig])
  assert.equal(store.username, 'next-user')
  assert.equal(store.role, 'user')
  assert.deepEqual([...store.permissions], ['monitors'])
  assert.equal(store.config.tunnel_id, 'next-tunnel')
  assert.equal(store.config.site_name, 'Current panel')
  assert.equal(store.loading, false)
  await store.fetchMe(false)
  assert.equal(count('/auth/me'), 2, 'fresh identity cache is reused during navigation')
  console.log('PASS: concurrent reads deduplicate, and old-session responses cannot overwrite current identity/config')

  const legacyConfig = store.fetchConfig()
  await flush()
  reply('/config', { tunnel_id: 'legacy-tunnel', tunnel_name: '', site_name: 'Current panel' }, 'next-session')
  await legacyConfig
  await flush()
  assert.ok(pending('/tunnels'), 'legacy tunnel display lookup is still available')
  assert.equal(store.loading, false, 'configuration is usable before slow Cloudflare display lookup completes')
  store.config.tunnel_id = 'manually-selected'
  store.config.tunnel_name = 'Manual selection'
  reply('/tunnels', [{ id: 'legacy-tunnel', name: 'Late lookup result' }])
  await flush()
  assert.equal(store.config.tunnel_id, 'manually-selected')
  assert.equal(store.config.tunnel_name, 'Manual selection')
  assert.ok(!requests.some(request => request.config.method !== 'get'), 'display lookup must not mutate the server')
  console.log('PASS: slow legacy name resolution does not block config or replace a newer selection')

  // SSR mounts the real setup function with its real provider; omitting its
  // template isolates data/form behavior and needs no browser DOM renderer.
  Reflect.deleteProperty(globalThis, 'window')
  const { default: MonitorDetail } = await server.ssrLoadModule('/src/views/MonitorDetail.vue')
  const { NMessageProvider, NDialogProvider } = await import('naive-ui')
  async function componentState(Component, params) {
    const fixture = { ...Component, ssrRender: undefined, render: () => null }
    let result
    const app = vue.createSSRApp({ render: () => vue.h(NMessageProvider, null, {
      default: () => vue.h(NDialogProvider, null, { default: () => vue.h(fixture) }),
    }) })
    app.use(instance)
    app.provide(routing.routeLocationKey, { params })
    app.provide(routing.routerKey, router)
    app.mixin({ created() { if (this.$options.setup === Component.setup) result = this.$.setupState } })
    await renderToString(app)
    return result
  }
  const state = await componentState(MonitorDetail, { id: 'test-monitor' })
  assert.ok(state?.load, 'monitor setup exposes the real polling load function to the runtime harness')
  Object.defineProperty(globalThis, 'window', { configurable: true, value: windowFixture })
  const firstLoad = state.load()
  await flush()
  reply('/monitors/test-monitor', { id: 'test-monitor', name: 'Test service', targets: [], public_title: 'Stored title', announcement: 'Stored notice', alert_enabled: true, alert_emails: 'stored@example.test' })
  await firstLoad
  assert.equal(state.pubTitle, 'Stored title')
  state.pubTitle = 'Unsaved title'
  state.announcement = 'Unsaved notice'
  state.alertEnabled = false
  state.alertEmails = 'draft@example.test'
  const poll = state.load()
  await flush()
  reply('/monitors/test-monitor', { id: 'test-monitor', name: 'Updated observation', targets: [], public_title: 'Remote title', announcement: 'Remote notice', alert_enabled: true, alert_emails: 'remote@example.test' })
  await poll
  assert.equal(state.monitor.name, 'Updated observation')
  assert.equal(state.pubTitle, 'Unsaved title')
  assert.equal(state.announcement, 'Unsaved notice')
  assert.equal(state.alertEnabled, false)
  assert.equal(state.alertEmails, 'draft@example.test')
  console.log('PASS: monitor polling updates observations while preserving unsaved public-page and alert fields')
  async function verifyForcedRefresh(state, method, url, original, stale, fresh, snapshot) {
    const baseline = state[method]()
    await flush()
    reply(url, original)
    await baseline
    const before = snapshot(state)
    const previousCount = count(url)
    const oldRead = state[method]()
    const sameRead = state[method]()
    await flush()
    assert.equal(count(url), previousCount + 1, `${url}: ordinary concurrent reads merge`)
    const afterWrite = state[method](true)
    reply(url, stale)
    await Promise.all([oldRead, sameRead])
    await flush()
    assert.deepEqual(snapshot(state), before, `${url}: invalidated pre-write response cannot overwrite current state`)
    assert.equal(count(url), previousCount + 2, `${url}: post-write refresh must send a fresh read`)
    reply(url, fresh)
    await afterWrite
    assert.notDeepEqual(snapshot(state), before, `${url}: fresh post-write response is applied`)
  }
  const { default: Tunnels } = await server.ssrLoadModule('/src/views/Tunnels.vue')
  const tunnelsState = await componentState(Tunnels, {})
  await verifyForcedRefresh(tunnelsState, 'loadTunnels', '/tunnels',
    [{ id: 'initial', name: 'Initial', status: 'healthy' }],
    [{ id: 'old', name: 'Old pending read', status: 'healthy' }],
    [{ id: 'current', name: 'Current after write', status: 'healthy' }],
    state => [...state.tunnels].map(tunnel => tunnel.id))
  const oldSelectionConfig = store.fetchConfig()
  await flush()
  const selection = tunnelsState.selectTunnel({ id: 'selected-tunnel', name: 'Selected tunnel', status: 'healthy' })
  await flush()
  reply('/config/tunnel', {}, undefined, 'post')
  await selection
  reply('/config', { tunnel_id: 'before-selection', tunnel_name: 'Obsolete config', site_name: 'Current panel' })
  await oldSelectionConfig
  assert.equal(store.config.tunnel_id, 'selected-tunnel', 'selection invalidates older pending configuration read')
  assert.equal(store.config.tunnel_name, 'Selected tunnel')
  console.log('PASS: selecting a tunnel during config loading cannot be undone by the older config response')
  const { default: TunnelDetail } = await server.ssrLoadModule('/src/views/TunnelDetail.vue')
  const detailState = await componentState(TunnelDetail, { id: 'test-tunnel' })
  await verifyForcedRefresh(detailState, 'load', '/tunnels/test-tunnel',
    { id: 'test-tunnel', name: 'Initial', ingress: [] },
    { id: 'test-tunnel', name: 'Old pending read', ingress: [] },
    { id: 'test-tunnel', name: 'Current after write', ingress: [] },
    state => state.detail.name)
  await verifyForcedRefresh(state, 'load', '/monitors/test-monitor',
    { id: 'test-monitor', name: 'Initial', targets: [], publish_enabled: false },
    { id: 'test-monitor', name: 'Old pending read', targets: [], publish_enabled: false },
    { id: 'test-monitor', name: 'Current after write', targets: [], publish_enabled: false },
    state => state.monitor.name)
  console.log('PASS: tunnels, tunnel detail and monitor detail reread after writes and reject obsolete pending snapshots')

  // Exercise an actual mutation callback while a poll is already pending.
  const oldPoll = state.load()
  await flush()
  const toggle = state.togglePublish()
  await flush()
  reply('/monitors/test-monitor', { id: 'test-monitor', name: 'Published service', targets: [], publish_enabled: true }, undefined, 'put')
  await flush()
  assert.equal(state.monitor.publish_enabled, true, 'publish mutation applies returned state')
  reply('/monitors/test-monitor', { id: 'test-monitor', name: 'Old poll before publishing', targets: [], publish_enabled: false })
  await oldPoll
  await flush()
  assert.equal(state.monitor.publish_enabled, true, 'pending pre-mutation poll cannot undo publishing')
  if (pending('/monitors/test-monitor')) {
    reply('/monitors/test-monitor', { id: 'test-monitor', name: 'Published service', targets: [], publish_enabled: true })
  }
  await toggle
  assert.equal(state.monitor.publish_enabled, true)
  console.log('PASS: publishing during a pending poll remains published when that older poll resolves')
  router.options.history.destroy()
} finally {
  if (api) api.defaults.adapter = oldAdapter
  await server.close()
  for (const [name, descriptor] of globals) {
    if (descriptor) Object.defineProperty(globalThis, name, descriptor)
    else Reflect.deleteProperty(globalThis, name)
  }
}
