import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..')
const read = path => readFileSync(resolve(root, path), 'utf8')
const json = path => JSON.parse(read(path))
const version = json('frontend/package.json').version
const releaseVersion = `${version}-slim`
const gradle = read('android/app/build.gradle.kts')

test('release versions agree across Android, backend and packages', () => {
  assert.match(version, /^\d+\.\d+\.\d+$/)
  assert.equal(gradle.match(/versionName\s*=\s*"([^"]+)"/)?.[1], releaseVersion)
  assert.equal(read('backend/main.go').match(/const Version = "v([^"]+)"/)?.[1], releaseVersion)
  const lock = json('frontend/package-lock.json')
  assert.equal(lock.version, version, 'frontend lockfile version')
  assert.equal(lock.packages[''].version, version, 'frontend root package version')
})

test('both Android packaging paths use the same version metadata', () => {
  const buildCode = gradle.match(/versionCode\s*=\s*(\d+)/)?.[1]
  assert.ok(Number(buildCode) > 0)
  const fallback = read('android/tools/build-nogradle.sh')
  assert.equal(fallback.match(/--version-code\s+(\d+)/)?.[1], buildCode)
  assert.equal(fallback.match(/--version-name\s+(\S+)/)?.[1], releaseVersion)
})

test('web highlights describe the current slim release', () => {
  assert.ok(read('frontend/src/views/About.vue').includes(`v${releaseVersion} 的重点变化`))
})

test('native and web release highlights stay aligned', () => {
  const native = read('android/app/src/main/java/com/tunnelmanager/app/AboutFragment.java')
    .match(/String\[\] HIGHLIGHTS = \{([\s\S]*?)\n    \};/)?.[1] || ''
  const web = read('frontend/src/views/About.vue')
    .match(/<ul class="changelog-list">([\s\S]*?)<\/ul>/)?.[1] || ''
  const nativeItems = [...native.matchAll(/"([^"]+)"/g)].map(match => match[1])
  const webItems = [...web.matchAll(/<li>([^<]+)<\/li>/g)].map(match => match[1])
  assert.ok(nativeItems.length > 0)
  assert.deepEqual(nativeItems, webItems)
})
