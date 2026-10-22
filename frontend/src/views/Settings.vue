<template>
  <div class="page-container">
    <div class="page-header settings-heading">
            <h2>{{ tr('m_3f754f38a654') }}</h2>
      <p>{{ tr('m_3729a3070b01') }}</p>
    </div>
    <div class="settings-grid section">
      <section class="settings-card settings-card-wide" :class="{ '': visible }" style="animation-delay: 0.04s;">
        <div class="settings-card-header">
          <div>
            <div class="settings-card-title">{{ tr('m_a719b867f3fc') }}</div>
            <div class="settings-card-desc">{{ tr('m_22196a1072dc') }}</div>
          </div>
          <button class="btn btn-primary" :disabled="savingSite" @click="saveSite">
            {{ savingSite ? tr('m_917b1c1f18d0') : tr('m_bcf1ace464e5') }}
          </button>
        </div>
        <div class="brand-editor">
          <div class="brand-preview">
            <span class="brand-preview-label caption-mono">{{ tr('m_a76fdaad7519') }}</span>
            <div class="brand-preview-content">
              <span class="brand-preview-icon">
                <img v-if="site.icon" :src="site.icon" :alt="tr('m_2d68220dd51e')" />
                <svg v-else width="24" height="24" viewBox="0 0 76 76" fill="none" aria-hidden="true">
                  <path d="M49 26H27v24l22-24z" fill="currentColor"/>
                  <path d="M38 38L27 50h22L38 38z" fill="currentColor" fill-opacity="0.42"/>
                </svg>
              </span>
              <div>
                <strong>{{ site.name || 'Tunnel Manager' }}</strong>
                <p>{{ site.description || tr('m_c2c8982623de') }}</p>
              </div>
            </div>
          </div>
          <div class="brand-fields">
            <label class="field">
              <span class="field-label">{{ tr('m_8ac4f0447b99') }}</span>
              <input v-model="site.name" maxlength="60" class="vercel-input" placeholder="Tunnel Manager" />
            </label>
            <label class="field">
              <span class="field-label">{{ tr('m_9ddde2b92952') }}</span>
              <input v-model="site.description" maxlength="160" class="vercel-input" :placeholder="tr('m_c2c8982623de')" />
            </label>
            <label class="field">
              <span class="field-label">{{ tr('m_e220e37bbb93') }}</span>
              <input v-model="site.icon" class="vercel-input" :placeholder="tr('m_22f956aa4f06')" />
            </label>
            <div class="icon-actions">
              <input ref="iconInput" type="file" accept="image/*" class="file-input" @change="handleIconUpload" />
              <button class="btn btn-secondary" type="button" @click="iconInput?.click()">{{ tr('m_9fbb4750bb39') }}</button>
              <button v-if="site.icon" class="btn btn-ghost" type="button" @click="site.icon = ''">{{ tr('m_10e0fd099bd2') }}</button>
              <span>{{ tr('m_8647947e6b61') }}</span>
            </div>
            <label class="field">
              <span class="field-label">{{ tr('m_d22b7b6d0776') }}</span>
              <input v-model="site.panelHost" class="vercel-input" placeholder="panel.example.com" />
              <span class="field-help">{{ tr('m_51b4048beb35') }}</span>
            </label>
            <div class="landing-toggle">
              <div class="landing-toggle-text">
                <span class="field-label">{{ tr('m_87b31918524a') }}</span>
                <span class="landing-toggle-help">{{ tr('m_f4746045e90d') }}</span>
              </div>
              <n-switch v-model:value="site.landingEnabled" size="small" :disabled="savingSite" @update:value="saveSite" />
            </div>
          </div>
        </div>
      </section>
      <section class="settings-card settings-card-wide" :class="{ '': visible }" style="animation-delay: 0.1s;">
        <div class="settings-card-header">
          <div>
            <div class="settings-card-title">{{ tr('m_f60a2d511962') }}</div>
            <div class="settings-card-desc">{{ tr('m_7e7f12cf948f') }}</div>
          </div>
          <button class="btn btn-primary" :disabled="savingPresets" @click="savePresets">
            {{ savingPresets ? tr('m_917b1c1f18d0') : tr('m_801a8daafa2e') }}
          </button>
        </div>
        <div class="default-cname-row">
          <div class="field">
            <span class="field-label">{{ tr('m_71b3aeb8f73c') }}</span>
            <span class="field-help">{{ tr('m_bb8bd65fc13f') }}</span>
            <cname-picker v-model="preferredCNAME" :presets="cnamePresets" placeholder="cf.090227.xyz" />
          </div>
          <button class="btn btn-secondary" :disabled="savingCNAME" @click="savePreferredCNAME">
            {{ savingCNAME ? tr('m_917b1c1f18d0') : tr('m_9e9ef2c6a1ec') }}
          </button>
        </div>
        <div class="preset-list">
          <div v-for="(item, index) in cnamePresets" :key="index" class="preset-row">
            <span class="preset-index">{{ index + 1 }}</span>
            <label class="field">
              <span class="field-label">{{ tr('m_2035432e8a18') }}</span>
              <input v-model="item.name" maxlength="40" class="vercel-input" :placeholder="tr('m_93bc08a447ce')" />
            </label>
            <label class="field preset-value">
              <span class="field-label">{{ tr('m_ce05579d19d5') }}</span>
              <input v-model="item.value" class="vercel-input" :placeholder="tr('m_91d03241c65a')" />
            </label>
            <button class="remove-button" type="button" :disabled="cnamePresets.length === 1" :aria-label="tr('m_cd2e36b1f425')" @click="removePreset(index)">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 5l14 14M19 5 5 19"/></svg>
            </button>
          </div>
        </div>
        <button class="add-preset" type="button" :disabled="cnamePresets.length >= 20" @click="addPreset">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>
          {{ tr('m_f47cac452640') }}
        </button>
      </section>
      <section class="settings-card" :class="{ '': visible }" style="animation-delay: 0.16s;">
        <div class="settings-card-header compact">
          <div>
            <div class="settings-card-title">{{ tr('m_119e1fd67f70') }}</div>
            <div class="settings-card-desc">{{ tr('m_855cdae806a9') }}</div>
          </div>
        </div>
        <div class="settings-input-row">
          <input v-model="fallbackDomain" :placeholder="tr('m_078389cf3046')" class="vercel-input" />
          <button class="btn btn-secondary" :disabled="savingFallback" @click="saveFallback">
            {{ savingFallback ? tr('m_599be7aa10be') : tr('m_0bb58707f90c') }}
          </button>
        </div>
      </section>
      <section class="settings-card" :class="{ '': visible }" style="animation-delay: 0.22s;">
        <div class="settings-card-header compact">
          <div>
            <div class="settings-card-title">{{ tr('m_b903d203f70a') }}</div>
            <div class="settings-card-desc">{{ tr('m_8401fe7d0746') }}</div>
          </div>
        </div>
        <div v-if="config.tunnel_id" class="tunnel-summary">
          <div>
            <strong>{{ config.tunnel_name || tr('m_8ce5320404d5') }}</strong>
            <code>{{ config.tunnel_id }}</code>
          </div>
          <router-link to="/tunnels" class="btn btn-secondary">{{ tr('m_42b8762044e2') }}</router-link>
        </div>
        <div v-else class="tunnel-empty">
          <span>{{ tr('m_0e2cf15a2702') }}</span>
          <router-link to="/tunnels" class="btn btn-primary">{{ tr('m_acfb8f32676c') }}</router-link>
        </div>
      </section>
    </div>
  </div>
</template>
<script setup lang="ts">
import { t as tr } from '../i18n'

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
    message.error(tr('m_85c2f8365c82'))
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
    message.success(tr('m_1964b400b352'))
  } catch (e: any) {
    message.error(tr('m_84bd3b614686') + (e.response?.data?.error || e.message))
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
    message.error(tr('m_35fc560913c0'))
    return
  }
  if (file.size > 512 * 1024) {
    message.error(tr('m_78c1f649e77d'))
    return
  }
  const reader = new FileReader()
  reader.onload = () => { site.icon = String(reader.result || '') }
  reader.onerror = () => message.error(tr('m_c72ad3f4aa4e'))
  reader.readAsDataURL(file)
}
async function savePreferredCNAME() {
  const value = preferredCNAME.value.trim()
  if (!value) {
    message.error(tr('m_5799ba3fe8dc'))
    return
  }
  savingCNAME.value = true
  try {
    await setPreferredCNAME(value)
    preferredCNAME.value = value
    config.preferred_cname = value
    message.success(tr('m_20737048a527'))
  } catch (e: any) {
    message.error(tr('m_84bd3b614686') + (e.response?.data?.error || e.message))
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
    message.error(tr('m_ea92a5f61f83'))
    return
  }
  savingPresets.value = true
  try {
    const { data } = await setCNAMEPresets(items)
    cnamePresets.value = data.cname_presets.map((item) => ({ ...item }))
    config.cname_presets = data.cname_presets
    message.success(tr('m_d12fd29ea414'))
  } catch (e: any) {
    message.error(tr('m_84bd3b614686') + (e.response?.data?.error || e.message))
  } finally {
    savingPresets.value = false
  }
}
async function saveFallback() {
  const domain = fallbackDomain.value.trim()
  if (!domain) {
    message.error(tr('m_6301facafab9'))
    return
  }
  savingFallback.value = true
  try {
    const { data } = await setFallbackOrigin(domain)
    message.success(data.message || tr('m_b9ed9651aa57'))
  } catch (e: any) {
    message.error(tr('m_885260b488be') + (e.response?.data?.error || e.message))
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
