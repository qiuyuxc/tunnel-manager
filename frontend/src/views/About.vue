<template>
  <div class="page-container">
    <div class="page-header about-heading">
      <router-link to="/" class="back-link">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
        返回控制面板
      </router-link>
      <h2>关于</h2>
      <p>版本信息、项目仓库与更新动态。</p>
    </div>

    <div class="settings-grid section">
      <section class="settings-card settings-card-wide card-transition">
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

      <section class="settings-card settings-card-wide card-transition">
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

        <template v-if="latestBody">
          <div class="release-title">最新发布更新内容（{{ latestTag }}）</div>
          <div class="release-body">{{ latestBody }}</div>
        </template>
        <template v-else-if="checkError">
          <div class="release-title">更新检查</div>
          <div class="release-body release-error">{{ checkError }}</div>
        </template>

        <div class="changelog-note">
          <div class="changelog-note-title">本版本亮点（{{ currentVersion }}）</div>
          <ul>
            <li>在线新建与删除隧道：创建后直接给出连接令牌、cloudflared 运行命令与 Debian / Ubuntu 安装脚本</li>
            <li>支持删除应用程序路由，确认时可勾选一并清理该主机名对应的 DNS 记录</li>
            <li>DNS 批量删除：多选后一次性删除并实时显示进度，失败项自动保留供重试</li>
            <li>DNS 操作按钮补上边框，编辑与删除入口更易辨认</li>
          </ul>
        </div>
      </section>

      <section class="settings-card settings-card-wide card-transition">
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

      <section class="settings-card settings-card-wide card-transition">
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

const REPO_URL = 'https://github.com/qiuyuxc/tunnel-manager'
const RELEASE_API = 'https://api.github.com/repos/qiuyuxc/tunnel-manager/releases/latest'

const currentVersion = ref('—')
const latestTag = ref('')
const latestVersion = ref('')
const latestBody = ref('')
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
    latestBody.value = (data.body || '').slice(0, 2000)
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
.about-heading p { max-width: 560px; }

.about-app {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}

.about-logo {
  flex-shrink: 0;
  width: 56px;
  height: 56px;
  border-radius: var(--radius-lg);
  background: var(--color-ink);
  color: var(--color-canvas);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--color-canvas) 12%, transparent);
}

.about-app-text strong { font-size: 15px; color: var(--color-ink); }

.about-app-text p {
  margin: 6px 0 10px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--color-body);
  max-width: 640px;
}

.about-repo {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--color-accent, var(--color-ink));
  text-decoration: none;
  word-break: break-all;
}

.about-repo:hover { text-decoration: underline; }

.version-table {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 4px;
}

.version-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 0;
  border-bottom: 1px solid var(--color-hairline);
  font-size: 13px;
}

.version-row span { color: var(--color-mute); }

.version-row strong {
  font-family: var(--font-mono);
  font-size: 13px;
  color: var(--color-ink);
}

.status-ok { color: var(--color-success, #1f8a5b) !important; }
.status-new { color: var(--color-accent, #b45309) !important; }
.status-ahead { color: var(--color-mute) !important; }

.release-title {
  margin-top: 16px;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-ink);
}

.release-body {
  margin-top: 8px;
  padding: 12px 14px;
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  background: var(--color-canvas-soft);
  font-size: 12.5px;
  line-height: 1.75;
  color: var(--color-body);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 320px;
  overflow-y: auto;
}

.release-error { color: var(--color-error); }

.changelog-note {
  margin-top: 18px;
  padding-top: 14px;
  border-top: 1px dashed var(--color-hairline);
}

.changelog-note-title { font-size: 13px; font-weight: 600; color: var(--color-ink); }

.changelog-note ul {
  margin: 8px 0 0;
  padding-left: 18px;
  font-size: 12.5px;
  line-height: 1.9;
  color: var(--color-body);
}

.feature-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 12px;
  margin-top: 4px;
}

.feature-item {
  padding: 14px;
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  background: var(--color-canvas-soft);
}

.feature-item strong { font-size: 13.5px; color: var(--color-ink); }

.feature-item p {
  margin: 6px 0 0;
  font-size: 12.5px;
  line-height: 1.7;
  color: var(--color-body);
}

.stack-row {
  display: flex;
  align-items: baseline;
  gap: 12px;
  padding: 9px 0;
  border-bottom: 1px solid var(--color-hairline);
  font-size: 13px;
}

.stack-row:last-child { border-bottom: none; }

.stack-row span { color: var(--color-mute); flex-shrink: 0; width: 44px; }

.stack-row code {
  font-family: var(--font-mono);
  font-size: 12.5px;
  color: var(--color-body);
  word-break: break-all;
}

@media (max-width: 600px) {
  .about-app { flex-direction: column; }
}
</style>
