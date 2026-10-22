import { readFileSync, writeFileSync, mkdirSync } from 'node:fs'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '../..')
const xml = value => value.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll("'", "\\'").replaceAll('"', '\\"').replaceAll('\n', '\\n').replaceAll('\r', '\\r')
for (const [locale, folder] of [['zh-CN', 'values-zh'], ['en', 'values']]) {
  const catalog = JSON.parse(readFileSync(resolve(root, `locales/${locale}.json`), 'utf8'))
  const content = '<?xml version="1.0" encoding="utf-8"?>\n<!-- Generated from locales; edit the catalogs, then run tools/i18n/generate.mjs. -->\n<resources>\n' +
    Object.entries(catalog).map(([id, value]) => `    <string name="${id.replaceAll('.', '_')}" formatted="false">"${xml(value)}"</string>`).join('\n') + '\n</resources>\n'
  const directory = resolve(root, `android/app/src/main/res/${folder}`)
  mkdirSync(directory, { recursive: true })
  const file = resolve(directory, 'messages.xml')
  if (process.argv.includes('--check')) {
    if (readFileSync(file, 'utf8') !== content) throw new Error(`Generated translations are stale: ${file}`)
  } else writeFileSync(file, content)
}
console.log('Android translations verified/generated')
for (const locale of ['zh-CN', 'en']) {
  const content = readFileSync(resolve(root, `locales/${locale}.json`), 'utf8')
  const file = resolve(root, `backend/i18n/${locale}.json`)
  if (process.argv.includes('--check')) {
    if (readFileSync(file, 'utf8') !== content) throw new Error(`Generated backend translations are stale: ${file}`)
  } else writeFileSync(file, content)
}
