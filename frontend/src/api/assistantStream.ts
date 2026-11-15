import { t as tr, language, translateError } from '../i18n.ts'
import { readAssistantStream, type AssistantStreamEvent } from '../utils/assistantStream'

export async function sendAIStream(id: string, message: string, includeResources: boolean, signal: AbortSignal, onEvent: (event: AssistantStreamEvent) => void) {
  const token = localStorage.getItem('auth_token') || ''
  const response = await fetch(`/api/assistant/conversations/${encodeURIComponent(id)}/messages`, {
    method: 'POST', headers: { 'Content-Type': 'application/json', Accept: 'text/event-stream', 'X-Auth-Token': token, 'Accept-Language': language.value },
    body: JSON.stringify({ message, include_resources: includeResources, stream: true }), signal,
  })
  if (!response.ok) {
    const failure = await response.json().catch(() => ({}))
    if (response.status === 401 && token && token === localStorage.getItem('auth_token')) {
      for (const key of ['token', 'username', 'role', 'nickname', 'avatar', 'email']) localStorage.removeItem(`auth_${key}`)
      window.location.href = '/login'
    }
    throw new Error(translateError(failure.error || tr('m_f21a09b0eb04', { p0: response.status })))
  }
  if (response.headers.get('Content-Type')?.includes('text/event-stream')) {
    if (!response.body) throw new Error(tr('m_a676f235699a'))
    await readAssistantStream(response.body, onEvent)
  } else onEvent({ type: 'done', data: await response.json() })
}
