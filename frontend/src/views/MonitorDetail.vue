<template>
  <div class="page-container">
    <router-link to="/monitors" class="back-link">{{ tr('m_dfc19cac6dd8') }}</router-link>

    <div class="page-header det-head">
      <div class="det-title">
        <h2>{{ monitor?.name || tr('m_4927a53bcc88') }}</h2>
        <span v-if="monitor" class="text-muted">{{ tr('m_d8b82d5765fa') }} {{ monitor.interval_sec }}{{ tr('m_48ba2cb7a48d') }}</span>
      </div>
      <div v-if="monitor" class="det-actions">
        <span class="pub-chip" :class="{ on: monitor.publish_enabled }">
          {{ monitor.publish_enabled ? tr('m_9f1e25609b1a') : tr('m_abcd7f608563') }}
        </span>
        <button class="btn btn-secondary btn-sm" @click="togglePublish">
          {{ monitor.publish_enabled ? tr('m_6327d765e2fd') : tr('m_e2c3d2906783') }}
        </button>
        <button v-if="monitor.public_token && monitor.publish_enabled" class="btn btn-secondary btn-sm" @click="copyLink">{{ tr('m_2596d7de6c7b') }}</button>
        <button class="btn btn-primary btn-sm" :disabled="checking" @click="checkNow()">{{ checking ? tr('m_3a1abf3422d2') : tr('m_14ca09ce1fd2') }}</button>
      </div>
    </div>

    <!-- 公开链接区 -->
    <div v-if="monitor?.publish_enabled && monitor.public_token" class="card pub-card">
      <span class="caption-mono">{{ tr('m_6e3331e264ce') }}</span>
      <code class="pub-url">{{ fullPublicUrl }}</code>
      <code v-if="monitor.public_domain" class="pub-url">https://{{ monitor.public_domain }}</code>
      <button class="show-toggle" @click="regenToken">{{ tr('m_03fb8dc8b53e') }}</button>
    </div>

    <!-- 公开页设置 -->
    <details v-if="monitor" class="card section pub-set">
      <summary>{{ tr('m_c3d82c9ce085') }}<span class="ps-sub">{{ tr('m_ca7b7db374d1') }}</span></summary>
      <div class="ps-body">
        <label class="fld"><span>{{ tr('m_6787f82d0c1e') }}</span>
          <div class="icon-row">
            <img v-if="iconPreview" :src="pubIcon" class="icon-prev" alt="" />
            <input v-model="pubIcon" :placeholder="tr('m_2f32e0f1eccc')" class="vercel-input icon-url" />
            <button type="button" class="btn btn-secondary btn-sm up-btn" :disabled="uploading" @click="pickIconFile">{{ uploading ? tr('m_ce1c9d61565f') : tr('m_9fbb4750bb39') }}</button>
            <input ref="iconFile" type="file" accept="image/png,image/jpeg,image/gif,image/webp" class="hidden-file" @change="onIconFile" />
          </div></label>
        <label class="fld"><span>{{ tr('m_0c3421deb8a7') }}</span>
          <select v-model="pubTheme" class="vercel-input theme-sel">
            <option value="">{{ tr('m_5e938f31fea4') }}</option>
            <option value="warm">{{ tr('m_78e143c20da6') }}</option>
          </select></label>
        <label class="fld"><span>{{ tr('m_0c1da4d3cd29') }}</span>
          <div class="icon-row">
            <code class="path-prefix">/status/</code>
            <input v-model.trim="pubSlug" maxlength="32" :placeholder="tr('m_ad293852688f')" class="vercel-input icon-url" />
          </div>
          <p class="slug-hint text-muted">{{ tr('m_a9abfdabe0d6') }}{{ slugPreview }}</p></label>
        <label class="fld"><span>{{ tr('m_ef8506a52a38') }}</span>
          <input v-model.trim="pubDomain" maxlength="253" :placeholder="tr('m_f8edce7be00e')" class="vercel-input" />
          <p class="slug-hint text-muted">
            {{ tr('m_4bad264a232a') }} <code class="inline-code">https://{{ pubDomain || 'status.example.com' }}</code> {{ tr('m_caefaccadda8') }}
            {{ pubDomainMode === 'simple' ? tr('m_17f020263f9c') : tr('m_d68cfb5fdd0e') }}
          </p></label>
        <div class="fld domain-mode-field">
          <span>{{ tr('m_9fe789442ad9') }}</span>
          <div class="status-mode-selector" role="radiogroup" :aria-label="tr('m_ae6511ef6bc9')">
            <button type="button" class="status-mode-option" :class="{ active: pubDomainMode === 'simple' }" role="radio" :aria-checked="pubDomainMode === 'simple'" @click="pubDomainMode = 'simple'">
              <strong>{{ tr('m_b06325c5660f') }}</strong><span>{{ tr('m_ac0b8ee0bf7f') }}</span>
            </button>
            <button type="button" class="status-mode-option" :class="{ active: pubDomainMode === 'preferred' }" role="radio" :aria-checked="pubDomainMode === 'preferred'" @click="pubDomainMode = 'preferred'">
              <strong>{{ tr('m_34e9234df233') }}</strong><span>{{ tr('m_93d8a08cba49') }}</span>
            </button>
          </div>
        </div>
        <div v-if="pubDomainMode === 'preferred'" class="preferred-domain-fields">
          <label class="fld preferred-domain-field"><span>{{ tr('m_ceb8422bcb9d') }}</span>
            <input v-model.trim="pubAuxDomain" maxlength="253" :placeholder="tr('m_facc063f438f')" class="vercel-input" />
            <p class="slug-hint text-muted">{{ tr('m_d60a71f83845') }}</p>
          </label>
          <label class="fld preferred-domain-field"><span>{{ tr('m_309710e63b6d') }}</span>
            <cname-picker
              v-model="pubPreferredCNAME"
              class="status-cname-picker"
              :presets="configStore.config.cname_presets"
              :show-default="true"
              :default-value="configStore.config.preferred_cname"
              :placeholder="tr('m_817c0185fe75')"
            />
            <p class="slug-hint text-muted">{{ tr('m_5a4d6bfe1c64') }}{{ configStore.config.preferred_cname || tr('m_80a57e03f071') }}</p>
          </label>
        </div>
        <div v-if="pubDomain" class="domain-guide">
          <span class="caption-mono">{{ tr('m_b8be77b7fb5f') }}</span>
          <div class="copy-row">
            <code class="token-box">{{ dnsGuide }}</code>
            <button type="button" class="btn btn-secondary btn-sm" @click="copyText(dnsGuide)">{{ tr('m_63d90d977348') }}</button>
          </div>
          <span class="caption-mono">{{ tr('m_2b5207563079') }}</span>
          <div class="copy-row">
            <code class="token-box">{{ saasGuide }}</code>
            <button type="button" class="btn btn-secondary btn-sm" @click="copyText(saasGuide)">{{ tr('m_63d90d977348') }}</button>
          </div>
          <span class="caption-mono">{{ tr('m_bb4a26ca3cca') }}</span>
          <div class="copy-row">
            <code class="token-box">{{ ingressGuide }}</code>
            <button type="button" class="btn btn-secondary btn-sm" @click="copyText(ingressGuide)">{{ tr('m_63d90d977348') }}</button>
          </div>
          <p v-if="pubDomainMode === 'preferred'" class="slug-hint text-muted">
            {{ tr('m_f7e03718bee2') }} <code class="inline-code">httpHostHeader</code>{{ tr('m_78a3a2b2c3e8') }}
          </p>
          <p v-else class="slug-hint text-muted">
            {{ tr('m_1a1a1f926cba') }} <code class="inline-code">httpHostHeader</code>。
          </p>
        </div>
        <label class="fld"><span>{{ tr('m_5c7cd4454796') }}</span>
          <input v-model="pubTitle" maxlength="120" :placeholder="monitor.name" /></label>
        <label class="fld"><span>{{ tr('m_1a124340fa56') }}</span>
          <input v-model="announcement" maxlength="200" :placeholder="tr('m_fda4c67aae87')" /></label>
        <div class="ps-actions"><button class="btn btn-secondary btn-sm" @click="savePublishSettings">{{ tr('m_c8550237ba70') }}</button></div>
      </div>
    </details>

    <!-- 告警设置 -->
    <details v-if="monitor" class="card section pub-set">
      <summary>{{ tr('m_41cd995272ce') }}<span class="ps-sub">{{ tr('m_c5b1608fa825') }}</span></summary>
      <div class="ps-body">
        <label class="fld alert-switch row-between"><span>{{ tr('m_8e14f7e4b3be') }}</span><n-switch v-model:value="alertEnabled" size="small" /></label>
        <label class="fld"><span>{{ tr('m_08b01e987e4e') }}</span>
          <input v-model="alertEmails" placeholder="ops@example.com, boss@example.com" /></label>
        <div class="ps-actions"><button class="btn btn-secondary btn-sm" :disabled="savingAlert" @click="saveAlertSettings">{{ savingAlert ? tr('m_ff509c9ba052') : tr('m_cc95d97c64a8') }}</button></div>
        <div v-if="alertLogs.length" class="alert-log">
          <span class="caption-mono">{{ tr('m_199edd1a1d6f') }}</span>
          <table class="alert-table">
            <thead><tr><th>{{ tr('m_8b6ff498515b') }}</th><th>{{ tr('m_57060c88a36b') }}</th><th>{{ tr('m_6320b4a8722a') }}</th><th>{{ tr('m_4b6ded8a7f18') }}</th><th>{{ tr('m_4262c45dc797') }}</th></tr></thead>
            <tbody>
              <tr v-for="log in alertLogs" :key="log.id">
                <td>{{ new Date(log.created_at * 1000).toLocaleString(locale) }}</td>
                <td>{{ log.target_name || log.target_id }}</td>
                <td>{{ log.state }}</td>
                <td>{{ log.notified ? tr('m_60823aaec73b') : tr('m_28384d7afd2e') }}</td>
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
        <input v-model.trim="tName" class="vercel-input" :placeholder="tr('m_8636d6879dff')" />
        <select v-model="probeType" class="vercel-input type-sel"><option value="http">HTTP</option><option value="tcp">{{ tr('m_3f54e7dbbb3e') }}</option><option value="icmp">Ping (ICMP)</option></select>
        <select v-if="probeType === 'http'" v-model="probeMethod" class="vercel-input method-sel"><option value="GET">GET</option><option value="POST">POST</option></select>
        <label v-if="probeType === 'http'" class="link-chk"><n-switch v-model:value="linkEnabled" size="small" /> {{ tr('m_bee2d0d2d313') }}</label>
        <input v-model.trim="tUrl" class="vercel-input url" :placeholder="urlHint" @keyup.enter="addTarget" />
        <button class="btn btn-primary btn-sm" :disabled="adding || !tUrl" @click="addTarget">{{ adding ? tr('m_f470193bcdb6') : tr('m_6f760cbd09bb') }}</button>
      </div>
      <div v-if="routeHosts.length" class="route-pick">
        <span class="rp-label">{{ tr('m_2063c5f171ad') }}</span>
        <n-select v-model:value="routePick" size="small" :placeholder="tr('m_f95348371b73')" class="rp-sel" :options="routeOptions" @update:value="onRoutePicked" />
      </div>
    </div>

    <!-- 服务列表 -->
    <div v-if="monitor && monitor.targets.length === 0" class="empty-note text-muted">{{ tr('m_e856d7de8100') }}</div>
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
        <button class="row-edit" :title="tr('m_55895abae0c5')" @click="openEdit(t)">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"/></svg>
        </button>
        <button class="row-delete" :title="tr('m_73869310cfc3')" @click="askRemove(t)">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      </div>
    </div>

    <!-- 编辑服务 -->
<n-modal v-model:show="editOpen" preset="card" :title="tr('m_a1944feec290')" class="edit-modal">
  <div class="edit-body">
    <label class="fld"><span>{{ tr('m_d44e9b3d3b31') }}</span><input v-model="editName" class="vercel-input" /></label>
    <label class="fld"><span>{{ tr('m_86b4208b5178') }}</span>
      <select v-model="editType" class="vercel-input"><option value="http">HTTP</option><option value="tcp">{{ tr('m_3f54e7dbbb3e') }}</option><option value="icmp">Ping (ICMP)</option></select></label>
    <label class="fld"><span>{{ tr('m_37aac5b4e639') }}</span><input v-model="editUrl" class="vercel-input" :placeholder="editType === 'tcp' ? 'example.com:443' : editType === 'icmp' ? tr('m_b2a95d595694') : 'https://example.com'" /></label>
    <label v-if="editType === 'http'" class="fld"><span>{{ tr('m_af1029a24f14') }}</span><select v-model="editMethod" class="vercel-input"><option value="GET">GET</option><option value="POST">POST</option></select></label>
    <label v-if="editType === 'http'" class="fld row-between"><span>{{ tr('m_bee2d0d2d313') }}</span><n-switch v-model:value="editLink" size="small" /></label>
    <p class="edit-note text-muted">{{ tr('m_a317f2fd5bae') }}</p>
  </div>
  <template #footer>
    <div class="ps-actions">
      <button class="btn btn-secondary btn-sm" @click="editOpen = false">{{ tr('m_2cd0f3be8738') }}</button>
      <button class="btn btn-primary btn-sm" :disabled="savingEdit || !editName || !editUrl" @click="saveEdit">{{ savingEdit ? tr('m_ff509c9ba052') : tr('m_991bb7cfe5a8') }}</button>
    </div>
  </template>
</n-modal>
<!-- 移除确认 -->
    <Teleport to="body">
      <div v-if="removing" class="dlg-mask" @click.self="removing = null">
        <div class="dlg">
          <h3>{{ tr('m_2aa8a423ad8d') }}</h3>
          <p class="dlg-desc">{{ tr('m_a68874fc731c') }}</p>
          <code class="dlg-code">{{ removing.url }}</code>
          <div class="dlg-actions">
            <button class="btn btn-secondary btn-sm" :disabled="busy" @click="removing = null">{{ tr('m_2cd0f3be8738') }}</button>
            <button class="btn btn-danger btn-sm" :disabled="busy" @click="confirmRemove">{{ busy ? tr('m_19c4ca651c6d') : tr('m_ee7dea070d88') }}</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { locale, t as tr, localized } from '../i18n'

import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { NModal, NSelect, NSwitch, useMessage } from 'naive-ui'
import { useConfigStore } from '../stores/config'
import CnamePicker from '../components/CNAMEPicker.vue'
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
  type BindMode,
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
const pubDomainMode = ref<BindMode>('preferred')
const pubAuxDomain = ref('')
const pubPreferredCNAME = ref('')
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
  const seg = pubSlug.value.trim().toLowerCase() || monitor.value?.public_token || tr('m_60eec4a66f92')
  return '/status/' + seg
})

const iconPreview = computed(() => /^(https?:\/\/|\/uploads\/|data:image\/)/.test(pubIcon.value.trim()))

const dnsGuide = computed(() => {
  const tunnelID = configStore.config.tunnel_id
  const tunnelTarget = tunnelID ? tunnelID + '.cfargotunnel.com' : tr('m_bb80f332b422')
  if (pubDomainMode.value === 'simple') {
    return [
      tr('m_0950c73c1224'),
      tr('m_601e88260714', { p0: pubDomain.value }),
      tr('m_f057d85938ac', { p0: tunnelTarget }),
      tr('m_451921dbbbb9'),
    ].join('\n')
  }
  const preferred = pubPreferredCNAME.value.trim() || configStore.config.preferred_cname
  return [
    tr('m_e732d152ef62'),
    tr('m_601e88260714', { p0: pubDomain.value }),
    tr('m_f057d85938ac', { p0: preferred || '<本次或全局优选 CNAME>' }),
    tr('m_2b62848b39cb'),
    ``,
    tr('m_d43173fb0bc5'),
    tr('m_601e88260714', { p0: pubAuxDomain.value || '<辅助回源域名>' }),
    tr('m_f057d85938ac', { p0: tunnelTarget }),
    tr('m_451921dbbbb9'),
  ].join('\n')
})

const saasGuide = computed(() => {
  if (pubDomainMode.value === 'simple') {
    return [
      tr('m_06c2c6838c32', { p0: pubDomain.value }),
      tr('m_cda311735a2c'),
    ].join('\n')
  }
  return [
    tr('m_06c2c6838c32', { p0: pubDomain.value }),
    tr('m_a508a980c029', { p0: pubAuxDomain.value || '<辅助回源域名>' }),
    tr('m_e21d76d29b99'),
  ].join('\n')
})

const ingressGuide = computed(() => {
  const service = configStore.config.service_url || tr('m_5f6877db2d5d')
  const lines = localized(() => ([
    tr('m_cd9259cf6aa1', { p0: pubDomain.value }),
    tr('m_5d8e9fc14e37', { p0: service }),
  ]))
  if (pubDomainMode.value === 'preferred') {
    lines.push(
      tr('m_a48a07d28158', { p0: pubAuxDomain.value || '<辅助回源域名>' }),
      tr('m_5d8e9fc14e37', { p0: service }),
    )
  }
  lines.push(tr('m_aa1fa243759a'))
  return lines.join('\n')
})

const urlHint = computed(() =>
  probeType.value === 'tcp' ? tr('m_37f3e56fb8cf') :
  probeType.value === 'icmp' ? tr('m_03e9b42f3d49') :
  tr('m_2998fbabc22f'))

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
    document.title = (data.name || tr('m_fdc390061698')) + tr('m_812af8bfea68')
    pubTitle.value = data.public_title || ''
    pubIcon.value = data.public_icon || ''
    pubTheme.value = data.public_theme === 'warm' ? 'warm' : ''
    pubSlug.value = data.public_slug || ''
    pubDomain.value = data.public_domain || ''
    pubDomainMode.value = data.public_domain_mode === 'preferred' ? 'preferred' : 'simple'
    pubAuxDomain.value = data.public_aux_domain || ''
    pubPreferredCNAME.value = data.public_preferred_cname || ''
    announcement.value = data.announcement || ''
    alertEnabled.value = Boolean(data.alert_enabled)
    alertEmails.value = data.alert_emails || ''
    void loadAlertLogs()
  } finally {
    loading.value = false
  }
}

function stateText(s?: string) {
  return s === 'ok' ? tr('m_296de0e31f8c') : s === 'warn' ? tr('m_428fb8bfeecf') : s === 'down' ? tr('m_8d7c03019f1d') : tr('m_d80109d7decd')
}

function latencyClass(ms: number) {
  if (!ms) return ''
  if (ms <= 300) return ''
  if (ms <= 1000) return 'mid'
  return 'slow'
}

function barTitle(b: Heartbeat) {
  const d = new Date(b.t)
  const time = isNaN(d.getTime()) ? '' : d.toLocaleString(locale.value) + ' · '
  return time + (b.s === 'ok' ? tr('m_296de0e31f8c') : b.s === 'warn' ? tr('m_428fb8bfeecf') : tr('m_8d7c03019f1d')) + (b.ms != null ? ' · ' + b.ms + 'ms' : '')
}

async function addTarget() {
  if (!tUrl.value || adding.value) return
  adding.value = true
  try {
    const { data } = await addMonitorTarget(monitorId, tName.value, tUrl.value, probeType.value, probeMethod.value, linkEnabled.value)
    monitor.value = data
    message.success(tr('m_45500ea63957'))
    tName.value = ''
    tUrl.value = ''
    void checkNow(true)
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    message.error(msg || tr('m_2ca15f1e2bb7'))
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
    message.success(alertEnabled.value ? tr('m_448c83281819') : tr('m_e2fc2905c1e5'))
  } catch (_) {
    message.error(tr('m_6309a3bb5ba4'))
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
      public_domain_mode: pubDomainMode.value,
      public_aux_domain: pubAuxDomain.value.toLowerCase(),
      public_preferred_cname: pubPreferredCNAME.value.toLowerCase(),
      announcement: announcement.value.trim(),
    })
    monitor.value = data
    if (data.domain_warning) {
      message.warning(tr('m_adfd7207d786', { p0: data.domain_warning }))
    } else if (data.public_domain) {
      message.success(pubDomainMode.value === 'simple'
        ? tr('m_8c5c89b2a677')
        : tr('m_6c3b8ed249d1')
      )
    } else {
      message.success(tr('m_8b6d94c37d84'))
    }
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    message.error(msg || tr('m_6309a3bb5ba4'))
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
    message.success(tr('m_6522a5dd6164'))
    editOpen.value = false
  } catch (_) {
    message.error(tr('m_6309a3bb5ba4'))
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
    message.success(tr('m_9c72008bce55'))
  } catch (_) {
    message.error(tr('m_359d821f4906'))
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
    message.success(tr('m_174688c755b9'))
    removing.value = null
    await load()
  } catch (_) {
    message.error(tr('m_4757fc1d4709'))
  } finally {
    busy.value = false
  }
}

async function checkNow(silent = false) {
  if (!silent) checking.value = true
  try {
    const { data } = await checkMonitorNow(monitorId)
    applyOutcomes(data.outcomes || [])
    if (!silent) message.success(tr('m_37888055c912'))
  } catch (_) {
    if (!silent) message.error(tr('m_34f7713ccce7'))
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
    message.success(data.publish_enabled ? tr('m_9f1e25609b1a') : tr('m_4741640d846b'))
  } catch (_) {
    message.error(tr('m_0c3b4cf7aa25'))
  }
}

async function regenToken() {
  try {
    await updateMonitor(monitorId, { regenerate_token: true })
    await load()
    message.success(tr('m_1babceb12194'))
  } catch (_) {
    message.error(tr('m_470f92db3d64'))
  }
}

function copyLink() {
  navigator.clipboard.writeText(fullPublicUrl.value)
    .then(() => message.success(tr('m_ce710b698fbc')))
    .catch(() => message.error(tr('m_753d8bb0da99')))
}

function copyText(text: string) {
  navigator.clipboard.writeText(text)
    .then(() => message.success(tr('m_8f6f8d979c98')))
    .catch(() => message.warning(tr('m_03ae6cf8db7c')))
}

onMounted(() => {
  load()
  void configStore.fetchConfig().then(loadRoutes)
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
/* Flex, not a fixed grid: the probe method and the jump switch are optional, so
   the column count changes with the probe type. A flex row just reflows. */
.add-row { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; }
.add-row .vercel-input { flex: 0 1 180px; width: auto; min-width: 0; }
.type-sel { flex: 0 0 118px; }
.method-sel { flex: 0 0 92px; }
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
.domain-mode-field { margin-bottom: 12px; }
.status-mode-selector {
  display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); width: min(100%, 320px); padding: 3px;
  background: var(--color-canvas-soft); border: 1px solid var(--color-hairline); border-radius: 7px;
}
.status-mode-option {
  display: flex; flex-direction: column; align-items: flex-start; justify-content: center; min-width: 0; height: 48px;
  padding: 6px 11px; color: var(--color-body); text-align: left; background: transparent;
  border: 1px solid transparent; border-radius: 5px; cursor: pointer;
  transition: background-color 140ms ease, border-color 140ms ease, color 140ms ease;
}
.status-mode-option strong { font-size: 12.5px; line-height: 1.2; color: var(--color-ink); }
.status-mode-option span { max-width: 100%; overflow: hidden; font-size: 10.5px; line-height: 1.4; text-overflow: ellipsis; white-space: nowrap; }
.status-mode-option.active {
  color: var(--color-ink); background: var(--color-canvas-raised); border-color: var(--color-link);
}
.status-mode-option:hover:not(.active) { background: color-mix(in srgb, var(--color-link) 6%, transparent); }
.preferred-domain-fields {
  display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; margin-bottom: 12px;
}
.pub-set .preferred-domain-field { min-width: 0; margin-bottom: 0; }
.status-cname-picker :deep(.cname-input) { height: 34px; font-size: 13px; }
.status-cname-picker :deep(.picker-toggle) { width: 34px; height: 32px; }
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
.add-row .url { flex: 1 1 280px; font-family: var(--font-mono); font-size: 12px; }

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
@media (max-width: 1024px) {
  .add-row > * { flex: 1 1 100%; }
  .preferred-domain-fields { grid-template-columns: 1fr; }
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
