import assert from 'node:assert/strict'
import { mkdir, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const origin = process.env.LAB_TEST_ORIGIN || 'http://127.0.0.1:18084'
const debuggerOrigin = process.env.LAB_TEST_CDP || 'http://127.0.0.1:19223'
const output = process.env.LAB_TEST_OUTPUT || join(tmpdir(), 'lab-screenshots')
assert.ok(['127.0.0.1', 'localhost', '[::1]'].includes(new URL(origin).hostname))
assert.ok(['127.0.0.1', 'localhost', '[::1]'].includes(new URL(debuggerOrigin).hostname))
const target = await (await fetch(`${debuggerOrigin}/json/new?about:blank`, { method: 'PUT' })).json()
const socket = new WebSocket(target.webSocketDebuggerUrl)
await new Promise(resolve => socket.addEventListener('open', resolve, { once: true }))
const pending = new Map()
const failures = []
const mutations = []
let sequence = 0
let role = 'admin'
let enabled = true
let txtPresent = false
let dnsBound = false
let dnsDenied = false
const dnsRecords = new Set()
let statusUnavailable = false
let statusRequests = 0
let labRequests = 0
let runs = []
let adminSaves = 0
let releaseListRequested = false
let appSettings = { experimental_features_enabled: true,
  lab_daily_request_limit: 100000, lab_requests_per_second: 10, lab_max_workers: 32 }
let settings = {
  host: 'probe.example.com', sni: 'probe.example.com', path: '/ip-check.txt', statuses: '200',
  timeout: 2, workers: 32, top: 10, ip_targets: '1.1.1.1\n1.0.0.1', schedule: false,
  interval_minutes: 30, update_dns: false, zone: 'example.com', zone_id: '',
  record: 'edge.example.com', ttl: 300, endpoint: 'https://dns.myhuaweicloud.com',
  access_key: 'test-access', has_secret_key: false,
}
const state = {
  running: false, phase: '', verification: {}, suggested_domain: '',
  budget_day: '2026-10-04', budget_used: 0, budget_limit: 100000, requests_per_second: 10,
  max_workers: 32, next_allowed_at: 0, last_error: '',
  progress: { scanned: 0, matched: 0, total: 0, percent: 0 },
}

function command(method, params = {}) {
  const id = ++sequence
  const result = new Promise((resolve, reject) => pending.set(id, { resolve, reject }))
  socket.send(JSON.stringify({ id, method, params }))
  return result
}

socket.addEventListener('message', async event => {
  const message = JSON.parse(event.data)
  if (message.id) {
    const request = pending.get(message.id)
    pending.delete(message.id)
    if (message.error) request?.reject(new Error(JSON.stringify(message.error)))
    else request?.resolve(message.result)
    return
  }
  if (message.method === 'Runtime.exceptionThrown') failures.push(message.params.exceptionDetails.text)
  if (message.method === 'Page.javascriptDialogOpening') {
    await command('Page.handleJavaScriptDialog', { accept: true })
    return
  }
  if (message.method !== 'Fetch.requestPaused') return
  const { requestId, request } = message.params
  const url = new URL(request.url)
  if (url.href === 'https://api.github.com/repos/qiuyuxc/tunnel-manager/releases?per_page=100') {
    releaseListRequested = true
    await command('Fetch.fulfillRequest', { requestId, responseCode: 200,
      responseHeaders: [{ name: 'Content-Type', value: 'application/json' },
        { name: 'Access-Control-Allow-Origin', value: origin }],
      body: Buffer.from(JSON.stringify([
        { tag_name: 'v9.0.0', body: 'Full edition', draft: false },
        { tag_name: 'v2.7.0-slim', body: 'Slim edition', draft: false, prerelease: true },
      ])).toString('base64') })
    return
  }
  if (url.origin !== origin) {
    failures.push(`Blocked external request: ${url.origin}`)
    await command('Fetch.failRequest', { requestId, errorReason: 'BlockedByClient' })
    return
  }
  if (!url.pathname.startsWith('/api/')) {
    await command('Fetch.continueRequest', { requestId })
    return
  }
  let status = 200
  let result = {}
  const body = request.postData ? JSON.parse(request.postData) : {}
  if (url.pathname === '/api/setup/status') status = 404
  else if (url.pathname === '/api/health') result = { version: '2.7.0-slim' }
  else if (url.pathname === '/api/site') result = { name: 'Tunnel Manager', icon: '', landing_enabled: false, experimental_features_enabled: enabled }
  else if (url.pathname === '/api/auth/me') result = { id: 'test-admin', username: 'reviewer', role, permissions: [] }
  else if (url.pathname === '/api/config') result = { site_name: 'Tunnel Manager', experimental_features_enabled: enabled, cname_presets: [] }
  else if (url.pathname === '/api/monitors' || url.pathname === '/api/tunnels') result = []
  else if (url.pathname === '/api/admin/settings') {
    if (request.method === 'PUT') {
      adminSaves++
      appSettings = { ...appSettings, ...body }
      state.budget_limit = appSettings.lab_daily_request_limit
      state.requests_per_second = appSettings.lab_requests_per_second
      state.max_workers = appSettings.lab_max_workers
    }
    result = appSettings
  }
  else if (url.pathname.startsWith('/api/lab/ip-selector')) {
    labRequests++
    if (!enabled || role !== 'admin') {
      status = enabled ? 403 : 404
      result = { error: '实验室不可访问' }
    } else if (url.pathname.endsWith('/status')) {
      statusRequests++
      status = statusUnavailable ? 503 : 200
      result = statusUnavailable ? { error: '模拟网络故障' } : { status: state, runs }
    } else {
      if (request.method !== 'GET') mutations.push({ path: url.pathname, body })
      if (url.pathname.endsWith('/ownership/dns')) {
        if (request.method === 'GET') {
          result = { available: dnsBound, account_name: dnsBound ? '测试绑定账户' : '',
            message: dnsBound ? '使用当前绑定的 Cloudflare 账户' : '当前账户未绑定 Cloudflare，请手动添加 TXT' }
        } else if (!dnsBound || dnsDenied) {
          status = 400
          result = { error: '当前账户没有该域名的 DNS 写入权限，请手动添加 TXT' }
        } else {
          if (!state.verification.token || state.verification.domain !== body.domain) {
            state.verification = { domain: body.domain, host: settings.host, sni: settings.sni,
              owner_id: 'test-admin', token: `tunnel-manager=${'c'.repeat(64)}.${'d'.repeat(64)}`,
              record_name: `_tunnel-manager-verify.${body.domain}`, verified_at: 0, issued_at: 1, checked_at: 0, last_error: '' }
          }
          const created = !dnsRecords.has(state.verification.token)
          dnsRecords.add(state.verification.token)
          settings.schedule = false
          state.verification.verified_at = txtPresent ? 123 : 0
          result = { verification: state.verification, record_created: created, verified: txtPresent,
            message: txtPresent ? 'TXT 已写入并通过 DNS 验证，定时任务仍关闭。' : 'TXT 已提交，请等待解析生效后点击检查 TXT，不必重新生成记录。' }
        }
      } else if (url.pathname.endsWith('/ownership/challenge')) {
        state.verification = { domain: body.domain, host: settings.host, sni: settings.sni,
          owner_id: 'test-admin', token: `tunnel-manager=${'a'.repeat(64)}.${'b'.repeat(64)}`,
          record_name: `_tunnel-manager-verify.${body.domain}`, verified_at: 0, issued_at: 1, checked_at: 0, last_error: '' }
        settings.schedule = false
        result = state.verification
      } else if (url.pathname.endsWith('/ownership/verify')) {
        state.verification.verified_at = txtPresent ? 123 : 0
        state.verification.last_error = txtPresent ? '' : 'DNS TXT 验证值未找到'
        status = txtPresent ? 200 : 403
        result = txtPresent ? state.verification : { error: state.verification.last_error, verification: state.verification }
      } else if (url.pathname.endsWith('/run')) {
        assert.ok(state.verification.verified_at)
        state.running = true
        state.phase = 'scanning'
        state.progress = { scanned: 1, matched: 1, total: 2, percent: 50 }
        state.budget_used += 2
        state.next_allowed_at = Math.floor(Date.now() / 1000) + 60
        status = 202
        result = { started: true }
      } else if (url.pathname.endsWith('/stop')) {
        state.running = false
        state.phase = 'failed'
        settings.schedule = false
        result = { stopped: true }
      } else {
        if (request.method === 'PUT') {
          const { secret_key: secret, ...safe } = body
          settings = { ...settings, ...safe, has_secret_key: settings.has_secret_key || !!secret }
        }
        result = settings
      }
    }
  }
  await command('Fetch.fulfillRequest', { requestId, responseCode: status,
    responseHeaders: [{ name: 'Content-Type', value: 'application/json' }], body: Buffer.from(JSON.stringify(result)).toString('base64') })
})

const delay = milliseconds => new Promise(resolve => setTimeout(resolve, milliseconds))
async function evaluate(expression) {
  const result = await command('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true })
  if (result.exceptionDetails) throw new Error(JSON.stringify(result.exceptionDetails))
  return result.result.value
}
async function waitFor(expression) {
  for (let attempt = 0; attempt < 100; attempt++) {
    if (await evaluate(expression)) return
    await delay(100)
  }
  throw new Error(`Timed out: ${expression}`)
}
const button = text => `Array.from(document.querySelectorAll('button')).find(button => button.textContent.trim() === ${JSON.stringify(text)})`
async function click(text) {
  await waitFor(`${button(text)} && !${button(text)}.disabled`)
  await evaluate(`${button(text)}.click()`)
  await delay(150)
}
async function input(selector, value) {
  await evaluate(`(() => { const input = document.querySelector(${JSON.stringify(selector)}); input.value = ${JSON.stringify(value)}; input.dispatchEvent(new Event('input', { bubbles: true })) })()`)
}

try {
  await mkdir(output, { recursive: true })
  await command('Page.enable')
  await command('Runtime.enable')
  await command('Fetch.enable', { patterns: [{ urlPattern: '*' }] })
  const identityScript = await command('Page.addScriptToEvaluateOnNewDocument', { source: "localStorage.setItem('auth_token','lab-test');localStorage.setItem('auth_role','admin')" })
  await command('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false })
  await command('Page.navigate', { url: `${origin}/settings` })
  await waitFor(`!!${button('保存优选限制')} && !${button('保存优选限制')}.disabled`)
  assert.equal(await evaluate("!!document.querySelector('a[href=\"/admin\"], a[href=\"/telegram\"]')"), false, 'slim must not restore removed navigation')
  await input('input[max="256"]', '0')
  await click('保存优选限制')
  await waitFor("document.body.textContent.includes('并发上限须为')")
  assert.equal(adminSaves, 0, 'invalid limit must not be submitted')
  await input('input[max="10000000"]', '250000')
  await input('input[max="1000"]', '40')
  await input('input[max="256"]', '64')
  await click('保存优选限制')
  await waitFor(`!${button('保存优选限制')}.disabled`)
  assert.equal(adminSaves, 1)
  assert.equal(appSettings.lab_daily_request_limit, 250000)
  assert.equal(appSettings.lab_requests_per_second, 40)
  assert.equal(appSettings.lab_max_workers, 64)
  await waitFor("!document.querySelector('.n-message')")
  for (const width of [1440, 375, 320]) {
    await command('Emulation.setDeviceMetricsOverride', { width, height: 900, deviceScaleFactor: 1, mobile: false })
    await evaluate("document.querySelector('input[max=\"256\"]').closest('.admin-card').scrollIntoView({block:'center'})")
    await delay(200)
    assert.equal(await evaluate('document.documentElement.scrollWidth <= innerWidth'), true, `admin limits overflow at ${width}`)
    const screenshot = await command('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false })
    await writeFile(join(output, `${width}-admin-limits.png`), Buffer.from(screenshot.data, 'base64'))
  }
  await command('Page.navigate', { url: `${origin}/lab/ip-selector` })
  await waitFor("!!document.querySelector('.lab-page') && document.querySelector('input[placeholder=\"cdn.example.com\"]')?.value === 'probe.example.com'")
  assert.equal(await evaluate("document.querySelector('input[max=\"64\"]')?.max"), '64')
  assert.ok(await evaluate("document.querySelector('.budget-summary').textContent.includes('250000')"))
  assert.ok(await evaluate("document.querySelector('.budget-summary').textContent.includes('40 请求/秒')"))
  assert.equal(await evaluate("document.querySelector('.lab-help').open"), false, 'long guidance should be collapsed')
  await waitFor("document.querySelector('.dns-automation-info')?.textContent.includes('未绑定')")
  assert.equal(await evaluate(`!!${button('一键填写并验证')}`), false)
  assert.equal(await evaluate(`${button('保存并执行')}.disabled`), true)
  const ownershipInput = '[aria-label="域名授权和请求预算"] input:not([readonly])'
  await input(ownershipInput, 'owner.example.org')
  await input('input[placeholder="cdn.example.com"]', '')
  const beforeChallenge = mutations.length
  await click('生成 TXT')
  await waitFor("!!document.querySelector('.verification-record')")
  assert.equal(mutations.length, beforeChallenge + 1, 'generating TXT must not save probe settings')
  assert.deepEqual(mutations.at(-1).body, { domain: 'owner.example.org' })
  assert.equal(settings.host, 'probe.example.com')
  assert.equal(await evaluate('document.querySelector(\'input[placeholder="cdn.example.com"]\').value'), '')
  await input('input[placeholder="cdn.example.com"]', 'probe.example.com')
  assert.equal(await evaluate(`${button('保存并执行')}.disabled`), true)
  await click('检查 TXT 记录')
  await waitFor("document.querySelector('.verification-record')?.textContent.includes('DNS TXT 验证值未找到')")
  assert.equal(await evaluate(`${button('保存并执行')}.disabled`), true)
  txtPresent = true
  await click('检查 TXT 记录')
  await waitFor(`!${button('保存并执行')}.disabled`)
  assert.equal(await evaluate("!!document.querySelector('.verification-record')"), false, 'verified TXT must collapse')
  assert.equal(await evaluate(`document.querySelector('.lab-page').textContent.includes(${JSON.stringify(state.verification.token)})`), false)
  await input('input[placeholder="cdn.example.com"]', 'other.example.com')
  assert.equal(await evaluate(`${button('保存并执行')}.disabled`), false, 'probe Host edits must not invalidate ownership')
  await input('input[placeholder="cdn.example.com"]', 'probe.example.com')
  const draftSelector = 'input[placeholder="/healthz"]'
  await input(draftSelector, '/unsaved-draft.txt')
  const beforePoll = statusRequests
  await evaluate('window.scrollTo(0, 300)')
  const scrollPosition = await evaluate('window.scrollY')
  const pollDeadline = Date.now() + 10000
  while (statusRequests === beforePoll && Date.now() < pollDeadline) await delay(100)
  assert.ok(statusRequests > beforePoll, 'status polling did not run')
  assert.equal(await evaluate(`document.querySelector(${JSON.stringify(draftSelector)}).value`), '/unsaved-draft.txt')
  assert.equal(await evaluate('window.scrollY'), scrollPosition)
  await evaluate("document.querySelectorAll('.lab-switch [role=switch]')[0].click()")
  await click('保存并执行')
  await waitFor(`${button('停止并关闭定时')} && !${button('停止并关闭定时')}.disabled`)
  assert.equal(settings.path, '/unsaved-draft.txt')
  assert.equal(settings.schedule, true)
  await click('停止并关闭定时')
  await waitFor(`!!${button('保存并执行')}`)
  assert.equal(await evaluate(`${button('保存并执行')}.disabled`), true, 'cooldown must block restarts')
  assert.equal(settings.schedule, false)
  await evaluate("document.querySelectorAll('.lab-switch [role=switch]')[1].click()")
  await waitFor("!!document.querySelector('input[type=password]')")
  await input('input[type=password]', 'mock-secret-never-persist')
  await click('保存配置')
  await waitFor("document.querySelector('input[type=password]')?.value === ''")
  assert.equal(mutations.at(-1).body.secret_key, 'mock-secret-never-persist')
  assert.equal(await evaluate("!JSON.stringify(localStorage).includes('mock-secret-never-persist') && !JSON.stringify(sessionStorage).includes('mock-secret-never-persist')"), true)
  state.next_allowed_at = 0
  state.budget_used = state.budget_limit
  await click('刷新状态')
  assert.equal(await evaluate(`${button('保存并执行')}.disabled`), true, 'exhausted budget must block runs')
  state.budget_used = 2
  statusUnavailable = true
  await click('刷新状态')
  await waitFor("document.querySelector('[role=alert]')?.textContent.includes('模拟网络故障')")
  statusUnavailable = false
  state.phase = 'completed'
  settings.ip_targets = '1.1.1.1\n43.175.131.30\n192.0.2.2'
  await evaluate(`(()=>{const input=document.querySelector('textarea[placeholder]');input.value=${JSON.stringify(settings.ip_targets)};input.dispatchEvent(new Event('input',{bubbles:true}))})()`)
  state.progress = { scanned: 3, matched: 1, total: 3, percent: 100 }
  runs = [{ id: 'test-run', started_at: 123, finished_at: 124, success: true, scanned: 3, matched: 1,
    selected_ips: ['1.1.1.1'], results: [{ ip: '1.1.1.1', status: 200, latency_ms: 12 }],
    rejected: 2, probe: { host: 'probe.example.com', sni: 'probe.example.com', path: '/ip-check.txt', statuses: '200' },
    diagnostics: [{ ip: '43.175.131.30', status: 403, latency_ms: 18 }, { ip: '192.0.2.2', error: 'TLS handshake timeout', latency_ms: 2000 }],
    segments: [{ target: '1.1.1.1', scanned: 1, matched: 1 }, { target: '43.175.131.30', scanned: 1, matched: 0 }, { target: '192.0.2.2', scanned: 1, matched: 0 }], dns_updated: false }]
  await click('刷新状态')
  await waitFor("document.querySelector('.lab-table')?.textContent.includes('1.1.1.1')")
  await waitFor("document.querySelector('[aria-label=\"未命中原因\"]')?.textContent.includes('HTTP 403')")
  assert.ok(await evaluate("document.querySelector('[aria-label=\"未命中原因\"]').textContent.includes('TLS handshake timeout')"))
  assert.ok(await evaluate("document.querySelector('[aria-label=\"未命中原因\"]').textContent.includes('probe.example.com/ip-check.txt')"))
  await waitFor("!document.querySelector('.n-message')")
  await command('Emulation.setDeviceMetricsOverride', { width: 375, height: 900, deviceScaleFactor: 1, mobile: true })
  await evaluate("document.querySelector('[aria-label=\"未命中原因\"]').scrollIntoView({block:'center'})")
  assert.ok(await evaluate("[...document.querySelectorAll('.diagnostic-list p')].every(row=>row.scrollWidth<=row.clientWidth && getComputedStyle(row).whiteSpace==='normal')"))
  await delay(250)
  const diagnosticsImage = await command('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false })
  await writeFile(join(output, '375-diagnostics.png'), Buffer.from(diagnosticsImage.data, 'base64'))
  await waitFor("!document.querySelector('.n-message')")
  for (const width of [1440, 375, 320]) {
    await command('Emulation.setDeviceMetricsOverride', { width, height: width === 1440 ? 1000 : 800, deviceScaleFactor: 1, mobile: false })
    for (const dark of [true, false]) {
      await evaluate(`(async () => { (await import('/src/stores/config.ts')).useConfigStore().darkMode = ${dark}; window.scrollTo(0, 0) })()`)
      await delay(200)
      assert.equal(await evaluate('document.documentElement.scrollWidth <= innerWidth'), true, `overflow at ${width}`)
      const screenshot = await command('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false })
      await writeFile(join(output, `${width}-${dark ? 'dark' : 'light'}.png`), Buffer.from(screenshot.data, 'base64'))
    }
  }
  await click('一键剔除并保存')
  await waitFor("document.querySelector('textarea[placeholder]')?.value === '1.1.1.1'")
  assert.equal(settings.ip_targets, '1.1.1.1')
  assert.equal(mutations.at(-1).body.secret_key, undefined, 'blank secret must retain the stored value')
  await click('生成 TXT')
  await waitFor(`${button('保存并执行')}.disabled`)
  assert.equal(settings.schedule, false)
  dnsBound = true
  await click('刷新状态')
  await waitFor(`!!${button('一键填写并验证')}`)
  dnsDenied = true
  await click('一键填写并验证')
  await waitFor("document.querySelector('[role=status]')?.textContent.includes('没有该域名的 DNS 写入权限')")
  assert.equal(await evaluate(`${button('保存并执行')}.disabled`), true)
  assert.equal(dnsRecords.size, 0)
  dnsDenied = false
  txtPresent = false
  const automaticToken = state.verification.token
  await input('input[placeholder="cdn.example.com"]', '')
  const beforeAutomatic = mutations.length
  await click('一键填写并验证')
  await waitFor("document.querySelector('[role=status]')?.textContent.includes('等待解析生效')")
  assert.equal(mutations.length, beforeAutomatic + 1, 'automatic TXT must not save probe settings')
  assert.deepEqual(mutations.at(-1).body, { domain: 'owner.example.org' })
  assert.equal(settings.host, 'probe.example.com')
  assert.equal(await evaluate('document.querySelector(\'input[placeholder="cdn.example.com"]\').value'), '')
  assert.equal(await evaluate(`${button('保存并执行')}.disabled`), true)
  assert.equal(dnsRecords.size, 1)
  assert.equal(state.verification.token, automaticToken)
  txtPresent = true
  await click('一键填写并验证')
  await waitFor("document.querySelector('[role=status]')?.textContent.includes('通过 DNS 验证')")
  assert.equal(dnsRecords.size, 1, 'retries must not duplicate TXT records')
  assert.equal(state.verification.token, automaticToken, 'retry must reuse the existing challenge')
  assert.equal(settings.schedule, false)
  assert.equal(state.running, false, 'verification must not automatically scan')
  assert.equal(await evaluate("!!document.querySelector('.verification-record')"), false, 'automatic verification must collapse TXT')
  await input('input[placeholder="cdn.example.com"]', 'probe.example.com')
  await waitFor("!document.querySelector('.n-message')")
  for (const width of [1440, 375, 320]) {
    await command('Emulation.setDeviceMetricsOverride', { width, height: 900, deviceScaleFactor: 1, mobile: false })
    await evaluate('window.scrollTo(0, 0)')
    await delay(200)
    assert.equal(await evaluate('document.documentElement.scrollWidth <= innerWidth'), true, `automatic DNS layout overflow at ${width}`)
    const screenshot = await command('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false })
    await writeFile(join(output, `${width}-automatic-dns.png`), Buffer.from(screenshot.data, 'base64'))
  }
  dnsBound = false
  await click('刷新状态')
  await waitFor(`!${button('一键填写并验证')}`)
  assert.equal(await evaluate(`!!${button('生成 TXT')}`), true, 'manual flow must remain after disconnect')
  enabled = false
  await click('刷新状态')
  await waitFor("location.pathname === '/dashboard' && !document.querySelector('.lab-page')")
  await command('Page.navigate', { url: `${origin}/about` })
  await waitFor("document.querySelector('.release-body')?.textContent.includes('Slim edition') && document.querySelector('.feature-grid')")
  assert.equal(await evaluate("document.querySelectorAll('.version-row strong')[1].textContent"), 'v2.7.0-slim')
  assert.ok(releaseListRequested, 'slim update checks must use the release list rather than full-edition latest')
  assert.equal(await evaluate("document.querySelector('.feature-grid').textContent.includes('多用户与管理后台')"), false)
  assert.ok(await evaluate("document.querySelector('.feature-grid').textContent.includes('AI 助手') && document.querySelector('.feature-grid').textContent.includes('IP 优选实验室')"))
  assert.equal(await evaluate("document.body.textContent.includes('v9.0.0')"), false, 'full-edition version must not appear as a slim update')
  enabled = true
  await command('Page.removeScriptToEvaluateOnNewDocument', { identifier: identityScript.identifier })
  await evaluate("localStorage.removeItem('auth_token'); localStorage.removeItem('auth_role')")
  const beforeDenied = labRequests
  await command('Page.navigate', { url: `${origin}/lab/ip-selector` })
  await waitFor("location.pathname === '/login' && !document.querySelector('.lab-page')")
  assert.equal(labRequests, beforeDenied, 'logged-out route must not query lab data')
  assert.equal(mutations.filter(item => item.path.endsWith('/run')).length, 1)
  assert.equal(mutations.filter(item => item.path.endsWith('/stop')).length, 1)
  assert.deepEqual(failures, [])
  console.log(`Lab browser checks passed; screenshots: ${output}`)
} finally {
  socket.close()
  await fetch(`${debuggerOrigin}/json/close/${target.id}`)
}
