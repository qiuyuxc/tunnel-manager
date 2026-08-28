<template>
  <div class="login-page">
    <div class="login-card" :class="{ 'login-card-shake': shaking }">
      <div class="login-logo">
        <span class="logo-mark" aria-hidden="true">
          <img v-if="store.config.site_icon" :src="store.config.site_icon" alt="" class="brand-icon" />
          <svg v-else width="18" height="18" viewBox="0 0 76 76" fill="none">
            <path d="M49 26H27v24l22-24z" fill="currentColor"/>
            <path d="M38 38L27 50h22L38 38z" fill="currentColor" fill-opacity="0.42"/>
          </svg>
        </span>
      </div>
      <h1 class="login-title">{{ store.config.site_name }}</h1>
      <p class="login-subtitle">{{ store.config.site_description || '登录以继续' }}</p>
      <div v-if="modeSwitchable" class="mode-tabs">
        <button type="button" class="mode-tab" :class="{ active: mode === 'login' }" @click="switchMode('login')">登录</button>
        <button type="button" class="mode-tab" :class="{ active: mode === 'register' }" @click="switchMode('register')">注册</button>
      </div>
      <Transition name="step-fade" mode="out-in">
        <div :key="mode + '-' + step">
          <template v-if="mode === 'forgot'">
            <div class="factor-heading">
              <span class="security-stamp">RESET PASSWORD</span>
              <p class="login-subtitle">通过注册邮箱重置密码（需管理员已配置邮件服务）</p>
            </div>
            <form class="login-form" @submit.prevent="handleResetPassword">
              <div class="field">
                <label class="field-label" for="fp-email">注册邮箱</label>
                <div class="email-row">
                  <input id="fp-email" v-model="forgotForm.email" type="email" placeholder="you@example.com" class="vercel-input" />
                  <button type="button" class="btn btn-secondary code-btn" :disabled="codeCooldown > 0 || codeSending" @click="handleForgotSend">
                    {{ codeCooldown > 0 ? codeCooldown + 's' : (codeSending ? '发送中' : '发送验证码') }}
                  </button>
                </div>
              </div>
              <div class="field">
                <label class="field-label" for="fp-code">重置验证码</label>
                <input id="fp-code" v-model="forgotForm.code" type="text" placeholder="6 位数字" class="vercel-input" maxlength="6" inputmode="numeric" />
              </div>
              <div class="field">
                <label class="field-label" for="fp-password">新密码</label>
                <input id="fp-password" v-model="forgotForm.newPassword" type="password" placeholder="至少 6 位" class="vercel-input" autocomplete="new-password" />
              </div>
              <div v-if="error" class="login-error" role="alert">{{ error }}</div>
              <div v-if="turnstileEnabled" ref="forgotTurnstile" class="turnstile-wrap"></div>
              <button type="submit" class="btn btn-primary login-btn" :disabled="loading">
                <span v-if="loading" class="spinner"></span>
                {{ loading ? '重置中...' : '重置密码' }}
              </button>
              <button type="button" class="btn btn-ghost login-btn" @click="switchMode('login')">返回登录</button>
            </form>
          </template>
          <template v-else-if="mode === 'register'">
            <form class="login-form" @submit.prevent="handleRegister">
              <div class="field">
                <label class="field-label" for="reg-username">用户名</label>
                <input id="reg-username" v-model="regForm.username" type="text" placeholder="2-32 位字母、数字、_ 或 -" class="vercel-input" autocomplete="username" />
              </div>
              <div class="field">
                <label class="field-label" for="reg-email">邮箱</label>
                <div class="email-row">
                  <input id="reg-email" v-model="regForm.email" type="email" placeholder="you@example.com" class="vercel-input" autocomplete="email" />
                  <button
                    v-if="authConfig.email_verify_enabled"
                    type="button"
                    class="btn btn-secondary code-btn"
                    :disabled="codeCooldown > 0 || codeSending"
                    @click="handleSendCode"
                  >
                    {{ codeCooldown > 0 ? codeCooldown + 's' : (codeSending ? '发送中' : '发送验证码') }}
                  </button>
                </div>
              </div>
              <div v-if="authConfig.email_verify_enabled" class="field">
                <label class="field-label" for="reg-code">邮箱验证码</label>
                <input id="reg-code" v-model="regForm.verifyCode" type="text" placeholder="6 位数字" class="vercel-input" maxlength="6" inputmode="numeric" />
              </div>
              <div class="field">
                <label class="field-label" for="reg-password">密码</label>
                <input id="reg-password" v-model="regForm.password" type="password" placeholder="至少 6 位" class="vercel-input" autocomplete="new-password" />
              </div>
              <div v-if="authConfig.invite_mode !== 'off'" class="field">
                <label class="field-label" for="reg-invite">
                  邀请码{{ authConfig.invite_mode === 'required' ? '' : '（选填）' }}
                </label>
                <input id="reg-invite" v-model="regForm.invite" type="text" placeholder="邀请码" class="vercel-input" />
              </div>
              <div v-if="error" class="login-error" role="alert">{{ error }}</div>
              <div v-if="turnstileEnabled" ref="regTurnstile" class="turnstile-wrap"></div>
              <button type="submit" class="btn btn-primary login-btn" :disabled="loading">
                <span v-if="loading" class="spinner"></span>
                {{ loading ? '注册中...' : '注册并登录' }}
              </button>
              <button type="button" class="link-btn" @click="switchMode('login')">已有账号？去登录</button>
            </form>
          </template>
          <template v-else-if="step === 'credentials'">
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
              <div v-if="turnstileEnabled" ref="loginTurnstile" class="turnstile-wrap"></div>
              <button type="submit" class="btn btn-primary login-btn" :disabled="loading">
                <span v-if="loading" class="spinner"></span>
                {{ loading ? '登录中...' : '登录' }}
              </button>
              <button type="button" class="link-btn" @click="switchMode('forgot')">忘记密码？</button>
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
              <button type="button" class="btn btn-ghost login-btn" :disabled="loading" @click="resetChallenge()">
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
import { getAuthConfig, sendRegisterCode, register, forgotPassword, resetPassword, type AuthConfig } from '../api/admin'
import { useConfigStore } from '../stores/config'
const router = useRouter()
const store = useConfigStore()
const step = ref<'credentials' | 'factor'>('credentials')
const mode = ref<'login' | 'register' | 'forgot'>('login')
const authConfig = ref<AuthConfig>({ registration_enabled: false, invite_mode: 'off', email_verify_enabled: false, turnstile_enabled: false, turnstile_site_key: '' })
const turnstileEnabled = ref(false)
const turnstileSiteKey = ref('')
const turnstileLoaded = ref(false)
const turnstileWidgetID = ref('')
const turnstileToken = ref('')
const loginTurnstile = ref<HTMLElement | null>(null)
const regTurnstile = ref<HTMLElement | null>(null)
const forgotTurnstile = ref<HTMLElement | null>(null)
let turnstileScriptPromise: Promise<void> | null = null
const modeSwitchable = ref(false)
const regForm = reactive({ username: '', email: '', password: '', invite: '', verifyCode: '' })
const forgotForm = reactive({ email: '', code: '', newPassword: '' })
const codeCooldown = ref(0)
const codeSending = ref(false)
let cooldownTimer: number | undefined
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
  getAuthConfig().then(({ data }) => {
    authConfig.value = data
    modeSwitchable.value = data.registration_enabled
    turnstileEnabled.value = data.turnstile_enabled
    turnstileSiteKey.value = data.turnstile_site_key || ''
    if (data.turnstile_enabled) void renderTurnstile()
    if (router.currentRoute.value.query.mode === 'register' && data.registration_enabled) {
      mode.value = 'register'
      nextTick(() => document.getElementById('reg-username')?.focus())
    }
  }).catch(() => {})
})
onBeforeUnmount(() => {
  disposed = true
  clearChallengeTimer()
  destroyTurnstile()
  if (cooldownTimer !== undefined) window.clearInterval(cooldownTimer)
  if (entranceFrame !== undefined) cancelAnimationFrame(entranceFrame)
  if (shakeFrame !== undefined) cancelAnimationFrame(shakeFrame)
  if (shakeTimer !== undefined) window.clearTimeout(shakeTimer)
})
function loadTurnstileScript(): Promise<void> {
  if (turnstileLoaded.value) return Promise.resolve()
  if (turnstileScriptPromise) return turnstileScriptPromise
  turnstileScriptPromise = new Promise<void>((resolve) => {
    const existing = document.querySelector<HTMLScriptElement>('script[data-turnstile]')
    if (existing) {
      if (window.turnstile) {
        resolve()
      } else {
        existing.addEventListener('load', () => resolve(), { once: true })
      }
      return
    }
    const script = document.createElement('script')
    script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
    script.async = true
    script.defer = true
    script.dataset.turnstile = 'true'
    script.addEventListener('load', () => resolve(), { once: true })
    document.head.appendChild(script)
  }).then(() => { turnstileLoaded.value = true })
  return turnstileScriptPromise
}

function turnstileContainer(): HTMLElement | null {
  if (mode.value === 'register') return regTurnstile.value
  if (mode.value === 'forgot') return forgotTurnstile.value
  return loginTurnstile.value
}

async function renderTurnstile() {
  destroyTurnstile()
  if (!turnstileEnabled.value || !turnstileSiteKey.value) return
  if (mode.value === 'login' && step.value !== 'credentials') return
  await loadTurnstileScript().catch(() => {})
  await nextTick()
  const container = turnstileContainer()
  if (!container || !window.turnstile) return
  turnstileWidgetID.value = window.turnstile.render(container, {
    sitekey: turnstileSiteKey.value,
    action: mode.value,
    callback: (token: string) => { turnstileToken.value = token },
    'expired-callback': () => { turnstileToken.value = '' },
    'error-callback': () => { turnstileToken.value = '' },
  })
}

function destroyTurnstile() {
  if (turnstileWidgetID.value && window.turnstile) {
    try { window.turnstile.remove(turnstileWidgetID.value) } catch { /* ignore */ }
  }
  turnstileWidgetID.value = ''
  turnstileToken.value = ''
}

// Consumes the current single-use token and removes the widget. The caller
// must re-render a fresh widget afterwards.
function takeTurnstileToken(): string {
  const token = turnstileEnabled.value ? turnstileToken.value : ''
  destroyTurnstile()
  return token
}

async function handleLogin() {
  if (!form.username || !form.password) {
    error.value = '请输入用户名和密码'
    triggerShake()
    return
  }
  const turnstileTokenValue = takeTurnstileToken()
  if (turnstileEnabled.value && !turnstileTokenValue) {
    error.value = '请完成人机验证'
    triggerShake()
    void renderTurnstile()
    return
  }
  loading.value = true
  error.value = ''
  try {
    const response = await loginApi(form.username, form.password, turnstileTokenValue)
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
      await router.replace('/dashboard')
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
    void renderTurnstile()
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
    await router.replace('/dashboard')
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
  nextTick(() => {
    passwordInput.value?.focus()
    void renderTurnstile()
  })
}
function clearChallengeTimer() {
  if (countdownTimer !== undefined) {
    window.clearInterval(countdownTimer)
    countdownTimer = undefined
  }
}
function switchMode(target: 'login' | 'register' | 'forgot') {
  if (mode.value === target) return
  mode.value = target
  error.value = ''
  step.value = 'credentials'
  nextTick(() => {
    void renderTurnstile()
    if (target === 'register') {
      document.getElementById('reg-username')?.focus()
    } else if (target === 'forgot') {
      document.getElementById('fp-email')?.focus()
    } else {
      usernameInput.value?.focus()
    }
  })
}

async function handleSendCode() {
  const email = regForm.email.trim()
  if (!email) {
    error.value = '请先填写邮箱'
    return
  }
  const turnstileTokenValue = takeTurnstileToken()
  if (turnstileEnabled.value && !turnstileTokenValue) {
    error.value = '请完成人机验证'
    void renderTurnstile()
    return
  }
  codeSending.value = true
  error.value = ''
  try {
    await sendRegisterCode(email, turnstileTokenValue)
    codeCooldown.value = 60
    cooldownTimer = window.setInterval(() => {
      codeCooldown.value -= 1
      if (codeCooldown.value <= 0) {
        window.clearInterval(cooldownTimer)
        cooldownTimer = undefined
      }
    }, 1000)
  } catch (e: any) {
    error.value = e.response?.data?.error || '验证码发送失败'
  } finally {
    codeSending.value = false
    void renderTurnstile()
  }
}

async function handleForgotSend() {
  const email = forgotForm.email.trim()
  if (!email) {
    error.value = '请先填写注册邮箱'
    return
  }
  const turnstileTokenValue = takeTurnstileToken()
  if (turnstileEnabled.value && !turnstileTokenValue) {
    error.value = '请完成人机验证'
    void renderTurnstile()
    return
  }
  codeSending.value = true
  error.value = ''
  try {
    await forgotPassword(email, turnstileTokenValue)
    codeCooldown.value = 60
    cooldownTimer = window.setInterval(() => {
      codeCooldown.value -= 1
      if (codeCooldown.value <= 0) {
        window.clearInterval(cooldownTimer)
        cooldownTimer = undefined
      }
    }, 1000)
  } catch (e: any) {
    error.value = e.response?.data?.error || '验证码发送失败'
  } finally {
    codeSending.value = false
    void renderTurnstile()
  }
}

async function handleResetPassword() {
  const email = forgotForm.email.trim()
  if (!email || !forgotForm.code.trim() || !forgotForm.newPassword) {
    error.value = '请填写邮箱、验证码和新密码'
    return
  }
  if (forgotForm.newPassword.length < 6) {
    error.value = '新密码至少 6 位'
    return
  }
  const turnstileTokenValue = takeTurnstileToken()
  if (turnstileEnabled.value && !turnstileTokenValue) {
    error.value = '请完成人机验证'
    triggerShake()
    void renderTurnstile()
    return
  }
  loading.value = true
  error.value = ''
  try {
    await resetPassword(email, forgotForm.code.trim(), forgotForm.newPassword, turnstileTokenValue)
    forgotForm.email = ''
    forgotForm.code = ''
    forgotForm.newPassword = ''
    switchMode('login')
    error.value = '密码已重置，请使用新密码登录'
  } catch (e: any) {
    error.value = e.response?.data?.error || '重置失败'
  } finally {
    loading.value = false
    void renderTurnstile()
  }
}

async function handleRegister() {
  const username = regForm.username.trim()
  const email = regForm.email.trim()
  if (!username || !email || !regForm.password) {
    error.value = '请填写用户名、邮箱和密码'
    triggerShake()
    return
  }
  if (authConfig.value.email_verify_enabled && !regForm.verifyCode.trim()) {
    error.value = '请输入邮箱验证码'
    triggerShake()
    return
  }
  if (authConfig.value.invite_mode === 'required' && !regForm.invite.trim()) {
    error.value = '邀请码不能为空'
    triggerShake()
    return
  }
  const turnstileTokenValue = takeTurnstileToken()
  if (turnstileEnabled.value && !turnstileTokenValue) {
    error.value = '请完成人机验证'
    triggerShake()
    void renderTurnstile()
    return
  }
  loading.value = true
  error.value = ''
  try {
    const { data } = await register({
      username,
      email,
      password: regForm.password,
      invite: regForm.invite.trim() || undefined,
      verify_code: regForm.verifyCode.trim() || undefined,
      cf_turnstile_response: turnstileTokenValue,
    })
    store.setAuth(data.token, data.username, data.role || 'user')
    regForm.password = ''
    await router.replace('/dashboard')
  } catch (e: any) {
    error.value = e.response?.data?.error || '注册失败: ' + (e.message || '')
    triggerShake()
  } finally {
    loading.value = false
    void renderTurnstile()
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
  background: var(--color-canvas-soft);
  box-sizing: border-box;
}

.login-card {
  width: 100%;
  max-width: 400px;
  background: var(--color-canvas-raised);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  padding: var(--spacing-2xl);
  text-align: center;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
}

.login-card-shake { animation: shake 0.45s ease-out; }

.login-logo { margin-bottom: var(--spacing-md); display: flex; justify-content: center; }

.logo-mark {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  background: var(--color-link);
  color: #fff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.brand-icon { width: 100%; height: 100%; object-fit: cover; border-radius: inherit; }

.login-title {
  font-size: 22px;
  font-weight: 600;
  line-height: 1.2;
  color: var(--color-ink);
  margin: 0 0 6px;
}

.login-subtitle { font-size: 14px; color: var(--color-mute); margin: 0 0 var(--spacing-xl); }

.login-form { display: flex; flex-direction: column; gap: var(--spacing-md); text-align: left; }

.mode-tabs {
  display: flex;
  gap: 4px;
  padding: 4px;
  margin-bottom: var(--spacing-lg);
  background: var(--color-canvas-soft);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
}

.mode-tab {
  flex: 1;
  padding: 7px 0;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-mute);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 120ms ease, color 120ms ease;
}

.mode-tab.active {
  background: var(--color-canvas-raised);
  color: var(--color-ink);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
}

.email-row { display: flex; gap: 8px; }
.email-row .vercel-input { flex: 1; }
.code-btn { white-space: nowrap; font-size: 13px; }

.field { display: flex; flex-direction: column; gap: 6px; }

.field-label { font-size: 13px; font-weight: 500; color: var(--color-ink); }

.factor-heading { margin-top: var(--spacing-md); }

.security-stamp {
  display: inline-flex;
  padding: 5px 9px;
  margin-bottom: var(--spacing-sm);
  border: 1px solid var(--color-status-degraded-border);
  border-radius: var(--radius-sm);
  color: var(--color-status-degraded-text);
  background: var(--color-status-degraded-bg);
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
.factor-meta .countdown-warning { color: var(--color-warning); }
.factor-input { font-family: var(--font-mono); letter-spacing: 0.04em; }

.login-error {
  font-size: 13px;
  color: var(--color-error);
  text-align: center;
  padding: 8px 10px;
  background: var(--color-status-down-bg);
  border: 1px solid var(--color-status-down-border);
  border-radius: var(--radius-md);
}

.login-btn { width: 100%; justify-content: center; margin-top: var(--spacing-xs); }

.link-btn {
  background: none;
  border: none;
  color: var(--color-mute);
  font-size: 12px;
  cursor: pointer;
  padding: 2px 0;
  text-decoration: underline;
  text-underline-offset: 3px;
}

.link-btn:hover { color: var(--color-ink); }

.turnstile-wrap {
  margin: 12px 0 4px;
  min-height: 65px;
  display: flex;
  justify-content: center;
}

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

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  20% { transform: translateX(-4px); }
  40% { transform: translateX(4px); }
  60% { transform: translateX(-3px); }
  80% { transform: translateX(3px); }
}

@media (max-width: 480px) {
  .login-card { padding: var(--spacing-xl) var(--spacing-lg); }
}
</style>