<template>
  <fieldset class="dns-fields" :disabled="disabled">
    <label><span>{{ tr('m_8f6363220575') }}</span><n-select :value="model.type" :options="typeOptions" :disabled="disabled" @update:value="changeType" /></label>
    <label><span>{{ tr('m_d44e9b3d3b31') }}</span><n-input v-model:value="model.name" :disabled="disabled" :placeholder="model.type === 'SRV' ? '_sip._tcp.example.com' : tr('m_8be91866ed29')" /></label>
    <template v-if="model.type === 'SRV' && model.data">
      <label><span>{{ tr('m_d4dfa43887a7') }}</span><n-input-number v-model:value="model.data.priority" :disabled="disabled" :min="0" :max="65535" /></label>
      <label><span>{{ tr('m_db18831a0457') }}</span><n-input-number v-model:value="model.data.weight" :disabled="disabled" :min="0" :max="65535" /></label>
      <label><span>{{ tr('m_e71ac32b544b') }}</span><n-input-number v-model:value="model.data.port" :disabled="disabled" :min="0" :max="65535" /></label>
      <label><span>{{ tr('m_4e9a6d265824') }}</span><n-input v-model:value="model.data.target" :disabled="disabled" :placeholder="tr('m_bf962d5f502f')" /></label>
    </template>
    <template v-else-if="model.type === 'CAA' && model.data">
      <label><span>CAA flags</span><n-input-number v-model:value="model.data.flags" :disabled="disabled" :min="0" :max="255" /></label>
      <label><span>CAA tag</span><n-input v-model:value="model.data.tag" :disabled="disabled" placeholder="issue / issuewild / iodef" /></label>
      <label class="wide"><span>CAA value</span><n-input v-model:value="model.data.value" :disabled="disabled" placeholder="letsencrypt.org / mailto:security@example.com" /></label>
    </template>
    <label v-else class="wide"><span>{{ tr('m_5632a6a4e542') }}</span><n-input v-model:value="model.content" :disabled="disabled" type="textarea" :autosize="{ minRows: model.type === 'TXT' ? 3 : 1, maxRows: 6 }" :placeholder="contentPlaceholder" /></label>
    <label v-if="model.type === 'MX'"><span>{{ tr('m_3b1f7c9f5b21') }}</span><n-input-number v-model:value="model.priority" :disabled="disabled" :min="0" :max="65535" /></label>
    <label><span>{{ tr('m_5fada8da5e20') }}</span><n-input-number v-model:value="model.ttl" :disabled="disabled" :min="1" :max="86400" /></label>
    <label v-if="proxyEligible(model.type)"><span>{{ tr('m_b90665e74c01') }}</span><n-switch v-model:value="model.proxied" :disabled="disabled"><template #checked>{{ tr('m_5c7bbf98286f') }}</template><template #unchecked>{{ tr('m_80848ec9f9a4') }}</template></n-switch></label>
    <p v-if="model.type === 'NS'" class="wide hint">{{ tr('m_6dbed5aae4d1') }}</p>
  </fieldset>
</template>

<script setup lang="ts">
import { t as tr } from '../i18n'

import { computed } from 'vue'
import { NInput, NInputNumber, NSelect, NSwitch } from 'naive-ui'
import type { DNSRecordInput, DNSRecordType } from '../api'
import { defaultDNSData, DNS_TYPES, proxyEligible } from '../utils/dnsValidation'

const model = defineModel<DNSRecordInput>({ required: true })
defineProps<{ disabled?: boolean }>()
const typeOptions = DNS_TYPES.map(value => ({ label: value, value }))
const contentPlaceholder = computed(() => ({ A: '192.0.2.1', AAAA: '2001:db8::1', TXT: tr('m_a5c7d57bd1f8'), MX: 'mail.example.com', NS: 'ns.example.com', PTR: 'host.example.com', CNAME: 'target.example.com', SRV: '', CAA: '' }[model.value.type]))
function changeType(type: DNSRecordType) {
  model.value = { ...model.value, type, content: '', data: defaultDNSData(type), priority: type === 'MX' ? 0 : undefined, proxied: false }
}
</script>

<style scoped>
.dns-fields{display:grid;grid-template-columns:1fr 1fr;gap:16px;border:0;padding:0;margin:0;min-width:0}.dns-fields label{display:flex;flex-direction:column;gap:6px;min-width:0}.dns-fields label>span{color:var(--color-ink);font-size:13px;font-weight:600}.wide{grid-column:1/-1}.hint{margin:0;color:var(--color-mute);font-size:12px}@media(max-width:540px){.dns-fields{grid-template-columns:1fr}.wide{grid-column:auto}}
</style>
