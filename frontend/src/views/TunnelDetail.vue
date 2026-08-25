<template>
  <div class="page-container">
    <div class="page-header">
      <router-link to="/tunnels" class="back-link">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
        返回隧道列表
      </router-link>
      <div class="detail-title-row">
        <div>
          <h2>隧道详情</h2>
          <p v-if="detail" class="detail-subtitle">{{ detail.name }}</p>
        </div>
      </div>
    </div>
    <div v-if="loading" class="empty-state">
      <svg class="spin" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="var(--color-mute)" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
      <span class="empty-text">加载中...</span>
    </div>
    <template v-else-if="detail">
      <!-- Tunnel Info -->
      <div class="info-grid section">
        <div class="info-card" :class="{ '': visible }" style="animation-delay: 0.08s;">
          <span class="info-label caption-mono">名称</span>
          <span class="info-value">{{ detail.name }}</span>
        </div>
        <div class="info-card" :class="{ '': visible }" style="animation-delay: 0.12s;">
          <span class="info-label caption-mono">隧道 ID</span>
          <code class="inline-code">{{ detail.id }}</code>
        </div>
        <div class="info-card" :class="{ '': visible }" style="animation-delay: 0.16s;">
          <span class="info-label caption-mono">状态</span>
          <span class="status-tag" :class="detail.status">{{ detail.status }}</span>
        </div>
      </div>
      <!-- Ingress Routes -->
      <div class="section route-section">
        <div class="section-header">
          <span class="caption-mono section-label">已发布应用程序路由</span>
          <div class="section-actions">
            <span class="route-count">{{ routes.length }} 条规则</span>
            <button class="btn btn-primary" @click="startAdd">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
              新增路由
            </button>
          </div>
        </div>
        <!-- Add/Edit Modal -->
        <n-modal v-model:show="showForm" preset="card" :title="editing ? '修改路由' : '新增路由'" class="route-modal" :bordered="false" :segmented="{ content: true, footer: true }" :auto-focus="false">
          <div class="modal-form">
            <div class="form-field">
              <label class="form-label">主机名</label>
              <input v-model="form.hostname" placeholder="example.com" class="vercel-input" />
            </div>
            <div class="form-field">
              <label class="form-label">服务地址</label>
              <input v-model="form.service" placeholder="http://localhost:3000" class="vercel-input" />
            </div>
          </div>
          <template #footer>
            <div class="modal-footer">
              <button class="btn btn-ghost" @click="showForm = false">取消</button>
              <button class="btn btn-primary" :disabled="saving || !form.hostname || !form.service" @click="submitForm">
                {{ saving ? '保存中...' : (editing ? '更新' : '添加') }}
              </button>
            </div>
          </template>
        </n-modal>
        <!-- Delete Confirmation -->
        <n-modal v-model:show="showDelete" preset="card" title="删除路由" class="route-modal" :bordered="false" :segmented="{ content: true, footer: true }" :auto-focus="false" :mask-closable="!deleting">
          <div class="modal-form">
            <p class="delete-hint">确定删除路由 <code>{{ deleteTarget?.hostname }}</code> 吗？此操作无法撤销。</p>
            <n-checkbox v-model:checked="deleteDNS">同时删除该主机名对应的 DNS 记录</n-checkbox>
          </div>
          <template #footer>
            <div class="modal-footer">
              <button class="btn btn-ghost" :disabled="deleting" @click="showDelete = false">取消</button>
              <button class="btn btn-danger" :disabled="deleting" @click="submitDelete">
                {{ deleting ? '删除中...' : '确认删除' }}
              </button>
            </div>
          </template>
        </n-modal>
        <div v-if="routes.length > 0" class="route-grid">
          <div v-for="(rule, idx) in routes" :key="idx" class="route-card" :class="{ '': visible }" :style="{ animationDelay: `${0.2 + idx * 0.04}s` }">
            <div class="route-icon">
              <svg v-if="rule.hostname" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
              <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M9 3v18"/><path d="M3 9h6"/></svg>
            </div>
            <div class="route-body">
              <div class="route-hostname">
                <template v-if="rule.hostname">{{ rule.hostname }}</template>
                <em v-else class="route-catchall">Catch-all</em>
              </div>
              <div class="route-service">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="var(--color-mute)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>
                <code>{{ rule.service }}</code>
              </div>
            </div>
            <div v-if="rule.hostname" class="route-actions">
              <button class="btn-icon" @click="startEdit(rule)" title="编辑">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
              </button>
              <button class="btn-icon btn-icon-danger" @click="startDelete(rule)" title="删除">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/><path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/></svg>
              </button>
            </div>
          </div>
        </div>
        <div v-else class="empty-state">
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="var(--color-mute)" stroke-width="1" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M9 3v18"/><path d="M3 9h6"/></svg>
          <span class="empty-text">暂无路由规则</span>
        </div>
      </div>
    </template>
    <div v-else class="empty-state">
      <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="var(--color-mute)" stroke-width="1" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 8v4"/><path d="M12 16h.01"/></svg>
      <span class="empty-text">无法加载隧道详情</span>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useMessage, NModal, NCheckbox } from 'naive-ui'
import { getTunnelDetail, addIngressRule, updateIngressRule, deleteIngressRule, type TunnelDetail as TunnelDetailType, type IngressRule } from '../api'
const route = useRoute()
const message = useMessage()
const detail = ref<TunnelDetailType | null>(null)
const loading = ref(true)
const visible = ref(false)
const routes = computed(() => detail.value?.ingress ?? [])
// Form state
const showForm = ref(false)
const editing = ref(false)
const saving = ref(false)
const editOldHostname = ref('')
const form = ref({ hostname: '', service: '' })
// Delete state
const showDelete = ref(false)
const deleting = ref(false)
const deleteDNS = ref(true)
const deleteTarget = ref<IngressRule | null>(null)
async function load() {
  loading.value = true
  try {
    const { data } = await getTunnelDetail(route.params.id as string)
    detail.value = data
    requestAnimationFrame(() => { visible.value = true })
  } catch (e: any) {
    message.error('获取隧道详情失败: ' + (e.response?.data?.error || e.message))
  } finally {
    loading.value = false
  }
}
function startAdd() {
  editing.value = false
  editOldHostname.value = ''
  form.value = { hostname: '', service: '' }
  showForm.value = true
}
function startEdit(rule: IngressRule) {
  editing.value = true
  editOldHostname.value = rule.hostname || ''
  form.value = { hostname: rule.hostname || '', service: rule.service }
  showForm.value = true
}
async function submitForm() {
  saving.value = true
  const tunnelID = route.params.id as string
  try {
    if (editing.value) {
      await updateIngressRule(tunnelID, editOldHostname.value, form.value.hostname, form.value.service)
      message.success('路由已更新')
    } else {
      await addIngressRule(tunnelID, form.value.hostname, form.value.service)
      message.success('路由已添加')
    }
    showForm.value = false
    await load()
  } catch (e: any) {
    message.error('操作失败: ' + (e.response?.data?.error || e.message))
  } finally {
    saving.value = false
  }
}
function startDelete(rule: IngressRule) {
  deleteTarget.value = rule
  deleteDNS.value = true
  showDelete.value = true
}
async function submitDelete() {
  const hostname = deleteTarget.value?.hostname
  if (!hostname) return
  deleting.value = true
  try {
    const { data } = await deleteIngressRule(route.params.id as string, hostname, deleteDNS.value)
    if (data.dns_warning) {
      message.warning(`路由已删除，但 DNS 记录未清理: ${data.dns_warning}`)
    } else if (deleteDNS.value) {
      message.success(`路由已删除，同时清理了 ${data.dns_deleted ?? 0} 条 DNS 记录`)
    } else {
      message.success('路由已删除')
    }
    showDelete.value = false
    await load()
  } catch (e: any) {
    message.error('删除失败: ' + (e.response?.data?.error || e.message))
  } finally {
    deleting.value = false
  }
}
onMounted(() => { load() })
</script>
<style scoped>
.page-header { margin-bottom: var(--spacing-xl); }
.back-link { display: inline-flex; align-items: center; gap: 6px; margin-bottom: 12px; color: var(--color-mute); font-size: 12px; font-weight: 500; text-decoration: none; }
.back-link:hover { color: var(--color-link); }
.detail-title-row { display: flex; align-items: flex-end; justify-content: space-between; gap: var(--spacing-lg); }
.detail-title-row h2 { margin-bottom: 4px; }
.detail-subtitle { margin: 0; color: var(--color-body); font-size: 13px; }
.section { margin-bottom: var(--spacing-xl); }
.info-grid { display: grid; grid-template-columns: minmax(180px, 0.8fr) minmax(280px, 1.5fr) minmax(160px, 0.7fr); gap: var(--spacing-lg); margin-bottom: var(--spacing-xl); }
.info-card { display: flex; flex-direction: column; gap: 8px; min-width: 0; padding: var(--spacing-lg); background: var(--color-canvas-raised); border: 1px solid var(--color-hairline); border-radius: var(--radius-lg); box-shadow: 0 1px 3px rgba(15, 23, 42, 0.06); }
.info-label { color: var(--color-mute); font-size: 12px; font-weight: 500; }
.info-value { color: var(--color-ink); font-size: 16px; font-weight: 600; overflow-wrap: anywhere; }
.info-card .inline-code { display: block; max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.section-header { display: flex; align-items: center; justify-content: space-between; gap: var(--spacing-md); margin-bottom: var(--spacing-md); }
.section-label { color: var(--color-ink); font-size: 14px; font-weight: 600; }
.section-actions { display: flex; align-items: center; gap: var(--spacing-sm); }
.route-count { display: inline-flex; align-items: center; height: 28px; padding: 0 10px; color: var(--color-mute); background: var(--color-canvas-soft); border: 1px solid var(--color-hairline); border-radius: 999px; font-family: var(--font-mono); font-size: 12px; }
.route-grid { display: flex; flex-direction: column; gap: 1px; overflow: hidden; background: var(--color-hairline); border: 1px solid var(--color-hairline); border-radius: var(--radius-lg); box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08); }
.route-card { display: grid; grid-template-columns: 40px minmax(180px, 0.7fr) minmax(240px, 1fr) auto; align-items: center; gap: var(--spacing-md); min-width: 0; padding: 14px 16px; background: var(--color-canvas-raised); transition: background-color 140ms ease-out; }
.route-card:hover { background: var(--color-canvas-soft); }
.route-icon { display: inline-flex; align-items: center; justify-content: center; width: 34px; height: 34px; color: var(--color-link); background: color-mix(in srgb, var(--color-link) 9%, var(--color-canvas-raised)); border: 1px solid color-mix(in srgb, var(--color-link) 18%, var(--color-hairline)); border-radius: var(--radius-md); }
.route-body { display: contents; }
.route-hostname { min-width: 0; color: var(--color-ink); font-size: 14px; font-weight: 600; overflow-wrap: anywhere; }
.route-catchall { color: var(--color-mute); font-style: italic; }
.route-service { display: flex; align-items: center; gap: 8px; min-width: 0; color: var(--color-mute); font-size: 12px; }
.route-service code { min-width: 0; overflow: hidden; color: var(--color-body); font-family: var(--font-mono); font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
.route-service svg { flex-shrink: 0; color: var(--color-mute); }
.route-actions { display: flex; align-items: center; gap: 6px; }
.btn-icon { display: inline-flex; align-items: center; justify-content: center; width: 32px; height: 32px; color: var(--color-mute); background: transparent; border: 1px solid var(--color-hairline); border-radius: var(--radius-md); cursor: pointer; transition: transform 120ms ease-out, background-color 140ms ease-out, border-color 140ms ease-out, color 140ms ease-out; }
.btn-icon:hover { color: var(--color-ink); background: var(--color-canvas-soft); border-color: var(--color-hairline-strong); }
.btn-icon:active { transform: scale(0.96); }
.btn-icon-danger:hover { color: var(--color-error); border-color: var(--color-error); background: var(--color-status-down-bg); }
.status-tag { display: inline-flex; align-items: center; align-self: flex-start; height: 24px; padding: 0 10px; border-radius: 999px; font-family: var(--font-mono); font-size: 12px; text-transform: uppercase; }
.status-tag.healthy { color: var(--color-status-healthy-text); background: var(--color-status-healthy-bg); border: 1px solid var(--color-status-healthy-border); }
.status-tag.degraded { color: var(--color-status-degraded-text); background: var(--color-status-degraded-bg); border: 1px solid var(--color-status-degraded-border); }
.status-tag.down, .status-tag.inactive { color: var(--color-status-down-text); background: var(--color-status-down-bg); border: 1px solid var(--color-status-down-border); }
.empty-state { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: var(--spacing-sm); padding: var(--spacing-3xl) var(--spacing-lg); background: var(--color-canvas-raised); border: 1px solid var(--color-hairline); border-radius: var(--radius-lg); }
.empty-text { color: var(--color-body); font-size: 16px; font-weight: 500; }
.modal-form { display: flex; flex-direction: column; gap: var(--spacing-md); }
.form-label { display: block; margin-bottom: 6px; color: var(--color-mute); font-size: 12px; font-weight: 500; }
.vercel-input { width: 100%; height: 36px; box-sizing: border-box; }
.modal-footer { display: flex; justify-content: flex-end; gap: 10px; }
.route-modal { width: 100%; max-width: 480px; }
.delete-hint { margin: 0; color: var(--color-body); font-size: 14px; line-height: 1.7; }
.delete-hint code { padding: 1px 5px; color: var(--color-ink); background: var(--color-canvas-soft-2); border-radius: 4px; font-family: var(--font-mono); font-size: 13px; overflow-wrap: anywhere; }
@keyframes spin { to { transform: rotate(360deg); } }
.spin { animation: spin 1s linear infinite; }
@media (max-width: 860px) {
  .info-grid { grid-template-columns: 1fr 1fr; }
  .info-card:nth-child(2) { grid-column: span 2; }
  .route-card { grid-template-columns: 36px minmax(0, 1fr) auto; }
  .route-body { display: flex; flex-direction: column; gap: 5px; min-width: 0; }
  .route-service { grid-column: 2 / -1; }
}
@media (max-width: 600px) {
  .info-grid { grid-template-columns: 1fr; }
  .info-card:nth-child(2) { grid-column: auto; }
  .detail-title-row { align-items: flex-start; flex-direction: column; }
  .section-header { align-items: flex-start; flex-direction: column; }
  .section-actions { width: 100%; justify-content: space-between; }
  .route-card { grid-template-columns: 34px minmax(0, 1fr) auto; padding: 13px 12px; }
  .route-service { grid-column: 2 / -1; }
  .route-service code { white-space: normal; overflow-wrap: anywhere; }
}
</style>
