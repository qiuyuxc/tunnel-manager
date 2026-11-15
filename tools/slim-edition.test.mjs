import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import test from 'node:test'

const root = resolve(import.meta.dirname, '..')
const read = file => readFileSync(resolve(root, file), 'utf8')

test('slim keeps registration, account administration, audit and bot commands absent', () => {
  for (const file of [
    'docs', 'backend/handlers/audit.go', 'backend/store/audit.go',
    'backend/handlers/oidc.go', 'backend/models/oidc.go', 'backend/services/oidc.go', 'backend/store/oidc.go',
    'backend/handlers/telegram.go', 'backend/services/telegram.go',
    'backend/services/telegram_commands.go', 'backend/services/user_telegram.go',
    'frontend/src/views/Admin.vue', 'frontend/src/views/TelegramSettings.vue',
    'android/app/src/main/java/com/tunnelmanager/app/AdminFragment.java',
    'android/app/src/main/java/com/tunnelmanager/app/TelegramFragment.java',
  ]) assert.equal(existsSync(resolve(root, file)), false, `${file} returned to the slim build`)
  const routes = read('backend/main.go')
  const removed = /^(?:\/auth\/(?:register|send-code|oidc)(?:\/|$)|\/account\/oidc(?:\/|$)|\/telegram(?:\/|$)|\/(?:admin\/)?(?:users|groups|invites|audit|oidc\/providers)(?:\/|$))/
  for (const [, route] of routes.matchAll(/r\.(?:Get|Post|Put|Delete)\("([^"]+)"/g)) {
    assert.ok(!removed.test(route), `removed API route ${route} returned`)
  }
  const frontendRoutes = read('frontend/src/router/index.ts')
  assert.doesNotMatch(frontendRoutes, /path:\s*['"]\/(?:admin|telegram)['"]|views\/(?:Admin|TelegramSettings)\.vue/)
})

test('shared monitoring, DNS, AI and security settings remain reachable', () => {
  const routes = read('backend/main.go')
  assert.ok(routes.includes('r.Route("/admin"'), 'global settings API group is missing')
  for (const route of ['/monitors', '/tunnels', '/zones', '/assistant/settings', '/settings', '/smtp', '/account/passkeys', '/public/status']) {
    assert.ok(routes.includes(route), `missing shared route ${route}`)
  }
  const webSettings = read('frontend/src/views/Settings.vue')
  for (const method of ['getAppSettings', 'updateAppSettings', 'getSMTP', 'updateSMTP', 'getOAuthConfig', 'saveEncryptionKey']) {
    assert.ok(webSettings.includes(method), `single-user settings lost ${method}`)
  }
})

test('slim version and update checks stay separate from the full edition', () => {
  const packageVersion = JSON.parse(read('frontend/package.json')).version
  const version = packageVersion.endsWith('-slim') ? packageVersion : `${packageVersion}-slim`
  assert.equal(read('backend/main.go').match(/const Version = "v([^"]+)"/)?.[1], version)
  assert.equal(read('android/app/build.gradle.kts').match(/versionName\s*=\s*"([^"]+)"/)?.[1], version)
  const webAbout = read('frontend/src/views/About.vue')
  const nativeAbout = read('android/app/src/main/java/com/tunnelmanager/app/AboutFragment.java')
  assert.ok(webAbout.includes('latestSlimRelease(releases)'), 'web update check must use the slim release filter')
  assert.ok(read('frontend/src/utils/slimRelease.ts').includes('-slim'), 'web release filter must distinguish the slim edition')
  assert.ok(nativeAbout.includes('-slim'), 'native update check must distinguish the slim edition')
  for (const source of [webAbout, nativeAbout]) {
    assert.ok(source.includes('releases?per_page='), 'update check cannot find the slim prerelease')
  }
})


test('slim publishing keeps the existing image paths and never tags latest', () => {
  const build = read('.github/workflows/build.yml')
  const release = read('.github/workflows/release.yml')
  assert.match(build, /branches: \[slim\/single-user\]/)
  assert.match(build, /tags: \['v\*-slim'\]/)
  assert.match(build, /GHCR_IMAGE: ghcr\.io\/\$\{\{ github\.repository_owner \}\}\/tunnel-manager\n/)
  assert.match(build, /type=raw,value=slim,/)
  assert.doesNotMatch(build, /type=raw,value=latest|type=semver/)
  assert.match(release, /tags: \['v\*-slim'\]/)
  assert.match(release, /draft: true/)
  assert.match(release, /make_latest: false/)
  assert.match(release, /prerelease: true/)
})
