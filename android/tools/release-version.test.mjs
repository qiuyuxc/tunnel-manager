import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..')
const read = path => readFileSync(resolve(root, path), 'utf8')
const json = path => JSON.parse(read(path))
const version = json('frontend/package.json').version
const gradle = read('android/app/build.gradle.kts')

test('release versions agree across Android, backend and packages', () => {
  assert.match(version, /^\d+\.\d+\.\d+$/)
  assert.equal(gradle.match(/versionName\s*=\s*"([^"]+)"/)?.[1], version)
  assert.equal(read('backend/main.go').match(/const Version = "v([^"]+)"/)?.[1], version)
  assert.equal(json('docs/package.json').version, version)
  for (const directory of ['frontend', 'docs']) {
    const manifest = json(`${directory}/package.json`)
    const lock = json(`${directory}/package-lock.json`)
    assert.equal(lock.version, manifest.version, `${directory} lockfile version`)
    assert.equal(lock.packages[''].version, manifest.version, `${directory} root package version`)
  }
})

test('both Android packaging paths use the same version metadata', () => {
  const buildCode = gradle.match(/versionCode\s*=\s*(\d+)/)?.[1]
  assert.ok(Number(buildCode) > 0)
  const fallback = read('android/tools/build-nogradle.sh')
  assert.equal(fallback.match(/--version-code\s+(\d+)/)?.[1], buildCode)
  assert.equal(fallback.match(/--version-name\s+([\d.]+)/)?.[1], version)
})

test('both changelogs and web highlights describe the current release', () => {
  for (const file of ['docs/changelog.md', 'docs/en/changelog.md']) {
    assert.equal(read(file).match(/^## v([^\s]+)$/m)?.[1], version, file)
  }
  assert.ok(read('frontend/src/views/About.vue').includes(`v${version} 的重点变化`))
})
