<template>
  <div class="page-container lab-page">
    <div class="page-header">
      <div>
        <h2>IP 优选实验室</h2>
        <p>直连探测指定 IP 段，按延迟选择可用 IP，并可自动更新华为云 DNS A 记录</p>
      </div>
      <div class="header-actions">
        <span class="tag" :class="running ? 'tag-warn' : 'tag-down'">{{ running ? '任务运行中' : '空闲' }}</span>
        <button class="btn btn-secondary" :disabled="loading || running" @click="refresh()">刷新状态</button>
        <button class="btn btn-primary" :disabled="saving || running" @click="run">{{ running ? '运行中…' : '立即执行' }}</button>
      </div>
    </div>

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
          <input v-model.number="form.workers" type="number" min="1" max="256" class="vercel-input" />
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
          <n-switch v-model:value="form.schedule" size="small" />
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
              <td><span class="tag tag-down">可剔除</span></td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="empty-state">暂无未命中的 IP 段。</div>
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
import {
  getLabIPSelectorSettings,
  getLabIPSelectorStatus,
  runLabIPSelector,
  saveLabIPSelectorSettings,
  type LabIPSelectorRun,
  type LabIPSelectorSettings,
  type LabIPSelectorProgress,
  type SaveLabIPSelectorSettings,
} from '../api/lab'

const message = useMessage()
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

const latest = computed(() => runs.value[0])
const missedSegments = computed(() => latest.value?.segments?.filter((item) => item.matched === 0) || [])
const missedIPCount = computed(() => missedSegments.value.reduce((total, item) => total + item.scanned, 0))
const segmentSummary = computed(() => {
  const total = latest.value?.segments?.length || 0
  if (!total) return '本轮暂无分段统计。'
  return `共 ${total} 个输入段，其中 ${missedSegments.value.length} 段 0 命中，可减少 ${missedIPCount.value} 个 IP 的后续探测压力。`
})
const phaseText = computed(() => {
  if (phase.value === 'preparing') return '准备扫描…'
  if (phase.value === 'scanning') return '正在探测 IP…'
  if (phase.value === 'updating_dns') return '扫描完成，正在更新华为云 DNS…'
  if (phase.value === 'completed') return '任务已完成'
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
  const { data } = await getLabIPSelectorStatus()
  running.value = data.status.running
  progress.value = data.status.progress || { scanned: 0, matched: 0, total: 0, percent: 0 }
  phase.value = data.status.phase || ''
  runs.value = data.runs
  if (running.value && !pollTimer) {
    pollTimer = window.setInterval(async () => {
      await refresh(false)
      if (!running.value && pollTimer) {
        window.clearInterval(pollTimer)
        pollTimer = undefined
      }
    }, 1000)
  } else if (!running.value && pollTimer) {
    window.clearInterval(pollTimer)
    pollTimer = undefined
  }
}

async function refresh(showError = true) {
  try {
    await Promise.all([loadSettings(), loadStatus()])
  } catch (e: any) {
    if (showError) message.error('加载失败: ' + (e.response?.data?.error || e.message))
  }
}

async function save() {
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
    message.success('实验功能配置已保存')
  } catch (e: any) {
    message.error('保存失败: ' + (e.response?.data?.error || e.message))
  } finally {
    saving.value = false
  }
  return saved
}

async function run() {
  try {
    await runLabIPSelector()
    running.value = true
    message.success('任务已开始执行')
    await loadStatus()
  } catch (e: any) {
    message.error('执行失败: ' + (e.response?.data?.error || e.message))
  }
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
  await refresh()
  loading.value = false
})

onBeforeUnmount(() => {
  if (pollTimer) window.clearInterval(pollTimer)
})
</script>

<style scoped>
.lab-page { display: flex; flex-direction: column; gap: 18px; }
.progress-card { padding-bottom: 18px; }
.progress-head { display: flex; justify-content: space-between; gap: 12px; font-size: 14px; font-weight: 600; margin-bottom: 10px; }
.progress-track { height: 10px; border-radius: 999px; background: var(--color-hairline); overflow: hidden; }
.progress-fill { height: 100%; border-radius: inherit; background: var(--color-btn-primary); transition: width .25s ease; }
.progress-fill.active::after { content: ""; display: block; height: 100%; background: linear-gradient(90deg, transparent, rgba(255,255,255,.35), transparent); animation: progress-sheen 1.2s linear infinite; }
.progress-meta { display: flex; justify-content: space-between; gap: 12px; margin-top: 8px; color: var(--color-mute); font-size: 13px; }
@keyframes progress-sheen { from { transform: translateX(-100%); } to { transform: translateX(100%); } }
.header-actions { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.lab-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 14px; margin-bottom: 16px; }
.lab-switch-row { display: flex; align-items: end; gap: 22px; flex-wrap: wrap; margin-bottom: 16px; }
.lab-switch { display: inline-flex; align-items: center; gap: 8px; padding-bottom: 10px; font-size: 14px; }
.interval-field { min-width: 150px; }
.table-wrap { overflow-x: auto; }
.lab-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.lab-table th, .lab-table td { padding: 9px 10px; border-bottom: 1px solid var(--color-hairline); text-align: left; white-space: nowrap; }
.lab-table th { color: var(--color-mute); font-size: 12px; font-weight: 500; }
.lab-table .mono { font-family: var(--font-mono); }
.error-cell { max-width: 420px; overflow: hidden; text-overflow: ellipsis; }
.empty-state { padding: 28px 0; color: var(--color-mute); text-align: center; font-size: 14px; }
.segment-wrap { max-height: 360px; overflow-y: auto; }
.segment-actions { display: flex; gap: 8px; flex-wrap: wrap; }
@media (max-width: 768px) { .header-actions { width: 100%; } }
</style>
