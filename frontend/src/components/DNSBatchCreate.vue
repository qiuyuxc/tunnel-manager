<template>
  <n-modal :show="show" :mask-closable="!running" :close-on-esc="!running" :closable="!running" preset="card" title="批量新增 DNS 记录" class="dns-create-modal" @update:show="emit('update:show', $event)">
    <div class="batch-create">
      <p class="hint">区域：<strong>{{ zoneName }}</strong>。每批最多 {{ DNS_BATCH_LIMIT }} 条非空行，逐条串行新增。仅支持 A / AAAA / TXT / MX / NS / PTR；CNAME、SRV、CAA 请单条添加。</p>
      <div class="fields">
        <label><span>记录类型</span><n-select v-model:value="form.type" :options="typeOptions" :disabled="running || locked" @update:value="changeType" /></label>
        <label><span>名称</span><n-input v-model:value="form.name" :disabled="running || locked" placeholder="www 或 example.com" /></label>
        <label><span>TTL（1 为自动）</span><n-input-number v-model:value="form.ttl" :disabled="running || locked" :min="1" :max="86400" /></label>
        <label v-if="form.type === 'MX'"><span>MX 优先级（共用）</span><n-input-number v-model:value="form.priority" :disabled="running || locked" :min="0" :max="65535" /></label>
        <label v-if="proxyEligible(form.type)"><span>代理状态</span><n-switch v-model:value="form.proxied" :disabled="running || locked" /></label>
      </div>
      <label class="values"><span>解析值，每行一条</span><n-input v-model:value="text" :disabled="running" type="textarea" :autosize="{ minRows: 4, maxRows: 10 }" placeholder="192.0.2.1&#10;192.0.2.2" /></label>
      <p class="hint">空行忽略，重复值及已加载的同名同类型值跳过。TXT 保留空格，不按逗号拆分；一行是一个完整值。跨行 TXT 请单条添加。</p>
      <p v-if="validationError" class="error" role="alert">{{ validationError }}</p>
      <section class="preview" aria-label="逐行预览">
        <strong>预览：{{ pending.length }} 条待新增</strong>
        <ol><li v-for="row in preview.rows" :key="row.line"><span>第 {{ row.line }} 行</span><code>{{ row.content }}</code><span :class="{ error: row.error }">{{ row.error || row.skipped || '待新增' }}</span></li></ol>
      </section>
      <section v-if="results.length" class="preview" aria-live="polite" aria-label="逐条结果">
        <strong>本次结果：成功 {{ results.filter(row => row.success).length }}，失败 {{ results.filter(row => !row.success && !row.uncertain && !row.notSent).length }}，未知 {{ results.filter(row => row.uncertain).length }}，未发送 {{ results.filter(row => row.notSent).length }}</strong>
        <ol><li v-for="row in results" :key="row.line"><span>第 {{ row.line }} 行</span><code>{{ row.content }}</code><span :class="{ error: !row.success }">{{ row.message }}</span></li></ol>
        <p class="hint">输入框仅保留失败、未知和未发送项；不会自动重发成功行。结果未知时队列立即停止，请在 Cloudflare 或刷新记录核对，删掉实际已创建的行后再重试。关闭窗口会清空本批输入。</p>
        <n-checkbox v-if="reviewRequired" v-model:checked="reviewed" :disabled="running">我已核对未知结果，并从输入框移除实际已创建的行</n-checkbox>
      </section>
    </div>
    <template #footer><div class="actions"><button class="btn btn-secondary" :disabled="running" @click="emit('update:show', false)">关闭</button><button class="btn btn-primary" :disabled="running || !!validationError || !pending.length || (reviewRequired && !reviewed)" @click="submit">{{ running ? `正在新增 ${results.length}/${total}` : locked ? '重试剩余项' : `新增 ${pending.length} 条记录` }}</button></div></template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { NCheckbox, NInput, NInputNumber, NModal, NSelect, NSwitch } from 'naive-ui'
import { createDNSRecord, type DNSRecord, type DNSRecordInput } from '../api'
import { DNS_BATCH_LIMIT, DNS_BATCH_TYPES, DNSBatchStoppedError, normalizeDNSInput, parseDNSBatch, proxyEligible, runDNSBatch, validateDNSRecord, type DNSBatchResult } from '../utils/dnsValidation'

const props = defineProps<{ show: boolean; zoneID: string; zoneName: string; records: DNSRecord[] }>()
const emit = defineEmits<{ 'update:show': [value: boolean]; saved: [zoneID: string] }>()
const blank = (): DNSRecordInput => ({ type: 'A', name: '', content: '', ttl: 1, proxied: false })
const form = ref(blank()), text = ref(''), running = ref(false), locked = ref(false), total = ref(0)
const results = ref<DNSBatchResult[]>([]), successfulValues = ref<string[]>([])
const reviewRequired = ref(false), reviewed = ref(false)
const contextError = ref('')
let active = true, batchToken: string | null = null, batchZone = ''
onBeforeUnmount(() => { active = false })
function validateSession(token: string | null) {
  if (!active || !props.show || props.zoneID !== batchZone) contextError.value ||= '页面或区域已变更，请重新打开批量新增'
  if (!token || token !== batchToken) contextError.value ||= '登录会话已变更，请关闭窗口并刷新后重试'
  if (contextError.value) throw new DNSBatchStoppedError(contextError.value)
}
const typeOptions = DNS_BATCH_TYPES.map(value => ({ label: value, value }))
const absoluteName = computed(() => {
  const name = form.value.name.trim().toLowerCase().replace(/\.$/, '')
  const zone = props.zoneName.toLowerCase()
  return name === '@' ? zone : name === zone || name.endsWith(`.${zone}`) ? name : `${name}.${zone}`
})
const preview = computed(() => {
  const parsed = parseDNSBatch(form.value.type, text.value, [
    ...successfulValues.value,
    ...props.records.filter(record => record.type === form.value.type && record.name.toLowerCase().replace(/\.$/, '') === absoluteName.value).map(record => record.content),
  ])
  for (const row of parsed.rows) if (!row.error && !row.skipped) row.error = validateDNSRecord({ ...form.value, content: row.content }) || undefined
  return parsed
})
const pending = computed(() => preview.value.rows.filter(row => !row.skipped && !row.error))
const validationError = computed(() => contextError.value || preview.value.error || (preview.value.rows.some(row => row.error) ? '请先修正标注行，整个批次尚未提交' : '') || (pending.value.length ? validateDNSRecord({ ...form.value, content: pending.value[0].content }) : ''))
function changeType() { form.value.proxied = false; form.value.priority = form.value.type === 'MX' ? 0 : undefined }
watch(() => props.show, show => { if (show) { form.value = blank(); text.value = ''; results.value = []; successfulValues.value = []; locked.value = false; reviewRequired.value = false; reviewed.value = false; contextError.value = ''; batchToken = localStorage.getItem('auth_token'); batchZone = props.zoneID } })
async function submit() {
  if (running.value || validationError.value || !pending.value.length || (reviewRequired.value && !reviewed.value)) return
  const rows = pending.value.map(row => ({ ...row })), zone = props.zoneID, base = { ...form.value }
  running.value = true; locked.value = true; results.value = []; total.value = rows.length
  reviewRequired.value = false; reviewed.value = false
  try {
    const completed = await runDNSBatch(rows, async row => {
      const response = await createDNSRecord(zone, normalizeDNSInput({ ...base, content: row.content }), validateSession)
      if (!response.data?.id) throw new Error('parse response failed: 缺少创建记录确认')
    }, result => {
      results.value.push(result)
      if (result.success) successfulValues.value.push(result.content)
      if (result.uncertain) reviewRequired.value = true
    })
    text.value = completed.filter(row => !row.success).map(row => row.content).join('\n')
    if (active && !contextError.value && batchToken === localStorage.getItem('auth_token')) emit('saved', zone)
  } finally { running.value = false }
}
</script>

<style scoped>
.batch-create{display:flex;flex-direction:column;gap:14px}.fields{display:grid;grid-template-columns:1fr 1fr;gap:14px}.fields label,.values{display:flex;flex-direction:column;gap:6px;min-width:0}.hint{margin:0;color:var(--color-mute);font-size:12px;line-height:1.6}.preview{padding:12px;background:var(--color-canvas-soft);border:1px solid var(--color-hairline);border-radius:var(--radius-sm)}.preview ol{padding:0;list-style:none;max-height:230px;overflow:auto}.preview li{display:grid;grid-template-columns:64px 1fr;gap:4px 8px;padding:8px 0;border-top:1px solid var(--color-hairline);font-size:12px}.preview code{overflow-wrap:anywhere;white-space:pre-wrap}.preview li>span:last-child{grid-column:2;overflow-wrap:anywhere}.error{color:var(--color-error)}.actions{display:flex;justify-content:flex-end;gap:8px}@media(max-width:540px){.fields{grid-template-columns:1fr}.actions .btn{flex:1;justify-content:center}}
</style>

<style>
.dns-create-modal{width:min(680px,calc(100vw - 32px))!important;max-height:calc(100dvh - 32px);display:flex;flex-direction:column;overflow:hidden}.dns-create-modal>.n-card-content{flex:1 1 auto;overflow:auto;min-height:0;overscroll-behavior:contain}.dns-create-modal>.n-card-header,.dns-create-modal>.n-card__footer{flex-shrink:0}
</style>
