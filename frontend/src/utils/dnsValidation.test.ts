import assert from 'node:assert/strict'
import test from 'node:test'
import { validateDNSContent } from './dnsValidation.ts'

test('accepts values compatible with each DNS record type', () => {
  assert.equal(validateDNSContent('A', '8.8.8.8'), null)
  assert.equal(validateDNSContent('AAAA', '2001:db8::1'), null)
  assert.equal(validateDNSContent('CNAME', 'target.example.com'), null)
  assert.equal(validateDNSContent('MX', 'mail.example.com.'), null)
  assert.equal(validateDNSContent('TXT', '8.8.8.8'), null)
})

test('rejects an IP address before applying it to CNAME records', () => {
  assert.equal(
    validateDNSContent('CNAME', '8.8.8.8'),
    'CNAME 记录必须填写域名目标，不能填写 IP 地址',
  )
})

test('rejects values for the wrong address family', () => {
  assert.equal(validateDNSContent('A', '2001:db8::1'), 'A 记录必须填写有效的 IPv4 地址')
  assert.equal(validateDNSContent('AAAA', '8.8.8.8'), 'AAAA 记录必须填写有效的 IPv6 地址')
})
