import { createServer } from 'node:http'
import { readFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { resolve } from 'node:path'

const root = fileURLToPath(new URL('.', import.meta.url))
const fonts = resolve(root, '../../public/fonts')
const assets = new Map([
  ['/', ['index.html', 'text/html; charset=utf-8']],
  ['/prototype/ai-assistant', ['index.html', 'text/html; charset=utf-8']],
  ['/styles.css', ['styles.css', 'text/css; charset=utf-8']],
  ['/app.js', ['app.js', 'text/javascript; charset=utf-8']],
])

const server = createServer(async (request, response) => {
  const pathname = new URL(request.url, 'http://localhost').pathname
  const asset = assets.get(pathname)
  const font = /^\/fonts\/misans-(400|600)-core\.woff2$/.test(pathname)
  if (!['GET', 'HEAD'].includes(request.method) || (!asset && !font)) {
    response.writeHead(404, { 'Content-Type': 'text/plain; charset=utf-8' })
    response.end('设计预览中没有这个页面。')
    return
  }
  try {
    const content = await readFile(font ? resolve(fonts, pathname.split('/').pop()) : resolve(root, asset[0]))
    response.writeHead(200, {
      'Content-Type': font ? 'font/woff2' : asset[1],
      'Cache-Control': 'no-store',
      'X-Content-Type-Options': 'nosniff',
      'Content-Security-Policy': "default-src 'self'; script-src 'self'; style-src 'self'; font-src 'self'; img-src 'self' data:; connect-src 'none'; object-src 'none'; base-uri 'none'; frame-ancestors 'self'",
    })
    response.end(request.method === 'HEAD' ? undefined : content)
  } catch {
    response.writeHead(404)
    response.end('Preview asset unavailable')
  }
})

server.on('error', error => {
  console.error(`无法启动 8083 预览：${error.message}`)
  process.exitCode = 1
})
server.listen(8083, '0.0.0.0', () => {
  console.log('AI 助手设计预览：http://127.0.0.1:8083')
  console.log('独立静态设计稿，仅使用内存示例，不连接真实 AI 或业务 API。')
})
