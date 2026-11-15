import assert from 'node:assert/strict'
import test from 'node:test'
import { latestSlimRelease } from './slimRelease.ts'

test('slim updates cannot select a main release or draft regardless of numeric version', () => {
  const current = { tag_name: 'v3.1.0-slim', body: 'slim notes' }
  assert.equal(latestSlimRelease([{ tag_name: 'v100.0.0' }, { tag_name: 'v9.0.0-slim', draft: true }, { tag_name: 'v5.0.0-slim-other' }, current]), current)
  assert.equal(latestSlimRelease([{ tag_name: 'v9.9.9' }]), undefined)
})

test('slim updates compare semantic version parts rather than release order', () => {
  const newest = { tag_name: 'v3.10.0-slim' }
  assert.equal(latestSlimRelease([{ tag_name: 'v3.9.9-slim' }, newest, { tag_name: 'v2.99.99-slim' }]), newest)
})
