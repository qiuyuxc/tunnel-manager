<template>
  <div class="page-container">
    <div class="page-header">
            <h2>TG 机器人设置</h2>
      <p>通过 Telegram Bot 远程管理隧道配置</p>
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
      </div>
      <!-- Admin TG IDs -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.32s' }">
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
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.40s' }">
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
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.48s' }">
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

@media (max-width: 640px) {
  .status-row { align-items: flex-start; flex-direction: column; }
  .radio-group { flex-direction: column; }
  .actions-row { flex-direction: column; }
  .actions-row .btn { width: 100%; justify-content: center; }
}
</style>
