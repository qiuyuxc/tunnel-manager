<template>
  <section class="tasks" aria-label="任务与变更确认">
    <div class="task-heading"><h2>任务清单 <span>{{ tasks.length }}</span></h2><button type="button" class="quiet" @click="emit('refresh')">刷新</button></div>
    <p class="explanation">参数可以修改，保存后按最终内容勾选确认。修改和删除前请核对原值及对象。执行中的请求无法撤回，暂停仅影响待执行任务。</p>
    <p v-if="!tasks.length" class="empty">这里将显示助手准备的变更。对话本身不会修改任何配置。</p>
    <article v-for="task in tasks" :key="task.id" class="task" :class="task.status">
      <div class="task-top">
        <label class="task-title"><input v-if="task.status === 'pending'" v-model="selected" type="checkbox" :value="task.id" :disabled="busy.has(task.id) || editing === task.id" /><span>{{ task.title }}</span></label>
        <span class="status">{{ busy.has(task.id) ? '处理中' : statusLabels[task.status] }}</span>
      </div>
      <div class="change"><span>{{ task.before ? '变更前：' : '操作：新增配置' }}</span></div>
      <dl v-if="task.before" class="before"><template v-for="(value, key) in task.before" :key="key"><dt>{{ argumentLabels[key] || key }}</dt><dd>{{ displayValue(value) }}</dd></template></dl>
      <p v-if="task.tool.startsWith('delete_')" class="result">变更后：删除上述对象。可能中断域名访问或停止探测；删除监控项目也会清理目标与历史记录。隧道关联的 DNS 不会自动删除。</p>
      <div v-else-if="task.before" class="change"><strong>变更后：应用以下配置（未列出的字段保持不变）</strong></div>
      <div v-if="task.tool.startsWith('delete_')" class="change"><strong>删除对象 ID</strong></div>
      <dl><template v-for="(value, key) in task.arguments" :key="key"><dt>{{ argumentLabels[key] || key }}</dt><dd>{{ displayValue(value) }}</dd></template></dl>
      <p v-if="task.tool === 'bind_domain'" class="result">将服务公开到互联网，新增代理 CNAME → {{ task.arguments.tunnel_id }}.cfargotunnel.com，TTL 自动，并添加源站路由。DNS 与路由不是原子事务，失败时请核对两处。</p>
      <p v-if="task.result" class="result" role="status">{{ task.result }}</p>
      <AssistantTaskEditor v-if="editing === task.id" :task="task" :busy="busy.has(task.id)" :save-task="saveTask" @saved="editing = ''" @cancel="editing = ''" />
      <div v-if="!['succeeded', 'cancelled', 'running'].includes(task.status)" class="task-actions">
        <button v-if="task.status === 'pending' && !task.tool.startsWith('delete_') && !editing" :disabled="busy.has(task.id)" @click="beginEdit(task.id)">修改参数</button>
        <button v-if="task.status === 'pending'" :disabled="busy.has(task.id) || !!editing" @click="emit('action', task.id, 'pause')">暂停</button>
        <button v-if="task.status === 'paused'" :disabled="busy.has(task.id)" @click="emit('action', task.id, 'resume')">恢复待确认</button>
        <button v-if="task.status === 'unknown'" :disabled="busy.has(task.id)" @click="emit('retry', task.id)">核对后重试</button>
        <button :disabled="busy.has(task.id)" @click="emit('action', task.id, 'cancel')">取消任务</button>
      </div>
    </article>
    <div v-if="tasks.some(task => task.status === 'pending')" class="confirm-bar">
      <p>修改或删除可能中断服务；请核对每项原值和变更内容。</p>
      <button class="confirm" :disabled="!selected.length || busy.size > 0 || !!editing" @click="emit('execute', [...selected]); selected = []">核对并执行 {{ selected.length }} 项</button>
    </div>
  </section>
</template>

<script setup lang="ts">
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
const statusLabels: Record<AITask['status'], string> = { pending: '待确认', paused: '已暂停', running: '执行中', succeeded: '已完成', unknown: '需核对', cancelled: '已取消' }
function displayValue(value: unknown): string {
  if (typeof value === 'boolean') return value ? '开启' : '关闭'
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
