export interface AssistantStreamEvent { type: string; data: Record<string, unknown> }

export async function readAssistantStream(body: ReadableStream<Uint8Array>, onEvent: (event: AssistantStreamEvent) => void): Promise<void> {
  const reader = body.getReader()
  const decoder = new TextDecoder()
  let buffer = '', total = 0, completed = false
  try {
    while (!completed) {
      const { value, done } = await reader.read()
      if (done) break
      total += value.length
      if (total > 4 * 1024 * 1024) throw new Error('回复过长，请缩小请求范围')
      buffer += decoder.decode(value, { stream: true })
      let boundary: RegExpMatchArray | null
      while ((boundary = buffer.match(/\r?\n\r?\n/))) {
        const frame = buffer.slice(0, boundary.index)
        buffer = buffer.slice(boundary.index! + boundary[0].length)
        let type = 'message'
        const data: string[] = []
        for (const line of frame.split(/\r?\n/)) {
          if (line.startsWith('event:')) type = line.slice(6).trim()
          if (line.startsWith('data:')) data.push(line.slice(5).replace(/^ /, ''))
        }
        if (!data.length) continue
        const payload = JSON.parse(data.join('\n')) as Record<string, unknown>
        if (type === 'error') throw new Error(typeof payload.error === 'string' ? payload.error : '回复未完成，请重试')
        onEvent({ type, data: payload })
        if (type === 'done') { completed = true; break }
      }
    }
    if (!completed) throw new Error('连接已中断，回复未完整接收，尚未生成新任务')
  } finally { await reader.cancel().catch(() => {}); reader.releaseLock() }
}
