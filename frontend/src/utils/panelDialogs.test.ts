import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync, readdirSync } from 'node:fs'
import { dirname, join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'
import ts from 'typescript'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

function nativeCalls(text: string) {
  const source = ts.createSourceFile('source.ts', text, ts.ScriptTarget.Latest, true)
  const calls: string[] = []
  const names = new Set(['alert', 'confirm', 'prompt'])
  function visit(node: ts.Node) {
    if (ts.isCallExpression(node)) {
      const callee = node.expression
      if (ts.isIdentifier(callee) && names.has(callee.text)) calls.push(callee.text)
      if (ts.isPropertyAccessExpression(callee) && ts.isIdentifier(callee.expression)
        && ['window', 'globalThis', 'self'].includes(callee.expression.text) && names.has(callee.name.text)) {
        calls.push(callee.getText(source))
      }
    }
    ts.forEachChild(node, visit)
  }
  visit(source)
  return calls
}

test('native dialog guard distinguishes panel methods and inert strings', () => {
  assert.deepEqual(nativeCalls('window.confirm("delete?"); prompt("name"); globalThis.alert("error")'), ['window.confirm', 'prompt', 'globalThis.alert'])
  assert.deepEqual(nativeCalls('panel.confirm({ title: "window.confirm()" }); panel.prompt({ title: "name" })'), [])
})

test('production UI uses panel dialogs instead of browser-native dialogs', () => {
  const failures: string[] = []
  function scan(directory: string) {
    for (const entry of readdirSync(directory, { withFileTypes: true })) {
      const path = join(directory, entry.name)
      if (entry.isDirectory()) scan(path)
      else if (/\.(vue|ts)$/.test(entry.name) && !entry.name.endsWith('.test.ts')) {
        const content = readFileSync(path, 'utf8')
        const scripts = entry.name.endsWith('.vue')
          ? [...content.matchAll(/<script\b[^>]*>([\s\S]*?)<\/script>/g)].map(match => match[1])
          : [content]
        for (const script of scripts) {
          failures.push(...nativeCalls(script).map(call => `${relative(root, path)}: ${call}`))
        }
      }
    }
  }
  scan(root)
  assert.deepEqual(failures, [])
})
