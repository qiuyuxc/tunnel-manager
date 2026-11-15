import { after, before, test } from 'node:test'
import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { spawnSync } from 'node:child_process'

const root = resolve(import.meta.dirname, '..')
const directory = mkdtempSync(join(tmpdir(), 'sheet-sizing-'))
const executable = name => process.env.JAVA_HOME ? join(process.env.JAVA_HOME, 'bin', name) : name
before(() => {
  const compile = spawnSync(executable('javac'), ['-d', directory,
    resolve(root, 'app/src/main/java/com/tunnelmanager/app/SheetSizing.java'),
    resolve(root, 'tools/java/com/tunnelmanager/app/SheetSizingTest.java')], { encoding: 'utf8' })
  assert.equal(compile.status, 0, compile.error?.message || compile.stderr)
})
after(() => rmSync(directory, { recursive: true, force: true }))
for (const scenario of ['fixed-scenes', 'keyboard-bounds', 'rotation']) {
  test(`native sheet sizing: ${scenario}`, () => {
    const run = spawnSync(executable('java'), ['-cp', directory, 'com.tunnelmanager.app.SheetSizingTest', scenario], { encoding: 'utf8' })
    assert.equal(run.status, 0, run.error?.message || run.stderr)
  })
}
