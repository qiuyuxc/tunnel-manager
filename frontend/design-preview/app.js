const iconPaths = {
  tunnel: '<path d="M5 21V10a7 7 0 0 1 14 0v11M9 21V10a3 3 0 0 1 6 0v11M3 21h8m2 0h8"/>',
  grid: '<rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/>',
  globe: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3a17 17 0 0 1 0 18 17 17 0 0 1 0-18Z"/>',
  activity: '<path d="M2 12h5l3-8 4 16 3-8h5"/>',
  link: '<path d="m10 13 4-4m-6 6-1 1a4 4 0 0 1-6-6l4-4a4 4 0 0 1 6 0m2 3 1-1a4 4 0 0 1 6 6l-4 4a4 4 0 0 1-6 0" transform="translate(1 0)"/>',
  dns: '<rect x="3" y="3" width="18" height="7" rx="2"/><rect x="3" y="14" width="18" height="7" rx="2"/><path d="M7 6.5h.01M7 17.5h.01M12 6.5h5M12 17.5h5"/>',
  server: '<rect x="4" y="3" width="16" height="18" rx="3"/><path d="M4 9h16M4 15h16M8 6h.01M8 12h.01M8 18h.01"/>',
  flask: '<path d="M9 3h6M10 3v7L5 19a1.4 1.4 0 0 0 1 2h12a1.4 1.4 0 0 0 1-2l-5-9V3M8 15h8"/>',
  settings: '<path d="M4 7h16M4 17h16"/><circle cx="9" cy="7" r="3" fill="var(--bg)"/><circle cx="15" cy="17" r="3" fill="var(--bg)"/>',
  send: '<path d="m21 3-7 18-4-7-7-4L21 3Zm0 0L10 14"/>',
  bell: '<path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4"/>',
  shield: '<path d="m12 3 8 3v6c0 5-8 9-8 9s-8-4-8-9V6l8-3Z"/><path d="m8 12 3 3 5-6"/>',
  user: '<circle cx="12" cy="8" r="4"/><path d="M4 21v-2a8 8 0 0 1 16 0v2"/>',
  users: '<circle cx="9" cy="8" r="3"/><path d="M2 21v-2a7 7 0 0 1 14 0v2m0-17a3 3 0 0 1 0 6m3 4a6 6 0 0 1 3 5v2"/>',
  help: '<circle cx="12" cy="12" r="9"/><path d="M9.5 9a2.5 2.5 0 0 1 5 0c0 2-2.5 2-2.5 4m0 4h.01"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  search: '<circle cx="10.5" cy="10.5" r="6.5"/><path d="m16 16 5 5"/>',
  right: '<path d="m9 5 7 7-7 7"/>',
  arrow: '<path d="M4 12h16m-6-6 6 6-6 6"/>',
  back: '<path d="M20 12H4m6-6-6 6 6 6"/>',
  external: '<path d="M14 3h7v7m0-7L10 14M10 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-5"/>',
  chevron: '<path d="m6 9 6 6 6-6"/>',
  sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1 1m12 12 1 1M5 19l1-1M18 6l1-1"/>',
  moon: '<path d="M20 15.5A9 9 0 0 1 8.5 4 9 9 0 1 0 20 15.5Z"/>',
  check: '<path d="m5 12 4 4L19 6"/>',
  circlecheck: '<circle cx="12" cy="12" r="9"/><path d="m8 12 3 3 5-6"/>',
  alert: '<path d="m12 3 10 18H2L12 3Z"/><path d="M12 9v5m0 3h.01"/>',
  clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
  copy: '<rect x="8" y="8" width="12" height="13" rx="2"/><path d="M16 8V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h3"/>',
  refresh: '<path d="M20 7v5h-5M4 17v-5h5M6 6a8 8 0 0 1 13 3M5 15a8 8 0 0 0 13 3"/>',
  close: '<path d="m6 6 12 12M6 18 18 6"/>',
  menu: '<path d="M4 6h16M4 12h16M4 18h16"/>',
  more: '<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>',
  logout: '<path d="M9 4H4v16h5m5-4 4-4-4-4m-6 4h13"/>',
  key: '<circle cx="8" cy="9" r="5"/><path d="m12 13 8 8m-5-5 3-3m-1 5 3-3"/>',
  mail: '<rect x="3" y="5" width="18" height="14" rx="2"/><path d="m3 6 9 7 9-7"/>',
  lock: '<rect x="5" y="10" width="14" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3M12 14v3"/>',
  code: '<path d="m8 6-6 6 6 6m8-12 6 6-6 6M14 4l-4 16"/>',
  book: '<path d="M12 5v16M3 3l9 2 9-2v16l-9 2-9-2V3Z"/>',
  edit: '<path d="m15 4 5 5M4 20l5-1L21 7a2 2 0 0 0-4-4L5 15l-1 5Z"/>',
  trash: '<path d="M3 6h18M9 6V3h6v3M5 6l1 15h12l1-15M10 10v7m4-7v7"/>',
  download: '<path d="M12 3v12m-5-5 5 5 5-5M4 16v5h16v-5"/>',
  cloud: '<path d="M6 18a5 5 0 1 1 1-10 6 6 0 0 1 11 2 4 4 0 1 1 0 8H6Z"/>',
}

const icons = name => `<svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${iconPaths[name] || iconPaths.grid}</svg>`
const escape = value => String(value).replace(/[&<>"']/g, character => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[character])
const pages = [
  { id: 'dashboard', name: '控制面板', icon: 'grid', group: '工作空间', route: '/dashboard' },
  { id: 'tunnels', name: '隧道管理', icon: 'tunnel', group: '工作空间', route: '/tunnels' },
  { id: 'monitors', name: '服务监控', icon: 'activity', group: '工作空间', route: '/monitors' },
  { id: 'domain', name: '域名绑定', icon: 'link', group: '网络', route: '/domain' },
  { id: 'dns', name: 'DNS 管理', icon: 'dns', group: '网络', route: '/dns' },
  { id: 'lab', name: 'IP 优选实验室', icon: 'flask', group: '网络', route: '/lab/ip-selector' },
  { id: 'settings', name: '全局设置', icon: 'settings', group: '管理', route: '/settings' },
  { id: 'telegram', name: 'TG 机器人', icon: 'send', group: '管理', route: '/telegram' },
  { id: 'notifications', name: '通知设置', icon: 'bell', group: '管理', route: '/notifications' },
  { id: 'admin', name: '管理后台', icon: 'shield', group: '管理', route: '/admin' },
  { id: 'account', name: '账户设置', icon: 'user', group: '个人', route: '/account' },
  { id: 'about', name: '关于与更新', icon: 'help', group: '个人', route: '/about' },
  { id: 'tunnel-detail', name: '隧道详情', icon: 'tunnel', parent: 'tunnels', route: '/tunnels/:id' },
  { id: 'monitor-detail', name: '监控详情', icon: 'activity', parent: 'monitors', route: '/monitors/:id' },
  { id: 'domain-batch', name: '批量绑定', icon: 'link', parent: 'domain', route: '/domain/batch' },
  { id: 'login', name: '登录', icon: 'lock', public: true, route: '/login' },
  { id: 'status', name: '公开状态页', icon: 'activity', public: true, route: '/status/:token' },
  { id: 'landing', name: '访客首页', icon: 'globe', public: true, route: '/' },
  { id: 'setup', name: '首次安装', icon: 'settings', public: true, route: '/setup' },
  { id: 'design', name: '设计说明与审核', icon: 'book', route: '仅预览' },
]

const state = {
  page: 'dashboard',
  scenario: 'normal',
  theme: new URLSearchParams(location.search).get('theme') === 'light' ? 'light' : 'dark',
  tab: {},
  range: '7 天',
  tunnelId: 'home',
  currentTunnel: 'home',
  monitorId: 'blog',
  dnsFilter: '全部',
  tunnelFilter: '全部',
  selectedDns: new Set(),
  tunnels: [
    { id: 'home', name: '家庭实验室', status: 'online', uuid: '7ca921b4-12e8-4a9b-a534-000000000001', routes: 3, created: '2026-09-16', edge: 'HKG · NRT', service: 'http://localhost:8080' },
    { id: 'studio', name: '工作室服务', status: 'online', uuid: 'a185b32e-87dc-4b11-956c-000000000002', routes: 2, created: '2026-09-18', edge: 'HKG · SIN', service: 'http://localhost:3000' },
    { id: 'backup', name: '备用节点', status: 'offline', uuid: 'f2d85a19-d3ba-4d7a-9bcb-000000000003', routes: 0, created: '2026-09-21', edge: '未连接', service: 'http://localhost:9000' },
  ],
  monitors: [
    { id: 'blog', name: '个人博客', host: 'blog.tunnel.example', latency: 38, uptime: '100.00', status: 'online' },
    { id: 'nas', name: '家庭 NAS', host: 'home.tunnel.example', latency: 42, uptime: '100.00', status: 'online' },
    { id: 'api', name: 'API 服务', host: 'api.tunnel.example', latency: 68, uptime: '99.76', status: 'warn' },
    { id: 'status-page', name: '状态页面', host: 'status.tunnel.example', latency: 36, uptime: '100.00', status: 'online' },
  ],
  dns: [
    { id: 'dns-blog', type: 'CNAME', name: 'blog', value: '7ca921b4.cfargotunnel.com', proxy: true, ttl: '自动' },
    { id: 'dns-home', type: 'CNAME', name: 'home', value: '7ca921b4.cfargotunnel.com', proxy: true, ttl: '自动' },
    { id: 'dns-api', type: 'CNAME', name: 'api', value: 'a185b32e.cfargotunnel.com', proxy: true, ttl: '自动' },
    { id: 'dns-root', type: 'A', name: '@', value: '192.0.2.28', proxy: false, ttl: '300 秒' },
    { id: 'dns-mail', type: 'TXT', name: '_dmarc', value: 'v=DMARC1; p=quarantine', proxy: false, ttl: '自动' },
    { id: 'dns-status', type: 'CNAME', name: 'status', value: 'pages.tunnel.example', proxy: true, ttl: '自动' },
  ],
  routes: [
    { host: 'home.tunnel.example', service: 'http://localhost:8080', tunnel: 'home' },
    { host: 'blog.tunnel.example', service: 'http://localhost:3000', tunnel: 'home' },
    { host: 'files.tunnel.example', service: 'http://localhost:8096', tunnel: 'home' },
  ],
  switches: {},
  drafts: {},
}

const badge = (text, kind = '', dot = true) => `<span class="badge ${kind}">${dot ? '<span class="dot"></span>' : ''}${escape(text)}</span>`
const button = (text, icon, action, style = '', attrs = '') => `<button type="button" class="btn ${style}" data-action="${action}" ${attrs}>${icon ? icons(icon) : ''}${text}</button>`
const link = (text, page, icon = 'arrow') => `<a class="text-link" href="#${page}">${text}${icons(icon)}</a>`
const panelHead = (title, subtitle = '', action = '') => `<div class="panel-heading"><div><h2>${title}</h2>${subtitle ? `<p>${subtitle}</p>` : ''}</div>${action}</div>`
const heading = (title, subtitle, eyebrow, action = '') => `<div class="page-heading"><div><span class="eyebrow">${eyebrow}</span><h1>${title}</h1><p>${subtitle}</p></div>${action}</div>`
const kv = (key, value) => `<div class="kv"><span>${key}</span><span>${value}</span></div>`
const field = (label, name, value = '', hint = '', type = 'text', attrs = '') => `<label class="field"><span>${label}</span><input name="${name}" type="${type}" value="${escape(value)}" ${attrs}>${hint ? `<small>${hint}</small>` : ''}</label>`
const selectField = (label, name, options, hint = '') => `<label class="field"><span>${label}</span><select name="${name}">${options.map(option => `<option value="${escape(Array.isArray(option) ? option[0] : option)}">${escape(Array.isArray(option) ? option[1] : option)}</option>`).join('')}</select>${hint ? `<small>${hint}</small>` : ''}</label>`
const notice = (text, kind = '') => `<div class="notice ${kind}">${icons(kind === 'neutral' ? 'help' : 'alert')}<span>${text}</span></div>`
const saveFooter = text => `<div class="form-footer"><button class="btn primary" type="submit">${icons('check')}${text || '保存更改'}</button></div>`
const tabs = options => `<div class="tabs" role="tablist">${options.map((option, index) => `<button role="tab" aria-selected="${(state.tab[state.page] || options[0]) === option}" class="${(state.tab[state.page] || options[0]) === option ? 'active' : ''}" data-action="tab" data-tab="${option}" id="tab-${index}" aria-controls="tab-content">${option}</button>`).join('')}</div>`
const segment = (options, current, action) => `<div class="segment" role="group" aria-label="${action === 'range' ? '图表时间范围' : '筛选列表'}">${options.map(option => `<button class="${current === option ? 'active' : ''}" aria-pressed="${current === option}" data-action="${action}" data-value="${option}">${option}</button>`).join('')}</div>`
const toggle = (id, title, description, initial = true) => `<div class="setting-row"><div><strong>${title}</strong><p>${description}</p></div><button class="switch" role="switch" aria-label="${title}" aria-checked="${state.switches[id] ?? initial}" data-action="toggle" data-id="${id}"><span></span></button></div>`
const stats = items => `<div class="stats-strip">${items.map(item => `<div><span class="metric-label">${item[0]}</span><strong class="metric-value ${item[3] || ''}">${item[1]}</strong><span class="metric-note">${item[2]}</span></div>`).join('')}</div>`
const table = (headers, body, count, extra = '') => `<div class="panel table-panel"><div class="table-scroll" tabindex="0" role="region" aria-label="${pages.find(page => page.id === state.page)?.name || '数据'}列表，可横向滚动"><table><thead><tr>${headers.map(header => `<th scope="col">${header}</th>`).join('')}</tr></thead><tbody>${body}</tbody></table></div><div class="table-footer"><span data-count>${count}</span><span>示例数据 · 当前会话有效</span></div></div>${extra}`
const search = placeholder => `<label class="search-field">${icons('search')}<input type="search" data-filter="list" placeholder="${placeholder}" aria-label="${placeholder}"></label>`
const brand = () => `<a class="brand" href="#dashboard" aria-label="Tunnel Manager 控制面板"><span class="brand-mark">${icons('tunnel')}</span><div><div class="brand-name">Tunnel Manager</div><div class="brand-sub">YOUR CONNECTION HUB</div></div></a>`

function previewBar() {
  return `<div class="preview-bar"><div class="preview-label"><strong>WEB / 01</strong><span class="draft">待审核</span><span class="preview-explainer secondary">交互设计稿 · 示例数据，不连接真实服务</span></div><div class="preview-tools"><label><span>页面</span><select aria-label="切换设计稿页面" data-control="page">${pages.map(page => `<option value="${page.id}" ${state.page === page.id ? 'selected' : ''}>${page.name}</option>`).join('')}</select></label><label><span>状态</span><select class="state-select" aria-label="预览页面状态" data-control="scenario" ${['dashboard', 'tunnels', 'monitors', 'dns'].includes(state.page) ? '' : 'disabled'}>${[['normal', '正常'], ['empty', '空列表'], ['error', '请求失败'], ['loading', '加载中']].map(([value, label]) => `<option value="${value}" ${state.scenario === value ? 'selected' : ''}>${label}</option>`).join('')}</select></label></div></div>`
}

function sidebar() {
  const current = pages.find(page => page.id === state.page)
  const selected = current.parent || state.page
  const groups = ['工作空间', '网络', '管理']
  const navItem = page => `<a class="nav-link ${page.id === selected ? 'active' : ''}" href="#${page.id}" ${page.id === selected ? 'aria-current="page"' : ''}>${icons(page.icon)}<span>${page.name}</span>${page.id === 'tunnels' ? `<span class="nav-count">${state.tunnels.length}</span>` : page.id === 'lab' ? '<span class="nav-count">实验</span>' : ''}</a>`
  return `<button class="sidebar-backdrop" data-action="menu-close" aria-label="关闭导航"></button><aside class="sidebar" aria-label="主导航">${brand()}<button class="icon-btn drawer-close" data-action="menu-close" aria-label="关闭导航">${icons('close')}</button><button class="workspace-select" data-action="workspace"><span class="workspace-icon">${icons('server')}</span><span class="grow"><strong>Homelab 工作空间</strong><small>个人实例 · 管理员视角</small></span>${icons('chevron')}</button><nav class="nav-scroll">${groups.map(group => `<div class="nav-group"><span class="nav-caption">${group}</span>${pages.filter(page => page.group === group).map(navItem).join('')}</div>`).join('')}</nav><div class="sidebar-bottom">${pages.filter(page => page.group === '个人').map(navItem).join('')}<div class="sidebar-foot"><span class="row" style="gap:5px"><span class="dot accent"></span>示例实例</span><a href="#design">设计说明 ↗</a></div></div></aside>`
}

function mobileNav() {
  const current = pages.find(page => page.id === state.page)
  const selected = current.parent || state.page
  return `<nav class="mobile-nav" aria-label="移动端快捷导航"><a href="#dashboard" class="${selected === 'dashboard' ? 'active' : ''}">${icons('grid')}<span>概览</span></a><a href="#monitors" class="${selected === 'monitors' ? 'active' : ''}">${icons('activity')}<span>监控</span></a><button class="mobile-create" data-action="create" aria-label="新建项目">${icons('plus')}</button><a href="#dns" class="${selected === 'dns' ? 'active' : ''}">${icons('dns')}<span>DNS</span></a><button data-action="menu" aria-label="展开更多导航">${icons('more')}<span>更多</span></button></nav>`
}

function topbar() {
  const current = pages.find(page => page.id === state.page)
  return `<header class="topbar"><div class="row"><button class="icon-btn mobile-menu-btn" data-action="menu" aria-label="展开导航">${icons('menu')}</button><div class="breadcrumb"><span class="workspace-crumb">Homelab 工作空间</span><span class="workspace-crumb">/</span>${current.parent ? `<a href="#${current.parent}">${pages.find(page => page.id === current.parent).name}</a><span>/</span>` : ''}<span>${current.name}</span></div></div><div class="topbar-actions"><button class="search-trigger" data-action="search">${icons('search')}快速前往<kbd>⌘ K</kbd></button><button class="icon-btn" data-action="theme" aria-label="切换深浅主题">${icons(state.theme === 'dark' ? 'sun' : 'moon')}</button><a class="icon-btn notification-btn" href="#notifications" aria-label="通知设置">${icons('bell')}</a><a class="avatar" href="#account" aria-label="Homelab 账户设置">HL</a></div></header>`
}

const samples = [
  [39, 105], [42, 134], [36, 92], [40, 148], [43, 126], [38, 97], [45, 181],
  [39, 118], [44, 165], [41, 144], [37, 110], [48, 217], [51, 386], [43, 161],
  [42, 124], [40, 157], [38, 133], [47, 221], [43, 175], [46, 206], [41, 159],
  [39, 116], [44, 165], [40, 138], [45, 178], [43, 148], [39, 125], [42, 153],
]

function chart() {
  const rangeSamples = state.range === '24 小时' ? samples.slice(-12) : samples
  const labels = state.range === '24 小时' ? ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00'] : ['09.16', '09.17', '09.18', '09.19', '09.20', '09.21', '09.22']
  return `<div class="chart-wrap"><div class="chart-y"><span>400 ms</span><span>300</span><span>200</span><span>100</span><span>0</span></div><div class="chart-grid" role="group" aria-label="${state.range}示例平均与峰值延迟柱状图，点击柱子查看精确数值">${[0, 25, 50, 75, 100].map(value => `<span class="chart-gridline" style="bottom:${value}%"></span>`).join('')}${rangeSamples.map(([average, peak], index) => {
    const label = state.range === '24 小时' ? `09.22 ${String(index * 2).padStart(2, '0')}:00` : `09.${16 + Math.floor(index / 4)} ${String(index % 4 * 6).padStart(2, '0')}:00`
    const title = `${label}，平均 ${average} ms，峰值 ${peak} ms${peak === 386 ? '，API 服务曾出现异常' : '，正常'}`
    return `<button class="chart-column" data-action="chart-point" data-average="${average}" data-peak="${peak}" data-label="${label}" aria-label="${title}" title="${title}"><span class="chart-peak ${peak === 386 ? 'warn' : ''}" style="--peak:${peak / 4}%;--average:${average / peak * 100}%"><span class="chart-avg"></span></span></button>`
  }).join('')}</div><div class="chart-axis">${labels.map(label => `<span>${label}</span>`).join('')}</div></div><div class="chart-legend"><span><i class="legend-swatch peak"></i>峰值</span><span><i class="legend-swatch"></i>平均（正常）</span><span><i class="legend-swatch warn"></i>出现异常</span><span><i class="legend-swatch danger"></i>出现不可达</span><span style="margin-left:auto">点击查看明细</span></div>`
}

function serviceBars(status) {
  return `<div class="service-bars" aria-label="最近 24 个探测时段${status === 'warn' ? '有 2 次延迟异常' : '全部正常'}">${Array.from({ length: 24 }, (_, index) => `<span class="${status === 'warn' && [14, 15].includes(index) ? 'warn' : ''}"></span>`).join('')}</div>`
}

function monitorRows() {
  return `<div class="service-list">${state.monitors.map(monitor => `<a href="#monitor-detail" data-monitor="${monitor.id}" class="service-row"><div><strong class="service-name">${escape(monitor.name)}</strong><span class="service-url mono">${escape(monitor.host)}</span></div>${serviceBars(monitor.status)}<span class="numeric small ${monitor.status === 'warn' ? 'amber' : 'secondary'}">${monitor.latency} <span class="muted">ms</span></span>${badge(monitor.status === 'warn' ? '延迟偏高' : '运行正常', monitor.status === 'warn' ? 'warn' : '')}${icons('right')}</a>`).join('')}</div>`
}

function quickItem(icon, title, description, page) {
  return `<a class="quick-item" href="#${page}"><span class="quick-icon">${icons(icon)}</span><span class="grow"><strong>${title}</strong><small>${description}</small></span>${icons('right')}</a>`
}

function dashboard() {
  const current = state.tunnels.find(tunnel => tunnel.id === state.currentTunnel)
  const healthy = state.monitors.filter(monitor => monitor.status === 'online').length
  const warning = state.monitors.length - healthy
  return heading('控制面板', '连接的每一处，都心中有数。', 'Workspace / Overview', `<div class="row">${button('刷新', 'refresh', 'refresh')}${button('绑定域名', 'plus', 'goto-domain', 'primary')}</div>`) +
    `<section class="panel connection-strip" aria-label="当前连接"><div class="row"><span class="connection-symbol">${icons('tunnel')}</span><div><span class="eyebrow">当前隧道</span><h2 class="connection-name">${escape(current?.name || '尚未选择')}</h2>${badge(current ? '配置就绪' : '待配置', current ? '' : 'warn')}</div></div><div class="endpoint"><span class="eyebrow">转发地址</span><code>${escape(current?.service || '尚未配置')}</code></div><div class="endpoint preferred-endpoint"><span class="eyebrow">优选 CNAME</span><code>edge.tunnel.example</code></div><a class="btn small-btn" href="#tunnels">管理隧道${icons('right')}</a></section>` +
    `<div class="dashboard-grid"><div class="stack"><section class="panel">${panelHead('监控概览', '服务可用性与响应延迟', segment(['24 小时', '7 天'], state.range, 'range'))}<div class="metrics"><div><span class="metric-label">${state.range === '7 天' ? '7 天' : '24 小时'}可用率</span><strong class="metric-value">${state.range === '7 天' ? '99.94' : '100.00'}<span class="unit">%</span></strong><span class="metric-note accent">${state.range === '7 天' ? '整体运行稳定' : '本时段无不可达'}</span></div><div><span class="metric-label">平均延迟</span><strong class="metric-value">${state.range === '7 天' ? '46' : '43'}<span class="unit">ms</span></strong><span class="metric-note">所有监控项目</span></div><div><span class="metric-label">峰值延迟</span><strong class="metric-value">${state.range === '7 天' ? '386' : '221'}<span class="unit">ms</span></strong><span class="metric-note">保留异常，不平滑峰值</span></div><div><span class="metric-label">项目健康状态</span><strong class="metric-value metric-split"><span class="accent">${healthy}</span><em>/</em><span class="amber">${warning}</span><em>/</em><span class="secondary">0</span></strong><span class="metric-note">正常 / 异常 / 不可达</span></div></div>${chart()}</section><section class="panel">${panelHead('监控项目', `${state.monitors.length} 个服务 · 每 60 秒探测一次`, link('查看全部', 'monitors'))}${monitorRows()}</section></div><aside class="stack dashboard-aside"><section class="panel">${panelHead('快捷操作')}${quickItem('link', '绑定域名', '将本地服务发布到公网', 'domain')}${quickItem('dns', '管理 DNS', '查看与维护解析记录', 'dns')}${quickItem('activity', '服务监控', '探测可用性与响应时间', 'monitors')}${quickItem('globe', '公开状态页', '分享服务运行情况', 'status')}</section><section class="panel">${panelHead('连接配置', '', badge('已就绪', '', false))}<div class="status-summary">${icons('shield')}<div><strong>发布配置已就绪</strong><p>配置就绪不代表所有服务正常</p></div></div>${kv('当前隧道', escape(current?.name || '尚未选择'))}${kv('转发地址', current ? '已配置' : '待配置')}${kv('优选 CNAME', '<code>edge.tunnel.example</code>')}${kv('Cloudflare 授权', '<span class="accent">已配置 · 示例</span>')}<div class="mt">${link('查看全局配置', 'settings')}</div></section>${notice('API 服务存在延迟异常。可前往监控详情查看探测记录，而不必离开当前工作空间。')}</aside></div>`
}

function tunnels() {
  const filtered = state.tunnels.filter(tunnel => state.tunnelFilter === '全部' || (state.tunnelFilter === '在线' ? tunnel.status === 'online' : tunnel.status !== 'online'))
  const body = filtered.map(tunnel => `<tr data-search="${escape(`${tunnel.name} ${tunnel.uuid}`.toLowerCase())}"><td><div class="row"><span class="quick-icon">${icons('tunnel')}</span><div><a href="#tunnel-detail" data-tunnel="${tunnel.id}"><strong>${escape(tunnel.name)}</strong></a><small class="mono">${tunnel.uuid.slice(0, 8)}</small></div>${tunnel.id === state.currentTunnel ? badge('当前', 'neutral', false) : ''}</div></td><td>${badge(tunnel.status === 'online' ? '在线' : '未连接', tunnel.status === 'online' ? '' : 'neutral')}</td><td class="mono">${tunnel.edge}</td><td class="numeric">${tunnel.routes}</td><td class="secondary number">${tunnel.created}</td><td><a class="text-link" href="#tunnel-detail" data-tunnel="${tunnel.id}">查看详情${icons('right')}</a></td></tr>`).join('')
  return heading('隧道管理', '让本地服务，安全地连接更大的世界。', 'Workspace / Tunnels', button('创建隧道', 'plus', 'new-tunnel', 'primary')) + stats([['全部隧道', state.tunnels.length.toString().padStart(2, '0'), '当前账户下的隧道'], ['在线', state.tunnels.filter(tunnel => tunnel.status === 'online').length.toString().padStart(2, '0'), '连接器正常运行', 'accent'], ['未连接', state.tunnels.filter(tunnel => tunnel.status !== 'online').length.toString().padStart(2, '0'), '等待连接器上线', 'secondary']]) + `<div class="toolbar">${search('搜索隧道名称或 ID')}${segment(['全部', '在线', '未连接'], state.tunnelFilter, 'tunnel-filter')}</div>` + table(['隧道名称', '状态', '连接节点', '发布路由', '创建时间', '操作'], body, `${filtered.length} 条隧道`) + `<div class="mt">${notice('「当前隧道」用于域名绑定默认配置。切换当前隧道不会中断其他正在运行的隧道。', 'neutral')}</div>`
}

function tunnelDetail() {
  const tunnel = state.tunnels.find(item => item.id === state.tunnelId) || state.tunnels[0]
  if (!tunnel) return emptyState('tunnels')
  const routes = state.routes.filter(route => route.tunnel === tunnel.id)
  return heading(escape(tunnel.name), '连接信息、发布路由与凭据，都在这里。', 'Workspace / Tunnel detail', `<div class="row">${badge(tunnel.status === 'online' ? '在线' : '未连接', tunnel.status === 'online' ? '' : 'neutral')}${button(tunnel.id === state.currentTunnel ? '当前隧道' : '设为当前隧道', 'check', 'select-tunnel', 'primary', `data-id="${tunnel.id}" ${tunnel.id === state.currentTunnel ? 'disabled' : ''}`)}</div>`) + `<div class="two-column"><div class="stack"><section class="panel">${panelHead('连接信息', '', button('复制 ID', 'copy', 'copy', 'small-btn', `data-value="${tunnel.uuid}"`))}${kv('隧道 ID', `<code>${tunnel.uuid}</code>`)}${kv('连接节点', tunnel.edge)}${kv('创建时间', tunnel.created)}${kv('转发地址', `<code>${escape(tunnel.service)}</code>`)}</section><section class="panel">${panelHead('已发布路由', `${routes.length} 条路由`, link('绑定域名', 'domain'))}${routes.length ? routes.map(route => `<div class="setting-row"><div><strong class="mono">${escape(route.host)}</strong><p class="mono">${escape(route.service)}</p></div>${badge('已发布')}</div>`).join('') : '<p class="secondary small">暂无路由，可通过域名绑定发布服务。</p>'}</section><section class="panel danger-zone">${panelHead('危险操作')}<div class="setting-row"><div><strong>删除隧道</strong><p>正式操作会影响相关路由，必须再次确认。</p></div>${button('删除', 'trash', 'delete-tunnel', 'danger', `data-id="${tunnel.id}"`)}</div></section></div><aside class="stack"><section class="panel">${panelHead('连接器令牌')}<p class="secondary small">令牌可用于连接隧道，请勿分享或提交到代码仓库。</p><div class="notice neutral mt"><code>••••••••••••••••••••••••</code></div><div class="mt">${button('查看示例令牌', 'key', 'token', '', `data-id="${tunnel.id}"`)}</div></section><section class="panel">${panelHead('下一步')}${quickItem('link', '发布本地服务', '将域名绑定到这条隧道', 'domain')}${quickItem('activity', '添加服务监控', '持续了解服务状态', 'monitors')}</section></aside></div>`
}

function monitors() {
  return heading('服务监控', '及时发现变化，让每一次访问都有回应。', 'Workspace / Monitoring', `<div class="row"><a class="btn" href="#status">${icons('external')}公开状态页</a>${button('添加监控', 'plus', 'new-monitor', 'primary')}</div>`) + stats([['监控项目', String(state.monitors.length).padStart(2, '0'), 'HTTP / HTTPS 可用性探测'], ['运行正常', String(state.monitors.filter(item => item.status === 'online').length).padStart(2, '0'), '最近一次探测成功', 'accent'], ['需要关注', String(state.monitors.filter(item => item.status === 'warn').length).padStart(2, '0'), '有服务响应延迟偏高', 'amber']]) + `<div class="toolbar">${search('搜索监控名称或地址')}${button('刷新探测结果', 'refresh', 'refresh')}</div>` + table(['监控项目', '最近探测记录', '可用率（7 天）', '响应时间', '状态', '操作'], state.monitors.map(monitor => `<tr data-search="${escape(`${monitor.name} ${monitor.host}`.toLowerCase())}"><td><a href="#monitor-detail" data-monitor="${monitor.id}"><strong>${escape(monitor.name)}</strong></a><small class="mono">${escape(monitor.host)}</small></td><td style="min-width:150px">${serviceBars(monitor.status)}</td><td class="numeric number">${monitor.uptime}%</td><td class="numeric number">${monitor.latency} ms</td><td>${badge(monitor.status === 'warn' ? '延迟偏高' : '运行正常', monitor.status === 'warn' ? 'warn' : '')}</td><td><a class="text-link" href="#monitor-detail" data-monitor="${monitor.id}">详情${icons('right')}</a></td></tr>`).join(''), `${state.monitors.length} 个监控项目`) + `<div class="mt">${notice('可用率、平均延迟与峰值保持独立语义。黄色表示延迟异常，红色表示不可达；状态始终同时提供文字。', 'neutral')}</div>`
}

function monitorDetail() {
  const monitor = state.monitors.find(item => item.id === state.monitorId) || state.monitors[0]
  if (!monitor) return emptyState('monitors')
  const activeTab = state.tab['monitor-detail'] || '运行概览'
  return heading(escape(monitor.name), escape(`https://${monitor.host}`), 'Workspace / Monitor detail', `<div class="row">${badge(monitor.status === 'warn' ? '延迟偏高' : '运行正常', monitor.status === 'warn' ? 'warn' : '')}${button('立即探测', 'refresh', 'probe')}</div>`) + tabs(['运行概览', '探测记录', '监控设置']) + `<div id="tab-content" role="tabpanel">${activeTab === '监控设置' ? `<form class="panel form-panel" data-form="monitor-settings">${panelHead('监控配置')}<div class="stack">${field('项目名称', 'name', monitor.name, '', 'text', 'required')}${field('探测地址', 'url', `https://${monitor.host}`, '', 'url', 'required')}${selectField('探测间隔', 'interval', ['60 秒', '120 秒', '300 秒'])}${toggle('monitor-enabled', '启用监控', '持续探测这个服务的可用性。')}</div>${saveFooter()}</form>` : activeTab === '探测记录' ? table(['时间', 'HTTP 状态', '响应延迟', '探测结果'], samples.slice(-8).reverse().map(([average], index) => `<tr><td class="number">09-22 14:${String(32 - index).padStart(2, '0')}:00</td><td class="mono">200 OK</td><td class="numeric number">${average} ms</td><td>${badge('成功')}</td></tr>`).join(''), '最近 8 条示例探测记录') : `<div class="two-column"><div class="stack"><section class="panel">${panelHead('响应延迟', '聚合视图 · 示例数据', segment(['24 小时', '7 天'], state.range, 'range'))}${stats([['可用率', `${monitor.uptime}<span class="unit">%</span>`, '最近 7 天', 'accent'], ['平均延迟', `${monitor.latency}<span class="unit">ms</span>`, '当前项目'], ['探测间隔', '60<span class="unit">s</span>', 'HTTP GET']])}${chart()}</section><section class="panel">${panelHead('事件记录')}<div class="event-row"><time>09-22 14:32</time><div><p>${monitor.status === 'warn' ? '检测到响应延迟偏高' : '探测正常'}</p><small>${monitor.status === 'warn' ? 'API 服务延迟超过配置阈值，请检查源站。' : '服务返回 HTTP 200，探测成功。'}</small></div></div><div class="event-row"><time>09-16 09:12</time><div><p>监控已创建</p><small>开始定期检测服务状态。</small></div></div></section></div><aside class="stack"><section class="panel">${panelHead('探测配置')}${kv('请求方法', 'GET')}${kv('间隔', '60 秒')}${kv('超时', '10 秒')}${kv('预期状态码', '200–299')}${kv('公开展示', '已启用')}</section><section class="panel">${panelHead('公开状态页')}<p class="small secondary">分享可用性，不暴露源站地址与敏感配置。</p><div class="mt">${link('预览公开页面', 'status')}</div></section></aside></div>`}</div>`
}

function connectionFlow() {
  return `<div class="flow"><div class="flow-node">${icons('globe')}<div><strong>你的域名</strong><small class="mono">app.tunnel.example</small></div></div><div class="flow-arrow">↓</div><div class="flow-node">${icons('cloud')}<div><strong>Cloudflare 网络</strong><small>DNS 解析与代理</small></div></div><div class="flow-arrow">↓</div><div class="flow-node">${icons('tunnel')}<div><strong>家庭实验室</strong><small>Cloudflare Tunnel</small></div></div><div class="flow-arrow">↓</div><div class="flow-node">${icons('server')}<div><strong>本地服务</strong><small class="mono">http://localhost:8080</small></div></div></div>`
}

function domain(batch = false) {
  return heading(batch ? '批量绑定域名' : '绑定域名', batch ? '一组服务，一次完成发布配置。' : '从一个域名开始，让你的服务触手可及。', 'Network / Domain binding', link(batch ? '单个绑定' : '批量绑定', batch ? 'domain' : 'domain-batch')) + `<div class="two-column"><form class="panel form-panel" data-form="${batch ? 'batch-bind' : 'bind'}"><div class="steps"><div class="step active"><span>1</span>配置连接</div><i class="step-line"></i><div class="step"><span>2</span>确认绑定</div><i class="step-line"></i><div class="step"><span>3</span>完成</div></div>${panelHead('域名与源站', '保留现有 DNS、路由与 CNAME 配置能力')}<div class="stack">${selectField('Cloudflare 区域', 'zone', ['tunnel.example'], '示例区域，不会读取你的真实账户。')}${batch ? `<label class="field"><span>子域名与服务地址</span><textarea name="entries" required placeholder="blog,http://localhost:3000">blog,http://localhost:3000\nfiles,http://localhost:8096</textarea><small>每行一个：子域名,服务地址。提交前统一确认。</small></label>` : `<div class="equal-column">${field('子域名', 'hostname', '', '仅填写主机名，如 app。', 'text', 'placeholder="app" required pattern="[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?"')}${field('服务地址', 'service', 'http://localhost:8080', '支持 http:// 或 https://。', 'url', 'required')}</div>`}${selectField('使用隧道', 'tunnel', state.tunnels.map(tunnel => [tunnel.id, `${tunnel.name}${tunnel.id === state.currentTunnel ? ' · 当前' : ''}`]))}${selectField('优选 CNAME', 'cname', ['使用全局默认 · edge.tunnel.example', '直接连接 Cloudflare Tunnel'])}${toggle('domain-proxy', '开启 Cloudflare 代理', '通过 Cloudflare 网络代理访问。')}${notice('预览只展示配置与确认流程，不会创建 DNS 记录、发布路由或发起网络请求。', 'neutral')}</div>${saveFooter('预览绑定配置')}</form><aside class="stack"><section class="panel">${panelHead('连接路径', '从公网访问到本地服务')}${connectionFlow()}</section><section class="panel">${panelHead('绑定前检查')}${kv('Cloudflare 授权', '<span class="accent">已配置 · 示例</span>')}${kv('可用隧道', String(state.tunnels.length))}${kv('DNS 冲突检查', '确认时执行 · 演示')}</section></aside></div>`
}

function dns() {
  const records = state.dns.filter(record => state.dnsFilter === '全部' || (state.dnsFilter === 'A / AAAA' ? ['A', 'AAAA'].includes(record.type) : record.type === state.dnsFilter))
  const body = records.map(record => `<tr data-search="${escape(`${record.name} ${record.value}`.toLowerCase())}"><td><input type="checkbox" data-dns-select="${record.id}" aria-label="选择 ${escape(record.name)}" ${state.selectedDns.has(record.id) ? 'checked' : ''}></td><td>${badge(record.type, 'neutral', false)}</td><td><strong>${escape(record.name)}</strong><small>tunnel.example</small></td><td><code>${escape(record.value)}</code></td><td><span class="row ${record.proxy ? 'accent' : 'muted'}">${icons('cloud')}<span>${record.proxy ? '已代理' : '仅 DNS'}</span></span></td><td class="secondary">${record.ttl}</td><td><div class="row"><button class="icon-btn" data-action="edit-dns" data-id="${record.id}" aria-label="编辑 ${escape(record.name)}">${icons('edit')}</button><button class="icon-btn red" data-action="delete-dns" data-id="${record.id}" aria-label="删除 ${escape(record.name)}">${icons('trash')}</button></div></td></tr>`).join('')
  return heading('DNS 管理', '每一条解析，都井然有序。', 'Network / DNS records', button('添加记录', 'plus', 'new-dns', 'primary')) + `<section class="panel connection-strip"><div class="row"><span class="connection-symbol">${icons('globe')}</span><div><span class="eyebrow">当前区域</span><h2 class="connection-name">tunnel.example</h2><span class="small muted">Cloudflare 托管 · ${state.dns.length} 条记录</span></div></div><div class="endpoint"><span class="eyebrow">解析状态</span><div style="margin-top:7px">${badge('区域已激活')}</div></div><div class="preferred-endpoint endpoint"><span class="eyebrow">默认 TTL</span><code>自动</code></div>${button('切换区域', 'chevron', 'zone', 'small-btn')}</section><div class="toolbar">${search('搜索主机名或记录值')}${segment(['全部', 'A / AAAA', 'CNAME', 'TXT'], state.dnsFilter, 'dns-filter')}</div>` + table(['<input type="checkbox" data-dns-select="all" aria-label="选择当前列表全部记录">', '类型', '主机名', '记录值', '代理状态', 'TTL', '操作'], body, `共 ${records.length} 条记录`, `<div class="selection-bar" id="dns-selection" ${state.selectedDns.size ? '' : 'hidden'}><span>已选择 <strong>${state.selectedDns.size}</strong> 条记录</span>${button('删除选中', 'trash', 'delete-selected', 'danger small-btn')}</div>`) + `<p class="small muted mt">代理状态与 TTL 独立展示。移动端可横向滚动查看完整记录值。</p>`
}

function settings() {
  const tab = state.tab.settings || '站点信息'
  const content = tab === '站点信息' ? `${panelHead('站点信息', '定义控制台对外呈现的名称与描述')}<div class="stack">${field('站点名称', 'site-name', 'Tunnel Manager', '', 'text', 'required')}${field('站点描述', 'site-description', '让每一次连接，都更简单。')}${toggle('landing-enabled', '展示访客首页', '未登录用户可以先了解本站。')}${toggle('register-enabled', '允许注册', '注册策略仍由后台权限与安全配置控制。', false)}</div>` : tab === '网络配置' ? `${panelHead('网络默认值', '供域名绑定与隧道发布使用')}<div class="stack">${field('默认优选 CNAME', 'preferred-cname', 'edge.tunnel.example')}${field('回退源地址', 'fallback', 'https://fallback.tunnel.example', '', 'url')}${selectField('当前隧道', 'current-tunnel', state.tunnels.map(tunnel => [tunnel.id, tunnel.name]))}${notice('修改默认配置不应悄悄覆盖已有记录。正式接入时保留现有提交与校验逻辑。', 'neutral')}</div>` : `${panelHead('外观与体验', '与 App 保持同一视觉语言，不牺牲网页端效率')}<div class="setting-row"><div><strong>深浅主题</strong><p>切换不会重建页面，也不会清空表单。</p></div>${button('切换主题', state.theme === 'dark' ? 'sun' : 'moon', 'theme')}</div>${toggle('light-effects', '轻量效果', '关闭非必要动画与玻璃模糊。', false)}<div class="setting-row"><div><strong>系统字体</strong><p>优先 MiSans，回退到系统中文字体；不请求远程字体。</p></div>${badge('跟随系统', 'neutral', false)}</div>`
  return heading('全局设置', '你的空间，你来定义。', 'Manage / Settings', badge('管理员', 'neutral', false)) + tabs(['站点信息', '网络配置', '外观体验']) + `<div class="two-column"><form class="panel form-panel" data-form="settings-${tab}" id="tab-content" role="tabpanel">${content}${saveFooter()}</form><aside class="stack"><section class="panel">${panelHead('设置范围')}<p class="small secondary">站点与网络设置属于实例级配置。个人安全、邮箱和通行密钥，请前往账户设置。</p><div class="mt">${link('账户设置', 'account')}</div></section>${notice('当前为隔离设计稿。保存只演示确认反馈，不会改变正在运行的实例。', 'neutral')}</aside></div>`
}

function telegram() {
  return heading('TG 机器人', '消息，随时可达。', 'Manage / Telegram', badge('运行中')) + `<div class="two-column"><form class="panel form-panel" data-form="telegram">${panelHead('机器人配置', '连接 Telegram，接收通知与远程指令')}<div class="stack"><div class="status-summary">${icons('send')}<div><strong>Tunnel Assistant</strong><p>@homelab_tunnel_bot · 示例机器人</p></div></div>${field('Bot Token', 'bot-token', 'DEMO_TOKEN_NOT_A_SECRET', '仅使用演示占位值，不要在设计稿中填写真实密钥。', 'password', 'required')}${field('Chat ID', 'chat-id', '123456789', '', 'text', 'required')}${toggle('telegram-notify', '接收状态通知', '隧道与服务状态变化时推送消息。')}${toggle('telegram-bound', '仅允许绑定会话', '拒绝未授权会话的指令。')}</div>${saveFooter('保存配置')}</form><aside class="stack"><section class="panel">${panelHead('运行状态')}${kv('连接方式', 'Webhook')}${kv('会话状态', '<span class="accent">已绑定 · 示例</span>')}${kv('最近响应', '2 分钟前 · 示例')}<div class="mt">${button('发送测试消息', 'send', 'telegram-test')}</div></section><section class="panel">${panelHead('安全提示')}<p class="small secondary">Bot Token 是敏感凭据。正式页面需沿用服务端加密存储、掩码显示和会话授权逻辑。</p></section></aside></div>`
}

function notifications() {
  return heading('通知设置', '只在需要你的时候，打扰你。', 'Manage / Notifications', '') + `<div class="two-column"><form class="stack" data-form="notifications"><section class="panel">${panelHead('通知渠道')}<div class="setting-row"><div class="row"><span class="quick-icon">${icons('mail')}</span><div><strong>邮件</strong><p>homelab@tunnel.example</p></div></div>${button('配置', 'edit', 'email-config', 'small-btn')}</div><div class="setting-row"><div class="row"><span class="quick-icon">${icons('send')}</span><div><strong>Telegram</strong><p>@homelab_tunnel_bot</p></div></div>${link('管理', 'telegram')}</div><div class="setting-row"><div class="row"><span class="quick-icon">${icons('code')}</span><div><strong>Webhook</strong><p>推送至你自己的通知服务</p></div></div>${button('配置', 'edit', 'webhook-config', 'small-btn')}</div></section><section class="panel">${panelHead('触发事件')}${toggle('notify-down', '服务不可达', '探测失败时立即通知。')}${toggle('notify-recovery', '服务恢复', '故障恢复后发送一条通知。')}${toggle('notify-latency', '响应延迟异常', '响应时间超过阈值时通知。')}${toggle('notify-tunnel', '隧道状态变化', '连接器上线或离线时通知。')}${saveFooter('保存通知偏好')}</section></form><aside class="stack"><section class="panel">${panelHead('测试通知')}<p class="small secondary">先检查渠道配置，再发送测试消息，确认通知能够正常抵达。</p><div class="mt">${button('发送测试通知', 'bell', 'notification-test')}</div></section>${notice('演示操作不会发送邮件、Telegram 消息或 Webhook 请求。', 'neutral')}</aside></div>`
}

function admin() {
  const tab = state.tab.admin || '用户'
  const people = [['homelab', 'homelab@tunnel.example', '管理员', '全部权限'], ['studio', 'studio@tunnel.example', '成员', '工作室'], ['observer', 'observer@tunnel.example', '只读', '观察者']]
  let content
  if (tab === '用户') content = `<div class="toolbar">${search('搜索用户名或邮箱')}${button('添加用户', 'plus', 'new-user', 'primary')}</div>` + table(['用户', '角色', '用户组', '状态', '操作'], people.map(([name, email, role, group]) => `<tr data-search="${name} ${email}"><td><div class="row"><span class="avatar">${name.slice(0, 2).toUpperCase()}</span><div><strong>${name}</strong><small>${email}</small></div></div></td><td>${badge(role, role === '管理员' ? '' : 'neutral', false)}</td><td>${group}</td><td>${badge('正常')}</td><td>${button('管理', 'edit', 'manage-user', 'small-btn', `data-id="${name}"`)}</td></tr>`).join(''), '3 位示例用户')
  else if (tab === '用户组') content = `<div class="panel">${panelHead('用户组与权限')}${['全部权限', '工作室', '观察者'].map((name, index) => `<div class="setting-row"><div><strong>${name}</strong><p>${['实例管理与所有资源', '指定隧道与 DNS 记录的读写权限', '只读访问，禁止修改配置'][index]}</p></div>${badge(['管理员', '读写', '只读'][index], 'neutral', false)}</div>`).join('')}</div>`
  else if (tab === '安全策略') content = `<form class="panel form-panel" data-form="security">${panelHead('注册与安全')}${toggle('admin-register', '开放注册', '关闭后仅允许管理员创建账户。', false)}${toggle('admin-2fa', '管理员双重验证', '管理账户必须完成第二因素验证。')}${toggle('admin-audit', '审计日志', '记录敏感配置和权限变更。')}${saveFooter()}</form>`
  else content = table(['时间', '操作者', '操作', '结果'], [['14:32:08', 'homelab', '更新 DNS 记录'], ['13:16:42', 'studio', '创建服务监控'], ['09:08:15', 'homelab', '登录控制台']].map(([time, user, action]) => `<tr><td class="number">2026-09-22 ${time}</td><td>${user}</td><td>${action}</td><td>${badge('成功')}</td></tr>`).join(''), '3 条示例审计记录')
  return heading('管理后台', '权限有边界，操作有记录。', 'Manage / Administration', badge('仅管理员', 'neutral', false)) + stats([['注册用户', '03', '示例账户'], ['用户组', '03', '分组管理资源权限'], ['安全策略', '已启用', '保留角色与权限过滤', 'accent']]) + tabs(['用户', '用户组', '安全策略', '审计日志']) + `<div id="tab-content" role="tabpanel">${content}</div>`
}

function account() {
  const tab = state.tab.account || '个人资料'
  const content = tab === '个人资料' ? `<form class="panel form-panel" data-form="profile">${panelHead('个人资料')}<div class="row" style="margin-bottom:24px"><span class="avatar" style="width:52px;height:52px;font-size:17px">HL</span><div><h2>homelab</h2><span class="small muted">个人实例管理员</span></div></div><div class="stack">${field('用户名', 'username', 'homelab', '', 'text', 'required')}${field('邮箱', 'email', 'homelab@tunnel.example', '', 'email', 'required')}</div>${saveFooter()}</form>` : tab === '登录与安全' ? `<div class="stack"><section class="panel">${panelHead('登录方式')}<div class="setting-row"><div><strong>账户密码</strong><p>为当前账户设置独立且安全的密码。</p></div>${button('修改', 'key', 'password', 'small-btn')}</div><div class="setting-row"><div><strong>通行密钥</strong><p>使用设备指纹、面容或安全密钥登录。</p></div>${button('添加', 'plus', 'passkey', 'small-btn')}</div><div class="setting-row"><div><strong>双重验证</strong><p>密码之外的第二道保护。</p></div>${button('查看配置', 'shield', 'two-factor', 'small-btn')}</div></section><section class="panel">${panelHead('会话管理')}<div class="setting-row"><div><strong>当前浏览器</strong><p>Web 设计预览 · 无真实登录会话</p></div>${badge('当前', 'neutral', false)}</div>${button('退出登录演示', 'logout', 'logout', 'danger')}</section></div>` : `<form class="panel form-panel" data-form="cloudflare">${panelHead('Cloudflare 授权', '保留 Token 与账户的原有授权流程')}<div class="stack">${field('API Token', 'cf-token', 'DEMO_TOKEN_NOT_A_SECRET', '不要在设计稿输入真实凭据。', 'password')}${field('Account ID', 'account-id', 'demo-account-id')}${notice('正式页面继续使用已有权限校验与安全存储，不扩大 Token 权限。', 'neutral')}</div>${saveFooter('保存授权演示')}</form>`
  return heading('账户设置', '管理身份，也守护每一次连接。', 'Personal / Account') + tabs(['个人资料', '登录与安全', 'Cloudflare 授权']) + `<div class="two-column"><div id="tab-content" role="tabpanel">${content}</div><aside class="stack"><section class="panel">${panelHead('账户概览')}${kv('账户', 'homelab')}${kv('角色', badge('管理员', '', false))}${kv('邮箱', '<span class="break">homelab@tunnel.example</span>')}${kv('当前环境', '隔离设计预览')}</section></aside></div>`
}

function lab() {
  return heading('IP 优选实验室', '寻找更合适的连接线路。', 'Network / IP laboratory', badge('实验性 · 管理员', 'warn', false)) + `<div class="two-column"><div class="stack"><form class="panel form-panel" data-form="probe">${panelHead('探测配置')}<div class="stack">${selectField('IP 来源', 'source', ['Cloudflare IPv4', 'Cloudflare IPv6'])}<div class="equal-column">${field('并发数', 'concurrency', '16', '', 'number', 'min="1" max="64" required')}${field('超时（秒）', 'timeout', '2', '', 'number', 'min="1" max="30" required')}</div>${notice('这里只演示完成态，不发出探测流量。结果使用保留的示例 IP 地址。', 'neutral')}</div>${saveFooter('开始探测演示')}</form><section class="panel">${panelHead('优选结果', '上次探测的示例结果', badge('已完成', '', false))}${table(['IP 地址', '节点', '延迟', '丢包'], [['192.0.2.18', '香港', '21'], ['192.0.2.28', '香港', '24'], ['198.51.100.8', '东京', '38']].map(([ip, region, latency]) => `<tr><td class="mono">${ip}</td><td>${region}</td><td class="numeric accent">${latency} ms</td><td class="numeric">0%</td></tr>`).join(''), '3 条示例结果')}</section></div><aside class="stack"><section class="panel">${panelHead('自动化与 DNS')}${toggle('lab-auto', '定时优选', '按既有任务配置自动更新线路。', false)}<div class="setting-row"><div><strong>华为云 DNS</strong><p>沿用现有授权与记录更新流程。</p></div>${button('配置', 'settings', 'lab-config', 'small-btn')}</div></section><section class="panel">${panelHead('执行历史')}${kv('最近任务', '2026-09-22 10:00')}${kv('状态', badge('已完成'))}${kv('可用 IP', '23 / 256')}${kv('耗时', '18 秒')}</section></aside></div>`
}

function about() {
  return heading('关于与更新', '让连接更简单，让管理更从容。', 'Personal / About') + `<div class="two-column"><section class="panel form-panel"><div class="row" style="margin-bottom:28px"><span class="brand-mark" style="width:56px;height:56px;border-radius:18px">${icons('tunnel')}</span><div><h2 style="font-size:23px">Tunnel Manager</h2><p class="muted small">Cloudflare Tunnel 管理工具</p></div></div><p class="secondary">隧道、域名、监控与通知，在同一个工作空间里协作。</p><div class="mt">${kv('前端项目版本', '<code>2.6.0</code>')}${kv('版本来源', '<code>frontend/package.json</code>')}${kv('设计稿日期', '2026-09-22')}${kv('当前状态', badge('设计审核中', 'warn', false))}</div><div class="form-footer">${button('检查更新演示', 'refresh', 'update-check')}</div></section><aside class="stack"><section class="panel">${panelHead('本次设计')}${quickItem('book', '设计说明与审核', '查看规范、页面范围与验收项', 'design')}${quickItem('globe', '访客首页', '预览未登录时的产品入口', 'landing')}${quickItem('lock', '登录与安装', '预览身份验证和首次初始化', 'login')}</section></aside></div>`
}

function authShell(content, secondary = '') {
  return `${previewBar()}<div class="auth-shell"><header class="auth-top">${brand()}<div class="row">${secondary}<button class="icon-btn" data-action="theme" aria-label="切换深浅主题">${icons(state.theme === 'dark' ? 'sun' : 'moon')}</button></div></header>${content}<footer class="auth-bottom"><span>Tunnel Manager · 让每一次连接，都更简单。</span><a href="#design">独立设计预览 · 无真实请求 ↗</a></footer></div>`
}

function login() {
  return authShell(`<main class="auth-main" id="main-content"><div class="auth-copy"><span class="eyebrow">Your services. Within reach.</span><h1>连接你的世界，<br><span>从这里开始。</span></h1><p>安全发布本地服务，掌握每一次连接。属于你的轻量工作空间。</p>${connectionFlow()}</div><form class="panel auth-panel" data-form="login"><h2>欢迎回来</h2><p>登录你的 Tunnel Manager 工作空间。</p><div class="stack">${field('服务器地址', 'server', 'https://panel.tunnel.example', '演示地址，无需输入真实服务器。', 'url', 'required')}${field('用户名', 'username', 'homelab', '', 'text', 'autocomplete="off" required')}${field('密码', 'password', '', '请填写任意演示密码，不要输入真实密码。', 'password', 'placeholder="输入演示密码" required minlength="4" autocomplete="off"')}<button class="btn primary full-width" type="submit">登录演示${icons('arrow')}</button><div class="row between"><span class="small muted">或使用无密码登录</span><span class="small muted">安全设备验证</span></div>${button('使用通行密钥', 'key', 'passkey', 'full-width')}${notice('密码、通行密钥与双重验证均为流程演示，不会创建真实会话。', 'neutral')}</div></form></main>`, '<a class="text-link" href="#landing">返回首页</a>')
}

function statusPage() {
  return authShell(`<main class="public-main" id="main-content"><span class="eyebrow">Homelab / Public status</span><h1 style="margin-top:10px">服务运行状态</h1><p class="secondary small mt">透明呈现每一项服务的可用性。</p><section class="public-health">${icons('alert')}<div><h2>部分服务响应延迟偏高</h2><p>API 服务需要关注，其余服务运行正常。</p></div></section><section class="panel">${panelHead('服务可用性', '最近 24 个探测时段 · 示例快照', badge('示例', 'neutral', false))}${state.monitors.map(monitor => `<div class="setting-row"><div class="grow"><div class="row between"><strong>${escape(monitor.name)}</strong>${badge(monitor.status === 'warn' ? '延迟偏高' : '正常', monitor.status === 'warn' ? 'warn' : '')}</div><div class="mt">${serviceBars(monitor.status)}</div><div class="row between small muted" style="margin-top:8px"><span>24 个时段前</span><span>${monitor.uptime}% 可用率</span><span>最近一次</span></div></div></div>`).join('')}</section><section class="mt panel">${panelHead('近期事件')}<div class="event-row"><time>09-22 14:32</time><div><p>API 服务响应延迟升高</p><small>我们正在观察服务响应情况。此为示例事件。</small></div></div></section></main>`, '<a class="text-link" href="#login">管理入口</a>')
}

function landing() {
  return authShell(`<main class="auth-main" id="main-content"><div class="auth-copy"><span class="eyebrow">Private infrastructure, connected.</span><h1>本地的服务，<br><span>世界的入口。</span></h1><p>在一个轻量控制台里，发布服务、管理域名、观察状态。</p><div class="mt"><a class="btn primary" href="#login">进入工作空间${icons('arrow')}</a></div><div class="mt">${link('查看服务状态', 'status')}</div></div><section class="panel form-panel">${panelHead('从连接到安心', '无需在多个工具之间来回切换')}${quickItem('tunnel', '安全隧道', '通过 Cloudflare Tunnel 连接本地服务', 'login')}${quickItem('globe', '域名与解析', '统一管理服务入口和 DNS 记录', 'login')}${quickItem('activity', '持续监控', '了解可用性，及时发现异常', 'status')}${quickItem('bell', '及时通知', '让重要的状态变化主动找到你', 'login')}<div class="mt">${notice('此首页同属审核稿，功能入口使用示例数据。', 'neutral')}</div></section></main>`, '<a class="text-link" href="#login">登录</a>')
}

function setup() {
  return authShell(`<main class="auth-main" id="main-content"><div class="auth-copy"><span class="eyebrow">A space of your own</span><h1>欢迎来到，<br><span>你的工作空间。</span></h1><p>创建首个管理员账户，完成实例初始化。已初始化的正式实例不会显示这个页面。</p><div class="mt">${notice('独立安装页，不显示控制台侧栏。初始化状态判断仍由现有后端负责。', 'neutral')}</div></div><form class="panel auth-panel" data-form="setup"><h2>初始化实例</h2><p>以下信息仅供演示，不会创建真实管理员。</p><div class="stack">${field('站点名称', 'site', 'Tunnel Manager', '', 'text', 'required')}${field('管理员用户名', 'username', 'homelab', '', 'text', 'required')}${field('邮箱', 'email', '', '', 'email', 'placeholder="you@example.com" required')}${field('管理员密码', 'password', '', '演示密码至少 8 位，请勿使用真实密码。', 'password', 'minlength="8" required autocomplete="off"')}<button class="btn primary full-width" type="submit">完成初始化演示${icons('arrow')}</button></div></form></main>`)
}

function design() {
  return heading('同一种语言，更宽的工作空间。', 'Web 第一版设计提案 · 等待审核，不修改正式页面。', 'Design / Review 01', button('回到控制面板', 'arrow', 'goto-dashboard', 'primary')) + `<div class="two-column"><div class="stack"><section class="panel">${panelHead('01 / 视觉基因', '来自 design.md，经网页端适配')}<p class="secondary small">石墨底承载长期管理任务，薄荷绿留给主要动作与正常状态。实体面板承载正文、表格和图表；透光效果只用于移动导航与轻浮层。</p><div class="color-swatches mt">${[['页面', '#0B1211'], ['面板', '#151D1C'], ['嵌套表面', '#1E2625'], ['强调', '#6EDDAD']].map(([label, color]) => `<div><div class="swatch" style="background:${color}"></div><strong class="small">${label}</strong><div class="mono muted small">${color}</div></div>`).join('')}</div></section><section class="panel">${panelHead('02 / 从 App 到 Web')}${kv('宽屏导航', '固定侧栏，按工作空间 / 网络 / 管理分组')}${kv('移动导航', '概览 / 监控 / 新建 / DNS / 更多')}${kv('列表与详情', '桌面表格，详情页明确返回；长值完整展示')}${kv('组件轮廓', '22px 面板 / 14px 输入 / 胶囊按钮')}${kv('信息密度', '28px 页面标题 / 14px 正文 / 等宽技术字段')}${kv('主题与动效', '深浅主题 / 150ms 反馈 / 减少动态效果')}</section><section class="panel">${panelHead('03 / 审核重点')}<div class="stack-sm">${['控制面板信息层级：先连接配置，再监控健康与常用操作。', '隧道和 DNS 列表：桌面密度、长字段、筛选与操作入口。', '域名绑定流程：输入、预览确认、完成反馈和返回路径。', '全局设置与账户：管理权限和个人安全是否清楚区分。', '375px 手机宽度：底栏、更多导航、表单与横向表格。', '顶栏切换状态：空列表、加载、请求失败是否清楚。'].map(text => `<label class="checkbox-line"><input type="checkbox">${text}</label>`).join('')}</div></section></div><aside class="stack"><section class="panel">${panelHead('交付边界')}<p class="small secondary">20 个可切换视图，覆盖现有正式路由和设计说明。创建、修改、删除仅修改内存示例数据；刷新重置。</p><div class="mt">${notice('审核后才会将共同设计令牌和组件接入 Vue / Naive UI，再逐页更新。', 'neutral')}</div></section><section class="panel">${panelHead('页面索引')}${pages.filter(page => page.id !== 'design').map(page => `<a class="quick-item" href="#${page.id}" style="padding:9px 0"><span class="grow small">${page.name}</span>${icons('right')}</a>`).join('')}</section></aside></div>`
}

function emptyState(page, failed = false) {
  const config = { tunnels: ['还没有隧道', '先创建一条隧道，再连接本地服务。', '创建隧道', 'new-tunnel'], monitors: ['还没有监控项目', '添加一个服务地址，开始观察可用性与延迟。', '添加监控', 'new-monitor'], dns: ['这个区域还没有记录', '添加第一条 DNS 记录，让域名有一个明确的去处。', '添加记录', 'new-dns'], dashboard: ['开始你的第一条连接', '创建隧道并配置服务地址后，就可以发布你的服务。', '创建隧道', 'new-tunnel'] }[page] || ['暂无数据', '先创建一个项目。', '返回概览', 'goto-dashboard']
  return `<section class="panel empty-state">${icons(failed ? 'alert' : pages.find(item => item.id === page)?.icon || 'grid')}<h2>${failed ? '暂时无法加载数据' : config[0]}</h2><p>${failed ? '设计稿中的请求失败状态。正式页面保留已有数据与重试入口，不将错误展示为空列表。' : config[1]}</p>${button(failed ? '重试' : config[2], failed ? 'refresh' : 'plus', failed ? 'retry' : config[3], 'primary')}<span class="small muted">${failed ? 'REQUEST_FAILED · 演示错误' : '此为独立空状态演示，不会删除示例数据'}</span></section>`
}

const renderers = { dashboard, tunnels, 'tunnel-detail': tunnelDetail, monitors, 'monitor-detail': monitorDetail, domain: () => domain(false), 'domain-batch': () => domain(true), dns, settings, telegram, notifications, admin, account, lab, about, design }

function render() {
  const pageId = location.hash.slice(1).split('?')[0] || 'dashboard'
  state.page = pages.some(page => page.id === pageId) ? pageId : 'dashboard'
  document.documentElement.dataset.theme = state.theme
  document.title = `${pages.find(page => page.id === state.page).name} · Tunnel Manager 设计预览`
  const dialog = document.querySelector('#dialog')
  if (dialog.open) dialog.close()
  let content
  if (['login', 'status', 'landing', 'setup'].includes(state.page)) content = ({ login, status: statusPage, landing, setup })[state.page]()
  else {
    const hasScenario = ['dashboard', 'tunnels', 'monitors', 'dns'].includes(state.page)
    let pageContent = renderers[state.page]()
    if (hasScenario && state.scenario !== 'normal') {
      pageContent = heading(pages.find(page => page.id === state.page).name, '使用顶栏「状态」切换，检查页面的不同反馈。', `Preview / ${state.scenario}`) + (state.scenario === 'loading' ? `<section class="panel stack" role="status" aria-busy="true"><span class="small secondary">正在加载示例数据…</span><div class="metrics loading"><div class="skeleton"></div><div class="skeleton"></div><div class="skeleton"></div><div class="skeleton"></div></div><div class="skeleton loading" style="height:200px"></div><span class="small muted">此状态保持展示，方便审核。选择「正常」返回。</span></section>` : emptyState(state.page, state.scenario === 'error'))
    }
    content = `${previewBar()}${sidebar()}<div class="app-area">${topbar()}<main class="main-content" id="main-content" tabindex="-1">${pageContent}<footer class="page-footer"><span>所有账户、域名与运行状态均为示例，不代表真实服务。</span><a href="#design">Web 设计稿 01 · 查看设计说明 ↗</a></footer></main></div>${mobileNav()}`
  }
  document.querySelector('#app').innerHTML = content
  restoreDrafts()
}

function rememberDrafts() {
  document.querySelectorAll('#app form[data-form]').forEach(form => {
    state.drafts[form.dataset.form] = Object.fromEntries(new FormData(form))
  })
}

function restoreDrafts() {
  document.querySelectorAll('#app form[data-form]').forEach(form => {
    const draft = state.drafts[form.dataset.form]
    if (!draft) return
    Object.entries(draft).forEach(([name, value]) => {
      const field = form.elements.namedItem(name)
      if (field && field.type !== 'file') field.value = value
    })
  })
}

let toastTimer
function toast(message) {
  const element = document.querySelector('#toast')
  element.textContent = message
  element.classList.add('visible')
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => element.classList.remove('visible'), 3800)
}

function showDialog(title, content) {
  const dialog = document.querySelector('#dialog')
  dialog.innerHTML = `<div class="dialog-head"><h2 id="dialog-title">${title}</h2><button class="icon-btn" data-action="close" aria-label="关闭弹窗">${icons('close')}</button></div>${content}`
  if (!dialog.open) dialog.showModal()
  setTimeout(() => dialog.querySelector('input:not([type="hidden"]), textarea, select')?.focus(), 0)
}

function newTunnel() {
  showDialog('创建隧道', `<p class="dialog-note">创建内存中的示例隧道，不会调用 Cloudflare API。</p><form data-form="new-tunnel"><div class="stack">${field('隧道名称', 'name', '', '为用途起一个容易识别的名字。', 'text', 'required maxlength="80" placeholder="例如：家庭媒体服务"')}${field('本地服务地址', 'service', 'http://localhost:8080', '', 'url', 'required')}</div>${saveFooter('创建示例隧道')}</form>`)
}

function newMonitor() {
  showDialog('添加服务监控', `<p class="dialog-note">只添加到设计稿，不发出任何探测请求。</p><form data-form="new-monitor"><div class="stack">${field('监控名称', 'name', '', '', 'text', 'required maxlength="80" placeholder="例如：个人博客"')}${field('服务地址', 'url', '', '', 'url', 'required placeholder="https://blog.tunnel.example"')}${selectField('探测间隔', 'interval', ['60 秒', '120 秒', '300 秒'])}</div>${saveFooter('添加示例监控')}</form>`)
}

function dnsEditor(id) {
  const record = state.dns.find(item => item.id === id)
  showDialog(record ? '编辑 DNS 记录' : '添加 DNS 记录', `<p class="dialog-note">tunnel.example · 更改只保留在当前设计稿会话。</p><form data-form="dns-edit" data-id="${record?.id || ''}"><div class="stack">${selectField('记录类型', 'type', [record?.type || 'CNAME', ...['CNAME', 'A', 'AAAA', 'TXT', 'MX'].filter(type => type !== (record?.type || 'CNAME'))])}${field('主机名', 'name', record?.name || '', '使用 @ 代表根域名。', 'text', 'required placeholder="app"')}${field('记录值', 'value', record?.value || '', '正式接入后复用现有 DNS 类型校验。', 'text', 'required placeholder="target.example.com"')}${selectField('TTL', 'ttl', record?.ttl === '300 秒' ? ['300 秒', '自动'] : ['自动', '300 秒'])}<label class="checkbox-line"><input type="checkbox" name="proxy" ${record?.proxy !== false ? 'checked' : ''}>开启 Cloudflare 代理（A / AAAA / CNAME）</label></div>${saveFooter(record ? '保存记录' : '添加记录')}</form>`)
}

function confirmDelete(kind, ids) {
  const tunnel = kind === 'tunnel' && state.tunnels.find(item => item.id === ids[0])
  const title = kind === 'tunnel' ? `删除「${escape(tunnel?.name || '隧道')}」？` : `删除 ${ids.length} 条 DNS 记录？`
  showDialog(title, `<p class="secondary small">这里只从当前会话的示例数据中移除。正式操作会影响服务访问，必须保留二次确认。</p><div class="form-footer">${button('取消', '', 'close')}${button('确认删除示例', 'trash', 'confirm-delete', 'danger', `data-kind="${kind}" data-ids="${ids.join(',')}"`)}</div>`)
}

function searchDialog() {
  showDialog('快速前往', `<label class="search-field" style="max-width:none">${icons('search')}<input data-filter="pages" aria-label="搜索页面" placeholder="搜索隧道、DNS、设置…"></label><div class="search-results">${pages.map(page => `<a href="#${page.id}" data-search-page="${page.name.toLowerCase()} ${page.id}">${icons(page.icon)}${page.name}<small>${page.public ? '公共页面' : page.group || '详情与工具'}</small></a>`).join('')}</div>`)
}

function handleAction(action, element) {
  if (action.startsWith('goto-')) {
    location.hash = action.slice(5)
    return
  }
  if (action === 'theme') {
    state.theme = state.theme === 'dark' ? 'light' : 'dark'
    document.documentElement.dataset.theme = state.theme
    document.querySelectorAll('[data-action="theme"]').forEach(button => {
      const icon = button.querySelector('svg')
      if (icon) icon.outerHTML = icons(state.theme === 'dark' ? 'sun' : 'moon')
    })
    return
  }
  if (action === 'menu' || action === 'menu-close') {
    const open = action === 'menu'
    document.querySelector('.sidebar')?.classList.toggle('open', open)
    document.querySelector('.sidebar-backdrop')?.classList.toggle('open', open)
    if (open) document.querySelector('.drawer-close')?.focus()
    else document.querySelector('.mobile-menu-btn')?.focus()
    return
  }
  if (action === 'close') return document.querySelector('#dialog').close()
  if (action === 'search') return searchDialog()
  if (action === 'new-tunnel') return newTunnel()
  if (action === 'new-monitor') return newMonitor()
  if (action === 'new-dns' || action === 'edit-dns') return dnsEditor(element.dataset.id)
  if (action === 'delete-dns') return confirmDelete('dns', [element.dataset.id])
  if (action === 'delete-selected') return confirmDelete('dns', [...state.selectedDns])
  if (action === 'delete-tunnel') return confirmDelete('tunnel', [element.dataset.id])
  if (action === 'confirm-delete') {
    const ids = element.dataset.ids.split(',')
    if (element.dataset.kind === 'tunnel') {
      state.tunnels = state.tunnels.filter(tunnel => !ids.includes(tunnel.id))
      state.routes = state.routes.filter(route => !ids.includes(route.tunnel))
      if (ids.includes(state.currentTunnel)) state.currentTunnel = state.tunnels[0]?.id || ''
      location.hash = 'tunnels'
    } else {
      state.dns = state.dns.filter(record => !ids.includes(record.id))
      state.selectedDns.clear()
    }
    render()
    return toast('已删除示例数据，真实服务未改变。')
  }
  if (action === 'tab') {
    rememberDrafts()
    state.tab[state.page] = element.dataset.tab
    render()
    const selected = [...document.querySelectorAll('[role="tab"]')].find(tab => tab.dataset.tab === state.tab[state.page])
    selected?.focus()
    return
  }
  if (['dns-filter', 'tunnel-filter', 'range'].includes(action)) {
    if (action === 'dns-filter') state.dnsFilter = element.dataset.value
    if (action === 'tunnel-filter') state.tunnelFilter = element.dataset.value
    if (action === 'range') state.range = element.dataset.value
    return render()
  }
  if (action === 'select-tunnel') {
    state.currentTunnel = element.dataset.id
    render()
    return toast('已切换默认隧道，仅影响设计稿。')
  }
  if (action === 'toggle') {
    const enabled = element.getAttribute('aria-checked') !== 'true'
    element.setAttribute('aria-checked', String(enabled))
    state.switches[element.dataset.id] = enabled
    if (element.dataset.id === 'light-effects') document.documentElement.dataset.effects = enabled ? 'light' : 'normal'
    return
  }
  if (action === 'refresh' || action === 'retry' || action === 'probe') {
    rememberDrafts()
    state.scenario = 'normal'
    render()
    return toast(action === 'probe' ? '示例探测完成：HTTP 200，响应 42 ms。未发起真实请求。' : '已展示最新示例快照，未请求真实服务。')
  }
  if (action === 'chart-point') {
    return showDialog(`${element.dataset.label} · 探测明细`, `<p class="dialog-note">固定示例聚合时段，峰值与平均值分别展示。</p>${kv('平均延迟', `${element.dataset.average} ms`)}${kv('峰值延迟', `${element.dataset.peak} ms`)}${kv('状态', badge(element.dataset.peak === '386' ? 'API 服务出现异常' : '正常', element.dataset.peak === '386' ? 'warn' : ''))}<div class="mt">${link('查看监控项目', 'monitors')}</div>`)
  }
  if (action === 'copy') {
    if (navigator.clipboard) navigator.clipboard.writeText(element.dataset.value).then(() => toast('已复制示例 ID。')).catch(() => showDialog('复制示例内容', `<code class="break">${escape(element.dataset.value)}</code><p class="small muted mt">浏览器未授予剪贴板权限，可选中文本复制。</p>`))
    else showDialog('复制示例内容', `<code class="break">${escape(element.dataset.value)}</code><p class="small muted mt">可选中文本复制。</p>`)
    return
  }
  if (action === 'token') return showDialog('示例连接器令牌', `${notice('这不是有效令牌，仅演示敏感内容的显示方式。')}<div class="panel mt"><code>DEMO_TUNNEL_TOKEN_NOT_VALID</code></div><div class="form-footer">${button('复制示例值', 'copy', 'copy', '', 'data-value="DEMO_TUNNEL_TOKEN_NOT_VALID"')}</div>`)
  if (action === 'create') return showDialog('新建', `<div class="stack">${button('创建隧道', 'tunnel', 'new-tunnel')}${button('添加监控', 'activity', 'new-monitor')}<a class="btn" href="#domain">${icons('link')}绑定域名</a>${button('添加 DNS 记录', 'dns', 'new-dns')}</div>`)
  if (action === 'workspace') return showDialog('当前工作空间', `<div class="status-summary">${icons('server')}<div><strong>Homelab 工作空间</strong><p>个人实例 · 管理员视角 · 示例</p></div></div><p class="small secondary">此选择器用于交代实例上下文，本提案不引入多租户或多工作空间能力。</p><div class="mt">${link('管理账户与授权', 'account')}</div>`)
  if (action === 'zone') return showDialog('选择 Cloudflare 区域', `<p class="dialog-note">当前只有一个示例区域，不读取真实账户。</p>${button('tunnel.example · 当前区域', 'check', 'close', 'full-width')}`)
  if (action === 'logout') { location.hash = 'login'; return toast('已返回登录设计稿，无真实会话被注销。') }
  if (action === 'passkey') return showDialog('通行密钥验证', `<div class="status-summary">${icons('key')}<div><strong>设备验证入口</strong><p>不调用 WebAuthn、不读取生物识别数据。</p></div></div><p class="small secondary">正式接入保留平台支持检测、取消操作与验证失败反馈。</p><div class="form-footer">${button('取消', '', 'close')}${button('模拟验证通过', 'check', 'goto-dashboard', 'primary')}</div>`)
  if (action === 'two-factor') return showDialog('双重验证', `<p class="dialog-note">演示第二因素输入。任意六位数字均可通过，不保存恢复代码。</p><form data-form="two-factor">${field('六位验证码', 'code', '', '', 'text', 'required pattern="[0-9]{6}" inputmode="numeric" maxlength="6" autocomplete="off" placeholder="000000"')}${saveFooter('验证演示')}</form>`)
  if (action === 'password') return showDialog('修改密码', `<p class="dialog-note">不要输入真实密码。提交仅验证两次演示输入一致。</p><form data-form="password"><div class="stack">${field('新密码', 'password', '', '至少 8 位。', 'password', 'required minlength="8" autocomplete="off"')}${field('确认密码', 'confirmation', '', '', 'password', 'required minlength="8" autocomplete="off"')}</div>${saveFooter('保存演示密码')}</form>`)
  if (action === 'email-config') return showDialog('邮件通知', `<form data-form="email-config"><div class="stack">${field('收件邮箱', 'email', 'homelab@tunnel.example', '只演示配置，不发送邮件。', 'email', 'required')}</div>${saveFooter()}</form>`)
  if (action === 'webhook-config') return showDialog('Webhook 配置', `<form data-form="webhook-config"><div class="stack">${field('Webhook 地址', 'url', 'https://hooks.tunnel.example/notify', '不会发起网络请求。', 'url', 'required')}</div>${saveFooter()}</form>`)
  if (action === 'new-user' || action === 'manage-user') return showDialog(action === 'new-user' ? '添加示例用户' : '管理用户', `<form data-form="user"><div class="stack">${field('用户名', 'username', element.dataset.id || '', '', 'text', 'required')}${selectField('角色', 'role', ['成员', '只读', '管理员'])}${notice('此处仅演示权限编辑，不改变真实用户或授权。', 'neutral')}</div>${saveFooter('确认演示')}</form>`)
  if (action === 'lab-config') return showDialog('自动化与华为云 DNS', `<form data-form="lab-config"><div class="stack">${field('记录名称', 'record', 'edge.tunnel.example')}${selectField('执行周期', 'schedule', ['每 6 小时', '每天'])}${notice('正式接入使用已有授权配置；设计稿不收集云账户密钥。', 'neutral')}</div>${saveFooter()}</form>`)
  if (action === 'telegram-test' || action === 'notification-test') return showDialog('测试通知已模拟', `<div class="status-summary">${icons('circlecheck')}<div><strong>发送结果反馈示例</strong><p>没有向任何真实渠道发送消息。</p></div></div><p class="secondary small">正式页面会区分渠道未配置、发送成功与服务端错误，不以统一成功提示掩盖失败。</p>`)
  if (action === 'update-check') return showDialog('更新检查演示', `${kv('当前前端版本', '2.6.0')}${kv('检查状态', '未请求远程版本源')}<p class="small secondary mt">设计稿不能判断是否存在更新。正式页面继续显示真实版本与检查结果。</p>`)
  if (action === 'confirm-bind') {
    const pending = state.pendingBindings || []
    state.routes.push(...pending)
    pending.forEach(route => {
      const tunnel = state.tunnels.find(item => item.id === route.tunnel)
      if (tunnel) tunnel.routes += 1
    })
    state.pendingBindings = null
    return showDialog('示例绑定已完成', `<div class="status-summary">${icons('circlecheck')}<div><strong>${pending.length} 条路由已加入设计稿</strong><p>仅更新内存示例，不会解析到真实服务。</p></div></div><div class="stack-sm">${pending.map(route => `<code>${escape(route.host)}</code>`).join('')}</div><div class="form-footer"><a class="btn primary" href="#tunnel-detail">查看隧道详情${icons('arrow')}</a></div>`)
  }
}

document.addEventListener('click', event => {
  const anchor = event.target.closest('a[href^="#"]')
  if (anchor) {
    if (anchor.getAttribute('href') === '#main-content') return
    rememberDrafts()
    if (anchor.dataset.tunnel) state.tunnelId = anchor.dataset.tunnel
    if (anchor.dataset.monitor) state.monitorId = anchor.dataset.monitor
    if (anchor.hash === location.hash) {
      event.preventDefault()
      render()
    }
  }
  const action = event.target.closest('[data-action]')
  if (action) handleAction(action.dataset.action, action)
})

document.addEventListener('input', event => {
  if (event.target.matches('input')) event.target.setCustomValidity('')
  const kind = event.target.dataset.filter
  if (!kind) return
  const query = event.target.value.trim().toLowerCase()
  const items = document.querySelectorAll(kind === 'pages' ? '[data-search-page]' : 'tr[data-search]')
  let count = 0
  items.forEach(item => {
    const matches = (kind === 'pages' ? item.dataset.searchPage : item.dataset.search).includes(query)
    item.hidden = !matches
    if (matches) count += 1
  })
  if (kind === 'list') {
    const counter = document.querySelector('[data-count]')
    if (counter) counter.textContent = count ? `显示 ${count} 条匹配结果` : '没有匹配的记录，请换个关键词。'
  }
})

document.addEventListener('change', event => {
  const element = event.target
  if (element.dataset.control === 'page') { rememberDrafts(); state.scenario = 'normal'; location.hash = element.value }
  if (element.dataset.control === 'scenario') { state.scenario = element.value; render() }
  if (element.dataset.dnsSelect) {
    if (element.dataset.dnsSelect === 'all') {
      document.querySelectorAll('tr[data-search]:not([hidden]) input[data-dns-select]').forEach(input => {
        input.checked = element.checked
        if (element.checked) state.selectedDns.add(input.dataset.dnsSelect)
        else state.selectedDns.delete(input.dataset.dnsSelect)
      })
    } else if (element.checked) state.selectedDns.add(element.dataset.dnsSelect)
    else state.selectedDns.delete(element.dataset.dnsSelect)
    const selection = document.querySelector('#dns-selection')
    selection.hidden = state.selectedDns.size === 0
    selection.querySelector('strong').textContent = state.selectedDns.size
    const checkboxes = [...document.querySelectorAll('tr[data-search]:not([hidden]) input[data-dns-select]')]
    const all = document.querySelector('[data-dns-select="all"]')
    all.checked = checkboxes.length > 0 && checkboxes.every(input => input.checked)
    all.indeterminate = checkboxes.some(input => input.checked) && !all.checked
  }
})

function validateProtocol(input) {
  if (!/^https?:\/\//i.test(input.value)) {
    input.setCustomValidity('请填写以 http:// 或 https:// 开头的地址。')
    input.reportValidity()
    return false
  }
  return true
}

document.addEventListener('submit', event => {
  const form = event.target
  if (!form.dataset.form) return
  event.preventDefault()
  const values = Object.fromEntries(new FormData(form))
  const type = form.dataset.form
  if (['new-monitor', 'monitor-settings', 'webhook-config'].includes(type) && !validateProtocol(form.elements.url)) return
  if (['new-tunnel', 'bind'].includes(type) && !validateProtocol(form.elements.service)) return
  if (type === 'new-tunnel') {
    const id = `demo-${Date.now()}`
    state.tunnels.push({ id, name: values.name.trim(), status: 'offline', uuid: `demo-${state.tunnels.length + 1}-not-a-real-tunnel`, routes: 0, created: '2026-09-22', edge: '未连接', service: values.service })
    state.scenario = 'normal'
    location.hash = 'tunnels'
    render()
    return toast('示例隧道已创建，等待连接器上线。')
  }
  if (type === 'new-monitor') {
    state.monitors.push({ id: `demo-${Date.now()}`, name: values.name.trim(), host: new URL(values.url).host, latency: 42, uptime: '100.00', status: 'online' })
    state.scenario = 'normal'
    location.hash = 'monitors'
    render()
    return toast('已添加示例监控，数据不是实际探测结果。')
  }
  if (type === 'dns-edit') {
    const existing = state.dns.find(record => record.id === form.dataset.id)
    const record = { id: existing?.id || `dns-${Date.now()}`, type: values.type, name: values.name.trim(), value: values.value.trim(), ttl: values.ttl, proxy: values.proxy === 'on' && ['A', 'AAAA', 'CNAME'].includes(values.type) }
    if (existing) Object.assign(existing, record)
    else state.dns.push(record)
    state.scenario = 'normal'
    state.dnsFilter = '全部'
    location.hash = 'dns'
    render()
    return toast('DNS 示例记录已保存，真实解析未改变。')
  }
  if (type === 'bind' || type === 'batch-bind') {
    rememberDrafts()
    let entries = [[values.hostname, values.service]]
    if (type === 'batch-bind') entries = values.entries.trim().split('\n').map(line => line.split(',').map(value => value.trim()))
    const valid = entries.every(([hostname, service, extra]) => {
      if (extra || !hostname || !service || !/^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$/.test(hostname)) return false
      try { return ['http:', 'https:'].includes(new URL(service).protocol) } catch { return false }
    })
    if (!valid || !values.tunnel) {
      const input = type === 'batch-bind' ? form.elements.entries : form.elements.hostname
      input.setCustomValidity('请检查每行格式、子域名、HTTP(S) 服务地址，并选择可用隧道。')
      input.reportValidity()
      input.addEventListener('input', () => input.setCustomValidity(''), { once: true })
      return
    }
    state.tunnelId = values.tunnel
    state.pendingBindings = entries.map(([hostname, service]) => ({ host: `${hostname}.${values.zone}`, service, tunnel: values.tunnel }))
    return showDialog('确认绑定配置', `<p class="dialog-note">请核对以下示例路由。确认后只更新设计稿内存。</p><div class="stack-sm">${state.pendingBindings.map(route => `<div class="panel">${kv('域名', `<code>${escape(route.host)}</code>`)}${kv('源站', `<code>${escape(route.service)}</code>`)}</div>`).join('')}</div><div class="form-footer">${button('返回修改', 'back', 'close')}${button('确认示例绑定', 'check', 'confirm-bind', 'primary')}</div>`)
  }
  if (type === 'password' && values.password !== values.confirmation) {
    form.elements.confirmation.setCustomValidity('两次输入的密码不一致。')
    form.elements.confirmation.reportValidity()
    return
  }
  if (type === 'login') return handleAction('two-factor', form)
  if (type === 'two-factor') { location.hash = 'dashboard'; render(); return toast('验证流程演示完成，未创建真实会话。') }
  if (type === 'setup') { location.hash = 'login'; return toast('初始化演示完成，未创建真实账户。') }
  if (type === 'probe') return showDialog('示例探测完成', `<div class="status-summary">${icons('circlecheck')}<div><strong>找到 23 个可用 IP</strong><p>256 个示例地址 · 模拟耗时 18 秒</p></div></div><p class="small muted">页面结果为固定示例，没有产生真实探测流量。</p>`)
  rememberDrafts()
  if (type === 'monitor-settings') {
    const monitor = state.monitors.find(item => item.id === state.monitorId)
    if (monitor) Object.assign(monitor, { name: values.name, host: new URL(values.url).host })
  }
  document.querySelector('#dialog').close()
  toast('配置已在设计稿中确认，真实数据未改变。')
})

document.addEventListener('keydown', event => {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') { event.preventDefault(); searchDialog() }
  if (event.key === 'Escape') handleAction('menu-close', event.target)
  if (event.target.getAttribute('role') === 'tab' && ['ArrowLeft', 'ArrowRight'].includes(event.key)) {
    event.preventDefault()
    const choices = [...document.querySelectorAll('[role="tab"]')]
    const index = choices.indexOf(event.target)
    const next = choices[(index + (event.key === 'ArrowRight' ? 1 : -1) + choices.length) % choices.length]
    handleAction('tab', next)
  }
})

window.addEventListener('hashchange', () => {
  render()
  window.scrollTo(0, 0)
  document.querySelector('#main-content')?.focus({ preventScroll: true })
})

render()
