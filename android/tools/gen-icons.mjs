/**
 * Generates the nav VectorDrawables from the web console's icon table.
 *
 * navigation.ts is the single place the console defines its icons; the sidebar
 * and the phone tab bar already share it. Generating the native copies from the
 * same table keeps a third consumer from inventing its own drawing.
 *
 *   node android/tools/gen-icons.mjs
 */
import { readFileSync, readdirSync, writeFileSync, mkdirSync, rmSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const root = resolve(here, '..', '..')
const src = readFileSync(resolve(root, 'frontend/src/navigation.ts'), 'utf8')

const block = /export const icons = \{([\s\S]*?)\n\} as const/.exec(src)
if (!block) throw new Error('icons object not found in navigation.ts')

const outDir = resolve(root, 'android/app/src/main/res/drawable')
mkdirSync(outDir, { recursive: true })

const num = (attrs, name, fallback = 0) => {
  const m = new RegExp(`${name}="([-0-9.]+)"`).exec(attrs)
  return m ? parseFloat(m[1]) : fallback
}

function attrsOf(tag) {
  const m = /^<([a-zA-Z]+)([\s\S]*?)\/>$/.exec(tag.trim())
  if (!m) throw new Error(`unparsed element: ${tag.slice(0, 60)}`)
  return { name: m[1], attrs: m[2] }
}

/** Feather draws with primitives; a VectorDrawable only understands paths. */
function toPath(tag) {
  const { name, attrs } = attrsOf(tag)
  switch (name) {
    case 'path':
      return /d="([^"]+)"/.exec(attrs)[1]
    case 'line': {
      const [x1, y1, x2, y2] = ['x1', 'y1', 'x2', 'y2'].map((a) => num(attrs, a))
      return `M${x1},${y1} L${x2},${y2}`
    }
    case 'rect': {
      const x = num(attrs, 'x'), y = num(attrs, 'y')
      const w = num(attrs, 'width'), h = num(attrs, 'height')
      const r = num(attrs, 'rx', 0)
      if (r > 0) {
        // Rounded rectangle as two arcs plus two edges.
        return `M${x + r},${y} h${w - 2 * r} a${r},${r} 0 0 1 ${r},${r} v${h - 2 * r}`
          + ` a${r},${r} 0 0 1 ${-r},${r} h${-(w - 2 * r)} a${r},${r} 0 0 1 ${-r},${-r}`
          + ` v${-(h - 2 * r)} a${r},${r} 0 0 1 ${r},${-r} z`
      }
      return `M${x},${y} h${w} v${h} h${-w} z`
    }
    case 'circle': {
      const cx = num(attrs, 'cx'), cy = num(attrs, 'cy'), r = num(attrs, 'r')
      return `M${cx - r},${cy} a${r},${r} 0 1 0 ${2 * r},0 a${r},${r} 0 1 0 ${-2 * r},0`
    }
    case 'polyline':
    case 'polygon': {
      const points = /points="([^"]+)"/.exec(attrs)[1].trim()
        .split(/[\s,]+/).map(Number)
      const parts = []
      for (let i = 0; i < points.length; i += 2) {
        parts.push(`${i === 0 ? 'M' : 'L'}${points[i]},${points[i + 1]}`)
      }
      return parts.join(' ') + (name === 'polygon' ? ' z' : '')
    }
    default:
      throw new Error(`no VectorDrawable mapping for <${name}>`)
  }
}

// The terminator allows end-of-block: the last entry in the table has no
// trailing newline, and a `\\n`-only match silently dropped the final icon —
// which then looked "stale" to the cleanup pass below and got deleted.
const entries = [...block[1].matchAll(/([a-zA-Z]+):\s*'([\s\S]*?)',(?=\n|$)/g)]
if (!entries.length) throw new Error('no icons parsed')

const written = []
for (const [, key, svg] of entries) {
  const body = svg.replace(/^<svg[^>]*>/, '').replace(/<\/svg>$/, '')
  const tags = body.match(/<[a-zA-Z]+[\s\S]*?\/>/g) || []
  const paths = tags.map(toPath)
  // Feather icons are 24x24 stroke drawings; the shell tints them per state,
  // so the colour baked in here is only a placeholder.
  const xml = [
    '<?xml version="1.0" encoding="utf-8"?>',
    `<!-- Generated from frontend/src/navigation.ts (icons.${key}) by android/tools/gen-icons.mjs -->`,
    '<vector xmlns:android="http://schemas.android.com/apk/res/android"',
    '    android:width="24dp"',
    '    android:height="24dp"',
    '    android:viewportWidth="24"',
    '    android:viewportHeight="24">',
    ...paths.map((d) => [
      '    <path',
      '        android:fillColor="#00000000"',
      '        android:strokeColor="#FF000000"',
      '        android:strokeWidth="2"',
      '        android:strokeLineCap="round"',
      '        android:strokeLineJoin="round"',
      `        android:pathData="${d}" />`,
    ].join('\n')),
    '</vector>',
    '',
  ].join('\n')
  const file = resolve(outDir, `ic_nav_${key}.xml`)
  writeFileSync(file, xml)
  written.push(key)
}

// Drop icons that used to exist but no longer do, so a rename cannot leave a
// stale drawable behind that nothing references and nothing cleans up.
const known = new Set(written.map((k) => `ic_nav_${k}.xml`))
for (const name of readdirSync(outDir)) {
  if (/^ic_nav_.*\.xml$/.test(name) && !known.has(name)) {
    rmSync(resolve(outDir, name))
    console.log(`removed stale ${name}`)
  }
}


console.log(`wrote ${written.length} icons: ${written.join(', ')}`)
