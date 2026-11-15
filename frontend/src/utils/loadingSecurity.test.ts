import assert from 'node:assert/strict'
import test from 'node:test'
import { api, getSiteSettings } from '../api/index.ts'
import { setupStatus, waitForPanelRestart } from '../api/setup.ts'
import { fetchPublicStatus } from '../api/publicStatus.ts'

test('panel credentials stay on the panel API and public bootstrap is anonymous', async () => {
  const oldStorage = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
  const oldWindow = Object.getOwnPropertyDescriptor(globalThis, 'window')
  const oldAdapter = api.defaults.adapter
  const sent: Array<{ url?: string; token: unknown }> = []
  Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: { getItem: () => 'panel-session' } })
  Object.defineProperty(globalThis, 'window', { configurable: true, value: { location: { origin: 'https://panel.example', pathname: '/dashboard' } } })
  api.defaults.adapter = async config => {
    sent.push({ url: config.url, token: config.headers['X-Auth-Token'] })
    return { status: 200, statusText: 'OK', headers: {}, config, data: {} }
  }
  try {
    await api.get('/config')
    await getSiteSettings()
    await api.get('https://external.example/api/config', { headers: { 'X-Auth-Token': 'explicit-token' } })
    await api.get('https://panel.example/private')
    assert.equal(sent[0]?.token, 'panel-session')
    assert.equal(sent[1]?.token, undefined)
    assert.equal(sent[2]?.token, undefined)
    assert.equal(sent[3]?.token, undefined)
  } finally {
    api.defaults.adapter = oldAdapter
    if (oldStorage) Object.defineProperty(globalThis, 'localStorage', oldStorage)
    else Reflect.deleteProperty(globalThis, 'localStorage')
    if (oldWindow) Object.defineProperty(globalThis, 'window', oldWindow)
    else Reflect.deleteProperty(globalThis, 'window')
  }
})

test('setup bootstrap retries temporary failures, shares concurrent reads, and caches confirmed 404', async () => {
  const oldAdapter = api.defaults.adapter
  const oldStorage = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
  let calls = 0
  let retrySucceeded = false
  Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: { getItem: () => null } })
  api.defaults.adapter = async config => {
    calls++
    await new Promise(resolve => setTimeout(resolve, 10))
    throw Object.assign(new Error('upstream unavailable'), { config, response: { status: retrySucceeded ? 404 : 502 } })
  }
  try {
    assert.deepEqual(await Promise.all([setupStatus(), setupStatus()]), [null, null])
    assert.equal(calls, 1)
    retrySucceeded = true
    assert.equal(await setupStatus(), null)
    assert.equal(calls, 2)
    assert.equal(await setupStatus(), null)
    assert.equal(calls, 2)
  } finally {
    api.defaults.adapter = oldAdapter
    if (oldStorage) Object.defineProperty(globalThis, 'localStorage', oldStorage)
    else Reflect.deleteProperty(globalThis, 'localStorage')
  }
})

test('a gateway error is not mistaken for a completed setup restart', async () => {
  const oldAdapter = api.defaults.adapter
  const oldStorage = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
  let calls = 0
  Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: { getItem: () => null } })
  api.defaults.adapter = async config => {
    calls++
    return { status: calls === 1 ? 502 : 404, statusText: '', headers: {}, config, data: {} }
  }
  try {
    assert.equal(await waitForPanelRestart(5000), true)
    assert.equal(calls, 2)
  } finally {
    api.defaults.adapter = oldAdapter
    if (oldStorage) Object.defineProperty(globalThis, 'localStorage', oldStorage)
    else Reflect.deleteProperty(globalThis, 'localStorage')
  }
})

test('public status requests encode identifiers, omit sessions and preserve cancellation', async () => {
  const previousFetch = globalThis.fetch
  const controller = new AbortController()
  const observed: Array<{ url: string; init?: RequestInit }> = []
  globalThis.fetch = async (input, init) => {
    observed.push({ url: String(input), init })
    return new Response(JSON.stringify({ name: 'Public service' }), { status: 200 })
  }
  try {
    assert.equal((await fetchPublicStatus('name/with?query', controller.signal)).name, 'Public service')
    assert.equal(observed[0]?.url, '/api/public/status/name%2Fwith%3Fquery')
    assert.equal(observed[0]?.init?.credentials, 'omit')
    assert.equal(observed[0]?.init?.signal, controller.signal)
    assert.equal(observed[0]?.init?.headers, undefined)
  } finally {
    globalThis.fetch = previousFetch
  }
})
