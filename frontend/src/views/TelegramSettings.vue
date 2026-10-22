<template>
  <div class="page-container">
    <div class="page-header">
            <h2>{{ tr('m_3ba721cb5dd3') }}</h2>
      <p>{{ tr('m_d4463e3d9fb2') }}</p>
    </div>
    <div class="settings-list section">
      <!-- Status card -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.08s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">{{ tr('m_6888e1669b1b') }}</div>
        </div>
        <div class="status-row">
          <div class="status-left">
            <span class="status-dot" :class="status.running ? 'on' : 'off'" />
            <span class="status-text">
              <template v-if="status.running">
                {{ tr('m_72316475f6b7') }}{{ status.bot_username }} · {{ modeLabel }}
              </template>
              <template v-else>
                {{ tr('m_f006455e3baf') }}
              </template>
            </span>
          </div>
          <button class="btn btn-ghost btn-sm" @click="fetchStatus">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
            {{ tr('m_aee887434131') }}
          </button>
        </div>
        <div v-if="status.last_error" class="status-error">{{ status.last_error }}</div>
        <div v-if="status.last_update_at" class="status-meta">{{ tr('m_d150bcc25633') }} {{ formatTime(status.last_update_at) }}</div>
      </div>
      <!-- Enable toggle -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.16s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">{{ tr('m_13df38f938ce') }}</div>
          <div class="settings-card-desc">{{ tr('m_80de49d10357') }}</div>
        </div>
        <div class="toggle-row">
          <label class="switch">
            <input type="checkbox" v-model="settings.enabled" />
            <span class="switch-slider" />
          </label>
          <span class="toggle-label">{{ settings.enabled ? tr('m_dfb802238b38') : tr('m_bc5a87a757a5') }}</span>
        </div>
      </div>
      <!-- Mode: polling / webhook -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.20s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">{{ tr('m_87ed126f7bd1') }}</div>
          <div class="settings-card-desc">{{ tr('m_1c9d99d7be2c') }}</div>
        </div>
        <div class="radio-group">
          <label class="radio-option" :class="{ active: settings.mode === 'polling' }">
            <input type="radio" value="polling" v-model="settings.mode" />
            <span>{{ tr('m_be8c427c4c2c') }}</span>
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
          <p class="webhook-note">{{ tr('m_6e30809e1ea2') }} <code>/api/telegram/webhook/{userID}</code>{{ tr('m_0a4ee354241f') }}</p>
        </div>
      </div>
      <!-- Bot Token -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.24s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">Bot Token</div>
          <div class="settings-card-desc">
            {{ tr('m_4b81c912b6aa') }}
            <span v-if="settings.bot_token_set" class="token-hint">{{ tr('m_8e5c79c01a0e') }} {{ settings.bot_token_hint }}</span>
          </div>
        </div>
        <div class="settings-input-row">
          <div class="input-wrapper">
            <input
              v-model="settings.bot_token"
              type="password"
              :placeholder="settings.bot_token_set ? tr('m_607873e79fce') : tr('m_5e76895396d6')"
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
          {{ reusing ? tr('m_7b2d8b6a9ebf') : tr('m_888c2f8b0daf') }}
        </button>
      </div>
      <!-- Admin TG IDs -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.32s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">{{ tr('m_fa8620c6138e') }}</div>
          <div class="settings-card-desc">
            {{ tr('m_90ffb63a8969') }}
          </div>
        </div>
        <div class="settings-input-row">
          <div class="input-wrapper">
            <input
              v-model="settings.admin_tg_ids"
              :placeholder="tr('m_05e89e31b863')"
              class="vercel-input"
            />
          </div>
        </div>
      </div>
      <!-- API Endpoint (panel-wide, admin editable) -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.40s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">{{ tr('m_3d1ab62be024') }}</div>
          <div class="settings-card-desc">
            {{ tr('m_77e5f35acfb7') }}
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
            {{ savingEndpoint ? tr('m_917b1c1f18d0') : tr('m_459cf7f8c7e9') }}
          </button>
        </div>
        <p v-if="!isAdmin" class="field-hint">{{ tr('m_4ce7ea1d6b95') }}</p>
      </div>
      <!-- Actions -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.56s' }">
        <div class="actions-row">
          <button class="btn btn-primary" :disabled="saving" @click="handleSave">
            <svg v-if="saving" class="spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
            {{ saving ? tr('m_917b1c1f18d0') : tr('m_a7ec01a65f09') }}
          </button>
          <button class="btn btn-secondary" :disabled="testing" @click="handleTest">
            {{ testing ? tr('m_9d2ee1e4d8c3') : tr('m_15041725283e') }}
          </button>
        </div>
      </div>
      <!-- Setup guide -->
      <div class="settings-card" :class="{ '': visible }" :style="{ animationDelay: '0.64s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">{{ tr('m_1a1b001776ed') }}</div>
        </div>
        <ol class="guide-list">
          <li>{{ tr('m_18ddc12c857f') }} <strong>@BotFather</strong> {{ tr('m_0c8d3811377b') }} <code>/newbot</code> {{ tr('m_24fdd328e093') }}</li>
          <li>{{ tr('m_de1d5dd076fb') }} <strong>@userinfobot</strong> {{ tr('m_f8fe94a47d01') }}</li>
          <li>{{ tr('m_5e8f0e26ee1b') }}</li>
          <li>{{ tr('m_5aa90e74a202') }} <code>/help</code> {{ tr('m_48498890c5ea') }}</li>
        </ol>
        <div class="command-list">
          <span class="command-label">{{ tr('m_5c5959808bc9') }}</span>
          <code>{{ tr('m_a7abb08280bc') }}</code> <code>{{ tr('m_8d842f736d2c') }}</code> <code>{{ tr('m_470ea7ab4fc4') }}</code> <code>{{ tr('m_fe04b040c3e7') }}</code>
          <code>{{ tr('m_25bb6b27d7e8') }}</code> <code>{{ tr('m_8e3c17dabfb4') }}</code> <code>{{ tr('m_ae34c518af79') }}</code> <code>{{ tr('m_0ff4af2bf55e') }}</code>
          <code>{{ tr('m_98eac9a936ab') }}</code> <code>{{ tr('m_792dcb89058f') }}</code> <code>{{ tr('m_b7b575ef731b') }}</code> <code>{{ tr('m_9c55f982ca74') }}</code> <code>{{ tr('m_31affd89f87a') }}</code> <code>{{ tr('m_0ed55bf1a072') }}</code>
          <code>{{ tr('m_1b68ba88d96b') }}</code> <code>{{ tr('m_ad8f2f05e741') }}</code> <code>/help</code>
        </div>
        <p class="guide-note">{{ tr('m_0c09f6b0b1b0') }}</p>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { locale, t as tr } from '../i18n'

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
  return status.value.mode === 'webhook' ? tr('m_96011210ccb6') : tr('m_d1d1fb7d3129')
})
function formatTime(ts: string): string {
  try {
    const d = new Date(ts)
    return d.toLocaleString(locale.value)
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
      message.error(tr('m_bcd372c0a699'))
      return
    }
    const { data } = await saveTelegramAPIEndpoint(endpoint)
    settings.value.api_endpoint = data.api_endpoint
    message.success(tr('m_f07cfdf5580f'))
  } catch (e: any) {
    message.error(tr('m_84bd3b614686') + (e.response?.data?.error || e.message))
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
    message.success(tr('m_8f2a14658df1'))
  } catch (e: any) {
    message.error(tr('m_c3b5ab812dfe') + (e.response?.data?.error || e.message))
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
      message.warning(tr('m_c9d309e48308') + data.error)
    } else {
      message.success(tr('m_bd03c0aad701'))
    }
  } catch (e: any) {
    message.error(tr('m_84bd3b614686') + (e.response?.data?.error || e.message))
  } finally {
    saving.value = false
  }
}
async function handleTest() {
  testing.value = true
  try {
    const { data } = await testTelegram()
    message.success(data.message || tr('m_a422e5f7f4c4'))
  } catch (e: any) {
    message.error(tr('m_df54bb0e45fb') + (e.response?.data?.error || e.message))
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
