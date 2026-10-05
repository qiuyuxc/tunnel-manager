<template>
  <fieldset class="dns-fields" :disabled="disabled">
    <label><span>记录类型</span><n-select :value="model.type" :options="typeOptions" :disabled="disabled" @update:value="changeType" /></label>
    <label><span>名称</span><n-input v-model:value="model.name" :disabled="disabled" :placeholder="model.type === 'SRV' ? '_sip._tcp.example.com' : 'www 或 example.com'" /></label>
    <template v-if="model.type === 'SRV' && model.data">
      <label><span>SRV 优先级</span><n-input-number v-model:value="model.data.priority" :disabled="disabled" :min="0" :max="65535" /></label>
      <label><span>权重</span><n-input-number v-model:value="model.data.weight" :disabled="disabled" :min="0" :max="65535" /></label>
      <label><span>端口</span><n-input-number v-model:value="model.data.port" :disabled="disabled" :min="0" :max="65535" /></label>
      <label><span>服务目标</span><n-input v-model:value="model.data.target" :disabled="disabled" placeholder="sip.example.com 或 ." /></label>
    </template>
    <template v-else-if="model.type === 'CAA' && model.data">
      <label><span>CAA flags</span><n-input-number v-model:value="model.data.flags" :disabled="disabled" :min="0" :max="255" /></label>
      <label><span>CAA tag</span><n-input v-model:value="model.data.tag" :disabled="disabled" placeholder="issue / issuewild / iodef" /></label>
      <label class="wide"><span>CAA value</span><n-input v-model:value="model.data.value" :disabled="disabled" placeholder="letsencrypt.org / mailto:security@example.com" /></label>
    </template>
    <label v-else class="wide"><span>解析值</span><n-input v-model:value="model.content" :disabled="disabled" type="textarea" :autosize="{ minRows: model.type === 'TXT' ? 3 : 1, maxRows: 6 }" :placeholder="contentPlaceholder" /></label>
    <label v-if="model.type === 'MX'"><span>MX 优先级</span><n-input-number v-model:value="model.priority" :disabled="disabled" :min="0" :max="65535" /></label>
    <label><span>TTL（1 为自动）</span><n-input-number v-model:value="model.ttl" :disabled="disabled" :min="1" :max="86400" /></label>
    <label v-if="proxyEligible(model.type)"><span>代理状态</span><n-switch v-model:value="model.proxied" :disabled="disabled"><template #checked>已代理</template><template #unchecked>仅 DNS</template></n-switch></label>
    <p v-if="model.type === 'NS'" class="wide hint">NS 用于子域委派，不会修改注册商处的权威名称服务器。</p>
  </fieldset>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NInput, NInputNumber, NSelect, NSwitch } from 'naive-ui'
import type { DNSRecordInput, DNSRecordType } from '../api'
import { defaultDNSData, DNS_TYPES, proxyEligible } from '../utils/dnsValidation'

const model = defineModel<DNSRecordInput>({ required: true })
defineProps<{ disabled?: boolean }>()
const typeOptions = DNS_TYPES.map(value => ({ label: value, value }))
const contentPlaceholder = computed(() => ({ A: '192.0.2.1', AAAA: '2001:db8::1', TXT: '保留原样的文本', MX: 'mail.example.com', NS: 'ns.example.com', PTR: 'host.example.com', CNAME: 'target.example.com', SRV: '', CAA: '' }[model.value.type]))
function changeType(type: DNSRecordType) {
  model.value = { ...model.value, type, content: '', data: defaultDNSData(type), priority: type === 'MX' ? 0 : undefined, proxied: false }
}
</script>

<style scoped>
.dns-fields{display:grid;grid-template-columns:1fr 1fr;gap:16px;border:0;padding:0;margin:0;min-width:0}.dns-fields label{display:flex;flex-direction:column;gap:6px;min-width:0}.dns-fields label>span{color:var(--color-ink);font-size:13px;font-weight:600}.wide{grid-column:1/-1}.hint{margin:0;color:var(--color-mute);font-size:12px}@media(max-width:540px){.dns-fields{grid-template-columns:1fr}.wide{grid-column:auto}}
</style>
