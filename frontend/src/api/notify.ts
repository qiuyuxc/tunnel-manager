// Per-user notification preferences and delivery.
import { api } from './index'

export interface NotifySettings {
  channels: string[]
  events: Record<string, boolean>
  emails: string
  tg_bot_token_set: boolean
  tg_notify_chat_id: string
}

export interface SaveNotifyPayload {
  channels: string[]
  events: Record<string, boolean>
  emails: string
  tg_bot_token?: string
  tg_notify_chat_id: string
}

export function getNotifySettings() {
  return api.get<NotifySettings>('/notify/settings')
}

export function updateNotifySettings(payload: SaveNotifyPayload) {
  return api.put<NotifySettings>('/notify/settings', payload)
}

export function testNotify() {
  return api.post<{ message: string }>('/notify/test')
}
