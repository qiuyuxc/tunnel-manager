<template>
  <div class="landing-page">
    <!-- top-nav: 56px sticky, canvas bg, secondary Sign in + primary CTA right -->
    <header class="lp-nav">
      <router-link to="/" class="lp-brand">
        <span class="lp-logo">
          <img v-if="store.config.site_icon" :src="store.config.site_icon" alt="" />
          <svg v-else width="16" height="16" viewBox="0 0 76 76" fill="none" aria-hidden="true">
            <path d="M49 26H27v24l22-24z" fill="currentColor"/>
            <path d="M38 38L27 50h22L38 38z" fill="currentColor" fill-opacity="0.42"/>
          </svg>
        </span>
        <span class="lp-brand-name">{{ store.config.site_name || 'Tunnel Manager' }}</span>
      </router-link>
      <nav class="lp-nav-actions">
        <button class="lp-theme-toggle" type="button"
          :title="store.darkMode ? '切换亮色模式' : '切换暗色模式'"
          :aria-label="store.darkMode ? '切换亮色模式' : '切换暗色模式'"
          @click="store.toggleDarkMode()">
          <svg v-if="store.darkMode" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <circle cx="12" cy="12" r="4"/>
            <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>
          </svg>
          <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/>
          </svg>
        </button>
        <template v-if="!store.isAuthenticated">
          <router-link to="/login" class="lp-btn lp-btn-secondary nav-secondary">登录</router-link>
          <router-link v-if="registrationEnabled" to="/login?mode=register" class="lp-btn lp-btn-primary">注册</router-link>
        </template>
      </nav>
    </header>

    <main class="lp-main">
      <!-- Hero: dark canvas + product screenshot panel as protagonist -->
      <section class="lp-hero">
        <p class="lp-eyebrow">
          <span class="lp-eyebrow-dot"></span>
          Cloudflare Tunnel 管理面板
        </p>
        <h1 class="lp-h1">{{ store.config.site_name || 'Tunnel Manager' }}</h1>
        <p class="lp-sub">{{ store.config.site_description || 'Cloudflare 隧道管理中心' }}</p>
        <div class="lp-hero-actions">
          <template v-if="store.isAuthenticated">
            <router-link to="/dashboard" class="lp-btn lp-btn-primary lp-btn-lg">进入控制台</router-link>
            <router-link to="/notifications" class="lp-btn lp-btn-secondary lp-btn-lg">通知设置</router-link>
          </template>
          <template v-else>
            <router-link to="/login" class="lp-btn lp-btn-primary lp-btn-lg">立即登录</router-link>
            <router-link v-if="registrationEnabled" to="/login?mode=register" class="lp-btn lp-btn-secondary lp-btn-lg">注册账号</router-link>
          </template>
        </div>

        <!-- Product screenshot card (the protagonist) -->
        <div class="lp-shot" aria-hidden="true">
          <div class="lp-shot-bar">
            <span class="lp-shot-dot"></span>
            <span class="lp-shot-dot"></span>
            <span class="lp-shot-dot"></span>
            <span class="lp-shot-title">控制面板 · 概览</span>
          </div>
          <div class="lp-shot-body">
            <div class="lp-shot-stat">
              <span class="lp-stat-label">tunnel</span>
              <strong>cf-090227</strong>
              <span class="lp-badge lp-badge-ok">运行中</span>
            </div>
            <div class="lp-shot-rows">
              <div class="lp-shot-row">
                <span class="lp-row-label">公网入口</span>
                <span class="lp-bar w55"></span>
                <span class="lp-badge">3 ACTIVE</span>
              </div>
              <div class="lp-shot-row">
                <span class="lp-row-label">DNS 记录</span>
                <span class="lp-bar w38"></span>
                <span class="lp-badge">已同步</span>
              </div>
              <div class="lp-shot-row">
                <span class="lp-row-label">服务监控</span>
                <span class="lp-bar w46"></span>
                <span class="lp-badge lp-badge-ok">健康</span>
              </div>
            </div>
            <div class="lp-shot-foot">
              <span class="lp-foot-label">telegram bot</span>
              <span class="lp-badge">CONNECTED</span>
            </div>
          </div>
        </div>
      </section>

      <!-- Features: charcoal card grid, 3-up -->
      <section class="lp-section lp-features">
        <p class="lp-eyebrow">功能</p>
        <h2 class="lp-h2">一站式隧道管理</h2>
        <p class="lp-sub">从隧道、DNS 到监控与通知，一个控制台全部搞定。</p>
        <div class="lp-feature-grid">
          <div class="lp-feature">
            <span class="lp-feature-icon">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18M9 3v18"/></svg>
            </span>
            <h3>隧道管理</h3>
            <p>集中管理 Cloudflare Tunnel，随时查看运行状态与公网入口。</p>
          </div>
          <div class="lp-feature">
            <span class="lp-feature-icon">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3a15 15 0 0 1 0 18M12 3a15 15 0 0 0 0 18"/></svg>
            </span>
            <h3>DNS 解析</h3>
            <p>可视化管理域名解析记录，批量维护 A、CNAME、TXT 等类型。</p>
          </div>
          <div class="lp-feature">
            <span class="lp-feature-icon">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 17l6-6 4 4 8-8"/><path d="M15 7h6v6"/></svg>
            </span>
            <h3>状态监控</h3>
            <p>探测服务可用性，异常时通过邮件或 Telegram 及时告警。</p>
          </div>
          <div class="lp-feature">
            <span class="lp-feature-icon">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 11.5a8.5 8.5 0 0 1-12.4 7.6L3 21l1.9-5.6A8.5 8.5 0 1 1 21 11.5z"/></svg>
            </span>
            <h3>通知与远程控制</h3>
            <p>自定义通知渠道，登录、告警及时送达；支持 Telegram 远程指令。</p>
          </div>
        </div>
      </section>

      <!-- CTA banner -->
      <section class="lp-section lp-cta">
        <div class="lp-cta-card">
          <h2 class="lp-h3">准备好了吗？</h2>
          <p class="lp-sub">立即登录，开始管理你的 Cloudflare 隧道。</p>
          <div class="lp-hero-actions">
            <template v-if="store.isAuthenticated">
              <router-link to="/dashboard" class="lp-btn lp-btn-primary lp-btn-lg">进入控制台</router-link>
            </template>
            <template v-else>
              <router-link to="/login" class="lp-btn lp-btn-primary lp-btn-lg">立即登录</router-link>
              <router-link v-if="registrationEnabled" to="/login?mode=register" class="lp-btn lp-btn-secondary lp-btn-lg">注册账号</router-link>
            </template>
          </div>
        </div>
      </section>
    </main>

    <footer class="lp-footer">
      <div class="lp-footer-brand">{{ store.config.site_name || 'Tunnel Manager' }}</div>
      <nav class="lp-footer-links">
        <a href="https://github.com/qiuyuxc/tunnel-manager" target="_blank" rel="noopener">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M12 .5C5.65.5.5 5.65.5 12c0 5.08 3.29 9.39 7.86 10.91.58.11.79-.25.79-.56 0-.28-.01-1.02-.02-2-3.2.7-3.88-1.54-3.88-1.54-.53-1.34-1.29-1.7-1.29-1.7-1.05-.72.08-.71.08-.71 1.16.08 1.77 1.2 1.77 1.2 1.03 1.77 2.7 1.26 3.36.96.1-.75.4-1.26.73-1.55-2.55-.29-5.23-1.28-5.23-5.68 0-1.26.45-2.29 1.19-3.1-.12-.29-.52-1.46.11-3.05 0 0 .97-.31 3.18 1.18a11.1 11.1 0 0 1 5.79 0c2.2-1.49 3.17-1.18 3.17-1.18.63 1.59.23 2.76.12 3.05.74.81 1.18 1.84 1.18 3.1 0 4.41-2.69 5.38-5.25 5.66.41.36.78 1.06.78 2.14 0 1.55-.01 2.8-.01 3.18 0 .31.21.68.8.56A11.5 11.5 0 0 0 23.5 12C23.5 5.65 18.35.5 12 .5z"/>
          </svg>
          <span>tunnel-manager</span>
        </a>
      </nav>
      <p class="lp-footer-note">Powered by Cloudflare Tunnel</p>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getAuthConfig } from '../api/admin'
import { useConfigStore } from '../stores/config'

const store = useConfigStore()
const registrationEnabled = ref(false)

onMounted(() => {
  store.fetchSiteSettings()
  getAuthConfig().then(({ data }) => {
    registrationEnabled.value = !!data.registration_enabled
  }).catch(() => {})
})
</script>

<style scoped>
/* uDESIGN.md — Linear design language: near-black canvas #010102, lavender
   #5e6ad2, charcoal surface ladder + hairline borders, product screenshot as
   protagonist, no gradients, 8px-radius buttons (never pill). */
.landing-page,
.landing-page *,
.landing-page *::before,
.landing-page *::after {
  box-sizing: border-box;
}

.landing-page {
  /* Light: inverse Linear surface ladder */
  --lp-canvas: #ffffff;
  --lp-surface-1: #f5f6f6;
  --lp-surface-2: #ededee;
  --lp-hairline: #e6e7e8;
  --lp-hairline-strong: #d0d2d5;
  --lp-ink: #171717;
  --lp-ink-muted: #3d3d3d;
  --lp-ink-subtle: #6b6f76;
  --lp-ink-tertiary: #9ca0a6;
  --lp-primary: #5e6ad2;
  --lp-primary-hover: #4b57c0;
  --lp-primary-text: #ffffff;
  --lp-success-bg: rgba(39, 166, 68, 0.12);
  --lp-success-text: #1c8a37;
  --lp-edge: rgba(255, 255, 255, 0.8);
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--lp-canvas);
  color: var(--lp-ink);
  font-family: var(--font-sans);
  overflow-x: hidden;
}

[data-theme="dark"] .landing-page {
  /* uDESIGN.md Linear dark canvas */
  --lp-canvas: #010102;
  --lp-surface-1: #0f1011;
  --lp-surface-2: #141516;
  --lp-hairline: #23252a;
  --lp-hairline-strong: #34343a;
  --lp-ink: #f7f8f8;
  --lp-ink-muted: #d0d6e0;
  --lp-ink-subtle: #8a8f98;
  --lp-ink-tertiary: #62666d;
  --lp-primary: #5e6ad2;
  --lp-primary-hover: #828fff;
  --lp-primary-text: #ffffff;
  --lp-success-bg: rgba(39, 166, 68, 0.14);
  --lp-success-text: #4ade80;
  --lp-edge: rgba(255, 255, 255, 0.06);
}

/* ---------- top-nav (56px, sticky) ---------- */
.lp-nav {
  position: sticky;
  top: 0;
  z-index: 20;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
  padding: 0 32px;
  background: var(--lp-canvas);
  border-bottom: 1px solid var(--lp-hairline);
}

.lp-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  color: var(--lp-ink);
  text-decoration: none;
  font-size: 14px;
  font-weight: 500;
}

.lp-logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 6px;
  background: var(--lp-surface-1);
  border: 1px solid var(--lp-hairline);
  color: var(--lp-primary);
}

.lp-logo img {
  width: 16px;
  height: 16px;
  border-radius: 4px;
  object-fit: cover;
}

.lp-brand-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.lp-nav-actions {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  gap: 10px;
}

.lp-theme-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: var(--lp-surface-1);
  border: 1px solid var(--lp-hairline);
  color: var(--lp-ink-subtle);
  cursor: pointer;
  transition: background-color 0.15s ease, border-color 0.15s ease, color 0.15s ease;
}

.lp-theme-toggle:hover {
  background: var(--lp-surface-2);
  border-color: var(--lp-hairline-strong);
  color: var(--lp-ink);
}

/* ---------- buttons (8px radius, 14px/500) ---------- */
.lp-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  line-height: 1.2;
  padding: 8px 14px;
  text-decoration: none;
  white-space: nowrap;
  max-width: 100%;
  cursor: pointer;
  transition: background-color 0.15s ease, border-color 0.15s ease;
}

.lp-btn-lg {
  padding: 12px 20px;
  font-size: 15px;
  min-height: 44px;
}

.lp-btn-primary {
  background: var(--lp-primary);
  color: var(--lp-primary-text);
  border: 1px solid var(--lp-primary);
}

.lp-btn-primary:hover { background: var(--lp-primary-hover); border-color: var(--lp-primary-hover); }

.lp-btn-secondary {
  background: var(--lp-surface-1);
  color: var(--lp-ink);
  border: 1px solid var(--lp-hairline);
}

.lp-btn-secondary:hover { background: var(--lp-surface-2); border-color: var(--lp-hairline-strong); }

/* ---------- hero ---------- */
.lp-main {
  flex: 1;
  width: 100%;
  max-width: 1280px;
  margin: 0 auto;
  padding: 0 32px;
}

.lp-hero {
  text-align: center;
  padding: 96px 0 24px;
}

.lp-eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 13px;
  font-weight: 500;
  letter-spacing: 0.4px;
  color: var(--lp-ink-subtle);
}

.lp-eyebrow-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--lp-primary);
}

.lp-h1 {
  margin: 20px 0 0;
  font-size: 80px;
  font-weight: 600;
  line-height: 1.05;
  letter-spacing: -3px;
  overflow-wrap: break-word;
  word-break: break-word;
}

.lp-h2 {
  margin: 16px 0 0;
  font-size: 56px;
  font-weight: 600;
  line-height: 1.1;
  letter-spacing: -1.8px;
}

.lp-h3 {
  margin: 0;
  font-size: 28px;
  font-weight: 600;
  line-height: 1.2;
  letter-spacing: -0.6px;
}

.lp-sub {
  margin: 20px auto 0;
  max-width: 620px;
  font-size: 18px;
  font-weight: 400;
  line-height: 1.5;
  letter-spacing: -0.1px;
  color: var(--lp-ink-subtle);
}

.lp-hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: center;
  margin-top: 36px;
}

/* ---------- product screenshot panel ---------- */
.lp-shot {
  max-width: 820px;
  margin: 64px auto 0;
  text-align: left;
  background: var(--lp-surface-1);
  border: 1px solid var(--lp-hairline);
  border-radius: 16px;
  box-shadow: inset 0 1px 0 var(--lp-edge);
  overflow: hidden;
}

.lp-shot-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--lp-hairline);
}

.lp-shot-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--lp-hairline-strong);
}

.lp-shot-title {
  margin-left: 10px;
  font-size: 12px;
  color: var(--lp-ink-subtle);
  letter-spacing: 0;
}

.lp-shot-body {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 24px;
}

.lp-shot-stat {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 15px;
}

.lp-stat-label {
  font-family: ui-monospace, "SF Mono", Menlo, monospace;
  font-size: 12px;
  color: var(--lp-ink-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.4px;
}

.lp-shot-stat strong { font-weight: 600; color: var(--lp-ink); }

.lp-shot-rows {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-top: 4px;
}

.lp-shot-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.lp-row-label {
  flex: 0 0 88px;
  font-size: 13px;
  color: var(--lp-ink-muted);
}

.lp-bar {
  height: 8px;
  border-radius: 4px;
  background: var(--lp-hairline);
  min-width: 40px;
}

.lp-bar.w55 { width: 55%; }
.lp-bar.w38 { width: 38%; }
.lp-bar.w46 { width: 46%; }

.lp-badge {
  margin-left: auto;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--lp-surface-2);
  color: var(--lp-ink-muted);
  font-size: 11px;
  letter-spacing: 0.4px;
  white-space: nowrap;
}

.lp-badge-ok {
  background: var(--lp-success-bg);
  color: var(--lp-success-text);
}

.lp-shot-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 16px;
  border-top: 1px solid var(--lp-hairline);
}

.lp-foot-label {
  font-family: ui-monospace, "SF Mono", Menlo, monospace;
  font-size: 12px;
  color: var(--lp-ink-tertiary);
  letter-spacing: 0.4px;
}

/* ---------- sections ---------- */
.lp-section {
  padding: 96px 0 24px;
}

.lp-features { text-align: center; }

.lp-feature-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-top: 48px;
  text-align: left;
}

.lp-feature {
  padding: 24px;
  background: var(--lp-surface-1);
  border: 1px solid var(--lp-hairline);
  border-radius: 12px;
  transition: background-color 0.15s ease, border-color 0.15s ease;
}

.lp-feature:hover {
  background: var(--lp-surface-2);
  border-color: var(--lp-hairline-strong);
}

.lp-feature-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: 8px;
  background: var(--lp-surface-2);
  border: 1px solid var(--lp-hairline);
  color: var(--lp-ink-subtle);
}

.lp-feature h3 {
  margin: 16px 0 8px;
  font-size: 18px;
  font-weight: 500;
  letter-spacing: -0.3px;
  color: var(--lp-ink);
}

.lp-feature p {
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--lp-ink-subtle);
}

/* ---------- CTA banner ---------- */
.lp-cta { padding-bottom: 96px; }

.lp-cta-card {
  padding: 48px 32px;
  text-align: center;
  background: var(--lp-surface-1);
  border: 1px solid var(--lp-hairline);
  border-radius: 12px;
  box-shadow: inset 0 1px 0 var(--lp-edge);
}

.lp-cta-card .lp-sub { margin-top: 12px; }

/* ---------- footer ---------- */
.lp-footer {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  padding: 40px 32px 56px;
  border-top: 1px solid var(--lp-hairline);
  text-align: center;
}

.lp-footer-brand {
  font-size: 14px;
  font-weight: 500;
  color: var(--lp-ink);
}

.lp-footer-links {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 20px;
}

.lp-footer-links a {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--lp-ink-subtle);
  text-decoration: none;
}

.lp-footer-links a:hover { color: var(--lp-ink); }

.lp-footer-note {
  margin: 0;
  font-size: 12px;
  color: var(--lp-ink-tertiary);
}

/* ---------- responsive (uDESIGN.md breakpoints) ---------- */
@media (max-width: 1024px) {
  .lp-feature-grid { grid-template-columns: repeat(2, 1fr); }
}

@media (max-width: 768px) {
  .lp-nav { padding: 0 20px; }
  .lp-main { padding: 0 20px; }
  .lp-hero { padding-top: 72px; }
  .lp-h1 { font-size: 48px; letter-spacing: -1.6px; }
  .lp-h2 { font-size: 40px; letter-spacing: -1px; }
  .lp-section { padding-top: 72px; }
  .nav-secondary { display: none; }
}

@media (max-width: 640px) {
  .lp-feature-grid { grid-template-columns: 1fr; }
  .lp-h1 { font-size: 40px; letter-spacing: -1px; }
  .lp-shot { margin-top: 48px; }
  .lp-row-label { flex-basis: 72px; }
}

@media (max-width: 480px) {
  .lp-h1 { font-size: 36px; letter-spacing: -0.8px; }
  .lp-sub { font-size: 16px; }
  .lp-hero-actions { flex-direction: column; align-items: stretch; }
  .lp-hero-actions .lp-btn { width: 100%; }
  .lp-shot { display: none; }
  .lp-cta-card { padding: 32px 20px; }
  .lp-footer { padding: 32px 20px 48px; }
}
</style>
