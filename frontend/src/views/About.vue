<template>
  <div class="page-container">
    <div class="page-header about-heading">
      <h2>关于</h2>
      <p>版本信息、项目仓库与更新动态。</p>
    </div>
    <div class="settings-grid section">
      <section class="settings-card settings-card-wide">
        <div class="settings-card-header">
          <div>
            <div class="settings-card-title">Tunnel Manager</div>
            <div class="settings-card-desc">Cloudflare Tunnel 可视化管理面板</div>
          </div>
        </div>
        <div class="about-app">
          <span class="about-logo" aria-hidden="true">
            <svg width="34" height="34" viewBox="0 0 76 76" fill="none">
              <path d="M49 26H27v24l22-24z" fill="currentColor"/>
              <path d="M38 38L27 50h22L38 38z" fill="currentColor" fill-opacity="0.42"/>
            </svg>
          </span>
          <div class="about-app-text">
            <strong>Cloudflare Tunnel 可视化管理面板</strong>
            <p>通过 Web UI 管理隧道、绑定域名、配置 DNS 优选与回退源，并提供 Telegram Bot 和管理员双重身份验证。</p>
            <a class="about-repo" :href="REPO_URL" target="_blank" rel="noopener">
              仓库地址：{{ REPO_URL }}
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
            </a>
          </div>
        </div>
      </section>
      <section class="settings-card settings-card-wide">
        <div class="settings-card-header">
          <div>
            <div class="settings-card-title">版本与更新</div>
            <div class="settings-card-desc">对比 GitHub 上发布的最新版本，查看是否有新内容。</div>
          </div>
          <button class="btn btn-primary" :disabled="checking" @click="checkUpdate">
            {{ checking ? '检查中...' : '检查更新' }}
          </button>
        </div>
        <div class="version-table">
          <div class="version-row">
            <span>当前版本</span>
            <strong>{{ currentVersion }}</strong>
          </div>
          <div class="version-row">
            <span>线上最新</span>
            <strong>{{ latestVersion || '—' }}</strong>
          </div>
          <div class="version-row">
            <span>更新状态</span>
            <strong :class="statusClass">{{ statusText }}</strong>
          </div>
        </div>
      </section>
      <section class="settings-card settings-card-wide changelog-card">
        <div class="settings-card-header">
          <div>
            <div class="settings-card-title">本版本亮点</div>
            <div class="settings-card-desc">当前版本（{{ currentVersion }}）的重点变化。</div>
          </div>
        </div>
        <ul class="changelog-list">
          <li>新增服务监控模块：HTTP / TCP / ICMP 三种探测方式，可配置检测间隔，支持 GET / POST 探测请求</li>
          <li>公开状态页上线：自定义短路径（/status/你起的名字）、标题、公告与顶部品牌图标（支持上传动图）</li>
          <li>公开状态页主题与主面板完全隔离，访客可在科技蓝 / 暖米金色系内独立切换明暗</li>
          <li>监控目标支持二次编辑与外链跳转开关，公开页服务卡可直接点击跳转</li>
          <li>站点图标支持自定义上传，浏览器标签页同步展示</li>
        </ul>
      </section>
      <section v-if="latestBody || checkError" class="settings-card settings-card-wide release-card">
        <div class="settings-card-header">
          <div>
            <div class="settings-card-title">最新发布更新内容</div>
            <div class="settings-card-desc">来自 GitHub Release{{ latestTag ? ` · ${latestTag}` : '' }}。</div>
          </div>
        </div>
        <div v-if="latestBody" class="release-body release-markdown" v-html="latestBodyHtml"></div>
        <div v-else class="release-body release-error">{{ checkError }}</div>
      </section>
      <section class="settings-card settings-card-wide">
        <div class="settings-card-header">
          <div>
            <div class="settings-card-title">功能特性</div>
            <div class="settings-card-desc">主要能力一览。</div>
          </div>
        </div>
        <div class="feature-grid">
          <div class="feature-item">
            <strong>隧道管理</strong>
            <p>新建、删除、列出与选择 Cloudflare Tunnel，增删改应用路由（Ingress）并可连带清理 DNS</p>
          </div>
          <div class="feature-item">
            <strong>域名绑定</strong>
            <p>简化直连 / 优选模式，支持批量绑定，自动配置 Tunnel 路由与 DNS</p>
          </div>
          <div class="feature-item">
            <strong>DNS 管理</strong>
            <p>按 Zone 增删改查 A / AAAA / CNAME / TXT / MX 记录，支持多选批量修改与批量删除</p>
          </div>
          <div class="feature-item">
            <strong>Telegram Bot</strong>
            <p>在手机上远程管理隧道、域名与 DNS，支持长轮询与 Webhook</p>
          </div>
          <div class="feature-item">
            <strong>Cloudflare OAuth</strong>
            <p>授权连接 Cloudflare 账户，免去手动复制 API Token</p>
          </div>
          <div class="feature-item">
            <strong>安全认证</strong>
            <p>Argon2id 密码哈希与 TOTP 双重验证，恢复码防丢失</p>
          </div>
        </div>
      </section>
      <section class="settings-card settings-card-wide">
        <div class="settings-card-header">
          <div>
            <div class="settings-card-title">技术栈</div>
            <div class="settings-card-desc">构建本项目使用的技术。</div>
          </div>
        </div>
        <div class="stack-row"><span>前端</span><code>Vue 3 · TypeScript · Naive UI · Vite · Pinia</code></div>
        <div class="stack-row"><span>后端</span><code>Go · chi · JSON 文件存储</code></div>
        <div class="stack-row"><span>集成</span><code>Cloudflare API · Telegram Bot API · GitHub Actions</code></div>
      </section>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import MarkdownIt from 'markdown-it'

const REPO_URL = 'https://github.com/qiuyuxc/tunnel-manager'
const RELEASE_API = 'https://api.github.com/repos/qiuyuxc/tunnel-manager/releases/latest'
const markdown = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
})
markdown.renderer.rules.link_open = (tokens, idx, options, _env, self) => {
  const token = tokens[idx]
  token.attrSet('target', '_blank')
  token.attrSet('rel', 'noopener noreferrer')
  return self.renderToken(tokens, idx, options)
}

const currentVersion = ref('—')
const latestTag = ref('')
const latestVersion = ref('')
const latestBody = ref('')
const latestBodyHtml = computed(() => markdown.render(latestBody.value))
const checking = ref(false)
const checkError = ref('')
function parseVersion(tag: string): number[] {
  const m = tag.match(/v?(\d+)\.(\d+)\.(\d+)/)
  if (!m) return []
  return [Number(m[1]), Number(m[2]), Number(m[3])]
}
function compareVersion(a: number[], b: number[]): number {
  for (let i = 0; i < Math.max(a.length, b.length); i++) {
    const x = a[i] ?? 0
    const y = b[i] ?? 0
    if (x !== y) return x < y ? -1 : 1
  }
  return 0
}
const statusText = computed(() => {
  if (checking.value) return '检查中...'
  if (!latestVersion.value) return checkError.value ? '检查失败' : '尚未检查'
  const c = parseVersion(currentVersion.value)
  const l = parseVersion(latestVersion.value)
  if (c.length && l.length) {
    const diff = compareVersion(l, c)
    if (diff > 0) return '发现新版本 ' + latestVersion.value
    if (diff === 0) return '已是最新版本'
    return '当前版本领先线上发布'
  }
  return '—'
})
const statusClass = computed(() => {
  if (!latestVersion.value) return ''
  const c = parseVersion(currentVersion.value)
  const l = parseVersion(latestVersion.value)
  if (c.length && l.length) {
    const diff = compareVersion(l, c)
    if (diff > 0) return 'status-new'
    if (diff === 0) return 'status-ok'
  }
  return 'status-ahead'
})
async function checkUpdate() {
  checking.value = true
  checkError.value = ''
  latestBody.value = ''
  try {
    const res = await fetch(RELEASE_API, { headers: { Accept: 'application/vnd.github+json' } })
    if (!res.ok) throw new Error('GitHub API ' + res.status)
    const data = await res.json()
    latestTag.value = data.tag_name || ''
    latestVersion.value = data.tag_name || ''
    latestBody.value = (data.body || '').slice(0, 20_000)
  } catch (_e) {
    checkError.value = '无法连接 GitHub，请检查网络后重试。'
  } finally {
    checking.value = false
  }
}
onMounted(async () => {
  try {
    const res = await fetch('/api/health')
    if (res.ok) {
      const data = await res.json()
      currentVersion.value = data.version || '—'
    }
  } catch (_e) { /* ignore */ }
  checkUpdate()
})
</script>
<style scoped>
.page-header { margin-bottom: var(--spacing-lg); }
.section { margin-bottom: var(--spacing-xl); }

.settings-grid {
  display: grid;
  gap: var(--spacing-lg);
  max-width: 800px;
}

.settings-card {
  background: var(--color-canvas-raised);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  padding: var(--spacing-xl);
}

.settings-card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--spacing-lg);
  margin-bottom: var(--spacing-lg);
}

.settings-card-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-ink);
  margin: 0 0 4px;
}

.settings-card-desc {
  font-size: 14px;
  color: var(--color-mute);
  line-height: 1.6;
}

.about-app {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-lg);
  padding: var(--spacing-lg);
  background: var(--color-canvas-soft);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
}

.about-logo {
  display: inline-flex;
  width: 52px;
  height: 52px;
  flex: none;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-lg);
  background: var(--color-ink);
  color: #fff;
}

.about-app-text { min-width: 0; }

.about-app-text strong {
  display: block;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-ink);
  margin-bottom: 8px;
}

.about-app-text p {
  font-size: 14px;
  color: var(--color-body);
  line-height: 1.6;
  margin: 0 0 12px;
}

.about-repo {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: var(--color-link);
  text-decoration: none;
  font-weight: 500;
}

.about-repo:hover { color: var(--color-link-hover); text-decoration: underline; }

.version-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-md);
  padding: 10px 0;
  border-bottom: 1px solid var(--color-hairline);
}

.version-row:last-child { border-bottom: none; }

.version-row span {
  color: var(--color-mute);
  font-weight: 500;
}

.version-row strong {
  color: var(--color-ink);
  font-weight: 600;
  text-align: right;
}

.version-row .status-new { color: var(--color-warning); }
.version-row .status-ok { color: var(--color-success); }
.version-row .status-ahead { color: var(--color-link); }

.release-body {
  padding: var(--spacing-lg);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  background: var(--color-canvas-soft);
  color: var(--color-body);
  font-size: 14px;
  line-height: 1.7;
  overflow-wrap: anywhere;
}

.release-body.release-error { color: var(--color-error); }

.changelog-list {
  display: grid;
  gap: 10px;
  margin: 0;
  padding-left: 1.35em;
  color: var(--color-body);
  font-size: 14px;
  line-height: 1.65;
}

.changelog-list li::marker { color: var(--color-mute); }

.release-markdown :deep(:first-child) { margin-top: 0; }
.release-markdown :deep(:last-child) { margin-bottom: 0; }
.release-markdown :deep(h1),
.release-markdown :deep(h2),
.release-markdown :deep(h3),
.release-markdown :deep(h4) {
  margin: 1.15em 0 0.5em;
  color: var(--color-ink);
  line-height: 1.35;
}

.release-markdown :deep(h1) { font-size: 20px; }
.release-markdown :deep(h2) { font-size: 18px; }
.release-markdown :deep(h3) { font-size: 16px; }
.release-markdown :deep(h4) { font-size: 14px; }
.release-markdown :deep(p) { margin: 0 0 0.8em; }
.release-markdown :deep(ul),
.release-markdown :deep(ol) {
  margin: 0 0 0.8em;
  padding-left: 1.5em;
}

.release-markdown :deep(li + li) { margin-top: 0.3em; }
.release-markdown :deep(blockquote) {
  margin: 0 0 0.8em;
  padding: 0.15em 0 0.15em 1em;
  border-left: 3px solid var(--color-hairline);
  color: var(--color-mute);
}

.release-markdown :deep(code) {
  padding: 0.12em 0.35em;
  border-radius: 4px;
  background: var(--color-canvas-soft-2);
  font-family: var(--font-mono);
  font-size: 0.9em;
}

.release-markdown :deep(pre) {
  margin: 0 0 0.8em;
  padding: 12px 14px;
  overflow-x: auto;
  border-radius: 6px;
  background: var(--color-canvas-soft-2);
}

.release-markdown :deep(pre code) {
  padding: 0;
  background: transparent;
  font-size: 13px;
}

.release-markdown :deep(a) {
  color: var(--color-link);
  text-decoration: underline;
  text-underline-offset: 2px;
}

.release-markdown :deep(hr) {
  margin: 1em 0;
  border: 0;
  border-top: 1px solid var(--color-hairline);
}

@media (max-width: 640px) {
  .about-app { flex-direction: column; align-items: flex-start; }
  .settings-card-header { flex-direction: column; }
  .version-row { flex-direction: column; align-items: flex-start; gap: 4px; }
  .version-row strong { text-align: left; }
  .about-repo { word-break: break-all; }
  .release-body { padding: 12px; }
}
</style>
