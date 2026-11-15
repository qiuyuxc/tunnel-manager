import { readFileSync, writeFileSync, mkdirSync, readdirSync } from 'node:fs'
import { resolve, relative, dirname } from 'node:path'
import { createHash } from 'node:crypto'
import { createRequire } from 'node:module'

const root = resolve(import.meta.dirname, '../..')
const require = createRequire(resolve(root, 'frontend/package.json'))
const ts = require('typescript')
const { parse: parseSFC } = require('@vue/compiler-sfc')
const { parse: baseParse } = require('@vue/compiler-dom')
const han = /[\u3400-\u9fff]/u
const messages = {}
const sources = {}
const changes = []
function key(text, source) {
  const id = 'm_' + createHash('sha256').update(text).digest('hex').slice(0, 12)
  messages[id] = text
  ;(sources[id] ||= []).push(source)
  return id
}
function expression(text, file) { return `t('${key(text, file)}')` }
function apply(text, edits) {
  for (const [start, end, value] of edits.sort((a, b) => b[0] - a[0])) text = text.slice(0, start) + value + text.slice(end)
  return text
}
function script(text, file) {
  const ast = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS)
  const edits = []
  function visit(node) {
    if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) {
      if (!han.test(node.text)) return
      const parent = node.parent
      if ((ts.isPropertyAssignment(parent) && parent.name === node) || ts.isImportDeclaration(parent)) return
      edits.push([node.getStart(ast), node.end, expression(node.text, file)])
      return
    }
    if (ts.isTemplateExpression(node) && han.test(node.getText(ast))) {
      const params = {}
      let message = node.head.text
      node.templateSpans.forEach((span, index) => {
        const name = `p${index}`
        params[name] = span.expression.getText(ast)
        message += `{${name}}` + span.literal.text
      })
      const id = key(message, file)
      edits.push([node.getStart(ast), node.end, `t('${id}', { ${Object.entries(params).map(([name, value]) => `${name}: ${value}`).join(', ')} })`])
      return
    }
    ts.forEachChild(node, visit)
  }
  visit(ast)
  return apply(text, edits)
}
function template(text, file) {
  const ast = baseParse(text)
  const edits = []
  function visit(node) {
    if (node.type === 2 && han.test(node.content)) {
      const value = node.content.trim()
      const raw = node.loc.source
      const prefix = raw.match(/^\s*/)[0], suffix = raw.match(/\s*$/)[0]
      edits.push([node.loc.start.offset, node.loc.end.offset, `${prefix}{{ ${expression(value, file)} }}${suffix}`])
    } else if (node.type === 5) {
      const exp = node.content
      const transformed = script(exp.content, file)
      if (transformed !== exp.content) edits.push([exp.loc.start.offset, exp.loc.end.offset, transformed])
    } else if (node.type === 1) {
      for (const prop of node.props) {
        if (prop.type === 6 && prop.value && han.test(prop.value.content)) {
          edits.push([prop.loc.start.offset, prop.loc.end.offset, `:${prop.name}="${expression(prop.value.content, file)}"`])
        } else if (prop.type === 7 && prop.exp) {
          const transformed = script(prop.exp.content, file)
          if (transformed !== prop.exp.content) edits.push([prop.exp.loc.start.offset, prop.exp.loc.end.offset, transformed.replaceAll('"', '&quot;')])
        }
      }
    }
    for (const child of node.children || []) visit(child)
  }
  visit(ast)
  return apply(text, edits)
}
function files(directory) {
  return readdirSync(directory, { withFileTypes: true }).flatMap(entry => entry.isDirectory() ? files(resolve(directory, entry.name)) : [resolve(directory, entry.name)])
}
for (const file of files(resolve(root, 'frontend/src'))) {
  if (!/\.(vue|ts)$/.test(file) || file.endsWith('.test.ts') || file.endsWith('.d.ts')) continue
  const source = relative(root, file)
  const original = readFileSync(file, 'utf8')
  let output = original
  if (file.endsWith('.vue')) {
    const { descriptor } = parseSFC(original)
    const edits = []
    if (descriptor.template) {
      const block = descriptor.template
      edits.push([block.loc.start.offset, block.loc.end.offset, template(block.content, source)])
    }
    for (const block of [descriptor.scriptSetup, descriptor.script].filter(Boolean)) {
      let content = script(block.content, source)
      if (content !== block.content || (descriptor.template && template(descriptor.template.content, source) !== descriptor.template.content)) {
        const path = relative(dirname(file), resolve(root, 'frontend/src/i18n')).replaceAll('\\', '/')
        content = `\nimport { t } from '${path.startsWith('.') ? path : './' + path}'\n` + content
      }
      edits.push([block.loc.start.offset, block.loc.end.offset, content])
    }
    output = apply(original, edits)
  } else {
    output = script(original, source)
    if (output !== original) {
      const path = relative(dirname(file), resolve(root, 'frontend/src/i18n')).replaceAll('\\', '/')
      output = `import { t } from '${path.startsWith('.') ? path : './' + path}.ts'\n` + output
    }
  }
  if (output !== original) changes.push({ file, output })
}
// Java string literals are replaced at their source, never by scanning rendered views.
const javaToken = /\/\/[^\n]*|\/\*[\s\S]*?\*\/|"(?:[^"\\]|\\.)*"/g
for (const file of files(resolve(root, 'android/app/src/main/java'))) {
  if (!file.endsWith('.java')) continue
  const original = readFileSync(file, 'utf8'), edits = []
  const source = relative(root, file)
  for (const match of original.matchAll(javaToken)) {
    if (!match[0].startsWith('"') || !han.test(match[0])) continue
    const value = JSON.parse(match[0])
    if (value.startsWith('/') && !value.includes('\n')) continue // Telegram command identifiers.
    if (value.includes('<html') || value.includes('<style')) continue
    const id = key(value, source)
    edits.push([match.index, match.index + match[0].length, `I18n.text("${id}")`])
  }
  if (edits.length) changes.push({ file, output: apply(original, edits) })
}
for (const file of files(resolve(root, 'backend'))) {
  if (!file.endsWith('.go') || file.endsWith('_test.go') || file.includes('/experiments/')) continue
  const source = relative(root, file)
  for (const match of readFileSync(file, 'utf8').matchAll(javaToken)) {
    if (!match[0].startsWith('"') || !han.test(match[0])) continue
    try { key(JSON.parse(match[0]), source) } catch {}
  }
}
mkdirSync(resolve(root, 'locales'), { recursive: true })
const sorted = Object.fromEntries(Object.entries(messages).sort())
const existing = JSON.parse(readFileSync(resolve(root, 'locales/zh-CN.json'), 'utf8'))
writeFileSync(resolve(root, 'locales/zh-CN.json'), JSON.stringify(Object.fromEntries(Object.entries({ ...existing, ...sorted }).sort()), null, 2) + '\n')
const existingSources = JSON.parse(readFileSync(resolve(root, 'locales/sources.json'), 'utf8'))
for (const [id, paths] of Object.entries(sources)) existingSources[id] = [...new Set([...(existingSources[id] || []), ...paths])]
writeFileSync(resolve(root, 'locales/sources.json'), JSON.stringify(Object.fromEntries(Object.entries(existingSources).sort()), null, 2) + '\n')
const output = resolve(root, 'locales/extracted-changes.json')
writeFileSync(output, JSON.stringify(changes, null, 2))
console.log(`Extracted ${Object.keys(messages).length} messages from ${changes.length} files. Changes await review in locales/extracted-changes.json.`)
