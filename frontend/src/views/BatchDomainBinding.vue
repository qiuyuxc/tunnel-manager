<template>
  <div class="page-container">
    <div class="page-header">
            <h2>批量绑定</h2>
      <p>每组独立设置转发地址，按顺序绑定主域名和辅助域名。</p>
    </div>
    <div v-if="!config.tunnel_id" class="prereq-banner section">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--color-banner-warning-text)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
      <div>
        <div class="prereq-title">前置条件未满足</div>
        <div class="prereq-desc">请先在「隧道管理」选择隧道。</div>
      </div>
    </div>
    <div v-else class="tunnel-context section">
      <span class="caption-mono">当前隧道</span>
      <div>
        <strong>{{ config.tunnel_name || '已选隧道' }}</strong>
      </div>
    </div>
    <div v-for="(group, index) in groups" :key="group.id" class="binding-group section">
      <div class="group-header">
        <div>
          <span class="caption-mono">绑定组 {{ index + 1 }}</span>
          <p>该组的两个域名将转发到同一地址。</p>
        </div>
        <div class="group-actions">
          <button class="icon-button" type="button" :disabled="submitting" aria-label="添加绑定组" title="添加一组" @click="addGroup">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M12 5v14M5 12h14"/></svg>
          </button>
          <button v-if="groups.length > 1" class="icon-button danger" type="button" :disabled="submitting" :aria-label="`删除绑定组 ${index + 1}`" title="删除此组" @click="removeGroup(index)">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M5 5l14 14M19 5 5 19"/></svg>
          </button>
        </div>
      </div>
      <div class="mode-selector" role="radiogroup" :aria-label="`绑定组 ${index + 1} 模式`">
        <button type="button" class="mode-option" :class="{ active: group.mode === 'simple' }" role="radio" :aria-checked="group.mode === 'simple'" :disabled="submitting" @click="group.mode = 'simple'">
          <strong>简单模式</strong><span>主域名直连服务</span>
        </button>
        <button type="button" class="mode-option" :class="{ active: group.mode === 'preferred' }" role="radio" :aria-checked="group.mode === 'preferred'" :disabled="submitting" @click="group.mode = 'preferred'">
          <strong>优选模式</strong><span>使用优选线路和辅助回源</span>
        </button>
      </div>
      <div class="form-fields" :class="{ 'simple-fields': group.mode === 'simple' }">
        <div class="field">
          <label class="field-label" :for="`service-${group.id}`">转发地址</label>
          <input :id="`service-${group.id}`" v-model="group.service_url" class="vercel-input" placeholder="http://localhost:3000" :class="{ 'input-error': group.errors.service_url }" :disabled="submitting" @blur="validateGroup(group, 'service_url')" />
          <span v-if="group.errors.service_url" class="field-error">{{ group.errors.service_url }}</span>
        </div>
        <div v-if="group.mode === 'preferred'" class="field">
          <label class="field-label" :for="`cname-${group.id}`">优选 CNAME <span class="field-note">选填，留空使用全局配置</span></label>
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
          <label class="field-label" :for="`main-${group.id}`">主域名 <span class="field-note">对外访问域名</span></label>
          <input :id="`main-${group.id}`" v-model="group.main_domain" class="vercel-input" placeholder="例如: kukie.cn" :class="{ 'input-error': group.errors.main_domain }" :disabled="submitting" @blur="validateGroup(group, 'main_domain')" />
          <span v-if="group.errors.main_domain" class="field-error">{{ group.errors.main_domain }}</span>
        </div>
        <div v-if="group.mode === 'preferred'" class="field">
          <label class="field-label" :for="`aux-${group.id}`">辅助域名 <span class="field-note">用作回源</span></label>
          <input :id="`aux-${group.id}`" v-model="group.aux_domain" class="vercel-input" placeholder="例如: fallback.169977.xyz" :class="{ 'input-error': group.errors.aux_domain }" :disabled="submitting" @blur="validateGroup(group, 'aux_domain')" />
          <span v-if="group.errors.aux_domain" class="field-error">{{ group.errors.aux_domain }}</span>
        </div>
      </div>
      <div v-if="group.result" class="group-result" :class="group.result.success ? 'success' : 'error'">
        <strong>{{ group.result.success ? '绑定成功' : '绑定失败' }}</strong>
        <span>{{ group.result.message }}</span>
        <span v-if="group.result.success && group.result.mode === 'preferred'">优选 CNAME：{{ group.result.preferred_cname }}</span>
      </div>
    </div>
    <div class="form-action">
      <button class="btn btn-primary" :disabled="submitting || !config.tunnel_id" @click="handleBatchBind">
        <svg v-if="submitting" class="spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
        <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
        {{ submitting ? '批量绑定中...' : `绑定 ${groups.length} 组域名` }}
      </button>
      <span v-if="summary" class="summary" :class="summary.success === summary.total ? 'summary-success' : 'summary-error'">{{ summary.success }}/{{ summary.total }} 组绑定成功</span>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useMessage } from 'naive-ui'
import { bindDomainsBatch, type BatchBindItem, type BatchBindResult } from '../api'
import CnamePicker from '../components/CNAMEPicker.vue'
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
  group.errors[field] = group[field].trim() ? '' : '此字段不能为空'
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
  if (!validateGroups()) {
    message.error('请补全每组的转发地址和域名')
    return
  }
  submitting.value = true
  try {
    const items = groups.value.map(({ mode, service_url, preferred_cname, main_domain, aux_domain }) => ({ mode, service_url: service_url.trim(), preferred_cname: preferred_cname.trim(), main_domain: main_domain.trim(), aux_domain: aux_domain.trim() }))
    const { data } = await bindDomainsBatch(items)
    groups.value.forEach((group, index) => {
      group.result = data.results[index] || { ...items[index], success: false, message: '未收到该组的执行结果' }
    })
    const successCount = data.results.filter((item) => item.success).length
    message[successCount === data.results.length ? 'success' : 'warning'](`批量绑定完成：${successCount}/${data.results.length} 成功`)
  } catch (e: any) {
    message.error('批量绑定失败: ' + (e.response?.data?.error || e.message))
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
.section { margin-bottom: var(--spacing-xl); }
.page-header { margin-bottom: var(--spacing-lg); }

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
@media (max-width: 720px) {
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
