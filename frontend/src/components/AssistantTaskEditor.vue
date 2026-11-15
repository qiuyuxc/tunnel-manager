<template>
  <form ref="editorElement" class="task-editor" @submit.prevent="save">
    <p>{{ tr('m_f9871e460d83') }}</p>
    <p v-if="stale" role="alert">{{ tr('m_627ec605a45e') }}</p>
    <fieldset class="editor-fields" :disabled="busy || stale"><template v-for="(value, key) in draft" :key="key">
      <p v-if="task.tool === 'update_dns_record' && ['zone_id', 'record_id'].includes(key)">{{ labels[key] }}：{{ value }}</p>
      <label v-else-if="key === 'type' && task.tool.includes('dns_record')" class="dns-type">{{ tr('m_8f6363220575') }}<select :value="value" @change="draft = changeAssistantDNSType(draft, ($event.target as HTMLSelectElement).value)"><option v-for="type in dnsTypes" :key="type" :value="type">{{ type }}</option></select></label>
      <AssistantArgumentField v-else :field="key" :label="labels[key] || key" :model-value="value" @update:model-value="draft[key] = $event" />
    </template></fieldset>
    <div class="editor-actions"><button type="submit" :disabled="busy || stale">{{ busy ? tr('m_6bdb4435095e') : tr('m_991bb7cfe5a8') }}</button><button type="button" :disabled="busy" @click="emit('cancel')">{{ tr('m_4b29ce8e4093') }}</button></div>
  </form>
</template>

<script setup lang="ts">
import { t as tr } from '../i18n'

import { computed, nextTick, onMounted, ref } from 'vue'
import type { AITask } from '../api/assistant'
import AssistantArgumentField from './AssistantArgumentField.vue'
import { assistantArgumentLabels as labels, changeAssistantDNSType } from '../utils/assistantTaskEdits'
const props = defineProps<{ task: AITask; busy: boolean; saveTask: (id: string, args: Record<string, unknown>, revision: number) => Promise<boolean | undefined> }>()
const emit = defineEmits<{ cancel: []; saved: [] }>()
const revision = props.task.revision || 0
const editorElement = ref<HTMLElement | null>(null)
onMounted(async () => {
  await nextTick()
  const editor = editorElement.value
  const container = editor?.closest<HTMLElement>('.messages, .desktop-tasks')
  if (editor && container) container.scrollTo({ top: container.scrollTop + editor.getBoundingClientRect().top - container.getBoundingClientRect().top - 12, behavior: 'auto' })
})
const draft = ref<Record<string, unknown>>(JSON.parse(JSON.stringify(props.task.arguments)))
const stale = computed(() => (props.task.revision || 0) !== revision || props.task.status !== 'pending')
const dnsTypes = ['A', 'AAAA', 'CNAME', 'TXT', 'MX', 'NS', 'SRV', 'CAA', 'PTR']
async function save() { if (!stale.value && await props.saveTask(props.task.id, JSON.parse(JSON.stringify(draft.value)), revision)) emit('saved') }
</script>

<style scoped>
.task-editor { margin-top: 12px; padding-top: 8px; border-top: 1px solid var(--color-hairline); }
.editor-fields { min-width: 0; margin: 0; padding: 0; border: 0; }
p, .dns-type { font-size: 12px; color: var(--color-mute); line-height: 1.7; overflow-wrap: anywhere; }
.dns-type { display: grid; gap: 6px; }
select { min-height: 44px; border: 1px solid var(--color-hairline); border-radius: 8px; padding: 9px; background: var(--color-canvas-raised); color: var(--color-ink); }
.editor-actions { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 14px; }
button { min-height: 44px; padding: 8px 12px; background: var(--color-canvas-soft); color: var(--color-ink); border: 1px solid var(--color-hairline); border-radius: 8px; cursor: pointer; }
button:disabled { opacity: .5; cursor: default; }
</style>
