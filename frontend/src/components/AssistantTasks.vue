<template>
  <section class="tasks" aria-label="任务与变更确认">
    <div class="task-heading"><h2>任务清单 <span>{{ tasks.length }}</span></h2><button type="button" class="quiet" @click="emit('refresh')">刷新</button></div>
    <p class="explanation">只执行你勾选并确认的新增操作，不覆盖或删除已有记录与路由。执行中的请求无法撤回，暂停仅影响待执行任务。</p>
    <p v-if="!tasks.length" class="empty">这里将显示助手准备的变更。对话本身不会修改任何配置。</p>
    <article v-for="task in tasks" :key="task.id" class="task" :class="task.status">
      <div class="task-top">
        <label class="task-title"><input v-if="task.status === 'pending'" v-model="selected" type="checkbox" :value="task.id" :disabled="busy.has(task.id)" /><span>{{ task.title }}</span></label>
        <span class="status">{{ busy.has(task.id) ? '处理中' : statusLabels[task.status] }}</span>
      </div>
      <div class="change"><span>变更前：不由助手修改现有资源</span><strong>变更后：新增以下配置</strong></div>
      <dl><template v-for="(value, key) in task.arguments" :key="key"><dt>{{ argumentLabels[key] || key }}</dt><dd>{{ typeof value === 'boolean' ? (value ? '开启' : '关闭') : value }}</dd></template></dl>
      <p v-if="task.tool === 'bind_domain'" class="result">将服务公开到互联网，新增代理 CNAME → {{ task.arguments.tunnel_id }}.cfargotunnel.com，TTL 自动，并添加源站路由。DNS 与路由不是原子事务，失败时请核对两处。</p>
      <p v-if="task.result" class="result" role="status">{{ task.result }}</p>
      <div v-if="!['succeeded', 'cancelled', 'running'].includes(task.status)" class="task-actions">
        <button v-if="task.status === 'pending'" :disabled="busy.has(task.id)" @click="emit('action', task.id, 'pause')">暂停</button>
        <button v-if="task.status === 'paused'" :disabled="busy.has(task.id)" @click="emit('action', task.id, 'resume')">恢复待确认</button>
        <button v-if="task.status === 'unknown'" :disabled="busy.has(task.id)" @click="emit('retry', task.id)">核对后重试</button>
        <button :disabled="busy.has(task.id)" @click="emit('action', task.id, 'cancel')">取消任务</button>
      </div>
    </article>
    <div v-if="tasks.some(task => task.status === 'pending')" class="confirm-bar">
      <p>域名绑定将公开服务；监控目标会开始周期探测。</p>
      <button class="confirm" :disabled="!selected.length || busy.size > 0" @click="emit('execute', [...selected]); selected = []">核对并执行 {{ selected.length }} 项</button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { AITask } from '../api/assistant'
const props = defineProps<{ tasks: AITask[]; busy: Set<string> }>()
const emit = defineEmits<{ action: [id: string, action: string]; execute: [ids: string[]]; retry: [id: string]; refresh: [] }>()
const selected = ref<string[]>([])
watch(() => props.tasks, tasks => { selected.value = selected.value.filter(id => tasks.some(task => task.id === id && task.status === 'pending')) }, { deep: true })
const statusLabels: Record<AITask['status'], string> = { pending: '待确认', paused: '已暂停', running: '执行中', succeeded: '已完成', unknown: '需核对', cancelled: '已取消' }
const argumentLabels: Record<string, string> = { name: '名称', zone_id: '区域 ID', tunnel_id: '隧道 ID', hostname: '公开域名', service_url: '源站地址', type: '类型', content: '记录值', data: '记录数据（SRV / CAA）', ttl: 'TTL（1 为自动）', proxied: '代理', priority: 'MX 优先级', interval_sec: '间隔（秒）', monitor_id: '监控项目 ID', url: '探测地址' }
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
