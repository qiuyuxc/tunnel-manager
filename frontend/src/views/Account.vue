<template>
  <div class="page-container account-page">
    <div class="page-header">
            <h2>账户设置</h2>
      <p>管理登录凭据、Cloudflare 账户连接与账户安全</p>
    </div>

    <div class="settings-list">
      <section class="settings-card security-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.04s' }" aria-labelledby="two-factor-title">
        <div class="security-heading">
          <div>
            <p class="security-kicker">Operations security</p>
            <h3 id="two-factor-title" class="settings-card-title">双重身份验证</h3>
            <p class="settings-card-desc">使用验证器动态口令保护管理操作，即使密码泄露也能阻止直接登录。</p>
          </div>
          <div class="status-stamp" :class="`stamp-${statusStamp.tone}`" role="status" aria-live="polite">
            <span class="stamp-mark" aria-hidden="true"></span>
            <span>
              <small>2FA status</small>
              <strong>{{ statusStamp.label }}</strong>
            </span>
          </div>
        </div>

        <div class="security-rule" aria-hidden="true"></div>

        <div class="security-content" aria-live="polite">
          <div v-if="statusLoading" class="security-state compact-state">
            <span class="state-pulse" aria-hidden="true"></span>
            <div><strong>正在核验安全状态</strong><p>正在读取当前账户的双重验证配置。</p></div>
          </div>

          <div v-else-if="statusError" class="security-state compact-state state-error">
            <div><strong>无法读取双重验证状态</strong><p>{{ statusError }}</p></div>
            <button class="btn btn-secondary" type="button" @click="loadTwoFactorStatus">重试</button>
          </div>

          <div v-else-if="recoveryCodes.length" ref="recoveryPanel" class="recovery-panel" tabindex="-1" aria-labelledby="recovery-title">
            <div class="operation-header">
              <div>
                <span class="operation-index">完成前的最后一步</span>
                <h4 id="recovery-title">保存恢复代码</h4>
              </div>
              <span class="operation-state state-critical">仅显示此次</span>
            </div>
            <p class="operation-copy">验证器不可用时，每个代码可用于一次登录。离开此页前请复制或下载，完成后无法再次查看。</p>
            <div class="recovery-actions">
              <button class="btn btn-secondary" type="button" @click="copyRecoveryCodes">复制全部</button>
              <button class="btn btn-secondary" type="button" @click="downloadRecoveryCodes">下载文本文件</button>
            </div>
            <ol class="recovery-grid" aria-label="恢复代码">
              <li v-for="code in recoveryCodes" :key="code"><code>{{ code }}</code></li>
            </ol>
            <label class="completion-check">
              <input v-model="recoveryAcknowledged" type="checkbox" />
              <span>我已将这些恢复代码保存到安全位置</span>
            </label>
            <button class="btn btn-primary finish-button" type="button" :disabled="!recoveryAcknowledged" @click="finishRecovery">完成并重新登录</button>
          </div>

          <div v-else-if="setupSession" class="setup-panel">
            <div class="operation-header">
              <div>
                <span class="operation-index">Enrollment / 01</span>
                <h4>连接验证器</h4>
              </div>
              <span class="operation-state" :class="setupExpired ? 'state-critical' : 'state-live'">
                {{ setupExpired ? '设置已过期' : `剩余 ${countdownLabel}` }}
              </span>
            </div>

            <div class="setup-grid">
              <div class="qr-column">
                <div class="qr-frame" :class="{ 'qr-error': qrError }">
                  <canvas ref="qrCanvas" role="img" aria-label="双重验证二维码">双重验证二维码无法显示，请使用右侧密钥手动添加。</canvas>
                  <p v-if="qrError">二维码生成失败，请使用右侧密钥手动添加。</p>
                </div>
                <span class="qr-caption">仅在本机生成，不发送至外部服务</span>
              </div>
              <div class="setup-instructions">
                <ol class="instruction-list">
                  <li><span>01</span><p>在验证器应用中选择“扫描二维码”或“输入设置密钥”。</p></li>
                  <li><span>02</span><p>确认账户信息后，输入应用生成的 6 位动态口令。</p></li>
                </ol>
                <div class="secret-block">
                  <span class="field-label">手动设置密钥</span>
                  <div class="secret-row">
                    <code>{{ setupSession.secret }}</code>
                    <button class="text-button" type="button" @click="copySecret">复制密钥</button>
                  </div>
                </div>
                <form class="confirm-form" @submit.prevent="confirmSetup">
                  <label for="confirm-code" class="field-label">6 位动态口令</label>
                  <div class="confirm-row">
                    <input id="confirm-code" ref="confirmCodeInput" v-model="confirmCode" class="vercel-input code-input" inputmode="numeric" autocomplete="one-time-code" maxlength="6" pattern="[0-9]{6}" placeholder="000000" :disabled="confirming || setupExpired" @input="normalizeConfirmCode" />
                    <button class="btn btn-primary" type="submit" :disabled="confirming || setupExpired || confirmCode.length !== 6">
                      {{ confirming ? '验证中...' : '验证并启用' }}
                    </button>
                  </div>
                </form>
              </div>
            </div>
            <button class="text-button cancel-action" type="button" @click="cancelSetup">取消此次设置</button>
          </div>

          <div v-else-if="disableOpen" class="disable-panel">
            <div class="operation-header">
              <div>
                <span class="operation-index">Security override</span>
                <h4>停用双重身份验证</h4>
              </div>
              <span class="operation-state state-critical">高风险操作</span>
            </div>
            <p class="operation-copy">停用后账户将仅由密码保护。请输入当前密码，以及验证器动态口令或一枚恢复代码。</p>
            <form class="disable-form" @submit.prevent="submitDisable">
              <label>
                <span class="field-label">当前密码</span>
                <input ref="disablePasswordInput" v-model="disablePassword" class="vercel-input" type="password" autocomplete="current-password" placeholder="输入当前密码" />
              </label>
              <label>
                <span class="field-label">动态口令或恢复代码</span>
                <input ref="disableCodeInput" v-model="disableCode" class="vercel-input" type="text" autocomplete="one-time-code" placeholder="输入验证代码" />
              </label>
              <div class="form-actions">
                <button class="btn btn-secondary" type="button" :disabled="disabling" @click="cancelDisable">取消</button>
                <button class="btn btn-danger" type="submit" :disabled="disabling || !disablePassword || !disableCode.trim()">
                  {{ disabling ? '停用中...' : '确认停用并退出' }}
                </button>
              </div>
            </form>
          </div>

          <div v-else-if="twoFactorStatus?.enabled" class="security-state enabled-state">
            <div class="status-copy">
              <strong>双重身份验证正在保护此账户</strong>
              <p>登录时需要密码与第二重凭据。当前还有 <b>{{ twoFactorStatus.recovery_codes_remaining }}</b> 枚恢复代码可用。</p>
            </div>
            <button class="btn btn-secondary danger-outline" type="button" @click="openDisable">停用 2FA</button>
          </div>

          <div v-else-if="twoFactorStatus && !twoFactorStatus.setup_available" class="security-state unavailable-state">
            <div class="status-copy">
              <strong>当前无法开始设置</strong>
              <p>后端尚未配置 <code>APP_ENCRYPTION_KEY</code>。请设置密钥并重启后端，现有用户名和密码功能不受影响。</p>
            </div>
          </div>

          <div v-else class="security-state disabled-state">
            <div class="status-copy">
              <strong>账户目前仅由密码保护</strong>
              <p v-if="setupNotice" class="setup-notice" role="status">{{ setupNotice }}</p>
              <p v-else>连接任意兼容 TOTP 的验证器应用，设置过程不会将密钥发送到外部二维码服务。</p>
            </div>
            <button class="btn btn-primary" type="button" :disabled="startingSetup" @click="startSetup">
              {{ startingSetup ? '准备中...' : '设置双重验证' }}
            </button>
          </div>
        </div>
      </section>

      <section class="settings-card cloudflare-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.08s' }" aria-labelledby="cloudflare-title">
        <div class="cloudflare-heading">
          <div>
            <p class="security-kicker">Account connection</p>
            <h3 id="cloudflare-title" class="settings-card-title">Cloudflare 账户连接</h3>
            <p class="settings-card-desc">通过 OAuth 管理 Cloudflare 账户凭据。OAuth 连接存在时始终优先用于账户、隧道与 Zone 请求。</p>
          </div>
          <span class="connection-badge" :class="cloudflare.connected ? 'connected' : 'disconnected'">
            {{ loadingCloudflare ? '读取中' : cloudflare.connected ? '已连接' : '未连接' }}
          </span>
        </div>

        <div v-if="loadingCloudflare" class="cloudflare-loading" role="status">
          <span class="state-pulse" aria-hidden="true"></span>
          <span>正在读取 Cloudflare 连接状态</span>
        </div>

        <div v-else class="cloudflare-panel">
          <div class="cloudflare-status">
            <div>
              <span class="cloudflare-label">凭据来源</span>
              <strong>{{ cloudflareSourceLabel }}</strong>
            </div>
            <div>
              <span class="cloudflare-label">当前账户</span>
              <strong>{{ cloudflare.account_name || '尚未选择' }}</strong>
              <code v-if="cloudflare.account_id">{{ cloudflare.account_id }}</code>
            </div>
            <div>
              <span class="cloudflare-label">OAuth 有效期</span>
              <strong>{{ cloudflare.expires_at ? formatOAuthExpiry(cloudflare.expires_at) : '不适用' }}</strong>
            </div>
          </div>

          <label v-if="cloudflare.source === 'oauth' && cloudflare.accounts?.length" class="account-selector">
            <span class="field-label">切换 Cloudflare 账户</span>
            <select class="vercel-input" :value="cloudflare.account_id" :disabled="selectingAccount" @change="changeCloudflareAccount">
              <option v-for="account in cloudflare.accounts" :key="account.id" :value="account.id">
                {{ account.name }} · {{ account.id }}
              </option>
            </select>
          </label>

          <p v-if="cloudflareStatusError || cloudflare.error" class="cloudflare-error" role="alert">
            {{ cloudflareStatusError || cloudflare.error }}
          </p>

          <div v-if="!cloudflare.configured" class="oauth-setup-hint">
            <strong>服务器尚未配置 OAuth 客户端</strong>
            <span>请设置 <code>CF_OAUTH_CLIENT_ID</code>、<code>CF_OAUTH_CLIENT_SECRET</code> 与有效的 <code>APP_ENCRYPTION_KEY</code>。</span>
          </div>

          <div class="cloudflare-actions">
            <button v-if="cloudflare.configured" class="btn btn-primary" type="button" :disabled="startingOAuth" @click="connectCloudflare">
              {{ startingOAuth ? '正在跳转...' : cloudflare.source === 'oauth' ? '重新授权' : '连接 Cloudflare' }}
            </button>
            <button v-if="cloudflare.source === 'oauth'" class="btn btn-secondary danger-outline" type="button" :disabled="disconnectingOAuth" @click="confirmDisconnectCloudflare">
              {{ disconnectingOAuth ? '断开中...' : '断开授权' }}
            </button>
            <button v-if="cloudflareStatusError" class="btn btn-secondary" type="button" @click="loadCloudflareStatus">重试</button>
            <span v-if="cloudflare.source === 'api_token'" class="legacy-note">当前使用环境变量 API Token；完成 OAuth 授权后会自动优先使用 OAuth。</span>
          </div>
        </div>
      </section>

      <div class="settings-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.12s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">修改用户名</div>
          <div class="settings-card-desc">当前用户名: <strong>{{ store.username }}</strong></div>
        </div>
        <div class="form-stack">
          <label class="field">
            <span class="field-label">新用户名</span>
            <input v-model="newUsername" placeholder="输入新用户名" class="vercel-input" />
          </label>
          <label class="field">
            <span class="field-label">当前密码确认</span>
            <input v-model="usernamePassword" type="password" placeholder="输入当前密码确认" class="vercel-input" />
          </label>
          <div class="form-actions">
            <button class="btn btn-secondary" :disabled="savingUsername" @click="saveUsername">
              {{ savingUsername ? '保存中...' : '更新用户名' }}
            </button>
          </div>
        </div>
      </div>

      <div class="settings-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.16s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">修改密码</div>
          <div class="settings-card-desc">密码长度不少于 6 位</div>
        </div>
        <div class="form-stack">
          <label class="field">
            <span class="field-label">当前密码</span>
            <input v-model="currentPassword" type="password" placeholder="当前密码" class="vercel-input" />
          </label>
          <label class="field">
            <span class="field-label">新密码</span>
            <input v-model="newPassword" type="password" placeholder="新密码" class="vercel-input" />
          </label>
          <div class="form-actions">
            <button class="btn btn-primary" :disabled="savingPassword" @click="savePassword">
              {{ savingPassword ? '保存中...' : '更新密码' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { useDialog, useMessage } from 'naive-ui'
import QRCode from 'qrcode'
import {
  changePassword,
  changeUsername,
  confirmTwoFactor,
  disconnectCloudflareOAuth,
  disableTwoFactor,
  getCloudflareOAuthStatus,
  getTwoFactorStatus,
  selectCloudflareAccount,
  setupTwoFactor,
  startCloudflareOAuth,
  type CloudflareOAuthStatus,
  type TOTPSetupResponse,
  type TOTPStatusResponse,
} from '../api'
import { useConfigStore } from '../stores/config'

const message = useMessage()
const dialog = useDialog()
const route = useRoute()
const router = useRouter()
const store = useConfigStore()
const visible = ref(false)

const newUsername = ref('')
const usernamePassword = ref('')
const currentPassword = ref('')
const newPassword = ref('')
const savingUsername = ref(false)
const savingPassword = ref(false)

const loadingCloudflare = ref(true)
const startingOAuth = ref(false)
const disconnectingOAuth = ref(false)
const selectingAccount = ref(false)
const cloudflareStatusError = ref('')
const cloudflare = reactive<CloudflareOAuthStatus>({
  configured: false,
  connected: false,
  source: 'none',
  account_id: '',
  account_name: '',
  accounts: [],
  redirect_uri: '',
})

const statusLoading = ref(true)
const statusError = ref('')
const twoFactorStatus = ref<TOTPStatusResponse | null>(null)
const startingSetup = ref(false)
const setupSession = ref<TOTPSetupResponse | null>(null)
const setupNotice = ref('')
const confirmCode = ref('')
const confirming = ref(false)
const qrCanvas = ref<HTMLCanvasElement | null>(null)
const qrError = ref(false)
const expiresAtMs = ref(0)
const nowMs = ref(Date.now())
const recoveryCodes = ref<string[]>([])
const recoveryAcknowledged = ref(false)
const recoveryPanel = ref<HTMLElement | null>(null)
const confirmCodeInput = ref<HTMLInputElement | null>(null)
const disablePasswordInput = ref<HTMLInputElement | null>(null)
const disableCodeInput = ref<HTMLInputElement | null>(null)
const disableOpen = ref(false)
const disablePassword = ref('')
const disableCode = ref('')
const disabling = ref(false)
let countdownTimer: ReturnType<typeof setInterval> | undefined
let visibleFrame: number | undefined
let disposed = false
let statusRequestGeneration = 0
let setupRequestGeneration = 0
let confirmRequestGeneration = 0
let oauthRequestGeneration = 0

const setupExpired = computed(() => Boolean(setupSession.value) && expiresAtMs.value <= nowMs.value)
const countdownLabel = computed(() => {
  const seconds = Math.max(0, Math.ceil((expiresAtMs.value - nowMs.value) / 1000))
  const minutes = Math.floor(seconds / 60).toString().padStart(2, '0')
  return `${minutes}:${(seconds % 60).toString().padStart(2, '0')}`
})

const statusStamp = computed(() => {
  if (statusLoading.value) return { label: '读取中', tone: 'neutral' }
  if (statusError.value) return { label: '状态错误', tone: 'error' }
  if (recoveryCodes.value.length) return { label: '恢复代码待保存', tone: 'warning' }
  if (setupSession.value) return { label: setupExpired.value ? '设置已过期' : '设置进行中', tone: setupExpired.value ? 'error' : 'warning' }
  if (disableOpen.value) return { label: '停用确认中', tone: 'error' }
  if (twoFactorStatus.value?.enabled) return { label: '已启用', tone: 'success' }
  if (twoFactorStatus.value && !twoFactorStatus.value.setup_available) return { label: '设置不可用', tone: 'neutral' }
  return { label: '未启用', tone: 'neutral' }
})

const cloudflareSourceLabel = computed(() => {
  if (cloudflare.source === 'oauth') return 'Cloudflare OAuth 2.0'
  if (cloudflare.source === 'api_token') return '环境变量 API Token'
  return '未配置'
})

function apiError(error: any, fallback: string) {
  return error.response?.data?.error || error.message || fallback
}

async function loadCloudflareStatus() {
  const generation = ++oauthRequestGeneration
  loadingCloudflare.value = true
  cloudflareStatusError.value = ''
  try {
    const { data } = await getCloudflareOAuthStatus()
    if (disposed || generation !== oauthRequestGeneration) return
    Object.assign(cloudflare, {
      configured: false,
      connected: false,
      source: 'none',
      account_id: '',
      account_name: '',
      accounts: [],
      expires_at: undefined,
      redirect_uri: '',
      error: undefined,
    }, data, { accounts: data.accounts || [] })
  } catch (error: any) {
    if (disposed || generation !== oauthRequestGeneration) return
    cloudflareStatusError.value = apiError(error, '请稍后重试')
    message.error('读取 Cloudflare 状态失败: ' + cloudflareStatusError.value)
  } finally {
    if (!disposed && generation === oauthRequestGeneration) loadingCloudflare.value = false
  }
}

async function connectCloudflare() {
  if (startingOAuth.value) return
  startingOAuth.value = true
  try {
    const { data } = await startCloudflareOAuth()
    window.location.assign(data.authorization_url)
  } catch (error: any) {
    startingOAuth.value = false
    message.error('启动 OAuth 失败: ' + apiError(error, '请稍后重试'))
  }
}

async function disconnectCloudflare() {
  if (disconnectingOAuth.value) return
  disconnectingOAuth.value = true
  try {
    const { data } = await disconnectCloudflareOAuth()
    if (data.warning) message.warning(data.warning)
    else message.success('Cloudflare OAuth 已断开')
    await loadCloudflareStatus()
  } catch (error: any) {
    message.error('断开失败: ' + apiError(error, '请稍后重试'))
  } finally {
    disconnectingOAuth.value = false
  }
}

function confirmDisconnectCloudflare() {
  if (disconnectingOAuth.value) return
  dialog.warning({
    title: '断开 Cloudflare OAuth',
    content: '断开后将撤销并删除当前 OAuth 凭据，已选择的 Cloudflare 账户与隧道也会被清除。',
    positiveText: '确认断开',
    negativeText: '取消',
    positiveButtonProps: { type: 'error' },
    onPositiveClick: disconnectCloudflare,
  })
}

async function changeCloudflareAccount(event: Event) {
  const accountID = (event.target as HTMLSelectElement).value
  if (!accountID || selectingAccount.value || accountID === cloudflare.account_id) return
  selectingAccount.value = true
  try {
    const { data } = await selectCloudflareAccount(accountID)
    cloudflare.account_id = data.id
    cloudflare.account_name = data.name
    message.success(`已切换到 ${data.name}`)
  } catch (error: any) {
    message.error('切换账户失败: ' + apiError(error, '请稍后重试'))
    await loadCloudflareStatus()
  } finally {
    selectingAccount.value = false
  }
}

function formatOAuthExpiry(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '自动刷新' : date.toLocaleString()
}

async function handleCloudflareOAuthResult() {
  const result = typeof route.query.cloudflare_oauth === 'string' ? route.query.cloudflare_oauth : ''
  if (!result) return
  const oauthMessage = typeof route.query.message === 'string' ? route.query.message : ''
  const query = { ...route.query }
  delete query.cloudflare_oauth
  delete query.message
  await router.replace({ path: route.path, query })
  if (result === 'success') message.success('Cloudflare OAuth 授权成功')
  if (result === 'error') message.error('Cloudflare OAuth 授权失败: ' + (oauthMessage || '未知错误'))
}

async function loadTwoFactorStatus() {
  const generation = ++statusRequestGeneration
  statusLoading.value = true
  statusError.value = ''
  try {
    const { data } = await getTwoFactorStatus()
    if (disposed || generation !== statusRequestGeneration) return
    twoFactorStatus.value = data
  } catch (error: any) {
    if (disposed || generation !== statusRequestGeneration) return
    statusError.value = apiError(error, '请稍后重试')
  } finally {
    if (!disposed && generation === statusRequestGeneration) statusLoading.value = false
  }
}

function clearCountdown() {
  if (countdownTimer) clearInterval(countdownTimer)
  countdownTimer = undefined
}

function beginCountdown(expiresAt: number) {
  clearCountdown()
  expiresAtMs.value = expiresAt
  nowMs.value = Date.now()
  countdownTimer = setInterval(() => {
    if (disposed) return
    nowMs.value = Date.now()
    if (expiresAtMs.value <= nowMs.value) expireSetup()
  }, 1000)
}

function expireSetup() {
  ++setupRequestGeneration
  clearSetup()
  setupNotice.value = '设置会话已过期，密钥已清除。请重新开始设置。'
}

async function renderQr(generation: number) {
  qrError.value = false
  await nextTick()
  if (disposed || generation !== setupRequestGeneration || !qrCanvas.value || !setupSession.value) return
  try {
    await QRCode.toCanvas(qrCanvas.value, setupSession.value.otpauth_uri, {
      width: 220,
      margin: 2,
      errorCorrectionLevel: 'M',
      color: { dark: '#000000', light: '#ffffff' },
    })
  } catch {
    if (!disposed && generation === setupRequestGeneration) qrError.value = true
  }
}

async function startSetup() {
  const generation = ++setupRequestGeneration
  startingSetup.value = true
  setupNotice.value = ''
  try {
    const { data } = await setupTwoFactor()
    if (disposed || generation !== setupRequestGeneration) return
    const expiresAt = Date.parse(data.expires_at)
    if (!Number.isFinite(expiresAt) || expiresAt <= Date.now()) {
      clearSetup()
      setupNotice.value = '服务器返回的设置会话无效或已过期，请重试。'
      message.error(setupNotice.value)
      return
    }
    setupSession.value = data
    confirmCode.value = ''
    beginCountdown(expiresAt)
    await renderQr(generation)
    await nextTick()
    if (!disposed && generation === setupRequestGeneration) confirmCodeInput.value?.focus()
  } catch (error: any) {
    if (disposed || generation !== setupRequestGeneration) return
    message.error('无法开始设置: ' + apiError(error, '请稍后重试'))
    await loadTwoFactorStatus()
  } finally {
    if (!disposed && generation === setupRequestGeneration) startingSetup.value = false
  }
}

function clearSetup() {
  clearCountdown()
  setupSession.value = null
  confirmCode.value = ''
  expiresAtMs.value = 0
  qrError.value = false
  if (qrCanvas.value) {
    const context = qrCanvas.value.getContext('2d')
    context?.clearRect(0, 0, qrCanvas.value.width, qrCanvas.value.height)
  }
}

function cancelSetup() {
  ++setupRequestGeneration
  clearSetup()
  setupNotice.value = ''
}

function normalizeConfirmCode() {
  confirmCode.value = confirmCode.value.replace(/\D/g, '').slice(0, 6)
}

async function confirmSetup() {
  if (!setupSession.value || !/^\d{6}$/.test(confirmCode.value) || setupExpired.value) return
  const generation = ++confirmRequestGeneration
  const setupToken = setupSession.value.setup_token
  const code = confirmCode.value
  confirming.value = true
  try {
    const { data } = await confirmTwoFactor(setupToken, code)
    if (disposed || generation !== confirmRequestGeneration) return
    recoveryCodes.value = [...data.recovery_codes]
    recoveryAcknowledged.value = false
    twoFactorStatus.value = { enabled: data.enabled, recovery_codes_remaining: data.recovery_codes.length, setup_available: false }
    clearSetup()
    store.clearAuth()
    await nextTick()
    if (disposed || generation !== confirmRequestGeneration) return
    recoveryPanel.value?.focus()
    message.success('双重身份验证已启用，请先保存恢复代码')
  } catch (error: any) {
    if (disposed || generation !== confirmRequestGeneration) return
    message.error('验证失败: ' + apiError(error, '请检查动态口令'))
    confirmCodeInput.value?.focus()
    confirmCodeInput.value?.select()
  } finally {
    if (!disposed && generation === confirmRequestGeneration) confirming.value = false
  }
}

async function copyText(value: string, successText: string) {
  try {
    await navigator.clipboard.writeText(value)
    message.success(successText)
  } catch {
    message.error('复制失败，请手动选择并复制')
  }
}

function copySecret() {
  if (setupSession.value) void copyText(setupSession.value.secret, '设置密钥已复制')
}

function copyRecoveryCodes() {
  void copyText(recoveryCodes.value.join('\n'), '恢复代码已复制')
}

function downloadRecoveryCodes() {
  const blob = new Blob([recoveryCodes.value.join('\n') + '\n'], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'tunnel-manager-recovery-codes.txt'
  link.click()
  window.setTimeout(() => URL.revokeObjectURL(url), 0)
}

async function finishRecovery() {
  if (!recoveryAcknowledged.value) return
  recoveryCodes.value = []
  recoveryAcknowledged.value = false
  await router.replace('/login')
}

function openDisable() {
  disableOpen.value = true
  disablePassword.value = ''
  disableCode.value = ''
  nextTick(() => disablePasswordInput.value?.focus())
}

function cancelDisable() {
  disableOpen.value = false
  disablePassword.value = ''
  disableCode.value = ''
}

async function submitDisable() {
  if (!disablePassword.value || !disableCode.value.trim()) return
  disabling.value = true
  try {
    await disableTwoFactor(disablePassword.value, disableCode.value.trim())
    cancelDisable()
    store.clearAuth()
    message.success('双重身份验证已停用，请重新登录')
    await router.replace('/login')
  } catch (error: any) {
    message.error('停用失败: ' + apiError(error, '请检查密码与验证代码'))
    if (error.response?.status === 401) {
      disableCode.value = ''
      await nextTick()
      disableCodeInput.value?.focus()
    }
  } finally {
    disabling.value = false
  }
}

function handleBeforeUnload(event: BeforeUnloadEvent) {
  if (!confirming.value && !recoveryCodes.value.length) return
  event.preventDefault()
  event.returnValue = ''
}

onBeforeRouteLeave(() => {
  if (confirming.value || recoveryCodes.value.length) return false
})

async function saveUsername() {
  if (!newUsername.value || !usernamePassword.value) {
    message.error('请填写新用户名和当前密码')
    return
  }
  savingUsername.value = true
  try {
    await changeUsername(usernamePassword.value, newUsername.value)
    store.clearAuth()
    message.success('用户名已更新，请重新登录')
    await router.replace('/login')
  } catch (e: any) {
    message.error('更新失败: ' + (e.response?.data?.error || e.message))
  } finally {
    savingUsername.value = false
  }
}

async function savePassword() {
  if (!currentPassword.value || !newPassword.value) {
    message.error('请填写当前密码和新密码')
    return
  }
  if (newPassword.value.length < 6) {
    message.error('新密码长度不能少于 6 位')
    return
  }
  savingPassword.value = true
  try {
    await changePassword(currentPassword.value, newPassword.value)
    store.clearAuth()
    message.success('密码已更新，请重新登录')
    await router.replace('/login')
  } catch (e: any) {
    message.error('更新失败: ' + (e.response?.data?.error || e.message))
  } finally {
    savingPassword.value = false
  }
}

onMounted(() => {
  window.addEventListener('beforeunload', handleBeforeUnload)
  visibleFrame = requestAnimationFrame(() => {
    visibleFrame = undefined
    if (!disposed) visible.value = true
  })
  void loadTwoFactorStatus()
  void loadCloudflareStatus()
  void handleCloudflareOAuthResult()
})

onBeforeUnmount(() => {
  disposed = true
  ++statusRequestGeneration
  ++setupRequestGeneration
  ++oauthRequestGeneration
  window.removeEventListener('beforeunload', handleBeforeUnload)
  if (visibleFrame !== undefined) cancelAnimationFrame(visibleFrame)
  clearCountdown()
  clearSetup()
  recoveryCodes.value = []
  disablePassword.value = ''
  disableCode.value = ''
})
</script>

<style scoped>
.page-header { margin-bottom: var(--spacing-lg); }

.settings-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.settings-card {
  min-width: 0;
  background: var(--color-canvas);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
  box-shadow: 0 1px 2px rgba(58, 47, 34, 0.05);
}
.settings-card:hover {
  border-color: var(--color-hairline-strong);
  box-shadow: 0 12px 28px rgba(58, 47, 34, 0.08);
}

.settings-card-header { margin-bottom: var(--spacing-md); }
.settings-card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-ink);
  margin: 0 0 4px;
}
.settings-card-desc {
  font-size: 14px;
  color: var(--color-mute);
  line-height: 1.65;
  margin: 0;
}

.cloudflare-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--spacing-lg);
  margin-bottom: var(--spacing-md);
}
.cloudflare-heading .settings-card-title { font-size: 19px; }
.cloudflare-heading .settings-card-desc { max-width: 680px; }
.connection-badge { display: inline-flex; flex: none; align-items: center; padding: 6px 10px; border-radius: 999px; font-size: 12px; font-weight: 700; }
.connection-badge.connected { color: var(--color-success); background: var(--color-result-success-bg); }
.connection-badge.disconnected { color: var(--color-mute); background: var(--color-canvas-soft); }
.cloudflare-loading { display: flex; align-items: center; gap: var(--spacing-sm); min-height: 72px; color: var(--color-mute); font-size: 14px; }
.cloudflare-panel { display: flex; min-width: 0; flex-direction: column; gap: var(--spacing-md); }
.cloudflare-status { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: var(--spacing-sm); }
.cloudflare-status > div { display: flex; min-width: 0; flex-direction: column; gap: 4px; padding: var(--spacing-md); border: 1px solid var(--color-hairline); border-radius: var(--radius-md); background: var(--color-canvas-soft); }
.cloudflare-label { color: var(--color-mute); font-family: var(--font-mono); font-size: 10px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; }
.cloudflare-status strong { color: var(--color-ink); font-size: 14px; overflow-wrap: anywhere; }
.cloudflare-status code, .oauth-setup-hint code { color: var(--color-mute); font: 12px/1.5 var(--font-mono); overflow-wrap: anywhere; word-break: break-all; }
.account-selector { display: flex; min-width: 0; max-width: 680px; flex-direction: column; gap: 5px; }
.account-selector select { width: 100%; min-width: 0; }
.cloudflare-error { margin: 0; padding: var(--spacing-sm) var(--spacing-md); color: var(--color-error); border-radius: var(--radius-md); background: var(--color-result-error-bg); font-size: 13px; line-height: 1.6; overflow-wrap: anywhere; }
.oauth-setup-hint { display: flex; min-width: 0; flex-direction: column; gap: 5px; padding: var(--spacing-md); border: 1px dashed var(--color-hairline-strong); border-radius: var(--radius-md); color: var(--color-mute); font-size: 13px; overflow-wrap: anywhere; }
.oauth-setup-hint strong { color: var(--color-ink); }
.cloudflare-actions { display: flex; align-items: center; flex-wrap: wrap; gap: var(--spacing-sm); }
.legacy-note { min-width: 0; color: var(--color-mute); font-size: 12px; line-height: 1.6; overflow-wrap: anywhere; }

.security-card {
  padding: 0;
  overflow: hidden;
  border-color: var(--color-hairline-strong);
}

.security-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--spacing-lg);
  padding: 20px var(--spacing-lg) 18px;
}

.security-kicker,
.operation-index,
.field-label,
.qr-caption {
  display: block;
  margin: 0 0 5px;
  color: var(--color-mute);
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  line-height: 1.4;
  letter-spacing: 0.09em;
  text-transform: uppercase;
}

.security-heading .settings-card-title { font-size: 19px; }
.security-heading .settings-card-desc { max-width: 620px; }
.security-rule { height: 1px; background: var(--color-hairline); }
.security-content { min-height: 92px; }
.security-content > * { animation: security-fade 160ms ease-out both; }

.status-stamp {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 9px;
  min-width: 150px;
  padding: 10px 12px;
  border: 1px solid var(--color-hairline-strong);
  border-radius: var(--radius-sm);
  background: var(--color-canvas-soft);
  transform: rotate(-0.6deg);
}
.status-stamp small,
.status-stamp strong { display: block; line-height: 1.2; }
.status-stamp small {
  margin-bottom: 3px;
  font-family: var(--font-mono);
  font-size: 9px;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  opacity: 0.72;
}
.status-stamp strong { font-size: 13px; }
.stamp-mark { width: 8px; height: 8px; border-radius: 50%; background: currentColor; box-shadow: 0 0 0 3px color-mix(in srgb, currentColor 16%, transparent); }
.stamp-success { color: var(--color-status-healthy-text); border-color: var(--color-status-healthy-border); background: var(--color-status-healthy-bg); }
.stamp-warning { color: var(--color-status-degraded-text); border-color: var(--color-status-degraded-border); background: var(--color-status-degraded-bg); }
.stamp-error { color: var(--color-status-down-text); border-color: var(--color-status-down-border); background: var(--color-status-down-bg); }
.stamp-neutral { color: var(--color-body); }

.security-state {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-lg);
  padding: var(--spacing-md) var(--spacing-lg);
}
.security-state strong { color: var(--color-ink); font-size: 14px; }
.security-state p,
.operation-copy { margin: 3px 0 0; color: var(--color-body); font-size: 13px; line-height: 1.65; }
.security-state b { color: var(--color-ink); font-family: var(--font-mono); }
.status-copy { min-width: 0; }
.compact-state { justify-content: flex-start; }
.compact-state .btn { margin-left: auto; }
.state-error strong { color: var(--color-error); }
.state-pulse { flex: 0 0 auto; width: 9px; height: 9px; border-radius: 50%; background: var(--color-warning); animation: pulse 1.4s ease-in-out infinite; }

.operation-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--spacing-md);
  margin-bottom: var(--spacing-sm);
}
.operation-header h4 { margin: 0; color: var(--color-ink); font-size: 17px; }
.operation-state {
  flex: 0 0 auto;
  padding: 4px 7px;
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}
.state-live { color: var(--color-status-healthy-text); border-color: var(--color-status-healthy-border); background: var(--color-status-healthy-bg); }
.state-critical { color: var(--color-status-down-text); border-color: var(--color-status-down-border); background: var(--color-status-down-bg); }

.setup-panel,
.disable-panel,
.recovery-panel { padding: 20px var(--spacing-lg) var(--spacing-lg); }
.setup-grid {
  display: grid;
  grid-template-columns: minmax(220px, 0.72fr) minmax(0, 1.6fr);
  gap: var(--spacing-lg);
  margin-top: var(--spacing-md);
}
.qr-column { display: flex; flex-direction: column; align-items: center; justify-content: flex-start; min-width: 0; }
.qr-frame {
  display: grid;
  place-items: center;
  width: min(100%, 220px);
  aspect-ratio: 1;
  overflow: hidden;
  border: 1px solid var(--color-hairline-strong);
  border-radius: var(--radius-md);
  background: #fff;
}
.qr-frame canvas { display: block; width: 100% !important; height: auto !important; max-width: 220px; }
.qr-frame p { padding: var(--spacing-sm); color: #7e271c; font-size: 12px; text-align: center; }
.qr-error canvas { display: none; }
.qr-caption { margin: 8px 0 0; text-align: center; text-transform: none; letter-spacing: 0.02em; }
.setup-instructions { min-width: 0; }
.instruction-list { display: grid; gap: 10px; margin: 0 0 var(--spacing-md); padding: 0; list-style: none; }
.instruction-list li { display: grid; grid-template-columns: 28px 1fr; align-items: start; gap: 9px; }
.instruction-list span { color: var(--color-link); font-family: var(--font-mono); font-size: 11px; font-weight: 700; line-height: 1.7; }
.instruction-list p { margin: 0; color: var(--color-body); font-size: 13px; line-height: 1.7; }
.secret-block { padding: 10px 12px; border: 1px solid var(--color-hairline); border-radius: var(--radius-md); background: var(--color-canvas-soft); }
.secret-row { display: flex; align-items: center; justify-content: space-between; gap: var(--spacing-sm); }
.secret-row code { min-width: 0; color: var(--color-ink); font-family: var(--font-mono); font-size: 13px; font-weight: 700; overflow-wrap: anywhere; word-break: break-all; }
.text-button { appearance: none; padding: 0; border: 0; background: transparent; color: var(--color-link); font-size: 12px; font-weight: 700; cursor: pointer; }
.text-button:hover { text-decoration: underline; }
.confirm-form { margin-top: var(--spacing-md); }
.confirm-row { display: flex; gap: var(--spacing-sm); align-items: stretch; }
.code-input { font-family: var(--font-mono); font-size: 16px; letter-spacing: 0.18em; }
.cancel-action { margin-top: var(--spacing-md); }

.disable-form { display: grid; grid-template-columns: 1fr 1fr; gap: var(--spacing-sm); margin-top: var(--spacing-md); }
.disable-form label { min-width: 0; }
.form-actions { display: flex; grid-column: 1 / -1; justify-content: flex-end; gap: var(--spacing-sm); margin-top: 4px; }
.btn-danger { color: var(--color-status-down-text); border: 1px solid var(--color-status-down-border); background: var(--color-status-down-bg); }
.btn-danger:hover:not(:disabled) { border-color: var(--color-error); }
.danger-outline { color: var(--color-error); }

.recovery-panel { outline: none; }
.recovery-panel:focus {
  outline: 3px solid color-mix(in srgb, var(--color-link) 42%, transparent);
  outline-offset: -3px;
}
.recovery-actions { display: flex; gap: var(--spacing-xs); margin: var(--spacing-md) 0; }
.recovery-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 7px var(--spacing-sm);
  margin: 0;
  padding: var(--spacing-md) var(--spacing-md) var(--spacing-md) 42px;
  border: 1px solid var(--color-hairline-strong);
  border-radius: var(--radius-md);
  background: var(--color-canvas-soft);
}
.recovery-grid li { padding-left: 3px; color: var(--color-mute); font-family: var(--font-mono); font-size: 11px; }
.recovery-grid code { color: var(--color-ink); font-size: 13px; font-weight: 700; overflow-wrap: anywhere; }
.completion-check { display: flex; align-items: flex-start; gap: 9px; margin: var(--spacing-md) 0 var(--spacing-sm); color: var(--color-body); font-size: 13px; cursor: pointer; }
.completion-check input { flex: 0 0 auto; margin-top: 4px; accent-color: var(--color-link); }
.finish-button { min-width: 210px; }

.settings-input-row {
  display: flex;
  gap: var(--spacing-sm);
  align-items: flex-start;
}
.settings-action-row { margin-top: var(--spacing-sm); }
.input-wrapper { flex: 1; min-width: 0; }

@keyframes pulse { 50% { opacity: 0.35; transform: scale(0.75); } }
@keyframes security-fade { from { opacity: 0; transform: translateY(3px); } to { opacity: 1; transform: translateY(0); } }

@media (prefers-reduced-motion: reduce) {
  .state-pulse { animation: none; }
  .status-stamp { transform: none; }
  .settings-card { transition-duration: 1ms; }
  .security-content > * { animation-duration: 1ms; }
}

@media (max-width: 768px) {
  .security-heading,
  .security-state,
  .cloudflare-heading { align-items: stretch; flex-direction: column; gap: var(--spacing-sm); }
  .status-stamp { align-self: flex-start; min-width: 0; }
  .connection-badge { align-self: flex-start; }
  .cloudflare-status { grid-template-columns: 1fr; }
  .cloudflare-actions { align-items: stretch; flex-direction: column; }
  .cloudflare-actions .btn { width: 100%; max-width: 100%; justify-content: center; }
  .security-state .btn,
  .compact-state .btn { width: 100%; max-width: 100%; margin-left: 0; justify-content: center; }
  .setup-grid { grid-template-columns: 1fr; }
  .confirm-row,
  .disable-form { grid-template-columns: 1fr; flex-direction: column; }
  .confirm-row .btn,
  .form-actions .btn,
  .recovery-actions .btn,
  .finish-button { width: 100%; max-width: 100%; justify-content: center; }
  .form-actions { flex-direction: column; }
  .recovery-actions { flex-direction: column; }
  .recovery-grid { grid-template-columns: 1fr; }
  .settings-input-row { flex-direction: column; }
  .settings-input-row .btn { width: 100%; max-width: 100%; justify-content: center; }
}

@media (max-width: 420px) {
  .account-page { padding-left: var(--spacing-md); padding-right: var(--spacing-md); }
  .security-heading,
  .security-state,
  .setup-panel,
  .disable-panel,
  .recovery-panel,
  .settings-card:not(.security-card) { padding-left: var(--spacing-md); padding-right: var(--spacing-md); }
  .operation-header { flex-direction: column; }
  .secret-row { align-items: flex-start; flex-direction: column; }
  .recovery-grid { padding-left: 34px; }
}
</style>
