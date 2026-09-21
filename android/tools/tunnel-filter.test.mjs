import { after, before, test } from 'node:test'
import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, resolve, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawnSync } from 'node:child_process'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const directory = mkdtempSync(join(tmpdir(), 'tunnel-filter-'))
const executable = name => process.env.JAVA_HOME ? join(process.env.JAVA_HOME, 'bin', name) : name

before(() => {
  const result = spawnSync(executable('javac'), ['-encoding', 'UTF-8', '-d', directory,
    resolve(root, 'app/src/main/java/com/tunnelmanager/app/TunnelFilter.java'),
    resolve(root, 'tools/java/com/tunnelmanager/app/TunnelFilterTest.java'),
  ], { encoding: 'utf8' })
  assert.equal(result.status, 0, result.error?.message || result.stderr)
})

after(() => rmSync(directory, { recursive: true, force: true }))

for (const scenario of ['status', 'search', 'combined', 'locale']) {
  test(`native tunnel filtering: ${scenario}`, () => {
    const result = spawnSync(executable('java'), ['-cp', directory,
      'com.tunnelmanager.app.TunnelFilterTest', scenario,
    ], { encoding: 'utf8' })
    assert.equal(result.status, 0, result.error?.message || result.stderr)
  })
}
