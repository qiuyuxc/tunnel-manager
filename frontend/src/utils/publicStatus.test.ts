import assert from 'node:assert/strict'
import test from 'node:test'
import { safePublicIcon, safePublicLink, statusSummary } from './publicStatus.ts'

test('empty or unobserved services cannot report all services healthy', () => {
  assert.equal(statusSummary([]).state, 'unknown')
  assert.equal(statusSummary([{ state: 'ok' }, { state: 'unknown' }]).state, 'unknown')
  assert.equal(statusSummary([{ state: '' }]).counts.unknown, 1)
  assert.equal(statusSummary([{ state: 'ok' }]).state, 'ok')
})

test('summary preserves outages and warnings alongside unknown targets', () => {
  assert.equal(statusSummary([{ state: 'down' }, { state: 'unknown' }]).state, 'down')
  assert.equal(statusSummary([{ state: 'warn' }, { state: 'unknown' }]).state, 'warn')
  assert.deepEqual(statusSummary([{ state: 'down' }, { state: 'ok' }]).counts, { ok: 1, warn: 0, down: 1, unknown: 0 })
})

test('public links block executable and malformed URLs and strip credentials', () => {
  assert.equal(safePublicLink('javascript:alert(1)'), '')
  assert.equal(safePublicLink('javascript://example.com/attack'), '')
  assert.equal(safePublicLink('data:text/html,<script>'), '')
  assert.equal(safePublicLink('https://u:p@example.com/path'), 'https://example.com/path')
  assert.equal(safePublicLink('https://example.com/path?q=hello'), 'https://example.com/path?q=hello')
})

test('public icons allow image uploads and raster data, reject path escapes and SVG data', () => {
  assert.equal(safePublicIcon('/uploads/icon.png'), '/uploads/icon.png')
  assert.equal(safePublicIcon('/uploads/../private'), '')
  assert.equal(safePublicIcon('data:image/svg+xml,<svg onload="alert(1)"/>'), '')
  assert.equal(safePublicIcon('data:image/png;base64,aGVsbG8='), 'data:image/png;base64,aGVsbG8=')
})
