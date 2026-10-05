import assert from 'node:assert/strict'
import test from 'node:test'
import { api, createDNSRecord } from '../api/index.ts'
import { DNSBatchStoppedError } from './dnsValidation.ts'

test('DNS session validation runs before dispatch and receives the actual outgoing token', async () => {
  const previousStorage = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
  const previousAdapter = api.defaults.adapter
  let token = 'original-session'
  const sent: string[] = []
  Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: { getItem: () => token } })
  api.defaults.adapter = async config => {
    sent.push(String(config.headers['X-Auth-Token']))
    return { status: 201, statusText: 'Created', headers: {}, config, data: { id: 'created' } }
  }
  const record = { type: 'A' as const, name: 'www', content: '192.0.2.1', ttl: 1, proxied: false }
  const validateSession = (currentToken: string | null) => {
    if (currentToken !== 'original-session') throw new DNSBatchStoppedError('登录会话已变更')
  }
  try {
    await createDNSRecord('zone', record, validateSession)
    const pending = createDNSRecord('zone', record, validateSession)
    token = 'different-session'
    await assert.rejects(pending, DNSBatchStoppedError)
    assert.deepEqual(sent, ['original-session'])
    await createDNSRecord('zone', record)
    assert.deepEqual(sent, ['original-session', 'different-session'])
  } finally {
    api.defaults.adapter = previousAdapter
    if (previousStorage) Object.defineProperty(globalThis, 'localStorage', previousStorage)
    else Reflect.deleteProperty(globalThis, 'localStorage')
  }
})

test('an expired in-flight DNS batch request cannot clear a newer login session', async () => {
  const previousStorage = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
  const previousWindow = Object.getOwnPropertyDescriptor(globalThis, 'window')
  const previousAdapter = api.defaults.adapter
  let token = 'original-session'
  const location = { pathname: '/dns', href: '/dns' }
  Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: {
    getItem: () => token,
    removeItem: () => { token = '' },
  } })
  Object.defineProperty(globalThis, 'window', { configurable: true, value: { location } })
  api.defaults.adapter = async config => {
    token = 'newer-session'
    throw Object.assign(new Error('old session expired'), { config, response: { status: 401 } })
  }
  try {
    await assert.rejects(createDNSRecord('zone', { type: 'A', name: 'www', content: '192.0.2.1', ttl: 1, proxied: false }, currentToken => {
      assert.equal(currentToken, 'original-session')
    }), /old session expired/)
    assert.equal(token, 'newer-session')
    assert.equal(location.href, '/dns')
  } finally {
    api.defaults.adapter = previousAdapter
    if (previousStorage) Object.defineProperty(globalThis, 'localStorage', previousStorage)
    else Reflect.deleteProperty(globalThis, 'localStorage')
    if (previousWindow) Object.defineProperty(globalThis, 'window', previousWindow)
    else Reflect.deleteProperty(globalThis, 'window')
  }
})
