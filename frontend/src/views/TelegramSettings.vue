<template>
  <div class="page-container">
    <div class="page-header">
      <router-link to="/" class="back-link">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
        返回控制面板
      </router-link>
      <h2>TG 机器人设置</h2>
      <p>通过 Telegram Bot 远程管理隧道配置</p>
    </div>

    <div class="settings-list section">

      <!-- Status card -->
      <div class="settings-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.08s' }">
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
      <div class="settings-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.16s' }">
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

      <!-- Bot Token -->
      <div class="settings-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.24s' }">
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
      </div>

      <!-- Admin TG IDs -->
      <div class="settings-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.32s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">管理员 TG ID</div>
          <div class="settings-card-desc">
            逗号分隔的数字 ID。与 @userinfobot 对话可获取你的 ID。多个管理员用英文逗号隔开。
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

      <!-- API Endpoint -->
      <div class="settings-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.40s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">API 端点</div>
          <div class="settings-card-desc">默认使用 Telegram 官方 API。如果你有自建 Bot API 服务器，可在此指定地址。</div>
        </div>
        <div class="settings-input-row">
          <div class="input-wrapper">
            <input
              v-model="settings.api_endpoint"
              placeholder="https://api.telegram.org"
              class="vercel-input"
            />
          </div>
        </div>
      </div>

      <!-- Mode -->
      <div class="settings-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.48s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">运行模式</div>
          <div class="settings-card-desc">长轮询无需公网地址，适合内网环境。Webhook 需要服务器有公网 HTTPS 地址。</div>
        </div>
        <div class="mode-row">
          <label class="radio-item" :class="{ active: settings.mode === 'polling' }">
            <input type="radio" v-model="settings.mode" value="polling" />
            <span class="radio-dot" />
            <span class="radio-label">长轮询（推荐）</span>
          </label>
          <label class="radio-item" :class="{ active: settings.mode === 'webhook' }">
            <input type="radio" v-model="settings.mode" value="webhook" />
            <span class="radio-dot" />
            <span class="radio-label">Webhook</span>
          </label>
        </div>
        <div v-if="settings.mode === 'webhook'" class="webhook-url-row">
          <div class="input-wrapper">
            <input
              v-model="settings.webhook_url"
              placeholder="https://panel.example.com"
              class="vercel-input"
            />
          </div>
          <span class="webhook-note">后端将自动追加 /api/telegram/webhook</span>
        </div>
      </div>

      <!-- Actions -->
      <div class="settings-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.56s' }">
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
      <div class="settings-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.64s' }">
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
          <code>/DNS列表</code> <code>/DNS添加</code> <code>/DNS修改</code> <code>/DNS删除</code> <code>/确认删除</code>
          <code>/全局优选</code> <code>/设置回退源</code> <code>/help</code>
        </div>
        <p class="guide-note">Bot 与面板共享同一配置，两边操作实时同步。</p>
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
  type TelegramSettings,
  type TelegramStatus,
} from '../api'

const message = useMessage()
const visible = ref(false)

const settings = ref({
  enabled: false,
  bot_token: '',
  admin_tg_ids: '',
  mode: 'polling',
  webhook_url: '',
  api_endpoint: '',
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
    settings.value.mode = data.mode || 'polling'
    settings.value.webhook_url = data.webhook_url
    settings.value.api_endpoint = data.api_endpoint
    settings.value.bot_token_set = data.bot_token_set
    settings.value.bot_token_hint = data.bot_token_hint
    settings.value.bot_token = '' // never prefill the token
  } catch {
    // settings may not be available
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
      api_endpoint: settings.value.api_endpoint,
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
.section { margin-bottom: 0; }

.settings-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.settings-card {
  background: var(--color-canvas);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
  box-shadow: 0 1px 2px rgba(58, 47, 34, 0.05);
  transition: border-color 160ms ease-out, box-shadow 180ms ease-out, transform 180ms ease-out;
}
.settings-card:hover {
  border-color: var(--color-hairline-strong);
  box-shadow: 0 12px 28px rgba(58, 47, 34, 0.08);
  transform: translateY(-1px);
}

.settings-card-header { margin-bottom: var(--spacing-md); }
.settings-card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-ink);
  margin-bottom: 4px;
}
.settings-card-desc {
  font-size: 14px;
  color: var(--color-mute);
  line-height: 1.65;
}

.settings-input-row {
  display: flex;
  gap: var(--spacing-sm);
  align-items: flex-start;
}
.input-wrapper { flex: 1; min-width: 0; }

.token-hint {
  display: block;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--color-success);
  margin-top: 4px;
}

/* Status */
.status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.status-left {
  display: flex;
  align-items: center;
  gap: 8px;
}
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.status-dot.on {
  background: var(--color-success);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-success) 18%, transparent);
}
.status-dot.off {
  background: var(--color-mute);
}
.status-text { font-size: 14px; }
.status-error {
  margin-top: 8px;
  font-size: 13px;
  color: var(--color-error);
  font-family: var(--font-mono);
}
.status-meta {
  margin-top: 6px;
  font-size: 12px;
  color: var(--color-mute);
}
.btn-sm { height: 30px; padding: 0 10px; font-size: 12px; }

/* Toggle switch */
.toggle-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.toggle-label {
  font-size: 14px;
  color: var(--color-ink);
}
.switch {
  position: relative;
  display: inline-block;
  width: 40px;
  height: 22px;
}
.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}
.switch-slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--color-hairline-strong);
  border-radius: 22px;
  transition: background-color 160ms ease-out;
}
.switch-slider::before {
  content: '';
  position: absolute;
  height: 16px;
  width: 16px;
  left: 3px;
  bottom: 3px;
  background: var(--color-canvas);
  border-radius: 50%;
  transition: transform 160ms ease-out;
}
.switch input:checked + .switch-slider {
  background: var(--color-ink);
}
.switch input:checked + .switch-slider::before {
  transform: translateX(18px);
}

/* Mode radio */
.mode-row {
  display: flex;
  gap: 24px;
}
.radio-item {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  opacity: 0.5;
  transition: opacity 160ms ease-out;
}
.radio-item.active { opacity: 1; }
.radio-item input { display: none; }
.radio-dot {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 1.5px solid var(--color-hairline-strong);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: border-color 160ms ease-out;
}
.radio-item.active .radio-dot {
  border-color: var(--color-ink);
}
.radio-item.active .radio-dot::after {
  content: '';
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-ink);
}
.radio-label { font-size: 14px; }

.webhook-url-row {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--color-hairline);
}
.webhook-note {
  display: block;
  font-size: 12px;
  color: var(--color-mute);
  margin-top: 4px;
}

/* Actions */
.actions-row {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

/* Guide */
.guide-list {
  margin: 0;
  padding-left: 18px;
  font-size: 14px;
  color: var(--color-body);
  line-height: 24px;
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

@keyframes spin { to { transform: rotate(360deg); } }
.spin { animation: spin 1s linear infinite; }

@media (max-width: 768px) {
  .mode-row { flex-direction: column; gap: var(--spacing-sm); }
  .actions-row { flex-direction: column; }
  .actions-row .btn { width: 100%; justify-content: center; box-sizing: border-box; }
}

@media (max-width: 480px) {
  .status-row { align-items: flex-start; flex-direction: column; gap: var(--spacing-sm); }
  .status-row .btn { width: 100%; justify-content: center; }
  .toggle-row { justify-content: space-between; }
}
</style>
