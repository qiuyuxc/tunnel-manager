<template>
  <div class="page-container" style="padding-top: 0;">
    <div class="page-header">
      <router-link to="/tunnels" class="back-link">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
        返回隧道列表
      </router-link>
      <h2>隧道详情</h2>
    </div>

    <div v-if="loading" class="empty-state">
      <svg class="spin" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="var(--color-mute)" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
      <span class="empty-text">加载中...</span>
    </div>

    <template v-else-if="detail">
      <!-- Tunnel Info -->
      <div class="info-grid section">
        <div class="info-card card-transition" :class="{ 'stagger-item': visible }" style="animation-delay: 0.08s;">
          <span class="info-label caption-mono">名称</span>
          <span class="info-value">{{ detail.name }}</span>
        </div>
        <div class="info-card card-transition" :class="{ 'stagger-item': visible }" style="animation-delay: 0.12s;">
          <span class="info-label caption-mono">隧道 ID</span>
          <code class="inline-code">{{ detail.id }}</code>
        </div>
        <div class="info-card card-transition" :class="{ 'stagger-item': visible }" style="animation-delay: 0.16s;">
          <span class="info-label caption-mono">状态</span>
          <span class="status-tag" :class="detail.status">{{ detail.status }}</span>
        </div>
      </div>

      <!-- Ingress Routes -->
      <div class="section">
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

        <div v-if="routes.length > 0" class="route-grid">
          <div v-for="(rule, idx) in routes" :key="idx" class="route-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: `${0.2 + idx * 0.04}s` }">
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
            <button v-if="rule.hostname" class="btn-icon" @click="startEdit(rule)" title="编辑">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
            </button>
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
import { useMessage, NModal } from 'naive-ui'
import { getTunnelDetail, addIngressRule, updateIngressRule, type TunnelDetail as TunnelDetailType, type IngressRule } from '../api'

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

onMounted(() => { load() })
</script>

<style scoped>
.page-header { margin-bottom: var(--spacing-lg); }
.section { margin-bottom: var(--spacing-md); }

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: var(--spacing-md);
}
.info-card {
  background: var(--color-canvas);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
  box-shadow: 0 1px 2px rgba(58, 47, 34, 0.05);
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.info-label {
  color: var(--color-mute);
}
.info-value {
  font-weight: 500;
  color: var(--color-ink);
  word-break: break-all;
}

.section-label { color: var(--color-mute); }
.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--spacing-md);
}
.route-count {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--color-mute);
  background: var(--color-canvas-soft-2);
  padding: 0 8px;
  height: 22px;
  line-height: 22px;
  border-radius: 9999px;
}

.route-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--spacing-md);
}
.route-card {
  position: relative;
  background: var(--color-canvas);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
  box-shadow: 0 1px 2px rgba(58, 47, 34, 0.05);
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-md);
  transition: border-color 160ms ease-out, box-shadow 180ms ease-out, transform 180ms ease-out;
}
.route-card:hover {
  border-color: var(--color-hairline-strong);
  box-shadow: 0 12px 28px rgba(58, 47, 34, 0.08);
  transform: translateY(-1px);
}
.route-icon {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  background: var(--color-canvas-soft-2);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--color-body);
}
.route-body { min-width: 0; flex: 1; }
.route-hostname {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-ink);
  margin-bottom: 4px;
  word-break: break-all;
}
.route-catchall {
  color: var(--color-mute);
  font-style: italic;
}
.route-service {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}
.route-service code {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--color-mute);
  word-break: break-all;
}

.status-tag {
  font-family: var(--font-mono);
  font-size: 12px;
  padding: 0 8px;
  height: 22px;
  line-height: 22px;
  border-radius: 9999px;
  text-transform: uppercase;
  align-self: flex-start;
}
.status-tag.healthy {
  background: var(--color-status-healthy-bg);
  color: var(--color-status-healthy-text);
  border: 1px solid var(--color-status-healthy-border);
}
.status-tag.degraded {
  background: var(--color-status-degraded-bg);
  color: var(--color-status-degraded-text);
  border: 1px solid var(--color-status-degraded-border);
}
.status-tag.down,
.status-tag.inactive {
  background: var(--color-status-down-bg);
  color: var(--color-status-down-text);
  border: 1px solid var(--color-status-down-border);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-3xl) var(--spacing-lg);
  background: var(--color-canvas);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  box-shadow: 0 1px 2px rgba(58, 47, 34, 0.05);
  gap: var(--spacing-sm);
}
.empty-text { color: var(--color-body); font-size: 16px; font-weight: 500; }

@keyframes spin { to { transform: rotate(360deg); } }
.spin { animation: spin 1s linear infinite; }

@media (max-width: 768px) {
  .info-grid { grid-template-columns: 1fr; }
  .route-grid { grid-template-columns: 1fr; }
  .route-card { padding: var(--spacing-md); }
  .section-actions { flex-wrap: wrap; }
}

.section-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.modal-form {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}
.form-label {
  display: block;
  font-size: 12px;
  color: var(--color-mute);
  margin-bottom: 6px;
  font-family: var(--font-mono);
  text-transform: uppercase;
  letter-spacing: 0.4px;
}
.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

/* Modal size */
.route-modal {
  max-width: 480px;
  width: 100%;
}
@media (max-width: 768px) {
  .route-modal {
    max-width: calc(100vw - 2 * var(--spacing-md));
  }
}

/* Speed up modal transition */
:deep(.n-modal) {
  --n-duration: 0.15s;
}
:deep(.n-mask) {
  --n-duration: 0.15s;
}

.btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-hairline);
  background: transparent;
  color: var(--color-mute);
  cursor: pointer;
  flex-shrink: 0;
  transition: transform 120ms ease-out, background-color 160ms ease-out, border-color 160ms ease-out, color 160ms ease-out;
}
.btn-icon:hover {
  border-color: var(--color-hairline-strong);
  background: var(--color-canvas-soft);
  color: var(--color-ink);
}
.btn-icon:active { transform: scale(0.96); }
</style>
