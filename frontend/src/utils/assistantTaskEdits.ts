export const assistantArgumentLabels: Record<string, string> = {
  id: '对象 ID', name: '名称', zone_id: '区域 ID', record_id: 'DNS 记录 ID', tunnel_id: '隧道 ID', hostname: '公开域名', service_url: '源站地址',
  type: '类型', content: '记录值', data: '记录数据（SRV / CAA）', ttl: 'TTL（1 为自动）', proxied: '代理', priority: '优先级', weight: '权重', port: '端口', target: '目标主机', flags: '标记', tag: '标签', value: '值',
  interval_sec: '间隔（秒）', monitor_id: '监控项目 ID', target_id: '探测目标 ID', url: '探测地址', description: '站点描述', icon: '站点图标', landing_enabled: '落地页',
  registration_enabled: '开放注册', invite_mode: '邀请码模式', experimental_features_enabled: '实验功能', items: '优选 CNAME 预设', targets: '探测目标', publish_enabled: '公开监控', alert_enabled: '监控告警',
}

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
