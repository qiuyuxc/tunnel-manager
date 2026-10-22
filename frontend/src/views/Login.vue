<template>
  <div class="login-page">
    <div class="login-language"><LanguageSelect /></div>
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
      <p class="login-subtitle">{{ store.config.site_description || tr('m_f702792b3358') }}</p>
      <div v-if="modeSwitchable" class="mode-tabs">
        <button type="button" class="mode-tab" :class="{ active: mode === 'login' }" @click="switchMode('login')">{{ tr('m_1e2df9c3075a') }}</button>
        <button type="button" class="mode-tab" :class="{ active: mode === 'register' }" @click="switchMode('register')">{{ tr('m_c4fb62202bad') }}</button>
      </div>
      <Transition name="step-fade" mode="out-in">
        <div :key="mode + '-' + step">
          <template v-if="step === 'oidc-email'">
            <form class="login-form" @submit.prevent="finishOIDCLogin">
              <p class="login-subtitle">{{ tr('m_4af6d732bd51') }}</p>
              <div class="field">
                <label class="field-label" for="oidc-email">{{ tr('m_f8f1b032f460') }}</label>
                <input id="oidc-email" :value="oidcEmail" class="vercel-input" type="email" readonly />
              </div>
              <div class="field">
                <label class="field-label" for="oidc-code">{{ tr('m_2d60deda8e1e') }}</label>
                <input id="oidc-code" v-model="oidcCode" class="vercel-input" type="text" inputmode="numeric" autocomplete="one-time-code" maxlength="6" :placeholder="tr('m_7ce6963f19ce')" />
              </div>
              <div v-if="error" class="login-error" role="alert">{{ error }}</div>
              <button type="button" class="btn btn-secondary login-btn" :disabled="codeSending || codeCooldown > 0 || loading" @click="handleOIDCSendCode">{{ codeSending ? tr('m_9d2ee1e4d8c3') : codeCooldown > 0 ? codeCooldown + tr('m_9ccacca8a78c') : tr('m_4656fed515ae') }}</button>
              <button type="submit" class="btn btn-primary login-btn" :disabled="loading || !oidcCode.trim()">{{ loading ? tr('m_6c128e0ad353') : tr('m_4e34537e5d0c') }}</button>
              <button type="button" class="btn btn-ghost login-btn" :disabled="loading" @click="cancelOIDC">{{ tr('m_747b0f9082e6') }}</button>
            </form>
          </template>
          <template v-else-if="mode === 'forgot'">
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
          <template v-else-if="mode === 'register'">
            <form class="login-form" @submit.prevent="handleRegister">
              <div class="field">
                <label class="field-label" for="reg-username">{{ tr('m_1a3f0617d6de') }}</label>
                <input id="reg-username" v-model="regForm.username" type="text" :placeholder="tr('m_f5ec6ffdb41a')" class="vercel-input" autocomplete="username" />
              </div>
              <div class="field">
                <label class="field-label" for="reg-email">{{ tr('m_73075237fd0f') }}</label>
                <div class="email-row">
                  <input id="reg-email" v-model="regForm.email" type="email" placeholder="you@example.com" class="vercel-input" autocomplete="email" />
                  <button
                    v-if="authConfig.email_verify_enabled"
                    type="button"
                    class="btn btn-secondary code-btn"
                    :disabled="codeCooldown > 0 || codeSending"
                    @click="handleSendCode"
                  >
                    {{ codeCooldown > 0 ? codeCooldown + 's' : (codeSending ? tr('m_9eacfc58bf9f') : tr('m_4656fed515ae')) }}
                  </button>
                </div>
              </div>
              <div v-if="authConfig.email_verify_enabled" class="field">
                <label class="field-label" for="reg-code">{{ tr('m_3acdd163e67a') }}</label>
                <input id="reg-code" v-model="regForm.verifyCode" type="text" :placeholder="tr('m_7ce6963f19ce')" class="vercel-input" maxlength="6" inputmode="numeric" />
              </div>
              <div class="field">
                <label class="field-label" for="reg-password">{{ tr('m_a621ab606db2') }}</label>
                <input id="reg-password" v-model="regForm.password" type="password" :placeholder="tr('m_041568c2db7a')" class="vercel-input" autocomplete="new-password" />
              </div>
              <div v-if="authConfig.invite_mode !== 'off'" class="field">
                <label class="field-label" for="reg-invite">
                  {{ tr('m_c8b56dca7aed') }}{{ authConfig.invite_mode === 'required' ? '' : tr('m_968333deb095') }}
                </label>
                <input id="reg-invite" v-model="regForm.invite" type="text" :placeholder="tr('m_c8b56dca7aed')" class="vercel-input" />
              </div>
              <div v-if="error" class="login-error" role="alert">{{ error }}</div>
              <div v-if="turnstileEnabled" ref="regTurnstile" class="turnstile-wrap"></div>
              <button type="submit" class="btn btn-primary login-btn" :disabled="loading">
                <span v-if="loading" class="spinner"></span>
                {{ loading ? tr('m_bda8d590a9f7') : tr('m_e1ec379a4043') }}
              </button>
              <button type="button" class="link-btn" @click="switchMode('login')">{{ tr('m_31e31254f5f0') }}</button>
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
              <div v-if="oidcProviders.length" class="oidc-login-options" :aria-label="tr('m_ca7af643043a')">
                <p class="field-label">{{ tr('m_ca7af643043a') }}</p>
                <button v-for="provider in oidcProviders" :key="provider.id" type="button" class="btn btn-secondary login-btn" :disabled="loading || passkeyBusy" @click="handleOIDCLogin(provider.id)">{{ tr('m_cdfd0b34e491') }} {{ provider.name }} {{ tr('m_1e2df9c3075a') }}</button>
              </div>
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
import { t as tr } from '../i18n'

import { nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { browserSupportsWebAuthn, startAuthentication } from '@simplewebauthn/browser'
import { completeTwoFactorLogin, login as loginApi } from '../api'
import { getAuthConfig, sendRegisterCode, register, forgotPassword, resetPassword, type AuthConfig } from '../api/admin'
import { beginPasskeyLogin, beginPasskeyTwoFactor, finishPasskeyLogin, finishPasskeyTwoFactor } from '../api/passkey'
import { useConfigStore } from '../stores/config'
import { listOIDCProviders, startOIDCLogin, completeOIDC, sendOIDCCode, type OIDCPublicProvider } from '../api/oidc'
const router = useRouter()
const store = useConfigStore()
const step = ref<'credentials' | 'factor' | 'oidc-email'>('credentials')
const oidcProviders = ref<OIDCPublicProvider[]>([])
const oidcTicket = ref('')
const oidcEmail = ref('')
const oidcCode = ref('')
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
  void listOIDCProviders().then(({ data }) => { if (!disposed) oidcProviders.value = data }).catch(() => {})
  const query = router.currentRoute.value.query
  if (typeof query.oidc_error === 'string') {
    error.value = query.oidc_error
    void router.replace({ path: '/login', query: {} })
  } else if (typeof query.oidc_ticket === 'string') {
    oidcTicket.value = query.oidc_ticket
    void router.replace({ path: '/login', query: {} }).then(() => finishOIDCLogin())
  }
  passkeySupported.value = browserSupportsWebAuthn()
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
    if (!oidcTicket.value && router.currentRoute.value.query.mode === 'register' && data.registration_enabled) {
      mode.value = 'register'
    }
    if (data.turnstile_enabled) nextTick(() => void renderTurnstile())
    if (mode.value === 'register') nextTick(() => document.getElementById('reg-username')?.focus())
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
    // 注册/找回密码页共用一个验证组件，却要服务多个被校验的接口
    // （send-code/register、forgot-password/reset-password）。带页面级
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
  if (passkeyBusy.value) return
  passkeyBusy.value = true
  error.value = ''
  try {
    const { data } = await beginPasskeyLogin(form.username.trim())
    const credential = await startAuthentication({ optionsJSON: data.public_key })
    const response = await finishPasskeyLogin({ ceremony_token: data.ceremony_token, credential })
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
  if (passkeyBusy.value || !challengeToken.value) return
  passkeyBusy.value = true
  error.value = ''
  try {
    const { data } = await beginPasskeyTwoFactor(challengeToken.value)
    const credential = await startAuthentication({ optionsJSON: data.public_key })
    const response = await finishPasskeyTwoFactor({
      challenge_token: challengeToken.value,
      ceremony_token: data.ceremony_token,
      credential,
    })
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
function switchMode(target: 'login' | 'register' | 'forgot') {
  if (mode.value === target) return
  mode.value = target
  oidcTicket.value = ''
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

async function handleOIDCLogin(id: string) {
  if (loading.value || passkeyBusy.value || disposed) return
  const token = takeTurnstileToken()
  if (turnstileEnabled.value && !token) {
    error.value = tr('m_10ef93c129e5')
    void renderTurnstile()
    return
  }
  loading.value = true
  error.value = ''
  try {
    const { data } = await startOIDCLogin(id, token)
    if (!disposed) window.location.assign(data.authorization_url)
  } catch (e: any) {
    if (disposed) return
    error.value = e.response?.data?.error || tr('m_f0473741a7ca')
    loading.value = false
    void renderTurnstile()
  }
}

async function finishOIDCLogin() {
  if (!oidcTicket.value || loading.value || disposed) return
  loading.value = true
  error.value = ''
  try {
    const { data } = await completeOIDC(oidcTicket.value, oidcCode.value.trim())
    if (disposed) return
    if ('email_verification_required' in data) {
      oidcEmail.value = data.email
      step.value = 'oidc-email'
      destroyTurnstile()
      await nextTick()
      document.getElementById('oidc-code')?.focus()
    } else if ('two_factor_required' in data) {
      challengeToken.value = data.challenge_token
      challengeExpiresAt.value = Date.parse(data.expires_at)
      passkeyFactorAvailable.value = !!data.passkeys_available
      factorCode.value = ''
      failedFactorAttempts.value = 0
      oidcTicket.value = ''
      step.value = 'factor'
      destroyTurnstile()
      startChallengeTimer()
      await nextTick()
      factorInput.value?.focus()
    } else if ('token' in data) {
      oidcTicket.value = ''
      store.setAuth(data.token, data.username, data.role)
      await router.replace('/dashboard')
    } else error.value = tr('m_4fe7e7ab747e')
  } catch (e: any) {
    if (!disposed) error.value = e.response?.data?.error || tr('m_349ec35c5154')
  } finally {
    if (!disposed) loading.value = false
  }
}

async function handleOIDCSendCode() {
  if (codeSending.value || codeCooldown.value > 0 || disposed) return
  codeSending.value = true
  error.value = ''
  try {
    await sendOIDCCode(oidcTicket.value)
    if (disposed) return
    codeCooldown.value = 60
    if (cooldownTimer !== undefined) window.clearInterval(cooldownTimer)
    cooldownTimer = window.setInterval(() => {
      codeCooldown.value -= 1
      if (codeCooldown.value <= 0) window.clearInterval(cooldownTimer)
    }, 1000)
  } catch (e: any) {
    if (!disposed) error.value = e.response?.data?.error || tr('m_a98d3ab9e1bb')
  } finally {
    if (!disposed) codeSending.value = false
  }
}

function cancelOIDC() {
  oidcTicket.value = ''
  oidcEmail.value = ''
  oidcCode.value = ''
  error.value = ''
  step.value = 'credentials'
  nextTick(() => { void renderTurnstile(); usernameInput.value?.focus() })
}

async function handleSendCode() {
  const email = regForm.email.trim()
  if (!email) {
    error.value = tr('m_bed85797d991')
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
    error.value = e.response?.data?.error || tr('m_f2d0fada0803')
  } finally {
    codeSending.value = false
    void renderTurnstile()
  }
}

async function handleForgotSend() {
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

async function handleRegister() {
  const username = regForm.username.trim()
  const email = regForm.email.trim()
  if (!username || !email || !regForm.password) {
    error.value = tr('m_194dcb264083')
    triggerShake()
    return
  }
  if (authConfig.value.email_verify_enabled && !regForm.verifyCode.trim()) {
    error.value = tr('m_c25caca78469')
    triggerShake()
    return
  }
  if (authConfig.value.invite_mode === 'required' && !regForm.invite.trim()) {
    error.value = tr('m_9cf670228f66')
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
    error.value = e.response?.data?.error || tr('m_4f07718f79cf') + (e.message || '')
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
.oidc-login-options { display: flex; flex-direction: column; gap: var(--spacing-sm); padding-top: var(--spacing-md); border-top: 1px solid var(--color-hairline); }
.oidc-login-options p { margin: 0; color: var(--color-mute); }
.oidc-login-options .btn { white-space: normal; overflow-wrap: anywhere; }

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
