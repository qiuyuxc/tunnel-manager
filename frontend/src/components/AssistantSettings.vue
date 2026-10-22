<template>
  <n-modal :show="open" preset="card" :title="tr('m_036413404922')" class="ai-settings-modal" :style="{ width: 'min(560px, calc(100vw - 24px))' }" @update:show="close">
    <div class="settings-body">
      <p v-if="settings?.shared_enabled && !settings?.is_admin" class="notice">{{ tr('m_72e1d25a4b93') }}</p>
      <template v-else-if="settings">
        <label v-if="settings.is_admin">{{ tr('m_d77674f90811') }}
          <select v-model="scope" :disabled="saving" @change="fill"><option value="personal">{{ tr('m_eeb62339a596') }}</option><option value="shared">{{ tr('m_0372d8590b30') }}</option></select>
        </label>
        <label v-if="scope === 'shared'" class="check"><input v-model="sharedEnabled" type="checkbox" :disabled="saving" />{{ tr('m_f6f63440e200') }}</label>
        <p class="notice">{{ scope === 'shared' ? tr('m_b8aa668dda8a') : tr('m_2a6990243b0a') }}</p>
        <form @submit.prevent="save">
          <label>{{ tr('m_3d1ab62be024') }}<input v-model="endpoint" type="url" placeholder="https://api.example.com/v1" autocomplete="off" :disabled="saving" /></label>
          <label>{{ tr('m_77ff2042727b') }}<select v-model="apiFormat" :disabled="saving"><option value="chat_completions">{{ tr('m_82026a7b6020') }}</option><option value="responses">{{ tr('m_bc31ab86ad29') }}</option></select></label>
          <label v-if="apiFormat === 'responses'" class="check"><input v-model="reasoningSummary" type="checkbox" :disabled="saving" />{{ tr('m_884e4a50553e') }}</label>
          <label>{{ tr('m_cb2f1709f983') }}<input v-model="model" :placeholder="tr('m_198031fe42c1')" maxlength="200" autocomplete="off" :disabled="saving" /></label>
          <label>{{ tr('m_5f600b307b4e') }}<input v-model="apiKey" type="password" :placeholder="keySet ? tr('m_a025b0c0b478') : tr('m_f6277fc55bc2')" autocomplete="new-password" :disabled="saving" /></label>
          <p class="note">{{ tr('m_66774fdd35ac') }}</p>
          <p v-if="error" class="error" role="alert">{{ error }}</p>
          <button class="primary" :disabled="saving" type="submit">{{ saving ? tr('m_ff509c9ba052') : tr('m_6e584e3d5ce6') }}</button>
        </form>
      </template>
      <p v-else>{{ tr('m_8a044dd74276') }}</p>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { t as tr } from '../i18n'

import { ref, watch } from 'vue'
import { NModal } from 'naive-ui'
import { saveAISettings, type AISettings } from '../api/assistant'
import { aiError } from '../stores/assistant'

const props = defineProps<{ open: boolean; settings: AISettings | null }>()
const emit = defineEmits<{ close: []; saved: [settings: AISettings] }>()
const scope = ref<'personal' | 'shared'>('personal')
const endpoint = ref('')
const model = ref('')
const apiFormat = ref<'chat_completions' | 'responses'>('chat_completions')
const reasoningSummary = ref(false)
const apiKey = ref('')
const keySet = ref(false)
const sharedEnabled = ref(false)
const saving = ref(false)
const error = ref('')

function fill() {
  const connection = props.settings?.[scope.value]
  endpoint.value = connection?.endpoint || ''
  model.value = connection?.model || ''
  apiFormat.value = connection?.api_format || 'chat_completions'
  reasoningSummary.value = !!connection?.reasoning_summary
  keySet.value = !!connection?.key_set
  apiKey.value = ''
  sharedEnabled.value = !!props.settings?.shared_enabled
  error.value = ''
}
watch(() => props.open, open => {
  apiKey.value = ''
  if (open) { scope.value = props.settings?.shared_enabled && props.settings.is_admin ? 'shared' : 'personal'; fill() }
})
function close() { if (!saving.value) { apiKey.value = ''; emit('close') } }
async function save() {
  saving.value = true
  error.value = ''
  try {
    const { data } = await saveAISettings({ scope: scope.value, endpoint: endpoint.value.trim(), model: model.value.trim(), api_key: apiKey.value, api_format: apiFormat.value, reasoning_summary: reasoningSummary.value, ...(scope.value === 'shared' ? { shared_enabled: sharedEnabled.value } : {}) })
    apiKey.value = ''
    emit('saved', data)
    emit('close')
  } catch (failure) { error.value = aiError(failure) }
  finally { saving.value = false }
}
</script>

<style scoped>
.settings-body { color: var(--color-ink); }
label { display: grid; gap: 8px; margin-bottom: 18px; font-weight: 600; }
input:not([type='checkbox']), select { width: 100%; min-height: 44px; box-sizing: border-box; padding: 10px 12px; border: 1px solid var(--color-hairline-strong); border-radius: 12px; background: var(--color-canvas-soft); color: var(--color-ink); font: inherit; }
.check { display: flex; align-items: center; min-height: 44px; }
.notice, .note { color: var(--color-mute); line-height: 1.7; margin: 0 0 18px; }
.note { font-size: 12px; }
.error { color: var(--color-error); overflow-wrap: anywhere; }
.primary { min-height: 44px; padding: 10px 20px; border: none; border-radius: 12px; background: var(--color-btn-primary-bg); color: var(--color-btn-primary-text); font: inherit; font-weight: 650; cursor: pointer; }
button:disabled { opacity: .55; cursor: wait; }
:is(input, select, button):focus-visible { outline: 2px solid var(--color-success); outline-offset: 3px; }
</style>
