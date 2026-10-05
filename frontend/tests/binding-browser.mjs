import assert from 'node:assert/strict'
import { mkdir, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const origin = process.env.BINDING_TEST_ORIGIN || 'http://127.0.0.1:18085'
const cdp = process.env.BINDING_TEST_CDP || 'http://127.0.0.1:19224'
const output = process.env.BINDING_TEST_OUTPUT || join(tmpdir(), 'binding-screenshots')
for (const address of [origin, cdp]) assert.ok(['127.0.0.1', 'localhost', '[::1]'].includes(new URL(address).hostname))
const target = await (await fetch(`${cdp}/json/new?about:blank`, { method: 'PUT' })).json()
const socket = new WebSocket(target.webSocketDebuggerUrl)
await new Promise(resolve => socket.addEventListener('open', resolve, { once: true }))
const pending = new Map(), mutations = [], failures = []
let sequence = 0, bindingRequest
const config = { tunnel_id: 'test-tunnel', tunnel_name: 'Test tunnel', service_url: 'http://localhost:3000', preferred_cname: '', cname_presets: [] }
function command(method, params = {}) {
  const id = ++sequence
  const promise = new Promise((resolve, reject) => pending.set(id, { resolve, reject }))
  socket.send(JSON.stringify({ id, method, params }))
  return promise
}
async function fulfill(requestId, responseCode, body) {
  await command('Fetch.fulfillRequest', { requestId, responseCode, responseHeaders: [{ name: 'Content-Type', value: 'application/json' }], body: Buffer.from(JSON.stringify(body)).toString('base64') })
}
socket.addEventListener('message', async event => {
  const message = JSON.parse(event.data)
  if (message.id) {
    const task = pending.get(message.id); pending.delete(message.id)
    if (message.error) task?.reject(new Error(JSON.stringify(message.error)))
    else task?.resolve(message.result)
    return
  }
  if (message.method === 'Runtime.exceptionThrown') failures.push(message.params.exceptionDetails)
  if (message.method !== 'Fetch.requestPaused') return
  const { requestId, request } = message.params
  const url = new URL(request.url)
  if (url.origin !== origin) {
    failures.push(`blocked external request: ${url.origin}`)
    await command('Fetch.failRequest', { requestId, errorReason: 'BlockedByClient' }); return
  }
  if (!url.pathname.startsWith('/api/')) { await command('Fetch.continueRequest', { requestId }); return }
  if (request.method === 'POST' && ['/api/domain/bind', '/api/domain/bind-batch'].includes(url.pathname)) {
    mutations.push({ path: url.pathname, body: JSON.parse(request.postData) })
    assert.equal(bindingRequest, undefined, 'only one mutation may be in flight')
    bindingRequest = requestId
    return
  }
  let body = {}, status = 200
  if (url.pathname === '/api/setup/status') status = 404
  else if (url.pathname === '/api/site') body = { name: 'Tunnel Manager', landing_enabled: false }
  else if (url.pathname === '/api/auth/me') body = { id: 'binding-user', username: 'Binding tester', role: 'admin', permissions: [] }
  else if (url.pathname === '/api/config') body = config
  else if (url.pathname === '/api/config/service') { config.service_url = JSON.parse(request.postData).value; body = { status: 'ok' } }
  else if (['/api/tunnels', '/api/monitors'].includes(url.pathname)) body = []
  else { status = 404; body = { error: 'mock endpoint not configured' } }
  await fulfill(requestId, status, body)
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
    await delay(70)
  }
  throw new Error(`Timed out: ${expression}`)
}
async function setInput(selector, value) {
  await evaluate(`(()=>{const input=document.querySelector(${JSON.stringify(selector)});input.value=${JSON.stringify(value)};input.dispatchEvent(new Event('input',{bubbles:true}))})()`)
}
async function finishBinding(status, body) {
  const request = bindingRequest
  bindingRequest = undefined
  await fulfill(request, status, body)
  await waitFor("!document.querySelector('.operation-status')")
}
async function screenshot(name) {
  await delay(250)
  const image = await command('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false })
  await writeFile(join(output, `${name}.png`), Buffer.from(image.data, 'base64'))
}
try {
  await mkdir(output, { recursive: true })
  await command('Runtime.enable'); await command('Page.enable')
  await command('Fetch.enable', { patterns: [{ urlPattern: '*' }] })
  await command('Page.addScriptToEvaluateOnNewDocument', { source: "localStorage.setItem('auth_token','binding-mock');localStorage.setItem('auth_role','admin')" })
  await command('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false })
  await command('Page.navigate', { url: `${origin}/domain/batch` })
  await waitFor("!!document.querySelector('#main-1') && document.querySelector('#service-1').value!==''")
  await setInput('#main-1', 'batch.example.com')
  await evaluate("document.querySelector('.form-action .btn-primary').click();document.querySelector('.form-action .btn-primary').click()")
  await waitFor("!!document.querySelector('.operation-status')")
  assert.equal(mutations.length, 1)
  assert.ok(await evaluate("document.querySelector('#main-1').disabled"))
  assert.equal(await evaluate("getComputedStyle(document.querySelector('.operation-spinner')).animationName"), 'spin')
  await delay(1100)
  assert.match(await evaluate("document.querySelector('.operation-status').textContent"), /已等待 [1-9]\d* 秒/)
  await screenshot('desktop-batch-running')
  await finishBinding(200, { results: [{ success: true, message: '模拟绑定成功', mode: 'simple' }] })
  await waitFor("document.querySelector('.group-result')?.textContent.includes('绑定成功')")
  await evaluate("window.__transitions=[];new MutationObserver(events=>events.forEach(event=>{const value=event.target.getAttribute?.('class')||'';if(value.includes('page-enter-active'))window.__transitions.push(value)})).observe(document.querySelector('.app-main'),{attributes:true,subtree:true,attributeFilter:['class']});document.querySelector('a[href=\"/domain\"]').click()")
  await waitFor("location.pathname==='/domain' && !!document.querySelector('.form-fields')")
  await delay(350)
  assert.ok(await evaluate('window.__transitions.length>0'), 'page navigation plays a transition')
  await command('Emulation.setDeviceMetricsOverride', { width: 375, height: 900, deviceScaleFactor: 1, mobile: true })
  await setInput('input[placeholder="例如: kukie.cn"]', 'single.example.com')
  await evaluate("document.querySelector('.form-action .btn-primary').click();document.querySelector('.form-action .btn-primary').click()")
  await waitFor("!!document.querySelector('.operation-status')")
  assert.equal(mutations.length, 2)
  assert.equal(await evaluate("document.querySelector('.binding-fields').disabled"), true)
  await evaluate("document.querySelector('.operation-status').scrollIntoView({block:'center'})")
  await screenshot('mobile-bind-running')
  assert.ok(await evaluate('document.documentElement.scrollWidth<=innerWidth'))
  await command('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-reduced-motion', value: 'reduce' }] })
  assert.ok(await evaluate("getComputedStyle(document.querySelector('.operation-spinner')).animationName==='none'"))
  await finishBinding(400, { error: '模拟域名校验失败' })
  assert.equal(await evaluate("document.querySelector('.binding-fields').disabled"), false)
  assert.equal(await evaluate("document.querySelector('input[placeholder=\"例如: kukie.cn\"]').value"), 'single.example.com')
  await waitFor("document.querySelector('.result-card')?.textContent.includes('模拟域名校验失败')")
  await screenshot('mobile-bind-failed')
  assert.deepEqual(failures, [])
  console.log(`Binding browser checks passed: delayed feedback, elapsed time, duplicate guard, preserved form, recovery, route transition, reduced motion and 375px layout. Screenshots: ${output}`)
} finally {
  if (bindingRequest) await command('Fetch.failRequest', { requestId: bindingRequest, errorReason: 'Aborted' }).catch(() => {})
  await command('Fetch.disable').catch(() => {})
  socket.close()
  await fetch(`${cdp}/json/close/${target.id}`).catch(() => {})
}
