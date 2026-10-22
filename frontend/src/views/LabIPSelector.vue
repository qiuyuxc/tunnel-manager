<template>
  <div class="page-container lab-page">
    <div class="page-header">
      <div>
        <h2>{{ tr('m_3db29474a184') }}</h2>
        <p>{{ tr('m_c903232627f5') }}</p>
      </div>
      <div class="header-actions">
        <span class="tag" :class="running ? 'tag-warn' : 'tag-down'">{{ running ? tr('m_5df4f96d6d93') : tr('m_dae661d17c2d') }}</span>
        <button class="btn btn-secondary" :disabled="loading" @click="refresh()">{{ tr('m_802a407c7743') }}</button>
        <button v-if="running || form.schedule" class="btn btn-secondary" :disabled="acting" @click="stop">{{ tr('m_db53b055cad4') }}</button>
        <button class="btn btn-primary" :disabled="!canRun" @click="run">{{ running ? tr('m_14d9f2d830ad') : tr('m_126a24ed51ca') }}</button>
      </div>
    </div>

    <p v-if="loadError" role="alert">{{ loadError }}</p>
    <section class="settings-card" :aria-label="tr('m_ee87a1453607')">
      <div class="settings-card-header">
        <div>
          <div class="settings-card-title">{{ tr('m_fad1f3ce9254') }}</div>
          <div class="settings-card-desc">{{ tr('m_efff3af6baf6') }}</div>
        </div>
        <span class="tag" :class="authorized ? 'tag-ok' : 'tag-warn'">{{ authorized ? tr('m_521e65ffc7d0') : tr('m_76a2e875d985') }}</span>
      </div>
      <label class="field">
        <span class="field-label">{{ tr('m_ca1c8872d2ec') }}</span>
        <input v-model="authorizationDomain" class="vercel-input" placeholder="example.com" :disabled="acting || running" />
      </label>
      <p class="dns-automation-info">{{ dnsAvailability.available ? tr('m_72f21fa88edc', { p0: dnsAvailability.account_name || '已绑定' }) : dnsAvailability.message }}</p>
      <div class="header-actions">
        <button v-if="dnsAvailability.available" class="btn btn-primary" :disabled="loading || saving || acting || running || !authorizationDomain.trim()" @click="provisionDNS">{{ tr('m_6f6769158cd2') }}</button>
        <button class="btn btn-secondary" :disabled="loading || saving || acting || running || !authorizationDomain.trim()" @click="challenge">{{ tr('m_b906b98d56bf') }}</button>
        <button class="btn btn-primary" :disabled="!verification?.token || acting || running" @click="verify">{{ tr('m_45ef5bfe7b6f') }}</button>
      </div>
      <p v-if="dnsMessage" role="status">{{ dnsMessage }}</p>
      <p v-if="authorized" role="status">{{ tr('m_00097d89c4fe') }}{{ verification?.domain }}</p>
      <div v-if="verification?.token && !authorized" class="verification-record">
        <label class="field"><span class="field-label">{{ tr('m_d5684e061cde') }}</span><input class="vercel-input" readonly :value="verification.record_name" /></label>
        <label class="field"><span class="field-label">{{ tr('m_fae98107cabc') }}</span><textarea class="vercel-input" readonly rows="3" :value="verification.token"></textarea></label>
        <div class="header-actions">
          <button class="btn btn-secondary" @click="copyText(verification.record_name)">{{ tr('m_24b0fd0474b8') }}</button>
          <button class="btn btn-secondary" @click="copyText(verification.token)">{{ tr('m_82554e2b0462') }}</button>
        </div>
        <p v-if="verification.checked_at">{{ tr('m_d308a2437c93') }}{{ formatTime(verification.checked_at) }}</p>
        <p v-if="verification.last_error" role="alert">{{ verification.last_error }}</p>
      </div>
      <p class="budget-summary">{{ tr('m_19653c640000') }} {{ security?.budget_used || 0 }} / {{ security?.budget_limit || 100000 }} · {{ security?.requests_per_second || 10 }} {{ tr('m_7412c06ecb92') }} {{ security?.max_workers || 32 }} {{ tr('m_2b66a2b53a76') }}</p>
      <details class="lab-help"><summary>{{ tr('m_cb4ba40bf823') }}</summary>
        <p>{{ tr('m_23ee3a59baf4') }}</p>
        <p>{{ tr('m_abc9c75b3058') }}</p>
        <p>{{ tr('m_2382c1a8328d') }}</p>
      </details>
      <p v-if="cooldownSeconds > 0">{{ tr('m_ec8fd7fdbb89') }} {{ cooldownSeconds }} {{ tr('m_b84a191d9dc8') }}</p>
      <p v-if="security?.last_error" role="alert">{{ security.last_error }}</p>
      <p v-if="security?.next_run">{{ tr('m_55623af3870c') }}{{ new Date(security.next_run).toLocaleString(locale) }}</p>
    </section>

    <section v-if="progress.total > 0" class="settings-card progress-card">
      <div class="progress-head">
        <span>{{ phaseText }}</span>
        <span class="mono">{{ progress.percent }}%</span>
      </div>
      <div class="progress-track" role="progressbar" :aria-valuenow="progress.percent" aria-valuemin="0" aria-valuemax="100">
        <div class="progress-fill" :class="{ active: running }" :style="{ width: progress.percent + '%' }"></div>
      </div>
      <div class="progress-meta">
        <span>{{ tr('m_36ec02ee7514') }} {{ progress.scanned }} / {{ progress.total }}</span>
        <span>{{ tr('m_fd047ccb2020') }} {{ progress.matched }}</span>
      </div>
    </section>

    <section class="settings-card">
      <div class="settings-card-header">
        <div>
          <div class="settings-card-title">{{ tr('m_d9cb32d240c7') }}</div>
          <div class="settings-card-desc">{{ tr('m_2d028c40657b') }}</div>
        </div>
      </div>
      <div class="lab-grid">
        <label class="field">
          <span class="field-label">Host（HTTP Host Header）</span>
          <input v-model="form.host" type="text" class="vercel-input" placeholder="cdn.example.com" />
        </label>
        <label class="field">
          <span class="field-label">SNI（TLS Server Name）</span>
          <input v-model="form.sni" type="text" class="vercel-input" :placeholder="tr('m_263273daac57')" />
        </label>
        <label class="field">
          <span class="field-label">{{ tr('m_fc12218cc64a') }}</span>
          <input v-model="form.path" type="text" class="vercel-input" placeholder="/healthz" />
        </label>
        <label class="field">
          <span class="field-label">{{ tr('m_f881100e7e8f') }}</span>
          <input v-model="form.statuses" type="text" class="vercel-input" placeholder="200,204" />
        </label>
        <label class="field">
          <span class="field-label">{{ tr('m_e814e078437b') }}</span>
          <input v-model.number="form.timeout" type="number" min="1" max="15" class="vercel-input" />
        </label>
        <label class="field">
          <span class="field-label">{{ tr('m_f33bc7f6cdbb') }}</span>
          <input v-model.number="form.workers" type="number" min="1" :max="security?.max_workers || 32" class="vercel-input" />
        </label>
        <label class="field">
          <span class="field-label">{{ tr('m_7ae5ac99ec7d') }}</span>
          <input v-model.number="form.top" type="number" min="1" max="100" class="vercel-input" />
        </label>
      </div>
      <label class="field">
        <span class="field-label">{{ tr('m_273f5800a334') }}</span>
        <textarea v-model="form.ip_targets" rows="5" class="vercel-input" placeholder="1.2.3.4&#10;1.2.4.0/24&#10;1.2.5.10-1.2.5.20"></textarea>
      </label>
    </section>

    <section class="settings-card">
      <div class="settings-card-header">
        <div>
          <div class="settings-card-title">{{ tr('m_951e770ec529') }}</div>
          <div class="settings-card-desc">{{ tr('m_0b9783ce7495') }}</div>
        </div>
      </div>
      <div class="lab-switch-row">
        <label class="lab-switch">
          <n-switch v-model:value="form.schedule" size="small" :disabled="!authorized || running" />
          <span>{{ tr('m_978cb333292d') }}</span>
        </label>
        <label class="field interval-field">
          <span class="field-label">{{ tr('m_63f674fb44eb') }}</span>
          <input v-model.number="form.interval_minutes" type="number" min="5" max="1440" class="vercel-input" />
        </label>
        <label class="lab-switch">
          <n-switch v-model:value="form.update_dns" size="small" />
          <span>{{ tr('m_7ce4f8dc5c94') }}</span>
        </label>
      </div>
      <div class="lab-grid" v-if="form.update_dns">
        <label class="field">
          <span class="field-label">{{ tr('m_dd7848177924') }}</span>
          <input v-model="form.zone" type="text" class="vercel-input" placeholder="example.com" />
        </label>
        <label class="field">
          <span class="field-label">{{ tr('m_61aa3db2f187') }}</span>
          <input v-model="form.zone_id" type="text" class="vercel-input" :placeholder="tr('m_8a3962db8e0b')" />
        </label>
        <label class="field">
          <span class="field-label">{{ tr('m_1ff08664dd50') }}</span>
          <input v-model="form.record" type="text" class="vercel-input" placeholder="cdn.example.com" />
        </label>
        <label class="field">
          <span class="field-label">TTL</span>
          <input v-model.number="form.ttl" type="number" min="60" max="86400" class="vercel-input" />
        </label>
        <label class="field">
          <span class="field-label">API Endpoint</span>
          <input v-model="form.endpoint" type="url" class="vercel-input" placeholder="https://dns.myhuaweicloud.com" />
        </label>
        <label class="field">
          <span class="field-label">Access Key</span>
          <input v-model="form.access_key" type="text" class="vercel-input" autocomplete="off" />
        </label>
        <label class="field">
          <span class="field-label">Secret Key</span>
          <input v-model="secretKey" type="password" class="vercel-input" :placeholder="hasSecretKey ? tr('m_3260348163d0') : tr('m_2fa84bb7ab9b')" autocomplete="off" />
          <span class="tag" :class="hasSecretKey ? 'tag-ok' : 'tag-down'">{{ hasSecretKey ? tr('m_248a0bb2c9db') : tr('m_83fe22d7f5b2') }}</span>
        </label>
      </div>
      <div class="actions-row">
        <button class="btn btn-primary" :disabled="saving" @click="save">{{ saving ? tr('m_ff509c9ba052') : tr('m_6e584e3d5ce6') }}</button>
      </div>
    </section>

    <section class="settings-card">
      <div class="settings-card-header">
        <div>
          <div class="settings-card-title">{{ tr('m_8f0ef3b2b86d') }}</div>
          <div class="settings-card-desc">{{ latestSummary }}</div>
        </div>
      </div>
      <div v-if="latest?.results?.length" class="table-wrap">
        <table class="lab-table">
          <thead><tr><th>{{ tr('m_6570a98f2850') }}</th><th>IP</th><th>{{ tr('m_06f7d7341212') }}</th><th>{{ tr('m_18045b8c40f1') }}</th></tr></thead>
          <tbody>
            <tr v-for="(item, index) in latest.results" :key="item.ip">
              <td>{{ index + 1 }}</td>
              <td class="mono">{{ item.ip }}</td>
              <td>{{ item.status || '—' }}</td>
              <td>{{ item.latency_ms }} ms</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="empty-state">{{ latest ? tr('m_3318ecab73f4') : tr('m_a1b1a1640ed1') }}</div>
    </section>

    <section class="settings-card">
      <div class="settings-card-header">
        <div>
          <div class="settings-card-title">{{ tr('m_f3cb5e38d5d7') }}</div>
          <div class="settings-card-desc">{{ segmentSummary }}</div>
        </div>
        <div class="segment-actions">
          <button class="btn btn-secondary btn-sm" type="button" :disabled="!missedSegments.length" @click="copyMissedSegments">{{ tr('m_72a7b6671168') }}</button>
          <button class="btn btn-danger btn-sm" type="button" :disabled="!missedSegments.length || running || saving" @click="removeMissedSegments">{{ tr('m_5aefa3f30318') }}</button>
        </div>
      </div>
      <div v-if="missedSegments.length" class="table-wrap segment-wrap">
        <table class="lab-table">
          <thead><tr><th>{{ tr('m_dd9e35831b7f') }}</th><th>{{ tr('m_b6fe8f96b0d6') }}</th><th>{{ tr('m_393df9bb13ea') }}</th><th>{{ tr('m_abee4974732a') }}</th></tr></thead>
          <tbody>
            <tr v-for="item in missedSegments" :key="item.target">
              <td class="mono">{{ item.target }}</td>
              <td>{{ item.scanned }}</td>
              <td>{{ item.matched }}</td>
              <td><span class="tag tag-down">{{ tr('m_0688a3f24ede') }}</span></td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="empty-state">{{ tr('m_46dfefce779d') }}</div>
    </section>

    <section v-if="latest?.diagnostics?.length" class="settings-card" :aria-label="tr('m_7148c33da141')">
      <div class="settings-card-header"><div>
        <div class="settings-card-title">{{ tr('m_7148c33da141') }}</div>
        <div class="settings-card-desc">{{ tr('m_4e1449e7d5e5') }} {{ latest.diagnostics.length }} / {{ latest.rejected }} {{ tr('m_835808f0c9d6') }}</div>
      </div></div>
      <p v-if="latest.probe" class="settings-card-desc diagnostic-context">{{ tr('m_0b33b09c41d2') }}{{ latest.probe.host }}{{ latest.probe.path }} · SNI {{ latest.probe.sni || latest.probe.host }} {{ tr('m_17a0b00c2611') }} {{ latest.probe.statuses }}</p>
      <ul class="diagnostic-list"><li v-for="(item, index) in latest.diagnostics" :key="`${item.ip}-${index}`">
        <div><code>{{ item.ip }}</code><span>{{ item.latency_ms }} ms</span></div>
        <p>{{ describeLabFailure(item, latest.probe?.statuses) }}</p>
      </li></ul>
    </section>

    <section class="settings-card">
      <div class="settings-card-header">
        <div>
          <div class="settings-card-title">{{ tr('m_02451fd58cd3') }}</div>
          <div class="settings-card-desc">{{ tr('m_03bba0057f3d') }}</div>
        </div>
      </div>
      <div v-if="runs.length" class="table-wrap">
        <table class="lab-table">
          <thead><tr><th>{{ tr('m_6a9906c79f26') }}</th><th>{{ tr('m_6320b4a8722a') }}</th><th>{{ tr('m_52142dd33847') }}</th><th>DNS</th><th>{{ tr('m_0bc1fb72ae1b') }}</th></tr></thead>
          <tbody>
            <tr v-for="item in runs" :key="item.id">
              <td>{{ formatTime(item.started_at) }}</td>
              <td><span class="tag" :class="item.success ? 'tag-ok' : 'tag-down'">{{ item.success ? tr('m_053461ce86d2') : tr('m_28384d7afd2e') }}</span></td>
              <td>{{ item.scanned }} / {{ item.matched }}</td>
              <td>{{ item.dns_updated ? tr('m_434203a723b4') : tr('m_bb44f79f1933') }}</td>
              <td class="error-cell">{{ item.error || '—' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="empty-state">{{ tr('m_69c0d2ee422f') }}</div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { usePanelDialogs } from '../composables/usePanelDialogs'
const panel = usePanelDialogs()
import { locale, t as tr } from '../i18n'

import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useMessage, NSwitch } from 'naive-ui'
import { useRouter } from 'vue-router'
import { getMe } from '../api/admin'
import { useConfigStore } from '../stores/config'
import { describeLabFailure, isLabAuthorized } from '../utils/lab'
import {
  getLabIPSelectorSettings,
  getLabIPSelectorStatus,
  runLabIPSelector,
  saveLabIPSelectorSettings,
  createLabChallenge,
  verifyLabOwnership,
  stopLabIPSelector,
  getLabDNSAvailability,
  provisionLabDNS,
  type LabDNSAvailability,
  type LabIPSelectorStatusResponse,
  type LabIPSelectorRun,
  type LabIPSelectorSettings,
  type LabIPSelectorProgress,
  type SaveLabIPSelectorSettings,
} from '../api/lab'

const message = useMessage()
const router = useRouter()
const configStore = useConfigStore()
const ownerId = ref('')
const acting = ref(false)
const loadError = ref('')
const authorizationDomain = ref('')
const dnsAvailability = ref<LabDNSAvailability>({ available: false, message: tr('m_96ef6383be03') })
const dnsMessage = ref('')
const security = ref<LabIPSelectorStatusResponse['status']>()
const verification = computed(() => security.value?.verification)
const now = ref(Date.now())
const authorized = computed(() => isLabAuthorized(verification.value, ownerId.value))
const cooldownSeconds = computed(() => Math.max(0, Math.ceil((security.value?.next_allowed_at || 0) - now.value / 1000)))
const canRun = computed(() => authorized.value && !loading.value && !saving.value && !acting.value && !running.value
  && cooldownSeconds.value === 0 && !!security.value && security.value.budget_used < security.value.budget_limit)
const loading = ref(true)
const saving = ref(false)
const running = ref(false)
const progress = ref<LabIPSelectorProgress>({ scanned: 0, matched: 0, total: 0, percent: 0 })
const phase = ref('')
const hasSecretKey = ref(false)
const secretKey = ref('')
const runs = ref<LabIPSelectorRun[]>([])
const form = ref<LabIPSelectorSettings>({
  host: '',
  sni: '',
  path: '/',
  statuses: '200',
  timeout: 2,
  workers: 32,
  top: 10,
  ip_targets: '',
  schedule: false,
  interval_minutes: 30,
  update_dns: false,
  zone: '',
  zone_id: '',
  record: '',
  ttl: 300,
  endpoint: 'https://dns.myhuaweicloud.com',
  access_key: '',
  has_secret_key: false,
})
let pollTimer: number | undefined
let disposed = false
let statusLoading = false

const latest = computed(() => runs.value[0])
const missedSegments = computed(() => latest.value?.segments?.filter((item) => item.matched === 0) || [])
const missedIPCount = computed(() => missedSegments.value.reduce((total, item) => total + item.scanned, 0))
const segmentSummary = computed(() => {
  const total = latest.value?.segments?.length || 0
  if (!total) return tr('m_2fd51e19890c')
  return tr('m_9c97ed38c9f8', { p0: total, p1: missedSegments.value.length, p2: missedIPCount.value })
})
const phaseText = computed(() => {
  if (phase.value === 'preparing') return tr('m_81c4d554eaac')
  if (phase.value === 'scanning') return tr('m_37425cd304f1')
  if (phase.value === 'updating_dns') return tr('m_5b74adf3e039')
  if (phase.value === 'completed') return tr('m_e9a880e59521')
  if (phase.value === 'failed') return tr('m_3ef426846721')
  return tr('m_4f021f59121c')
})
const latestSummary = computed(() => {
  if (!latest.value) return tr('m_8854c2f7d83b')
  return tr('m_c31381502adf', { p0: formatTime(latest.value.started_at), p1: latest.value.scanned, p2: latest.value.matched, p3: latest.value.selected_ips?.length ? ` · 选中 ${latest.value.selected_ips.length} 个` : '' })
})

async function loadSettings() {
  const { data } = await getLabIPSelectorSettings()
  form.value = data
  hasSecretKey.value = data.has_secret_key
  secretKey.value = ''
}

async function loadStatus() {
  if (statusLoading || disposed) return
  statusLoading = true
  try {
    const { data } = await getLabIPSelectorStatus()
    if (disposed) return
    security.value = data.status
    now.value = Date.now()
    if (!authorizationDomain.value) authorizationDomain.value = data.status.verification.domain || data.status.suggested_domain
    running.value = data.status.running
    progress.value = data.status.progress || { scanned: 0, matched: 0, total: 0, percent: 0 }
    phase.value = data.status.phase || ''
    runs.value = data.runs
    loadError.value = ''
  } finally {
    statusLoading = false
  }
}

async function refresh(showError = true) {
  try {
    await loadStatus()
    if (showError) await loadDNSAvailability()
  } catch (e: any) {
    loadError.value = tr('m_4ac99dfe3af5') + (e.response?.data?.error || e.message)
    if (e.response?.status === 404) {
      configStore.setExperimentalFeatures(false)
      await router.replace('/dashboard')
    }
    if (showError) message.error(tr('m_08417baa06d4') + (e.response?.data?.error || e.message))
  }
}

async function loadDNSAvailability() {
  try {
    const { data } = await getLabDNSAvailability()
    if (!disposed) dnsAvailability.value = data
  } catch {
    if (!disposed) dnsAvailability.value = { available: false, message: tr('m_0ac80d4f8a89') }
  }
}

async function provisionDNS() {
  if (acting.value || saving.value || running.value || !dnsAvailability.value.available) return
  acting.value = true
  dnsMessage.value = ''
  form.value.schedule = false
  try {
    const { data } = await provisionLabDNS(authorizationDomain.value.trim())
    dnsMessage.value = data.message
    if (data.verified) message.success(tr('m_767d6789caeb'))
    else message.warning(tr('m_217a4e26c9df'))
  } catch (error: any) {
    dnsMessage.value = tr('m_33342a67bf61') + (error.response?.data?.error || error.message) + tr('m_0db26bcf68fb')
    message.error(dnsMessage.value)
  } finally {
    await refresh(false)
    acting.value = false
  }
}

async function save() {
  if (!Number.isInteger(form.value.workers) || form.value.workers < 1 || form.value.workers > (security.value?.max_workers || 32)) {
    message.error(tr('m_01121b5c6bb2', { p0: security.value?.max_workers || 32 }))
    return false
  }
  saving.value = true
  let saved = false
  try {
    const { has_secret_key: _hasSecretKey, ...formPayload } = form.value
    const payload: SaveLabIPSelectorSettings = formPayload
    if (secretKey.value) payload.secret_key = secretKey.value
    const { data } = await saveLabIPSelectorSettings(payload)
    form.value = data
    hasSecretKey.value = data.has_secret_key
    secretKey.value = ''
    saved = true
    await loadStatus()
    message.success(tr('m_2890332be041'))
  } catch (e: any) {
    message.error(tr('m_84bd3b614686') + (e.response?.data?.error || e.message))
  } finally {
    saving.value = false
  }
  return saved
}

async function run() {
  if (!canRun.value || !(await save())) return
  acting.value = true
  try {
    await runLabIPSelector()
    running.value = true
    message.success(tr('m_7f935d40a91e'))
    await loadStatus()
  } catch (e: any) {
    message.error(tr('m_1243370e674f') + (e.response?.data?.error || e.message))
    await refresh(false)
  } finally {
    acting.value = false
  }
}

async function challenge() {
  if (acting.value || saving.value || running.value) return
  dnsMessage.value = ''
  if (verification.value?.token && !await panel.confirm(tr('m_ad26c362ad25'))) return
  acting.value = true
  try {
    await createLabChallenge(authorizationDomain.value.trim())
    form.value.schedule = false
    await loadStatus()
    message.success(tr('m_d6390eeb1913'))
  } catch (error: any) {
    message.error(error.response?.data?.error || error.message)
  } finally { acting.value = false }
}

async function verify() {
  acting.value = true
  try {
    await verifyLabOwnership()
    message.success(tr('m_31a1025984de'))
  } catch (error: any) {
    form.value.schedule = false
    message.error(error.response?.data?.error || error.message)
  } finally {
    await refresh(false)
    acting.value = false
  }
}

async function stop() {
  acting.value = true
  try {
    await stopLabIPSelector()
    form.value.schedule = false
    message.success(tr('m_c1a4539fd21f'))
    await loadStatus()
  } catch (error: any) { message.error(error.response?.data?.error || error.message) }
  finally { acting.value = false }
}

async function copyText(text: string) {
  try { await navigator.clipboard.writeText(text); message.success(tr('m_8f6f8d979c98')) }
  catch { message.error(tr('m_03ae6cf8db7c')) }
}

async function poll() {
  await refresh(false)
  if (!disposed) pollTimer = window.setTimeout(poll, running.value ? 1000 : 5000)
}

async function copyMissedSegments() {
  try {
    await navigator.clipboard.writeText(missedSegments.value.map((item) => item.target).join('\n'))
    message.success(tr('m_b364d30acd30'))
  } catch (_) {
    message.error(tr('m_a5f260d9204a'))
  }
}

async function removeMissedSegments() {
  if (!missedSegments.value.length || running.value) return
  if (!await panel.confirm(tr('m_34e8ee3dbdda', { p0: missedSegments.value.length, p1: missedIPCount.value }))) return

  const missedTargets = new Set(missedSegments.value.map((item) => item.target))
  const currentTargets = form.value.ip_targets.split(/[,\s]+/).filter(Boolean)
  const keptTargets = currentTargets.filter((target) => !missedTargets.has(target))
  const removedCount = currentTargets.length - keptTargets.length
  if (!removedCount) {
    message.warning(tr('m_3d3ce77e1cc8'))
    return
  }

  const previousTargets = form.value.ip_targets
  form.value.ip_targets = keptTargets.join('\n')
  const saved = await save()
  if (!saved) form.value.ip_targets = previousTargets
}

function formatTime(seconds: number) {
  return new Date(seconds * 1000).toLocaleString(locale.value)
}

onMounted(async () => {
  try {
    const [user] = await Promise.all([getMe(), loadSettings(), loadStatus(), loadDNSAvailability()])
    ownerId.value = user.data.id
  } catch (error: any) {
    loadError.value = error.response?.data?.error || error.message
  } finally {
    loading.value = false
    if (!disposed) pollTimer = window.setTimeout(poll, 1000)
  }
})

onBeforeUnmount(() => {
  disposed = true
  if (pollTimer) window.clearTimeout(pollTimer)
})
</script>

<style scoped>
/* The heading sits in a row with the toolbar, so it drops the shared block
   layout and the shared bottom margin. */
.lab-page { display: flex; flex-direction: column; gap: 20px; }
.lab-page .page-header { display: flex; align-items: flex-end; justify-content: space-between; gap: var(--spacing-xl); flex-wrap: wrap; margin-bottom: 0; }
.progress-card { padding-bottom: 20px; }
.progress-head { display: flex; justify-content: space-between; gap: 12px; font-size: 14px; font-weight: 600; margin-bottom: 12px; }
.progress-track { height: 10px; border-radius: 999px; background: var(--color-canvas-soft-2); overflow: hidden; }
.progress-fill { height: 100%; border-radius: inherit; background: var(--color-btn-primary); transition: width .25s ease; }
.progress-fill.active::after { content: ""; display: block; height: 100%; background: linear-gradient(90deg, transparent, rgba(255,255,255,.35), transparent); animation: progress-sheen 1.2s linear infinite; }
.progress-meta { display: flex; justify-content: space-between; gap: 12px; margin-top: 10px; color: var(--color-body); font-size: 13px; }
@keyframes progress-sheen { from { transform: translateX(-100%); } to { transform: translateX(100%); } }
.header-actions { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.verification-record { margin-top: 18px; overflow-wrap: anywhere; }
.budget-summary { margin-top: 18px; color: var(--color-mute); font-variant-numeric: tabular-nums; }
.lab-help { color: var(--color-mute); font-size: 13px; line-height: 1.7; overflow-wrap: anywhere; }
.lab-help summary { cursor: pointer; }
.dns-automation-info { margin: 12px 0; color: var(--color-mute); overflow-wrap: anywhere; }
.lab-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 18px 20px; margin-bottom: 22px; }
.lab-switch-row { display: flex; align-items: flex-end; gap: 24px; flex-wrap: wrap; margin-bottom: 22px; }
/* Match the input height so the switch and the field beside it share a baseline. */
.lab-switch { display: inline-flex; align-items: center; gap: 8px; height: 36px; font-size: 14px; }
/* A status pill inside a stacked field must not stretch to the full column. */
.field > .tag { align-self: flex-start; }
.interval-field { min-width: 160px; }
.table-wrap { overflow-x: auto; }
.diagnostic-list { list-style: none; padding: 0; margin: 12px 0 0; }
.diagnostic-list li { padding: 14px 0; border-top: 1px solid var(--color-hairline); }
.diagnostic-list li>div { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; font-size: 13px; }
.diagnostic-list code { font-family: var(--font-mono); overflow-wrap: anywhere; }
.diagnostic-list span { flex: none; color: var(--color-mute); font-size: 12px; }
.diagnostic-list p { margin: 8px 0 0; color: var(--color-body); font-size: 13px; line-height: 1.7; overflow-wrap: anywhere; }
.lab-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.lab-table th, .lab-table td { padding: 12px 14px; border-bottom: 1px solid var(--color-hairline); text-align: left; white-space: nowrap; }
.lab-table tbody tr:last-child td { border-bottom: 0; }
.lab-table th { color: var(--color-mute); font-size: 12px; font-weight: 500; }
.lab-table .mono { font-family: var(--font-mono); }
.error-cell { max-width: 420px; overflow: hidden; text-overflow: ellipsis; }
.empty-state { padding: 40px 0; color: var(--color-mute); text-align: center; font-size: 14px; }
.segment-wrap { max-height: 360px; overflow-y: auto; }
.segment-actions { display: flex; gap: 8px; flex-wrap: wrap; }
@media (max-width: 768px) {
  .lab-page .page-header { align-items: flex-start; }
  .header-actions { width: 100%; }
  .header-actions .btn { flex: 1 1 auto; justify-content: center; }
}
</style>
