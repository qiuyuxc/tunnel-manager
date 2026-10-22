<template>
  <n-modal :show="show" :mask-closable="!running" :close-on-esc="!running" :closable="!running" preset="card" :title="tr('m_8c63234bfc31')" class="dns-create-modal" @update:show="emit('update:show', $event)">
    <div class="batch-create">
      <p class="hint">{{ tr('m_cfad1e9ab75f') }}<strong>{{ zoneName }}</strong>{{ tr('m_0c1053a7c74d') }} {{ DNS_BATCH_LIMIT }} {{ tr('m_f367f1f8d0e7') }}</p>
      <div class="fields">
        <label><span>{{ tr('m_8f6363220575') }}</span><n-select v-model:value="form.type" :options="typeOptions" :disabled="running || locked" @update:value="changeType" /></label>
        <label><span>{{ tr('m_d44e9b3d3b31') }}</span><n-input v-model:value="form.name" :disabled="running || locked" :placeholder="tr('m_8be91866ed29')" /></label>
        <label><span>{{ tr('m_5fada8da5e20') }}</span><n-input-number v-model:value="form.ttl" :disabled="running || locked" :min="1" :max="86400" /></label>
        <label v-if="form.type === 'MX'"><span>{{ tr('m_774642733b2d') }}</span><n-input-number v-model:value="form.priority" :disabled="running || locked" :min="0" :max="65535" /></label>
        <label v-if="proxyEligible(form.type)"><span>{{ tr('m_b90665e74c01') }}</span><n-switch v-model:value="form.proxied" :disabled="running || locked" /></label>
      </div>
      <label class="values"><span>{{ tr('m_c72c82545f49') }}</span><n-input v-model:value="text" :disabled="running" type="textarea" :autosize="{ minRows: 4, maxRows: 10 }" placeholder="192.0.2.1&#10;192.0.2.2" /></label>
      <p class="hint">{{ tr('m_c7ab7396d63c') }}</p>
      <p v-if="validationError" class="error" role="alert">{{ validationError }}</p>
      <section class="preview" :aria-label="tr('m_608e8e278c60')">
        <strong>{{ tr('m_e628f1c80e48') }}{{ pending.length }} {{ tr('m_68fc37cbb496') }}</strong>
        <ol><li v-for="row in preview.rows" :key="row.line"><span>{{ tr('m_af50a0fe2a50') }} {{ row.line }} {{ tr('m_4cc7b00050cb') }}</span><code>{{ row.content }}</code><span :class="{ error: row.error }">{{ row.error || row.skipped || tr('m_b2d9abb23920') }}</span></li></ol>
      </section>
      <section v-if="results.length" class="preview" aria-live="polite" :aria-label="tr('m_b2a5aace9969')">
        <strong>{{ tr('m_cd9a30658205') }} {{ results.filter(row => row.success).length }}{{ tr('m_d064e2db58fc') }} {{ results.filter(row => !row.success && !row.uncertain && !row.notSent).length }}{{ tr('m_028d360b8187') }} {{ results.filter(row => row.uncertain).length }}{{ tr('m_754fd44ce51f') }} {{ results.filter(row => row.notSent).length }}</strong>
        <ol><li v-for="row in results" :key="row.line"><span>{{ tr('m_af50a0fe2a50') }} {{ row.line }} {{ tr('m_4cc7b00050cb') }}</span><code>{{ row.content }}</code><span :class="{ error: !row.success }">{{ row.message }}</span></li></ol>
        <p class="hint">{{ tr('m_659bde320fc0') }}</p>
        <n-checkbox v-if="reviewRequired" v-model:checked="reviewed" :disabled="running">{{ tr('m_6d1e1798dfa9') }}</n-checkbox>
      </section>
    </div>
    <template #footer><div class="actions"><button class="btn btn-secondary" :disabled="running" @click="emit('update:show', false)">{{ tr('m_3fd47edce45b') }}</button><button class="btn btn-primary" :disabled="running || !!validationError || !pending.length || (reviewRequired && !reviewed)" @click="submit">{{ running ? tr('m_bf7811d28e0e', { p0: results.length, p1: total }) : locked ? tr('m_a055dcc4abd8') : tr('m_cdc55080127d', { p0: pending.length }) }}</button></div></template>
  </n-modal>
</template>

<script setup lang="ts">
import { t as tr } from '../i18n'

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
  if (!active || !props.show || props.zoneID !== batchZone) contextError.value ||= tr('m_10b992ea213a')
  if (!token || token !== batchToken) contextError.value ||= tr('m_93dff9b1b92b')
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
const validationError = computed(() => contextError.value || preview.value.error || (preview.value.rows.some(row => row.error) ? tr('m_f32916472225') : '') || (pending.value.length ? validateDNSRecord({ ...form.value, content: pending.value[0].content }) : ''))
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
      if (!response.data?.id) throw new Error(tr('m_b488b6db4579'))
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
