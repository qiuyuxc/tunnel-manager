import { createServer } from 'node:http'
import { readFile } from 'node:fs/promises'

const port = Number(process.env.PORT || 8083)
const files = new Map([
  ['/', ['index.html', 'text/html; charset=utf-8']],
  ['/index.html', ['index.html', 'text/html; charset=utf-8']],
  ['/styles.css', ['styles.css', 'text/css; charset=utf-8']],
  ['/app.js', ['app.js', 'text/javascript; charset=utf-8']],
])

const server = createServer(async (request, response) => {
  const pathname = new URL(request.url, 'http://localhost').pathname
  const file = files.get(pathname)
  if (!['GET', 'HEAD'].includes(request.method) || !file) {
    response.writeHead(404, { 'Content-Type': 'text/plain; charset=utf-8' })
    response.end('Not found. This server only serves the isolated design preview.')
    return
  }
  try {
    const content = await readFile(new URL(file[0], import.meta.url))
    response.writeHead(200, {
      'Content-Type': file[1],
      'Cache-Control': 'no-store',
      'X-Content-Type-Options': 'nosniff',
      'Content-Security-Policy': "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'none'; font-src 'self'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'self'",
    })
    response.end(request.method === 'HEAD' ? undefined : content)
  } catch {
    response.writeHead(500, { 'Content-Type': 'text/plain; charset=utf-8' })
    response.end('Unable to load preview asset.')
  }
})

server.on('error', error => {
  console.error(`Design preview could not start: ${error.message}`)
  process.exitCode = 1
})

server.listen(port, '0.0.0.0', () => {
  console.log(`Tunnel Manager design preview: http://localhost:${port}`)
  console.log('Isolated sample data. No production routes or APIs are connected.')
})
