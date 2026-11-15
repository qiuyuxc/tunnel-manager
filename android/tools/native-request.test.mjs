import { test } from 'node:test'
import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { spawnSync } from 'node:child_process'

const root = resolve(import.meta.dirname, '..')
const executable = name => process.env.JAVA_HOME ? join(process.env.JAVA_HOME, 'bin', name) : name

test('authenticated URLs and multipart metadata reject origin and header injection', () => {
  const directory = mkdtempSync(join(tmpdir(), 'native-request-'))
  try {
    const compile = spawnSync(executable('javac'), ['-d', directory,
      resolve(root, 'app/src/main/java/com/tunnelmanager/app/WebOrigin.java'),
      resolve(root, 'app/src/main/java/com/tunnelmanager/app/NativeRequestPolicy.java'),
      resolve(root, 'tools/java/com/tunnelmanager/app/NativeRequestPolicyTest.java')], { encoding: 'utf8' })
    assert.equal(compile.status, 0, compile.error?.message || compile.stderr)
    const run = spawnSync(executable('java'), ['-cp', directory, 'com.tunnelmanager.app.NativeRequestPolicyTest'], { encoding: 'utf8' })
    assert.equal(run.status, 0, run.error?.message || run.stderr)
  } finally { rmSync(directory, { recursive: true, force: true }) }
})
