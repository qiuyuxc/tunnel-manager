<template>
  <div class="login-page">
    <div class="login-card" :class="{ 'login-card-enter': mounted, 'login-card-shake': shaking }">
      <div class="login-logo">
        <span class="logo-mark" aria-hidden="true">
          <svg width="18" height="18" viewBox="0 0 76 76" fill="none">
            <path d="M49 26H27v24l22-24z" fill="currentColor"/>
            <path d="M38 38L27 50h22L38 38z" fill="currentColor" fill-opacity="0.42"/>
          </svg>
        </span>
      </div>
      <h1 class="login-title">Tunnel Manager</h1>

      <Transition name="step-fade" mode="out-in">
        <div :key="step">
          <template v-if="step === 'credentials'">
            <p class="login-subtitle">登录以继续</p>
            <form class="login-form" @submit.prevent="handleLogin">
              <div class="field">
                <label class="field-label" for="login-username">用户名</label>
                <input
                  id="login-username"
                  ref="usernameInput"
                  v-model="form.username"
                  type="text"
                  placeholder="admin"
                  class="vercel-input"
                  :class="{ 'input-error': error }"
                  autocomplete="username"
                />
              </div>
              <div class="field">
                <label class="field-label" for="login-password">密码</label>
                <input
                  id="login-password"
                  ref="passwordInput"
                  v-model="form.password"
                  type="password"
                  placeholder="••••••••"
                  class="vercel-input"
                  :class="{ 'input-error': error }"
                  autocomplete="current-password"
                />
              </div>

              <div v-if="error" class="login-error" role="alert">{{ error }}</div>

              <button type="submit" class="btn btn-primary login-btn" :disabled="loading">
                <span v-if="loading" class="spinner"></span>
                {{ loading ? '登录中...' : '登录' }}
              </button>
            </form>
          </template>

          <template v-else>
            <div class="factor-heading">
              <span class="security-stamp">2FA REQUIRED</span>
              <p class="login-subtitle">输入认证器验证码或一次性恢复码</p>
            </div>
            <form class="login-form" @submit.prevent="handleFactorLogin">
              <div class="factor-meta">
                <span>验证请求</span>
                <strong :class="{ 'countdown-warning': remainingSeconds <= 60 }">{{ countdown }}</strong>
              </div>
              <div class="field">
                <label class="field-label" for="login-factor">验证码或恢复码</label>
                <input
                  id="login-factor"
                  ref="factorInput"
                  v-model="factorCode"
                  type="text"
                  placeholder="000000 或 XXXX-XXXX-…"
                  class="vercel-input factor-input"
                  :class="{ 'input-error': error }"
                  autocomplete="one-time-code"
                  autocapitalize="characters"
                  spellcheck="false"
                  maxlength="32"
                />
              </div>

              <div v-if="error" class="login-error" role="alert" aria-live="polite">{{ error }}</div>

              <button type="submit" class="btn btn-primary login-btn" :disabled="loading">
                <span v-if="loading" class="spinner"></span>
                {{ loading ? '验证中...' : '验证并登录' }}
              </button>
              <button type="button" class="btn btn-ghost login-btn secondary-action" :disabled="loading" @click="resetChallenge()">
                返回密码登录
              </button>
            </form>
          </template>
        </div>
      </Transition>
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { completeTwoFactorLogin, login as loginApi } from '../api'
import { useConfigStore } from '../stores/config'

const router = useRouter()
const store = useConfigStore()

const step = ref<'credentials' | 'factor'>('credentials')
const form = reactive({ username: '', password: '' })
const factorCode = ref('')
const challengeToken = ref('')
const challengeExpiresAt = ref(0)
const remainingSeconds = ref(0)
const failedFactorAttempts = ref(0)
const loading = ref(false)
const error = ref('')
const mounted = ref(false)
const shaking = ref(false)
const usernameInput = ref<HTMLInputElement | null>(null)
const passwordInput = ref<HTMLInputElement | null>(null)
const factorInput = ref<HTMLInputElement | null>(null)
let countdownTimer: number | undefined
let entranceFrame: number | undefined
let shakeFrame: number | undefined
let shakeTimer: number | undefined
let disposed = false

const countdown = ref('05:00')

onMounted(() => {
  entranceFrame = requestAnimationFrame(() => {
    entranceFrame = undefined
    if (!disposed) mounted.value = true
  })
  nextTick(() => {
    if (!disposed) usernameInput.value?.focus()
  })
})

onBeforeUnmount(() => {
  disposed = true
  clearChallengeTimer()
  if (entranceFrame !== undefined) cancelAnimationFrame(entranceFrame)
  if (shakeFrame !== undefined) cancelAnimationFrame(shakeFrame)
  if (shakeTimer !== undefined) window.clearTimeout(shakeTimer)
})

async function handleLogin() {
  if (!form.username || !form.password) {
    error.value = '请输入用户名和密码'
    triggerShake()
    return
  }
  loading.value = true
  error.value = ''
  try {
    const response = await loginApi(form.username, form.password)
    if (response.status === 202 && 'two_factor_required' in response.data) {
      const expiresAt = Date.parse(response.data.expires_at)
      if (!Number.isFinite(expiresAt) || expiresAt <= Date.now()) {
        error.value = '验证请求无效或已过期，请重新登录'
        form.password = ''
        await nextTick()
        passwordInput.value?.focus()
        return
      }
      challengeToken.value = response.data.challenge_token
      challengeExpiresAt.value = expiresAt
      form.password = ''
      factorCode.value = ''
      failedFactorAttempts.value = 0
      step.value = 'factor'
      startChallengeTimer()
      await nextTick()
      factorInput.value?.focus()
      return
    }
    if ('token' in response.data) {
      store.setAuth(response.data.token, response.data.username)
      await router.replace('/')
    }
  } catch (e: any) {
    error.value = e.response?.status === 401
      ? '用户名或密码错误'
      : '登录失败: ' + (e.response?.data?.error || e.message)
    triggerShake()
    await nextTick()
    passwordInput.value?.focus()
  } finally {
    loading.value = false
  }
}

async function handleFactorLogin() {
  const code = factorCode.value.trim()
  if (!code) {
    error.value = '请输入验证码或恢复码'
    return
  }
  if (Date.now() >= challengeExpiresAt.value) {
    resetChallenge('验证请求已过期，请重新输入密码')
    return
  }
  loading.value = true
  error.value = ''
  try {
    const { data } = await completeTwoFactorLogin(challengeToken.value, code)
    clearChallengeTimer()
    store.setAuth(data.token, data.username)
    await router.replace('/')
  } catch (e: any) {
    if (e.response?.status === 401) {
      failedFactorAttempts.value += 1
      if (failedFactorAttempts.value >= 5) {
        resetChallenge('验证次数已用尽，请重新输入密码')
        return
      }
      error.value = '验证码或恢复码无效'
    } else {
      error.value = '验证失败: ' + (e.response?.data?.error || e.message)
    }
    factorCode.value = ''
    await nextTick()
    factorInput.value?.focus()
  } finally {
    loading.value = false
  }
}

function startChallengeTimer() {
  clearChallengeTimer()
  updateCountdown()
  countdownTimer = window.setInterval(updateCountdown, 1000)
}

function updateCountdown() {
  const seconds = Math.max(0, Math.ceil((challengeExpiresAt.value - Date.now()) / 1000))
  remainingSeconds.value = seconds
  countdown.value = `${String(Math.floor(seconds / 60)).padStart(2, '0')}:${String(seconds % 60).padStart(2, '0')}`
  if (seconds === 0) resetChallenge('验证请求已过期，请重新输入密码')
}

function resetChallenge(message = '') {
  clearChallengeTimer()
  step.value = 'credentials'
  challengeToken.value = ''
  challengeExpiresAt.value = 0
  factorCode.value = ''
  failedFactorAttempts.value = 0
  error.value = message
  nextTick(() => passwordInput.value?.focus())
}

function clearChallengeTimer() {
  if (countdownTimer !== undefined) {
    window.clearInterval(countdownTimer)
    countdownTimer = undefined
  }
}

function triggerShake() {
  if (shakeFrame !== undefined) cancelAnimationFrame(shakeFrame)
  if (shakeTimer !== undefined) window.clearTimeout(shakeTimer)
  shaking.value = false
  shakeFrame = requestAnimationFrame(() => {
    shakeFrame = undefined
    if (disposed) return
    shaking.value = true
    shakeTimer = window.setTimeout(() => {
      shakeTimer = undefined
      if (!disposed) shaking.value = false
    }, 500)
  })
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-lg);
  background:
    radial-gradient(circle at 18% 12%, color-mix(in srgb, var(--color-link) 12%, transparent), transparent 28%),
    linear-gradient(135deg, var(--color-canvas-soft), var(--color-canvas));
  box-sizing: border-box;
}

.login-card {
  width: 100%;
  max-width: 408px;
  background: var(--color-canvas);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  padding: var(--spacing-2xl);
  text-align: center;
  box-shadow: 0 24px 60px rgba(58, 47, 34, 0.12);
  opacity: 0;
  transform: translateY(16px) scale(0.98);
  transition: opacity 420ms ease-out, transform 420ms cubic-bezier(0.16, 1, 0.3, 1);
}

.login-card-enter { opacity: 1; transform: translateY(0) scale(1); }
.login-card-shake { animation: shake 0.45s ease-out; }
.login-logo { margin-bottom: var(--spacing-md); display: flex; justify-content: center; }

.logo-mark {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-lg);
  background: var(--color-ink);
  color: var(--color-canvas);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.login-title {
  font-family: var(--font-display);
  font-size: 25px;
  font-weight: 600;
  line-height: 1.15;
  color: var(--color-ink);
  margin: 0 0 6px;
}

.login-subtitle { font-size: 14px; color: var(--color-mute); margin: 0 0 var(--spacing-xl); }
.login-form { display: flex; flex-direction: column; gap: var(--spacing-md); text-align: left; }
.field { display: flex; flex-direction: column; gap: 6px; }
.field-label { font-size: 13px; font-weight: 700; color: var(--color-ink); }

.factor-heading { margin-top: var(--spacing-md); }
.security-stamp {
  display: inline-flex;
  padding: 5px 9px;
  margin-bottom: var(--spacing-sm);
  border: 1px solid var(--color-banner-warning-border);
  border-radius: var(--radius-sm);
  color: var(--color-banner-warning-text);
  background: var(--color-banner-warning-bg);
  font: 700 11px/1 var(--font-mono);
  letter-spacing: 0.08em;
}

.factor-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: var(--spacing-sm);
  border-bottom: 1px solid var(--color-hairline);
  color: var(--color-mute);
  font-size: 12px;
}

.factor-meta strong { color: var(--color-ink); font: 700 13px/1 var(--font-mono); }
.factor-meta .countdown-warning { color: var(--color-banner-warning-text); }
.factor-input { font-family: var(--font-mono); letter-spacing: 0.04em; }

.login-error {
  font-size: 14px;
  color: var(--color-result-error-text);
  text-align: center;
  padding: 9px 10px;
  background: var(--color-result-error-bg);
  border: 1px solid var(--color-result-error-border);
  border-radius: var(--radius-md);
  animation: fadeIn 180ms ease-out;
}

.login-btn { width: 100%; justify-content: center; margin-top: var(--spacing-xs); }
.secondary-action { margin-top: calc(var(--spacing-sm) * -1); }
.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid color-mix(in srgb, currentColor 26%, transparent);
  border-top-color: currentColor;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

.step-fade-enter-active, .step-fade-leave-active { transition: opacity 160ms ease; }
.step-fade-enter-from, .step-fade-leave-to { opacity: 0; }
@keyframes spin { to { transform: rotate(360deg); } }
@keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }

@media (max-width: 480px) {
  .login-card { padding: var(--spacing-xl) var(--spacing-lg); }
}
</style>
