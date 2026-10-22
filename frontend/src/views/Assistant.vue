<template>
  <div class="page-container assistant-page" :class="{ 'is-composing': composing }" :style="{ '--assistant-viewport-height': `${viewportHeight}px` }">
    <header class="assistant-header">
      <div class="identity"><span class="assistant-mark" v-html="icons.assistant" /><div><h1>{{ tr('m_1b0049fcfd41') }}</h1><p>{{ store.settings?.shared_enabled ? tr('m_cb87e5beac35') : tr('m_07240a74e246') }} {{ tr('m_072c5cf8dc67') }}</p></div></div>
      <div class="header-actions"><button type="button" @click="historyOpen = true">{{ tr('m_a63280253f17') }}</button><button type="button" class="icon-button" :aria-label="tr('m_036413404922')" @click="settingsOpen = true" v-html="icons.settings" /></div>
    </header>
    <div v-if="store.error" class="error-banner" role="alert">{{ store.error }}<button type="button" :aria-label="tr('m_ee618f2d7968')" @click="store.error = ''">×</button></div>
    <div v-if="store.loading" role="status" class="loading">{{ tr('m_8ad81dd8a671') }}</div>
    <div class="workspace">
      <section class="conversation" :aria-label="tr('m_d36acace9f24')">
        <div class="conversation-top"><span>{{ store.current?.title || tr('m_a114a8c281a2') }}</span><button class="mobile-tasks" :class="{ 'has-pending': pendingTasks > 0 }" :disabled="!store.current?.tasks.length" @click="revealTasks">{{ pendingTasks ? tr('m_fc2be1f048ed', { p0: pendingTasks }) : tr('m_20ab56cbff75', { p0: store.current?.tasks.length || 0 }) }}</button></div>
        <div ref="messagesElement" class="messages" role="log" :aria-label="tr('m_b53654e6b37c')" aria-live="polite">
          <div v-if="!store.current?.messages.length && !currentTurns.length" class="welcome">
            <span class="eyebrow">TUNNEL MANAGER / ASSISTANT</span><h2><span class="desktop-copy">{{ tr('m_2a0b54d5c2bf') }}<br />{{ tr('m_20ed1d0a4bd3') }}</span><span class="mobile-copy">{{ tr('m_ceb885e52275') }}</span></h2>
            <p><span class="desktop-copy">{{ tr('m_f32f485ea316') }}</span><span class="mobile-copy">{{ tr('m_1d096551d401') }}</span></p>
            <button v-for="prompt in prompts" :key="prompt" type="button" class="suggestion" @click="draft = prompt">{{ prompt }}<span aria-hidden="true">↗</span></button>
          </div>
          <article v-for="(message, index) in store.current?.messages || []" :key="index" class="message" :class="message.role"><span class="speaker">{{ message.role === 'user' ? tr('m_a0c7716669b5') : tr('m_1b0049fcfd41') }}</span><AssistantThinking v-if="message.role === 'assistant'" :reasoning="message.reasoning" /><div>{{ message.content }}</div></article>
          <template v-for="turn in currentTurns" :key="turn.id">
            <article class="message user"><span class="speaker">{{ tr('m_a0c7716669b5') }}</span><div>{{ turn.message }}</div></article>
            <article class="message assistant live-reply"><span class="speaker">{{ tr('m_1b0049fcfd41') }}</span><AssistantThinking :active="turn.active" :reasoning="turn.reasoning" :status="turn.status" :replying="!!turn.content" /><div>{{ turn.content }}</div><p v-if="turn.error" class="error" role="alert">{{ turn.error }}<button :disabled="store.sending" @click="store.send(turn.includeResources, turn.id)">{{ tr('m_723588bf300c') }}</button></p></article>
          </template>
          <div v-if="stacked && store.current?.tasks.length" ref="inlineTasksElement" class="inline-tasks">
            <AssistantTasks :key="store.currentID" :tasks="store.current.tasks" :busy="store.busyTasks" :save-task="saveTask" @action="taskAction" @execute="confirmExecute" @retry="retry" @refresh="store.load(true)" />
          </div>
          <p v-if="store.sending" class="thinking" role="status">{{ tr('m_ca264d6e420d') }}</p>
        </div>
        <div class="composer" @focusout="onComposerBlur">
          <div v-if="store.settings && !store.settings.configured" class="setup-note">{{ store.settings.shared_enabled ? tr('m_1a5fef6d88bd') : tr('m_304f23f8d7c9') }}<button @click="settingsOpen = true">{{ tr('m_e38ab48681dd') }}</button></div>
          <div class="composer-options">
            <label class="resource-check"><input v-model="includeResources" type="checkbox" /><span class="desktop-copy">{{ tr('m_750d0884694f') }}</span><span class="mobile-copy">{{ tr('m_03e9099ca5f2') }}</span></label>
            <n-popover trigger="click" placement="top-end" :width="260">
              <template #trigger><button class="privacy-toggle" type="button">{{ tr('m_9d06d61ee41f') }}</button></template>
              {{ tr('m_6a78a7d91819') }}
            </n-popover>
          </div>
          <form @submit.prevent="store.send(includeResources)"><textarea ref="composerElement" v-model="draft" :aria-label="tr('m_edcac8607f16')" :placeholder="compact ? tr('m_c3a37f287970') : tr('m_0dbfd9f6646b')" :rows="compact ? 1 : 3" maxlength="8000" @focus="composing = true" @keydown="onComposerKey" /><button class="send" :disabled="!draft.trim() || store.sending || !store.settings?.configured" type="submit">{{ store.sending ? tr('m_43ce458cae49') : tr('m_edecf0ae6e51') }}</button></form>
          <p class="privacy">{{ tr('m_e0c22e375b03') }}<span class="desktop-copy">{{ tr('m_81e12d17abce') }}</span></p>
        </div>
      </section>
      <aside v-if="!stacked" class="desktop-tasks"><AssistantTasks :key="store.currentID" :tasks="store.current?.tasks || []" :busy="store.busyTasks" :save-task="saveTask" @action="taskAction" @execute="confirmExecute" @retry="retry" @refresh="store.load(true)" /></aside>
    </div>
    <n-drawer v-model:show="historyOpen" :width="320" placement="left"><n-drawer-content :title="tr('m_ed008a400d30')" closable><button class="history-new" :disabled="store.sending" @click="createConversation">{{ tr('m_58e21b8737cc') }}</button><div v-for="conversation in store.conversations" :key="conversation.id" class="history-row"><button @click="store.currentID = conversation.id; historyOpen = false">{{ conversation.title }}</button><button :disabled="store.sending" :aria-label="tr('m_7b85126d4a6b', { p0: conversation.title })" @click="removeConversation(conversation.id)">×</button></div></n-drawer-content></n-drawer>
    <AssistantSettings :open="settingsOpen" :settings="store.settings" @close="settingsOpen = false" @saved="store.settings = $event" />
  </div>
</template>

<script setup lang="ts">
import { t as tr, localized } from '../i18n'

import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { NDrawer, NDrawerContent, NPopover, useDialog } from 'naive-ui'
import { icons } from '../navigation'
import { useAssistantStore } from '../stores/assistant'
import AssistantSettings from '../components/AssistantSettings.vue'
import AssistantTasks from '../components/AssistantTasks.vue'
import AssistantThinking from '../components/AssistantThinking.vue'

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
const currentTurns = computed(() => store.liveTurns.filter(turn => turn.conversationID === (store.currentID || 'new')))
const pendingTasks = computed(() => store.current?.tasks.filter(task => task.status === 'pending').length || 0)
const viewportHeight = ref(window.visualViewport?.height ?? window.innerHeight)
const draft = computed({ get: () => store.drafts[store.currentID || 'new'] || '', set: value => { store.drafts[store.currentID || 'new'] = value } })
const prompts = localized(() => ([tr('m_b1eeed66e829'), tr('m_bf45ef04554a'), tr('m_71f37711ac86')]))
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
  if (compact.value && textarea.value) textarea.style.height = `${Math.min(textarea.scrollHeight, 104)}px`
}
watch(draft, () => nextTick(resizeComposer))
watch(() => currentTurns.value.map(turn => turn.content.length + turn.reasoning.length).join(','), async () => {
  const pane = messagesElement.value
  const nearBottom = pane && pane.scrollHeight - pane.scrollTop - pane.clientHeight < 140
  await nextTick()
  if (pane && nearBottom) pane.scrollTo({ top: pane.scrollHeight, behavior: 'auto' })
})
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
function removeConversation(id: string) { dialog.warning({ title: tr('m_aea0944d46f2'), content: tr('m_b146d13ba283'), positiveText: tr('m_d51250d14a51'), negativeText: tr('m_670ec25af841'), onPositiveClick: () => store.remove(id) }) }
function taskAction(id: string, action: string) { if (store.currentID) void store.taskAction(store.currentID, id, action) }
async function saveTask(id: string, args: Record<string, unknown>, revision: number) {
  const conversationID = store.currentID
  if (!conversationID) return
  return store.taskAction(conversationID, id, 'edit', false, false, { arguments: args, expected_revision: revision })
}
function confirmExecute(ids: string[]) {
  const conversationID = store.currentID
  if (!conversationID) return
  const approved = ids.map(id => ({ id, revision: store.current?.tasks.find(task => task.id === id)?.revision || 0 }))
  dialog.warning({ title: tr('m_7524f24a3a86', { p0: ids.length }), content: tr('m_ddc6ab71df20'), positiveText: tr('m_68a35e224268'), negativeText: tr('m_24c716c56cdc'), onPositiveClick: () => {
    const queue = [...approved]
    async function worker() { while (queue.length) { const { id, revision } = queue.shift()!; const task = store.conversations.find(item => item.id === conversationID)?.tasks.find(item => item.id === id); if (task?.status === 'pending') await store.taskAction(conversationID, id, 'execute', true, false, { expected_revision: revision }) } }
    void Promise.all([worker(), worker()])
  } })
}
function retry(id: string) {
  const conversationID = store.currentID
  const revision = store.current?.tasks.find(task => task.id === id)?.revision || 0
  dialog.warning({ title: tr('m_61073c947ab2'), content: tr('m_6fe3c5ced59d'), positiveText: tr('m_d71a011288f0'), negativeText: tr('m_a3e39298c440'), onPositiveClick: () => store.taskAction(conversationID, id, 'retry', true, true, { expected_revision: revision }) })
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
