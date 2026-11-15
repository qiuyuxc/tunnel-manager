export type MonitorFilter = 'all' | 'issues' | 'public'
type SearchableMonitor = {
  name: string
  publish_enabled: boolean
  targets: ReadonlyArray<{ name: string; url: string; state?: string }>
}

export function filterMonitors<T extends SearchableMonitor>(monitors: ReadonlyArray<T>, query: string, filter: MonitorFilter): T[] {
  const needle = query.trim().toLocaleLowerCase()
  return monitors.filter(monitor => {
    if (filter === 'public' && !monitor.publish_enabled) return false
    if (filter === 'issues' && !monitor.targets.some(target => target.state === 'warn' || target.state === 'down')) return false
    if (!needle) return true
    return [monitor.name, ...monitor.targets.flatMap(target => [target.name, target.url])].some(text => text.toLocaleLowerCase().includes(needle))
  })
}
