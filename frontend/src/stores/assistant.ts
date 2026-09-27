import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { useConfigStore } from './config'
import { actOnAITask, deleteAIConversation, getAIConversations, getAISettings, newAIConversation, sendAIMessage, type AIConversation, type AISettings } from '../api/assistant'

export function aiError(error: unknown): string {
  const failure = error as { response?: { data?: { error?: string } }; message?: string }
  return failure.response?.data?.error || '请求失败，请检查网络后重试；执行请求失败时请先核对实际资源。'
}

export const useAssistantStore = defineStore('assistant', () => {
  const config = useConfigStore()
  const conversations = ref<AIConversation[]>([])
  const currentID = ref('')
  const settings = ref<AISettings | null>(null)
  const drafts = ref<Record<string, string>>({})
  const sending = ref(false)
  const loading = ref(false)
  const error = ref('')
  const busyTasks = ref(new Set<string>())
  const current = computed(() => conversations.value.find(conversation => conversation.id === currentID.value))
  let generation = 0

  watch(() => config.token, () => {
    generation++
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

  async function send(includeResources: boolean) {
    if (sending.value) return
    const started = generation
    let draftKey = currentID.value || 'new'
    const message = (drafts.value[draftKey] || '').trim()
    if (!message) return
    sending.value = true
    error.value = ''
    try {
      const id = currentID.value || await create()
      if (!id || started !== generation) return
      if (draftKey === 'new') { drafts.value[id] = drafts.value.new || ''; delete drafts.value.new; draftKey = id }
      const { data } = await sendAIMessage(id, message, includeResources)
      if (started !== generation) return
      merge(data)
      if ((drafts.value[draftKey] || '').trim() === message) drafts.value[draftKey] = ''
    } catch (failure) { if (started === generation) error.value = aiError(failure) }
    finally { if (started === generation) sending.value = false }
  }

  async function taskAction(conversationID: string, taskID: string, action: string, confirmed = false, checked = false) {
    if (busyTasks.value.has(taskID)) return
    const started = generation
    busyTasks.value.add(taskID)
    error.value = ''
    try {
      const { data } = await actOnAITask(conversationID, taskID, action, confirmed, checked)
      if (started !== generation) return
      const conversation = conversations.value.find(item => item.id === conversationID)
      const index = conversation?.tasks.findIndex(item => item.id === taskID) ?? -1
      if (conversation && index >= 0) conversation.tasks[index] = data
    } catch (failure) {
      if (started === generation) { error.value = aiError(failure); await load(true) }
    } finally { if (started === generation) busyTasks.value.delete(taskID) }
  }

  return { conversations, currentID, current, settings, drafts, sending, loading, error, busyTasks, load, create, remove, send, taskAction }
})
