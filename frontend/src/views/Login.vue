<template>
  <div class="login-page">
    <div class="login-language"><LanguageSelect /></div>
    <aside class="login-story">
      <BrandMark :size="52" />
      <div class="login-features">
        <div><span>01</span><strong>{{ tr('m_e920f47d0c05') }}</strong><small>{{ tr('m_91e0e3d19ebf') }}</small></div>
        <div><span>02</span><strong>{{ tr('m_7fab82054a8f') }}</strong><small>{{ tr('m_c9c94c3a6470') }}</small></div>
        <div><span>03</span><strong>{{ tr('m_474fb6002594') }}</strong><small>{{ tr('m_8f643f738546') }}</small></div>
      </div>
    </aside>
    <div class="login-card" :class="{ 'login-card-shake': shaking }">
      <div class="login-logo">
        <span class="logo-mark" aria-hidden="true">
          <img v-if="store.config.site_icon" :src="store.config.site_icon" alt="" class="brand-icon" />
<BrandMark v-else :size="32" />
        </span>
      </div>
      <h1 class="login-title">{{ store.config.site_name }}</h1>
      <p class="login-subtitle">{{ store.config.site_description || tr('m_f702792b3358') }}</p>

      <Transition name="step-fade" mode="out-in">
        <div :key="mode + '-' + step">

          <template v-if="mode === 'forgot'">
            <div class="factor-heading">
              <span class="security-stamp">RESET PASSWORD</span>
              <p class="login-subtitle">{{ tr('m_6bd2317c80bf') }}</p>
            </div>
            <form class="login-form" @submit.prevent="handleResetPassword">
              <div class="field">
                <label class="field-label" for="fp-email">{{ tr('m_a751637c0425') }}</label>
                <div class="email-row">
                  <input id="fp-email" v-model="forgotForm.email" type="email" placeholder="you@example.com" class="vercel-input" />
                  <button type="button" class="btn btn-secondary code-btn" :disabled="codeCooldown > 0 || codeSending" @click="handleForgotSend">
                    {{ codeCooldown > 0 ? codeCooldown + 's' : (codeSending ? tr('m_9eacfc58bf9f') : tr('m_4656fed515ae')) }}
                  </button>
                </div>
              </div>
              <div class="field">
                <label class="field-label" for="fp-code">{{ tr('m_23ca81d774ca') }}</label>
                <input id="fp-code" v-model="forgotForm.code" type="text" :placeholder="tr('m_7ce6963f19ce')" class="vercel-input" maxlength="6" inputmode="numeric" />
              </div>
              <div class="field">
                <label class="field-label" for="fp-password">{{ tr('m_515e9c7cf7b2') }}</label>
                <input id="fp-password" v-model="forgotForm.newPassword" type="password" :placeholder="tr('m_041568c2db7a')" class="vercel-input" autocomplete="new-password" />
              </div>
              <div v-if="error" class="login-error" role="alert">{{ error }}</div>
              <div v-if="turnstileEnabled" ref="forgotTurnstile" class="turnstile-wrap"></div>
              <button type="submit" class="btn btn-primary login-btn" :disabled="loading">
                <span v-if="loading" class="spinner"></span>
                {{ loading ? tr('m_9be1af6abe7d') : tr('m_d70928debc2e') }}
              </button>
              <button type="button" class="btn btn-ghost login-btn" @click="switchMode('login')">{{ tr('m_747b0f9082e6') }}</button>
            </form>
          </template>

          <template v-else-if="step === 'credentials'">
            <form class="login-form" @submit.prevent="handleLogin">
              <div class="field">
                <label class="field-label" for="login-username">{{ tr('m_1a3f0617d6de') }}</label>
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
                <label class="field-label" for="login-password">{{ tr('m_a621ab606db2') }}</label>
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
                {{ loading ? tr('m_c0b692e4f515') : tr('m_1e2df9c3075a') }}
              </button>
              <button v-if="passkeySupported" type="button" class="btn btn-secondary login-btn" :disabled="loading || passkeyBusy" @click="handlePasskeyLogin">
                <span v-if="passkeyBusy" class="spinner"></span>
                {{ passkeyBusy ? tr('m_7c93e5eb917e') : tr('m_e984682db258') }}
              </button>

              <button type="button" class="link-btn" @click="switchMode('forgot')">{{ tr('m_1d02d7f1c5c4') }}</button>
            </form>
          </template>
          <template v-else>
            <div class="factor-heading">
              <span class="security-stamp">2FA REQUIRED</span>
              <p class="login-subtitle">{{ tr('m_6c6cfabd59bf') }}</p>
            </div>
            <form class="login-form" @submit.prevent="handleFactorLogin">
              <div class="factor-meta">
                <span>{{ tr('m_8bd2fa0068b8') }}</span>
                <strong :class="{ 'countdown-warning': remainingSeconds <= 60 }">{{ countdown }}</strong>
              </div>
              <div class="field">
                <label class="field-label" for="login-factor">{{ tr('m_3b9346649649') }}</label>
                <input
                  id="login-factor"
                  ref="factorInput"
                  v-model="factorCode"
                  type="text"
                  :placeholder="tr('m_555c49803059')"
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
                {{ loading ? tr('m_6c128e0ad353') : tr('m_4e34537e5d0c') }}
              </button>
              <button v-if="passkeySupported && passkeyFactorAvailable" type="button" class="btn btn-secondary login-btn" :disabled="loading || passkeyBusy" @click="handlePasskeyFactor">
                <span v-if="passkeyBusy" class="spinner"></span>
                {{ passkeyBusy ? tr('m_7c93e5eb917e') : tr('m_28c3158cdf27') }}
              </button>
              <button type="button" class="btn btn-ghost login-btn" :disabled="loading" @click="resetChallenge()">
                {{ tr('m_f7e73e3576e7') }}
              </button>
            </form>
          </template>
        </div>
      </Transition>
    </div>
  </div>
</template>
<script setup lang="ts">
import LanguageSelect from '../components/LanguageSelect.vue'
import BrandMark from '../components/BrandMark.vue'
import { t as tr } from '../i18n'

import { nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { browserSupportsWebAuthn, startAuthentication } from '@simplewebauthn/browser'
import { completeTwoFactorLogin, login as loginApi } from '../api'
import { getAuthConfig, forgotPassword, resetPassword } from '../api/admin'
import { beginPasskeyLogin, beginPasskeyTwoFactor, finishPasskeyLogin, finishPasskeyTwoFactor } from '../api/passkey'
import { useConfigStore } from '../stores/config'
const router = useRouter()
const store = useConfigStore()
const step = ref<'credentials' | 'factor'>('credentials')
const mode = ref<'login' | 'forgot'>('login')
const turnstileEnabled = ref(false)
const turnstileSiteKey = ref('')
const turnstileLoaded = ref(false)
const turnstileWidgetID = ref('')
const turnstileToken = ref('')
const loginTurnstile = ref<HTMLElement | null>(null)
const forgotTurnstile = ref<HTMLElement | null>(null)
let turnstileScriptPromise: Promise<void> | null = null
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
// 通行密钥：仅浏览器支持 WebAuthn 时展示入口
const passkeySupported = ref(false)
const passkeyBusy = ref(false)
const passkeyFactorAvailable = ref(false)
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
  passkeySupported.value = browserSupportsWebAuthn()
  entranceFrame = requestAnimationFrame(() => {
    entranceFrame = undefined
    if (!disposed) mounted.value = true
  })
  nextTick(() => {
    if (!disposed) usernameInput.value?.focus()
  })
  getAuthConfig().then(({ data }) => {
    if (disposed) return
    turnstileEnabled.value = data.turnstile_enabled
    turnstileSiteKey.value = data.turnstile_site_key || ''
    if (data.turnstile_enabled) nextTick(() => void renderTurnstile())
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
    // 找回密码页共用一个验证组件，服务发送验证码与重置密码接口
    // （forgot-password/reset-password）。带页面级
    // action 会导致这些接口 action 不匹配被拒，因此这些页面不传 action
    // （siteverify 回显空 action，后端即放行）；登录页只对应 login，保留。
    action: mode.value === 'login' ? 'login' : undefined,
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

// 通行密钥免密登录：用户名可选，留空则由浏览器挑选已绑定的通行密钥。
async function handlePasskeyLogin() {
  if (passkeyBusy.value || loading.value || disposed) return
  passkeyBusy.value = true
  error.value = ''
  try {
    const { data } = await beginPasskeyLogin(form.username.trim())
    if (disposed) return
    const credential = await startAuthentication({ optionsJSON: data.public_key })
    if (disposed) return
    const response = await finishPasskeyLogin({ ceremony_token: data.ceremony_token, credential })
    if (disposed) return
    store.setAuth(response.data.token, response.data.username)
    await router.replace('/dashboard')
  } catch (e: any) {
    if (e?.name === 'NotAllowedError' || e?.name === 'AbortError') {
      error.value = tr('m_db900a083e3e')
    } else {
      error.value = e.response?.data?.error || e.message || tr('m_9b4340ac3382')
    }
    triggerShake()
  } finally {
    passkeyBusy.value = false
  }
}

// 2FA 的通行密钥分支：密码已通过，这里只证明持有通行密钥。
async function handlePasskeyFactor() {
  if (passkeyBusy.value || loading.value || disposed || !challengeToken.value) return
  passkeyBusy.value = true
  error.value = ''
  try {
    const { data } = await beginPasskeyTwoFactor(challengeToken.value)
    if (disposed) return
    const credential = await startAuthentication({ optionsJSON: data.public_key })
    if (disposed) return
    const response = await finishPasskeyTwoFactor({
      challenge_token: challengeToken.value,
      ceremony_token: data.ceremony_token,
      credential,
    })
    if (disposed) return
    store.setAuth(response.data.token, response.data.username)
    await router.replace('/dashboard')
  } catch (e: any) {
    if (e?.name === 'NotAllowedError' || e?.name === 'AbortError') {
      error.value = tr('m_db900a083e3e')
    } else {
      error.value = e.response?.data?.error || e.message || tr('m_17c96c45fc4c')
    }
    triggerShake()
  } finally {
    passkeyBusy.value = false
  }
}

async function handleLogin() {
  if (loading.value || passkeyBusy.value || disposed) return
  if (!form.username || !form.password) {
    error.value = tr('m_1793872ca1c9')
    triggerShake()
    return
  }
  const turnstileTokenValue = takeTurnstileToken()
  if (turnstileEnabled.value && !turnstileTokenValue) {
    error.value = tr('m_10ef93c129e5')
    triggerShake()
    void renderTurnstile()
    return
  }
  loading.value = true
  error.value = ''
  try {
    const response = await loginApi(form.username, form.password, turnstileTokenValue)
    if (disposed) return
    if (response.status === 202 && 'two_factor_required' in response.data) {
      const expiresAt = Date.parse(response.data.expires_at)
      if (!Number.isFinite(expiresAt) || expiresAt <= Date.now()) {
        error.value = tr('m_4c0839ec2ac6')
        form.password = ''
        await nextTick()
        passwordInput.value?.focus()
        return
      }
      challengeToken.value = response.data.challenge_token
      challengeExpiresAt.value = expiresAt
      passkeyFactorAvailable.value = !!response.data.passkeys_available
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
      if (disposed) return
      store.setAuth(response.data.token, response.data.username)
      await router.replace('/dashboard')
    }
  } catch (e: any) {
    error.value = e.response?.status === 401
      ? tr('m_a9a6a83a7d22')
      : tr('m_193e0b000ce7') + (e.response?.data?.error || e.message)
    triggerShake()
    await nextTick()
    passwordInput.value?.focus()
  } finally {
    loading.value = false
    void renderTurnstile()
  }
}
async function handleFactorLogin() {
  if (loading.value || passkeyBusy.value || disposed) return
  const code = factorCode.value.trim()
  if (!code) {
    error.value = tr('m_61eb3fa7e4f4')
    return
  }
  if (Date.now() >= challengeExpiresAt.value) {
    resetChallenge(tr('m_a6b11a4afb71'))
    return
  }
  loading.value = true
  error.value = ''
  try {
    const { data } = await completeTwoFactorLogin(challengeToken.value, code)
    clearChallengeTimer()
    if (disposed) return
    store.setAuth(data.token, data.username)
    await router.replace('/dashboard')
  } catch (e: any) {
    if (e.response?.status === 401) {
      failedFactorAttempts.value += 1
      if (failedFactorAttempts.value >= 5) {
        resetChallenge(tr('m_82fadaa985ec'))
        return
      }
      error.value = tr('m_84ac8c912896')
    } else {
      error.value = tr('m_1f752a49011b') + (e.response?.data?.error || e.message)
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
  if (seconds === 0) resetChallenge(tr('m_a6b11a4afb71'))
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
function switchMode(target: 'login' | 'forgot') {
  if (mode.value === target) return
  mode.value = target
  error.value = ''
  step.value = 'credentials'
  nextTick(() => {
    void renderTurnstile()
    if (target === 'forgot') {
      document.getElementById('fp-email')?.focus()
    } else {
      usernameInput.value?.focus()
    }
  })
}

async function handleForgotSend() {
  if (codeSending.value || codeCooldown.value > 0 || disposed) return
  const email = forgotForm.email.trim()
  if (!email) {
    error.value = tr('m_9643ca765953')
    return
  }
  const turnstileTokenValue = takeTurnstileToken()
  if (turnstileEnabled.value && !turnstileTokenValue) {
    error.value = tr('m_10ef93c129e5')
    void renderTurnstile()
    return
  }
  codeSending.value = true
  error.value = ''
  try {
    await forgotPassword(email, turnstileTokenValue)
    if (disposed) return
    codeCooldown.value = 60
    cooldownTimer = window.setInterval(() => {
      codeCooldown.value -= 1
      if (codeCooldown.value <= 0) {
        window.clearInterval(cooldownTimer)
        cooldownTimer = undefined
      }
    }, 1000)
  } catch (e: any) {
    error.value = e.response?.data?.error || tr('m_f2d0fada0803')
  } finally {
    codeSending.value = false
    void renderTurnstile()
  }
}

async function handleResetPassword() {
  if (loading.value || passkeyBusy.value || disposed) return
  const email = forgotForm.email.trim()
  if (!email || !forgotForm.code.trim() || !forgotForm.newPassword) {
    error.value = tr('m_3be67f5752f6')
    return
  }
  if (forgotForm.newPassword.length < 6) {
    error.value = tr('m_4e1f11e21737')
    return
  }
  const turnstileTokenValue = takeTurnstileToken()
  if (turnstileEnabled.value && !turnstileTokenValue) {
    error.value = tr('m_10ef93c129e5')
    triggerShake()
    void renderTurnstile()
    return
  }
  loading.value = true
  error.value = ''
  try {
    await resetPassword(email, forgotForm.code.trim(), forgotForm.newPassword, turnstileTokenValue)
    if (disposed) return
    forgotForm.email = ''
    forgotForm.code = ''
    forgotForm.newPassword = ''
    switchMode('login')
    error.value = tr('m_310a97dbe461')
  } catch (e: any) {
    error.value = e.response?.data?.error || tr('m_4230b353344a')
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
.login-language { position: fixed; top: max(16px, env(safe-area-inset-top)); right: max(16px, env(safe-area-inset-right)); z-index: 30; width: min(244px, calc(100vw - 32px)); background: var(--color-canvas-raised); border: 1px solid var(--color-hairline); border-radius: 14px; }

.login-story { width: min(420px, 40vw); min-width: 0; }
.login-features { display: flex; flex-direction: column; gap: 20px; margin-top: 28px; }
.login-features > div { display: grid; grid-template-columns: 30px 1fr; gap: 5px 12px; }
.login-features span { grid-row: 1 / 3; color: var(--color-link); font: 12px var(--font-mono); padding-top: 3px; }
.login-features strong { color: var(--color-ink); font-weight: 500; font-size: 14px; }
.login-features small { color: var(--color-mute); font-size: 12px; line-height: 1.7; }
@media (max-width: 768px) { .login-story { display: none; } .login-page { padding: 72px 16px 32px; } }

.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: clamp(40px, 8vw, 100px);
  padding: 72px 32px;
  background: var(--color-canvas-soft);
  box-sizing: border-box;
}

.login-card {
  width: 100%;
  max-width: 420px;
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
  background: var(--color-btn-primary-bg);
  color: var(--color-btn-primary-text);
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
