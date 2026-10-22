import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '../..')
const read = path => JSON.parse(readFileSync(resolve(root, path), 'utf8'))
const chinese = read('locales/zh-CN.json'), english = read('locales/en.json')
test('every shared message has English text and preserves dynamic parameters', () => {
  assert.deepEqual(Object.keys(chinese).sort(), Object.keys(english).sort())
  for (const [id, message] of Object.entries(chinese)) {
    assert.equal(typeof english[id], 'string', id)
    assert.ok(english[id].trim(), id)
    assert.deepEqual([...message.matchAll(/\{p\d+\}/g)].map(match => match[0]).sort(), [...english[id].matchAll(/\{p\d+\}/g)].map(match => match[0]).sort(), id)
    const format = /%(?:\[\d+\])?[+#0 -]*\d*(?:\.\d+)?[a-zA-Z]/g
    assert.deepEqual([...message.matchAll(format)].map(match => match[0]).sort(), [...english[id].matchAll(format)].map(match => match[0]).sort(), `${id} format arguments`)
  }
})
test('Android and backend translations match the shared catalog', () => {
  const result = spawnSync(process.execPath, ['tools/i18n/generate.mjs', '--check'], { cwd: root, encoding: 'utf8' })
  assert.equal(result.status, 0, result.stderr || result.stdout)
})

test('translations preserve numeric limits, addresses and Telegram command names', () => {
  const domains = /(?<![\w.])(?:https?:\/\/)?(?:[a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}(?::\d+)?(?![\w.])/g
  const commands = Object.values(chinese).filter(value => /^\/[A-Za-z]*[\u4e00-\u9fff]+$/.test(value))
  const numbers = message => message.replace(/\{\w+\}|%(?:\[\d+\])?[+#0 -]*\d*(?:\.\d+)?[a-zA-Z]/g, '').match(/\d+(?:\.\d+)*/g) ?? []
  for (const [id, message] of Object.entries(chinese)) {
    assert.deepEqual(numbers(message).sort(), numbers(english[id]).sort(), `${id} numeric values`)
    assert.deepEqual(message.match(domains) ?? [], english[id].match(domains) ?? [], `${id} addresses`)
    for (const command of commands) {
      assert.equal(english[id].split(command).length, message.split(command).length, `${id} command ${command}`)
    }
  }
})
