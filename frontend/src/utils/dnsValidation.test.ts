import assert from 'node:assert/strict'
import test from 'node:test'
import { DNS_BATCH_LIMIT, DNS_TYPES, DNSBatchStoppedError, defaultDNSData, dnsFailureUncertain, editableRecord, normalizeDNSInput, parseDNSBatch, recordInput, runDNSBatch, validateDNSContent, validateDNSRecord } from './dnsValidation.ts'
import type { DNSRecord } from '../api/index.ts'

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

test('validates all nine types, structured boundaries and proxy normalization', () => {
  const content = { A: '192.0.2.1', AAAA: '::', CNAME: 'target.example.com', TXT: ' raw ', MX: 'mail.example.com', NS: 'ns.example.com', PTR: 'host.example.com', SRV: '', CAA: '' }
  for (const type of DNS_TYPES) {
    const record = { type, name: type === 'SRV' ? '_sip._tcp.example.com' : 'example.com', ttl: 1, content: content[type], priority: 0, proxied: true, data: defaultDNSData(type) }
    if (type === 'SRV') record.data!.target = 'sip.example.com'
    assert.equal(validateDNSRecord(record), null, type)
    assert.equal(normalizeDNSInput(record).proxied, ['A', 'AAAA', 'CNAME'].includes(type))
  }
  assert.ok(validateDNSRecord({ type: 'SRV', name: '_sip._tcp.example.com', ttl: 1, content: '', data: { priority: 0, port: 443, target: 'a.example' } }))
  assert.ok(validateDNSRecord({ type: 'CAA', name: '@', ttl: 1, content: '', data: { flags: 256, tag: 'issue', value: 'ca.example' } }))
  assert.ok(validateDNSContent('A', '01.2.3.4'))
  assert.ok(validateDNSContent('AAAA', 'fe80::1%eth0'))
})

test('structured edits preserve SRV fields, zero priorities, CAA values and unknown read-only records', () => {
  const srv: DNSRecord = { id: 'srv', type: 'SRV', name: '_sip._tcp.example.com', content: '0 5 5060 sip.example.com', ttl: 120, data: { priority: 0, weight: 5, port: 5060, target: 'sip.example.com', service: '_sip', proto: '_tcp', name: 'example.com' } }
  assert.deepEqual(normalizeDNSInput(recordInput(srv)).data, srv.data)
  assert.equal(editableRecord(srv), true)
  const changed = normalizeDNSInput({ ...recordInput(srv), name: '_xmpp._udp.new.example.com' })
  assert.equal(changed.data?.service, '_xmpp')
  assert.equal(changed.data?.proto, '_udp')
  assert.equal(changed.data?.name, 'new.example.com')
  const caa: DNSRecord = { id: 'caa', type: 'CAA', name: 'example.com', content: '128 iodef "mailto:admin@example.com"', ttl: 1, data: { flags: 128, tag: 'iodef', value: 'mailto:admin@example.com' } }
  assert.deepEqual(normalizeDNSInput(recordInput(caa)).data, caa.data)
  assert.equal(editableRecord({ ...srv, type: 'HTTPS' }), false)
  assert.equal(editableRecord({ ...srv, data: undefined }), false)
  assert.equal(editableRecord({ ...caa, data: { ...caa.data, extra: 'future' } } as DNSRecord), false)
})

test('batch preview retains original line numbers, skips blank and duplicate/existing values', () => {
  const preview = parseDNSBatch('A', '192.0.2.1\r\n\r\nbad\n192.0.2.1\n192.0.2.2', ['192.0.2.2'])
  assert.deepEqual(preview.rows.map(row => row.line), [1, 3, 4, 5])
  assert.ok(preview.rows[1].error)
  assert.match(preview.rows[2].skipped!, /第 1 行/)
  assert.match(preview.rows[3].skipped!, /已存在/)
  assert.equal(parseDNSBatch('AAAA', '2001:db8::1\n2001:0db8:0:0:0:0:0:1').rows[1].skipped, '与第 1 行重复，跳过')
  assert.ok(parseDNSBatch('NS', 'NS.EXAMPLE.COM\nns.example.com.').rows[1].skipped)
})

test('batch preserves TXT whitespace, comma and case; rejects oversized and structured batches', () => {
  const rows = parseDNSBatch('TXT', '  text, A  \ntext, A\nTEXT, A').rows
  assert.equal(rows[0].content, '  text, A  ')
  assert.equal(rows.filter(row => row.skipped).length, 0)
  for (const type of ['CNAME', 'SRV', 'CAA'] as const) assert.ok(parseDNSBatch(type, 'value').error)
  assert.ok(parseDNSBatch('TXT', Array(DNS_BATCH_LIMIT + 1).fill('x').join('\n')).error)
  assert.ok(parseDNSBatch('TXT', '中'.repeat(1366)).rows[0].error)
})

test('batch sends serially and explicit retry never repeats successful or duplicate rows', async () => {
  let active = 0, maximum = 0
  const sent: string[] = []
  const rows = parseDNSBatch('A', '192.0.2.1\n192.0.2.2\n192.0.2.1\n192.0.2.3').rows
  const results = await runDNSBatch(rows, async row => {
    active++; maximum = Math.max(maximum, active); sent.push(row.content)
    await Promise.resolve(); active--
    if (row.content === '192.0.2.2') throw { response: { status: 400, data: { error: 'mock rejection' } } }
  })
  assert.equal(maximum, 1)
  assert.equal(results.length, 3)
  assert.deepEqual(results.map(row => row.success), [true, false, true])
  const failedText = results.filter(row => !row.success).map(row => row.content).join('\n')
  await runDNSBatch(parseDNSBatch('A', failedText).rows, async row => { sent.push(row.content) })
  assert.deepEqual(sent, ['192.0.2.1', '192.0.2.2', '192.0.2.3', '192.0.2.2'])
  await assert.rejects(runDNSBatch(parseDNSBatch('A', 'bad').rows, async () => assert.fail('invalid batch sent')))
})

test('ambiguous network results stop the queue and never claim definite failure', async () => {
  const sent: string[] = []
  const results = await runDNSBatch(parseDNSBatch('A', '192.0.2.1\n192.0.2.2\n192.0.2.3').rows, async row => {
    sent.push(row.content)
    if (row.content.endsWith('.2')) throw new Error('network disconnected')
  })
  assert.deepEqual(sent, ['192.0.2.1', '192.0.2.2'])
  assert.equal(results[0].success, true)
  assert.equal(results[1].uncertain, true)
  assert.match(results[1].message, /结果未知，请核对后再重试/)
  assert.equal(results[2].notSent, true)
  assert.equal(dnsFailureUncertain({ response: { status: 400, data: { error: 'request failed: EOF' } } }), true)
  assert.equal(dnsFailureUncertain({ response: { status: 400, data: { error: 'read response failed' } } }), true)
  assert.equal(dnsFailureUncertain({ response: { status: 502 } }), true)
  assert.equal(dnsFailureUncertain({ response: { status: 400, data: { error: 'cloudflare API error: duplicate record' } } }), false)
})

test('a changed batch context stops before sending and labels all remaining rows as unsent', async () => {
  const rows = parseDNSBatch('A', '192.0.2.1\n192.0.2.2\n192.0.2.3').rows
  for (const successfulCount of [0, 1]) {
    const sent: string[] = []
    const results = await runDNSBatch(rows, async row => {
      if (sent.length === successfulCount) throw new DNSBatchStoppedError('登录会话已变更')
      sent.push(row.content)
    })
    assert.equal(sent.length, successfulCount)
    assert.equal(results.filter(row => row.success).length, successfulCount)
    assert.ok(results.slice(successfulCount).every(row => row.notSent && !row.uncertain && row.message.includes('登录会话已变更')))
  }
})
