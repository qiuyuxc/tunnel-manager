import assert from 'node:assert/strict'
import test from 'node:test'
import { filterMonitors } from './monitorFilters.ts'

const monitors = [
  { name: '生产环境', publish_enabled: true, targets: [{ name: 'Gateway', url: 'https://API.example.com', state: 'ok' }] },
  { name: '备份', publish_enabled: false, targets: [{ name: 'Database', url: 'db.example.net:5432', state: 'down' }] },
  { name: '首次检查', publish_enabled: true, targets: [{ name: '新服务', url: 'new.example.net' }] },
]

test('monitor searches find project names, target names and domains without case sensitivity', () => {
  assert.deepEqual(filterMonitors(monitors, '生产', 'all'), [monitors[0]])
  assert.deepEqual(filterMonitors(monitors, ' gateway ', 'all'), [monitors[0]])
  assert.deepEqual(filterMonitors(monitors, 'API.EXAMPLE.COM', 'all'), [monitors[0]])
})

test('monitor filters combine with search and do not classify missing checks as outages', () => {
  assert.deepEqual(filterMonitors(monitors, '', 'issues'), [monitors[1]])
  assert.deepEqual(filterMonitors(monitors, '', 'public'), [monitors[0], monitors[2]])
  assert.deepEqual(filterMonitors(monitors, '备份', 'public'), [])
  assert.deepEqual(filterMonitors(monitors, '首次', 'issues'), [])
})
