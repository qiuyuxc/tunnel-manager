<template>
  <div class="pub-page">
    <button class="theme-toggle" :title="isDark ? '切换到亮色' : '切换到暗色'" @click="toggleTheme">
      <svg v-if="isDark" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
      <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
    </button>
    <div class="pub-wrap">
      <template v-if="data">
        <div class="pub-head">
          <div class="pub-brand" aria-hidden="true">
            <img v-if="isIconUrl" :src="data.public_icon" alt="" />
            <svg v-else-if="!data.public_icon" width="20" height="20" viewBox="0 0 76 76" fill="none"><path d="M49 26H27v24l22-24z" fill="currentColor"/><path d="M38 38L27 50h22L38 38z" fill="currentColor" fill-opacity=".42"/></svg>
            <span v-else class="brand-emoji">{{ data.public_icon }}</span>
          </div>
          <h1>{{ data.public_title || data.name }}</h1>
          <p v-if="updatedText" class="pub-sub">最近更新 {{ updatedText }} · 每 {{ data.interval_sec }}s 检测一次</p>
        </div>

        <div v-if="overall !== null" class="pub-overall" :class="overall">
          <span class="ov-dot"></span>
          <span v-if="overall === 'all'">全部服务运行正常</span>
          <span v-else-if="overall === 'some'">部分服务出现异常</span>
          <span v-else>多个服务不可达</span>
        </div>

        <div v-if="data.announcement" class="pub-notice">{{ data.announcement }}</div>

        <div class="pub-list">
          <section v-for="(t, i) in data.targets" :key="i" class="svc">
            <header class="svc-head">
              <span class="st-dot" :class="'d-' + t.state"></span>
              <strong class="svc-name">{{ t.name }}</strong>
              <a v-if="t.link" class="svc-link" :href="t.link" target="_blank" rel="noopener" :title="'打开 ' + t.name">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><line x1="7" y1="17" x2="17" y2="7"/><polyline points="7 7 17 7 17 17"/></svg>
              </a>
              <span class="up-pct">近24h可用 {{ fmtPct(t.uptime_24h) }}</span>
              <span class="lat">{{ t.latency_ms != null && t.state !== 'down' ? t.latency_ms + 'ms' : '' }}{{ t.state === 'down' ? (t.error || '无响应') : '' }}</span>
            </header>
            <div class="bars-lg" aria-hidden="true">
              <i v-for="(b, bi) in t.bars" :key="bi" :class="'b-' + b.s"></i>
            </div>
            <footer class="svc-foot">
              <span>过去 {{ Math.min(t.bars.length, 40) * data.interval_sec >= 60 ? Math.round(Math.min(t.bars.length, 40) * data.interval_sec / 60) + ' 分钟' : Math.min(t.bars.length, 40) * data.interval_sec + ' 秒' }}内</span>
              <span>{{ barOk(t.bars) }}/{{ t.bars.length }} 正常</span>
            </footer>
          </section>
        </div>

        <footer class="pub-footer">Powered by Tunnel Manager</footer>
      </template>

      <div v-else-if="error" class="pub-error">
        <h1>{{ error }}</h1>
        <router-link to="/" class="home-link">返回首页</router-link>
      </div>
      <div v-else class="pub-loading">加载中…</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { fetchPublicStatus, type PublicStatusData } from '../api'

const route = useRoute()
const token = route.params.token as string

const data = ref<PublicStatusData | null>(null)
const error = ref('')

const storedTheme = typeof localStorage !== 'undefined' ? localStorage.getItem('pub_dark_mode') : null
const isDark = ref(
  storedTheme === 'true' ||
  (storedTheme === null && typeof matchMedia !== 'undefined' && matchMedia('(prefers-color-scheme: dark)').matches)
)

function applyTheme() {
  document.documentElement.setAttribute('data-theme', isDark.value ? 'dark' : '')
  localStorage.setItem('pub_dark_mode', isDark.value ? 'true' : 'false')
}

function applyAdminTheme(theme: PublicStatusData['public_theme']) {
  const root = document.documentElement
  if (theme === 'warm') {
    root.setAttribute('data-visual-theme', 'warm')
  } else {
    root.removeAttribute('data-visual-theme')
  }
}

function toggleTheme() {
  isDark.value = !isDark.value
  applyTheme()
}

const isIconUrl = computed(() =>
  /^(https?:\/\/|\/uploads\/|data:image\/)/.test((data.value?.public_icon || '').trim())
)
let timer: ReturnType<typeof setInterval> | undefined

const updatedText = computed(() => {
  if (!data.value?.updated_at) return ''
  const d = new Date(data.value.updated_at)
  return isNaN(d.getTime()) ? '' : d.toLocaleTimeString()
})

const overall = computed<null | 'all' | 'some' | 'many'>(() => {
  if (!data.value || !data.value.targets.length) return null
  const down = data.value.targets.filter((t) => t.state === 'down').length
  const warn = data.value.targets.filter((t) => t.state === 'warn').length
  if (down > 0) return 'many'
  if (warn > 0) return 'some'
  return 'all'
})

function fmtPct(v: number) {
  return v.toFixed(2).replace(/\.00$/, '') + '%'
}

function barOk(bars: Array<{ s: string }>) {
  return bars.filter((b) => b.s === 'ok').length
}

async function load() {
  try {
    const res = await fetchPublicStatus(token)
    data.value = res
    applyAdminTheme(res.public_theme)
    document.title = (res.name || '状态页') + ' · 服务状态'
    error.value = ''
  } catch (_) {
    error.value = '页面不存在或已被关闭'
  }
}

let savedVisual: string | null = null
let savedDarkAttr: string | null = null

onMounted(() => {
  const root = document.documentElement
  savedVisual = root.getAttribute('data-visual-theme')
  savedDarkAttr = root.getAttribute('data-theme')
  root.setAttribute('data-theme', isDark.value ? 'dark' : '')
  load()
  timer = setInterval(load, 30000)
})
onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
  // 还原主面板属性：公开页的主题不外溢到后台
  const root = document.documentElement
  if (savedVisual === null) root.removeAttribute('data-visual-theme')
  else root.setAttribute('data-visual-theme', savedVisual)
  if (savedDarkAttr === null) root.removeAttribute('data-theme')
  else root.setAttribute('data-theme', savedDarkAttr)
})
</script>

<style scoped>
.pub-page { min-height: 100vh; background: var(--color-canvas-soft); color: var(--color-ink); padding: 48px 16px; }

.theme-toggle {
  position: fixed; top: 16px; right: 16px; z-index: 20;
  width: 34px; height: 34px; border-radius: 999px;
  display: grid; place-items: center; cursor: pointer;
  background: var(--color-canvas-raised); border: 1px solid var(--color-hairline-strong);
  color: var(--color-body); transition: border-color 120ms ease, color 120ms ease, transform 120ms ease;
}
.theme-toggle:hover { border-color: var(--color-focus); color: var(--color-ink); }

.svc-icon { margin-right: 7px; }
.svc-logo {
  width: 20px; height: 20px; object-fit: contain; vertical-align: -4px;
  margin-right: 8px; border-radius: 5px;
}
.svc-link {
  display: inline-flex; align-items: center; justify-content: center;
  width: 20px; height: 20px; border-radius: 5px; flex-shrink: 0;
  color: var(--color-mute); transition: color 120ms ease, background 120ms ease;
}
.svc-link:hover { color: var(--color-link, var(--color-ink)); background: var(--color-canvas-soft); }

.pub-notice {
  display: flex; align-items: center; gap: 9px;
  padding: 11px 14px; margin-bottom: 18px; border-radius: 10px;
  font-size: 13px; line-height: 1.6; color: var(--color-body);
  background: var(--color-canvas-raised); border: 1px dashed var(--color-hairline-strong);
}
.pub-wrap { max-width: 680px; margin: 0 auto; }
.pub-head { display: flex; flex-direction: column; align-items: center; gap: 10px; margin-bottom: 26px; text-align: center; }
.pub-brand { width: 52px; height: 52px; border-radius: 14px; display: grid; place-items: center; overflow: hidden;
  background: var(--color-canvas-raised); border: 1px solid var(--color-hairline); }
.pub-brand img {
  display: block; width: 100%; height: 100%; object-fit: cover;
}
.pub-brand svg { width: 24px; height: 24px; }
.pub-brand .brand-emoji { display: block; line-height: 1; font-size: 34px; }

.pub-head h1 { margin: 0; font-size: 22px; font-weight: 600; }
.pub-sub { margin: 0; font-size: 12.5px; color: var(--color-mute); }

.pub-overall {
  display: flex; align-items: center; justify-content: center; gap: 9px;
  padding: 13px; border-radius: 10px; font-size: 14px; font-weight: 600; margin-bottom: 18px;
  border: 1px solid var(--color-hairline); background: var(--color-canvas-raised);
}
.ov-dot { width: 9px; height: 9px; border-radius: 50%; background: var(--color-mute); }
.pub-overall.all .ov-dot { background: var(--color-success); box-shadow: 0 0 0 4px color-mix(in srgb, var(--color-success) 15%, transparent); }
.pub-overall.all { color: var(--color-success); }
.pub-overall.some .ov-dot { background: var(--color-warning); }
.pub-overall.some { color: var(--color-warning); }
.pub-overall.many .ov-dot { background: var(--color-error); }
.pub-overall.many { color: var(--color-error); }

.pub-list { display: flex; flex-direction: column; gap: 12px; }
.svc { background: var(--color-canvas-raised); border: 1px solid var(--color-hairline); border-radius: 10px; padding: 16px 18px; }
.svc-head { display: flex; align-items: center; gap: 9px; margin-bottom: 11px; }
.st-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--color-mute); flex-shrink: 0; }
.d-ok { background: var(--color-success); }
.d-warn { background: var(--color-warning); }
.d-down { background: var(--color-error); }
.svc-name { font-size: 14px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.up-pct { margin-left: auto; font-size: 11.5px; color: var(--color-body); flex-shrink: 0; }
.lat { font-family: var(--font-mono); font-size: 11px; color: var(--color-mute); min-width: 56px; text-align: right;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex-shrink: 0; }

.bars-lg { display: flex; gap: 3px; height: 30px; margin-bottom: 8px; }
.bars-lg i { flex: 1; border-radius: 2px; background: var(--color-canvas-soft-2); }
.bars-lg i.b-ok { background: var(--color-success); opacity: .85; }
.bars-lg i.b-warn { background: var(--color-warning); }
.bars-lg i.b-down { background: var(--color-error); }

.svc-foot { display: flex; justify-content: space-between; font-size: 11px; color: var(--color-mute); }

.pub-loading, .pub-error { text-align: center; padding: 80px 0; color: var(--color-mute); }
.pub-error h1 { font-size: 17px; color: var(--color-body); margin-bottom: 12px; }
.home-link { color: var(--color-link); }

.pub-footer { margin-top: 28px; text-align: center; font-size: 11.5px; color: var(--color-mute); }
@media (max-width: 640px) {
  .pub-page { padding-top: 32px; }
}
</style>
