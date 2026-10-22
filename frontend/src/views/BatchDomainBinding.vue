<template>
  <div class="page-container">
    <div class="page-header">
            <h2>{{ tr('m_6f7ce20a6c76') }}</h2>
      <p>{{ tr('m_df37ef8cbb2e') }}</p>
    </div>
    <div v-if="!config.tunnel_id" class="prereq-banner section">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--color-banner-warning-text)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
      <div>
        <div class="prereq-title">{{ tr('m_5141efda7b2c') }}</div>
        <div class="prereq-desc">{{ tr('m_30c7f125bf8d') }}</div>
      </div>
    </div>
    <div v-else class="tunnel-context section">
      <span class="caption-mono">{{ tr('m_b903d203f70a') }}</span>
      <div>
        <strong>{{ config.tunnel_name || tr('m_8ce5320404d5') }}</strong>
      </div>
    </div>
    <div v-for="(group, index) in groups" :key="group.id" class="binding-group section">
      <div class="group-header">
        <div>
          <span class="caption-mono">{{ tr('m_e9e60eacb48a') }} {{ index + 1 }}</span>
          <p>{{ tr('m_2ec4de3cf7b9') }}</p>
        </div>
        <div class="group-actions">
          <button class="icon-button" type="button" :disabled="submitting" :aria-label="tr('m_e293251015c7')" :title="tr('m_80c99aca45cb')" @click="addGroup">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M12 5v14M5 12h14"/></svg>
          </button>
          <button v-if="groups.length > 1" class="icon-button danger" type="button" :disabled="submitting" :aria-label="tr('m_d7a64ce1a859', { p0: index + 1 })" :title="tr('m_35bf8246a7bb')" @click="removeGroup(index)">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M5 5l14 14M19 5 5 19"/></svg>
          </button>
        </div>
      </div>
      <div class="mode-selector" role="radiogroup" :aria-label="tr('m_a8b3f6be920e', { p0: index + 1 })">
        <button type="button" class="mode-option" :class="{ active: group.mode === 'simple' }" role="radio" :aria-checked="group.mode === 'simple'" :disabled="submitting" @click="group.mode = 'simple'">
          <strong>{{ tr('m_fa521927b78a') }}</strong><span>{{ tr('m_7609b1e04ffb') }}</span>
        </button>
        <button type="button" class="mode-option" :class="{ active: group.mode === 'preferred' }" role="radio" :aria-checked="group.mode === 'preferred'" :disabled="submitting" @click="group.mode = 'preferred'">
          <strong>{{ tr('m_a7974623c82b') }}</strong><span>{{ tr('m_9940c95e2111') }}</span>
        </button>
      </div>
      <div class="form-fields" :class="{ 'simple-fields': group.mode === 'simple' }">
        <div class="field">
          <label class="field-label" :for="`service-${group.id}`">{{ tr('m_c762fbaaad6f') }}</label>
          <input :id="`service-${group.id}`" v-model="group.service_url" class="vercel-input" placeholder="http://localhost:3000" :class="{ 'input-error': group.errors.service_url }" :disabled="submitting" @blur="validateGroup(group, 'service_url')" />
          <span v-if="group.errors.service_url" class="field-error">{{ group.errors.service_url }}</span>
        </div>
        <div v-if="group.mode === 'preferred'" class="field">
          <label class="field-label" :for="`cname-${group.id}`">{{ tr('m_ff425d8a519d') }} <span class="field-note">{{ tr('m_e864583adc16') }}</span></label>
          <cname-picker
            v-model="group.preferred_cname"
            :input-id="`cname-${group.id}`"
            :presets="config.cname_presets"
            :show-default="true"
            :default-value="config.preferred_cname"
            :disabled="submitting"
          />
        </div>
        <div class="field">
          <label class="field-label" :for="`main-${group.id}`">{{ tr('m_13cfe9660cdf') }} <span class="field-note">{{ tr('m_90845d8dacde') }}</span></label>
          <input :id="`main-${group.id}`" v-model="group.main_domain" class="vercel-input" :placeholder="tr('m_fc3c7f8bc7aa')" :class="{ 'input-error': group.errors.main_domain }" :disabled="submitting" @blur="validateGroup(group, 'main_domain')" />
          <span v-if="group.errors.main_domain" class="field-error">{{ group.errors.main_domain }}</span>
        </div>
        <div v-if="group.mode === 'preferred'" class="field">
          <label class="field-label" :for="`aux-${group.id}`">{{ tr('m_0135beb32bd5') }} <span class="field-note">{{ tr('m_320f1773293f') }}</span></label>
          <input :id="`aux-${group.id}`" v-model="group.aux_domain" class="vercel-input" :placeholder="tr('m_8aa71f623840')" :class="{ 'input-error': group.errors.aux_domain }" :disabled="submitting" @blur="validateGroup(group, 'aux_domain')" />
          <span v-if="group.errors.aux_domain" class="field-error">{{ group.errors.aux_domain }}</span>
        </div>
      </div>
      <div v-if="group.result" class="group-result" :class="group.result.success ? 'success' : 'error'">
        <strong>{{ group.result.success ? tr('m_67076a58977b') : tr('m_cdc8411c0d72') }}</strong>
        <span>{{ group.result.message }}</span>
        <span v-if="group.result.success && group.result.mode === 'preferred'">{{ tr('m_cecc67ab64a2') }}{{ group.result.preferred_cname }}</span>
      </div>
    </div>
    <OperationStatus v-if="submitting" :title="tr('m_665ce00ce6d8', { p0: groups.length })" />
    <div class="form-action">
      <button class="btn btn-primary" :disabled="submitting || !config.tunnel_id" @click="handleBatchBind">
        <svg v-if="submitting" class="spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
        <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
        {{ submitting ? tr('m_f81d26c1ec21') : tr('m_6c05d2aee66e', { p0: groups.length }) }}
      </button>
      <span v-if="summary" class="summary" :class="summary.success === summary.total ? 'summary-success' : 'summary-error'">{{ summary.success }}/{{ summary.total }} {{ tr('m_d575ff14a1b5') }}</span>
    </div>
  </div>
</template>
<script setup lang="ts">
import { t as tr } from '../i18n'

import { computed, onMounted, ref } from 'vue'
import { useMessage } from 'naive-ui'
import { bindDomainsBatch, type BatchBindItem, type BatchBindResult } from '../api'
import CnamePicker from '../components/CNAMEPicker.vue'
import OperationStatus from '../components/OperationStatus.vue'
import { useConfigStore } from '../stores/config'
type Field = keyof BatchBindItem
type BindingGroup = BatchBindItem & {
  id: number
  errors: Partial<Record<Field, string>>
  result: BatchBindResult | null
}
const message = useMessage()
const configStore = useConfigStore()
const config = configStore.config
const groups = ref<BindingGroup[]>([])
const submitting = ref(false)
let nextGroupID = 1
const summary = computed(() => {
  const completed = groups.value.filter((group) => group.result)
  if (!completed.length) return null
  return { success: completed.filter((group) => group.result?.success).length, total: completed.length }
})
function createGroup(): BindingGroup {
  return {
    id: nextGroupID++,
    mode: 'simple',
    service_url: config.service_url,
    preferred_cname: '',
    main_domain: '',
    aux_domain: '',
    errors: {},
    result: null,
  }
}
function addGroup() {
  groups.value.push(createGroup())
}
function removeGroup(index: number) {
  groups.value.splice(index, 1)
}
function validateGroup(group: BindingGroup, field: Field) {
  group.errors[field] = group[field].trim() ? '' : tr('m_7541de4d1a3f')
}
function validateGroups() {
  let valid = true
  for (const group of groups.value) {
    group.result = null
    const requiredFields = group.mode === 'preferred' ? ['service_url', 'main_domain', 'aux_domain'] : ['service_url', 'main_domain']
    group.errors.aux_domain = ''
    for (const field of requiredFields as Field[]) {
      validateGroup(group, field)
      valid &&= !group.errors[field]
    }
  }
  return valid
}
async function handleBatchBind() {
  if (submitting.value) return
  if (!validateGroups()) {
    message.error(tr('m_8a67c4b01df0'))
    return
  }
  submitting.value = true
  try {
    const items = groups.value.map(({ mode, service_url, preferred_cname, main_domain, aux_domain }) => ({ mode, service_url: service_url.trim(), preferred_cname: preferred_cname.trim(), main_domain: main_domain.trim(), aux_domain: aux_domain.trim() }))
    const { data } = await bindDomainsBatch(items)
    groups.value.forEach((group, index) => {
      group.result = data.results[index] || { ...items[index], success: false, message: tr('m_10250dd969a2') }
    })
    const successCount = data.results.filter((item) => item.success).length
    message[successCount === data.results.length ? 'success' : 'warning'](tr('m_acc8acc23d83', { p0: successCount, p1: data.results.length }))
  } catch (e: any) {
    message.error(tr('m_b460c19e6551') + (e.response?.data?.error || e.message))
  } finally {
    submitting.value = false
  }
}
onMounted(async () => {
  await configStore.fetchConfig()
  groups.value.push(createGroup())
})
</script>
<style scoped>
.prereq-banner {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-sm);
  padding: var(--spacing-md);
  background: var(--color-status-degraded-bg);
  border: 1px solid var(--color-status-degraded-border);
  border-radius: var(--radius-md);
  color: var(--color-status-degraded-text);
}
.prereq-title { font-size: 14px; font-weight: 600; }
.prereq-desc { margin-top: 2px; font-size: 13px; opacity: 0.84; }

.tunnel-context {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-md);
  padding: var(--spacing-sm) var(--spacing-md);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  background: var(--color-canvas-raised);
}
.tunnel-context > span { color: var(--color-mute); }
.tunnel-context > div { display: flex; min-width: 0; align-items: flex-end; flex-direction: column; gap: 2px; }
.tunnel-context strong { color: var(--color-ink); font-size: 14px; }

.binding-group {
  padding: var(--spacing-lg);
  background: var(--color-canvas-raised);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
}
.group-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--spacing-md);
  padding-bottom: var(--spacing-md);
  margin-bottom: var(--spacing-md);
  border-bottom: 1px solid var(--color-hairline);
}
.group-header p { margin: 6px 0 0; color: var(--color-mute); font-size: 13px; }
.group-actions { display: flex; gap: var(--spacing-xs); }

.mode-selector {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--spacing-xs);
  margin-bottom: var(--spacing-md);
  padding: 4px;
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  background: var(--color-canvas-soft);
}
.mode-option {
  display: flex;
  align-items: baseline;
  gap: 8px;
  min-height: 46px;
  padding: 10px 12px;
  color: var(--color-body);
  text-align: left;
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background-color 120ms ease, border-color 120ms ease;
}
.mode-option strong { color: var(--color-ink); font-size: 14px; }
.mode-option span { font-size: 12px; }
.mode-option.active { background: var(--color-canvas-raised); border-color: var(--color-link); }
.mode-option:disabled { opacity: 0.5; cursor: not-allowed; }

.form-fields { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: var(--spacing-md); }
.form-fields.simple-fields { grid-template-columns: 1fr; }
.field { display: flex; flex-direction: column; gap: 4px; }
.field-label { font-size: 14px; font-weight: 600; color: var(--color-ink); }
.field-note { font-weight: 400; color: var(--color-mute); margin-left: 4px; }
.field-error { font-size: 12px; color: var(--color-error); }

.group-result {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 10px;
  margin-top: var(--spacing-md);
  padding: var(--spacing-sm);
  border-radius: var(--radius-md);
  font-size: 13px;
  overflow-wrap: anywhere;
}
.group-result.success { color: var(--color-status-healthy-text); background: var(--color-status-healthy-bg); border: 1px solid var(--color-status-healthy-border); }
.group-result.error { color: var(--color-status-down-text); background: var(--color-status-down-bg); border: 1px solid var(--color-status-down-border); }

.form-action {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--spacing-sm);
  margin-top: var(--spacing-lg);
  padding-top: var(--spacing-lg);
  border-top: 1px solid var(--color-hairline);
}
.summary { font-size: 14px; }
.summary-success { color: var(--color-success); }
.summary-error { color: var(--color-error); }

.icon-button {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-hairline);
  background: var(--color-canvas-raised);
  color: var(--color-body);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background-color 120ms ease, color 120ms ease;
}
.icon-button:hover { background: var(--color-canvas-soft); color: var(--color-ink); }
.icon-button.danger:hover { color: var(--color-error); border-color: var(--color-error); }
.icon-button:disabled { opacity: 0.5; cursor: not-allowed; }

.spin { animation: spin 1s linear infinite; }
@media (max-width: 768px) {
  .mode-selector, .form-fields, .form-fields.simple-fields { grid-template-columns: 1fr; }
  .mode-option { align-items: flex-start; flex-direction: column; gap: 2px; }
}
@media (max-width: 480px) {
  .tunnel-context { align-items: flex-start; flex-direction: column; }
  .tunnel-context > div { align-items: flex-start; }
  .group-header { align-items: stretch; flex-direction: column; }
  .group-actions { justify-content: flex-end; }
  .field-label { display: flex; flex-direction: column; gap: 2px; }
  .field-note { margin-left: 0; }
  .form-action .btn { width: 100%; justify-content: center; }
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
