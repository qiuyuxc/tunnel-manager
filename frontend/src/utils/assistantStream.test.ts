import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readAssistantStream, type AssistantStreamEvent } from './assistantStream.ts'

function stream(text: string) { const bytes = new TextEncoder().encode(text); return new ReadableStream<Uint8Array>({ start(controller) { for (const byte of bytes) controller.enqueue(Uint8Array.of(byte)); controller.close() } }) }

test('stream decodes split UTF-8, reasoning and final events without treating partial output as done', async () => {
  const events: AssistantStreamEvent[] = []
  await readAssistantStream(stream(': heartbeat\r\n\r\nevent: reasoning\r\ndata: {"delta":"计划"}\r\n\r\nevent: text\ndata: {"delta":"你好"}\n\nevent: done\ndata: {"id":"chat","tasks":[]}\n\n'), event => events.push(event))
  assert.deepEqual(events.map(event => event.type), ['reasoning', 'text', 'done'])
  assert.equal(events[1].data.delta, '你好')
})

test('truncation and explicit error preserve received text but never emit a final conversation', async () => {
  for (const ending of ['', 'event: error\ndata: {"error":"服务暂不可用"}\n\n']) {
    const events: AssistantStreamEvent[] = []
    await assert.rejects(readAssistantStream(stream('event: text\ndata: {"delta":"partial"}\n\n' + ending), event => events.push(event)), /中断|服务暂不可用/)
    assert.deepEqual(events.map(event => event.type), ['text'])
  }
})
