<template>
  <fieldset v-if="Array.isArray(modelValue)">
    <legend>{{ label }}</legend>
    <div v-for="(item, index) in modelValue" :key="index" class="array-item">
      <AssistantArgumentField :label="tr('m_ad9950fb4a2c', { p0: index + 1 })" :model-value="item" @update:model-value="setItem(index, $event)" />
      <button type="button" @click="removeItem(index)">{{ tr('m_b80f8ec9d893') }}</button>
    </div>
    <button v-if="field === 'items'" type="button" @click="emit('update:modelValue', [...modelValue, { name: '', value: '' }])">{{ tr('m_1073a088d1a0') }}</button>
  </fieldset>
  <fieldset v-else-if="modelValue !== null && typeof modelValue === 'object'">
    <legend>{{ label }}</legend>
    <AssistantArgumentField v-for="(value, key) in modelValue" :key="key" :field="String(key)" :label="labels[key] || String(key)" :model-value="value" @update:model-value="emit('update:modelValue', { ...modelValue, [key]: $event })" />
  </fieldset>
  <label v-else>
    <span>{{ label }}</span>
    <select v-if="typeof modelValue === 'boolean'" :value="String(modelValue)" @change="emit('update:modelValue', ($event.target as HTMLSelectElement).value === 'true')"><option value="true">{{ tr('m_8da97ddda990') }}</option><option value="false">{{ tr('m_3fd47edce45b') }}</option></select>
    <select v-else-if="field === 'invite_mode'" :value="modelValue" @change="emit('update:modelValue', ($event.target as HTMLSelectElement).value)"><option value="off">{{ tr('m_aeec6e7aa8ee') }}</option><option value="optional">{{ tr('m_0345fc011359') }}</option><option value="required">{{ tr('m_478c37677b78') }}</option></select>
    <input v-else-if="typeof modelValue === 'number' || ['ttl', 'interval_sec', 'priority', 'weight', 'port', 'flags'].includes(field || '')" type="number" step="1" required :value="modelValue" @input="emit('update:modelValue', ($event.target as HTMLInputElement).value === '' ? '' : Number(($event.target as HTMLInputElement).value))" />
    <textarea v-else-if="field === 'content' || field === 'description'" rows="2" :value="String(modelValue ?? '')" @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)" />
    <input v-else type="text" :value="String(modelValue ?? '')" @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)" />
  </label>
</template>

<script setup lang="ts">
import { t as tr } from '../i18n'

import { assistantArgumentLabels as labels } from '../utils/assistantTaskEdits'
const props = defineProps<{ label: string; field?: string; modelValue: unknown }>()
const emit = defineEmits<{ 'update:modelValue': [value: unknown] }>()
function setItem(index: number, value: unknown) { const items = [...props.modelValue as unknown[]]; items[index] = value; emit('update:modelValue', items) }
function removeItem(index: number) { emit('update:modelValue', (props.modelValue as unknown[]).filter((_, position) => position !== index)) }
</script>

<style scoped>
label { display: grid; gap: 6px; margin: 10px 0; font-size: 12px; color: var(--color-mute); }
input, textarea, select { box-sizing: border-box; width: 100%; min-width: 0; min-height: 44px; border: 1px solid var(--color-hairline); border-radius: 8px; padding: 9px; color: var(--color-ink); background: var(--color-canvas-raised); font: inherit; }
input:focus-visible, textarea:focus-visible, select:focus-visible { outline: 2px solid var(--color-success); outline-offset: 2px; }
textarea { resize: vertical; }
fieldset { min-width: 0; margin: 12px 0; padding: 8px; border: 1px solid var(--color-hairline); border-radius: 8px; }
legend { font-size: 12px; color: var(--color-mute); }
button { min-height: 44px; border: 1px solid var(--color-hairline); border-radius: 8px; background: var(--color-canvas-soft); color: var(--color-ink); padding: 8px 12px; cursor: pointer; }
</style>
