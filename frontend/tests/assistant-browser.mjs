import assert from 'node:assert/strict'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const origin = process.env.ASSISTANT_TEST_ORIGIN || 'http://127.0.0.1:8084'
const debuggerOrigin = process.env.ASSISTANT_TEST_CDP || 'http://127.0.0.1:9223'
const output = process.env.ASSISTANT_TEST_OUTPUT || join(tmpdir(), 'assistant-screenshots')
const target = await (await fetch(`${debuggerOrigin}/json/new?about:blank`, { method: 'PUT' })).json()
const socket = new WebSocket(target.webSocketDebuggerUrl)
await new Promise(resolve => socket.addEventListener('open', resolve, { once: true }))
let sequence = 0
let role = 'admin'
let shared = false
let keySet = false
let history = []
const executed = []
const failures = []
const pending = new Map()
const connection = { endpoint: 'https://example.com/v1', model: 'test-model', key_set: true }
const settings = () => ({ shared_enabled: shared, is_admin: role === 'admin', configured: true, ...(!shared || role === 'admin' ? { personal: { ...connection, key_set: keySet } } : {}), ...(role === 'admin' ? { shared: connection } : {}) })

function command(method, params = {}) {
  const id = ++sequence
  socket.send(JSON.stringify({ id, method, params }))
  return new Promise((resolve, reject) => pending.set(id, { resolve, reject }))
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
  if (message.method !== 'Fetch.requestPaused') return
  const { requestId, request } = message.params
  const url = new URL(request.url)
  if (url.origin !== origin && !['data:', 'about:'].includes(url.protocol)) {
    failures.push(`Unexpected external request: ${url.origin}`)
    await command('Fetch.failRequest', { requestId, errorReason: 'BlockedByClient' })
    return
  }
  if (!url.pathname.startsWith('/api/')) { await command('Fetch.continueRequest', { requestId }); return }
  let status = 200
  let result = {}
  const body = request.postData ? JSON.parse(request.postData) : {}
  if (url.pathname === '/api/setup/status') status = 404
  else if (url.pathname === '/api/site') result = { name: 'Tunnel Manager', description: '配置工作台', icon: '', landing_enabled: false }
  else if (url.pathname === '/api/auth/me') result = { id: `test-${role}`, username: 'reviewer', role, permissions: ['tunnels', 'dns', 'monitors'] }
  else if (url.pathname === '/api/monitors') result = []
  else if (url.pathname === '/api/assistant/settings') {
    if (request.method === 'PUT') { keySet ||= !!body.api_key; if (body.shared_enabled !== undefined) shared = body.shared_enabled }
    result = settings()
  } else if (url.pathname === '/api/assistant/conversations') {
    if (request.method === 'POST') { result = { id: `conversation-${history.length}`, title: '新会话', updated_at: 1, messages: [], tasks: [] }; history.unshift(result) }
    else result = { conversations: history }
  } else if (url.pathname.endsWith('/messages')) {
    const conversation = history.find(item => url.pathname.includes(`/${item.id}/`))
    conversation.title = '准备两个独立任务'
    conversation.messages.push({ role: 'user', content: body.message }, { role: 'assistant', content: '请核对任务参数，尚未执行。\n<img src=x onerror=alert(1)> 应当只作为文本显示。' })
    conversation.tasks = [{ id: 'task-1', tool: 'create_tunnel', title: '新建隧道 · 长中文配置测试', status: 'pending', arguments: { name: '手机网页与桌面共享同一后端' } }, { id: 'task-2', tool: 'create_dns_record', title: '新增 DNS · service.example.com', status: 'pending', arguments: { zone_id: 'zone-id', name: `${'long-domain-'.repeat(8)}example.com`, type: 'CNAME', content: 'origin.example.com', ttl: 1, proxied: false } }]
    result = conversation
  } else if (url.pathname.includes('/tasks/')) {
    const task = history.flatMap(item => item.tasks).find(item => url.pathname.endsWith(`/${item.id}`))
    if (body.action === 'execute') { assert.equal(body.confirmed, true); executed.push(task.id); task.status = 'succeeded'; task.result = '执行成功，测试资源 ID：mock-resource' }
    else task.status = body.action === 'pause' ? 'paused' : body.action === 'resume' ? 'pending' : 'cancelled'
    result = task
  } else if (url.pathname.startsWith('/api/assistant/conversations/') && request.method === 'DELETE') history = history.filter(item => !url.pathname.endsWith(`/${item.id}`))
  await command('Fetch.fulfillRequest', { requestId, responseCode: status, responseHeaders: [{ name: 'Content-Type', value: 'application/json' }], body: Buffer.from(JSON.stringify(result)).toString('base64') })
})

async function evaluate(expression) {
  const result = await command('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true })
  if (result.exceptionDetails) throw new Error(JSON.stringify(result.exceptionDetails))
  return result.result.value
}
const delay = milliseconds => new Promise(resolve => setTimeout(resolve, milliseconds))
async function waitFor(expression) {
  for (let attempt = 0; attempt < 100; attempt++) { if (await evaluate(expression)) return; await delay(150) }
  throw new Error(`Timed out: ${expression}`)
}
async function clickText(text) { await evaluate(`Array.from(document.querySelectorAll('button')).find(button => button.textContent.trim() === ${JSON.stringify(text)})?.click()`); await delay(200) }
async function screenshot(name) {
  await delay(300)
  const result = await command('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false })
  await writeFile(`${output}/${name}.png`, Buffer.from(result.data, 'base64'))
}
async function assertInlineTaskVisible() {
  assert.equal(await evaluate(`(() => {
    const messages = document.querySelector('.messages').getBoundingClientRect()
    const checkbox = document.querySelector('.messages .task.pending input')?.getBoundingClientRect()
    return !!checkbox && checkbox.top >= messages.top && checkbox.bottom < messages.bottom
  })()`), true, 'a pending task checkbox must be visible without opening a drawer or manually scrolling')
  assert.equal(await evaluate("document.querySelectorAll('.n-drawer .task').length"), 0)
}

try {
  await mkdir(output, { recursive: true })
  await command('Page.enable')
  await command('Page.bringToFront')
  await command('Runtime.enable')
  await command('Fetch.enable', { patterns: [{ urlPattern: '*' }] })
  await command('Page.addScriptToEvaluateOnNewDocument', { source: "localStorage.setItem('auth_token','browser-test');localStorage.setItem('auth_role','admin');localStorage.setItem('auth_username','reviewer')" })
  await command('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false })
  await command('Page.navigate', { url: `${origin}/assistant` })
  await waitFor("!!document.querySelector('.assistant-page .send:not(:disabled)') || (!!document.querySelector('.assistant-page textarea') && !document.querySelector('.loading'))")
  await evaluate('document.fonts.ready.then(() => true)')
  for (const [width, height] of [[375, 710], [320, 640]]) {
    await command('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile: false })
    await evaluate("(async () => { (await import('/src/stores/config.ts')).useConfigStore().darkMode = false })()")
    await delay(200)
    const layout = await evaluate(`(() => {
      const messages = document.querySelector('.messages').getBoundingClientRect()
      const composer = document.querySelector('.composer').getBoundingClientRect()
      const navigation = document.querySelector('.tabbar').getBoundingClientRect()
      const suggestions = [...document.querySelectorAll('.suggestion')]
      return {
        navigationHeight: navigation.height,
        composerHeight: composer.height,
        composerBottom: composer.bottom,
        navigationTop: navigation.top,
        visibleSuggestions: suggestions.every(button => button.getBoundingClientRect().bottom <= messages.bottom),
        messageHeight: messages.height,
        rows: document.querySelector('textarea').rows,
        resourceConsent: document.querySelector('.resource-check input').checked,
        pageHeight: document.documentElement.scrollHeight,
        textareaHeight: document.querySelector('textarea').getBoundingClientRect().height,
        textareaScrollHeight: document.querySelector('textarea').scrollHeight
      }
    })()`)
    assert.ok(layout.navigationHeight <= 50, `oversized navigation at ${width}: ${layout.navigationHeight}`)
    await screenshot(`${width}-welcome-compact`)
    assert.ok(layout.composerHeight <= 120, `oversized idle composer at ${width}: ${JSON.stringify(layout)}`)
    assert.ok(layout.composerBottom <= layout.navigationTop, `navigation covers composer at ${width}`)
    assert.ok(layout.messageHeight > layout.composerHeight * 2, `insufficient conversation space at ${width}`)
    assert.equal(layout.visibleSuggestions, true, `clipped welcome suggestions at ${width}`)
    assert.equal(layout.rows, 1)
    assert.equal(layout.resourceConsent, false)
    assert.ok(layout.pageHeight <= height + 1, `page has unnecessary scrolling at ${width}`)
    assert.equal(await evaluate('document.documentElement.scrollWidth <= innerWidth'), true)
  }
  await command('Emulation.setDeviceMetricsOverride', { width: 375, height: 710, deviceScaleFactor: 1, mobile: false })
  await delay(200)
  await evaluate("document.querySelector('textarea').focus()")
  await waitFor("document.querySelector('textarea').matches(':focus')")
  assert.equal(await evaluate("getComputedStyle(document.querySelector('.tabbar')).display"), 'none', 'navigation should yield during typing')
  await evaluate("document.querySelector('textarea').value = '尚未发送的草稿'; document.querySelector('textarea').dispatchEvent(new Event('input',{bubbles:true}))")
  await delay(100)
  const sendPosition = await evaluate("(() => { const bounds = document.querySelector('.send').getBoundingClientRect(); return { x: bounds.x + bounds.width / 2, y: bounds.y + bounds.height / 2 } })()")
  await command('Input.dispatchMouseEvent', { type: 'mousePressed', ...sendPosition, button: 'left', clickCount: 1 })
  await delay(100)
  const pressedPosition = await evaluate("(() => { const bounds = document.querySelector('.send').getBoundingClientRect(); return bounds.y + bounds.height / 2 })()")
  assert.equal(pressedPosition, sendPosition.y, 'send button jumps when focus leaves the textarea')
  await command('Input.dispatchMouseEvent', { type: 'mouseReleased', x: 0, y: 0, button: 'left', clickCount: 1 })
  await evaluate("document.querySelector('textarea').focus()")
  assert.equal(await evaluate("document.querySelector('textarea').dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true }))"), true, 'mobile Enter must allow a newline')
  await command('Emulation.setDeviceMetricsOverride', { width: 375, height: 460, deviceScaleFactor: 1, mobile: false })
  await evaluate("document.querySelector('textarea').value = '第一行需求\\n第二行需求\\n第三行需求\\n' + 'long-resource-name-'.repeat(40); document.querySelector('textarea').dispatchEvent(new Event('input',{bubbles:true}))")
  await delay(200)
  assert.ok(await evaluate("document.querySelector('textarea').getBoundingClientRect().height <= 104"), 'long input must stop growing')
  assert.ok(await evaluate("document.querySelector('textarea').scrollHeight > document.querySelector('textarea').clientHeight"), 'long input must scroll internally')
  assert.ok(await evaluate("document.querySelector('.composer').getBoundingClientRect().bottom <= innerHeight"), 'composer exceeds keyboard-sized viewport')
  assert.ok(await evaluate("document.querySelector('.messages').getBoundingClientRect().height >= 100"), 'keyboard leaves no room for messages')
  await screenshot('375-keyboard-long-input')
  await evaluate("document.querySelector('textarea').value = ''; document.querySelector('textarea').dispatchEvent(new Event('input',{bubbles:true})); document.querySelector('textarea').blur()")
  await command('Emulation.setDeviceMetricsOverride', { width: 375, height: 710, deviceScaleFactor: 1, mobile: false })
  await delay(200)
  assert.equal(await evaluate("getComputedStyle(document.querySelector('.tabbar')).display"), 'flex', 'navigation must return after typing')
  assert.ok(await evaluate("document.querySelector('textarea').getBoundingClientRect().height <= 44"), 'cleared input must return to one line')
  await clickText('隐私说明')
  await waitFor("!!document.querySelector('.n-popover')")
  assert.equal(await evaluate("document.querySelector('.n-popover').textContent.includes('请勿填写密码或连接令牌')"), true, 'privacy details missing')
  await screenshot('375-privacy')
  await clickText('隐私说明')
  await evaluate("document.querySelector('.tab[href=\"/monitors\"]').click()")
  await waitFor("!document.querySelector('.assistant-page')")
  assert.ok(await evaluate("document.querySelector('.tabbar').getBoundingClientRect().height > 50"), 'other pages must retain the original navigation')
  assert.equal(await evaluate("getComputedStyle(document.querySelector('.app-main')).paddingBottom"), '84px', 'assistant layout leaked to another page')
  await evaluate("document.querySelector('.tab-fab').click()")
  await waitFor("!!document.querySelector('.action-row[href=\"/assistant\"]')")
  await evaluate("document.querySelector('.action-row[href=\"/assistant\"]').click()")
  await waitFor("!!document.querySelector('.assistant-page textarea') && !document.querySelector('.loading')")
  await command('Emulation.setDeviceMetricsOverride', { width: 375, height: 710, deviceScaleFactor: 2, mobile: true })
  await delay(200)
  assert.equal(await evaluate('document.documentElement.scrollWidth <= innerWidth'), true, 'overflow in mobile device emulation')
  assert.ok(await evaluate("document.querySelector('.composer').getBoundingClientRect().bottom <= document.querySelector('.tabbar').getBoundingClientRect().top"), 'mobile navigation overlaps input')
  await screenshot('375-device-welcome')
  await command('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false })
  await delay(200)
  assert.equal(await evaluate("document.querySelector('textarea').rows"), 3, 'desktop composer changed')
  assert.equal(await evaluate("getComputedStyle(document.querySelector('.privacy')).display !== 'none'"), true, 'desktop privacy note hidden')
  await evaluate("document.querySelector('textarea').value = '准备两个独立配置任务'; document.querySelector('textarea').dispatchEvent(new Event('input',{bubbles:true}))")
  await clickText('发送')
  await waitFor("document.querySelectorAll('.desktop-tasks .task').length === 2")
  assert.equal(executed.length, 0)
  assert.equal(await evaluate("document.querySelector('.message.assistant img') === null"), true)
  await evaluate("document.querySelector('.desktop-tasks .task-title input').click()")
  await clickText('核对并执行 1 项')
  assert.equal(executed.length, 0)
  await clickText('确认执行')
  await waitFor("!!document.querySelector('.desktop-tasks .task.succeeded')")
  assert.deepEqual(executed, ['task-1'])
  await clickText('暂停')
  await waitFor("!!document.querySelector('.desktop-tasks .task.paused')")
  await clickText('恢复待确认')
  await waitFor("!!document.querySelector('.desktop-tasks .task.pending')")

  for (const width of [1440, 1280, 900, 375, 320]) {
    await command('Emulation.setDeviceMetricsOverride', { width, height: width < 700 ? 820 : 1000, deviceScaleFactor: 1, mobile: false })
    for (const dark of [true, false]) {
      await evaluate(`(async () => { (await import('/src/stores/config.ts')).useConfigStore().darkMode = ${dark} })()`)
      await delay(200)
      assert.equal(await evaluate('document.documentElement.scrollWidth <= innerWidth'), true, `overflow at ${width}`)
      if (width >= 1280) assert.equal(await evaluate("document.querySelector('.assistant-header').getBoundingClientRect().left >= document.querySelector('.sidebar').getBoundingClientRect().right"), true, 'sidebar covers assistant header')
      await screenshot(`${width}-${dark ? 'dark' : 'light'}`)
    }
  }
  assert.equal(await evaluate("document.querySelectorAll('.messages .task').length"), 2, 'mobile confirmation cards must appear in the conversation without opening a drawer')
  await evaluate("document.querySelector('.mobile-tasks').click()")
  await waitFor("!!document.querySelector('.messages .task')")
  assert.equal(await evaluate('document.documentElement.scrollWidth <= innerWidth'), true)
  await screenshot('320-tasks')
  await evaluate("document.querySelector('.messages .task.pending .task-title input').click()")
  await evaluate("document.querySelector('.messages .confirm').click()")
  assert.deepEqual(executed, ['task-1'], 'selecting a mobile card must still require confirmation')
  await clickText('确认执行')
  await waitFor("document.querySelectorAll('.messages .task.succeeded').length === 2")
  assert.deepEqual(executed, ['task-1', 'task-2'])
  await evaluate("document.querySelector('[aria-label=\"AI 连接配置\"]').click()")
  await waitFor("!!document.querySelector('input[type=password]')")
  await evaluate("const password = document.querySelector('input[type=password]'); password.value = 'mock-key-never-persist'; password.dispatchEvent(new Event('input',{bubbles:true}))")
  await screenshot('320-settings')
  await clickText('保存配置')
  await waitFor("!document.querySelector('input[type=password]')")
  assert.equal(await evaluate("!JSON.stringify(localStorage).includes('mock-key-never-persist') && !JSON.stringify(sessionStorage).includes('mock-key-never-persist')"), true)
  await evaluate("document.querySelector('.tab-fab').click()")
  await waitFor("!!Array.from(document.querySelectorAll('.action-title')).find(item => item.textContent === 'AI 助手')")
  await screenshot('320-plus-menu')
  history = []
  await command('Emulation.setDeviceMetricsOverride', { width: 375, height: 710, deviceScaleFactor: 1, mobile: true })
  await command('Page.reload')
  await waitFor("!!document.querySelector('.welcome') && !document.querySelector('.loading')")
  await evaluate("document.querySelector('textarea').value = '生成两项待确认任务'; document.querySelector('textarea').dispatchEvent(new Event('input',{bubbles:true}))")
  await clickText('发送')
  await waitFor("document.querySelectorAll('.messages .task.pending').length === 2 && !document.querySelector('.thinking')")
  await assertInlineTaskVisible()
  assert.equal(await evaluate("document.querySelectorAll('.messages .task-title input:checked').length"), 0, 'new tasks must not be preselected')
  assert.equal(await evaluate("document.querySelector('.messages .confirm').disabled"), true)
  assert.equal(await evaluate("document.querySelector('.mobile-tasks').textContent"), '待确认 2')
  assert.deepEqual(executed, ['task-1', 'task-2'], 'receiving mobile proposals must not execute anything')
  await screenshot('375-new-task-cards')
  await command('Page.reload')
  await waitFor("document.querySelectorAll('.messages .task.pending').length === 2 && !document.querySelector('.loading')")
  await assertInlineTaskVisible()
  await screenshot('375-restored-task-cards')
  history[0].tasks.push({ id: 'task-monitor-defaults', tool: 'create_monitor', title: '新建私有监控项目 · 默认频率', status: 'pending', arguments: { name: '默认频率', interval_sec: 60 } })
  const srvData = { priority: 0, weight: 5, port: 5060, target: 'sip.example.com' }
  history[0].tasks.push({ id: 'task-srv', tool: 'create_dns_record', title: '新增 DNS · _sip._tcp.example.com', status: 'pending', arguments: { zone_id: 'zone-id', name: '_sip._tcp.example.com', type: 'SRV', data: srvData, ttl: 1, proxied: false } })
  for (const width of [1440, 375, 320]) {
    await command('Emulation.setDeviceMetricsOverride', { width, height: 900, deviceScaleFactor: 1, mobile: width < 700 })
    await command('Page.reload')
    const container = width < 700 ? '.messages' : '.desktop-tasks'
    await waitFor(`document.querySelectorAll('${container} .task.pending').length === 4 && !document.querySelector('.loading')`)
    const fields = await evaluate(`Array.from(document.querySelectorAll('${container} .task')).map(task => Object.fromEntries(Array.from(task.querySelectorAll('dt')).map(label => [label.textContent, label.nextElementSibling.textContent.trim()])))`)
    assert.equal(fields.find(item => item['名称'] === '默认频率')?.['间隔（秒）'], '60', 'normalized monitor interval must appear on the confirmation card')
    assert.equal(fields.find(item => item['类型'] === 'CNAME')?.['TTL（1 为自动）'], '1')
    assert.equal(fields.find(item => item['类型'] === 'CNAME')?.['代理'], '关闭')
    assert.deepEqual(JSON.parse(fields.find(item => item['类型'] === 'SRV')?.['记录数据（SRV / CAA）']), srvData)
    assert.equal(await evaluate(`document.querySelectorAll('${container} .task-title input:checked').length`), 0)
    assert.equal(await evaluate(`document.querySelector('${container} .confirm').disabled`), true)
    assert.equal(await evaluate('document.documentElement.scrollWidth <= innerWidth'), true)
    assert.deepEqual(executed, ['task-1', 'task-2'], 'default values and reloads must not auto-execute proposals')
    await evaluate(`Array.from(document.querySelectorAll('${container} .task')).find(task => task.textContent.includes('默认频率')).scrollIntoView({ block: 'center' })`)
    await screenshot(`${width}-default-parameters`)
    await evaluate(`Array.from(document.querySelectorAll('${container} .task')).find(task => task.textContent.includes('_sip._tcp.example.com')).scrollIntoView({ block: 'center' })`)
    await screenshot(`${width}-structured-dns`)
  }
  role = 'user'
  shared = true
  await command('Page.reload')
  await waitFor("document.querySelector('.identity p')?.textContent.includes('管理员共享连接')")
  await evaluate("document.querySelector('[aria-label=\"AI 连接配置\"]').click()")
  await waitFor("document.querySelector('.settings-body')?.textContent.includes('由管理员提供')")
  assert.equal(await evaluate("document.querySelectorAll('.settings-body input').length"), 0)
  await screenshot('320-shared-user')
  await evaluate("(async () => { const state = (await import('/src/stores/assistant.ts')).useAssistantStore(); state.drafts[state.currentID] = 'private draft'; (await import('/src/stores/config.ts')).useConfigStore().token = 'other-account' })()")
  await delay(100)
  assert.equal(await evaluate("(async () => { const state = (await import('/src/stores/assistant.ts')).useAssistantStore(); return state.conversations.length === 0 && Object.keys(state.drafts).length === 0 && state.settings === null })()"), true)
  if (process.env.ASSISTANT_TEST_HISTORY) {
    history = JSON.parse(await readFile(process.env.ASSISTANT_TEST_HISTORY, 'utf8'))
    role = 'admin'
    shared = false
    const count = history[0].tasks.length
    assert.ok(count > 0, 'replayed conversation must contain tasks')
    for (const width of [375, 320]) {
      await command('Emulation.setDeviceMetricsOverride', { width, height: 710, deviceScaleFactor: 1, mobile: true })
      await command('Page.reload')
      await waitFor(`document.querySelectorAll('.messages .task').length === ${count} && !document.querySelector('.loading')`)
      if (history[0].tasks.some(task => task.status === 'pending')) await assertInlineTaskVisible()
      assert.equal(await evaluate('document.documentElement.scrollWidth <= innerWidth'), true)
      assert.equal(await evaluate("document.querySelector('.messages .task').getBoundingClientRect().top < document.querySelector('.messages').getBoundingClientRect().bottom"), true, 'saved task card is not visible')
      await screenshot(`${width}-saved-task-replay`)
    }
    assert.deepEqual(executed, ['task-1', 'task-2'], 'saved task replay must be read-only')
  }
  assert.deepEqual(failures, [])
  console.log(`Assistant browser checks passed; screenshots: ${output}`)
} finally {
  socket.close()
  await fetch(`${debuggerOrigin}/json/close/${target.id}`)
}
