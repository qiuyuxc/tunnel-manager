<template>
  <n-modal :show="open" preset="card" title="AI 连接配置" class="ai-settings-modal" :style="{ width: 'min(560px, calc(100vw - 24px))' }" @update:show="close">
    <div class="settings-body">
      <p v-if="settings?.shared_enabled && !settings?.is_admin" class="notice">由管理员提供全站共享连接。密钥与端点不可查看，会话和任务仍仅属于你。</p>
      <template v-else-if="settings">
        <p class="notice">配置当前管理员使用的 AI 连接。</p>
        <form @submit.prevent="save">
          <label>API 端点<input v-model="endpoint" type="url" placeholder="https://api.example.com/v1" autocomplete="off" :disabled="saving" /></label>
          <label>模型名称<input v-model="model" placeholder="填写服务商支持的模型 ID" maxlength="200" autocomplete="off" :disabled="saving" /></label>
          <label>API 密钥<input v-model="apiKey" type="password" :placeholder="keySet ? '已设置，留空保留原密钥' : '请输入 API 密钥'" autocomplete="new-password" :disabled="saving" /></label>
          <p class="note">兼容 Chat Completions 与工具调用。支持 HTTP/HTTPS、本机或内网地址及自定义端口；localhost 指运行后端的电脑或容器。HTTP 明文传输密钥与对话，仅在可信网络使用，公网建议 HTTPS。密钥在服务端加密保存，不回显；更换端点需重新填写密钥。</p>
          <p v-if="error" class="error" role="alert">{{ error }}</p>
          <button class="primary" :disabled="saving" type="submit">{{ saving ? '保存中…' : '保存配置' }}</button>
        </form>
      </template>
      <p v-else>正在读取配置…</p>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { NModal } from 'naive-ui'
import { saveAISettings, type AISettings } from '../api/assistant'
import { aiError } from '../stores/assistant'

const props = defineProps<{ open: boolean; settings: AISettings | null }>()
const emit = defineEmits<{ close: []; saved: [settings: AISettings] }>()
const scope = ref<'personal' | 'shared'>('personal')
const endpoint = ref('')
const model = ref('')
const apiKey = ref('')
const keySet = ref(false)
const sharedEnabled = ref(false)
const saving = ref(false)
const error = ref('')

function fill() {
  const connection = props.settings?.[scope.value]
  endpoint.value = connection?.endpoint || ''
  model.value = connection?.model || ''
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
    const { data } = await saveAISettings({ scope: scope.value, endpoint: endpoint.value.trim(), model: model.value.trim(), api_key: apiKey.value, ...(scope.value === 'shared' ? { shared_enabled: sharedEnabled.value } : {}) })
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
