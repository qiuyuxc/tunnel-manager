<template>
  <div class="page-container">
    <div class="page-header about-heading">
      <h2>{{ tr('m_52d25a9e30ba') }}</h2>
      <p>{{ tr('m_5ef9a1754675') }}</p>
    </div>
    <div class="settings-grid section">
      <section class="settings-card settings-card-wide">
        <div class="settings-card-header">
          <div>
            <div class="settings-card-title">Tunnel Manager</div>
            <div class="settings-card-desc">{{ tr('m_7d603117f21f') }}</div>
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
            <strong>{{ tr('m_7d603117f21f') }}</strong>
            <p>{{ tr('m_b8e95118d3f7') }}</p>
            <a class="about-repo" :href="REPO_URL" target="_blank" rel="noopener">
              {{ tr('m_074aab6d6d27') }}{{ REPO_URL }}
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
            </a>
            <a class="about-repo" href="https://docs.kukie.cn" target="_blank" rel="noopener">
              {{ tr('m_127285f0135b') }}
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
            </a>
          </div>
        </div>
      </section>
      <section class="settings-card settings-card-wide">
        <div class="settings-card-header">
          <div>
            <div class="settings-card-title">{{ tr('m_cebbc1f61ca0') }}</div>
            <div class="settings-card-desc">{{ tr('m_dfe1c1204ded') }}</div>
          </div>
          <button class="btn btn-primary" :disabled="checking" @click="checkUpdate">
            {{ checking ? tr('m_2a114c7e95b7') : tr('m_7f68ebad19ba') }}
          </button>
        </div>
        <div class="version-table">
          <div class="version-row">
            <span>{{ tr('m_4e87ee56bf2b') }}</span>
            <strong>{{ currentVersion }}</strong>
          </div>
          <div class="version-row">
            <span>{{ tr('m_7f364f52a105') }}</span>
            <strong>{{ latestVersion || '—' }}</strong>
          </div>
          <div class="version-row">
            <span>{{ tr('m_57106b8643ca') }}</span>
            <strong :class="statusClass">{{ statusText }}</strong>
          </div>
        </div>
      </section>
      <section class="settings-card settings-card-wide changelog-card">
        <div class="settings-card-header">
          <div>
            <div class="settings-card-title">{{ tr('m_42c169ac53c2') }}</div>
            <div class="settings-card-desc">{{ tr('m_6808369b92b1') }}</div>
          </div>
        </div>
        <ul class="changelog-list">
          <li>{{ tr('m_1ae15c6bd1be') }}</li>
          <li>{{ tr('m_251a77247045') }}</li>
          <li>{{ tr('m_4c2f23822e3d') }}</li>
          <li>{{ tr('m_19dff4cfa946') }}</li>
          <li>{{ tr('m_3c816b7f18b2') }}</li>
          <li>{{ tr('m_c046ce1f1e92') }}</li>
        </ul>
      </section>
      <section v-if="latestBody || checkError" class="settings-card settings-card-wide release-card">
        <div class="settings-card-header">
          <div>
            <div class="settings-card-title">{{ tr('m_25b119e0eede') }}</div>
            <div class="settings-card-desc">{{ tr('m_f9ee97eb4887') }}{{ latestTag ? ` · ${latestTag}` : '' }}。</div>
          </div>
        </div>
        <div v-if="latestBody" class="release-body release-markdown" v-html="latestBodyHtml"></div>
        <div v-else class="release-body release-error">{{ checkError }}</div>
      </section>
      <section class="settings-card settings-card-wide">
        <div class="settings-card-header">
          <div>
            <div class="settings-card-title">{{ tr('m_9351fb06ea54') }}</div>
            <div class="settings-card-desc">{{ tr('m_0e22df176d79') }}</div>
          </div>
        </div>
        <div class="feature-grid">
          <div class="feature-item">
            <strong>{{ tr('m_e920f47d0c05') }}</strong>
            <p>{{ tr('m_ff534389da66') }}</p>
          </div>
          <div class="feature-item">
            <strong>{{ tr('m_4bf8b66d7390') }}</strong>
            <p>{{ tr('m_b3c3831a9225') }}</p>
          </div>
          <div class="feature-item">
            <strong>{{ tr('m_474fb6002594') }}</strong>
            <p>{{ tr('m_8178ecd45796') }}</p>
          </div>
          <div class="feature-item">
            <strong>{{ tr('m_1b0049fcfd41') }}</strong>
            <p>{{ tr('m_23efa07a6b81') }}</p>
          </div>
          <div class="feature-item">
            <strong>{{ tr('m_3db29474a184') }}</strong>
            <p>{{ tr('m_165170c6d7f2') }}</p>
          </div>
          <div class="feature-item">
            <strong>Telegram Bot</strong>
            <p>{{ tr('m_d2acf85ae520') }}</p>
          </div>
          <div class="feature-item">
            <strong>Cloudflare OAuth</strong>
            <p>{{ tr('m_20c9a5fcb58a') }}</p>
          </div>
          <div class="feature-item">
            <strong>{{ tr('m_f6a93d54ba44') }}</strong>
            <p>{{ tr('m_4d4267727699') }}</p>
          </div>
          <div class="feature-item">
            <strong>{{ tr('m_384a5d60f314') }}</strong>
            <p>{{ tr('m_d33996c19397') }}</p>
          </div>
          <div class="feature-item">
            <strong>{{ tr('m_53b57f1e9171') }}</strong>
            <p>{{ tr('m_8716134bee6b') }}</p>
          </div>
        </div>
      </section>
      <section class="settings-card settings-card-wide">
        <div class="settings-card-header">
          <div>
            <div class="settings-card-title">{{ tr('m_265b45019e65') }}</div>
            <div class="settings-card-desc">{{ tr('m_3d9b89277530') }}</div>
          </div>
        </div>
        <div class="stack-row"><span>{{ tr('m_866a4fa82c25') }}</span><code>Vue 3 · TypeScript · Naive UI · Vite · Pinia</code></div>
        <div class="stack-row"><span>{{ tr('m_9b826346239e') }}</span><code>Go · chi · SQLite</code></div>
        <div class="stack-row"><span>{{ tr('m_4115d63bd549') }}</span><code>Cloudflare API · Huawei Cloud DNS API · Telegram Bot API · GitHub Actions</code></div>
      </section>
    </div>
  </div>
</template>
<script setup lang="ts">
import { t as tr } from '../i18n'

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
  if (checking.value) return tr('m_2a114c7e95b7')
  if (!latestVersion.value) return checkError.value ? tr('m_4f6f06254d16') : tr('m_cdbc0434f568')
  const c = parseVersion(currentVersion.value)
  const l = parseVersion(latestVersion.value)
  if (c.length && l.length) {
    const diff = compareVersion(l, c)
    if (diff > 0) return tr('m_596dd46cfde5') + latestVersion.value
    if (diff === 0) return tr('m_bc310480b4ce')
    return tr('m_681395cc358f')
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
    checkError.value = tr('m_ec5a2ccad49c')
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
.settings-grid {
  display: grid;
  gap: var(--spacing-lg);
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
