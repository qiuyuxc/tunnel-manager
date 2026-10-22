<template>
  <div class="page-container notify-page">
    <div class="page-header">
      <h2>{{ tr('m_247ca01ae280') }}</h2>
      <p>{{ tr('m_3767e83d1c67') }}</p>
    </div>

    <div class="settings-list">
      <section v-if="appShell" class="settings-card">
        <div class="settings-card-header">
          <div class="settings-card-title">{{ tr('m_5de89cb69dab') }}</div>
          <div class="settings-card-desc">
            {{ tr('m_6a7a66616ad4') }}
          </div>
        </div>
        <div class="event-rows">
          <div class="event-row">
            <div>
              <strong>{{ tr('m_9f98f9238b0b') }}</strong>
              <p>{{ hostHint }}</p>
            </div>
            <n-switch :value="host.enabled" size="small" :loading="hostBusy" @update:value="toggleHost" />
          </div>
          <div class="event-row">
            <div>
              <strong>{{ tr('m_b360487b1126') }}</strong>
              <p>{{ tr('m_0c7582ec3d7d') }}</p>
            </div>
            <button class="btn btn-secondary btn-sm" @click="openBatterySettings">
              {{ host.battery ? tr('m_75d889ac43a2') : tr('m_beea4ffde7bf') }}
            </button>
          </div>
        </div>
        <p class="field-hint host-note">
          {{ tr('m_55682d4cb56d') }}
        </p>
        <div class="actions-row host-actions">
          <button class="btn btn-secondary" @click="sendHostTestNotify">{{ tr('m_aba258defcc1') }}</button>
        </div>
      </section>

      <section class="settings-card">
        <div class="settings-card-header">
          <div class="settings-card-title">{{ tr('m_b2354cebb5bc') }}</div>
          <div class="settings-card-desc">{{ tr('m_3c8e6a869915') }}</div>
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
          <div class="settings-card-title">{{ tr('m_e2ecbfe6774d') }}</div>
          <div class="settings-card-desc">{{ tr('m_d216483c416a') }}</div>
        </div>
        <div class="event-rows">
          <div class="event-row">
            <div>
              <strong>{{ tr('m_f7141ed88f5b') }}</strong>
              <p>{{ tr('m_2c35d196666e') }}</p>
            </div>
            <n-switch v-model:value="events.login" size="small" />
          </div>
        </div>
      </section>

      <section class="settings-card">
        <div class="settings-card-header">
          <div class="settings-card-title">{{ tr('m_b925241c3e99') }}</div>
          <div class="settings-card-desc">{{ tr('m_a18a8b2ffdbe') }}</div>
        </div>
        <textarea v-model="emails" rows="4" class="vercel-input emails-input" placeholder="you@example.com&#10;ops@example.com"></textarea>
      </section>

      <section class="settings-card">
        <div class="settings-card-header">
          <div class="settings-card-title">{{ tr('m_b0edcbcc7bba') }}</div>
          <div class="settings-card-desc">{{ tr('m_2b6aee03cf2d') }}</div>
        </div>
        <div class="form-stack">
          <label class="field">
            <span class="field-label">Bot Token</span>
            <input v-model="tgBotToken" type="password" class="vercel-input" :placeholder="tgBotTokenSet ? tr('m_3260348163d0') : '123456:ABC…'" autocomplete="off" />
            <span class="tag" :class="tgBotTokenSet ? 'tag-ok' : 'tag-down'">{{ tgBotTokenSet ? tr('m_e025f1ff7199') : tr('m_2f5f1d6fbfb0') }}</span>
          </label>
          <label class="field">
            <span class="field-label">{{ tr('m_6da43c19f64c') }}</span>
            <input v-model="tgChatID" type="text" class="vercel-input" placeholder="123456789" />
          </label>
          <p class="field-hint">{{ tr('m_caa14e0e6f80') }}</p>
          <button
            v-if="!tgBotTokenSet && tgRemoteBotSet"
            class="btn btn-secondary btn-sm"
            :disabled="reusing"
            @click="reuseFromTelegram"
          >
            {{ reusing ? tr('m_7b2d8b6a9ebf') : tr('m_77765ef7d0e2') }}
          </button>
        </div>
      </section>

      <div class="actions-row">
        <button class="btn btn-primary" :disabled="saving" @click="save">{{ saving ? tr('m_917b1c1f18d0') : tr('m_c8550237ba70') }}</button>
        <button class="btn btn-secondary" :disabled="testing" @click="sendTest">{{ testing ? tr('m_9d2ee1e4d8c3') : tr('m_5e0022a1e907') }}</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { t as tr, localized } from '../i18n'

import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useMessage, NSwitch } from 'naive-ui'
import { getNotifySettings, updateNotifySettings, testNotify, reuseNotifyFromTelegram, type SaveNotifyPayload } from '../api/notify'
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
const tgRemoteBotSet = ref(false)
const saving = ref(false)
const testing = ref(false)
const reusing = ref(false)

// Native-only card. In a browser isAppShell is false and none of this renders.
const appShell = isAppShell()
const host = ref<HostNotifyState>({ enabled: false, permission: true, battery: false })
const hostBusy = ref(false)

const hostHint = computed(() => {
  if (!host.value.permission) return tr('m_d2df17878e54')
  if (!host.value.enabled) return tr('m_f5f1e934f984')
  if (!host.value.battery) return tr('m_f565e5e601f9')
  return tr('m_bc67ea42aa7f')
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

const channelOptions = localized(() => ([
  { value: '', label: tr('m_27413c315695') },
  { value: 'email', label: tr('m_128003be2c2c') },
  { value: 'telegram', label: tr('m_e7d8fb7fa65f') },
  { value: 'both', label: tr('m_2b36843d5a74') },
]))

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
    tgRemoteBotSet.value = !!data.tg_remote_bot_set
  } catch (e: any) {
    message.error(tr('m_08417baa06d4') + (e.response?.data?.error || e.message))
  }
})

onUnmounted(() => window.removeEventListener('tmhostchange', onHostChange))

async function reuseFromTelegram() {
  reusing.value = true
  try {
    const { data } = await reuseNotifyFromTelegram()
    tgBotToken.value = ''
    tgBotTokenSet.value = data.tg_bot_token_set
    tgRemoteBotSet.value = !!data.tg_remote_bot_set
    message.success(tr('m_63f4b00fe00c'))
  } catch (e: any) {
    message.error(tr('m_c3b5ab812dfe') + (e.response?.data?.error || e.message))
  } finally {
    reusing.value = false
  }
}

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
    message.success(tr('m_07a20770f7aa'))
  } catch (e: any) {
    message.error(tr('m_84bd3b614686') + (e.response?.data?.error || e.message))
  } finally {
    saving.value = false
  }
}

async function sendTest() {
  testing.value = true
  try {
    await testNotify()
    message.success(tr('m_33c45f088ec9'))
  } catch (e: any) {
    message.error(tr('m_df54bb0e45fb') + (e.response?.data?.error || e.message))
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
