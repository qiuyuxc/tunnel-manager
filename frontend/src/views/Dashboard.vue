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
          <template v-if="config.tunnel_id">{{ config.tunnel_name || '已选隧道' }}</template>
          <span v-else class="text-muted">未配置</span>
        </div>
        <div class="metric-foot">
          <router-link to="/tunnels" class="link">管理隧道</router-link>
        </div>
      </div>
      <div class="metric-card">
        <span class="metric-label">转发地址</span>
        <div class="metric-value">
          <code v-if="config.service_url" class="inline-code">{{ config.service_url }}</code>
          <span v-else class="text-muted">未配置</span>
        </div>
        <div class="metric-foot">
          <router-link to="/domain" class="link">设置地址</router-link>
        </div>
      </div>
      <div class="metric-card">
        <span class="metric-label">默认优选 CNAME</span>
        <div class="metric-value">
          <code class="inline-code">{{ config.preferred_cname }}</code>
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
        <span class="caption-mono">监控概览 · 近 12 小时</span>
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
            <span class="stat-num">{{ ov.uptime_24h }}<em>%</em></span>
            <span class="stat-lab">24h 可用率</span>
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
              :class="'h-' + bucketHealth(b)"
              :title="bucketTitle(b)"
            >
              <div class="col-inner">
                <div class="bar-peak" :style="{ height: pctH(b.peak_ms) }"></div>
                <div class="bar-avg" :style="{ height: pctH(b.avg_ms || (b.total ? Math.max(b.peak_ms * 0.35, 6) : 0)) }"></div>
              </div>
              <span class="bar-label">{{ hourLabel(b.hour) }}</span>
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
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { getMonitorOverview, type BucketStat, type OverviewResp } from '../api'
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

function hourLabel(sec: number) {
  const d = new Date(sec * 1000)
  return isNaN(d.getTime()) ? '' : d.getHours() + '时'
}

function bucketTitle(b: BucketStat) {
  if (!b.total) return hourLabel(b.hour) + ' · 无数据'
  return [
    hourLabel(b.hour),
    '平均 ' + b.avg_ms + 'ms',
    '峰值 ' + b.peak_ms + 'ms',
    '检测 ' + b.total + ' 次',
    b.warn ? '异常 ' + b.warn : '',
    b.down ? '不可达 ' + b.down : '',
  ].filter(Boolean).join(' · ')
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
.stat { background: var(--color-canvas-raised); padding: 16px 18px; display: flex; flex-direction: column; gap: 4px; }
.stat-num { font-size: 21px; font-weight: 600; color: var(--color-ink); line-height: 1.2; font-variant-numeric: tabular-nums; }
.stat-num em { font-style: normal; font-size: 12px; font-weight: 500; color: var(--color-mute); margin-left: 3px; }
.stat-split i { font-style: normal; margin-right: 10px; }
.okc { color: var(--color-success); }
.warnc { color: var(--color-warning); }
.downc { color: var(--color-error); }
.stat-lab { font-size: 11px; letter-spacing: .04em; color: var(--color-mute); }

.chart { position: relative; padding: 22px 18px 14px; background: var(--color-canvas-raised); }
.chart-grid-lines { position: absolute; inset: 22px 18px 34px; pointer-events: none; display: flex; flex-direction: column; justify-content: space-between; }
.chart-grid-lines i { border-top: 1px dashed var(--color-hairline); opacity: .5; }
.bars { position: relative; display: flex; align-items: flex-end; gap: 10px; height: 140px; z-index: 1; padding-top: 4px; }
.bar-col { position: relative; flex: 1; height: 100%; display: flex; flex-direction: column; justify-content: flex-end; cursor: default; }
.col-inner { position: relative; flex: 1; }
.bar-peak { position: absolute; bottom: 0; left: 10%; right: 10%; border-radius: 3px 3px 0 0;
  background: color-mix(in srgb, var(--color-ink) 10%, transparent); border: 1px solid color-mix(in srgb, var(--color-ink) 14%, transparent); border-bottom: 0; transition: height 300ms ease; }
.bar-avg { position: absolute; bottom: 0; left: 27%; right: 27%; border-radius: 3px; background: var(--color-ink); transition: height 300ms ease; }
.h-good .bar-avg { background: var(--color-ink); }
.h-mid .bar-avg { background: var(--color-warning); }
.h-bad .bar-avg { background: var(--color-error); }
.h-bad .bar-peak { background: color-mix(in srgb, var(--color-error) 12%, transparent); border-color: color-mix(in srgb, var(--color-error) 25%, transparent); }
.bar-label { margin-top: 8px; text-align: center; font-size: 10.5px; color: var(--color-mute); }

.legend { display: flex; justify-content: flex-end; gap: 16px; margin-top: 10px; font-size: 11px; color: var(--color-mute); }
.legend span { display: inline-flex; align-items: center; gap: 5px; }
.legend i { width: 10px; height: 10px; border-radius: 3px; display: inline-block; }
.lg-peak { background: color-mix(in srgb, var(--color-ink) 12%, transparent); border: 1px solid color-mix(in srgb, var(--color-ink) 20%, transparent); }
.lg-avg.good { background: var(--color-ink); }
.lg-avg.mid { background: var(--color-warning); }
.lg-avg.bad { background: var(--color-error); }

@media (max-width: 900px) {
  .ov-stats { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 768px) {
  .quick-actions { grid-template-columns: 1fr; }
}

.metric-card {
	background: var(--color-canvas-raised);
	border: 1px solid var(--color-hairline);
	border-radius: var(--radius-lg);
	padding: var(--spacing-lg);
	display: flex; flex-direction: column; gap: 8px;
}
.metric-label { font-size: 12px; color: var(--color-mute); font-weight: 500; }
.metric-value { min-height: 24px; font-size: 15px; font-weight: 600; color: var(--color-ink); word-break: break-all; display: flex; align-items: center; }
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
@media (max-width: 640px) { .quick-actions { grid-template-columns: 1fr; } .card-grid-4 { grid-template-columns: 1fr; } }
</style>
