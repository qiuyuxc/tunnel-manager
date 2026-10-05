<template>
  <div class="page-container assistant-page" :class="{ 'is-composing': composing }" :style="{ '--assistant-viewport-height': `${viewportHeight}px` }">
    <header class="assistant-header">
      <div class="identity"><span class="assistant-mark" v-html="icons.assistant" /><div><h1>AI 助手</h1><p>管理员连接 · 确认后执行</p></div></div>
      <div class="header-actions"><button type="button" @click="historyOpen = true">会话</button><button type="button" class="icon-button" aria-label="AI 连接配置" @click="settingsOpen = true" v-html="icons.settings" /></div>
    </header>
    <div v-if="store.error" class="error-banner" role="alert">{{ store.error }}<button type="button" aria-label="关闭错误提示" @click="store.error = ''">×</button></div>
    <div v-if="store.loading" role="status" class="loading">正在读取会话与连接状态…</div>
    <div class="workspace">
      <section class="conversation" aria-label="助手对话">
        <div class="conversation-top"><span>{{ store.current?.title || '新的配置计划' }}</span><button class="mobile-tasks" :class="{ 'has-pending': pendingTasks > 0 }" :disabled="!store.current?.tasks.length" @click="revealTasks">{{ pendingTasks ? `待确认 ${pendingTasks}` : `任务 ${store.current?.tasks.length || 0}` }}</button></div>
        <div ref="messagesElement" class="messages" role="log" aria-label="对话记录" aria-live="polite">
          <div v-if="!store.current?.messages.length" class="welcome">
            <span class="eyebrow">TUNNEL MANAGER / ASSISTANT</span><h2><span class="desktop-copy">描述目标，<br />一起核对每一步。</span><span class="mobile-copy">今天想配置什么？</span></h2>
            <p><span class="desktop-copy">查询资源、准备隧道与 DNS 配置，或安排多个监控任务。所有新增操作都需要你确认。</span><span class="mobile-copy">从隧道、DNS 或监控开始，执行前由你确认。</span></p>
            <button v-for="prompt in prompts" :key="prompt" type="button" class="suggestion" @click="draft = prompt">{{ prompt }}<span aria-hidden="true">↗</span></button>
          </div>
          <article v-for="(message, index) in store.current?.messages || []" :key="index" class="message" :class="message.role"><span class="speaker">{{ message.role === 'user' ? '你' : 'AI 助手' }}</span><div>{{ message.content }}</div></article>
          <div v-if="stacked && store.current?.tasks.length" ref="inlineTasksElement" class="inline-tasks">
            <AssistantTasks :key="store.currentID" :tasks="store.current.tasks" :busy="store.busyTasks" @action="taskAction" @execute="confirmExecute" @retry="retry" @refresh="store.load(true)" />
          </div>
          <p v-if="store.sending" class="thinking" role="status">正在准备回复与配置计划，尚未执行任何操作…</p>
        </div>
        <div class="composer" @focusout="onComposerBlur">
          <div v-if="store.settings && !store.settings.configured" class="setup-note">先配置 API 端点、模型和密钥。<button @click="settingsOpen = true">连接配置</button></div>
          <div class="composer-options">
            <label class="resource-check"><input v-model="includeResources" type="checkbox" /><span class="desktop-copy">允许发送我的资源名称与 ID 给配置的 AI 服务</span><span class="mobile-copy">允许发送资源名称与 ID</span></label>
            <n-popover trigger="click" placement="top-end" :width="260">
              <template #trigger><button class="privacy-toggle" type="button">隐私说明</button></template>
              会话仅对你可见。消息会发送给所配置的 AI 服务，请勿填写密码或连接令牌。勾选后还会附带你有权限访问的资源名称与 ID，不包含 Cloudflare 凭据。
            </n-popover>
          </div>
          <form @submit.prevent="store.send(includeResources)"><textarea ref="composerElement" v-model="draft" aria-label="发送给 AI 助手的消息" :placeholder="compact ? '描述你想完成的配置…' : '例如：创建一个私有监控项目，每 60 秒检查一次…'" :rows="compact ? 1 : 3" maxlength="8000" @focus="composing = true" @keydown="onComposerKey" /><button class="send" :disabled="!draft.trim() || store.sending || !store.settings?.configured" type="submit">{{ store.sending ? '思考中' : '发送' }}</button></form>
          <p class="privacy">会话仅对你可见。输入内容将发送给所配置的 AI 服务，请勿填写密码或连接令牌。<span class="desktop-copy">Enter 发送，Shift + Enter 换行。</span></p>
        </div>
      </section>
      <aside v-if="!stacked" class="desktop-tasks"><AssistantTasks :key="store.currentID" :tasks="store.current?.tasks || []" :busy="store.busyTasks" @action="taskAction" @execute="confirmExecute" @retry="retry" @refresh="store.load(true)" /></aside>
    </div>
    <n-drawer v-model:show="historyOpen" :width="320" placement="left"><n-drawer-content title="我的会话" closable><button class="history-new" :disabled="store.sending" @click="createConversation">新建会话</button><div v-for="conversation in store.conversations" :key="conversation.id" class="history-row"><button @click="store.currentID = conversation.id; historyOpen = false">{{ conversation.title }}</button><button :disabled="store.sending" :aria-label="`删除会话：${conversation.title}`" @click="removeConversation(conversation.id)">×</button></div></n-drawer-content></n-drawer>
    <AssistantSettings :open="settingsOpen" :settings="store.settings" @close="settingsOpen = false" @saved="store.settings = $event" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { NDrawer, NDrawerContent, NPopover, useDialog } from 'naive-ui'
import { icons } from '../navigation'
import { useAssistantStore } from '../stores/assistant'
import AssistantSettings from '../components/AssistantSettings.vue'
import AssistantTasks from '../components/AssistantTasks.vue'

const store = useAssistantStore()
const dialog = useDialog()
const settingsOpen = ref(false)
const historyOpen = ref(false)
const includeResources = ref(false)
const messagesElement = ref<HTMLElement | null>(null)
const inlineTasksElement = ref<HTMLElement | null>(null)
const composerElement = ref<HTMLTextAreaElement | null>(null)
const composing = ref(false)
const compactQuery = window.matchMedia('(max-width: 768px)')
const compact = ref(compactQuery.matches)
const stackedQuery = window.matchMedia('(max-width: 1050px)')
const stacked = ref(stackedQuery.matches)
const pendingTasks = computed(() => store.current?.tasks.filter(task => task.status === 'pending').length || 0)
const viewportHeight = ref(window.visualViewport?.height ?? window.innerHeight)
const draft = computed({ get: () => store.drafts[store.currentID || 'new'] || '', set: value => { store.drafts[store.currentID || 'new'] = value } })
const prompts = ['查看我现有的隧道与监控项目', '把 NAS 和相册接入域名，先问我所需信息', '创建一个名为「站点可用性」的私有监控项目']
let timer: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  void store.load()
  timer = setInterval(() => { if (store.conversations.some(conversation => conversation.tasks.some(task => task.status === 'running'))) void store.load(true) }, 5000)
  window.visualViewport?.addEventListener('resize', updateViewport)
  window.addEventListener('resize', updateViewport)
  compactQuery.addEventListener('change', updateViewport)
  stackedQuery.addEventListener('change', updateViewport)
  updateViewport()
  void document.fonts.ready.then(resizeComposer)
})
onUnmounted(() => {
  clearInterval(timer)
  window.visualViewport?.removeEventListener('resize', updateViewport)
  window.removeEventListener('resize', updateViewport)
  compactQuery.removeEventListener('change', updateViewport)
  stackedQuery.removeEventListener('change', updateViewport)
})
function updateViewport() {
  viewportHeight.value = window.visualViewport?.height ?? window.innerHeight
  compact.value = compactQuery.matches
  stacked.value = stackedQuery.matches
  void nextTick(resizeComposer)
}
function resizeComposer() {
  const textarea = composerElement.value
  if (!textarea) return
  textarea.style.height = ''
  if (compact.value) textarea.style.height = `${Math.min(textarea.scrollHeight, 104)}px`
}
watch(draft, () => nextTick(resizeComposer))
function onComposerBlur(event: FocusEvent) {
  if (!(event.relatedTarget instanceof Node) || !(event.currentTarget as HTMLElement).contains(event.relatedTarget)) composing.value = false
}
function revealTasks() {
  const messages = messagesElement.value
  const target = inlineTasksElement.value?.querySelector('.task.pending') || inlineTasksElement.value
  if (!messages || !target) return
  messages.scrollTo({ top: messages.scrollTop + target.getBoundingClientRect().top - messages.getBoundingClientRect().top - 12, behavior: 'auto' })
}
watch([() => store.currentID, () => store.current?.messages.length, () => store.current?.tasks.length || 0, () => store.sending, stacked], async ([conversationID, , taskCount, sending, narrow], previous) => {
  await nextTick()
  if (narrow && taskCount > 0 && !sending && (conversationID !== previous[0] || taskCount !== previous[2] || sending !== previous[3] || narrow !== previous[4])) revealTasks()
  else messagesElement.value?.scrollTo({ top: messagesElement.value.scrollHeight, behavior: 'auto' })
})
function onComposerKey(event: KeyboardEvent) { if (!compact.value && event.key === 'Enter' && !event.shiftKey && !event.isComposing) { event.preventDefault(); if (store.settings?.configured) void store.send(includeResources.value) } }
async function createConversation() { await store.create(); historyOpen.value = false }
function removeConversation(id: string) { dialog.warning({ title: '删除这个会话？', content: '删除会话和任务记录，不会删除已创建的业务资源。', positiveText: '删除会话', negativeText: '保留', onPositiveClick: () => store.remove(id) }) }
function taskAction(id: string, action: string) { if (store.currentID) void store.taskAction(store.currentID, id, action) }
function confirmExecute(ids: string[]) {
  const conversationID = store.currentID
  if (!conversationID) return
  dialog.warning({ title: `确认执行 ${ids.length} 项新增操作？`, content: '仅执行已勾选任务。域名绑定将把服务公开到互联网，任务可能创建收费资源，监控目标会开始周期探测。执行中的请求不能撤回。', positiveText: '确认执行', negativeText: '继续核对', onPositiveClick: () => {
    const queue = [...ids]
    async function worker() { while (queue.length) { const id = queue.shift()!; const task = store.conversations.find(item => item.id === conversationID)?.tasks.find(item => item.id === id); if (task?.status === 'pending') await store.taskAction(conversationID, id, 'execute', true) } }
    void Promise.all([worker(), worker()])
  } })
}
function retry(id: string) {
  const conversationID = store.currentID
  dialog.warning({ title: '已核对实际资源，确认没有重复创建？', content: '超时或失败不代表没有生效。请先到业务页面核对，只有确认尚未创建时才重试。', positiveText: '已核对，重新执行', negativeText: '先去核对', onPositiveClick: () => store.taskAction(conversationID, id, 'retry', true, true) })
}
</script>

<style scoped>
.assistant-page { color: var(--color-ink); max-width: 1600px; margin: auto; }
.assistant-header, .identity, .header-actions { display: flex; align-items: center; gap: 14px; }
.assistant-header { justify-content: space-between; margin-bottom: 22px; }
.assistant-mark { display: grid; place-items: center; width: 46px; height: 46px; border-radius: 15px; background: var(--color-canvas-soft-2); color: var(--color-success); }
.assistant-mark :deep(svg) { width: 24px; height: 24px; }
h1 { margin: 0; font-size: 24px; letter-spacing: -.6px; } .identity p { margin: 5px 0 0; color: var(--color-mute); font-size: 12px; }
button { min-height: 44px; padding: 9px 14px; border-radius: 12px; border: 1px solid var(--color-hairline); background: var(--color-canvas-raised); color: var(--color-ink); font: inherit; cursor: pointer; }
button:disabled { opacity: .5; cursor: default; } :is(button, textarea):focus-visible { outline: 2px solid var(--color-success); outline-offset: 3px; }
.icon-button { width: 44px; display: grid; place-items: center; padding: 0; }
.workspace { display: grid; grid-template-columns: minmax(0, 1.65fr) minmax(300px, 1fr); gap: 20px; height: min(800px, calc(100dvh - 190px)); min-height: 560px; }
.conversation { min-width: 0; display: flex; flex-direction: column; border: 1px solid var(--color-hairline); border-radius: 22px; background: var(--color-canvas-raised); overflow: hidden; }
.conversation-top { display: flex; align-items: center; justify-content: space-between; min-height: 52px; padding: 0 22px; border-bottom: 1px solid var(--color-hairline); gap: 12px; font-size: 12px; color: var(--color-mute); }
.conversation-top span { overflow-wrap: anywhere; }
.messages { overflow-y: auto; flex: 1; min-height: 0; padding: 24px; overscroll-behavior: contain; }
.welcome { max-width: 500px; padding: 20px 0; }
.eyebrow { font-size: 10px; letter-spacing: 1.2px; color: var(--color-success); }
.welcome h2 { font-size: clamp(24px, 3vw, 36px); line-height: 1.4; font-weight: 650; letter-spacing: -1px; margin: 20px 0 14px; }
.welcome p { color: var(--color-mute); line-height: 1.8; font-size: 13px; margin-bottom: 26px; }
.suggestion { display: flex; width: 100%; gap: 16px; justify-content: space-between; align-items: center; text-align: left; font-size: 12px; margin-top: 10px; background: var(--color-canvas-soft); }
.suggestion span { color: var(--color-success); flex: none; }
.message { margin: 0 0 26px; overflow-wrap: anywhere; }.speaker { display: block; color: var(--color-success); font-size: 11px; margin-bottom: 8px; }
.message div { white-space: pre-wrap; font-size: 14px; line-height: 1.9; }
.message.user { background: var(--color-canvas-soft-2); border-radius: 16px; padding: 15px 18px; margin-left: 24px; }.message.user .speaker { color: var(--color-mute); }
.thinking, .loading { color: var(--color-mute); font-size: 12px; padding: 8px 0; }
.composer { flex: none; padding: 16px 20px 12px; border-top: 1px solid var(--color-hairline); }
.composer form { display: flex; align-items: flex-end; gap: 10px; padding: 10px; background: var(--color-canvas-soft); border: 1px solid var(--color-hairline-strong); border-radius: 16px; }
textarea { flex: 1; min-width: 0; resize: none; border: none; padding: 3px; background: none; color: var(--color-ink); font: inherit; font-size: 14px; line-height: 1.6; }
.send { flex: none; padding: 8px 14px; background: var(--color-btn-primary-bg); color: var(--color-btn-primary-text); border: none; font-size: 12px; font-weight: 650; }
.privacy { font-size: 10px; line-height: 1.7; color: var(--color-mute); margin: 8px 0 0; }
.mobile-copy, .privacy-toggle { display: none; }
.resource-check { display: flex; gap: 6px; align-items: flex-start; margin-bottom: 10px; font-size: 11px; color: var(--color-mute); }.resource-check input { accent-color: var(--color-success); }
.setup-note { color: var(--color-mute); font-size: 12px; margin-bottom: 10px; }.setup-note button { margin-left: 8px; border: none; font-size: 12px; color: var(--color-success); padding: 0 4px; }
.desktop-tasks { overflow-y: auto; min-width: 0; border-radius: 22px; border: 1px solid var(--color-hairline); padding: 20px; background: var(--color-canvas-soft); }
.mobile-tasks { display: none; }
.mobile-tasks.has-pending { color: var(--color-success); font-weight: 650; }
.inline-tasks { margin-top: 20px; padding-top: 16px; border-top: 1px solid var(--color-hairline); }
.error-banner { display: flex; justify-content: space-between; gap: 12px; padding: 12px 16px; margin-bottom: 16px; border: 1px solid var(--color-error); border-radius: 14px; color: var(--color-error); font-size: 13px; overflow-wrap: anywhere; }.error-banner button { background: none; border: none; color: inherit; padding: 0 8px; }
.history-new { width: 100%; margin-bottom: 16px; }.history-row { display: flex; gap: 6px; margin: 8px 0; }.history-row button:first-child { text-align: left; flex: 1; min-width: 0; overflow-wrap: anywhere; }
@media (max-width: 1050px) { .workspace { grid-template-columns: minmax(0, 1fr); }.desktop-tasks { display: none; }.mobile-tasks { display: block; font-size: 12px; padding: 5px 10px; min-width: 72px; }.workspace { height: calc(100dvh - 190px); } }
@media (max-width: 768px) {
  .assistant-page { display: flex; flex-direction: column; height: calc(var(--assistant-viewport-height, 100dvh) - 56px - var(--tabbar-height) - env(safe-area-inset-bottom)); min-height: 0; padding: 10px 12px 4px; }
  .assistant-header { flex: none; margin-bottom: 10px; gap: 8px; }
  .identity { gap: 9px; }
  .assistant-mark { width: 32px; height: 32px; border-radius: 10px; flex: none; }
  h1 { font-size: 18px; }
  .identity p { font-size: 10px; margin-top: 2px; }
  .header-actions { gap: 6px; }
  .header-actions button { padding: 6px 10px; font-size: 12px; }
  .workspace { flex: 1; height: auto; min-height: 0; }
  .conversation { min-height: 0; border-radius: 16px; }
  .conversation-top { flex: none; min-height: 44px; padding: 0 12px; }
  .mobile-tasks { border: none; background: none; min-width: 60px; padding: 0 4px; }
  .messages { padding: 14px; }
  .composer { padding: 0 10px 10px; }
  .composer-options { display: flex; align-items: center; justify-content: space-between; gap: 4px; }
  .resource-check { align-items: center; min-height: 44px; margin: 0; font-size: 11px; gap: 6px; }
  .resource-check input { flex: none; width: 16px; height: 16px; margin: 0; }
  .privacy-toggle { display: block; flex: none; border: none; border-radius: 8px; background: none; padding: 0 2px; font-size: 11px; color: var(--color-mute); }
  .composer form { gap: 8px; padding: 5px; border-radius: 13px; align-items: flex-end; }
  textarea { box-sizing: border-box; min-height: 44px; max-height: 104px; padding: 10px 6px; font-size: 16px; line-height: 24px; overflow-y: auto; }
  .send { min-width: 52px; min-height: 44px; padding: 8px 10px; border-radius: 9px; }
  .privacy, .desktop-copy, .eyebrow { display: none; }
  .mobile-copy { display: inline; }
  .welcome { padding: 4px 0; }
  .welcome h2 { font-size: 22px; margin: 4px 0 8px; letter-spacing: -.5px; }
  .welcome p { font-size: 12px; margin: 0 0 16px; line-height: 1.7; }
  .suggestion { padding: 10px 12px; gap: 8px; font-size: 12px; margin-top: 8px; }
  .message.user { margin-left: 12px; }
  .setup-note { margin: 8px 0 0; }
}
@media (prefers-reduced-motion: no-preference) { button { transition: background 120ms ease, transform 120ms ease; } button:active:not(:disabled) { transform: scale(.98); } }
</style>
