import assert from 'node:assert/strict'
import test from 'node:test'
import { createRefreshTask } from './refreshTask.ts'

test('a write during an ongoing read discards that snapshot and waits for fresh data', async () => {
  let releaseOld!: () => void
  let requests = 0
  let shown = 'original'
  const refresh = createRefreshTask(async isCurrent => {
    const request = ++requests
    if (request === 1) await new Promise<void>(resolve => { releaseOld = resolve })
    if (isCurrent()) shown = request === 1 ? 'before-write' : 'after-write'
  })
  const first = refresh()
  assert.equal(refresh(), first)
  await Promise.resolve()
  const afterWrite = refresh(true)
  releaseOld()
  await first
  assert.equal(shown, 'original')
  await afterWrite
  assert.equal(shown, 'after-write')
  assert.equal(requests, 2)
})

test('multiple writes awaiting the same read merge into one fresh snapshot', async () => {
  let release!: () => void
  let requests = 0
  const refresh = createRefreshTask(async () => {
    requests++
    if (requests === 1) await new Promise<void>(resolve => { release = resolve })
  })
  void refresh()
  await Promise.resolve()
  const a = refresh(true), b = refresh(true)
  release()
  await Promise.all([a, b])
  assert.equal(requests, 2)
})

test('leaving a page invalidates pending data and prevents queued or future reads', async () => {
  let release!: () => void
  let requests = 0, applied = false
  const refresh = createRefreshTask(async isCurrent => {
    requests++
    await new Promise<void>(resolve => { release = resolve })
    applied = isCurrent()
  })
  const pending = refresh()
  await Promise.resolve()
  const queued = refresh(true)
  refresh.dispose()
  release()
  await Promise.all([pending, queued, refresh()])
  assert.equal(applied, false)
  assert.equal(requests, 1)
})

test('a pending API read is cancelled immediately before the post-write refresh', async () => {
  let requests = 0
  const signals: AbortSignal[] = []
  const refresh = createRefreshTask(async (_current, signal) => {
    requests++
    signals.push(signal)
    if (requests === 1) await new Promise<void>((_resolve, reject) => {
      signal.addEventListener('abort', () => reject(new Error('cancelled')), { once: true })
    })
  })
  const first = refresh().catch(error => assert.match(error.message, /cancelled/))
  await Promise.resolve()
  await refresh(true)
  await first
  assert.equal(signals[0]?.aborted, true)
  assert.equal(signals[1]?.aborted, false)
  assert.equal(requests, 2)
})

test('a mutation response invalidates a poll without scheduling a redundant read', async () => {
  let release!: () => void
  let displayed = 'before', requests = 0
  const refresh = createRefreshTask(async isCurrent => {
    requests++
    await new Promise<void>(resolve => { release = resolve })
    if (isCurrent()) displayed = 'stale poll'
  })
  const pending = refresh()
  await Promise.resolve()
  refresh.invalidate()
  displayed = 'mutation response'
  release()
  await pending
  assert.equal(displayed, 'mutation response')
  assert.equal(requests, 1)
})
