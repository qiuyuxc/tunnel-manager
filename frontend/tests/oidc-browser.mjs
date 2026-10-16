import assert from 'node:assert/strict'
import { readFile, writeFile } from 'node:fs/promises'
import { createServer } from 'node:net'

// Run only against the isolated panel + tea-house acceptance instances.
const root = process.env.OIDC_TEST_ROOT
assert.ok(root, 'Set OIDC_TEST_ROOT to the isolated acceptance directory')
const credentials = JSON.parse(await readFile(`${root}/credentials.json`, 'utf8'))
const verification = JSON.parse(await readFile(`${root}/verification.json`, 'utf8'))
const panel = process.env.OIDC_TEST_PANEL || 'http://127.0.0.1:8080'
const tea = process.env.OIDC_TEST_ISSUER || 'http://127.0.0.1:8085'
const cdp = process.env.OIDC_TEST_CDP || 'http://127.0.0.1:9223'
const adminOnly = process.env.OIDC_TEST_ADMIN_ONLY === '1'
const emailTest = process.env.OIDC_TEST_EMAIL === '1'
let receivedMail = ''
const smtp = emailTest ? createServer(socket => {
  let buffer = '', body = '', data = false
  socket.write('220 localhost OIDC test SMTP\r\n')
  socket.on('data', chunk => {
    buffer += chunk.toString()
    while (buffer.includes('\r\n')) {
      const index = buffer.indexOf('\r\n'), line = buffer.slice(0, index)
      buffer = buffer.slice(index + 2)
      if (data) {
        if (line === '.') { receivedMail = body; data = false; socket.write('250 accepted\r\n') }
        else body += line + '\r\n'
      } else if (/^EHLO|^HELO/i.test(line)) socket.write('250 localhost\r\n')
      else if (/^DATA/i.test(line)) { body = ''; data = true; socket.write('354 end with dot\r\n') }
      else if (/^QUIT/i.test(line)) { socket.end('221 goodbye\r\n') }
      else socket.write('250 OK\r\n')
    }
  })
}) : null
if (smtp) await new Promise(resolve => smtp.listen(2526, '127.0.0.1', resolve))
const delay = ms => new Promise(resolve => setTimeout(resolve, ms))
async function connect(url) {
  const ws = new WebSocket(url)
  await new Promise((resolve, reject) => { ws.onopen = resolve; ws.onerror = reject })
  let sequence = 0
  const pending = new Map()
  const events = []
  ws.onmessage = event => {
    const message = JSON.parse(event.data)
    if (!message.id) { events.push(message); return }
    const request = pending.get(message.id)
    if (!request) return
    pending.delete(message.id); clearTimeout(request.timer)
    if (message.error) request.reject(new Error(JSON.stringify(message.error)))
    else request.resolve(message.result)
  }
  return { events, close: () => ws.close(), send(method, params = {}) {
    const id = ++sequence
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => { pending.delete(id); reject(new Error(`Timeout: ${method}`)) }, 20000)
      pending.set(id, { resolve, reject, timer }); ws.send(JSON.stringify({ id, method, params }))
    })
  } }
}
const browser = await connect((await (await fetch(`${cdp}/json/version`)).json()).webSocketDebuggerUrl)
const context = (await browser.send('Target.createBrowserContext')).browserContextId
const target = (await browser.send('Target.createTarget', { url: 'about:blank', browserContextId: context })).targetId
const targets = await (await fetch(`${cdp}/json/list`)).json()
const page = await connect(targets.find(item => item.id === target).webSocketDebuggerUrl)
await page.send('Page.enable'); await page.send('Runtime.enable')
async function evaluate(expression) {
  const result = await page.send('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true })
  if (result.exceptionDetails) throw new Error(JSON.stringify(result.exceptionDetails))
  return result.result.value
}
async function until(expression) {
  for (let i = 0; i < 150; i++) { if (await evaluate(expression)) return; await delay(100) }
  throw new Error(`Not ready: ${expression}\n${await evaluate('document.body.innerText')}`)
}
async function navigate(url, ready) {
  await page.send('Page.navigate', { url }); await until(ready); await delay(150)
}
async function click(text, scope = 'document') {
  await evaluate(`Array.from(${scope}.querySelectorAll('button')).find(button => button.textContent.trim() === ${JSON.stringify(text)})?.click()`)
}
async function input(selector, value) {
  await evaluate(`(() => { const input = document.querySelector(${JSON.stringify(selector)}); if (!input) throw new Error('Missing input'); input.value = ${JSON.stringify(value)}; input.dispatchEvent(new Event('input', { bubbles: true })); })()`)
}
async function screenshot(name, width, selector = '') {
  await page.send('Emulation.setDeviceMetricsOverride', { width, height: width < 600 ? 900 : 1000, deviceScaleFactor: 1, mobile: width < 600 })
  if (selector) await evaluate(`document.querySelector(${JSON.stringify(selector)}).scrollIntoView({ block: 'start' })`)
  await delay(250)
  assert.equal(await evaluate('document.documentElement.scrollWidth > innerWidth'), false, `Horizontal overflow: ${name}`)
  const shot = await page.send('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false })
  await writeFile(`${root}/shots/${name}-${width}.png`, Buffer.from(shot.data, 'base64'))
}
async function authorize() {
  await until(`location.origin === ${JSON.stringify(tea)} && !!document.querySelector('.consent-actions')`)
  if (await evaluate("!!document.querySelector('.consent-reauth input[type=password]')")) await input('.consent-reauth input[type=password]', credentials.password)
  await click('允许并继续')
}
try {
  if (!adminOnly) {
  await navigate(`${tea}/login`, "document.readyState === 'complete'")
  const teaLogin = await evaluate(`(async () => {
    const site = await (await fetch('/api/v1/site')).json()
    const response = await fetch('/api/v1/session', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': site.csrf_token }, body: JSON.stringify({ name: ${JSON.stringify(credentials.username)}, password: ${JSON.stringify(credentials.password)} }) })
    return { status: response.status, body: await response.json() }
  })()`)
  assert.equal(teaLogin.status, 200)
  await navigate(`${panel}/login`, "!!document.querySelector('.oidc-login-options button')")
  await screenshot('login', 1440); await screenshot('login', 375)
  await click('使用 茶馆验证 登录')
  await authorize()
  if (emailTest) {
    await until(`location.origin === ${JSON.stringify(panel)} && !!document.querySelector('#oidc-code')`)
    await screenshot('email-verification', 375)
    await click('发送验证码')
    for (let i = 0; i < 100 && !receivedMail; i++) await delay(100)
    assert.ok(receivedMail.includes(credentials.email), 'Mail did not reach the claimed address')
    const plainPart = receivedMail.split('Content-Type: text/plain; charset=utf-8')[1]?.split(/\r\n--/)[0] || ''
    const decoded = Buffer.from(plainPart.replace(/=\r\n/g, '').replace(/=([\da-f]{2})/gi, (_, hex) => String.fromCharCode(parseInt(hex, 16))), 'latin1').toString('utf8')
    const code = decoded.match(/\b\d{6}\b/)?.[0]
    assert.ok(code, 'No verification code in received message')
    await input('#oidc-code', code); await click('验证并登录')
    console.log('Local SMTP delivery, code entry and required email verification passed')
  }
  await until(`location.origin === ${JSON.stringify(panel)} && location.pathname === '/dashboard'`)
  const username = await evaluate("localStorage.getItem('auth_username')")
  assert.match(username, /^oidc_/)
  console.log('Real tea-house authorization and first-login provisioning passed')
  await navigate(`${panel}/account`, "!!document.querySelector('.oidc-bindings')")
  await until("document.querySelector('.oidc-account').textContent.includes('尚未设置本站密码')")
  await screenshot('account-no-password', 1440, '.oidc-account'); await screenshot('account-no-password', 375, '.oidc-account')
  await click('解绑', "document.querySelector('.oidc-account')")
  await until("!!document.querySelector('.n-dialog')")
  await click('解绑', "document.querySelector('.n-dialog')")
  await until("document.querySelector('.oidc-account').textContent.includes('请先设置本站密码')")
  console.log('Last-login unlink protection passed')
  await click('重新认证并设置密码')
  await authorize()
  await until(`location.origin === ${JSON.stringify(panel)} && !!document.querySelector('.oidc-password')`)
  await screenshot('password-setup', 375)
  await input('.oidc-password input[type=password]', 'OIDC-local-2026!')
  await input('.oidc-password label:nth-of-type(2) input', 'OIDC-local-2026!')
  await evaluate("document.querySelector('.oidc-password').requestSubmit()")
  await until("document.querySelector('.oidc-account').textContent.includes('本站密码已设置')")
  assert.equal(await evaluate("!!document.querySelector('.oidc-password')"), false)
  console.log('Real OIDC reauthentication and local password setup passed')
  await click('解绑', "document.querySelector('.oidc-account')")
  await until("!!document.querySelector('.n-dialog')")
  await click('解绑', "document.querySelector('.n-dialog')")
  await until("!!Array.from(document.querySelectorAll('button')).find(b => b.textContent.trim() === '绑定 茶馆验证')")
  await input('.oidc-bind-password input', 'OIDC-local-2026!')
  await click('绑定 茶馆验证')
  await authorize()
  await until(`location.origin === ${JSON.stringify(panel)} && !!document.querySelector('.oidc-bindings') && document.querySelector('.oidc-account').textContent.includes('OIDC 身份已绑定')`)
  console.log('Unlink and authenticated account binding passed')
  await screenshot('account-bound', 375, '.oidc-account')
  await evaluate("fetch('/api/admin/logout', { method: 'POST', headers: { 'X-Auth-Token': localStorage.getItem('auth_token') } }).then(() => { localStorage.clear() })")
  await navigate(`${panel}/login`, "!!document.querySelector('.oidc-login-options button')")
  await click('使用 茶馆验证 登录'); await authorize()
  await until(`location.origin === ${JSON.stringify(panel)} && location.pathname === '/dashboard'`)
  assert.equal(await evaluate("localStorage.getItem('auth_username')"), username)
  console.log('Repeat login preserves the same local account')
  await evaluate("localStorage.clear()")
  await navigate(`${panel}/login`, "!!document.querySelector('.oidc-login-options button')")
  await click('使用 茶馆验证 登录')
  await until(`location.origin === ${JSON.stringify(tea)} && !!document.querySelector('.consent-actions')`)
  await click('取消')
  await until(`location.origin === ${JSON.stringify(panel)} && location.pathname === '/login' && !!document.querySelector('.login-error')`)
  assert.equal(await evaluate("localStorage.getItem('auth_token')"), null)
  console.log('Provider cancellation returns an error without a panel session')
  }
  if (adminOnly) await navigate(`${panel}/login`, "!!document.querySelector('.oidc-login-options button')")
  await evaluate(`localStorage.setItem('auth_token', ${JSON.stringify(verification.admin_token)}); localStorage.setItem('auth_username', ${JSON.stringify(credentials.panel_admin)}); localStorage.setItem('auth_role', 'admin')`)
  await navigate(`${panel}/admin`, "!!Array.from(document.querySelectorAll('button')).find(b => b.textContent.trim() === 'OIDC 认证')")
  await click('OIDC 认证'); await until("!!document.querySelector('.oidc-provider-list')")
  await screenshot('admin', 1440, '.oidc-admin'); await screenshot('admin', 375, '.oidc-admin')
  await click('编辑', "document.querySelector('.oidc-admin')")
  await until("!!document.querySelector('.oidc-provider-form')")
  assert.equal(await evaluate("document.querySelector('.oidc-provider-form input[type=password]').value"), '')
  await screenshot('admin-edit', 375, '.oidc-provider-form')
  await evaluate("localStorage.setItem('dark_mode', 'false')")
  await navigate(`${panel}/admin`, "!!document.querySelector('.admin-tabs')")
  await click('OIDC 认证'); await until("!!document.querySelector('.oidc-provider-list')")
  await screenshot('admin-light', 375, '.oidc-admin')
  await evaluate("localStorage.setItem('dark_mode', 'true')")
  const failures = page.events.filter(event => event.method === 'Runtime.exceptionThrown' || event.method === 'Page.javascriptDialogOpening')
  assert.deepEqual(failures, [])
  await writeFile(`${root}/${adminOnly ? 'admin' : 'browser'}-result.json`, JSON.stringify({ passed: true, screenshots: `${root}/shots`, runtimeExceptions: 0 }, null, 2))
  console.log('OIDC desktop/mobile UI and all browser flows passed')
} finally {
  page.close(); await browser.send('Target.disposeBrowserContext', { browserContextId: context }); browser.close()
  if (smtp) await new Promise(resolve => smtp.close(resolve))
}
