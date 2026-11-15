import { test } from 'node:test'
import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { spawnSync } from 'node:child_process'

const root = resolve(import.meta.dirname, '..')
const executable = name => process.env.JAVA_HOME ? join(process.env.JAVA_HOME, 'bin', name) : name

test('WebView trust requires the exact HTTP origin', () => {
  const directory = mkdtempSync(join(tmpdir(), 'web-origin-'))
  try {
    const compile = spawnSync(executable('javac'), ['-d', directory,
      resolve(root, 'app/src/main/java/com/tunnelmanager/app/WebOrigin.java'),
      resolve(root, 'tools/java/com/tunnelmanager/app/WebOriginTest.java')], { encoding: 'utf8' })
    assert.equal(compile.status, 0, compile.error?.message || compile.stderr)
    const run = spawnSync(executable('java'), ['-cp', directory, 'com.tunnelmanager.app.WebOriginTest'], { encoding: 'utf8' })
    assert.equal(run.status, 0, run.error?.message || run.stderr)
  } finally { rmSync(directory, { recursive: true, force: true }) }
})
