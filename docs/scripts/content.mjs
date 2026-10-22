import { readdirSync, readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs'
import { resolve, relative, dirname } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const source = resolve(root, 'content')
function files(directory) { return readdirSync(directory, { withFileTypes: true }).flatMap(entry => entry.isDirectory() ? files(resolve(directory, entry.name)) : [resolve(directory, entry.name)]) }
const items = files(source).filter(file => file.endsWith('.json'))
const sections = JSON.parse(readFileSync(resolve(root, '.vitepress/navigation.json'), 'utf8'))
const paths = ['index', ...sections.flatMap(section => section.groups.flatMap(group => group.pages.map(page => page.startsWith('/') ? page.slice(1) : `${section.path}/${page}`)))]
const actual = items.map(file => relative(source, file).replace(/\.json$/, '')).sort()
if (JSON.stringify(actual) !== JSON.stringify([...paths].sort())) throw new Error('Documentation content does not match the shared page manifest')
const labels = JSON.parse(readFileSync(resolve(root, '.vitepress/labels.json'), 'utf8'))
if (JSON.stringify(Object.keys(labels['zh-CN']).sort()) !== JSON.stringify(Object.keys(labels.en).sort())) throw new Error('Documentation label keys differ between languages')
for (const language of ['zh-CN', 'en']) {
  for (const path of paths.filter(path => path !== 'index')) {
    if (!labels[language][path]) throw new Error(`Missing ${language} navigation label: ${path}`)
  }
}
for (const file of items) {
  const page = JSON.parse(readFileSync(file, 'utf8'))
  if (typeof page['zh-CN'] !== 'string' || typeof page.en !== 'string' || !page['zh-CN'].trim() || !page.en.trim()) throw new Error(`Missing bilingual content: ${file}`)
  const path = relative(source, file).replace(/\.json$/, '.md')
  for (const [language, directory] of [['zh-CN', root], ['en', resolve(root, 'en')]]) {
    const target = resolve(directory, path)
    const content = page[language]
    if (process.argv.includes('--check')) {
      if (!existsSync(target) || readFileSync(target, 'utf8') !== content) throw new Error(`Generated documentation is stale: ${target}`)
    } else { mkdirSync(dirname(target), { recursive: true }); writeFileSync(target, content) }
  }
}
console.log(`Validated/generated ${items.length} bilingual documentation pages`)
