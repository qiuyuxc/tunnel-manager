<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ tr('m_4bf8b66d7390') }}</h2>
      <p>{{ tr('m_de782b9792c7') }}</p>
    </div>
    <div v-if="!config.tunnel_id || !config.service_url" class="prereq-banner section">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--color-banner-warning-text)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
      <div>
        <div class="prereq-title">{{ tr('m_5141efda7b2c') }}</div>
        <div class="prereq-desc">{{ tr('m_788ed2899d7d') }}</div>
      </div>
    </div>
    <div class="config-summary section">
      <div class="summary-row">
        <span class="summary-label caption-mono">{{ tr('m_b903d203f70a') }}</span>
        <div v-if="config.tunnel_id" class="summary-tunnel">
          <strong>{{ config.tunnel_name || tr('m_8ce5320404d5') }}</strong>
        </div>
        <span v-else class="summary-empty">{{ tr('m_80a57e03f071') }}</span>
      </div>
      <div class="summary-row summary-row-edit">
        <span class="summary-label caption-mono">{{ tr('m_c762fbaaad6f') }}</span>
        <div class="summary-edit">
          <input v-model="serviceURL" :disabled="binding || savingService" placeholder="http://localhost:3000" class="vercel-input summary-input" />
          <button class="btn btn-secondary btn-sm" :disabled="savingService || binding" @click="saveServiceURL">
            {{ savingService ? '...' : tr('m_a3030bf8f16d') }}
          </button>
        </div>
      </div>
      <div class="summary-row">
        <span class="summary-label caption-mono">{{ tr('m_54125d36ad63') }}</span>
        <code class="inline-code">{{ config.preferred_cname }}</code>
      </div>
    </div>
    <div class="form-card section">
      <div class="form-card-header">
        <span class="caption-mono form-card-label">{{ tr('m_bce4481a7278') }}</span>
      </div>
      <fieldset class="binding-fields" :disabled="binding || savingService">
      <div class="mode-selector" role="radiogroup" :aria-label="tr('m_fab2b113742d')">
        <button type="button" class="mode-option" :class="{ active: form.mode === 'simple' }" role="radio" :aria-checked="form.mode === 'simple'" @click="form.mode = 'simple'">
          <strong>{{ tr('m_fa521927b78a') }}</strong><span>{{ tr('m_e80df2700f06') }}</span>
        </button>
        <button type="button" class="mode-option" :class="{ active: form.mode === 'preferred' }" role="radio" :aria-checked="form.mode === 'preferred'" @click="form.mode = 'preferred'">
          <strong>{{ tr('m_a7974623c82b') }}</strong><span>{{ tr('m_ca4102dd0afd') }}</span>
        </button>
      </div>
      <div class="form-fields" :class="{ 'simple-fields': form.mode === 'simple' }">
        <div v-if="form.mode === 'preferred'" class="field cname-field">
          <label class="field-label">{{ tr('m_309710e63b6d') }} <span class="field-note">{{ tr('m_531883b0ad87') }}</span></label>
          <div class="input-wrapper">
            <cname-picker
              v-model="form.preferred_cname"
              :disabled="binding || savingService"
              :presets="config.cname_presets"
              :show-default="true"
              :default-value="config.preferred_cname"
              :placeholder="tr('m_d60669bb26a2')"
            />
            <span class="field-hint">{{ tr('m_c0ae8e199e92') }}{{ config.preferred_cname }}</span>
          </div>
        </div>
        <div class="field">
          <label class="field-label">{{ tr('m_13cfe9660cdf') }} <span class="field-note">{{ tr('m_90845d8dacde') }}</span></label>
          <div class="input-wrapper">
            <input
              v-model="form.main_domain"
              :placeholder="tr('m_fc3c7f8bc7aa')"
              class="vercel-input"
              :class="{ 'input-error': errors.main_domain }"
              @blur="validate('main_domain')"
            />
            <span v-if="errors.main_domain" class="field-error">{{ errors.main_domain }}</span>
          </div>
        </div>
        <div v-if="form.mode === 'preferred'" class="field">
          <label class="field-label">{{ tr('m_0135beb32bd5') }} <span class="field-note">{{ tr('m_320f1773293f') }}</span></label>
          <div class="input-wrapper">
            <input
              v-model="form.aux_domain"
              :placeholder="tr('m_8aa71f623840')"
              class="vercel-input"
              :class="{ 'input-error': errors.aux_domain }"
              @blur="validate('aux_domain')"
            />
            <span v-if="errors.aux_domain" class="field-error">{{ errors.aux_domain }}</span>
          </div>
        </div>
      </div>
      <OperationStatus v-if="binding || savingService" :title="binding ? tr('m_919b869a5aef') : tr('m_f0556b822864')" />
      <div class="form-action">
        <button class="btn btn-primary" @click="handleBind" :disabled="binding || !isValid">
          <svg v-if="binding" class="spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
          <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
          {{ binding ? tr('m_8220de90afa8') : tr('m_dd799f925f94') }}
        </button>
      </div>
      </fieldset>
    </div>
    <transition name="result-slide">
      <div v-if="result" class="result-card section" :class="result.success ? 'success' : 'error'">
      <div class="result-header">
        <svg v-if="result.success" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--color-success)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
        <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--color-error)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>
        <span>{{ result.success ? tr('m_67076a58977b') : tr('m_cdc8411c0d72') }}</span>
      </div>
      <div class="result-body">{{ result.message }}</div>
    </div>
    </transition>
  </div>
</template>
<script setup lang="ts">
import { t as tr } from '../i18n'

import { ref, computed, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { bindDomain, setServiceURL, type BindRequest } from '../api'
import CnamePicker from '../components/CNAMEPicker.vue'
import OperationStatus from '../components/OperationStatus.vue'
import { useConfigStore } from '../stores/config'
const message = useMessage()
const configStore = useConfigStore()
const config = configStore.config
const form = ref<BindRequest>({ mode: 'simple', preferred_cname: '', main_domain: '', aux_domain: '' })
const errors = ref<Record<string, string>>({})
const binding = ref(false)
const result = ref<{ success: boolean; message: string } | null>(null)
const serviceURL = ref(config.service_url)
const savingService = ref(false)
const isValid = computed(() => serviceURL.value.trim() && form.value.main_domain.trim() && (form.value.mode === 'simple' || form.value.aux_domain.trim()))
function validate(field: string) {
  const value = form.value[field as keyof BindRequest]
  const v = typeof value === 'string' ? value.trim() : value
  errors.value[field] = field === 'main_domain' || (field === 'aux_domain' && form.value.mode === 'preferred')
    ? (!v ? tr('m_7541de4d1a3f') : '') : ''
}
async function saveServiceURL() {
  if (savingService.value || binding.value) return
  savingService.value = true
  try {
    await setServiceURL(serviceURL.value)
    config.service_url = serviceURL.value
    message.success(tr('m_0aaac7dade14'))
  } catch (e: any) {
    message.error(tr('m_84bd3b614686') + (e.response?.data?.error || e.message))
  } finally {
    savingService.value = false
  }
}
async function handleBind() {
  if (binding.value || savingService.value || !isValid.value) return
  const payload = { ...form.value }
  binding.value = true
  result.value = null
  try {
    const nextServiceURL = serviceURL.value.trim()
    if (nextServiceURL !== config.service_url.trim()) {
      await setServiceURL(nextServiceURL)
      config.service_url = nextServiceURL
    }
    const { data } = await bindDomain(payload)
    result.value = { success: true, message: data.message || tr('m_27347bf9fa01') }
    message.success(tr('m_16330690424e'))
  } catch (e: any) {
    const errMsg = e.response?.data?.error || e.message
    result.value = { success: false, message: errMsg }
    message.error(tr('m_234f3125fccd') + errMsg)
  } finally {
    binding.value = false
  }
}
onMounted(async () => {
  await configStore.fetchConfig()
  serviceURL.value = config.service_url
})
</script>
<style scoped>
.binding-fields { padding: 0; margin: 0; border: 0; min-width: 0; }
.prereq-banner {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-sm);
  padding: var(--spacing-md);
  background: var(--color-status-degraded-bg);
  border: 1px solid var(--color-status-degraded-border);
  border-radius: var(--radius-md);
  color: var(--color-status-degraded-text);
  margin-bottom: var(--spacing-lg);
}
.prereq-title { font-size: 14px; font-weight: 600; }
.prereq-desc { margin-top: 2px; font-size: 13px; opacity: 0.84; }

.config-summary,
.form-card {
  background: var(--color-canvas-raised);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
}

.config-summary {
  overflow: hidden;
  margin-bottom: var(--spacing-xl);
}
.summary-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-md);
  padding: 14px var(--spacing-lg);
  border-bottom: 1px solid var(--color-hairline);
}
.summary-row:last-child { border-bottom: none; }
.summary-label { color: var(--color-mute); font-size: 12px; font-weight: 500; }
.summary-empty { color: var(--color-mute); font-size: 14px; }
.summary-row-edit { flex-wrap: wrap; }
.summary-edit {
  display: flex;
  gap: var(--spacing-sm);
  align-items: center;
  flex: 1;
  min-width: 0;
  justify-content: flex-end;
}
.summary-input { height: 36px; font-size: 13px; max-width: 320px; }
.btn-sm { height: 36px; padding: 0 14px; font-size: 13px; }
.summary-tunnel { display: flex; min-width: 0; align-items: flex-end; flex-direction: column; gap: 2px; text-align: right; }
.summary-tunnel strong { color: var(--color-ink); font-size: 14px; }

.form-card {
  padding: var(--spacing-xl);
}
.form-card-header {
  margin-bottom: var(--spacing-lg);
}
.form-card-label { color: var(--color-mute); font-size: 12px; font-weight: 500; }

.mode-selector {
  display: inline-flex;
  gap: 0;
  margin-bottom: var(--spacing-xl);
  padding: 4px;
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  background: var(--color-canvas-soft);
}

.mode-option {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  min-width: 180px;
  padding: 12px 16px;
  color: var(--color-body);
  text-align: left;
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background-color 150ms ease, border-color 150ms ease, color 150ms ease;
}
.mode-option strong { color: var(--color-ink); font-size: 14px; }
.mode-option span { font-size: 12px; line-height: 1.5; }
.mode-option.active {
  background: var(--color-canvas-raised);
  border-color: var(--color-link);
  color: var(--color-ink);
}
.mode-option:hover:not(.active) { background: rgba(37, 99, 235, 0.05); }

.form-fields {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--spacing-lg);
  margin-bottom: var(--spacing-xl);
}
.cname-field { grid-column: 1 / -1; }
.simple-fields { grid-template-columns: 1fr; }
.field { display: flex; flex-direction: column; gap: 6px; }
.field-label { font-size: 14px; font-weight: 500; color: var(--color-ink); }
.field-note { font-weight: 400; color: var(--color-mute); margin-left: 4px; }
.input-wrapper { display: flex; flex-direction: column; gap: 4px; }
.field-error { font-size: 12px; color: var(--color-error); }
.field-hint { font-size: 12px; color: var(--color-mute); margin-top: 4px; }

.form-action {
  margin-top: var(--spacing-xl);
  padding-top: var(--spacing-xl);
  border-top: 1px solid var(--color-hairline);
}

.result-card {
  padding: var(--spacing-lg);
  border-radius: var(--radius-lg);
  margin-top: var(--spacing-xl);
}
.result-card.success {
  background: var(--color-status-healthy-bg);
  border: 1px solid var(--color-status-healthy-border);
}
.result-card.error {
  background: var(--color-status-down-bg);
  border: 1px solid var(--color-status-down-border);
}
.result-header { display: flex; align-items: center; gap: var(--spacing-xs); font-size: 14px; font-weight: 600; margin-bottom: var(--spacing-xs); }
.result-card.success .result-header { color: var(--color-status-healthy-text); }
.result-card.error .result-header { color: var(--color-status-down-text); }
.result-body { font-size: 14px; color: var(--color-body); overflow-wrap: anywhere; }

@keyframes spin { to { transform: rotate(360deg); } }
.spin { animation: spin 1s linear infinite; }

@media (max-width: 768px) {
  .summary-row { align-items: flex-start; flex-direction: column; }
  .summary-edit { width: 100%; justify-content: stretch; }
  .summary-input { max-width: none; }
  .form-fields { grid-template-columns: 1fr; }
  .cname-field { grid-column: auto; }
  .summary-tunnel { align-items: flex-start; text-align: left; }
  .mode-selector { width: 100%; flex-direction: column; }
  .mode-option { width: 100%; box-sizing: border-box; }
}
</style>
