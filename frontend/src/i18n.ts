import { computed, ref } from 'vue'
import chinese from '../../locales/zh-CN.json' with { type: 'json' }
import english from '../../locales/en.json' with { type: 'json' }

export type Language = 'zh-CN' | 'en'
export type LanguagePreference = 'system' | Language
const storageKey = 'tm_language'
const catalogs: Record<Language, Record<string, string>> = { 'zh-CN': chinese, en: english }
const systemLanguages = () => typeof window === 'undefined' ? ['zh-CN'] : navigator.languages?.length ? navigator.languages : [navigator.language || 'en']
export function resolveLanguage(preference: string, languages: readonly string[]): Language {
  if (preference === 'zh-CN' || preference === 'en') return preference
  for (const language of languages) {
    if (/^zh(?:-|$)/i.test(language)) return 'zh-CN'
    if (/^en(?:-|$)/i.test(language)) return 'en'
  }
  return 'en'
}
function savedPreference(): LanguagePreference {
  try {
    const value = localStorage.getItem(storageKey)
    return value === 'zh-CN' || value === 'en' ? value : 'system'
  } catch { return 'system' }
}
export const languagePreference = ref<LanguagePreference>(savedPreference())
const browserLanguages = ref<readonly string[]>(systemLanguages())
export const language = computed(() => resolveLanguage(languagePreference.value, browserLanguages.value))
export function t(id: string, parameters: Record<string, unknown> = {}): string {
  const message = catalogs[language.value][id] ?? catalogs['zh-CN'][id] ?? id
  return message.replace(/\{(\w+)\}/g, (placeholder, name) => Object.prototype.hasOwnProperty.call(parameters, name) ? String(parameters[name]) : placeholder)
}
export function setLanguage(value: LanguagePreference) {
  if (!['system', 'zh-CN', 'en'].includes(value)) return
  languagePreference.value = value
  try { localStorage.setItem(storageKey, value) } catch {}
  updateDocumentLanguage()
  if (typeof window !== 'undefined') window.dispatchEvent(new CustomEvent('tm-language-changed'))
}
export function updateDocumentLanguage() {
  if (typeof document !== 'undefined') document.documentElement.lang = language.value
}
if (typeof window !== 'undefined') {
  updateDocumentLanguage()
  window.addEventListener('languagechange', () => { browserLanguages.value = systemLanguages(); updateDocumentLanguage() })
  window.addEventListener('storage', event => {
    if (event.key !== storageKey) return
    languagePreference.value = savedPreference()
    updateDocumentLanguage()
    window.dispatchEvent(new CustomEvent('tm-language-changed'))
  })
}
export const locale = computed(() => language.value === 'zh-CN' ? 'zh-CN' : 'en-US')
export function localized<T extends object>(factory: () => T): T {
  const current = computed(factory)
  return new Proxy(current.value, {
    get(_target, property) {
      const value = Reflect.get(current.value, property)
      return typeof value === 'function' ? value.bind(current.value) : value
    },
    ownKeys() { return Reflect.ownKeys(current.value) },
    getOwnPropertyDescriptor(_target, property) { return Reflect.getOwnPropertyDescriptor(current.value, property) },
  })
}
export function dateTime(value: Date | number | string, options?: Intl.DateTimeFormatOptions): string {
  const date = value instanceof Date ? value : new Date(value)
  return Number.isNaN(date.getTime()) ? '' : new Intl.DateTimeFormat(locale.value, options).format(date)
}

// Only application-generated errors are translated; API values and user content stay intact.
const errorEntries = Object.entries(chinese).filter(([id]) => english[id as keyof typeof english])
export function translateError(message: string): string {
  if (language.value === 'zh-CN') return message
  const exact = errorEntries.find(([, value]) => value === message)
  if (exact) return english[exact[0] as keyof typeof english]
  return message
}
