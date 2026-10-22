import { t as tr, localized } from '../i18n.ts'
export const assistantArgumentLabels: Record<string, string> = localized(() => ({
  id: tr('m_b0719015e486'), name: tr('m_d44e9b3d3b31'), zone_id: tr('m_d8399be73059'), record_id: tr('m_79081a3c812e'), tunnel_id: tr('m_17f261a9385c'), hostname: tr('m_1ac79aa3aff0'), service_url: tr('m_693f764b0b0b'),
  type: tr('m_ba40014ff496'), content: tr('m_7175aa5fa2ba'), data: tr('m_98d11221a1cd'), ttl: tr('m_5fada8da5e20'), proxied: tr('m_5e84ea61e838'), priority: tr('m_565d64601d4d'), weight: tr('m_db18831a0457'), port: tr('m_e71ac32b544b'), target: tr('m_252e26071ac9'), flags: tr('m_269635727321'), tag: tr('m_1d0fd5f9336d'), value: tr('m_cda1d55c5231'),
  interval_sec: tr('m_45455dc4b517'), monitor_id: tr('m_b57b23459372'), target_id: tr('m_f621ef3eb10c'), url: tr('m_c935cb3a1ded'), description: tr('m_9ddde2b92952'), icon: tr('m_e220e37bbb93'), landing_enabled: tr('m_6a654dc0b0c0'),
  registration_enabled: tr('m_04f83ab519b2'), invite_mode: tr('m_936bac542e64'), experimental_features_enabled: tr('m_001a51a407db'), items: tr('m_4a385e41088a'), targets: tr('m_100d9e4baf2a'), publish_enabled: tr('m_4521b52c08c9'), alert_enabled: tr('m_6406d409c964'),
}))

export function changeAssistantDNSType(arguments_: Record<string, unknown>, type: string): Record<string, unknown> {
  const next: Record<string, unknown> = { ...arguments_, type }
  if (arguments_.type === type) return next
  delete next.data
  delete next.priority
  delete next.content
  if (!['A', 'AAAA', 'CNAME'].includes(type)) next.proxied = false
  if (type === 'SRV') next.data = { priority: 0, weight: 0, port: 0, target: '' }
  else if (type === 'CAA') next.data = { flags: 0, tag: 'issue', value: '' }
  else { next.content = ''; if (type === 'MX') next.priority = 0 }
  return next
}
