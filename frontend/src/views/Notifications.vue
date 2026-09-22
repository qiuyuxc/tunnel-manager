<template>
  <div class="page-container notify-page">
    <div class="page-header">
      <h2>通知设置</h2>
      <p>选择通知渠道与事件，登录成功等事件会按你的配置发送提醒</p>
    </div>

    <div class="settings-list">
      <section v-if="appShell" class="settings-card">
        <div class="settings-card-header">
          <div class="settings-card-title">本机通知</div>
          <div class="settings-card-desc">
            由 App 自己在后台轮询服务器，不用开着浏览器也能收到告警；只在这台设备生效，需要手动开启。
          </div>
        </div>
        <div class="event-rows">
          <div class="event-row">
            <div>
              <strong>后台告警推送</strong>
              <p>{{ hostHint }}</p>
            </div>
            <n-switch :value="host.enabled" size="small" :loading="hostBusy" @update:value="toggleHost" />
          </div>
          <div class="event-row">
            <div>
              <strong>忽略电池优化</strong>
              <p>系统在息屏后可能限制后台轮询，允许后告警更及时。</p>
            </div>
            <button class="btn btn-secondary btn-sm" @click="openBatterySettings">
              {{ host.battery ? '已允许' : '去允许' }}
            </button>
          </div>
        </div>
        <p class="field-hint host-note">
          推送与邮件告警同源：需要在具体监控项目里开启「告警」，且只在状态发生变化时触发。
        </p>
        <div class="actions-row host-actions">
          <button class="btn btn-secondary" @click="sendHostTestNotify">发送本机测试通知</button>
        </div>
      </section>

      <section class="settings-card">
        <div class="settings-card-header">
          <div class="settings-card-title">通知渠道</div>
          <div class="settings-card-desc">可选邮箱、Telegram，或两者同时发送；选择「关闭通知」则不会发送任何提醒。</div>
        </div>
        <div class="channel-options">
          <label v-for="opt in channelOptions" :key="opt.value" class="channel-option" :class="{ active: channelMatches(opt.value) }">
            <input type="radio" :value="opt.value" :checked="channelMatches(opt.value)" @change="setChannels(opt.value)" />
            <span>{{ opt.label }}</span>
          </label>
        </div>
      </section>

      <section class="settings-card">
        <div class="settings-card-header">
          <div class="settings-card-title">通知事件</div>
          <div class="settings-card-desc">开启的事件才会发送提醒。</div>
        </div>
        <div class="event-rows">
          <div class="event-row">
            <div>
              <strong>登录通知</strong>
              <p>每次账户登录成功后发送提醒（含时间与 IP）。</p>
            </div>
            <n-switch v-model:value="events.login" size="small" />
          </div>
        </div>
      </section>

      <section class="settings-card">
        <div class="settings-card-header">
          <div class="settings-card-title">邮箱收件人</div>
          <div class="settings-card-desc">每行一个邮箱，支持多个收件人；仅在启用邮箱渠道时生效。</div>
        </div>
        <textarea v-model="emails" rows="4" class="vercel-input emails-input" placeholder="you@example.com&#10;ops@example.com"></textarea>
      </section>

      <section class="settings-card">
        <div class="settings-card-header">
          <div class="settings-card-title">Telegram 机器人</div>
          <div class="settings-card-desc">使用你自己的 Bot（由 @BotFather 创建），与面板管理员无关；仅在启用 Telegram 渠道时生效。</div>
        </div>
        <div class="form-stack">
          <label class="field">
            <span class="field-label">Bot Token</span>
            <input v-model="tgBotToken" type="password" class="vercel-input" :placeholder="tgBotTokenSet ? '留空保持不变' : '123456:ABC…'" autocomplete="off" />
            <span class="tag" :class="tgBotTokenSet ? 'tag-ok' : 'tag-down'">{{ tgBotTokenSet ? '已设置' : '未设置' }}</span>
          </label>
          <label class="field">
            <span class="field-label">接收 Chat ID</span>
            <input v-model="tgChatID" type="text" class="vercel-input" placeholder="123456789" />
          </label>
          <p class="field-hint">获取方式：先向自己的 Bot 发送任意消息，再打开 https://api.telegram.org/bot&lt;TOKEN&gt;/getUpdates，取返回结果中的 chat.id。</p>
        </div>
      </section>

      <div class="actions-row">
        <button class="btn btn-primary" :disabled="saving" @click="save">{{ saving ? '保存中...' : '保存设置' }}</button>
        <button class="btn btn-secondary" :disabled="testing" @click="sendTest">{{ testing ? '发送中...' : '发送测试通知' }}</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useMessage, NSwitch } from 'naive-ui'
import { getNotifySettings, updateNotifySettings, testNotify, type SaveNotifyPayload } from '../api/notify'
import {
  isAppShell,
  readHostNotifyState,
  enableHostNotify,
  disableHostNotify,
  sendHostTestNotify,
  openBatterySettings,
  type HostNotifyState,
} from '../utils/appHost'

const message = useMessage()
const channels = ref<string[]>([])
const events = ref<Record<string, boolean>>({ login: true })
const emails = ref('')
const tgBotToken = ref('')
const tgBotTokenSet = ref(false)
const tgChatID = ref('')
const saving = ref(false)
const testing = ref(false)

// Native-only card. In a browser isAppShell is false and none of this renders.
const appShell = isAppShell()
const host = ref<HostNotifyState>({ enabled: false, permission: true, battery: false })
const hostBusy = ref(false)

const hostHint = computed(() => {
  if (!host.value.permission) return '系统还没允许本机发送通知，打开开关时会弹出授权。'
  if (!host.value.enabled) return '已关闭。开启后 App 会常驻一条低调通知来维持轮询。'
  if (!host.value.battery) return '运行中；建议同时允许忽略电池优化，避免息屏后被系统掐掉。'
  return '运行中，服务端一有状态变化就会推送到这里。'
})

function refreshHost() {
  const state = readHostNotifyState()
  if (state) host.value = state
}

/** The permission dialog resolves later, so the shell signals us when it does. */
function onHostChange() {
  refreshHost()
  hostBusy.value = false
}

function toggleHost(value: boolean) {
  hostBusy.value = true
  if (value) enableHostNotify()
  else disableHostNotify()
  window.setTimeout(onHostChange, 350)
}

const channelOptions = [
  { value: '', label: '关闭通知' },
  { value: 'email', label: '仅邮箱' },
  { value: 'telegram', label: '仅 Telegram' },
  { value: 'both', label: '邮箱 + Telegram' },
]

function channelMatches(value: string) {
  if (value === '') return channels.value.length === 0
  if (value === 'both') return channels.value.includes('email') && channels.value.includes('telegram')
  return channels.value.length === 1 && channels.value[0] === value
}

function setChannels(value: string) {
  if (value === '') channels.value = []
  else if (value === 'both') channels.value = ['email', 'telegram']
  else channels.value = [value]
}

onMounted(async () => {
  refreshHost()
  window.addEventListener('tmhostchange', onHostChange)
  try {
    const { data } = await getNotifySettings()
    channels.value = data.channels || []
    events.value = { login: !!data.events?.login }
    emails.value = data.emails || ''
    tgBotTokenSet.value = data.tg_bot_token_set
    tgChatID.value = data.tg_notify_chat_id || ''
  } catch (e: any) {
    message.error('加载失败: ' + (e.response?.data?.error || e.message))
  }
})

onUnmounted(() => window.removeEventListener('tmhostchange', onHostChange))

async function save() {
  saving.value = true
  try {
    const payload: SaveNotifyPayload = {
      channels: channels.value,
      events: { ...events.value },
      emails: emails.value,
      tg_notify_chat_id: tgChatID.value.trim(),
    }
    if (tgBotToken.value) payload.tg_bot_token = tgBotToken.value
    const { data } = await updateNotifySettings(payload)
    tgBotToken.value = ''
    tgBotTokenSet.value = data.tg_bot_token_set
    message.success('通知设置已保存')
  } catch (e: any) {
    message.error('保存失败: ' + (e.response?.data?.error || e.message))
  } finally {
    saving.value = false
  }
}

async function sendTest() {
  testing.value = true
  try {
    await testNotify()
    message.success('测试通知已发送')
  } catch (e: any) {
    message.error('发送失败: ' + (e.response?.data?.error || e.message))
  } finally {
    testing.value = false
  }
}
</script>

<style scoped>
.settings-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
  max-width: 760px;
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

.channel-options {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 8px;
}

.channel-option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: 13px;
  color: var(--color-body);
  transition: border-color 120ms ease, background-color 120ms ease;
}

.channel-option:hover { border-color: var(--color-hairline-strong); }
.channel-option.active {
  border-color: var(--color-link);
  background: color-mix(in srgb, var(--color-link) 6%, transparent);
  color: var(--color-ink);
}
.channel-option input { accent-color: var(--color-focus); }

.event-rows { display: flex; flex-direction: column; gap: 10px; }
.event-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 14px;
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
}
.event-row strong { font-size: 14px; color: var(--color-ink); }
.event-row p { margin: 2px 0 0; font-size: 12px; color: var(--color-mute); line-height: 1.5; }

.emails-input {
  width: 100%;
  resize: vertical;
  font-family: var(--font-mono);
  font-size: 13px;
  line-height: 1.7;
}

.form-stack { display: flex; flex-direction: column; gap: var(--spacing-sm); }
.field { display: flex; flex-direction: column; gap: 6px; }
.field-label {
  color: var(--color-mute);
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.09em;
  text-transform: uppercase;
}
.field-hint {
  margin: 0;
  color: var(--color-mute);
  font-size: 12px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.tag {
  align-self: flex-start;
  margin-top: 4px;
  padding: 2px 9px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
}
.tag-ok { color: var(--color-status-up-text); border: 1px solid var(--color-status-up-border); background: var(--color-status-up-bg); }
.tag-down { color: var(--color-mute); border: 1px solid var(--color-hairline-strong); background: var(--color-canvas-soft); }

.actions-row {
  display: flex;
  gap: var(--spacing-sm);
}

.host-actions { margin-top: var(--spacing-md); }
.host-note { margin-top: var(--spacing-md); }

@media (max-width: 480px) {
  .notify-page { padding-left: var(--spacing-md); padding-right: var(--spacing-md); }
  .actions-row { flex-direction: column; }
  .actions-row .btn { width: 100%; justify-content: center; }
}
</style>
