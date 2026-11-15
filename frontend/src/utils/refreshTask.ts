/** Merge reads; after a write, invalidate the old snapshot and read again. */
export function createRefreshTask(read: (isCurrent: () => boolean, signal: AbortSignal) => Promise<void>) {
  let pending: Promise<void> | null = null
  let generation = 0
  let disposed = false
  let controller: AbortController | null = null

  function refresh(afterWrite = false): Promise<void> {
    if (disposed) return Promise.resolve()
    if (pending) {
      if (!afterWrite) return pending
      generation++
      controller?.abort()
      return pending.catch(() => {}).then(() => refresh())
    }
    const current = ++generation
    controller = new AbortController()
    const signal = controller.signal
    const request = Promise.resolve().then(() => {
      if (!disposed) return read(() => !disposed && current === generation, signal)
    }).finally(() => { if (pending === request) { pending = null; controller = null } })
    pending = request
    return request
  }
  refresh.dispose = () => { disposed = true; generation++; controller?.abort() }
  refresh.invalidate = () => { generation++; controller?.abort() }
  return refresh
}
