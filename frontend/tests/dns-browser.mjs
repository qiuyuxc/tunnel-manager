import assert from 'node:assert/strict'
import { mkdir, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const origin = process.env.DNS_TEST_ORIGIN || 'http://127.0.0.1:18085'
const cdp = process.env.DNS_TEST_CDP || 'http://127.0.0.1:19224'
const output = process.env.DNS_TEST_OUTPUT || join(tmpdir(), 'dns-screenshots')
for (const address of [origin, cdp]) assert.ok(['127.0.0.1', 'localhost', '[::1]'].includes(new URL(address).hostname))
const target = await (await fetch(`${cdp}/json/new?about:blank`, { method: 'PUT' })).json()
const socket = new WebSocket(target.webSocketDebuggerUrl)
await new Promise(resolve => socket.addEventListener('open', resolve, { once: true }))
const pending = new Map(), failures = [], mutations = []
let sequence = 0, failureMode = '', switchSession = false, leavePage = false
const types = ['A', 'AAAA', 'CNAME', 'TXT', 'MX', 'NS', 'SRV', 'CAA', 'PTR']
const content = { A: '192.0.2.10', AAAA: '2001:db8::10', CNAME: 'target.example.com', TXT: '  TXT intact  ', MX: 'mail.example.com', NS: 'ns.example.com', PTR: 'host.example.com', SRV: '0 5 5060 sip.example.com', CAA: '128 iodef "mailto:security@example.com"' }
const structured = { SRV: { priority: 0, weight: 5, port: 5060, target: 'sip.example.com', service: '_sip', proto: '_tcp', name: 'example.com' }, CAA: { flags: 128, tag: 'iodef', value: 'mailto:security@example.com' } }
let records = types.map(type => ({ id: `original-${type}`, type, name: type === 'SRV' ? '_sip._tcp.example.com' : `${type.toLowerCase()}.example.com`, content: content[type], ttl: 300, priority: 0, proxied: false, ...(structured[type] ? { data: structured[type] } : {}) }))
records.push({ id: 'unknown', type: 'HTTPS', name: 'unknown.example.com', content: '1 .', ttl: 300, data: { future: 'preserve' } })

function command(method, params = {}) {
  const id = ++sequence
  const promise = new Promise((resolve, reject) => pending.set(id, { resolve, reject }))
  socket.send(JSON.stringify({ id, method, params }))
  return promise
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
    failures.push(`Blocked external request: ${url.origin}`)
    await command('Fetch.failRequest', { requestId, errorReason: 'BlockedByClient' }); return
  }
  if (!url.pathname.startsWith('/api/')) { await command('Fetch.continueRequest', { requestId }); return }
  let result = {}, status = 200
  if (url.pathname === '/api/setup/status') status = 404
  else if (url.pathname === '/api/site') result = { name: 'Tunnel Manager', landing_enabled: false }
  else if (url.pathname === '/api/auth/me') result = { id: 'dns-test-user', username: 'DNS tester', role: 'admin', permissions: [] }
  else if (url.pathname === '/api/config') result = { site_name: 'Tunnel Manager', cname_presets: [] }
  else if (url.pathname === '/api/zones') result = [{ id: 'zone-test', name: 'example.com' }]
  else if (url.pathname.startsWith('/api/zones/zone-test/dns-records')) {
    if (request.method === 'GET') result = records
    else {
      const body = JSON.parse(request.postData || '{}')
      mutations.push({ method: request.method, path: url.pathname, body, token: request.headers['X-Auth-Token'] })
      if (switchSession && request.method === 'POST') {
        switchSession = false
        await evaluate("localStorage.setItem('auth_token', 'different-user-token')")
      }
      if (leavePage && request.method === 'POST') {
        leavePage = false
        await evaluate("document.querySelector('a[href=\"/tunnels\"]').click()")
        await waitFor("location.pathname==='/tunnels' && !document.querySelector('.dns-page')")
      }
      if (request.method === 'POST' && body.content === '192.0.2.2' && failureMode) {
        const mode = failureMode; failureMode = ''
        if (mode === 'network') { await command('Fetch.failRequest', { requestId, errorReason: 'ConnectionClosed' }); return }
        status = 400; result = { error: '模拟 Cloudflare 明确拒绝' }
      } else {
        const id = request.method === 'PUT' ? url.pathname.split('/').at(-1) : `created-${mutations.length}`
        result = { ...body, id, content: body.content || '', name: body.name?.includes('.') ? body.name : `${body.name}.example.com` }
        records = records.filter(record => record.id !== id).concat(result)
        status = request.method === 'POST' ? 201 : 200
      }
    }
  } else if (['/api/tunnels', '/api/monitors'].includes(url.pathname)) result = []
  else { status = 404; result = { error: 'mock endpoint not configured' } }
  await command('Fetch.fulfillRequest', { requestId, responseCode: status, responseHeaders: [{ name: 'Content-Type', value: 'application/json' }], body: Buffer.from(JSON.stringify(result)).toString('base64') })
})

const delay = milliseconds => new Promise(resolve => setTimeout(resolve, milliseconds))
async function evaluate(expression) {
  const result = await command('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true })
  if (result.exceptionDetails) throw new Error(JSON.stringify(result.exceptionDetails))
  return result.result.value
}
async function waitFor(expression) {
  for (let attempt = 0; attempt < 120; attempt++) { if (await evaluate(expression)) return; await delay(100) }
  throw new Error(`Timed out: ${expression}`)
}
const button = (text, scope = 'document') => `[...${scope}.querySelectorAll('button')].find(button=>button.textContent.trim()===${JSON.stringify(text)})`
async function clickButton(text, scope = 'document') { await waitFor(`${button(text, scope)} && !${button(text, scope)}.disabled`); await evaluate(`${button(text, scope)}.click()`); await delay(150) }
const modal = `document.querySelector('.record-modal')`
const bulk = `document.querySelector('.dns-create-modal')`
const field = (label, scope = modal) => `[...${scope}.querySelectorAll('label')].find(label=>label.querySelector('span')?.textContent===${JSON.stringify(label)})?.querySelector('input,textarea')`
async function setField(label, value, scope = modal) {
  const expression = field(label, scope)
  await waitFor(`!!${expression}`)
  await evaluate(`(()=>{const input=${expression};input.value=${JSON.stringify(String(value))};input.dispatchEvent(new Event('input',{bubbles:true}));input.dispatchEvent(new Event('change',{bubbles:true}));input.blur()})()`)
  await delay(70)
}
async function selectType(type, scope = modal) {
  await evaluate(`${scope}.querySelector('.n-base-selection').click()`)
  await waitFor(`[...document.querySelectorAll('.n-base-select-option')].some(option=>option.textContent.trim()===${JSON.stringify(type)})`)
  await evaluate(`[...document.querySelectorAll('.n-base-select-option')].find(option=>option.textContent.trim()===${JSON.stringify(type)}).click()`)
  await delay(100)
}
async function screenshot(name) {
  await delay(500)
  const result = await command('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false })
  await writeFile(join(output, `${name}.png`), Buffer.from(result.data, 'base64'))
}
async function openBulk() { await clickButton('批量新增'); await waitFor(`!!${bulk}`) }
async function closeBulk() { await clickButton('关闭', bulk); await waitFor(`!${bulk}`) }

try {
  await mkdir(output, { recursive: true })
  await command('Runtime.enable'); await command('Page.enable')
  await command('Fetch.enable', { patterns: [{ urlPattern: '*' }] })
  await command('Page.addScriptToEvaluateOnNewDocument', { source: "localStorage.setItem('auth_token','dns-mock-only');localStorage.setItem('auth_role','admin')" })
  await command('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false })
  await command('Page.navigate', { url: `${origin}/dns` })
  await waitFor("document.querySelectorAll('.dns-page tbody tr').length===10")
  assert.equal(await evaluate(`[...document.querySelectorAll('tbody tr')].find(row=>row.textContent.includes('HTTPS')).querySelectorAll('button').length`), 0)
  for (const type of types) {
    await evaluate(`[...document.querySelectorAll('tbody tr')].find(row=>row.querySelector('em')?.textContent===${JSON.stringify(type)}).querySelector('button[title="编辑"]').click()`)
    await waitFor(`!!${modal}`)
    assert.equal(await evaluate(`${modal}.querySelectorAll('[role=switch]').length`), ['A', 'AAAA', 'CNAME'].includes(type) ? 1 : 0, `${type} proxy UI`)
    if (type === 'SRV') {
      for (const [label, value] of [['SRV 优先级', '0'], ['权重', '5'], ['端口', '5060'], ['服务目标', 'sip.example.com']]) assert.equal(await evaluate(`${field(label)}.value`), value)
      await screenshot('desktop-srv')
    } else if (type === 'CAA') {
      for (const [label, value] of [['CAA flags', '128'], ['CAA tag', 'iodef'], ['CAA value', 'mailto:security@example.com']]) assert.equal(await evaluate(`${field(label)}.value`), value)
    } else assert.equal(await evaluate(`${field('解析值')}.value`), content[type])
    await setField('TTL（1 为自动）', 600)
    await clickButton('保存更改', modal); await waitFor(`!${modal}`)
    const update = mutations.at(-1).body
    assert.equal(update.ttl, 600)
    if (structured[type]) assert.deepEqual(update.data, structured[type])
    if (type === 'MX') assert.equal(update.priority, 0)
  }
  for (const type of types) {
    await clickButton('＋ 添加记录'); await waitFor(`!!${modal}`); await selectType(type)
    await setField('名称', type === 'SRV' ? '_sip._tcp.new.example.com' : `new-${type.toLowerCase()}.example.com`)
    if (type === 'SRV') { await setField('服务目标', 'sip.example.com'); await setField('端口', 5060); await setField('权重', 5) }
    else if (type === 'CAA') { await setField('CAA flags', 128); await setField('CAA tag', 'issuewild'); await setField('CAA value', 'ca.example.com') }
    else await setField('解析值', content[type])
    if (['A', 'AAAA', 'CNAME'].includes(type)) await evaluate(`${modal}.querySelector('[role=switch]').click()`)
    await clickButton('添加记录', modal); await waitFor(`!${modal}`)
    assert.equal(mutations.at(-1).body.type, type)
    assert.equal(mutations.at(-1).body.proxied, ['A', 'AAAA', 'CNAME'].includes(type))
  }
  await command('Emulation.setDeviceMetricsOverride', { width: 375, height: 900, deviceScaleFactor: 1, mobile: true })
  await openBulk(); await setField('名称', 'batch', bulk)
  await setField('解析值，每行一条', '192.0.2.1\n\nbad\n192.0.2.1', bulk)
  assert.ok(await evaluate(`${bulk}.textContent.includes('第 3 行') && ${bulk}.textContent.includes('IPv4')`))
  assert.equal(await evaluate(`${bulk}.querySelector('button.btn-primary').disabled`), true)
  await setField('解析值，每行一条', Array(101).fill('192.0.2.1').join('\n'), bulk)
  assert.ok(await evaluate(`${bulk}.textContent.includes('每批最多 100 条非空行')`))
  await setField('解析值，每行一条', '192.0.2.1\n192.0.2.2\n192.0.2.1\n192.0.2.3', bulk)
  await screenshot('mobile-batch-preview')
  assert.ok(await evaluate(`${bulk}.getBoundingClientRect().width <= 375`))
  assert.ok(await evaluate(`${bulk}.querySelector('button.btn-primary').getBoundingClientRect().bottom <= 900`), 'bulk submit remains visible on mobile')
  const before = mutations.length; failureMode = 'reject'
  await clickButton('新增 3 条记录', bulk)
  await waitFor(`${bulk}.textContent.includes('模拟 Cloudflare 明确拒绝') && ${field('解析值，每行一条', bulk)}.value==='192.0.2.2'`)
  assert.equal(mutations.length - before, 3)
  await screenshot('mobile-batch-partial-failure')
  await clickButton('重试剩余项', bulk)
  await waitFor(`${field('解析值，每行一条', bulk)}.value===''`)
  assert.deepEqual(mutations.slice(before).map(item => item.body.content), ['192.0.2.1', '192.0.2.2', '192.0.2.3', '192.0.2.2'])
  await closeBulk(); await openBulk(); await setField('名称', 'uncertain', bulk)
  await setField('解析值，每行一条', '192.0.2.1\n192.0.2.2\n192.0.2.3', bulk)
  failureMode = 'network'; const beforeUnknown = mutations.length
  await clickButton('新增 3 条记录', bulk)
  await waitFor(`${bulk}.textContent.includes('结果未知，请核对后再重试') && ${bulk}.textContent.includes('未发送')`)
  assert.equal(mutations.length - beforeUnknown, 2)
  assert.equal(await evaluate(`${bulk}.querySelector('button.btn-primary').disabled`), true)
  assert.equal(await evaluate(`${field('解析值，每行一条', bulk)}.value`), '192.0.2.2\n192.0.2.3')
  await evaluate(`${bulk}.querySelector('.n-card-content').scrollTop=99999`)
  await screenshot('mobile-unknown-gate')
  await setField('解析值，每行一条', '192.0.2.3', bulk)
  await evaluate(`${bulk}.querySelector('.n-checkbox').click()`)
  await clickButton('重试剩余项', bulk)
  await waitFor(`${field('解析值，每行一条', bulk)}.value===''`)
  assert.deepEqual(mutations.slice(beforeUnknown).map(item => item.body.content), ['192.0.2.1', '192.0.2.2', '192.0.2.3'])
  await closeBulk()
  await openBulk(); await setField('名称', 'session-switch', bulk)
  await setField('解析值，每行一条', '192.0.2.1\n192.0.2.2\n192.0.2.3', bulk)
  switchSession = true; const beforeSessionSwitch = mutations.length
  await clickButton('新增 3 条记录', bulk)
  await waitFor(`!${bulk}.querySelector('button.btn-primary').textContent.includes('正在')`)
  assert.equal(mutations.length - beforeSessionSwitch, 1, 'changing login sessions stops unsent rows instead of sending with the new token')
  assert.ok(await evaluate(`${bulk}.textContent.includes('未发送') && ${bulk}.textContent.includes('登录会话已变更')`))
  assert.equal(await evaluate(`${bulk}.querySelector('button.btn-primary').disabled`), true)
  await screenshot('mobile-session-switch')
  await closeBulk()
  await evaluate("localStorage.setItem('auth_token', 'dns-mock-only')")
  await evaluate(`[...document.querySelectorAll('tbody tr')].find(row=>row.querySelector('em')?.textContent==='CAA').querySelector('button[title="编辑"]').click()`)
  await waitFor(`!!${modal}`); await screenshot('mobile-caa')
  assert.ok(await evaluate(`(()=>{const rect=${modal}.getBoundingClientRect();return rect.width<=375&&rect.left>=0&&rect.right<=375})()`))
  await clickButton('取消', modal); await waitFor(`!${modal}`)
  await openBulk(); await setField('名称', 'leave-page', bulk)
  await setField('解析值，每行一条', '192.0.2.1\n192.0.2.2\n192.0.2.3', bulk)
  leavePage = true; const beforeLeaving = mutations.length
  await clickButton('新增 3 条记录', bulk)
  await waitFor("location.pathname==='/tunnels' && !document.querySelector('.dns-page')")
  await delay(500)
  assert.equal(mutations.length - beforeLeaving, 1, 'leaving the page stops remaining rows without aborting or mislabeling the in-flight request')
  assert.deepEqual(failures, [])
  console.log(`DNS mock regression passed: nine create/edit types, read-only protection, dedupe/line validation/limit, partial retry, unknown-result stop and acknowledgement, session switch/page teardown, 375px layouts. Screenshots: ${output}`)
} finally {
  await command('Fetch.disable').catch(() => {})
  socket.close()
  await fetch(`${cdp}/json/close/${target.id}`).catch(() => {})
}
