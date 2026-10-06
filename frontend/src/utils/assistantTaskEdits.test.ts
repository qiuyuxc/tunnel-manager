import assert from 'node:assert/strict'
import { test } from 'node:test'
import { changeAssistantDNSType } from './assistantTaskEdits.ts'

test('manual DNS type changes keep IDs and TTL while clearing incompatible record values', () => {
  const original = { zone_id: 'zone', record_id: 'record', name: 'a.example.com', type: 'A', content: '192.0.2.1', ttl: 300, proxied: true }
  const srv = changeAssistantDNSType(original, 'SRV')
  assert.deepEqual(srv, { zone_id: 'zone', record_id: 'record', name: 'a.example.com', type: 'SRV', data: { priority: 0, weight: 0, port: 0, target: '' }, ttl: 300, proxied: false })
  assert.equal(original.content, '192.0.2.1', 'editing must not mutate confirmed arguments')
  const mx = changeAssistantDNSType(srv, 'MX')
  assert.equal(mx.data, undefined)
  assert.equal(mx.priority, 0)
  assert.equal(mx.content, '')
  assert.deepEqual(changeAssistantDNSType(original, 'A'), original)
})
