import { strict as assert } from 'node:assert'
import { test } from 'node:test'
import { describeLabFailure, isLabAuthorized, validateLabLimits } from './lab.ts'

const verification = { verified_at: 123, owner_id: 'admin', domain: 'owner.example.org' }

test('lab diagnostics distinguish rejected HTTP statuses from transport errors without claiming an IP is unusable', () => {
  assert.equal(describeLabFailure({ status: 403 }, '200'), 'HTTP 403，不在本次接受状态码 200中')
  assert.match(describeLabFailure({ error: 'x509: certificate is not valid' }), /x509/)
  assert.match(describeLabFailure({ status: 200, error: 'timeout' }, '200'), /timeout/)
  assert.match(describeLabFailure({}), /未收到有效 HTTP 响应/)
})

test('lab authorization requires a verified challenge belonging to this administrator', () => {
  assert.equal(isLabAuthorized(undefined, 'admin'), false)
  assert.equal(isLabAuthorized({ ...verification, verified_at: 0 }, 'admin'), false)
  assert.equal(isLabAuthorized(verification, 'other-admin'), false)
  assert.equal(isLabAuthorized(verification, ''), false)
  assert.equal(isLabAuthorized({ ...verification, domain: '' }, 'admin'), false)
})

test('lab ownership depends on the verified domain and administrator, not probe Host or SNI', () => {
  assert.equal(isLabAuthorized(verification, 'admin'), true)
  const legacy = { ...verification, host: 'old.example.net', sni: 'old.example.net' }
  assert.equal(isLabAuthorized(legacy, 'admin'), true)
})

test('administrator lab limits require bounded integers', () => {
  const defaults = { lab_daily_request_limit: 100000, lab_requests_per_second: 10, lab_max_workers: 32 }
  assert.equal(validateLabLimits(defaults), '')
  assert.equal(validateLabLimits({ lab_daily_request_limit: 10000000, lab_requests_per_second: 1000, lab_max_workers: 256 }), '')
  for (const field of Object.keys(defaults)) {
    for (const value of [0, -1, 1.5, NaN, Infinity, undefined, 10000001]) {
      assert.notEqual(validateLabLimits({ ...defaults, [field]: value }), '')
    }
  }
})
