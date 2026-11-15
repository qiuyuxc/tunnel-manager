import assert from 'node:assert/strict'
import test from 'node:test'
import { acquireSheet, isTopSheet, releaseSheet, sheetLayer } from './sheetStack.ts'

test('nested sheets cover the parent and keep scrolling locked until the last sheet closes', () => {
  const previous = Object.getOwnPropertyDescriptor(globalThis, 'document')
  const style = { overflow: 'scroll' }
  Object.defineProperty(globalThis, 'document', { configurable: true, value: { body: { style } } })
  const parent = Symbol('parent'), child = Symbol('child')
  try {
    acquireSheet(parent)
    acquireSheet(child)
    assert.equal(style.overflow, 'hidden')
    assert.equal(isTopSheet(parent), false)
    assert.equal(isTopSheet(child), true)
    assert.ok(sheetLayer(child) > sheetLayer(parent), 'the child panel and scrim must be above the parent')
    releaseSheet(child)
    assert.equal(style.overflow, 'hidden')
    assert.equal(isTopSheet(parent), true)
    releaseSheet(parent)
    assert.equal(style.overflow, 'scroll')
  } finally {
    releaseSheet(child)
    releaseSheet(parent)
    if (previous) Object.defineProperty(globalThis, 'document', previous)
    else Reflect.deleteProperty(globalThis, 'document')
  }
})

test('removing an underlying page keeps its child modal locked and releases cleanly', () => {
  const previous = Object.getOwnPropertyDescriptor(globalThis, 'document')
  const style = { overflow: '' }
  Object.defineProperty(globalThis, 'document', { configurable: true, value: { body: { style } } })
  const parent = Symbol('parent'), child = Symbol('child')
  try {
    acquireSheet(parent)
    acquireSheet(child)
    releaseSheet(parent)
    assert.equal(style.overflow, 'hidden')
    assert.equal(isTopSheet(child), true)
    assert.equal(sheetLayer(child), 0)
    releaseSheet(child)
    releaseSheet(child)
    assert.equal(style.overflow, '')
  } finally {
    releaseSheet(child)
    releaseSheet(parent)
    if (previous) Object.defineProperty(globalThis, 'document', previous)
    else Reflect.deleteProperty(globalThis, 'document')
  }
})
