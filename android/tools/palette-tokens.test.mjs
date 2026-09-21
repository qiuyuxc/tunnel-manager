import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..', '..')
const { palettes } = JSON.parse(readFileSync(resolve(root, 'android/palette-tokens.json'), 'utf8'))

function luminance(color) {
  const channels = color.slice(-6).match(/../g).map(channel => parseInt(channel, 16) / 255)
    .map(channel => channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4)
  return channels[0] * 0.2126 + channels[1] * 0.7152 + channels[2] * 0.0722
}

function contrast(first, second) {
  const values = [luminance(first), luminance(second)].sort((first, second) => second - first)
  return (values[0] + 0.05) / (values[1] + 0.05)
}

test('native light and dark palettes have matching semantic colors', () => {
  assert.deepEqual(palettes.map(palette => palette.id), ['liquidLight', 'liquidDark'])
  assert.deepEqual(palettes.map(palette => palette.dark), [false, true])
  assert.deepEqual(Object.keys(palettes[0].colors).sort(), Object.keys(palettes[1].colors).sort())
  for (const palette of palettes) {
    for (const color of Object.values(palette.colors)) assert.match(color, /^#(?:[\dA-F]{6}|[\dA-F]{8})$/)
  }
})

test('body, secondary text and primary buttons meet normal-text contrast', () => {
  for (const { id, colors } of palettes) {
    for (const text of ['ink', 'body', 'mute']) {
      for (const surface of ['canvas', 'panel', 'raised']) {
        assert.ok(contrast(colors[text], colors[surface]) >= 4.5, `${id}: ${text} on ${surface}`)
      }
    }
    assert.ok(contrast(colors.accentText, colors.accent) >= 4.5, `${id}: primary button`)
  }
})

test('committed Java and XML palettes match the source tokens', () => {
  const result = spawnSync(process.execPath, ['android/tools/gen-tokens.mjs', '--check'], {
    cwd: root, encoding: 'utf8',
  })
  assert.equal(result.status, 0, result.stderr || result.stdout)
})
