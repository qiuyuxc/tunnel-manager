<template>
  <div class="page-container">
    <div class="page-header">
            <h2>TG 机器人设置</h2>
      <p>每人一个独立 Bot，远程管理自己的隧道、DNS 与域名，账户之间互相隔离</p>
    </div>
    <div class="settings-list section">
      <!-- Status card -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.08s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">Bot 状态</div>
        </div>
        <div class="status-row">
          <div class="status-left">
            <span class="status-dot" :class="status.running ? 'on' : 'off'" />
            <span class="status-text">
              <template v-if="status.running">
                运行中 @{{ status.bot_username }} · {{ modeLabel }}
              </template>
              <template v-else>
                已停止
              </template>
            </span>
          </div>
          <button class="btn btn-ghost btn-sm" @click="fetchStatus">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
            刷新
          </button>
        </div>
        <div v-if="status.last_error" class="status-error">{{ status.last_error }}</div>
        <div v-if="status.last_update_at" class="status-meta">最近更新: {{ formatTime(status.last_update_at) }}</div>
      </div>
      <!-- Enable toggle -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.16s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">启用 Bot</div>
          <div class="settings-card-desc">开启后 Bot 将在后台运行（长轮询模式），或注册 Webhook 接收消息。</div>
        </div>
        <div class="toggle-row">
          <label class="switch">
            <input type="checkbox" v-model="settings.enabled" />
            <span class="switch-slider" />
          </label>
          <span class="toggle-label">{{ settings.enabled ? '已启用' : '已禁用' }}</span>
        </div>
      </div>
      <!-- Mode: polling / webhook -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.20s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">运行模式</div>
          <div class="settings-card-desc">长轮询无需公网入口；Webhook 需要面板有可公网访问的 HTTPS 地址，消息推送更及时。</div>
        </div>
        <div class="radio-group">
          <label class="radio-option" :class="{ active: settings.mode === 'polling' }">
            <input type="radio" value="polling" v-model="settings.mode" />
            <span>长轮询（Polling）</span>
          </label>
          <label class="radio-option" :class="{ active: settings.mode === 'webhook' }">
            <input type="radio" value="webhook" v-model="settings.mode" />
            <span>Webhook</span>
          </label>
        </div>
        <div v-if="settings.mode === 'webhook'" class="settings-input-row" style="margin-top: 12px">
          <div class="input-wrapper">
            <input
              v-model="settings.webhook_url"
              type="url"
              placeholder="https://panel.example.com"
              class="vercel-input"
            />
          </div>
          <p class="webhook-note">系统会自动追加 <code>/api/telegram/webhook/{userID}</code>，这里只需填写面板公网 HTTPS 基础地址。</p>
        </div>
      </div>
      <!-- Bot Token -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.24s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">Bot Token</div>
          <div class="settings-card-desc">
            在 Telegram 中与 @BotFather 对话创建机器人并获取 Token。
            <span v-if="settings.bot_token_set" class="token-hint">已保存: {{ settings.bot_token_hint }}</span>
          </div>
        </div>
        <div class="settings-input-row">
          <div class="input-wrapper">
            <input
              v-model="settings.bot_token"
              type="password"
              :placeholder="settings.bot_token_set ? '留空则保留当前 Token' : '输入 Bot Token'"
              class="vercel-input"
            />
          </div>
        </div>
        <button
          v-if="!settings.bot_token_set && notifyBotSet"
          class="btn btn-secondary btn-sm"
          :disabled="reusing"
          @click="handleReuse"
          style="margin-top: 12px"
        >
          {{ reusing ? '复用中...' : '一键复用通知的 Bot' }}
        </button>
      </div>
      <!-- Admin TG IDs -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.32s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">授权 TG ID</div>
          <div class="settings-card-desc">
            逗号分隔的数字 ID。与 @userinfobot 对话可获取你的 ID。只有这些 TG 账号能向你的 Bot 发指令，可填多个，用英文逗号隔开。
          </div>
        </div>
        <div class="settings-input-row">
          <div class="input-wrapper">
            <input
              v-model="settings.admin_tg_ids"
              placeholder="例如: 123456789,987654321"
              class="vercel-input"
            />
          </div>
        </div>
      </div>
      <!-- API Endpoint (panel-wide, admin editable) -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.40s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">API 端点</div>
          <div class="settings-card-desc">
            面板级配置，所有用户的 Bot 都走该端点。国内网络建议使用自建反代（如 https://tele.example.com），默认官方 api.telegram.org 在国内无法直连。
          </div>
        </div>
        <div class="settings-input-row">
          <div class="input-wrapper">
            <input
              v-model="settings.api_endpoint"
              :readonly="!isAdmin"
              :placeholder="'https://api.telegram.org'"
              class="vercel-input"
            />
          </div>
          <button v-if="isAdmin" class="btn btn-primary btn-sm" :disabled="savingEndpoint" @click="handleSaveEndpoint">
            {{ savingEndpoint ? '保存中...' : '保存端点' }}
          </button>
        </div>
        <p v-if="!isAdmin" class="field-hint">端点由管理员统一配置，所有用户自动生效。</p>
      </div>
      <!-- Actions -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.56s' }">
        <div class="actions-row">
          <button class="btn btn-primary" :disabled="saving" @click="handleSave">
            <svg v-if="saving" class="spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
            {{ saving ? '保存中...' : '保存并应用' }}
          </button>
          <button class="btn btn-secondary" :disabled="testing" @click="handleTest">
            {{ testing ? '发送中...' : '发送测试消息' }}
          </button>
        </div>
      </div>
      <!-- Setup guide -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.64s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">设置教程</div>
        </div>
        <ol class="guide-list">
          <li>在 Telegram 中与 <strong>@BotFather</strong> 对话，发送 <code>/newbot</code> 创建机器人并复制 Token。</li>
          <li>与 <strong>@userinfobot</strong> 对话获取自己的数字 TG ID。</li>
          <li>在此页面填入 Token 和 ID，启用并保存。</li>
          <li>向你的 Bot 发送 <code>/help</code> 查看可用指令。</li>
        </ol>
        <div class="command-list">
          <span class="command-label">可用指令：</span>
          <code>/当前配置</code> <code>/列出隧道</code> <code>/选择隧道</code> <code>/转发</code>
          <code>/直连域名</code> <code>/优选绑定</code> <code>/绑定域名</code> <code>/列出区域</code>
          <code>/DNS列表</code> <code>/DNS详情</code> <code>/DNS添加</code> <code>/DNS修改</code> <code>/DNS删除</code> <code>/确认删除</code>
          <code>/全局优选</code> <code>/设置回退源</code> <code>/help</code>
        </div>
        <p class="guide-note">每个用户的 Bot 只操作自己的账户资源（隧道、DNS、域名绑定），互不影响。</p>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useMessage } from 'naive-ui'
import {
  getTelegramSettings,
  saveTelegramSettings,
  getTelegramStatus,
  testTelegram,
  reuseTelegramFromNotify,
  saveTelegramAPIEndpoint,
  type TelegramSettings,
  type TelegramStatus,
} from '../api'
import { useConfigStore } from '../stores/config'
const configStore = useConfigStore()
const message = useMessage()
const isAdmin = configStore.isAdmin()
const visible = ref(false)
const settings = ref({
  enabled: false,
  bot_token: '',
  admin_tg_ids: '',
  api_endpoint: '',
  mode: 'polling',
  webhook_url: '',
  bot_token_set: false,
  bot_token_hint: '',
})
const status = ref<TelegramStatus>({
  enabled: false,
  running: false,
  mode: 'polling',
  bot_username: '',
  last_error: '',
  last_update_at: '',
})
const saving = ref(false)
const testing = ref(false)
const reusing = ref(false)
const savingEndpoint = ref(false)
const notifyBotSet = ref(false)
let statusTimer: ReturnType<typeof setInterval> | null = null
const modeLabel = computed(() => {
  return status.value.mode === 'webhook' ? 'Webhook 模式' : '长轮询模式'
})
function formatTime(ts: string): string {
  try {
    const d = new Date(ts)
    return d.toLocaleString()
  } catch {
    return ts
  }
}
async function fetchSettings() {
  try {
    const { data } = await getTelegramSettings()
    settings.value.enabled = data.enabled
    settings.value.admin_tg_ids = data.admin_tg_ids
    settings.value.api_endpoint = data.api_endpoint || ''
    settings.value.mode = data.mode || 'polling'
    settings.value.webhook_url = data.webhook_url || ''
    settings.value.bot_token_set = data.bot_token_set
    settings.value.bot_token_hint = data.bot_token_hint
    settings.value.bot_token = '' // never prefill the token
    notifyBotSet.value = !!data.notify_bot_set
  } catch {
    // settings may not be available
  }
}

async function handleSaveEndpoint() {
  savingEndpoint.value = true
  try {
    const endpoint = settings.value.api_endpoint.trim().replace(/\/+$/, '')
    if (!endpoint) {
      message.error('API 端点不能为空')
      return
    }
    const { data } = await saveTelegramAPIEndpoint(endpoint)
    settings.value.api_endpoint = data.api_endpoint
    message.success('API 端点已保存，所有 Bot 已重启生效')
  } catch (e: any) {
    message.error('保存失败: ' + (e.response?.data?.error || e.message))
  } finally {
    savingEndpoint.value = false
  }
}

async function handleReuse() {
  reusing.value = true
  try {
    await reuseTelegramFromNotify()
    await fetchSettings()
    await fetchStatus()
    message.success('已复用通知的 Bot Token，请填写授权 TG ID 并保存启用')
  } catch (e: any) {
    message.error('复用失败: ' + (e.response?.data?.error || e.message))
  } finally {
    reusing.value = false
  }
}
async function fetchStatus() {
  try {
    const { data } = await getTelegramStatus()
    status.value = data
  } catch {
    // ignore
  }
}
async function handleSave() {
  saving.value = true
  try {
    const { data } = await saveTelegramSettings({
      enabled: settings.value.enabled,
      bot_token: settings.value.bot_token,
      admin_tg_ids: settings.value.admin_tg_ids,
      mode: settings.value.mode,
      webhook_url: settings.value.webhook_url,
    })
    settings.value.bot_token = '' // clear after save
    await fetchSettings()
    await fetchStatus()
    if (data.error) {
      message.warning('已保存，但启动失败: ' + data.error)
    } else {
      message.success('设置已保存')
    }
  } catch (e: any) {
    message.error('保存失败: ' + (e.response?.data?.error || e.message))
  } finally {
    saving.value = false
  }
}
async function handleTest() {
  testing.value = true
  try {
    const { data } = await testTelegram()
    message.success(data.message || '测试消息已发送')
  } catch (e: any) {
    message.error('发送失败: ' + (e.response?.data?.error || e.message))
  } finally {
    testing.value = false
  }
}
onMounted(() => {
  fetchSettings()
  fetchStatus()
  statusTimer = setInterval(fetchStatus, 10000)
  requestAnimationFrame(() => { visible.value = true })
})
onUnmounted(() => {
  if (statusTimer) clearInterval(statusTimer)
})
</script>
<style scoped>
.page-header { margin-bottom: var(--spacing-lg); }
.section { margin-bottom: var(--spacing-xl); }

.settings-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
}

.settings-card {
  background: var(--color-canvas-raised);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  padding: var(--spacing-xl);
}

.settings-card-header {
  margin-bottom: var(--spacing-lg);
}

.settings-card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-ink);
  margin: 0 0 4px;
}

.settings-card-desc {
  font-size: 13px;
  color: var(--color-mute);
  line-height: 1.6;
}

.status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-md);
  padding: var(--spacing-md);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  background: var(--color-canvas-soft);
}

.status-left {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-error);
}

.status-dot.on { background: var(--color-success); }
.status-dot.off { background: var(--color-error); }

.status-text { font-size: 14px; color: var(--color-ink); }

.field-row {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
  margin-bottom: var(--spacing-lg);
}

.field-row:last-child { margin-bottom: 0; }

.field-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-ink);
}

.field-desc {
  font-size: 12px;
  color: var(--color-mute);
  line-height: 1.5;
}

.radio-group {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-md);
  padding: var(--spacing-sm) 0;
}

.radio-option {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: 10px 14px;
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  background: var(--color-canvas-raised);
  cursor: pointer;
  transition: border-color 120ms ease, background-color 120ms ease;
}

.radio-option.active {
  border-color: var(--color-link);
  background: color-mix(in srgb, var(--color-link) 5%, transparent);
}

.radio-option input { margin: 0; width: 16px; height: 16px; accent-color: var(--color-focus); }
.radio-option span { font-size: 14px; color: var(--color-ink); }

.vercel-input {
  height: 36px;
  width: 100%;
  box-sizing: border-box;
}

.webhook-note {
  font-size: 12px;
  color: var(--color-mute);
  margin-top: 4px;
  word-break: break-all;
}

.webhook-note code {
  font-family: var(--font-mono);
  font-size: 11px;
  background: var(--color-canvas-soft-2);
  padding: 0 4px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-hairline);
}

.actions-row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-md);
  padding-top: var(--spacing-sm);
}

.actions-row .btn {
  min-width: 120px;
}

.guide-list {
  margin: 0;
  padding-left: 18px;
  font-size: 14px;
  color: var(--color-body);
  line-height: 1.6;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.guide-list code {
  font-family: var(--font-mono);
  font-size: 13px;
  background: var(--color-canvas-soft-2);
  padding: 0 5px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-hairline);
}

.command-list {
  margin-top: 12px;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.command-label {
  font-size: 13px;
  color: var(--color-mute);
}

.command-list code {
  font-family: var(--font-mono);
  font-size: 12px;
  background: var(--color-canvas-soft-2);
  padding: 2px 7px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-hairline);
}

.guide-note {
  margin-top: 12px;
  font-size: 13px;
  color: var(--color-mute);
}

.field-hint {
  font-size: 12px;
  color: var(--color-mute);
  margin-top: 8px;
}

@media (max-width: 640px) {
  .status-row { align-items: flex-start; flex-direction: column; }
  .radio-group { flex-direction: column; }
  .actions-row { flex-direction: column; }
  .actions-row .btn { width: 100%; justify-content: center; }
}
</style>
