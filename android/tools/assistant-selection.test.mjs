import { after, before, test } from 'node:test'
import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, resolve, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawnSync } from 'node:child_process'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const directory = mkdtempSync(join(tmpdir(), 'assistant-selection-'))
const executable = name => process.env.JAVA_HOME ? join(process.env.JAVA_HOME, 'bin', name) : name

before(() => {
  const result = spawnSync(executable('javac'), ['-encoding', 'UTF-8', '-d', directory,
    resolve(root, 'app/src/main/java/com/tunnelmanager/app/AssistantTaskSelection.java'),
    resolve(root, 'tools/java/com/tunnelmanager/app/AssistantTaskSelectionTest.java'),
  ], { encoding: 'utf8' })
  assert.equal(result.status, 0, result.error?.message || result.stderr)
})

after(() => rmSync(directory, { recursive: true, force: true }))

for (const scenario of ['explicit', 'refresh', 'isolation', 'snapshot', 'labels']) {
  test(`native assistant task selection: ${scenario}`, () => {
    const result = spawnSync(executable('java'), ['-cp', directory,
      'com.tunnelmanager.app.AssistantTaskSelectionTest', scenario,
    ], { encoding: 'utf8' })
    assert.equal(result.status, 0, result.error?.message || result.stderr)
  })
}
