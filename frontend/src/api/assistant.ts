import { api } from './index'

export interface AIConnection { endpoint: string; model: string; key_set: boolean }
export interface AISettings {
  shared_enabled: boolean
  is_admin: boolean
  configured: boolean
  personal?: AIConnection
  shared?: AIConnection
}
export interface AITask {
  id: string
  tool: string
  title: string
  arguments: Record<string, unknown>
  status: 'pending' | 'paused' | 'running' | 'succeeded' | 'unknown' | 'cancelled'
  result?: string
}
export interface AIConversation {
  id: string
  title: string
  updated_at: number
  messages: { role: 'user' | 'assistant'; content: string }[]
  tasks: AITask[]
}
export interface AISaveSettings {
  scope: 'personal' | 'shared'
  endpoint: string
  model: string
  api_key: string
  shared_enabled?: boolean
}

export const getAISettings = () => api.get<AISettings>('/assistant/settings')
export const saveAISettings = (payload: AISaveSettings) => api.put<AISettings>('/assistant/settings', payload)
export const getAIConversations = () => api.get<{ conversations: AIConversation[] }>('/assistant/conversations')
export const newAIConversation = () => api.post<AIConversation>('/assistant/conversations')
export const deleteAIConversation = (id: string) => api.delete(`/assistant/conversations/${encodeURIComponent(id)}`)
export const sendAIMessage = (id: string, message: string, includeResources: boolean) =>
  api.post<AIConversation>(`/assistant/conversations/${encodeURIComponent(id)}/messages`, { message, include_resources: includeResources }, { timeout: 180000 })
export const actOnAITask = (conversationID: string, taskID: string, action: string, confirmed = false, checkedResources = false) =>
  api.post<AITask>(`/assistant/conversations/${encodeURIComponent(conversationID)}/tasks/${encodeURIComponent(taskID)}`, { action, confirmed, checked_resources: checkedResources }, { timeout: 180000 })
