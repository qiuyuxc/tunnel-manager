<template>
  <div class="page-container">
    <div class="page-header">
      <h2>控制面板</h2>
      <p>系统概览与快捷操作</p>
    </div>
    <!-- Status Cards -->
    <div class="card-grid card-grid-4 section">
      <div class="metric-card">
        <span class="metric-label">当前隧道</span>
        <div class="metric-value">
          <template v-if="config.tunnel_id"><span class="metric-text" :title="config.tunnel_name">{{ config.tunnel_name || '已选隧道' }}</span></template>
          <span v-else class="text-muted">未配置</span>
        </div>
        <div class="metric-foot">
          <router-link to="/tunnels" class="link">管理隧道</router-link>
        </div>
      </div>
      <div class="metric-card">
        <span class="metric-label">转发地址</span>
        <div class="metric-value">
          <code v-if="config.service_url" class="inline-code" :title="config.service_url">{{ config.service_url }}</code>
          <span v-else class="text-muted">未配置</span>
        </div>
        <div class="metric-foot">
          <router-link to="/domain" class="link">设置地址</router-link>
        </div>
      </div>
      <div class="metric-card">
        <span class="metric-label">默认优选 CNAME</span>
        <div class="metric-value">
          <code class="inline-code" :title="config.preferred_cname">{{ config.preferred_cname }}</code>
        </div>
        <div class="metric-foot">
          <router-link to="/settings" class="link">修改默认</router-link>
        </div>
      </div>
      <div class="metric-card">
        <span class="metric-label">运行状态</span>
        <div class="metric-value">
          <span class="status-tag" :class="isReady ? 'healthy' : 'down'">
            {{ isReady ? '配置就绪' : '配置未就绪' }}
          </span>
        </div>
        <div class="metric-foot text-muted">
          {{ isReady ? '可以进行域名绑定' : '缺少必要配置' }}
        </div>
      </div>
    </div>

    <!-- Monitor Overview -->
    <div class="card section">
      <div class="card-header ov-header">
        <span class="caption-mono">监控概览 · 近 7 天</span>
        <router-link to="/monitors" class="link ov-more">前往监控管理</router-link>
      </div>

      <div v-if="ov && ov.targets === 0" class="ov-empty">
        <p>还没有监控项目。</p>
        <p class="ov-empty-sub">创建一个后，这里会展示服务可用率与延迟峰值走势，还能生成对外状态页。</p>
        <router-link to="/monitors" class="btn btn-primary btn-sm">创建监控</router-link>
      </div>

      <div v-else-if="ov" class="ov-body">
        <div class="ov-stats">
          <div class="stat">
            <span class="stat-num">{{ ov.uptime ?? ov.uptime_24h }}<em>%</em></span>
            <span class="stat-lab">7 天可用率</span>
          </div>
          <div class="stat">
            <span class="stat-num">{{ ov.avg_latency_ms || '—' }}<em v-if="ov.avg_latency_ms">ms</em></span>
            <span class="stat-lab">平均延迟</span>
          </div>
          <div class="stat">
            <span class="stat-num">{{ ov.peak_latency_ms || '—' }}<em v-if="ov.peak_latency_ms">ms</em></span>
            <span class="stat-lab">峰值延迟</span>
          </div>
          <div class="stat">
            <span class="stat-num stat-split"><i class="okc">{{ ov.ok }}</i><i class="warnc">{{ ov.warn }}</i><i class="downc">{{ ov.down }}</i></span>
            <span class="stat-lab">正常 / 异常 / 不可达</span>
          </div>
        </div>

        <div class="chart">
          <div class="chart-grid-lines" aria-hidden="true"><i></i><i></i><i></i></div>
          <div class="bars">
            <div
              v-for="b in ov.buckets"
              :key="b.hour"
              class="bar-col"
              :class="['h-' + bucketHealth(b), { 'is-empty': !b.total, 'is-clickable': b.total }]"
              :title="bucketTitle(b)"
              :role="b.total ? 'button' : undefined"
              :tabindex="b.total ? 0 : undefined"
              @click="openDetail(b)"
              @keydown.enter.prevent="openDetail(b)"
              @keydown.space.prevent="openDetail(b)"
            >
              <div class="col-inner">
                <div class="bar-peak" :style="{ height: pctH(b.peak_ms) }"></div>
                <div class="bar-avg" :style="{ height: pctH(b.avg_ms || (b.total ? Math.max(b.peak_ms * 0.35, 6) : 0)) }"></div>
              </div>
              <span class="bar-label">{{ bucketLabel(b.hour) }}</span>
            </div>
          </div>
          <div class="legend">
            <span><i class="lg-peak"></i>峰值</span>
            <span><i class="lg-avg good"></i>平均（正常）</span>
            <span><i class="lg-avg mid"></i>出现异常</span>
            <span><i class="lg-avg bad"></i>出现不可达</span>
          </div>
        </div>
      </div>

      <div v-else class="ov-empty"><p>加载中…</p></div>
    </div>

    <!-- Quick Actions -->
    <div class="card section ov-card">
      <div class="card-header">
        <span class="caption-mono">快捷操作</span>
      </div>
      <div class="quick-actions">
        <router-link to="/tunnels" class="action-tile">
          <div class="action-icon" v-html="icons.tunnel" />
          <div class="action-body">
            <strong>隧道管理</strong>
            <span>查看、创建和选择 Cloudflare Tunnel</span>
          </div>
        </router-link>
        <router-link to="/monitors" class="action-tile">
          <div class="action-icon" v-html="icons.monitor" />
          <div class="action-body">
            <strong>服务监控</strong>
            <span>可用性探测与公开状态页</span>
          </div>
        </router-link>
        <router-link to="/domain" class="action-tile" :class="{ disabled: !isReady }">
          <div class="action-icon" v-html="icons.domain" />
          <div class="action-body">
            <strong>域名绑定</strong>
            <span>将域名绑定到已选隧道</span>
          </div>
        </router-link>
        <router-link to="/dns" class="action-tile">
          <div class="action-icon" v-html="icons.dns" />
          <div class="action-body">
            <strong>DNS 管理</strong>
            <span>管理 Cloudflare DNS 记录</span>
          </div>
        </router-link>
        <router-link to="/settings" class="action-tile">
          <div class="action-icon" v-html="icons.settings" />
          <div class="action-body">
            <strong>全局设置</strong>
            <span>站点品牌、CNAME 预设、回退源</span>
          </div>
        </router-link>
      </div>
    </div>

    <!-- 柱子点开后的当日明细：概览的柱子把所有目标汇总成一根，这里补上是谁 -->
    <SheetPanel :open="detailOpen" :title="detailTitle" @close="detailOpen = false">
      <template v-if="detail">
        <div class="detail-stats">
          <span>检测 {{ detail.total }} 次</span>
          <span v-if="detail.warn" class="warnc">异常 {{ detail.warn }}</span>
          <span v-if="detail.down" class="downc">不可达 {{ detail.down }}</span>
          <span v-if="detail.peak_ms">峰值 {{ detail.peak_ms }}ms</span>
          <span v-if="detail.avg_ms">平均 {{ Math.round(detail.avg_ms) }}ms</span>
        </div>

        <div v-if="!detailIssues.length" class="detail-empty">这一天没有出现异常。</div>

        <template v-else>
          <div class="group-label">涉及 {{ detailIssues.length }} 个目标</div>
          <router-link
            v-for="issue in detailIssues"
            :key="issue.monitor_id + '/' + issue.target_id"
            class="issue-row"
            :to="'/monitors/' + issue.monitor_id"
            @click="detailOpen = false"
          >
            <div class="issue-head">
              <span class="issue-name">{{ issue.monitor_name }} · {{ issue.target_name }}</span>
              <span class="issue-chevron" v-html="navIcons.chevron" />
            </div>
            <div class="issue-meta">
              <span v-if="issue.down" class="downc">不可达 {{ issue.down }}</span>
              <span v-if="issue.warn" class="warnc">异常 {{ issue.warn }}</span>
              <span v-if="issue.peak_ms">峰值 {{ issue.peak_ms }}ms</span>
            </div>
            <div v-for="(inc, i) in issue.incidents" :key="i" class="incident">
              <span class="incident-time">{{ incidentRange(inc) }}</span>
              <span :class="inc.state === 'down' ? 'downc' : 'warnc'">
                {{ inc.state === 'down' ? '不可达' : '降级' }}
              </span>
              <span v-if="incidentDetail(inc)" class="incident-cause">{{ incidentDetail(inc) }}</span>
            </div>
            <div v-if="issue.incident_count > issue.incidents.length" class="incident-more">
              另有 {{ issue.incident_count - issue.incidents.length }} 段未列出
            </div>
          </router-link>
        </template>
      </template>
    </SheetPanel>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { getMonitorOverview, type BucketIncident, type BucketStat, type OverviewResp } from '../api'
import { icons as navIcons } from '../navigation'
import SheetPanel from '../components/SheetPanel.vue'
import { useConfigStore } from '../stores/config'

const configStore = useConfigStore()
const config = computed(() => configStore.config)
const isReady = computed(() => Boolean(config.value.tunnel_id && config.value.service_url))

const icons = {
  tunnel: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>',
  domain: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>',
  dns: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="16" rx="2"/><line x1="2" y1="10" x2="22" y2="10"/><line x1="6" y1="15" x2="10" y2="15"/></svg>',
  settings: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>',
  monitor: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>',
}

const ov = ref<OverviewResp | null>(null)
let timer: ReturnType<typeof setInterval> | undefined

async function loadOverview() {
  try {
    const { data } = await getMonitorOverview()
    ov.value = data
  } catch (_) {
    // keep previous snapshot; empty state handled by targets check
  }
}

function maxPeak() {
  const peaks = (ov.value?.buckets || []).map((b) => b.peak_ms)
  return Math.max(200, ...peaks)
}

function pctH(ms: number) {
  if (!ms) return '0%'
  return Math.max(2, Math.round((ms / maxPeak()) * 100)) + '%'
}

function bucketHealth(b: BucketStat) {
  if (b.down > 0) return 'bad'
  if (b.warn > 0) return 'mid'
  return 'good'
}

/** One column is a day now; the clock label stays for anything shorter. */
function bucketLabel(sec: number) {
  const d = new Date(sec * 1000)
  if (isNaN(d.getTime())) return ''
  // A server that predates the field only ever sent hourly buckets.
  if ((ov.value?.bucket_sec ?? 3600) < 86400) return d.getHours() + '时'
  return d.getMonth() + 1 + '/' + d.getDate()
}

function bucketTitle(b: BucketStat) {
  if (!b.total) return bucketLabel(b.hour) + ' · 无数据'
  return [
    bucketLabel(b.hour),
    '平均 ' + b.avg_ms + 'ms',
    '峰值 ' + b.peak_ms + 'ms',
    '检测 ' + b.total + ' 次',
    b.warn ? '异常 ' + b.warn : '',
    b.down ? '不可达 ' + b.down : '',
  ].filter(Boolean).join(' · ')
}

// ------------------------------------------------------- chart drill-down

const detail = ref<BucketStat | null>(null)
const detailOpen = ref(false)

const detailIssues = computed(() => detail.value?.issues ?? [])
const detailTitle = computed(() => (detail.value ? bucketLabel(detail.value.hour) + ' 的问题' : ''))

/** An empty day has nothing to drill into, so it stays inert. */
function openDetail(b: BucketStat) {
  if (!b.total) return
  detail.value = b
  detailOpen.value = true
}

function clock(sec: number) {
  const d = new Date(sec * 1000)
  return String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0')
}

/** A single-sample window would read as "09:05–09:05"; collapse it. */
function incidentRange(inc: BucketIncident) {
  const from = clock(inc.from)
  const to = clock(inc.to)
  return from === to ? from : from + '–' + to
}

/** The probe's own message is more specific than the bare status code. */
function incidentDetail(inc: BucketIncident) {
  const bits: string[] = []
  if (inc.count > 1) bits.push('连续 ' + inc.count + ' 次')
  const cause = inc.error || (inc.code ? 'HTTP ' + inc.code : '')
  if (cause) bits.push(cause)
  return bits.join(' · ')
}

onMounted(async () => {
  await configStore.fetchConfig()
  await loadOverview()
  timer = setInterval(loadOverview, 60000)
})
onBeforeUnmount(() => { if (timer) clearInterval(timer) })
</script>

<style scoped>
/* Monitor overview */
.ov-header { display: flex; align-items: center; justify-content: space-between; }
.ov-empty { padding: 34px 20px; text-align: center; color: var(--color-mute); font-size: 13.5px; display: flex; flex-direction: column; align-items: center; gap: 8px; }
.ov-empty p { margin: 0; }
.ov-empty-sub { color: var(--color-mute); opacity: .85; }

.ov-stats { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 1px; background: var(--color-hairline); border-bottom: 1px solid var(--color-hairline); }
.stat { background: var(--color-canvas-raised); padding: 16px 18px; display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.stat-num { font-size: 21px; font-weight: 600; color: var(--color-ink); line-height: 1.2; font-variant-numeric: tabular-nums; min-width: 0; overflow-wrap: anywhere; }
.stat-num em { font-style: normal; font-size: 12px; font-weight: 500; color: var(--color-mute); margin-left: 3px; }
.stat-split i { font-style: normal; margin-right: 10px; }
.okc { color: var(--color-success); }
.warnc { color: var(--color-warning); }
.downc { color: var(--color-error); }
.stat-lab { font-size: 11px; letter-spacing: .04em; color: var(--color-mute); }

.chart { position: relative; padding: 22px 18px 14px; background: var(--color-canvas-raised); }
.chart-grid-lines { position: absolute; inset: 22px 18px 34px; pointer-events: none; display: flex; flex-direction: column; justify-content: space-between; }
.chart-grid-lines i { border-top: 1px dashed var(--color-hairline); opacity: .5; }
/* Twelve hourly slots. On a phone the track scrolls sideways rather than
   squeezing every column into a sliver. */
.bars { position: relative; display: flex; align-items: flex-end; gap: 8px; height: 150px; z-index: 1; padding-top: 4px; min-width: 0; overflow-x: auto; overflow-y: hidden; overscroll-behavior-x: contain; }
.bar-col { position: relative; flex: 1 1 0; min-width: 20px; height: 100%; display: flex; flex-direction: column; justify-content: flex-end; cursor: default; }
.col-inner { position: relative; flex: 1; }
.bar-col.is-empty .col-inner::after { content: ""; position: absolute; left: 27%; right: 27%; bottom: 0; height: 3px; border-radius: 2px; background: var(--color-hairline-strong); opacity: .55; }
.bar-peak { position: absolute; bottom: 0; left: 10%; right: 10%; border-radius: 3px 3px 0 0;
  background: color-mix(in srgb, var(--color-ink) 10%, transparent); border: 1px solid color-mix(in srgb, var(--color-ink) 14%, transparent); border-bottom: 0; transition: height 300ms ease; }
.bar-avg { position: absolute; bottom: 0; left: 27%; right: 27%; border-radius: 3px; background: var(--color-ink); transition: height 300ms ease; }
.h-good .bar-avg { background: var(--color-ink); }
.h-mid .bar-avg { background: var(--color-warning); }
.h-bad .bar-avg { background: var(--color-error); }
.h-bad .bar-peak { background: color-mix(in srgb, var(--color-error) 12%, transparent); border-color: color-mix(in srgb, var(--color-error) 25%, transparent); }
/* The label is centred on its own tick and allowed to bleed past the column,
   so a narrow track never clips it into "1…". */
.bar-label { margin-top: 8px; text-align: center; font-size: 11px; color: var(--color-body); white-space: nowrap; }
.bar-col.is-clickable { cursor: pointer; }
.bar-col.is-clickable:focus-visible { outline: 2px solid var(--color-ink); outline-offset: 2px; border-radius: 4px; }
.bar-col.is-clickable:active { opacity: .7; }

/* Chart drill-down (inside SheetPanel) */
.detail-stats { display: flex; flex-wrap: wrap; gap: 6px 14px; padding: 12px 10px 4px; font-size: 12px; color: var(--color-body); }
.detail-empty { padding: 10px; font-size: 13px; color: var(--color-mute); }

.group-label { padding: 14px 10px 5px; font-size: 11px; font-weight: 600; letter-spacing: .06em; text-transform: uppercase; color: var(--color-mute); }

.issue-row { display: block; padding: 12px 10px; border: 1px solid var(--color-hairline); border-radius: 10px; color: inherit; text-decoration: none; }
.issue-row + .issue-row { margin-top: 8px; }
.issue-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.issue-name { font-size: 13px; font-weight: 600; color: var(--color-ink); }
.issue-chevron { flex: 0 0 auto; color: var(--color-mute); }
.issue-chevron :deep(svg) { display: block; width: 14px; height: 14px; }
.issue-meta { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 4px; font-size: 11px; color: var(--color-mute); }

.incident { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 6px; padding-top: 6px; border-top: 1px solid var(--color-hairline); font-size: 11px; color: var(--color-body); }
.incident-time { font-variant-numeric: tabular-nums; color: var(--color-ink); }
.incident-cause { color: var(--color-mute); }
.incident-more { margin-top: 6px; font-size: 11px; color: var(--color-mute); }

.legend { display: flex; justify-content: flex-end; gap: 16px; margin-top: 10px; font-size: 11px; color: var(--color-mute); }
.legend span { display: inline-flex; align-items: center; gap: 5px; }
.legend i { width: 10px; height: 10px; border-radius: 3px; display: inline-block; }
.lg-peak { background: color-mix(in srgb, var(--color-ink) 12%, transparent); border: 1px solid color-mix(in srgb, var(--color-ink) 20%, transparent); }
.lg-avg.good { background: var(--color-ink); }
.lg-avg.mid { background: var(--color-warning); }
.lg-avg.bad { background: var(--color-error); }

@media (max-width: 1024px) {
  .ov-stats { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (max-width: 640px) {
  .chart { padding: 18px 12px 12px; }
  .chart-grid-lines { inset: 18px 12px 32px; }
  /* Keep one label per three hours so they cannot collide in a narrow track. */
  .bar-col:not(:nth-child(3n + 1)) .bar-label { visibility: hidden; }
  .legend { flex-wrap: wrap; justify-content: flex-start; gap: 8px 14px; }
}

.metric-card {
	background: var(--color-canvas-raised);
	border: 1px solid var(--color-hairline);
	border-radius: var(--radius-lg);
	padding: var(--spacing-lg);
	display: flex; flex-direction: column; gap: 8px;
}
.metric-label { font-size: 12px; color: var(--color-mute); font-weight: 500; }
.metric-value { min-height: 24px; min-width: 0; font-size: 15px; font-weight: 600; color: var(--color-ink); display: flex; align-items: center; }
.metric-value .metric-text { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.metric-value .inline-code { display: inline-block; max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; vertical-align: middle; }
.metric-foot { margin-top: auto; padding-top: 8px; border-top: 1px solid var(--color-hairline); font-size: 12px; }
.ov-card { overflow: hidden; }
.quick-actions {
	display: grid;
	grid-template-columns: repeat(4, minmax(0, 1fr));
	gap: 1px;
	background: var(--color-hairline);
}
.action-tile {
	display: flex; align-items: flex-start; gap: 12px;
	padding: var(--spacing-lg);
	background: var(--color-canvas-raised);
	text-decoration: none; color: var(--color-ink);
	transition: background-color 120ms ease;
}
.action-tile:hover { background: var(--color-canvas-soft); }
.action-tile.disabled { opacity: 0.5; pointer-events: none; }
.action-icon { width: 40px; height: 40px; border-radius: var(--radius-md); background: var(--color-canvas-soft); color: var(--color-link); display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; }
.action-body { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.action-body strong { font-size: 14px; font-weight: 600; }
.action-body span { font-size: 12px; color: var(--color-body); line-height: 1.5; }
@media (max-width: 1024px) { .quick-actions { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 768px) { .quick-actions { grid-template-columns: 1fr; } }
</style>
