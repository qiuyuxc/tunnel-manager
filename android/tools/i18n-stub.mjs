import { readFileSync, mkdirSync, writeFileSync } from 'node:fs'
import { resolve, dirname, join } from 'node:path'

// JVM helper tests use real Chinese catalog values without depending on Android Resources.
export function writeI18nStub(directory) {
  const catalog = JSON.parse(readFileSync(resolve(import.meta.dirname, '../../locales/zh-CN.json'), 'utf8'))
  const entries = Object.entries(catalog)
  const chunks = Array.from({ length: Math.ceil(entries.length / 200) }, (_, index) =>
    `static void load${index}() { ${entries.slice(index * 200, (index + 1) * 200).map(([id, value]) => `messages.put(${JSON.stringify(id)}, ${JSON.stringify(value)});`).join('\n')} }`)
  const path = join(directory, 'com/tunnelmanager/app/I18n.java')
  mkdirSync(dirname(path), { recursive: true })
  writeFileSync(path, `package com.tunnelmanager.app;
final class I18n {
  static final java.util.Map<String, String> messages = new java.util.HashMap<>();
  static { ${chunks.map((_, index) => `load${index}();`).join(' ')} }
  ${chunks.join('\n')}
  static String text(String id) { return messages.getOrDefault(id, id); }
  static String languageTag() { return "zh-CN"; }
  static java.util.Locale locale() { return java.util.Locale.CHINA; }
}`)
  return path
}
