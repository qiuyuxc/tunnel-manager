const paths = {
  tunnel: '<path d="M4 20V10a8 8 0 0 1 16 0v10M8 20V10a4 4 0 0 1 8 0v10M3 20h18"/>',
  assistant: '<path d="M12 3v4m0 10v4M3 12h4m10 0h4M6 6l3 3m6 6 3 3M6 18l3-3m6-6 3-3"/><rect x="8" y="8" width="8" height="8" rx="3"/>',
  grid: '<rect x="3" y="3" width="7" height="7" rx="2"/><rect x="14" y="3" width="7" height="7" rx="2"/><rect x="3" y="14" width="7" height="7" rx="2"/><rect x="14" y="14" width="7" height="7" rx="2"/>',
  activity: '<path d="M3 12h4l3-8 4 16 3-8h4"/>',
  globe: '<circle cx="12" cy="12" r="9"/><ellipse cx="12" cy="12" rx="4" ry="9"/><path d="M3 12h18"/>',
  layers: '<path d="m12 3 10 5-10 5L2 8l10-5Zm-9 10 9 5 9-5M3 18l9 5 9-5"/>',
  settings: '<path d="M4 7h16M4 17h16"/><circle cx="9" cy="7" r="3"/><circle cx="15" cy="17" r="3"/>',
  shield: '<path d="m12 3 8 3v6c0 5-8 9-8 9s-8-4-8-9V6l8-3Z"/><path d="m8 12 3 3 5-6"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  chevron: '<path d="m9 5 7 7-7 7"/>',
  down: '<path d="m6 9 6 6 6-6"/>',
  left: '<path d="m15 5-7 7 7 7"/>',
  arrow: '<path d="M5 12h14m-6-6 6 6-6 6"/>',
  send: '<path d="M12 20V4m-6 6 6-6 6 6"/>',
  check: '<path d="m5 12 4 4L19 6"/>',
  checkCircle: '<circle cx="12" cy="12" r="9"/><path d="m8 12 3 3 5-6"/>',
  clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
  close: '<path d="m6 6 12 12M6 18 18 6"/>',
  laptop: '<rect x="4" y="3" width="16" height="13" rx="2"/><path d="M2 20h20M8 16v4m8-4v4"/>',
  phone: '<rect x="6" y="2" width="12" height="20" rx="3"/><path d="M10 5h4m-3 14h2"/>',
  app: '<rect x="4" y="3" width="16" height="18" rx="4"/><path d="M8 8h3v3H8zm5 0h3v3h-3zM9 17h6"/>',
  moon: '<path d="M20 15a9 9 0 0 1-11-11 9 9 0 1 0 11 11Z"/>',
  sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1 1m12 12 1 1M5 19l1-1M18 6l1-1"/>',
  reset: '<path d="M3 10a9 9 0 1 1 2 9M3 4v6h6"/>',
  menu: '<path d="M4 6h16M4 12h16M4 18h16"/>',
  chat: '<path d="M21 11a9 9 0 0 1-9 9H4l-2 2V11a9 9 0 0 1 19 0Z"/><path d="M7 9h9M7 13h6"/>',
  list: '<rect x="3" y="4" width="18" height="16" rx="3"/><path d="M7 8h1m3 0h6M7 12h1m3 0h6M7 16h1m3 0h6"/>',
  link: '<path d="m9 15 6-6M7 14l-2 2a4 4 0 0 0 6 6l3-3M10 5l3-3a4 4 0 0 1 6 6l-2 2" transform="translate(0 -1)"/>',
  file: '<path d="M14 3H5v18h14V8l-5-5Zm0 0v5h5M8 12h8M8 16h5"/>',
  info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v6m0-10v1"/>',
  pause: '<path d="M8 5v14M16 5v14"/>',
  play: '<path d="m8 4 12 8-12 8V4Z"/>',
  stop: '<rect x="6" y="6" width="12" height="12" rx="2"/>',
  more: '<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>',
  alert: '<path d="M12 3 2 21h20L12 3Zm0 6v5m0 3v1"/>',
  server: '<rect x="3" y="3" width="18" height="7" rx="2"/><rect x="3" y="14" width="18" height="7" rx="2"/><path d="M7 6h1m-1 11h1m5-11h4m-4 11h4"/>',
  bolt: '<path d="m13 2-9 12h7l-1 8 10-13h-7l1-7Z"/>',
  search: '<circle cx="10" cy="10" r="6"/><path d="m15 15 6 6"/>',
  wifi: '<path d="M3 8a14 14 0 0 1 18 0M6 12a9 9 0 0 1 12 0m-9 4a4 4 0 0 1 6 0m-3 3h.01"/>',
  battery: '<rect x="2" y="7" width="17" height="10" rx="2"/><path d="M22 10v4M5 10h11v4H5z"/>',
}

const icon = (name, className = '') => `<svg class="icon ${className}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths[name] || paths.assistant}</svg>`
const escape = value => String(value).replace(/[&<>"']/g, character => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[character]))
const variants = { A: '协作工作台', B: '任务指挥台', C: '分步配置' }
const labels = { waiting: '待确认', running: '执行中', paused: '已暂停', done: '已完成', failed: '需处理', canceled: '已取消' }
const params = new URLSearchParams(location.search)
const readParam = (name, values, fallback) => values.includes(params.get(name)) ? params.get(name) : fallback
const state = {
  variant: readParam('variant', ['A', 'B', 'C'], 'A'),
  device: readParam('device', ['desktop', 'mobile', 'app'], innerWidth <= 640 ? 'mobile' : 'desktop'),
  theme: readParam('theme', ['dark', 'light'], 'dark'),
  scenario: readParam('state', ['ready', 'empty', 'running', 'failed'], 'ready'),
  draft: '', prompt: '', tasks: [], selected: new Set(), step: 1, empty: false,
  thinking: false, dialog: null, responseMode: '均衡', sequence: 0,
  aiUser: 'admin', aiShared: false, aiDraft: null,
  aiProfiles: {
    admin: { name: 'qiuyuxc', admin: true, endpoint: '', hasKey: false },
    alice: { name: '用户 A', admin: false, endpoint: '', hasKey: false },
    bob: { name: '用户 B', admin: false, endpoint: '', hasKey: false },
  },
  aiSharedConfig: { endpoint: '', hasKey: false },
}
let toastTimer
let replyTimer

const presets = {
  connect: {
    prompt: '把 NAS 和相册接入我的域名，并给它们加上可用性监控。',
    reply: '可以。我会复用家庭隧道，将两个服务分别绑定域名，再添加可用性监控。先看一下配置计划。',
    tasks: [
      { title: '接入 NAS', resource: 'nas.home.example', icon: 'server', kind: '域名绑定', detail: '保留现有隧道，只新增一条访问规则。', changes: [['访问域名', '未配置', 'nas.home.example'], ['源站服务', '未配置', 'http://192.168.1.8:5000'], ['DNS 记录', '未配置', 'CNAME · 家庭隧道']] },
      { title: '接入家庭相册', resource: 'photos.home.example', icon: 'globe', kind: '域名绑定', detail: '使用独立域名访问相册，与 NAS 配置互不影响。', changes: [['访问域名', '未配置', 'photos.home.example'], ['源站服务', '未配置', 'http://192.168.1.8:2283'], ['TLS', '默认', '自动 HTTPS']] },
      { title: '添加可用性监控', resource: 'NAS + 家庭相册', icon: 'activity', kind: '服务监控', detail: '为两个服务添加 HTTP 检测，先不启用消息告警。', changes: [['检测目标', '0 个', '2 个'], ['检测间隔', '未配置', '每 60 秒'], ['通知方式', '未配置', '仅面板展示']] },
    ],
  },
  inspect: {
    prompt: '检查家庭相册为什么无法访问，先不要修改任何配置。',
    reply: '先做只读检查。我会分别检查隧道、DNS 与回源状态，汇总结果后再决定是否需要修改。',
    tasks: [
      { title: '检查隧道连接', resource: '家庭隧道', icon: 'tunnel', kind: '只读检查', readonly: true, detail: '读取连接器状态，不重启或修改隧道。', changes: [['检查范围', '未检测', '连接器与隧道状态']] },
      { title: '核对 DNS 指向', resource: 'photos.home.example', icon: 'globe', kind: '只读检查', readonly: true, detail: '核对域名与 CNAME 目标，不写入 DNS 记录。', changes: [['检查范围', '未检测', 'CNAME 与代理状态']] },
      { title: '验证源站响应', resource: '192.168.1.8:2283', icon: 'activity', kind: '只读检查', readonly: true, detail: '模拟检查回源连通性与 HTTP 响应。', changes: [['检查范围', '未检测', 'HTTP 状态与响应耗时']] },
    ],
  },
  dns: {
    prompt: '为博客和文档站批量绑定域名，并核对生成的 DNS 记录。',
    reply: '已拆成两个独立绑定任务和一个记录检查任务。每个任务的执行结果都会单独保留。',
    tasks: [
      { title: '绑定博客域名', resource: 'blog.home.example', icon: 'globe', kind: '域名绑定', detail: '把博客服务接入现有家庭隧道。', changes: [['访问域名', '未配置', 'blog.home.example'], ['源站', '未配置', 'http://192.168.1.8:3000']] },
      { title: '绑定文档域名', resource: 'docs.home.example', icon: 'file', kind: '域名绑定', detail: '新增文档站的独立路由。', changes: [['访问域名', '未配置', 'docs.home.example'], ['源站', '未配置', 'http://192.168.1.8:8080']] },
      { title: '核对 DNS 记录', resource: 'home.example', icon: 'search', kind: '只读检查', readonly: true, detail: '模拟查询两个域名的记录配置。', changes: [['检查范围', '未检测', '2 条 CNAME 记录']] },
    ],
  },
}

function createPlan(key = 'connect', prompt) {
  const preset = presets[key]
  state.preset = key
  state.prompt = prompt || preset.prompt
  state.reply = preset.reply
  state.empty = false
  state.tasks = preset.tasks.map(task => ({ ...task, id: `task-${++state.sequence}`, status: 'waiting', progress: 0 }))
  state.selected = new Set(state.tasks.map(task => task.id))
}

function applyScenario(scenario) {
  clearTimeout(replyTimer)
  state.thinking = false
  state.scenario = scenario
  state.draft = ''
  state.step = 1
  createPlan()
  if (scenario === 'empty') { state.empty = true; state.tasks = [] }
  if (scenario === 'running') state.tasks.forEach((task, index) => { task.status = 'running'; task.progress = 16 + index * 13 })
  if (scenario === 'failed') state.tasks.forEach((task, index) => { task.status = index === 2 ? 'failed' : 'done'; task.progress = index === 2 ? 48 : 100 })
}

function counts() {
  return state.tasks.reduce((result, task) => { result[task.status]++; return result }, { waiting: 0, running: 0, done: 0, failed: 0, paused: 0, canceled: 0 })
}

function button(text, action, symbol, className = '', extra = '') {
  return `<button ${extra.includes('type=') ? '' : 'type="button"'} class="button ${className}" data-action="${action}" ${extra}>${symbol ? icon(symbol) : ''}<span>${text}</span></button>`
}

function iconButton(symbol, action, label, extra = '') {
  return `<button type="button" class="icon-button" data-action="${action}" aria-label="${label}" title="${label}" ${extra}>${icon(symbol)}</button>`
}

function badge(status) {
  return `<span class="status status-${status}"><i></i>${labels[status]}</span>`
}

function reviewToolbar() {
  return `<header class="review-toolbar"><div class="review-brand"><span class="review-mark">TM</span><span><strong>AI 助手</strong><small>交互设计稿 <span class="review-version">/ 01</span></small></span><span class="prototype-tag">仅模拟</span></div><div class="review-controls"><div class="device-control" role="group" aria-label="预览设备">${[['desktop', 'laptop', '桌面'], ['mobile', 'phone', '网页移动端'], ['app', 'app', 'APP']].map(([device, symbol, label]) => `<button data-device="${device}" class="${state.device === device ? 'selected' : ''}" aria-pressed="${state.device === device}" title="${label}">${icon(symbol)}<span>${label}</span></button>`).join('')}</div><select class="scenario-select" aria-label="预览状态" data-control="scenario">${[['ready', '默认场景'], ['empty', '空会话'], ['running', '执行中'], ['failed', '失败与重试']].map(([value, title]) => `<option value="${value}" ${value === state.scenario ? 'selected' : ''}>${title}</option>`).join('')}</select>${iconButton(state.theme === 'dark' ? 'sun' : 'moon', 'theme', '切换明暗主题')}${iconButton('info', 'notes', '设计说明')}${iconButton('reset', 'reset', '重置示例')}</div></header>`
}

function sidebar() {
  const navigation = [['grid', '概览'], ['tunnel', '隧道管理'], ['globe', '域名绑定'], ['layers', 'DNS 管理'], ['activity', '服务监控']]
  return `<aside class="sidebar"><div class="brand"><span class="brand-symbol">${icon('tunnel')}</span><div>Tunnel Manager<small>个人工作区</small></div></div><div class="nav-heading">工作空间</div><nav aria-label="主要导航">${navigation.map(([symbol, text]) => button(text, 'placeholder', symbol, 'nav-item')).join('')}${button('AI 助手', 'workspace', 'assistant', 'nav-item active', 'aria-current="page"')}<span class="nav-section-line"></span>${button('全局设置', 'placeholder', 'settings', 'nav-item')}</nav><div class="sidebar-conversations"><div class="nav-heading">最近会话 ${iconButton('plus', 'new', '新建示例会话')}</div><button class="history-item current" data-preset="connect">${icon('chat')}<span>家庭网络接入方案</span></button><button class="history-item" data-preset="inspect">${icon('chat')}<span>相册访问异常排查</span></button></div><div class="sidebar-bottom"><div class="demo-connection">${icon('shield')}<span>示例工作区<small>未连接真实服务</small></span><span class="connection-dot"></span></div><button class="profile" data-action="placeholder"><span class="avatar">Q</span><span>${escape(state.aiProfiles[state.aiUser].name)}<small>${state.aiProfiles[state.aiUser].admin ? '管理员' : '普通用户'} · 示例</small></span>${icon('more')}</button></div></aside>`
}

function appHeader() {
  const count = counts()
  return `<header class="application-header">${state.device === 'app' ? `<div class="system-status"><span>9:41</span><div>${icon('wifi')}${icon('battery')}</div></div>` : ''}<div class="page-toolbar"><div class="page-identity">${iconButton('menu', 'history', '查看示例会话')}<span class="breadcrumb-desktop">工作空间<span>/</span></span><h1>AI 助手</h1><span class="beta-label">设计预览</span></div><div class="page-actions">${button(`演示助手 · ${state.responseMode}`, 'model', 'assistant', 'model-button')}${iconButton('settings', 'ai-settings', 'AI 连接配置')}${iconButton('plus', 'new', '新建会话')}<button class="task-toggle" data-action="queue" aria-label="打开任务队列">${icon('list')}<span class="task-toggle-label">任务</span><b>${count.running || state.tasks.length}</b></button></div></div></header>`
}

function bottomNav() {
  return `<nav class="bottom-nav" aria-label="移动端导航">${[['grid', '概览', 'placeholder'], ['activity', '监控', 'placeholder'], ['plus', '快捷操作', 'create'], ['layers', 'DNS', 'placeholder'], ['more', '更多', 'history']].map(([symbol, label, action]) => action === 'create' ? `<button type="button" class="bottom-create" data-action="create" aria-label="打开快捷操作" aria-haspopup="dialog" aria-controls="dialog">${icon(symbol)}</button>` : `<button type="button" data-action="${action}">${icon(symbol)}<span>${label}</span></button>`).join('')}</nav>${state.device === 'app' ? '<div class="gesture-zone"><span></span></div>' : ''}`
}

function contextPills() {
  return `<div class="context-pills"><button type="button" data-action="context">${icon('tunnel')}家庭隧道${icon('down')}</button><button type="button" data-action="context">${icon('globe')}home.example</button><span class="context-caption">示例上下文</span></div>`
}

function suggestions() {
  return `<div class="suggestions">${[['connect', 'plus', '接入服务'], ['dns', 'layers', '批量绑定'], ['inspect', 'search', '排查问题']].map(([key, symbol, title]) => `<button data-preset="${key}">${icon(symbol)}${title}${icon('arrow')}</button>`).join('')}</div>`
}

function composer(compact = false) {
  return `<form class="composer-wrap ${compact ? 'compact-composer' : ''}" data-form="prompt"><div class="composer"><label class="sr-only" for="prompt-input">描述配置需求</label><textarea id="prompt-input" name="prompt" rows="2" maxlength="800" placeholder="描述你想完成的配置，或输入 / 选择操作" ${state.thinking ? 'disabled' : ''}>${escape(state.draft)}</textarea><div class="composer-bottom"><div class="composer-tools">${iconButton('plus', 'context', '选择资源上下文')}<button type="button" class="composer-context" data-action="context">${icon('tunnel')}家庭隧道${icon('down')}</button><span class="composer-shortcut">Shift + Enter 换行</span></div><button type="submit" class="send-button" aria-label="发送需求" ${state.thinking ? 'disabled' : ''}>${state.thinking ? icon('clock') : icon('send')}</button></div></div><p class="composer-note">${icon('shield')}配置变更需确认 · 当前为离线模拟，不会修改真实资源</p></form>`
}

function welcome() {
  return `<div class="welcome"><span class="assistant-emblem">${icon('assistant')}</span><span class="eyebrow">你的网络配置搭档</span><h2>这次，想完成什么？</h2><p>说出目标，我来拆分任务。<br>从一个域名，到一组服务，先核对再执行。</p>${suggestions()}<div class="welcome-example"><span>试着这样说</span><button data-preset="connect">“把 NAS 和相册接入域名，并加上监控。”${icon('arrow')}</button></div></div>`
}

function taskRow(task, detailed = false) {
  return `<article class="task-row ${detailed ? 'task-row-detailed' : ''}" data-task-row="${task.id}"><button class="task-open" data-task="${task.id}"><span class="task-symbol ${task.status === 'done' ? 'complete' : ''}">${icon(task.status === 'done' ? 'check' : task.icon)}</span><span class="task-text"><strong>${escape(task.title)}</strong><small>${escape(task.resource)}</small></span>${badge(task.status)}</button>${task.status === 'running' || task.status === 'paused' ? `<div class="task-progress"><progress max="100" value="${task.progress}" aria-label="${escape(task.title)}进度"></progress><span>${task.progress}%</span></div>` : ''}${task.status === 'failed' ? `<div class="task-error">${icon('alert')}示例源站响应超时<button data-action="retry" data-id="${task.id}">重试</button></div>` : ''}${detailed ? `<div class="task-row-footer"><span>${escape(task.kind)} · ${task.readonly ? '只读' : '需确认写入'}</span>${button('详情', 'task-detail', 'chevron', 'text-button', `data-id="${task.id}"`)}</div>` : ''}</article>`
}

function planCard() {
  const count = counts()
  return `<section class="plan-card"><div class="plan-title"><div><span class="eyebrow">配置计划</span><h3>家庭服务接入${state.preset === 'inspect' ? '检查' : '方案'}</h3></div><span class="plan-number">${String(state.tasks.length).padStart(2, '0')}<small>个任务</small></span></div><ol class="plan-lines">${state.tasks.map((task, index) => `<li><span class="step-index ${task.status === 'done' ? 'is-done' : ''}">${task.status === 'done' ? icon('check') : String(index + 1).padStart(2, '0')}</span><div><strong>${escape(task.title)}</strong><small>${escape(task.resource)}</small></div><span class="plan-kind">${escape(task.kind)}</span></li>`).join('')}</ol><div class="plan-bottom"><span>${icon('shield')}${count.waiting ? `${state.tasks.filter(task => !task.readonly).length} 项配置写入，确认后执行` : count.running ? '任务正在并行模拟执行' : count.failed ? '其余任务不受失败任务影响' : '各任务结果已独立记录'}</span>${count.waiting ? button(`核对 ${count.waiting} 项变更`, 'review', 'arrow', 'primary') : button('查看执行结果', 'queue', 'list', 'primary')}</div></section>`
}

function conversation() {
  if (state.empty) return welcome()
  return `<div class="conversation"><div class="conversation-date">今天 · 示例会话</div><div class="user-message"><span class="avatar">Q</span><div><span class="message-author">你</span><p>${escape(state.prompt)}</p></div></div><div class="assistant-message"><span class="assistant-avatar">${icon('assistant')}</span><div class="assistant-body"><div class="message-author">Tunnel 助手 <span>${state.thinking ? '正在整理需求' : '演示响应'}</span></div>${state.thinking ? '<div class="thinking"><i></i><i></i><i></i><span>分析目标与资源上下文</span></div>' : `<p class="assistant-copy">${escape(state.reply)}</p>${planCard()}<div class="assistant-footnote">${icon('info')}示例源站地址已预填，可在核对时查看全部变更。</div>`}</div></div></div>`
}

function resourceSection() {
  return `<section class="resource-section"><div class="section-heading"><h3>资源上下文</h3><button data-action="context" class="quiet-link">查看</button></div><div class="resource-row">${icon('tunnel')}<span>家庭隧道<small>tunnel-home-01</small></span><span class="small-dot"></span></div><div class="resource-row">${icon('globe')}<span>home.example<small>示例 DNS 区域</small></span></div><div class="scope-note">${icon('shield')}仅在所选资源范围内制定计划</div></section>`
}

function taskInspector() {
  const count = counts()
  return `<aside class="task-inspector"><div class="inspector-heading"><div>${icon('list')}<h2>任务队列</h2><span class="counter">${state.tasks.length}</span></div>${iconButton('more', 'queue', '查看全部任务')}</div><div class="queue-overview"><div><strong>${String(count.waiting).padStart(2, '0')}</strong><span>待确认</span></div><div><strong class="accent-text">${String(count.running + count.paused).padStart(2, '0')}</strong><span>执行中</span></div><div><strong>${String(count.done).padStart(2, '0')}</strong><span>已完成</span></div></div><div class="queue-items scroll-region" id="inspector-scroll">${state.tasks.length ? state.tasks.map(task => taskRow(task)).join('') : '<div class="empty-queue">任务会出现在这里<br><small>从描述一个目标开始</small></div>'}</div><div class="parallel-note">${icon('layers')}任务独立执行，进度分别可见</div>${resourceSection()}<div class="inspector-safety">${icon('shield')}你确认的，才会被执行。</div></aside>`
}

function VariantA() {
  return `<div class="variant-a"><section class="chat-workspace"><div class="chat-subheader"><div><span class="session-dot"></span><h2>${state.empty ? '新会话' : '家庭网络接入方案'}</h2></div>${button('会话', 'history', 'chat', 'text-button')}</div><div class="chat-scroll scroll-region" id="chat-scroll">${contextPills()}${conversation()}</div><div class="chat-footer">${!state.empty ? suggestions() : ''}${composer()}</div></section>${taskInspector()}</div>`
}

function VariantB() {
  const count = counts()
  const groups = [['waiting', '待确认', ['waiting']], ['running', '进行中', ['running', 'paused', 'failed']], ['done', '已结束', ['done', 'canceled']]]
  return `<div class="variant-b"><div class="board-toolbar"><div><span class="eyebrow">多任务工作台</span><h2>每件事，都有进度。</h2><p>将一个目标拆成任务，独立执行、分别确认。</p></div><div class="board-actions">${count.running ? button('暂停全部', 'pause-all', 'pause') : button('新建任务', 'focus', 'plus')}${count.waiting ? button('核对并执行', 'review', 'play', 'primary') : ''}</div></div><div class="board-summary">${contextPills()}<span>${icon('layers')}支持并行模拟 · ${count.done}/${state.tasks.length} 已完成</span></div><div class="kanban scroll-region" id="board-scroll">${groups.map(([key, title, statuses]) => { const tasks = state.tasks.filter(task => statuses.includes(task.status)); return `<section class="kanban-column"><div class="column-heading"><span class="column-dot ${key}"></span><h3>${title}</h3><b>${tasks.length}</b>${iconButton('more', 'queue', `查看${title}任务`)}</div><div class="kanban-items">${tasks.map(task => `<div class="kanban-card">${taskRow(task, true)}${task.status === 'waiting' ? `<div class="kanban-hint">${icon('shield')}查看变更后，可与其他任务一起执行</div>` : ''}</div>`).join('') || `<div class="column-empty">${icon(key === 'done' ? 'checkCircle' : 'layers')}<span>${key === 'done' ? '完成的任务会归档在这里' : key === 'waiting' ? '暂时没有待确认任务' : '确认计划后，任务在这里开始'}</span></div>`}</div></section>` }).join('')}</div><div class="board-command"><div class="command-label">${icon('assistant')}下一件事，交给助手<span>演示任务不会操作真实资源</span></div>${composer(true)}</div></div>`
}

function changesList() {
  return `<div class="change-list">${state.tasks.map(task => `<section class="change-item"><div class="change-heading"><span class="task-symbol">${icon(task.icon)}</span><div><h3>${escape(task.title)}</h3><span>${task.readonly ? '只读检查，不写入配置' : '新增配置，不覆盖现有规则'}</span></div>${badge(task.status)}</div><div class="change-values">${task.changes.map(([key, before, after]) => `<div><span>${escape(key)}</span><del>${escape(before)}</del>${icon('arrow')}<strong>${escape(after)}</strong></div>`).join('')}</div></section>`).join('')}</div>`
}

function VariantC() {
  const steps = [['描述目标', '选好上下文，说出要做的事'], ['核对变更', '配置如何修改，一目了然'], ['查看结果', '分别查看每个任务的进度']]
  return `<div class="variant-c"><aside class="guide-rail"><span class="assistant-emblem small">${icon('assistant')}</span><span class="eyebrow">快速配置</span><h2>一步一步，<br>配置好你的服务。</h2><p>助手负责拆解，<br>决定权始终在你。</p><nav class="guide-steps" aria-label="配置步骤">${steps.map(([title, detail], index) => `<button data-step="${index + 1}" class="${state.step === index + 1 ? 'active' : ''}" ${state.step === index + 1 ? 'aria-current="step"' : ''}><span>${index + 1}</span><div><strong>${title}</strong><small>${detail}</small></div></button>`).join('')}</nav><div class="guide-safety">${icon('shield')}不会在你确认前修改任何配置</div></aside><section class="guide-canvas scroll-region" id="guide-scroll">${state.step === 1 ? `<div class="guide-section-head"><span class="eyebrow">01 / 描述目标</span><h2>你想接入什么服务？</h2><p>用日常语言描述，多个需求也可以一起说。</p></div><div class="goal-examples">${suggestions()}</div><form data-form="guide" class="goal-form"><label for="goal-input">这次的配置目标</label><textarea id="goal-input" rows="4" maxlength="800">${escape(state.draft || state.prompt || presets.connect.prompt)}</textarea><span class="field-hint">例如：绑定两个域名，然后添加可用性监控。</span><div class="environment-choice"><span>使用以下示例资源</span>${contextPills()}<p>${icon('info')}家庭隧道 · home.example，源站地址可在下一步核对。</p></div><div class="guide-next">${button('生成配置计划', 'guide-plan', 'arrow', 'primary', 'type="submit"')}<span>仅生成示例，不发起真实请求</span></div></form>` : state.step === 2 ? `<div class="guide-section-head"><span class="eyebrow">02 / 核对变更</span><h2>这就是将要发生的修改。</h2><p>每项任务独立执行，不会修改清单以外的资源。</p></div>${changesList()}<div class="guide-next">${button('返回修改目标', 'guide-back', 'left')}${button('核对并确认执行', 'review', 'check', 'primary')}</div>` : `<div class="guide-section-head"><span class="eyebrow">03 / 查看结果</span><h2>${counts().done === state.tasks.length && state.tasks.length ? '配置任务已完成。' : '每个任务的进度，都在这里。'}</h2><p>这是演示结果。真实服务与 DNS 没有发生变化。</p></div><div class="guide-results">${state.tasks.map(task => taskRow(task, true)).join('') || '<p class="empty-queue">还没有任务，先描述一个目标。</p>'}</div><div class="guide-next">${counts().waiting ? button('核对待执行任务', 'review', 'shield', 'primary') : button('再配置一组服务', 'new', 'plus', 'primary')}</div>`}</section></div>`
}

function switcher() {
  const count = counts()
  return `<footer class="prototype-switcher" aria-label="设计方案切换">${iconButton('left', 'previous-variant', '上一个方案')}<div><span class="variant-letter">${state.variant}</span><span class="variant-title"><strong>${variants[state.variant]}</strong><small>${state.tasks.length} 个任务 · ${count.waiting} 待确认 · ${count.running} 执行中</small></span></div>${iconButton('chevron', 'next-variant', '下一个方案')}<span class="switcher-divider"></span><button class="variant-all" data-action="variants" aria-label="查看所有设计方案">${icon('grid')}<span>3 个方案</span></button></footer>`
}

function render() {
  const focused = document.activeElement
  const focusId = focused?.id
  const selection = focused?.selectionStart
  const scrollPositions = [...document.querySelectorAll('.scroll-region')].map(element => [element.id, element.scrollTop])
  document.documentElement.dataset.theme = state.theme
  const renderer = { A: VariantA, B: VariantB, C: VariantC }[state.variant]
  document.querySelector('#root').innerHTML = `${reviewToolbar()}<div class="preview-stage" data-device="${state.device}"><div class="app-window" data-device="${state.device}"><div class="application">${sidebar()}<div class="app-main">${appHeader()}<main id="workspace" class="workspace" tabindex="-1">${renderer()}</main><div class="mobile-footer">${bottomNav()}</div></div></div></div>${state.device !== 'desktop' ? `<span class="device-caption">${state.device === 'app' ? 'Android APP 布局模拟' : '网页移动端 · 自适应布局'} · 非原生应用</span>` : ''}</div>${switcher()}`
  for (const [id, scrollTop] of scrollPositions) { const element = document.getElementById(id); if (element) element.scrollTop = scrollTop }
  if (focusId && ['prompt-input', 'goal-input'].includes(focusId)) {
    const element = document.getElementById(focusId)
    if (element && !element.disabled) { element.focus({ preventScroll: true }); element.setSelectionRange(selection, selection) }
  }
}

function updateURL() {
  const query = new URLSearchParams({ variant: state.variant, device: state.device, theme: state.theme, state: state.scenario })
  history.replaceState(null, '', `?${query}`)
}

function showLatestMessage() {
  requestAnimationFrame(() => {
    const conversation = document.querySelector('#chat-scroll')
    if (conversation && !state.empty) conversation.scrollTop = conversation.scrollHeight
  })
}

function aiSourceLabel() {
  const config = state.aiShared ? state.aiSharedConfig : state.aiProfiles[state.aiUser]
  return `${state.aiShared ? '全站共享' : '个人配置'} · ${config.endpoint && config.hasKey ? '已设置' : '未设置'}`
}

function prepareAISettings() {
  const profile = state.aiProfiles[state.aiUser]
  state.aiDraft = {
    user: state.aiUser,
    shared: state.aiShared,
    personal: { endpoint: profile.endpoint, hasKey: profile.hasKey, secret: '' },
    site: { ...state.aiSharedConfig, secret: '' },
  }
}

function aiSettingsContent() {
  const profile = state.aiProfiles[state.aiUser]
  const draft = state.aiDraft
  const managed = state.aiShared && !profile.admin
  const shared = profile.admin ? draft.shared : state.aiShared
  const config = shared ? draft.site : draft.personal
  const configured = state.aiSharedConfig.endpoint && state.aiSharedConfig.hasKey
  const previewIdentity = `<div class="ai-preview-identity"><label for="ai-preview-user">审核视角 <small>仅设计稿</small></label><select id="ai-preview-user" data-control="ai-user">${Object.entries(state.aiProfiles).map(([id, user]) => `<option value="${id}" ${id === state.aiUser ? 'selected' : ''}>${escape(user.name)} · ${user.admin ? '管理员' : '普通用户'}</option>`).join('')}</select><p>切换视角不是真实登录，会丢弃未保存的配置草稿。</p></div>`
  const sharingControl = profile.admin ? `<section class="ai-sharing"><div class="ai-sharing-heading"><div><strong>全站共享 AI 配置</strong><p>开启后，所有用户统一使用下方共享配置。</p></div><label class="ai-switch"><input id="ai-sharing" type="checkbox" role="switch" aria-label="全站共享 AI 配置" ${draft.shared ? 'checked' : ''}><span aria-hidden="true"></span></label></div><p class="ai-sharing-detail">${draft.shared ? '个人配置暂不使用，但会保留。共享密钥仅由管理员维护。' : '每位用户填写自己的端点和密钥，互不共享，包括管理员。'}</p></section>` : ''
  const fields = managed ? `<section class="ai-managed"><span class="ai-managed-icon">${icon('shield')}</span><h3>正在使用全站共享配置</h3><p>API 端点和密钥由管理员统一提供，你无需填写，也不能查看或修改共享密钥。</p><div class="ai-managed-status">${icon(configured ? 'checkCircle' : 'alert')}${configured ? '管理员已填写共享配置' : '共享配置尚未就绪，请联系管理员'}</div><p class="ai-managed-retained">个人配置${profile.endpoint && profile.hasKey ? '已保留，当前不使用' : '暂不需要填写'}。关闭全站共享后，恢复各自配置；共享失败不会改用个人密钥。</p></section><div class="dialog-footer">${button('知道了', 'close', 'check', 'primary')}</div>` : `<form data-form="ai-settings" novalidate autocomplete="off"><div class="ai-config-heading"><h3>${shared ? '管理员共享配置' : '我的 AI 配置'}</h3><span>${shared ? '供全站使用' : '仅当前用户'}</span></div><p class="ai-config-description">${shared ? '仅共享 AI 接口，不共享用户的会话、任务或资源权限。' : '端点和密钥与其他用户隔离，不会自动使用管理员配置。'}</p><div class="ai-field"><label for="ai-endpoint">API 端点</label><input id="ai-endpoint" name="endpoint" type="url" inputmode="url" value="${escape(config.endpoint)}" placeholder="https://api.example.com/v1" spellcheck="false" autocomplete="off" aria-describedby="ai-endpoint-hint"><small id="ai-endpoint-hint">填写服务商提供的 API 基础地址，不要在 URL 中加入密钥。</small></div><div class="ai-field"><label for="ai-secret">API 密钥 <span>${config.hasKey ? '已设置' : '未设置'}</span></label><input id="ai-secret" name="secret" type="password" value="${escape(config.secret)}" placeholder="${config.hasKey ? '留空保留已有密钥' : '请使用虚构密钥体验'}" autocomplete="new-password" spellcheck="false" aria-describedby="ai-secret-hint"><small id="ai-secret-hint">${config.hasKey ? '已保存的密钥不回显；输入新值可替换。' : '首次配置需要填写密钥；不展示明文。'}</small></div><p id="ai-settings-error" class="ai-form-error" role="alert" hidden></p><div class="ai-save-note">${icon('info')}仅模拟保存。请勿输入真实密钥；不会发送请求，刷新后清空。</div><div class="dialog-footer">${button('取消', 'close', '', 'secondary')}${button(profile.admin && draft.shared !== state.aiShared ? draft.shared ? '保存并开启共享' : '保存并关闭共享' : '保存配置', 'save-ai-settings', 'check', 'primary', 'type="submit"')}</div></form>`
  return `${previewIdentity}${sharingControl}${fields}`
}

function saveAISettings(form) {
  const profile = state.aiProfiles[state.aiUser]
  if (!state.aiDraft || state.aiDraft.user !== state.aiUser || (state.aiShared && !profile.admin)) return
  const draft = state.aiDraft
  const shared = profile.admin && draft.shared
  const target = shared ? draft.site : draft.personal
  const endpoint = form.querySelector('#ai-endpoint').value.trim()
  const secret = form.querySelector('#ai-secret').value.trim()
  const disablingOnly = profile.admin && state.aiShared && !shared && !endpoint && !secret && !target.hasKey
  let error = ''
  if (!disablingOnly) {
    try {
      const address = new URL(endpoint)
      if (address.protocol !== 'https:' || !address.hostname || address.username || address.password || address.search || address.hash) throw new Error('invalid endpoint')
    } catch {
      error = '填写有效的 HTTPS API 地址，不要包含账号、密钥、查询参数或片段。'
    }
    if (!error && !secret && !target.hasKey) error = '首次配置请填写一个虚构 API 密钥。'
  }
  if (error) {
    const message = form.querySelector('#ai-settings-error')
    message.hidden = false
    message.textContent = error
    return
  }
  if (!disablingOnly) {
    const destination = shared ? state.aiSharedConfig : profile
    destination.endpoint = endpoint
    destination.hasKey = Boolean(secret || target.hasKey)
  }
  if (profile.admin) state.aiShared = shared
  form.querySelector('#ai-secret').value = ''
  prepareAISettings()
  render()
  openDialog('ai-settings')
  toast('已在本次设计预览中保存。没有连接真实 AI，密钥内容未保留。')
}

function toast(message) {
  const element = document.querySelector('#toast')
  element.textContent = message
  element.classList.add('visible')
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => element.classList.remove('visible'), 3200)
}

function openDialog(type, id) {
  state.dialog = { type, id }
  const dialog = document.querySelector('#dialog')
  dialog.dataset.device = state.device
  let title = ''
  let content = ''
  if (type === 'ai-settings') {
    title = 'AI 连接配置'
    if (!state.aiDraft || state.aiDraft.user !== state.aiUser) prepareAISettings()
    content = aiSettingsContent()
  } else if (type === 'review') {
    title = '确认这次配置变更'
    const waiting = state.tasks.filter(task => task.status === 'waiting')
    state.selected = new Set(waiting.map(task => task.id))
    content = `<p class="dialog-description">逐项核对后再执行。所有内容均为示例，不会写入真实服务。</p><div class="review-scope">${icon('shield')}操作范围：家庭隧道 · home.example</div><div class="review-change-list">${waiting.map(task => `<section><label class="review-task-check"><input type="checkbox" data-select="${task.id}" checked><span><strong>${escape(task.title)}</strong><small>${task.readonly ? '只读检查' : '新增配置'}</small></span></label><div class="review-diff">${task.changes.map(([key, before, after]) => `<div><span>${escape(key)}</span><del>${escape(before)}</del><strong>${icon('arrow')}${escape(after)}</strong></div>`).join('')}</div></section>`).join('') || '<p>没有待确认的任务。</p>'}</div><div class="dialog-footer">${button('再想想', 'close', '', 'secondary')}${button(`确认并模拟执行 (${waiting.length})`, 'confirm', 'check', 'primary', `id="confirm-button" ${waiting.length ? '' : 'disabled'}`)}</div>`
  } else if (type === 'queue') {
    title = '全部任务'
    content = `<p class="dialog-description">任务独立执行，单项失败不会隐藏其他任务的结果。</p><div class="queue-dialog">${state.tasks.map(task => taskRow(task, true)).join('') || '<div class="empty-queue">还没有任务，从描述一个目标开始。</div>'}</div><div class="dialog-footer">${counts().running ? button('暂停全部', 'pause-all', 'pause') : ''}${counts().waiting ? button('核对待确认任务', 'review', 'shield', 'primary') : button('返回工作区', 'close', 'arrow', 'primary')}</div>`
  } else if (type === 'task') {
    const task = state.tasks.find(item => item.id === id)
    if (!task) return
    title = task.title
    content = `<div class="task-detail-status">${badge(task.status)}<span>${escape(task.kind)}</span></div><p class="dialog-description">${escape(task.detail)}</p><div class="detail-resource">${icon(task.icon)}<code>${escape(task.resource)}</code></div><h3 class="detail-title">执行记录 <small>模拟日志</small></h3><ol class="execution-log"><li class="done">已生成任务计划</li><li class="${task.status !== 'waiting' ? 'done' : ''}">${task.status === 'waiting' ? '等待你的确认' : '已确认当前操作范围'}</li><li class="${task.progress >= 30 ? 'done' : ''}">读取示例资源状态</li><li class="${task.status === 'done' ? 'done' : task.status === 'failed' ? 'error' : ''}">${task.status === 'failed' ? '示例源站响应超时，建议重试' : task.status === 'done' ? '模拟执行完成，结果已记录' : '等待模拟执行结果'}</li></ol><div class="dialog-footer">${button('返回队列', 'queue', 'left')}${task.status === 'running' ? button('暂停此任务', 'pause', 'pause', 'primary', `data-id="${task.id}"`) : task.status === 'paused' ? button('继续模拟', 'resume', 'play', 'primary', `data-id="${task.id}"`) : task.status === 'failed' ? button('重试此任务', 'retry', 'reset', 'primary', `data-id="${task.id}"`) : task.status === 'waiting' ? button('核对变更', 'review', 'shield', 'primary') : button('完成', 'close', 'check', 'primary')}</div>`
  } else if (type === 'context') {
    title = '这次使用的资源'
    content = `<p class="dialog-description">设计稿使用固定示例。正式功能将按账户权限提供可选资源，不向模型传递密钥。</p><div class="context-option">${icon('tunnel')}<div><strong>家庭隧道</strong><code>tunnel-home-01</code></div>${icon('checkCircle')}</div><div class="context-option">${icon('globe')}<div><strong>home.example</strong><small>示例 DNS 区域</small></div>${icon('checkCircle')}</div><div class="notice">${icon('info')}此处只展示上下文选择的设计，不读取当前账户。</div><div class="dialog-footer">${button('使用这些示例资源', 'close', 'check', 'primary')}</div>`
  } else if (type === 'create') {
    title = '快捷操作'
    content = `<div class="quick-actions"><button type="button" data-action="preview-create" data-title="新建隧道"><span class="quick-action-icon">${icon('tunnel')}</span><span class="quick-action-copy"><strong>新建隧道</strong><small>创建 Tunnel 并接入主机</small></span>${icon('chevron')}</button><button type="button" data-action="preview-create" data-title="绑定域名"><span class="quick-action-icon">${icon('globe')}</span><span class="quick-action-copy"><strong>绑定域名</strong><small>为当前服务配置访问地址</small></span>${icon('chevron')}</button><button type="button" class="quick-assistant" data-action="open-assistant"><span class="quick-action-icon">${icon('assistant')}</span><span class="quick-action-copy"><strong>AI 助手</strong><small>描述目标，快速配置与处理多项任务</small></span>${icon('chevron')}</button></div><p class="quick-actions-note">仅 AI 助手可交互体验，其他入口沿用现有功能。</p>`
  } else if (type === 'history') {
    title = '会话与工作空间'
    content = `<p class="dialog-description">切换示例会话会重置本次模拟任务。</p><div class="dialog-history">${button('新建示例会话', 'new', 'plus', 'primary')}<button data-preset="connect">${icon('chat')}家庭网络接入方案${icon('chevron')}</button><button data-preset="inspect">${icon('chat')}相册访问异常排查${icon('chevron')}</button><button data-preset="dns">${icon('chat')}博客与文档站批量绑定${icon('chevron')}</button></div>`
  } else if (type === 'model') {
    title = '演示助手'
    content = `<p class="dialog-description">这是策略选择的视觉示例，尚未接入任何模型供应商。</p><div class="ai-model-config"><span>${aiSourceLabel()}</span>${button('连接配置', 'ai-settings', 'settings', 'text-button')}</div><div class="mode-options">${[['快速', '简单查询与单项配置'], ['均衡', '日常配置与多任务协作'], ['深入', '复杂问题与详细执行计划']].map(([name, description]) => `<button data-mode="${name}"><span>${icon(name === '快速' ? 'bolt' : name === '均衡' ? 'assistant' : 'layers')}<strong>${name}</strong><small>${description}</small></span>${state.responseMode === name ? icon('checkCircle') : icon('chevron')}</button>`).join('')}</div>`
  } else {
    title = type === 'variants' ? '选择一个设计方向' : '关于这份设计稿'
    content = `<p class="dialog-description">三种布局共享同一套示例任务，但各有不同的信息重点。你可以组合喜欢的部分。</p><div class="variant-options">${Object.entries(variants).map(([key, name]) => `<button data-variant="${key}"><b>${key}</b><span><strong>${name}</strong><small>${{ A: '先对话，再确认，右侧持续查看任务', B: '任务优先，适合同时管理多项操作', C: '一步一确认，适合初次配置服务' }[key]}</small></span>${icon('chevron')}</button>`).join('')}</div><div class="notice">${icon('info')}APP 模式是浏览器内的原生布局模拟，不是 APK。所有资源、回复与执行日志均为示例；刷新即重置。</div><div class="dialog-footer">${button('继续体验', 'close', 'arrow', 'primary')}</div>`
  }
  dialog.innerHTML = `<div class="dialog-head"><div><span class="eyebrow">设计稿 · 模拟操作</span><h2 id="dialog-title">${title}</h2></div>${iconButton('close', 'close', '关闭面板')}</div><div class="dialog-body">${content}</div>`
  if (!dialog.open) dialog.showModal()
}

function closeDialog() {
  const secret = document.querySelector('#ai-secret')
  if (secret) secret.value = ''
  document.querySelector('#dialog').close()
  state.dialog = null
  state.aiDraft = null
}

function sendPrompt(value, guided = false) {
  const text = value.trim()
  if (!text) { toast('先描述一下你想完成的配置。'); document.querySelector('#prompt-input')?.focus(); return }
  if (counts().running) { toast('先暂停当前示例任务，再尝试新的配置目标。'); return }
  const key = /排查|检查|异常|无法|证书/.test(text) ? 'inspect' : /博客|文档|批量|DNS/i.test(text) ? 'dns' : 'connect'
  createPlan(key, text)
  state.draft = ''
  state.thinking = !guided
  if (guided) state.step = 2
  render()
  clearTimeout(replyTimer)
  if (!guided) replyTimer = setTimeout(() => { state.thinking = false; render(); showLatestMessage() }, 750)
}

function changeVariant(variant) { state.variant = variant; updateURL(); render(); showLatestMessage() }

function handleAction(action, element) {
  const id = element?.dataset.id
  const task = state.tasks.find(item => item.id === id)
  if (['ai-settings', 'review', 'queue', 'context', 'create', 'history', 'model', 'notes', 'variants'].includes(action)) return openDialog(action)
  if (action === 'open-assistant') {
    closeDialog()
    const target = document.querySelector('#prompt-input:not(:disabled), #goal-input') || document.querySelector('#workspace')
    target?.focus({ preventScroll: true })
    return
  }
  if (action === 'preview-create') { closeDialog(); return toast(`${element.dataset.title}沿用现有页面，本设计稿仅预览 AI 助手。`) }
  if (action === 'close') return closeDialog()
  if (action === 'theme') { state.theme = state.theme === 'dark' ? 'light' : 'dark'; updateURL(); return render() }
  if (action === 'reset' || action === 'new') { closeDialog(); applyScenario(action === 'new' ? 'empty' : 'ready'); updateURL(); render(); return toast(action === 'new' ? '已新建示例会话，模拟队列已清空。' : '示例已重置。') }
  if (action === 'previous-variant' || action === 'next-variant') { const keys = Object.keys(variants); return changeVariant(keys[(keys.indexOf(state.variant) + (action === 'next-variant' ? 1 : 2)) % keys.length]) }
  if (action === 'confirm') {
    state.tasks.forEach(item => { if (item.status === 'waiting' && state.selected.has(item.id)) { item.status = 'running'; item.progress = 0 } })
    if (state.variant === 'C') state.step = 3
    closeDialog(); render(); return toast('已开始并行模拟。没有真实配置被修改。')
  }
  if (action === 'pause-all') { state.tasks.forEach(item => { if (item.status === 'running') item.status = 'paused' }); render(); if (state.dialog) openDialog('queue'); return toast('已暂停所有模拟任务。') }
  if (['pause', 'resume', 'retry'].includes(action) && task) {
    task.status = action === 'pause' ? 'paused' : 'running'
    if (action === 'retry') task.progress = 0
    render()
    if (state.dialog) openDialog('task', id)
    return toast(action === 'pause' ? '此任务已暂停。' : '此任务已继续模拟执行。')
  }
  if (action === 'task-detail') return openDialog('task', id)
  if (action === 'guide-back') { state.step = 1; return render() }
  if (action === 'guide-plan') return
  if (action === 'focus') { if (state.variant === 'C') state.step = 1; render(); return (document.querySelector('#prompt-input') || document.querySelector('#goal-input'))?.focus() }
  if (action === 'workspace') return document.querySelector('#workspace')?.focus()
  if (action === 'placeholder') return toast('这份设计稿仅展示 AI 助手，其他功能保留现有页面。')
}

document.addEventListener('click', event => {
  const device = event.target.closest('button[data-device]')
  if (device) { state.device = device.dataset.device; updateURL(); render(); showLatestMessage(); return }
  const variant = event.target.closest('[data-variant]')
  if (variant) { closeDialog(); return changeVariant(variant.dataset.variant) }
  const preset = event.target.closest('[data-preset]')
  if (preset) { closeDialog(); return sendPrompt(presets[preset.dataset.preset].prompt, state.variant === 'C') }
  const step = event.target.closest('[data-step]')
  if (step) { state.step = Number(step.dataset.step); return render() }
  const task = event.target.closest('[data-task]')
  if (task) return openDialog('task', task.dataset.task)
  const mode = event.target.closest('button[data-mode]')
  if (mode) { state.responseMode = mode.dataset.mode; closeDialog(); render(); return toast('已切换演示策略，未连接真实模型。') }
  const action = event.target.closest('[data-action]')
  if (action) { if (action.closest('form') && action.type !== 'submit') event.preventDefault(); handleAction(action.dataset.action, action) }
})

document.addEventListener('input', event => {
  if (['prompt-input', 'goal-input'].includes(event.target.id)) state.draft = event.target.value
  if (state.aiDraft && ['ai-endpoint', 'ai-secret'].includes(event.target.id)) {
    const config = state.aiProfiles[state.aiUser].admin && state.aiDraft.shared ? state.aiDraft.site : state.aiDraft.personal
    config[event.target.id === 'ai-endpoint' ? 'endpoint' : 'secret'] = event.target.value
  }
})

document.addEventListener('change', event => {
  if (event.target.dataset.control === 'ai-user') {
    if (!Object.hasOwn(state.aiProfiles, event.target.value)) return
    state.aiUser = event.target.value
    prepareAISettings()
    render()
    openDialog('ai-settings')
  }
  if (event.target.id === 'ai-sharing' && state.aiProfiles[state.aiUser].admin) {
    state.aiDraft.shared = event.target.checked
    openDialog('ai-settings')
  }
  if (event.target.dataset.control === 'scenario') { applyScenario(event.target.value); updateURL(); render(); showLatestMessage() }
  if (event.target.dataset.select) {
    if (event.target.checked) state.selected.add(event.target.dataset.select)
    else state.selected.delete(event.target.dataset.select)
    const confirm = document.querySelector('#confirm-button')
    confirm.disabled = !state.selected.size
    confirm.querySelector('span').textContent = `确认并模拟执行 (${state.selected.size})`
  }
})

document.addEventListener('submit', event => {
  if (!event.target.dataset.form) return
  event.preventDefault()
  if (event.target.dataset.form === 'ai-settings') return saveAISettings(event.target)
  sendPrompt(event.target.querySelector('textarea').value, event.target.dataset.form === 'guide')
})

document.addEventListener('keydown', event => {
  if (event.target.id === 'prompt-input' && event.key === 'Enter' && !event.shiftKey && !event.isComposing) { event.preventDefault(); event.target.form.requestSubmit(); return }
  if (event.target.closest('input, textarea, select, [contenteditable="true"]') || document.querySelector('#dialog').open) return
  if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') { event.preventDefault(); handleAction(event.key === 'ArrowRight' ? 'next-variant' : 'previous-variant') }
})

document.querySelector('#dialog').addEventListener('click', event => { if (event.target === event.currentTarget) closeDialog() })
document.querySelector('#dialog').addEventListener('close', () => {
  const secret = document.querySelector('#ai-secret')
  if (secret) secret.value = ''
  state.dialog = null
  state.aiDraft = null
})

setInterval(() => {
  let changed = false
  state.tasks.forEach((task, index) => {
    if (task.status !== 'running') return
    changed = true
    task.progress = Math.min(100, task.progress + 11 + index * 3)
    if (task.progress === 100) task.status = 'done'
  })
  if (!changed) return
  render()
  if (state.dialog && ['queue', 'task'].includes(state.dialog.type)) openDialog(state.dialog.type, state.dialog.id)
}, 1200)

applyScenario(state.scenario)
render()
document.fonts.ready.then(showLatestMessage)
