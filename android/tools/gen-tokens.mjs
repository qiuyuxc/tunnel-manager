import { readFileSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..', '..')
const { palettes } = JSON.parse(readFileSync(resolve(root, 'android/palette-tokens.json'), 'utf8'))
const roles = {
  canvas: 'canvas', canvasSoft: 'canvas', canvasSoft2: 'raised', canvasRaised: 'panel',
  ink: 'ink', body: 'body', mute: 'mute', hairline: 'line', hairlineStrong: 'lineStrong',
  link: 'accent', linkHover: 'accent', focus: 'accent', success: 'accent',
  error: 'error', warning: 'warning', info: 'info', sidebar: 'sidebar',
  sidebarHover: 'raised', sidebarActive: 'accent', sidebarActiveBg: 'accentSoft',
  sidebarText: 'body', sidebarTextActive: 'accent', sidebarHoverText: 'ink', sidebarDivider: 'line',
  brandIcon: 'accent', headerBg: 'canvas', headerBorder: 'line',
  statusHealthyBg: 'accentSoft', statusHealthyBorder: 'accentSoft', statusHealthyText: 'accent',
  statusDownBg: 'errorSoft', statusDownBorder: 'errorLine', statusDownText: 'error',
  statusDegradedBg: 'warningSoft', statusDegradedBorder: 'warningLine', statusDegradedText: 'warning',
  btnPrimaryBg: 'accent', btnPrimaryText: 'accentText', btnPrimaryHover: 'accent',
  btnSecondaryBorder: 'line', btnSecondaryHoverBorder: 'lineStrong', btnGhostHover: 'ripple',
  bannerWarningBg: 'warningSoft', bannerWarningBorder: 'warningLine', bannerWarningText: 'warning',
  bannerInfoBg: 'infoSoft', bannerInfoBorder: 'infoLine', bannerInfoText: 'info',
  resultErrorBg: 'errorSoft', resultErrorBorder: 'errorLine', resultErrorText: 'error',
  resultSuccessBg: 'accentSoft', resultSuccessBorder: 'accentSoft', resultSuccessText: 'accent',
  glass: 'glass', glassHighlight: 'highlight',
}
const constantName = value => value.replace(/([a-z0-9])([A-Z])/g, '$1_$2').toUpperCase()
const hex = value => {
  if (!/^#(?:[\dA-F]{6}|[\dA-F]{8})$/i.test(value)) throw new Error(`Invalid ARGB color: ${value}`)
  return (value.length === 7 ? 'FF' + value.slice(1) : value.slice(1)).toUpperCase()
}
const fields = Object.keys(roles)
const java = ['package com.tunnelmanager.app;', '', 'final class Palette {', '']
for (const palette of palettes) {
  for (const role of Object.values(roles)) hex(palette.colors[role])
  java.push(`    static final Palette ${constantName(palette.id)} = new Palette("${palette.id}", "${palette.label}", ${palette.dark});`)
}
java.push('', '    final String id;', '    final String label;', '    final boolean dark;', '')
for (const field of fields) java.push(`    int ${field};`)
java.push('', '    private Palette(String id, String label, boolean dark) {',
  '        this.id = id;', '        this.label = label;', '        this.dark = dark;', '    }', '')
for (const palette of palettes) {
  java.push('    static {', `        Palette palette = ${constantName(palette.id)};`)
  for (const [field, role] of Object.entries(roles)) java.push(`        palette.${field} = 0x${hex(palette.colors[role])};`)
  java.push('    }', '')
}
java.push('    int remap(int color, Palette next) {')
for (const field of fields) java.push(`        if (color == ${field}) return next.${field};`)
java.push('        return color;', '    }', '}', '')
const xml = ['<?xml version="1.0" encoding="utf-8"?>', '<resources>']
for (const palette of palettes) {
  for (const [role, value] of Object.entries(palette.colors)) {
    xml.push(`    <color name="${constantName(palette.id).toLowerCase()}_${constantName(role).toLowerCase()}">#${hex(value)}</color>`)
  }
}
xml.push('</resources>', '')
const outputs = [
  ['android/app/src/main/java/com/tunnelmanager/app/Palette.java', java.join('\n')],
  ['android/app/src/main/res/values/native_palette.xml', xml.join('\n')],
]
for (const [relative, content] of outputs) {
  const target = resolve(root, relative)
  if (process.argv.includes('--check')) {
    if (readFileSync(target, 'utf8') !== content) throw new Error(`Regenerate ${relative}`)
  } else {
    writeFileSync(target, content)
  }
}
console.log(`Native palettes ${process.argv.includes('--check') ? 'verified' : 'generated'}: ${palettes.length}`)
