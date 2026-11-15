import { reactive } from 'vue'

const sheets = reactive<symbol[]>([])
let savedOverflow = ''

export function acquireSheet(token: symbol) {
  if (sheets.includes(token)) return
  if (sheets.length === 0) savedOverflow = document.body.style.overflow
  sheets.push(token)
  document.body.style.overflow = 'hidden'
}

export function isTopSheet(token: symbol) {
  return sheets[sheets.length - 1] === token
}

export function sheetLayer(token: symbol) {
  return Math.max(0, sheets.indexOf(token))
}

export function releaseSheet(token: symbol) {
  const index = sheets.indexOf(token)
  if (index < 0) return
  sheets.splice(index, 1)
  if (sheets.length === 0) document.body.style.overflow = savedOverflow
}
