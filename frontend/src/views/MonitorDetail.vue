<template>
  <div class="page-container">
    <router-link to="/monitors" class="back-link">← 返回监控列表</router-link>

    <div class="page-header det-head">
      <div class="det-title">
        <h2>{{ monitor?.name || '加载中…' }}</h2>
        <span v-if="monitor" class="text-muted">每 {{ monitor.interval_sec }}s 自动检测</span>
      </div>
      <div v-if="monitor" class="det-actions">
        <span class="pub-chip" :class="{ on: monitor.publish_enabled }">
          {{ monitor.publish_enabled ? '公开页已开启' : '未公开' }}
        </span>
        <button class="btn btn-secondary btn-sm" @click="togglePublish">
          {{ monitor.publish_enabled ? '关闭公开页' : '开启公开页' }}
        </button>
        <button v-if="monitor.public_token && monitor.publish_enabled" class="btn btn-secondary btn-sm" @click="copyLink">复制公开链接</button>
        <button class="btn btn-primary btn-sm" :disabled="checking" @click="checkNow()">{{ checking ? '检测中…' : '立即检测' }}</button>
      </div>
    </div>

    <!-- 公开链接区 -->
    <div v-if="monitor?.publish_enabled && monitor.public_token" class="card pub-card">
      <span class="caption-mono">公开状态页</span>
      <code class="pub-url">{{ fullPublicUrl }}</code>
      <code v-if="monitor.public_domain" class="pub-url">https://{{ monitor.public_domain }}</code>
      <button class="show-toggle" @click="regenToken">重新生成令牌</button>
    </div>

    <!-- 公开页设置 -->
    <details v-if="monitor" class="card section pub-set">
      <summary>公开页设置<span class="ps-sub">标题与公告展示在公开状态页顶部</span></summary>
      <div class="ps-body">
        <label class="fld"><span>公开页顶部图标（图片链接 / 上传，可留空用默认）</span>
          <div class="icon-row">
            <img v-if="iconPreview" :src="pubIcon" class="icon-prev" alt="" />
            <input v-model="pubIcon" placeholder="https:// 图片地址，或点右侧上传" class="vercel-input icon-url" />
            <button type="button" class="btn btn-secondary btn-sm up-btn" :disabled="uploading" @click="pickIconFile">{{ uploading ? '上传中…' : '上传图片' }}</button>
            <input ref="iconFile" type="file" accept="image/png,image/jpeg,image/gif,image/webp" class="hidden-file" @change="onIconFile" />
          </div></label>
        <label class="fld"><span>显示主题</span>
          <select v-model="pubTheme" class="vercel-input theme-sel">
            <option value="">科技蓝（默认配色）</option>
            <option value="warm">暖米金</option>
          </select></label>
        <label class="fld"><span>自定义路径（留空用系统令牌）</span>
          <div class="icon-row">
            <code class="path-prefix">/status/</code>
            <input v-model.trim="pubSlug" maxlength="32" placeholder="例如：team" class="vercel-input icon-url" />
          </div>
          <p class="slug-hint text-muted">短链接预览：{{ slugPreview }}</p></label>
        <label class="fld"><span>自定义域名（留空则只用面板域名访问）</span>
          <input v-model.trim="pubDomain" maxlength="253" placeholder="例如：status.example.com" class="vercel-input" />
          <p class="slug-hint text-muted">
            填写后访问 <code class="inline-code">https://{{ pubDomain || 'status.example.com' }}</code> 会直接打开本状态页。保存时会自动配置同一 Cloudflare 连接下的 DNS 与隧道路由。
          </p></label>
        <div v-if="pubDomain" class="domain-guide">
          <span class="caption-mono">自动配置失败时，手动检查 DNS CNAME</span>
          <div class="copy-row">
            <code class="token-box">{{ dnsGuide }}</code>
            <button type="button" class="btn btn-secondary btn-sm" @click="copyText(dnsGuide)">复制</button>
          </div>
          <span class="caption-mono">手动检查 cloudflared ingress 规则</span>
          <div class="copy-row">
            <code class="token-box">{{ ingressGuide }}</code>
            <button type="button" class="btn btn-secondary btn-sm" @click="copyText(ingressGuide)">复制</button>
          </div>
          <p class="slug-hint text-muted">
            隧道匹配不到该主机名会直接返回 404，请求到不了面板。也可以改用通配主机名（如
            <code class="inline-code">*.{{ apexHint }}</code>）或把隧道末尾的兜底规则指向面板，这样以后新增域名无需再改隧道。
            面板若是 A 记录直连或走 nginx 反代，则自动配置不适用，需手动配置 DNS 与反向代理。
          </p>
        </div>
        <label class="fld"><span>公开页标题</span>
          <input v-model="pubTitle" maxlength="120" :placeholder="monitor.name" /></label>
        <label class="fld"><span>公告文本</span>
          <input v-model="announcement" maxlength="200" placeholder="例如：本周日凌晨 2:00-4:00 例行维护" /></label>
        <div class="ps-actions"><button class="btn btn-secondary btn-sm" @click="savePublishSettings">保存设置</button></div>
      </div>
    </details>

    <!-- 告警设置 -->
    <details v-if="monitor" class="card section pub-set">
      <summary>告警设置<span class="ps-sub">仅在服务状态变化时发送邮件通知</span></summary>
      <div class="ps-body">
        <label class="fld alert-switch row-between"><span>启用状态变化邮件告警</span><n-switch v-model:value="alertEnabled" size="small" /></label>
        <label class="fld"><span>收件邮箱（多个用英文逗号分隔，留空发送到注册邮箱）</span>
          <input v-model="alertEmails" placeholder="ops@example.com, boss@example.com" /></label>
        <div class="ps-actions"><button class="btn btn-secondary btn-sm" :disabled="savingAlert" @click="saveAlertSettings">{{ savingAlert ? '保存中…' : '保存告警设置' }}</button></div>
        <div v-if="alertLogs.length" class="alert-log">
          <span class="caption-mono">最近告警记录</span>
          <table class="alert-table">
            <thead><tr><th>时间</th><th>目标</th><th>状态</th><th>通知</th><th>说明</th></tr></thead>
            <tbody>
              <tr v-for="log in alertLogs" :key="log.id">
                <td>{{ new Date(log.created_at * 1000).toLocaleString() }}</td>
                <td>{{ log.target_name || log.target_id }}</td>
                <td>{{ log.state }}</td>
                <td>{{ log.notified ? '已发送' : '失败' }}</td>
                <td class="text-muted">{{ log.detail }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </details>

    <!-- 添加服务 -->
    <div class="card section add-card">
      <div class="add-row">
        <input v-model.trim="tName" class="vercel-input" placeholder="显示名称（选填）" />
        <select v-model="probeType" class="vercel-input type-sel"><option value="http">HTTP</option><option value="tcp">TCP 端口</option><option value="icmp">Ping (ICMP)</option></select>
        <select v-if="probeType === 'http'" v-model="probeMethod" class="vercel-input method-sel"><option value="GET">GET</option><option value="POST">POST</option></select>
        <label v-if="probeType === 'http'" class="link-chk"><n-switch v-model:value="linkEnabled" size="small" /> 状态页跳转</label>
        <input v-model.trim="tUrl" class="vercel-input url" :placeholder="urlHint" @keyup.enter="addTarget" />
        <button class="btn btn-primary btn-sm" :disabled="adding || !tUrl" @click="addTarget">{{ adding ? '添加中…' : '添加服务' }}</button>
      </div>
      <div v-if="routeHosts.length" class="route-pick">
        <span class="rp-label">从隧道路由导入：</span>
        <n-select v-model:value="routePick" size="small" placeholder="从路由导入…" class="rp-sel" :options="routeOptions" @update:value="onRoutePicked" />
      </div>
    </div>

    <!-- 服务列表 -->
    <div v-if="monitor && monitor.targets.length === 0" class="empty-note text-muted">还没有监控目标，先添加一个吧</div>
    <div v-else-if="monitor" class="card target-list">
      <div v-for="(t, i) in monitor.targets" :key="t.id + '-' + i" class="target-row">
        <span
          class="status-tag"
          :class="t.state === 'ok' ? 'healthy' : t.state === 'warn' ? 'degraded' : t.state === 'down' ? 'down' : ''"
        >
          {{ stateText(t.state) }}
        </span>
        <span v-if="t.type === 'tcp'" class="probe-badge">TCP</span>
        <span v-else-if="t.type === 'icmp'" class="probe-badge">PING</span>
        <span v-else-if="t.method === 'POST'" class="probe-badge post">POST</span>
        <div class="svc-main">
          <strong>{{ t.name || t.url }}</strong>
          <code>{{ t.url }}</code>
        </div>
        <div class="bars" aria-hidden="true">
          <i
            v-for="(b, bi) in t.bars || []"
            :key="bi"
            :class="'b-' + b.s"
            :title="barTitle(b)"
          ></i>
        </div>
        <div class="svc-side">
          <span class="latency" :class="latencyClass(t.latency_ms || 0)">{{ t.latency_ms != null ? t.latency_ms + 'ms' : '—' }}</span>
          <span class="uptime">24h {{ t.uptime_24h }}%</span>
        </div>
        <button class="row-edit" title="编辑该服务" @click="openEdit(t)">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"/></svg>
        </button>
        <button class="row-delete" title="移除该服务" @click="askRemove(t)">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      </div>
    </div>

    <!-- 编辑服务 -->
<n-modal v-model:show="editOpen" preset="card" title="编辑服务" class="edit-modal">
  <div class="edit-body">
    <label class="fld"><span>名称</span><input v-model="editName" class="vercel-input" /></label>
    <label class="fld"><span>探测方式</span>
      <select v-model="editType" class="vercel-input"><option value="http">HTTP</option><option value="tcp">TCP 端口</option><option value="icmp">Ping (ICMP)</option></select></label>
    <label class="fld"><span>地址</span><input v-model="editUrl" class="vercel-input" :placeholder='probeType === "tcp" ? "example.com:443" : probeType === "icmp" ? "example.com 或 1.1.1.1" : "https://example.com"' /></label>
    <label v-if="editType === 'http'" class="fld"><span>HTTP 方法</span><select v-model="editMethod" class="vercel-input"><option value="GET">GET</option><option value="POST">POST</option></select></label>
    <label v-if="editType === 'http'" class="fld row-between"><span>状态页跳转</span><n-switch v-model:value="editLink" size="small" /></label>
    <p class="edit-note text-muted">修改地址、类型或方法会清空该服务的历史记录并重新探测。</p>
  </div>
  <template #footer>
    <div class="ps-actions">
      <button class="btn btn-secondary btn-sm" @click="editOpen = false">取消</button>
      <button class="btn btn-primary btn-sm" :disabled="savingEdit || !editName || !editUrl" @click="saveEdit">{{ savingEdit ? '保存中…' : '保存修改' }}</button>
    </div>
  </template>
</n-modal>
<!-- 移除确认 -->
    <Teleport to="body">
      <div v-if="removing" class="dlg-mask" @click.self="removing = null">
        <div class="dlg">
          <h3>移除监控目标</h3>
          <p class="dlg-desc">将停止对以下地址的监测并清理其历史记录：</p>
          <code class="dlg-code">{{ removing.url }}</code>
          <div class="dlg-actions">
            <button class="btn btn-secondary btn-sm" :disabled="busy" @click="removing = null">取消</button>
            <button class="btn btn-danger btn-sm" :disabled="busy" @click="confirmRemove">{{ busy ? '移除中…' : '确认移除' }}</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { NModal, NSelect, NSwitch, useMessage } from 'naive-ui'
import { useConfigStore } from '../stores/config'
import {
  addMonitorTarget,
  editMonitorTarget,
  getTunnelDetail,
  uploadImage,
  checkMonitorNow,
  getMonitor,
  publicStatusPath,
  removeMonitorTarget,
  updateMonitor,
  listMonitorAlerts,
  type AlertLog,
  type Heartbeat,
  type MonitorView,
  type TargetStatus,
} from '../api'

const route = useRoute()
const message = useMessage()
const configStore = useConfigStore()
const monitorId = route.params.id as string

const monitor = ref<MonitorView | null>(null)
const loading = ref(true)
const checking = ref(false)
const adding = ref(false)

const tName = ref('')
const tUrl = ref('')
const probeType = ref<'http' | 'tcp' | 'icmp'>('http')
const probeMethod = ref<'GET' | 'POST'>('GET')
const linkEnabled = ref(false)
const routeHosts = ref<string[]>([])
const routePick = ref<string | null>(null)
const pubTitle = ref('')
const pubIcon = ref('')
const pubTheme = ref<'' | 'warm'>('')
const pubSlug = ref('')
const pubDomain = ref('')
const announcement = ref('')

const editOpen = ref(false)
const editId = ref('')
const editName = ref('')
const editUrl = ref('')
const editType = ref<'http' | 'tcp' | 'icmp'>('http')
const editMethod = ref<'GET' | 'POST'>('GET')
const editLink = ref(false)
const savingEdit = ref(false)
const uploading = ref(false)
const iconFile = ref<HTMLInputElement | null>(null)

const routeOptions = computed(() => routeHosts.value.map((h) => ({ label: h, value: h })))
const slugPreview = computed(() => {
  const seg = pubSlug.value.trim().toLowerCase() || monitor.value?.public_token || '<令牌>'
  return '/status/' + seg
})

const iconPreview = computed(() => /^(https?:\/\/|\/uploads\/|data:image\/)/.test(pubIcon.value.trim()))

// Everything below the first label of the custom domain, used to suggest a
// wildcard ingress hostname.
const apexHint = computed(() => pubDomain.value.split('.').slice(1).join('.') || 'example.com')

const dnsGuide = computed(() => {
  const tunnel = configStore.config.tunnel_id
  return [
    `类型:   CNAME`,
    `名称:   ${pubDomain.value}`,
    `目标:   ${tunnel ? tunnel + '.cfargotunnel.com' : '<面板所在隧道的 ID>.cfargotunnel.com'}`,
    `代理:   已开启（橙云）`,
  ].join('\n')
})

const ingressGuide = computed(() => {
  const service = configStore.config.service_url
  return [
    `主机名: ${pubDomain.value}`,
    `服务:   ${service || '<面板的本机地址，例如 http://localhost:8080>'}`,
  ].join('\n')
})
const urlHint = computed(() =>
  probeType.value === 'tcp' ? '地址，例如 example.com:443 或 10.0.0.2:22' :
  probeType.value === 'icmp' ? '主机或 IP，例如 example.com 或 1.1.1.1' :
  '服务地址，例如 https://example.com 或 example.com')

const removing = ref<TargetStatus | null>(null)
const busy = ref(false)

let timer: ReturnType<typeof setInterval> | undefined

const fullPublicUrl = computed(() =>
  monitor.value?.publish_enabled && (monitor.value.public_slug || monitor.value.public_token) ? location.origin + publicStatusPath(monitor.value.public_slug || monitor.value.public_token || '') : ''
)

async function load() {
  try {
    const { data } = await getMonitor(monitorId)
    monitor.value = data
    document.title = (data.name || '监控') + ' · 状态'
    pubTitle.value = data.public_title || ''
    pubIcon.value = data.public_icon || ''
    pubTheme.value = data.public_theme === 'warm' ? 'warm' : ''
    pubSlug.value = data.public_slug || ''
    pubDomain.value = data.public_domain || ''
    announcement.value = data.announcement || ''
    alertEnabled.value = Boolean(data.alert_enabled)
    alertEmails.value = data.alert_emails || ''
    void loadAlertLogs()
  } finally {
    loading.value = false
  }
}

function stateText(s?: string) {
  return s === 'ok' ? '正常' : s === 'warn' ? '异常' : s === 'down' ? '不可达' : '待检测'
}

function latencyClass(ms: number) {
  if (!ms) return ''
  if (ms <= 300) return ''
  if (ms <= 1000) return 'mid'
  return 'slow'
}

function barTitle(b: Heartbeat) {
  const d = new Date(b.t)
  const time = isNaN(d.getTime()) ? '' : d.toLocaleString() + ' · '
  return time + (b.s === 'ok' ? '正常' : b.s === 'warn' ? '异常' : '不可达') + (b.ms != null ? ' · ' + b.ms + 'ms' : '')
}

async function addTarget() {
  if (!tUrl.value || adding.value) return
  adding.value = true
  try {
    const { data } = await addMonitorTarget(monitorId, tName.value, tUrl.value, probeType.value, probeMethod.value, linkEnabled.value)
    monitor.value = data
    message.success('已添加，马上开始首次检测')
    tName.value = ''
    tUrl.value = ''
    void checkNow(true)
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    message.error(msg || '添加失败')
  } finally {
    adding.value = false
  }
}

function onRoutePicked(v: string | null) {
  if (v) pickRoute(v)
  routePick.value = null
}

function pickRoute(h: string) {
  if (!tName.value) tName.value = h.split('.')[0]
  if (probeType.value === 'tcp') {
    tUrl.value = h.includes(':') ? h : h + ':443'
  } else if (probeType.value === 'icmp') {
    tUrl.value = h
  } else {
    tUrl.value = 'https://' + h
  }
}

async function loadRoutes() {
  try {
    const tid = configStore.config.tunnel_id
    if (!tid) return
    const { data } = await getTunnelDetail(tid)
    const hosts = (data.ingress || [])
      .map((r) => (r.hostname || '').trim())
      .filter((h) => h && h !== '*' && !h.startsWith('*'))
    routeHosts.value = Array.from(new Set(hosts))
  } catch (_) {
    // 隧道路由读取失败不阻塞主流程
  }
}

const alertEnabled = ref(false)
const alertEmails = ref('')
const savingAlert = ref(false)
const alertLogs = ref<AlertLog[]>([])

async function saveAlertSettings() {
  if (savingAlert.value) return
  savingAlert.value = true
  try {
    const { data } = await updateMonitor(monitorId, {
      alert_enabled: alertEnabled.value,
      alert_emails: alertEmails.value.trim(),
    })
    monitor.value = data
    message.success(alertEnabled.value ? '告警已开启' : '告警已关闭')
  } catch (_) {
    message.error('保存失败')
  } finally {
    savingAlert.value = false
  }
}

async function loadAlertLogs() {
  try {
    const { data } = await listMonitorAlerts(monitorId)
    alertLogs.value = data || []
  } catch (_) {
    alertLogs.value = []
  }
}

async function savePublishSettings() {
  try {
    const { data } = await updateMonitor(monitorId, {
      public_title: pubTitle.value.trim(),
      public_icon: pubIcon.value.trim(),
      public_theme: pubTheme.value,
      public_slug: pubSlug.value.toLowerCase(),
      public_domain: pubDomain.value.toLowerCase(),
      announcement: announcement.value.trim(),
    })
    monitor.value = data
    if (data.domain_warning) {
      message.warning(`设置已保存，但自动配置失败：${data.domain_warning}`)
    } else if (data.public_domain) {
      message.success('公开页设置已保存，DNS 与隧道路由已自动配置')
    } else {
      message.success('公开页设置已保存')
    }
  } catch (_) {
    message.error('保存失败')
  }
}

function openEdit(t: TargetStatus) {
  editId.value = t.id
  editName.value = t.name
  editUrl.value = t.url
  editType.value = t.type || 'http'
  editMethod.value = (t.method as 'GET' | 'POST') || 'GET'
  editLink.value = !!t.link_enabled
  editOpen.value = true
}

async function saveEdit() {
  if (!monitor.value) return
  savingEdit.value = true
  try {
    const { data } = await editMonitorTarget(monitor.value.id, editId.value, {
      name: editName.value.trim(),
      url: editUrl.value.trim(),
      type: editType.value,
      method: editMethod.value,
      link_enabled: editLink.value,
    })
    monitor.value = data
    message.success('已保存，正在重新探测')
    editOpen.value = false
  } catch (_) {
    message.error('保存失败')
  } finally {
    savingEdit.value = false
  }
}

function pickIconFile() {
  iconFile.value?.click()
}

async function onIconFile(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  uploading.value = true
  try {
    const { data } = await uploadImage(file)
    pubIcon.value = data.url
    message.success('图片已上传，别忘了保存设置')
  } catch (_) {
    message.error('上传失败（限 png/jpg/gif/webp，4MB 内）')
  } finally {
    uploading.value = false
  }
}

function askRemove(t: TargetStatus) {
  removing.value = t
}

async function confirmRemove() {
  if (!removing.value || busy.value) return
  busy.value = true
  try {
    await removeMonitorTarget(monitorId, removing.value.id)
    message.success('已移除')
    removing.value = null
    await load()
  } catch (_) {
    message.error('移除失败')
  } finally {
    busy.value = false
  }
}

async function checkNow(silent = false) {
  if (!silent) checking.value = true
  try {
    const { data } = await checkMonitorNow(monitorId)
    applyOutcomes(data.outcomes || [])
    if (!silent) message.success('检测完成')
  } catch (_) {
    if (!silent) message.error('检测失败，请稍后重试')
  } finally {
    checking.value = false
  }
}

function applyOutcomes(outcomes: Array<{ target_id: string; state: string; latency_ms: number; http_code?: number; error?: string }>) {
  if (!monitor.value) return
  for (const o of outcomes) {
    const t = monitor.value.targets.find((x) => x.id === o.target_id)
    if (!t) continue
    t.state = o.state as TargetStatus['state']
    t.latency_ms = o.latency_ms
    t.http_code = o.http_code
    t.error = o.error
  }
}

async function togglePublish() {
  if (!monitor.value) return
  try {
    const { data } = await updateMonitor(monitorId, { publish_enabled: !monitor.value.publish_enabled })
    monitor.value = data
    message.success(data.publish_enabled ? '公开页已开启' : '公开页已关闭')
  } catch (_) {
    message.error('操作失败')
  }
}

async function regenToken() {
  try {
    await updateMonitor(monitorId, { regenerate_token: true })
    await load()
    message.success('新令牌已生成，旧链接即刻失效')
  } catch (_) {
    message.error('生成失败')
  }
}

function copyLink() {
  navigator.clipboard.writeText(fullPublicUrl.value)
    .then(() => message.success('公开链接已复制'))
    .catch(() => message.error('复制失败'))
}

function copyText(text: string) {
  navigator.clipboard.writeText(text)
    .then(() => message.success('已复制'))
    .catch(() => message.warning('复制失败，请手动选择文本'))
}

onMounted(() => {
  load()
  void loadRoutes()
  timer = setInterval(() => { void load() }, 30000)
})
onBeforeUnmount(() => { if (timer) clearInterval(timer) })
</script>

<style scoped>
.back-link { display: inline-block; margin-bottom: 10px; font-size: 12.5px; color: var(--color-body); }
.back-link:hover { color: var(--color-ink); }
.det-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 16px; flex-wrap: wrap; }
.det-title h2 { margin: 0 0 4px; }
.pub-chip { font-size: 11px; font-weight: 600; padding: 3px 9px; border-radius: 999px;
  background: var(--color-canvas-soft); border: 1px solid var(--color-hairline); color: var(--color-mute); }
.pub-chip.on { color: var(--color-success); border-color: var(--color-success); background: color-mix(in srgb, var(--color-success) 8%, transparent); }
.det-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }

.pub-card { display: flex; align-items: center; gap: 12px; padding: 12px 16px; margin-bottom: 14px; overflow: hidden; }
.pub-url { flex: 1; min-width: 0; font-family: var(--font-mono); font-size: 11.5px; color: var(--color-body);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

.add-card { margin-bottom: 14px; }
.add-row { display: grid; grid-template-columns: 180px auto auto 1fr auto; gap: 8px; }
.type-sel { width: 118px; }
.method-sel { width: 92px; }
.link-chk {
  display: inline-flex; align-items: center; gap: 7px; font-size: 12px; color: var(--color-body);
  cursor: pointer; user-select: none; white-space: nowrap;
}
.rp-sel { min-width: 200px; max-width: 260px; }
.row-edit {
  display: inline-flex; width: 28px; height: 28px; align-items: center; justify-content: center;
  color: var(--color-mute); background: transparent; border: 1px solid transparent; border-radius: 6px;
  cursor: pointer; transition: color 120ms ease, background 120ms ease, border-color 120ms ease;
}
.row-edit:hover { color: var(--color-ink); background: var(--color-canvas-soft); border-color: var(--color-hairline); }
.icon-row { display: flex; align-items: center; gap: 8px; }
.icon-prev {
  width: 34px; height: 34px; object-fit: contain; border-radius: 8px;
  background: var(--color-canvas-soft); border: 1px solid var(--color-hairline); padding: 3px; flex-shrink: 0;
}
.icon-url { flex: 1 !important; }
.hidden-file { display: none; }
.theme-sel { max-width: 320px; }
.path-prefix {
  display: inline-flex; align-items: center; height: 34px; padding: 0 8px; flex-shrink: 0;
  font-family: var(--font-mono); font-size: 12px; color: var(--color-mute);
  background: var(--color-canvas-soft); border: 1px solid var(--color-hairline); border-radius: 6px;
}
.slug-hint { margin: 2px 0 0; font-size: 11px; }
.domain-guide { display: flex; flex-direction: column; gap: 6px; padding: 12px; background: var(--color-canvas-soft); border: 1px solid var(--color-hairline); border-radius: var(--radius-md); }
.copy-row { display: flex; align-items: flex-start; gap: 8px; }
.token-box { flex: 1; min-width: 0; padding: 10px 12px; color: var(--color-ink); background: var(--color-canvas); border: 1px solid var(--color-hairline); border-radius: var(--radius-md); font-family: var(--font-mono); font-size: 12px; line-height: 1.6; white-space: pre-wrap; word-break: break-all; }
.edit-modal { width: min(460px, calc(100vw - 32px)); }
.edit-body { display: flex; flex-direction: column; gap: 14px; }
.edit-note { margin: 0; font-size: 11.5px; line-height: 1.5; }

.route-pick { display: flex; align-items: center; flex-wrap: wrap; gap: 6px; margin-top: 10px; }
.rp-label { font-size: 12px; color: var(--color-mute); margin-right: 2px; }
.rp-chip {
  font-family: var(--font-mono); font-size: 11px; padding: 3px 9px; border-radius: 999px;
  border: 1px solid var(--color-hairline-strong); background: var(--color-canvas-soft);
  color: var(--color-body); cursor: pointer; transition: border-color 120ms ease, color 120ms ease;
}
.rp-chip:hover { border-color: var(--color-focus); color: var(--color-ink); }

.probe-badge {
  font-size: 10px; font-weight: 700; letter-spacing: .04em; flex-shrink: 0;
  padding: 2px 7px; border-radius: 5px; background: var(--color-canvas-soft);
  border: 1px solid var(--color-hairline); color: var(--color-body);
}
.probe-badge.post { color: var(--color-link); border-color: var(--color-hairline-strong); }

.pub-set summary {
  cursor: pointer; list-style: none; display: flex; align-items: baseline; gap: 10px;
  padding: 13px 16px; font-size: 13.5px; font-weight: 600; color: var(--color-ink);
}
.pub-set summary::-webkit-details-marker { display: none; }
.pub-set .ps-sub { font-size: 11.5px; font-weight: 400; color: var(--color-mute); }
.pub-set .ps-body { padding: 4px 16px 16px; display: flex; flex-direction: column; }
.pub-set .fld { display: flex; flex-direction: column; gap: 6px; margin-bottom: 12px; font-size: 12.5px; color: var(--color-body); }
.pub-set .fld input {
  height: 34px; border-radius: 6px; border: 1px solid var(--color-hairline-strong);
  background: var(--color-canvas-raised); color: var(--color-ink); padding: 0 10px; outline: none; font-size: 13px;
}
.pub-set .fld input:focus { border-color: var(--color-focus); }
.ps-actions { display: flex; justify-content: flex-end; }
.vercel-input { height: 34px; border-radius: 6px; border: 1px solid var(--color-hairline-strong);
  background: var(--color-canvas-raised); color: var(--color-ink); padding: 0 10px; font-size: 13px; outline: none; }
.vercel-input:focus { border-color: var(--color-focus); }
.add-row .url { font-family: var(--font-mono); font-size: 12px; }

.target-list { overflow: hidden; }
.target-row { display: flex; align-items: center; gap: 14px; padding: 13px 16px; border-bottom: 1px solid var(--color-hairline); background: var(--color-canvas-raised); }
.target-row:last-child { border-bottom: 0; }
.svc-main { min-width: 0; width: 260px; display: flex; flex-direction: column; gap: 3px; }
.svc-main strong { font-size: 13.5px; color: var(--color-ink); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.svc-main code { font-family: var(--font-mono); font-size: 11px; color: var(--color-mute); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.bars { flex: 1; min-width: 120px; display: flex; align-items: stretch; gap: 2px; height: 22px; }
.bars i { flex: 1; border-radius: 2px; background: var(--color-canvas-soft-2); }
.bars i.b-ok { background: var(--color-success); opacity: .85; }
.bars i.b-warn { background: var(--color-warning); }
.bars i.b-down { background: var(--color-error); }

.svc-side { display: flex; align-items: center; gap: 12px; flex-shrink: 0; }
.latency { font-family: var(--font-mono); font-size: 11px; font-weight: 700; min-width: 46px; text-align: right; }
.latency.mid { color: var(--color-warning); }
.latency.slow { color: var(--color-error); }
.uptime { font-size: 11.5px; color: var(--color-body); min-width: 62px; }

.empty-note { padding: 40px; text-align: center; }
.row-delete {
  display: inline-flex; align-items: center; justify-content: center; width: 26px; height: 26px; flex-shrink: 0;
  border-radius: 6px; border: 1px solid transparent; background: transparent; color: var(--color-mute); cursor: pointer;
  transition: color 120ms ease, background-color 120ms ease, border-color 120ms ease;
}
.row-delete:hover:not(:disabled) { color: var(--color-error); background: var(--color-status-down-bg); border-color: var(--color-status-down-border); }
.dlg-mask { position: fixed; inset: 0; z-index: 2000; display: grid; place-items: center; padding: 16px; background: rgba(8, 10, 18, .45); }
.dlg { width: min(420px, 100%); background: var(--color-canvas-raised); border: 1px solid var(--color-hairline);
  border-radius: 10px; padding: 20px; box-shadow: 0 16px 48px rgba(0, 0, 0, .22); }
.dlg h3 { margin: 0 0 8px; font-size: 15px; color: var(--color-ink); }
.dlg-desc { margin: 0 0 10px; font-size: 13px; line-height: 1.6; color: var(--color-body); }
.dlg-code { display: block; padding: 8px 10px; margin-bottom: 14px; border-radius: 6px; background: var(--color-canvas-soft);
  border: 1px solid var(--color-hairline); font-family: var(--font-mono); font-size: 11.5px; color: var(--color-ink); word-break: break-all; }
.dlg-actions { display: flex; justify-content: flex-end; gap: 8px; }
@media (max-width: 900px) {
  .add-row { grid-template-columns: 1fr; }
  .bars { order: 5; flex-basis: 100%; }
  .target-row { flex-wrap: wrap; row-gap: 8px; }
}

.show-toggle {
	height: 28px; padding: 0 10px; font-size: 12px; white-space: nowrap;
	border-radius: 6px; border: 1px solid var(--color-hairline-strong); background: transparent; color: var(--color-body); cursor: pointer;
	transition: border-color 120ms ease, color 120ms ease, background-color 120ms ease;
}
.show-toggle:hover { border-color: var(--color-focus); color: var(--color-ink); }
</style>
