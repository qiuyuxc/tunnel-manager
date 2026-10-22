import { test } from 'node:test'
import assert from 'node:assert/strict'
import { resolveLanguage, languagePreference, t, localized, translateError, setLanguage } from '../i18n.ts'

test('manual language overrides ordered supported browser languages', () => {
  assert.equal(resolveLanguage('en', ['zh-CN']), 'en')
  assert.equal(resolveLanguage('zh-CN', ['en-US']), 'zh-CN')
  assert.equal(resolveLanguage('system', ['fr-FR', 'zh-TW', 'en-US']), 'zh-CN')
  assert.equal(resolveLanguage('system', ['en-GB', 'zh-CN']), 'en')
  assert.equal(resolveLanguage('system', ['ja']), 'en')
})
test('manual preferences can return to system language and preserve interpolated values', () => {
  const previous = languagePreference.value
  try {
    setLanguage('en')
    const value = '用户内容.example.com'
    assert.ok(t('m_7b1c3d4be6d4', { p0: value }).includes(value))
    setLanguage('system')
    assert.equal(languagePreference.value, 'system')
    setLanguage('invalid' as never)
    assert.equal(languagePreference.value, 'system')
  } finally { languagePreference.value = previous }
})
test('changing locale updates shared labels without changing stored values', () => {
  const previous = languagePreference.value
  try {
    languagePreference.value = 'zh-CN'
    const labels = localized(() => [{ value: 'pending', label: t('m_a8b0bd9c65f7') }])
    assert.equal(labels[0].label, '待确认')
    languagePreference.value = 'en'
    assert.equal(labels[0].value, 'pending')
    assert.ok(!/[\u4e00-\u9fff]/.test(labels[0].label))
    assert.equal(translateError('Cloudflare: opaque external error'), 'Cloudflare: opaque external error')
  } finally { languagePreference.value = previous }
})
