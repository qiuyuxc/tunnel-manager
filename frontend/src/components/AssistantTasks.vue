<template>
  <section class="tasks" :aria-label="tr('m_a6cd1ccbb69c')">
    <div class="task-heading"><h2>{{ tr('m_a4ea7d95390b') }} <span>{{ tasks.length }}</span></h2><button type="button" class="quiet" @click="emit('refresh')">{{ tr('m_aee887434131') }}</button></div>
    <p class="explanation">{{ tr('m_b1a4629063e7') }}</p>
    <p v-if="!tasks.length" class="empty">{{ tr('m_7ae3131fbef4') }}</p>
    <article v-for="task in tasks" :key="task.id" class="task" :class="task.status">
      <div class="task-top">
        <label class="task-title"><input v-if="task.status === 'pending'" v-model="selected" type="checkbox" :value="task.id" :disabled="busy.has(task.id) || editing === task.id" /><span>{{ task.title }}</span></label>
        <span class="status">{{ busy.has(task.id) ? tr('m_694b71bc8013') : statusLabels[task.status] }}</span>
      </div>
      <div class="change"><span>{{ task.before ? tr('m_39f615acdc2b') : tr('m_df6bad194002') }}</span></div>
      <dl v-if="task.before" class="before"><template v-for="(value, key) in task.before" :key="key"><dt>{{ argumentLabels[key] || key }}</dt><dd>{{ displayValue(value) }}</dd></template></dl>
      <p v-if="task.tool.startsWith('delete_')" class="result">{{ tr('m_e32f672b8d4e') }}</p>
      <div v-else-if="task.before" class="change"><strong>{{ tr('m_9e1bec64e83a') }}</strong></div>
      <div v-if="task.tool.startsWith('delete_')" class="change"><strong>{{ tr('m_45a0b7135449') }}</strong></div>
      <dl><template v-for="(value, key) in task.arguments" :key="key"><dt>{{ argumentLabels[key] || key }}</dt><dd>{{ displayValue(value) }}</dd></template></dl>
      <p v-if="task.tool === 'bind_domain'" class="result">{{ tr('m_5c8a09719294') }} {{ task.arguments.tunnel_id }}{{ tr('m_82bf4e360710') }}</p>
      <p v-if="task.result" class="result" role="status">{{ task.result }}</p>
      <AssistantTaskEditor v-if="editing === task.id" :task="task" :busy="busy.has(task.id)" :save-task="saveTask" @saved="editing = ''" @cancel="editing = ''" />
      <div v-if="!['succeeded', 'cancelled', 'running'].includes(task.status)" class="task-actions">
        <button v-if="task.status === 'pending' && !task.tool.startsWith('delete_') && !editing" :disabled="busy.has(task.id)" @click="beginEdit(task.id)">{{ tr('m_87615fbf148a') }}</button>
        <button v-if="task.status === 'pending'" :disabled="busy.has(task.id) || !!editing" @click="emit('action', task.id, 'pause')">{{ tr('m_8d12fc0d4eb2') }}</button>
        <button v-if="task.status === 'paused'" :disabled="busy.has(task.id)" @click="emit('action', task.id, 'resume')">{{ tr('m_27f565a90066') }}</button>
        <button v-if="task.status === 'unknown'" :disabled="busy.has(task.id)" @click="emit('retry', task.id)">{{ tr('m_276ce6b7b87a') }}</button>
        <button :disabled="busy.has(task.id)" @click="emit('action', task.id, 'cancel')">{{ tr('m_537d17f1c531') }}</button>
      </div>
    </article>
    <div v-if="tasks.some(task => task.status === 'pending')" class="confirm-bar">
      <p>{{ tr('m_b6e825657ab8') }}</p>
      <button class="confirm" :disabled="!selected.length || busy.size > 0 || !!editing" @click="emit('execute', [...selected]); selected = []">{{ tr('m_b3d3b0b6fc80') }} {{ selected.length }} {{ tr('m_49ccde43a154') }}</button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { t as tr, localized } from '../i18n'

import { ref, watch } from 'vue'
import type { AITask } from '../api/assistant'
import AssistantTaskEditor from './AssistantTaskEditor.vue'
import { assistantArgumentLabels as argumentLabels } from '../utils/assistantTaskEdits'
const props = defineProps<{ tasks: AITask[]; busy: Set<string>; saveTask: (id: string, args: Record<string, unknown>, revision: number) => Promise<boolean | undefined> }>()
const emit = defineEmits<{ action: [id: string, action: string]; execute: [ids: string[]]; retry: [id: string]; refresh: [] }>()
const selected = ref<string[]>([])
const editing = ref('')
function beginEdit(id: string) { editing.value = id; selected.value = selected.value.filter(item => item !== id) }
watch(() => props.tasks, tasks => { selected.value = selected.value.filter(id => tasks.some(task => task.id === id && task.status === 'pending')) }, { deep: true })
const statusLabels: Record<AITask['status'], string> = localized(() => ({ pending: tr('m_a8b0bd9c65f7'), paused: tr('m_eb0c326b60ae'), running: tr('m_5026a63b58cd'), succeeded: tr('m_f28461bb49c8'), unknown: tr('m_db6343cfca18'), cancelled: tr('m_a37778f17c5f') }))
function displayValue(value: unknown): string {
  if (typeof value === 'boolean') return value ? tr('m_8da97ddda990') : tr('m_3fd47edce45b')
  if (value !== null && typeof value === 'object') return JSON.stringify(value, null, 2)
  return String(value ?? '—')
}
</script>

<style scoped>
.tasks { color: var(--color-ink); }
.task-heading { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
h2 { font-size: 16px; margin: 0; } h2 span { font-size: 12px; color: var(--color-mute); margin-left: 8px; font-variant-numeric: tabular-nums; }
.explanation, .empty, .result { color: var(--color-mute); font-size: 12px; line-height: 1.7; overflow-wrap: anywhere; }
.empty { margin: 40px 0; }
.task { margin-top: 14px; padding: 16px; border: 1px solid var(--color-hairline); border-radius: 16px; background: var(--color-canvas); }
.task-top { display: flex; gap: 10px; align-items: flex-start; justify-content: space-between; }
.task-title { display: flex; gap: 8px; min-width: 0; font-weight: 650; font-size: 13px; overflow-wrap: anywhere; }
.task-title input { flex-shrink: 0; width: 18px; height: 18px; accent-color: var(--color-success); }
.status { flex: none; font-size: 11px; padding: 3px 7px; border-radius: 20px; background: var(--color-canvas-soft-2); }
.succeeded .status { color: var(--color-success); } .unknown .status { color: var(--color-error); }
.change { display: grid; gap: 5px; font-size: 11px; margin: 14px 0; color: var(--color-mute); }
.change strong { font-weight: 500; color: var(--color-success); }
dl { display: grid; grid-template-columns: minmax(70px, .8fr) minmax(0, 1.5fr); gap: 8px; font-size: 12px; }
dt { color: var(--color-mute); } dd { margin: 0; overflow-wrap: anywhere; white-space: pre-wrap; font-variant-numeric: tabular-nums; }
.task-actions { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 14px; }
button { font: inherit; cursor: pointer; border: 1px solid var(--color-hairline); background: var(--color-canvas-soft); color: var(--color-ink); border-radius: 10px; padding: 8px 12px; min-height: 44px; }
button:disabled { opacity: .5; cursor: default; } button:focus-visible { outline: 2px solid var(--color-success); outline-offset: 3px; }
.quiet { border: none; color: var(--color-mute); background: none; font-size: 12px; }
.confirm-bar { position: sticky; bottom: 0; background: var(--color-canvas-raised); padding: 12px 0 0; }
.confirm-bar p { font-size: 11px; color: var(--color-mute); }
.confirm { width: 100%; background: var(--color-btn-primary-bg); color: var(--color-btn-primary-text); border: none; font-weight: 650; }
</style>
