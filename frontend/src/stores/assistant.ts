import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { useConfigStore } from './config'
import { sendAIStream } from '../api/assistantStream'
import { actOnAITask, deleteAIConversation, getAIConversations, getAISettings, newAIConversation, type AIConversation, type AISettings, type AITaskOptions } from '../api/assistant'

export function aiError(error: unknown): string {
  const failure = error as { response?: { data?: { error?: string } }; message?: string }
  return failure.response?.data?.error || failure.message || '请求失败，请检查网络后重试；执行请求失败时请先核对实际资源。'
}

export const useAssistantStore = defineStore('assistant', () => {
  const config = useConfigStore()
  const conversations = ref<AIConversation[]>([])
  const currentID = ref('')
  const settings = ref<AISettings | null>(null)
  const drafts = ref<Record<string, string>>({})
  const sending = ref(false)
  const liveTurns = ref<{ id: string; conversationID: string; message: string; content: string; reasoning: string; status: string; error: string; active: boolean; includeResources: boolean }[]>([])
  let streamController: AbortController | undefined
  const loading = ref(false)
  const error = ref('')
  const busyTasks = ref(new Set<string>())
  const current = computed(() => conversations.value.find(conversation => conversation.id === currentID.value))
  let generation = 0
  let turnSequence = 0

  watch(() => config.token, () => {
    generation++
    streamController?.abort()
    liveTurns.value = []
    conversations.value = []
    currentID.value = ''
    drafts.value = {}
    settings.value = null
    error.value = ''
    sending.value = false
    loading.value = false
    busyTasks.value = new Set()
  })

  function merge(conversation: AIConversation) {
    const index = conversations.value.findIndex(item => item.id === conversation.id)
    if (index < 0) conversations.value.unshift(conversation)
    else conversations.value[index] = conversation
  }

  async function load(quiet = false) {
    const started = generation
    if (!quiet) loading.value = true
    try {
      const [configuration, history] = await Promise.all([getAISettings(), getAIConversations()])
      if (started !== generation) return
      settings.value = configuration.data
      conversations.value = history.data.conversations
      if (!current.value) currentID.value = conversations.value[0]?.id || ''
    } catch (failure) {
      if (started === generation) error.value = aiError(failure)
    } finally {
      if (started === generation) loading.value = false
    }
  }

  async function create() {
    const started = generation
    error.value = ''
    try {
      const { data } = await newAIConversation()
      if (started !== generation) return
      merge(data)
      currentID.value = data.id
      return data.id
    } catch (failure) { if (started === generation) error.value = aiError(failure) }
  }

  async function remove(id: string) {
    const started = generation
    try {
      await deleteAIConversation(id)
      if (started !== generation) return
      conversations.value = conversations.value.filter(item => item.id !== id)
      delete drafts.value[id]
      if (currentID.value === id) currentID.value = conversations.value[0]?.id || ''
    } catch (failure) { if (started === generation) error.value = aiError(failure) }
  }

  async function send(includeResources: boolean, retryID?: string) {
    if (sending.value) return
    const started = generation
    const draftKey = currentID.value || 'new'
    let turn = retryID ? liveTurns.value.find(item => item.id === retryID) : undefined
    const message = turn?.message || (drafts.value[draftKey] || '').trim()
    if (!message) return
    if (!turn) {
      liveTurns.value.push({ id: `turn-${Date.now()}-${++turnSequence}`, conversationID: draftKey, message, content: '', reasoning: '', status: '正在发送…', error: '', active: true, includeResources })
      turn = liveTurns.value[liveTurns.value.length - 1]!
      drafts.value[draftKey] = ''
    } else { turn.content = ''; turn.reasoning = ''; turn.error = ''; turn.active = true; includeResources = turn.includeResources }
    sending.value = true
    error.value = ''
    streamController = new AbortController()
    const controller = streamController
    const timer = setTimeout(() => controller.abort(), 180000)
    try {
      const id = turn.conversationID !== 'new' ? turn.conversationID : await create()
      if (!id || started !== generation) throw new Error('会话创建失败，请重试')
      turn.conversationID = id
      if (draftKey === 'new' && drafts.value.new) { drafts.value[id] = drafts.value.new; delete drafts.value.new }
      await sendAIStream(id, message, includeResources, controller.signal, event => {
        if (started !== generation) return
        if (event.type === 'text') { turn!.content += String(event.data.delta || ''); turn!.status = '正在回复…' }
        if (event.type === 'reasoning') turn!.reasoning += String(event.data.delta || '')
        if (event.type === 'status') turn!.status = String(event.data.message || '')
        if (event.type === 'done') {
          merge(event.data as unknown as AIConversation)
          liveTurns.value = liveTurns.value.filter(item => item.id !== turn!.id)
        }
      })
    } catch (failure) {
      if (started === generation) { turn.error = controller.signal.aborted ? '回复超时或已中止，请重试' : aiError(failure); turn.status = '回复未完成'; turn.active = false }
    } finally { clearTimeout(timer); if (started === generation) { sending.value = false; streamController = undefined } }
  }

  async function taskAction(conversationID: string, taskID: string, action: string, confirmed = false, checked = false, options: AITaskOptions = {}) {
    if (busyTasks.value.has(taskID)) return
    const started = generation
    busyTasks.value.add(taskID)
    error.value = ''
    try {
      const { data } = await actOnAITask(conversationID, taskID, action, confirmed, checked, options)
      if (started !== generation) return
      const conversation = conversations.value.find(item => item.id === conversationID)
      const index = conversation?.tasks.findIndex(item => item.id === taskID) ?? -1
      if (conversation && index >= 0) conversation.tasks[index] = data
      return true
    } catch (failure) {
      if (started === generation) { error.value = aiError(failure); await load(true) }
    } finally { if (started === generation) busyTasks.value.delete(taskID) }
  }

  return { conversations, currentID, current, settings, drafts, liveTurns, sending, loading, error, busyTasks, load, create, remove, send, taskAction }
})
