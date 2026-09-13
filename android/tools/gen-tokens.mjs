/**
 * Generates Palette.java from the web console's theme tokens.
 *
 * The native app and the web console have to agree on colour, and hand-copying
 * fifty-five tokens into Java would drift the first time a palette is touched.
 * styles.css stays the single source of truth; run this after editing it:
 *
 *   node android/tools/gen-tokens.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const root = resolve(here, '..', '..')
const css = readFileSync(resolve(root, 'frontend/src/styles.css'), 'utf8')

// styles.css lays the palettes out as four flat blocks. Only the two Vercel
// ones are emitted: the Claude pair turned out to read as near-identical at
// phone size, so the console ships a single visual theme with a day/night axis
// rather than four choices nobody could tell apart.
const BLOCKS = [
  { selector: ':root {', id: 'enterprise', label: 'Vercel 亮色', dark: false },
  { selector: '[data-theme="dark"] {', id: 'enterpriseDark', label: 'Vercel 暗色', dark: true },
]

function blockBody(selector) {
  const start = css.indexOf(selector)
  if (start < 0) throw new Error(`selector not found in styles.css: ${selector}`)
  const open = css.indexOf('{', start)
  const close = css.indexOf('\n}', open)
  return css.slice(open + 1, close)
}

/** #rgb/#rrggbb/#rrggbbaa and rgb()/rgba() into a packed 0xAARRGGBB int. */
function parseColor(raw) {
  const value = raw.trim()
  let m = /^#([0-9a-fA-F]{3,8})$/.exec(value)
  if (m) {
    let hex = m[1]
    if (hex.length === 3) hex = hex.split('').map((c) => c + c).join('')
    if (hex.length === 6) hex = 'ff' + hex
    if (hex.length !== 8) return null
    return parseInt(hex, 16) >>> 0
  }
  m = /^rgba?\(([^)]+)\)$/.exec(value)
  if (m) {
    const parts = m[1].split(',').map((p) => p.trim())
    if (parts.length < 3) return null
    const channel = (p) => Math.max(0, Math.min(255, Math.round(parseFloat(p))))
    const [r, g, b] = parts.slice(0, 3).map(channel)
    const a = parts.length > 3 ? Math.max(0, Math.min(1, parseFloat(parts[3]))) : 1
    return (((Math.round(a * 255) << 24) | (b << 16) | (g << 8) | r) >>> 0)
  }
  return null
}

/** `--color-ink` -> `ink`; drops the house prefix. */
function fieldName(token) {
  return token.replace(/^--color-/, '').replace(/-([a-z0-9])/g, (_, c) => c.toUpperCase())
}

const palettes = []
for (const block of BLOCKS) {
  const colors = new Map()
  for (const line of blockBody(block.selector).split('\n')) {
    const m = /^\s*(--[a-z0-9-]+)\s*:\s*([^;]+);/i.exec(line)
    if (!m || !m[1].startsWith('--color-')) continue
    const argb = parseColor(m[2])
    if (argb === null) {
      console.warn(`skip non-colour token ${m[1]}: ${m[2].trim()}`)
      continue
    }
    colors.set(fieldName(m[1]), argb)
  }
  if (!colors.size) throw new Error(`no colours parsed for ${block.selector}`)
  palettes.push({ ...block, colors })
}

// Every palette must carry the same fields or the shell would inherit a light
// token in dark mode, which is exactly what the CSS comments warn about.
const fields = [...palettes[0].colors.keys()]
for (const p of palettes) {
  const missing = fields.filter((f) => !p.colors.has(f))
  const extra = [...p.colors.keys()].filter((f) => !fields.includes(f))
  if (missing.length || extra.length) {
    throw new Error(`palette ${p.id} token mismatch: missing=${missing} extra=${extra}`)
  }
}

const constName = (id) => id.replace(/([a-z0-9])([A-Z])/g, '$1_$2').toUpperCase()

const out = []
out.push('package com.tunnelmanager.app;')
out.push('')
out.push('/**')
out.push(' * Generated from frontend/src/styles.css by android/tools/gen-tokens.mjs.')
out.push(' *')
out.push(' * <p>Do not edit by hand: the web console owns the palette, and re-running the')
out.push(' * generator is the only way to keep the two from drifting apart.')
out.push(' */')
out.push('final class Palette {')
out.push('')
for (const p of palettes) {
  out.push(`    /** ${p.label} — ${p.selector.replace(/[{}]/g, '').trim()} */`)
  out.push(`    static final Palette ${constName(p.id)} = new Palette("${p.id}", "${p.label}", ${p.dark});`)
  out.push('')
}
out.push('    final String id;')
out.push('    final String label;')
out.push('    final boolean dark;')
out.push('')
for (const f of fields) out.push(`    int ${f};`)
out.push('')
out.push('    private Palette(String id, String label, boolean dark) {')
out.push('        this.id = id;')
out.push('        this.label = label;')
out.push('        this.dark = dark;')
out.push('    }')
out.push('')
for (const p of palettes) {
  out.push(`    static {`)
  out.push(`        Palette p = ${constName(p.id)};`)
  const width = Math.max(...fields.map((f) => f.length))
  for (const f of fields) {
    out.push(`        p.${f.padEnd(width)} = 0x${p.colors.get(f).toString(16).toUpperCase().padStart(8, '0')};`)
  }
  out.push('    }')
  out.push('')
}
// Declared last on purpose: an array initialiser may not forward-reference the
// palette constants, and by this point every static block has filled them in.
out.push('    static final Palette[] ALL = {')
out.push(palettes.map((p) => `        ${constName(p.id)},`).join('\n'))
out.push('    };')
out.push('')
out.push('    /** Looks a palette up by id, falling back to the Vercel light default. */')
out.push('    static Palette byId(String id) {')
out.push('        for (Palette p : ALL) {')
out.push('            if (p.id.equals(id)) return p;')
out.push('        }')
out.push('        return ENTERPRISE;')
out.push('    }')
out.push('}')
out.push('')

const target = resolve(root, 'android/app/src/main/java/com/tunnelmanager/app/Palette.java')
writeFileSync(target, out.join('\n'))
console.log(`wrote ${target}: ${palettes.length} palettes x ${fields.length} tokens`)
