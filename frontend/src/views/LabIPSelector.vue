<template>
  <div class="page-container lab-page">
    <div class="page-header">
      <div>
        <h2>IP 优选实验室</h2>
        <p>HTTP 探测、延迟排序与 DNS 更新</p>
      </div>
      <div class="header-actions">
        <span class="tag" :class="running ? 'tag-warn' : 'tag-down'">{{ running ? '任务运行中' : '空闲' }}</span>
        <button class="btn btn-secondary" :disabled="loading" @click="refresh()">刷新状态</button>
        <button v-if="running || form.schedule" class="btn btn-secondary" :disabled="acting" @click="stop">停止并关闭定时</button>
        <button class="btn btn-primary" :disabled="!canRun" @click="run">{{ running ? '运行中…' : '保存并执行' }}</button>
      </div>
    </div>

    <p v-if="loadError" role="alert">{{ loadError }}</p>
    <section class="settings-card" aria-label="域名授权和请求预算">
      <div class="settings-card-header">
        <div>
          <div class="settings-card-title">DNS TXT 域名验证</div>
          <div class="settings-card-desc">与探测 Host/SNI 独立。验证后请保留 DNS 中的 TXT。</div>
        </div>
        <span class="tag" :class="authorized ? 'tag-ok' : 'tag-warn'">{{ authorized ? '已授权' : '待验证' }}</span>
      </div>
      <label class="field">
        <span class="field-label">验证域名</span>
        <input v-model="authorizationDomain" class="vercel-input" placeholder="example.com" :disabled="acting || running" />
      </label>
      <p class="dns-automation-info">{{ dnsAvailability.available ? `DNS 账户：${dnsAvailability.account_name || '已绑定'}` : dnsAvailability.message }}</p>
      <div class="header-actions">
        <button v-if="dnsAvailability.available" class="btn btn-primary" :disabled="loading || saving || acting || running || !authorizationDomain.trim()" @click="provisionDNS">一键填写并验证</button>
        <button class="btn btn-secondary" :disabled="loading || saving || acting || running || !authorizationDomain.trim()" @click="challenge">生成 TXT</button>
        <button class="btn btn-primary" :disabled="!verification?.token || acting || running" @click="verify">检查 TXT 记录</button>
      </div>
      <p v-if="dnsMessage" role="status">{{ dnsMessage }}</p>
      <p v-if="authorized" role="status">已验证：{{ verification?.domain }}</p>
      <div v-if="verification?.token && !authorized" class="verification-record">
        <label class="field"><span class="field-label">TXT 完整记录名</span><input class="vercel-input" readonly :value="verification.record_name" /></label>
        <label class="field"><span class="field-label">TXT 记录值</span><textarea class="vercel-input" readonly rows="3" :value="verification.token"></textarea></label>
        <div class="header-actions">
          <button class="btn btn-secondary" @click="copyText(verification.record_name)">复制记录名</button>
          <button class="btn btn-secondary" @click="copyText(verification.token)">复制记录值</button>
        </div>
        <p v-if="verification.checked_at">上次检查：{{ formatTime(verification.checked_at) }}</p>
        <p v-if="verification.last_error" role="alert">{{ verification.last_error }}</p>
      </div>
      <p class="budget-summary">今日预算 {{ security?.budget_used || 0 }} / {{ security?.budget_limit || 100000 }} · {{ security?.requests_per_second || 10 }} 请求/秒 · 最多 {{ security?.max_workers || 32 }} 并发</p>
      <details class="lab-help"><summary>使用说明</summary>
        <p>预算由管理员设置，按候选 IP 数量预留，取消不退；UTC 次日重置。限额修改从下一轮生效，已有配置超出新并发上限时按上限运行。这不是云平台用量统计。</p>
        <p>一键验证只添加本次 TXT，不覆盖其他记录，也不启动扫描。验证通过后收起记录值，运行前仍会复查；请勿删除 DNS 中的 TXT。</p>
        <p>只探测自建或已获授权的 Host/SNI。TXT 验证不代表探针免计费，请向托管平台确认。</p>
      </details>
      <p v-if="cooldownSeconds > 0">启动冷却剩余 {{ cooldownSeconds }} 秒。</p>
      <p v-if="security?.last_error" role="alert">{{ security.last_error }}</p>
      <p v-if="security?.next_run">下次定时执行：{{ new Date(security.next_run).toLocaleString() }}</p>
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
        <span>已扫描 {{ progress.scanned }} / {{ progress.total }}</span>
        <span>匹配 {{ progress.matched }}</span>
      </div>
    </section>

    <section class="settings-card">
      <div class="settings-card-header">
        <div>
          <div class="settings-card-title">探测配置</div>
          <div class="settings-card-desc">Host 与 SNI 可指向托管在 Cloudflare 或其他平台的域名；SNI 留空时使用 Host。</div>
        </div>
      </div>
      <div class="lab-grid">
        <label class="field">
          <span class="field-label">Host（HTTP Host Header）</span>
          <input v-model="form.host" type="text" class="vercel-input" placeholder="cdn.example.com" />
        </label>
        <label class="field">
          <span class="field-label">SNI（TLS Server Name）</span>
          <input v-model="form.sni" type="text" class="vercel-input" placeholder="留空使用 Host" />
        </label>
        <label class="field">
          <span class="field-label">探测路径</span>
          <input v-model="form.path" type="text" class="vercel-input" placeholder="/healthz" />
        </label>
        <label class="field">
          <span class="field-label">接受状态码</span>
          <input v-model="form.statuses" type="text" class="vercel-input" placeholder="200,204" />
        </label>
        <label class="field">
          <span class="field-label">超时（秒）</span>
          <input v-model.number="form.timeout" type="number" min="1" max="15" class="vercel-input" />
        </label>
        <label class="field">
          <span class="field-label">并发数</span>
          <input v-model.number="form.workers" type="number" min="1" :max="security?.max_workers || 32" class="vercel-input" />
        </label>
        <label class="field">
          <span class="field-label">保留 Top N</span>
          <input v-model.number="form.top" type="number" min="1" max="100" class="vercel-input" />
        </label>
      </div>
      <label class="field">
        <span class="field-label">IP 段（支持单 IP、CIDR、范围，逗号或换行分隔）</span>
        <textarea v-model="form.ip_targets" rows="5" class="vercel-input" placeholder="1.2.3.4&#10;1.2.4.0/24&#10;1.2.5.10-1.2.5.20"></textarea>
      </label>
    </section>

    <section class="settings-card">
      <div class="settings-card-header">
        <div>
          <div class="settings-card-title">自动化与华为云 DNS</div>
          <div class="settings-card-desc">密钥使用应用加密密钥加密保存；关闭“自动更新 DNS”时仅扫描并展示结果。</div>
        </div>
      </div>
      <div class="lab-switch-row">
        <label class="lab-switch">
          <n-switch v-model:value="form.schedule" size="small" :disabled="!authorized || running" />
          <span>定时执行</span>
        </label>
        <label class="field interval-field">
          <span class="field-label">间隔（分钟）</span>
          <input v-model.number="form.interval_minutes" type="number" min="5" max="1440" class="vercel-input" />
        </label>
        <label class="lab-switch">
          <n-switch v-model:value="form.update_dns" size="small" />
          <span>自动更新华为云 DNS</span>
        </label>
      </div>
      <div class="lab-grid" v-if="form.update_dns">
        <label class="field">
          <span class="field-label">Zone 名称</span>
          <input v-model="form.zone" type="text" class="vercel-input" placeholder="example.com" />
        </label>
        <label class="field">
          <span class="field-label">Zone ID（可选）</span>
          <input v-model="form.zone_id" type="text" class="vercel-input" placeholder="填写后跳过 Zone 查询" />
        </label>
        <label class="field">
          <span class="field-label">A 记录</span>
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
          <input v-model="secretKey" type="password" class="vercel-input" :placeholder="hasSecretKey ? '留空保持不变' : '输入 Secret Access Key'" autocomplete="off" />
          <span class="tag" :class="hasSecretKey ? 'tag-ok' : 'tag-down'">{{ hasSecretKey ? '密钥已设置' : '密钥未设置' }}</span>
        </label>
      </div>
      <div class="actions-row">
        <button class="btn btn-primary" :disabled="saving" @click="save">{{ saving ? '保存中…' : '保存配置' }}</button>
      </div>
    </section>

    <section class="settings-card">
      <div class="settings-card-header">
        <div>
          <div class="settings-card-title">最近结果</div>
          <div class="settings-card-desc">{{ latestSummary }}</div>
        </div>
      </div>
      <div v-if="latest?.results?.length" class="table-wrap">
        <table class="lab-table">
          <thead><tr><th>排名</th><th>IP</th><th>状态码</th><th>延迟</th></tr></thead>
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
      <div v-else class="empty-state">{{ latest ? '本轮没有命中任何 IP。' : '暂无探测结果，请保存配置后立即执行。' }}</div>
    </section>

    <section class="settings-card">
      <div class="settings-card-header">
        <div>
          <div class="settings-card-title">未命中 IP 段</div>
          <div class="settings-card-desc">{{ segmentSummary }}</div>
        </div>
        <div class="segment-actions">
          <button class="btn btn-secondary btn-sm" type="button" :disabled="!missedSegments.length" @click="copyMissedSegments">复制未命中段</button>
          <button class="btn btn-danger btn-sm" type="button" :disabled="!missedSegments.length || running || saving" @click="removeMissedSegments">一键剔除并保存</button>
        </div>
      </div>
      <div v-if="missedSegments.length" class="table-wrap segment-wrap">
        <table class="lab-table">
          <thead><tr><th>输入段</th><th>已探测 IP</th><th>命中</th><th>建议</th></tr></thead>
          <tbody>
            <tr v-for="item in missedSegments" :key="item.target">
              <td class="mono">{{ item.target }}</td>
              <td>{{ item.scanned }}</td>
              <td>{{ item.matched }}</td>
              <td><span class="tag tag-down">先核对原因</span></td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="empty-state">暂无未命中的 IP 段。</div>
    </section>

    <section v-if="latest?.diagnostics?.length" class="settings-card" aria-label="未命中原因">
      <div class="settings-card-header"><div>
        <div class="settings-card-title">未命中原因</div>
        <div class="settings-card-desc">显示 {{ latest.diagnostics.length }} / {{ latest.rejected }} 条，最多 100 条。未命中仅代表本次探测失败。</div>
      </div></div>
      <p v-if="latest.probe" class="settings-card-desc diagnostic-context">本次探针：{{ latest.probe.host }}{{ latest.probe.path }} · SNI {{ latest.probe.sni || latest.probe.host }} · 接受 {{ latest.probe.statuses }}</p>
      <ul class="diagnostic-list"><li v-for="(item, index) in latest.diagnostics" :key="`${item.ip}-${index}`">
        <div><code>{{ item.ip }}</code><span>{{ item.latency_ms }} ms</span></div>
        <p>{{ describeLabFailure(item, latest.probe?.statuses) }}</p>
      </li></ul>
    </section>

    <section class="settings-card">
      <div class="settings-card-header">
        <div>
          <div class="settings-card-title">执行历史</div>
          <div class="settings-card-desc">最多保留 20 条记录。</div>
        </div>
      </div>
      <div v-if="runs.length" class="table-wrap">
        <table class="lab-table">
          <thead><tr><th>开始时间</th><th>状态</th><th>扫描 / 匹配</th><th>DNS</th><th>错误</th></tr></thead>
          <tbody>
            <tr v-for="item in runs" :key="item.id">
              <td>{{ formatTime(item.started_at) }}</td>
              <td><span class="tag" :class="item.success ? 'tag-ok' : 'tag-down'">{{ item.success ? '成功' : '失败' }}</span></td>
              <td>{{ item.scanned }} / {{ item.matched }}</td>
              <td>{{ item.dns_updated ? '已更新' : '未更新' }}</td>
              <td class="error-cell">{{ item.error || '—' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="empty-state">暂无执行历史。</div>
    </section>
  </div>
</template>

<script setup lang="ts">
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
const dnsAvailability = ref<LabDNSAvailability>({ available: false, message: '正在检查已绑定的 DNS 账户…' })
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
  if (!total) return '本轮暂无分段统计。'
  return `${total} 个输入段 · ${missedSegments.value.length} 段未命中 · ${missedIPCount.value} 个 IP。剔除前请核对原因。`
})
const phaseText = computed(() => {
  if (phase.value === 'preparing') return '准备扫描…'
  if (phase.value === 'scanning') return '正在探测 IP…'
  if (phase.value === 'updating_dns') return '扫描完成，正在更新华为云 DNS…'
  if (phase.value === 'completed') return '任务已完成'
  if (phase.value === 'failed') return '任务未完成，请检查错误信息'
  return '等待执行'
})
const latestSummary = computed(() => {
  if (!latest.value) return '尚未执行。'
  return `${formatTime(latest.value.started_at)} · 扫描 ${latest.value.scanned} 个，匹配 ${latest.value.matched} 个${latest.value.selected_ips?.length ? ` · 选中 ${latest.value.selected_ips.length} 个` : ''}`
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
    loadError.value = '状态加载失败：' + (e.response?.data?.error || e.message)
    if (e.response?.status === 404) {
      configStore.setExperimentalFeatures(false)
      await router.replace('/dashboard')
    }
    if (showError) message.error('加载失败: ' + (e.response?.data?.error || e.message))
  }
}

async function loadDNSAvailability() {
  try {
    const { data } = await getLabDNSAvailability()
    if (!disposed) dnsAvailability.value = data
  } catch {
    if (!disposed) dnsAvailability.value = { available: false, message: '暂时无法检查已绑定账户，可手动添加 TXT，或点击刷新状态重试。' }
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
    if (data.verified) message.success('TXT 已自动填写并验证通过')
    else message.warning('TXT 已提交，请等待 DNS 生效后再次检查')
  } catch (error: any) {
    dnsMessage.value = '自动填写未完成：' + (error.response?.data?.error || error.message) + '。可继续使用手动添加。'
    message.error(dnsMessage.value)
  } finally {
    await refresh(false)
    acting.value = false
  }
}

async function save() {
  if (!Number.isInteger(form.value.workers) || form.value.workers < 1 || form.value.workers > (security.value?.max_workers || 32)) {
    message.error(`并发数须为 1 至 ${security.value?.max_workers || 32} 的整数`)
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
    message.success('实验功能配置已保存')
  } catch (e: any) {
    message.error('保存失败: ' + (e.response?.data?.error || e.message))
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
    message.success('任务已开始执行')
    await loadStatus()
  } catch (e: any) {
    message.error('执行失败: ' + (e.response?.data?.error || e.message))
    await refresh(false)
  } finally {
    acting.value = false
  }
}

async function challenge() {
  if (acting.value || saving.value || running.value) return
  dnsMessage.value = ''
  if (verification.value?.token && !window.confirm('生成新记录会撤销旧授权并关闭定时任务，是否继续？')) return
  acting.value = true
  try {
    await createLabChallenge(authorizationDomain.value.trim())
    form.value.schedule = false
    await loadStatus()
    message.success('请添加 TXT 记录，然后点击检查')
  } catch (error: any) {
    message.error(error.response?.data?.error || error.message)
  } finally { acting.value = false }
}

async function verify() {
  acting.value = true
  try {
    await verifyLabOwnership()
    message.success('域名验证通过；定时任务需手动开启')
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
    message.success('已请求停止，并关闭定时任务')
    await loadStatus()
  } catch (error: any) { message.error(error.response?.data?.error || error.message) }
  finally { acting.value = false }
}

async function copyText(text: string) {
  try { await navigator.clipboard.writeText(text); message.success('已复制') }
  catch { message.error('复制失败，请手动选择文本') }
}

async function poll() {
  await refresh(false)
  if (!disposed) pollTimer = window.setTimeout(poll, running.value ? 1000 : 5000)
}

async function copyMissedSegments() {
  try {
    await navigator.clipboard.writeText(missedSegments.value.map((item) => item.target).join('\n'))
    message.success('未命中 IP 段已复制')
  } catch (_) {
    message.error('复制失败，请手动选择表格内容')
  }
}

async function removeMissedSegments() {
  if (!missedSegments.value.length || running.value) return
  if (!window.confirm(`确定剔除 ${missedSegments.value.length} 个未命中段吗？将减少 ${missedIPCount.value} 个 IP 的后续探测压力。`)) return

  const missedTargets = new Set(missedSegments.value.map((item) => item.target))
  const currentTargets = form.value.ip_targets.split(/[,\s]+/).filter(Boolean)
  const keptTargets = currentTargets.filter((target) => !missedTargets.has(target))
  const removedCount = currentTargets.length - keptTargets.length
  if (!removedCount) {
    message.warning('当前输入中没有找到与最近一轮完全一致的未命中段')
    return
  }

  const previousTargets = form.value.ip_targets
  form.value.ip_targets = keptTargets.join('\n')
  const saved = await save()
  if (!saved) form.value.ip_targets = previousTargets
}

function formatTime(seconds: number) {
  return new Date(seconds * 1000).toLocaleString()
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
