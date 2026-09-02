<template>
  <div class="page-container">
    <div class="page-header settings-heading">
            <h2>全局设置</h2>
      <p>集中管理站点品牌、域名绑定偏好、回退源与隧道设置。</p>
    </div>
    <div class="settings-grid section">
      <section class="settings-card settings-card-wide" :class="{ '': visible }" style="animation-delay: 0.04s;">
        <div class="settings-card-header">
          <div>
            <div class="settings-card-title">站点信息</div>
            <div class="settings-card-desc">自定义浏览器标题、导航品牌、登录页名称、描述与站点图标。</div>
          </div>
          <button class="btn btn-primary" :disabled="savingSite" @click="saveSite">
            {{ savingSite ? '保存中...' : '保存站点信息' }}
          </button>
        </div>
        <div class="brand-editor">
          <div class="brand-preview">
            <span class="brand-preview-label caption-mono">实时预览</span>
            <div class="brand-preview-content">
              <span class="brand-preview-icon">
                <img v-if="site.icon" :src="site.icon" alt="站点图标预览" />
                <svg v-else width="24" height="24" viewBox="0 0 76 76" fill="none" aria-hidden="true">
                  <path d="M49 26H27v24l22-24z" fill="currentColor"/>
                  <path d="M38 38L27 50h22L38 38z" fill="currentColor" fill-opacity="0.42"/>
                </svg>
              </span>
              <div>
                <strong>{{ site.name || 'Tunnel Manager' }}</strong>
                <p>{{ site.description || 'Cloudflare 隧道管理中心' }}</p>
              </div>
            </div>
          </div>
          <div class="brand-fields">
            <label class="field">
              <span class="field-label">站点名称</span>
              <input v-model="site.name" maxlength="60" class="vercel-input" placeholder="Tunnel Manager" />
            </label>
            <label class="field">
              <span class="field-label">站点描述</span>
              <input v-model="site.description" maxlength="160" class="vercel-input" placeholder="Cloudflare 隧道管理中心" />
            </label>
            <label class="field">
              <span class="field-label">站点图标</span>
              <input v-model="site.icon" class="vercel-input" placeholder="https://example.com/icon.png，或上传本地图片" />
            </label>
            <div class="icon-actions">
              <input ref="iconInput" type="file" accept="image/*" class="file-input" @change="handleIconUpload" />
              <button class="btn btn-secondary" type="button" @click="iconInput?.click()">上传图片</button>
              <button v-if="site.icon" class="btn btn-ghost" type="button" @click="site.icon = ''">清除图标</button>
              <span>建议使用 1:1 图片，文件不超过 512 KB。</span>
            </div>
            <label class="field">
              <span class="field-label">面板域名</span>
              <input v-model="site.panelHost" class="vercel-input" placeholder="panel.example.com" />
              <span class="field-help">状态页自定义域名会复制该域名 ingress 的服务与源站参数。留空时自动使用管理员首次登录时的访问域名。</span>
            </label>
            <div class="landing-toggle">
              <div class="landing-toggle-text">
                <span class="field-label">启用首页（落地页）</span>
                <span class="landing-toggle-help">开启后，未登录用户访问站点将先看到首页，而不是直接进入登录页；关闭则保持原来的登录页。</span>
              </div>
              <n-switch v-model:value="site.landingEnabled" size="small" :disabled="savingSite" @update:value="saveSite" />
            </div>
          </div>
        </div>
      </section>
      <section class="settings-card settings-card-wide" :class="{ '': visible }" style="animation-delay: 0.1s;">
        <div class="settings-card-header">
          <div>
            <div class="settings-card-title">常用 CNAME 组</div>
            <div class="settings-card-desc">维护常用优选线路。域名绑定时可以直接选择，也可以继续手动输入。</div>
          </div>
          <button class="btn btn-primary" :disabled="savingPresets" @click="savePresets">
            {{ savingPresets ? '保存中...' : '保存 CNAME 组' }}
          </button>
        </div>
        <div class="default-cname-row">
          <div class="field">
            <span class="field-label">默认优选 CNAME</span>
            <span class="field-help">未指定线路时自动使用此值。</span>
            <cname-picker v-model="preferredCNAME" :presets="cnamePresets" placeholder="cf.090227.xyz" />
          </div>
          <button class="btn btn-secondary" :disabled="savingCNAME" @click="savePreferredCNAME">
            {{ savingCNAME ? '保存中...' : '保存默认值' }}
          </button>
        </div>
        <div class="preset-list">
          <div v-for="(item, index) in cnamePresets" :key="index" class="preset-row">
            <span class="preset-index">{{ index + 1 }}</span>
            <label class="field">
              <span class="field-label">线路名称</span>
              <input v-model="item.name" maxlength="40" class="vercel-input" placeholder="例如：移动优选" />
            </label>
            <label class="field preset-value">
              <span class="field-label">CNAME 地址</span>
              <input v-model="item.value" class="vercel-input" placeholder="例如：cdn.example.com" />
            </label>
            <button class="remove-button" type="button" :disabled="cnamePresets.length === 1" aria-label="删除此 CNAME" @click="removePreset(index)">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 5l14 14M19 5 5 19"/></svg>
            </button>
          </div>
        </div>
        <button class="add-preset" type="button" :disabled="cnamePresets.length >= 20" @click="addPreset">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>
          添加常用 CNAME
        </button>
      </section>
      <section class="settings-card" :class="{ '': visible }" style="animation-delay: 0.16s;">
        <div class="settings-card-header compact">
          <div>
            <div class="settings-card-title">回退源设置</div>
            <div class="settings-card-desc">设置 Custom Hostnames 的 Fallback Origin。</div>
          </div>
        </div>
        <div class="settings-input-row">
          <input v-model="fallbackDomain" placeholder="例如: fallback.example.com" class="vercel-input" />
          <button class="btn btn-secondary" :disabled="savingFallback" @click="saveFallback">
            {{ savingFallback ? '设置中...' : '设置回退源' }}
          </button>
        </div>
      </section>
      <section class="settings-card" :class="{ '': visible }" style="animation-delay: 0.22s;">
        <div class="settings-card-header compact">
          <div>
            <div class="settings-card-title">当前隧道</div>
            <div class="settings-card-desc">隧道选择统一在隧道管理页完成，避免误填 ID。</div>
          </div>
        </div>
        <div v-if="config.tunnel_id" class="tunnel-summary">
          <div>
            <strong>{{ config.tunnel_name || '已选隧道' }}</strong>
            <code>{{ config.tunnel_id }}</code>
          </div>
          <router-link to="/tunnels" class="btn btn-secondary">切换隧道</router-link>
        </div>
        <div v-else class="tunnel-empty">
          <span>尚未选择隧道</span>
          <router-link to="/tunnels" class="btn btn-primary">选择隧道</router-link>
        </div>
      </section>
    </div>
  </div>
</template>
<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { NSwitch, useMessage } from 'naive-ui'
import {
  setCNAMEPresets,
  setFallbackOrigin,
  setPreferredCNAME,
  setSiteSettings,
  type CNAMEPreset,
} from '../api'
import CnamePicker from '../components/CNAMEPicker.vue'
import { useConfigStore } from '../stores/config'
const message = useMessage()
const store = useConfigStore()
const config = store.config
const visible = ref(false)
const iconInput = ref<HTMLInputElement | null>(null)
const site = reactive({ name: '', description: '', icon: '', panelHost: '', landingEnabled: false })
const cnamePresets = ref<CNAMEPreset[]>([])
const preferredCNAME = ref('')
const fallbackDomain = ref('')
const savingSite = ref(false)
const savingCNAME = ref(false)
const savingPresets = ref(false)
const savingFallback = ref(false)
function syncFormFromConfig() {
  site.name = config.site_name
  site.description = config.site_description
  site.icon = config.site_icon
  site.panelHost = config.panel_host || ''
  site.landingEnabled = store.landingEnabled
  preferredCNAME.value = config.preferred_cname
  cnamePresets.value = config.cname_presets.map((item) => ({ ...item }))
}
async function saveSite() {
  if (!site.name.trim()) {
    message.error('请输入站点名称')
    return
  }
  savingSite.value = true
  try {
    const payload = { name: site.name.trim(), description: site.description.trim(), icon: site.icon.trim(), panel_host: site.panelHost.trim(), landing_enabled: site.landingEnabled }
    const { data } = await setSiteSettings(payload)
    config.site_name = data.name
    config.site_description = data.description
    config.site_icon = data.icon
    config.panel_host = site.panelHost.trim()
    store.landingEnabled = data.landing_enabled
    Object.assign(site, data)
    message.success('站点信息已更新')
  } catch (e: any) {
    message.error('保存失败: ' + (e.response?.data?.error || e.message))
  } finally {
    savingSite.value = false
  }
}
function handleIconUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) {
    message.error('请选择图片文件')
    return
  }
  if (file.size > 512 * 1024) {
    message.error('图片不能超过 512 KB')
    return
  }
  const reader = new FileReader()
  reader.onload = () => { site.icon = String(reader.result || '') }
  reader.onerror = () => message.error('读取图片失败')
  reader.readAsDataURL(file)
}
async function savePreferredCNAME() {
  const value = preferredCNAME.value.trim()
  if (!value) {
    message.error('请输入默认优选 CNAME')
    return
  }
  savingCNAME.value = true
  try {
    await setPreferredCNAME(value)
    preferredCNAME.value = value
    config.preferred_cname = value
    message.success('默认优选 CNAME 已更新')
  } catch (e: any) {
    message.error('保存失败: ' + (e.response?.data?.error || e.message))
  } finally {
    savingCNAME.value = false
  }
}
function addPreset() {
  if (cnamePresets.value.length >= 20) return
  cnamePresets.value.push({ name: '', value: '' })
}
function removePreset(index: number) {
  if (cnamePresets.value.length === 1) return
  cnamePresets.value.splice(index, 1)
}
async function savePresets() {
  const items = cnamePresets.value.map((item) => ({ name: item.name.trim(), value: item.value.trim() }))
  if (items.some((item) => !item.name || !item.value)) {
    message.error('请补全所有线路名称和 CNAME 地址')
    return
  }
  savingPresets.value = true
  try {
    const { data } = await setCNAMEPresets(items)
    cnamePresets.value = data.cname_presets.map((item) => ({ ...item }))
    config.cname_presets = data.cname_presets
    message.success('常用 CNAME 组已更新')
  } catch (e: any) {
    message.error('保存失败: ' + (e.response?.data?.error || e.message))
  } finally {
    savingPresets.value = false
  }
}
async function saveFallback() {
  const domain = fallbackDomain.value.trim()
  if (!domain) {
    message.error('请输入回退源域名')
    return
  }
  savingFallback.value = true
  try {
    const { data } = await setFallbackOrigin(domain)
    message.success(data.message || '回退源已设置')
  } catch (e: any) {
    message.error('设置失败: ' + (e.response?.data?.error || e.message))
  } finally {
    savingFallback.value = false
  }
}
onMounted(async () => {
  await store.fetchConfig()
  syncFormFromConfig()
  requestAnimationFrame(() => { visible.value = true })
})
</script>
<style scoped>
.settings-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--spacing-lg);
}

.settings-card {
  min-width: 0;
  padding: var(--spacing-lg);
  background: var(--color-canvas-raised);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
}

.settings-card-wide { grid-column: 1 / -1; }

.settings-card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--spacing-lg);
  margin-bottom: var(--spacing-lg);
}

.settings-card-header.compact { margin-bottom: var(--spacing-md); }

.settings-card-title {
  margin-bottom: 4px;
  color: var(--color-ink);
  font-size: 16px;
  font-weight: 600;
}

.settings-card-desc {
  max-width: 700px;
  color: var(--color-mute);
  font-size: 13px;
  line-height: 1.6;
}

.field {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 6px;
}

.field-label {
  color: var(--color-ink);
  font-size: 13px;
  font-weight: 500;
  white-space: normal;
  word-break: normal;
  writing-mode: horizontal-tb;
}

.field-help {
  color: var(--color-mute);
  font-size: 12px;
}

/* Brand editor */
.brand-editor {
  display: grid;
  grid-template-columns: minmax(260px, 0.6fr) minmax(0, 1.4fr);
  gap: var(--spacing-lg);
}

.brand-preview {
  min-height: 160px;
  padding: var(--spacing-lg);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  background: var(--color-canvas-soft);
}

.brand-preview-label { color: var(--color-mute); font-size: 12px; }

.brand-preview-content {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  margin-top: var(--spacing-lg);
}

.brand-preview-content strong {
  display: block;
  max-width: 240px;
  color: var(--color-ink);
  font-size: 18px;
  line-height: 1.2;
  overflow-wrap: anywhere;
}

.brand-preview-content p {
  margin: 5px 0 0;
  color: var(--color-mute);
  font-size: 13px;
}

.brand-preview-icon {
  display: inline-flex;
  width: 48px;
  height: 48px;
  flex: none;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: var(--radius-md);
  background: var(--color-ink);
  color: var(--color-canvas-raised);
}

.brand-preview-icon img { width: 100%; height: 100%; object-fit: cover; }

.brand-fields {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--spacing-md);
}

.brand-fields .field:nth-child(2),
.brand-fields .field:nth-child(3),
.brand-fields .icon-actions {
  grid-column: 1 / -1;
}

.icon-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--spacing-sm);
}

.icon-actions span { color: var(--color-mute); font-size: 12px; }

.landing-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding-top: 16px;
  margin-top: 16px;
  border-top: 1px solid var(--color-hairline);
}

.landing-toggle-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.landing-toggle-help {
  font-size: 12px;
  line-height: 1.6;
  color: var(--color-mute);
}

.file-input { display: none; }

/* CNAME presets */
.default-cname-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: end;
  gap: var(--spacing-sm);
  padding: var(--spacing-md);
  margin-bottom: var(--spacing-md);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  background: var(--color-canvas-soft);
}

.preset-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.preset-row {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr) minmax(0, 1fr) 40px;
  align-items: end;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  background: var(--color-canvas-raised);
}

.preset-index {
  display: grid;
  width: 28px;
  height: 32px;
  place-items: center;
  color: var(--color-mute);
  font: 600 12px/1 var(--font-mono);
}

.remove-button {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  color: var(--color-mute);
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: color 120ms ease, background-color 120ms ease;
}

.remove-button:hover:not(:disabled) {
  color: var(--color-error);
  background: var(--color-status-down-bg);
}

.remove-button:disabled { opacity: 0.3; cursor: not-allowed; }

.add-preset {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: var(--spacing-sm);
  padding: 8px 0;
  color: var(--color-link);
  background: transparent;
  border: 0;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}

.add-preset:disabled { color: var(--color-mute); cursor: not-allowed; }

.settings-input-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--spacing-sm);
}

.tunnel-summary,
.tunnel-empty {
  display: flex;
  min-height: 48px;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-md);
}

.tunnel-summary > div {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.tunnel-summary strong { color: var(--color-ink); font-size: 15px; }
.tunnel-summary code { color: var(--color-mute); font: 12px/1.4 var(--font-mono); overflow-wrap: anywhere; }
.tunnel-empty span { color: var(--color-mute); font-size: 14px; }

@media (max-width: 1024px) {
  .settings-grid, .brand-editor { grid-template-columns: 1fr; }
  .settings-card-wide { grid-column: auto; }
}

@media (max-width: 768px) {
  .settings-card-header,
  .default-cname-row,
  .settings-input-row {
    flex-direction: column;
    align-items: stretch;
  }
  .settings-card-header .btn,
  .default-cname-row .btn,
  .settings-input-row .btn { width: 100%; justify-content: center; }
  .brand-fields { grid-template-columns: 1fr; }
  .brand-fields .field,
  .brand-fields .icon-actions { grid-column: auto !important; }
  .preset-row { grid-template-columns: 1fr 40px; gap: var(--spacing-xs); }
  .preset-index { display: none; }
  .preset-row .field { grid-column: 1 / -1; }
  .remove-button { grid-column: 2; grid-row: 1 / span 2; }
  .tunnel-summary, .tunnel-empty { align-items: stretch; flex-direction: column; }
  .tunnel-summary .btn, .tunnel-empty .btn { width: 100%; justify-content: center; }
}
</style>
