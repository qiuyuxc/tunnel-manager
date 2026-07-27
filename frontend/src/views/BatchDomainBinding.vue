<template>
  <div class="page-container" style="padding-top: 0;">
    <div class="page-header">
      <router-link to="/domain" class="back-link">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
        返回域名绑定
      </router-link>
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

      <div class="form-fields">
        <div class="field">
          <label class="field-label" :for="`service-${group.id}`">转发地址</label>
          <input :id="`service-${group.id}`" v-model="group.service_url" class="vercel-input" placeholder="http://localhost:3000" :class="{ 'input-error': group.errors.service_url }" :disabled="submitting" @blur="validateGroup(group, 'service_url')" />
          <span v-if="group.errors.service_url" class="field-error">{{ group.errors.service_url }}</span>
        </div>
        <div class="field">
          <label class="field-label" :for="`cname-${group.id}`">优选 CNAME <span class="field-note">选填，留空使用全局配置</span></label>
          <input :id="`cname-${group.id}`" v-model="group.preferred_cname" class="vercel-input" placeholder="例如: cf.090227.xyz" :disabled="submitting" />
        </div>
        <div class="field">
          <label class="field-label" :for="`main-${group.id}`">主域名 <span class="field-note">对外访问域名</span></label>
          <input :id="`main-${group.id}`" v-model="group.main_domain" class="vercel-input" placeholder="例如: kukie.cn" :class="{ 'input-error': group.errors.main_domain }" :disabled="submitting" @blur="validateGroup(group, 'main_domain')" />
          <span v-if="group.errors.main_domain" class="field-error">{{ group.errors.main_domain }}</span>
        </div>
        <div class="field">
          <label class="field-label" :for="`aux-${group.id}`">辅助域名 <span class="field-note">用作回源</span></label>
          <input :id="`aux-${group.id}`" v-model="group.aux_domain" class="vercel-input" placeholder="例如: fallback.169977.xyz" :class="{ 'input-error': group.errors.aux_domain }" :disabled="submitting" @blur="validateGroup(group, 'aux_domain')" />
          <span v-if="group.errors.aux_domain" class="field-error">{{ group.errors.aux_domain }}</span>
        </div>
      </div>

      <div v-if="group.result" class="group-result" :class="group.result.success ? 'success' : 'error'">
        <strong>{{ group.result.success ? '绑定成功' : '绑定失败' }}</strong>
        <span>{{ group.result.message }}</span>
        <span v-if="group.result.success">优选 CNAME：{{ group.result.preferred_cname }}</span>
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
    for (const field of ['service_url', 'main_domain', 'aux_domain'] as Field[]) {
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
    const items = groups.value.map(({ service_url, preferred_cname, main_domain, aux_domain }) => ({ service_url: service_url.trim(), preferred_cname: preferred_cname.trim(), main_domain: main_domain.trim(), aux_domain: aux_domain.trim() }))
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
.page-header { margin-bottom: var(--spacing-lg); }
.section { margin-bottom: var(--spacing-md); }
.prereq-banner {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-sm);
  padding: var(--spacing-md);
  background: var(--color-banner-warning-bg);
  border: 1px solid var(--color-banner-warning-border);
  border-radius: var(--radius-md);
}
.prereq-title { font-size: 14px; font-weight: 600; color: var(--color-banner-warning-text); }
.prereq-desc { margin-top: 2px; font-size: 14px; color: var(--color-banner-warning-text); opacity: 0.84; }
.binding-group {
  padding: var(--spacing-lg);
  background: var(--color-canvas);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  box-shadow: 0 1px 2px rgba(58, 47, 34, 0.05);
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
.icon-button {
  display: inline-grid;
  width: 32px;
  height: 32px;
  place-items: center;
  color: var(--color-body);
  background: transparent;
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: transform 120ms ease-out, color 160ms ease-out, background-color 160ms ease-out, border-color 160ms ease-out, opacity 160ms ease-out;
}
.icon-button:hover:not(:disabled) { color: var(--color-ink); background: var(--color-canvas-soft); border-color: var(--color-hairline-strong); }
.icon-button:active:not(:disabled) { transform: scale(0.96); }
.icon-button.danger:hover:not(:disabled) { color: var(--color-error); border-color: var(--color-error); }
.icon-button:disabled { cursor: not-allowed; opacity: 0.45; }
.form-fields { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: var(--spacing-md); }
.field { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.field-label { margin-bottom: 4px; font-size: 14px; font-weight: 600; color: var(--color-ink); }
.field-note { margin-left: 4px; color: var(--color-mute); font-weight: 400; }
.field-error { color: var(--color-error); font-size: 12px; }
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
.group-result.success { color: var(--color-result-success-text); background: var(--color-result-success-bg); border: 1px solid var(--color-result-success-border); }
.group-result.error { color: var(--color-result-error-text); background: var(--color-result-error-bg); border: 1px solid var(--color-result-error-border); }
.form-action { display: flex; align-items: center; flex-wrap: wrap; gap: var(--spacing-sm); margin-top: var(--spacing-lg); padding-top: var(--spacing-lg); border-top: 1px solid var(--color-hairline); }
.summary { font-size: 14px; }
.summary-success { color: var(--color-success); }
.summary-error { color: var(--color-error); }
@keyframes spin { to { transform: rotate(360deg); } }
.spin { animation: spin 1s linear infinite; }
@media (max-width: 720px) { .form-fields { grid-template-columns: 1fr; } }
@media (max-width: 480px) {
  .group-header { align-items: stretch; flex-direction: column; }
  .group-actions { justify-content: flex-end; }
  .field-label { display: flex; flex-direction: column; gap: 2px; }
  .field-note { margin-left: 0; }
  .form-action .btn { width: 100%; justify-content: center; }
}
</style>
