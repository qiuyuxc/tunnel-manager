<template>
  <div class="page-container">
    <div class="settings-grid section">
      <section class="settings-card settings-card-wide appearance-card">
        <div class="settings-card-header">
          <div><div class="settings-card-title">{{ tr('console.appearance') }}</div><div class="settings-card-desc">{{ tr('console.appearance_description') }}</div></div>
        </div>
        <div class="appearance-options">
          <label><span>{{ tr('console.dark_theme') }}</span><n-switch v-model:value="store.darkMode" /></label>
          <label><span>{{ tr('console.light_effects') }}</span><n-switch v-model:value="store.lightEffects" /></label>
        </div>
      </section>
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
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg> {{ tr('m_f47cac452640') }} </button>
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

    <div class="section-heading section">
      <h3>{{ tr('slim_dea436156cea') }}</h3>
      <p>{{ tr('slim_8523c968bd2b') }}</p>
    </div>
    <div class="security-grid section">
      <div class="admin-card">
        <h3>{{ tr('m_dd15bdd9da5a') }}</h3>
        <p class="admin-hint">{{ tr('slim_75c52d958508') }}</p>
        <div class="admin-form smtp-form">
          <input v-model="smtp.host" type="text" :placeholder="tr('m_546012b06a1a')" class="vercel-input" />
          <input v-model.number="smtp.port" type="number" :placeholder="tr('m_857bf0d4750f')" class="vercel-input narrow" />
          <input v-model="smtp.username" type="text" :placeholder="tr('m_6a432ca72cc9')" class="vercel-input" />
          <input v-model="smtp.password" type="password" :placeholder="tr('m_077af8674c6a')" class="vercel-input" />
          <input v-model="smtp.from" type="text" :placeholder="tr('m_f070a921c850')" class="vercel-input" />
          <select v-model="smtp.tlsMode" class="vercel-input narrow">
            <option value="ssl">{{ tr('m_320c254572c4') }}</option>
            <option value="plain">{{ tr('m_69f3b744f3f2') }}</option>
          </select>
          <button class="btn btn-primary" type="button" :disabled="busy" @click="saveSmtp">{{ tr('m_c8550237ba70') }}</button>
        </div>
        <div class="admin-form">
          <input v-model="testMailTo" type="email" :placeholder="tr('m_bed1e40c6cbe')" class="vercel-input" />
          <button class="btn btn-secondary" type="button" :disabled="busy || testingMail || !smtp.host" @click="sendTestMail">{{ testingMail ? tr('m_364970d8168c') : tr('m_f9ae1a31638d') }}</button>
        </div>
      </div>

      <div class="admin-card">
        <h3>{{ tr('m_37c24b911b27') }}</h3>
        <p class="admin-hint">{{ tr('slim_606c172de59e') }}</p>
        <div class="setting-row">
          <span class="setting-label">{{ tr('m_829140bedce5') }}</span>
          <n-switch v-model:value="settings.rate_limit_enabled" size="small" @update:value="saveSettings" />
        </div>
        <div class="admin-form">
          <label class="field-label2">{{ tr('m_671a08f4e738') }}<small>{{ tr('m_6728255e4bdb') }}</small></label>
          <input v-model.number="settings.rate_limit_per_account" type="number" class="vercel-input" />
          <label class="field-label2">{{ tr('m_cf5a28cff90f') }}<small>{{ tr('m_f1d1405b8d84') }}</small></label>
          <input v-model.number="settings.rate_limit_per_ip" type="number" class="vercel-input" />
          <label class="field-label2">{{ tr('m_088598d743e5') }}<small>{{ tr('m_5f566691080a') }}</small></label>
          <input v-model.number="settings.rate_limit_window_minutes" type="number" class="vercel-input" />
          <label class="field-label2">{{ tr('m_b9ca02da2aa9') }}<small>{{ tr('m_0fc65b562a9c') }}</small></label>
          <input v-model.number="settings.rate_limit_familiar_multiplier" type="number" class="vercel-input" />
          <button class="btn btn-primary" type="button" :disabled="busy" @click="saveSettings">{{ tr('m_a3030bf8f16d') }}</button>
        </div>
        <div class="setting-row">
          <span class="setting-label">{{ tr('slim_86eed9ba38db') }}</span>
          <n-switch v-model:value="settings.rate_limit_notify" size="small" @update:value="saveSettings" />
          <small class="text-muted">{{ tr('slim_828cd0a82eda') }}</small>
        </div>
      </div>

      <div class="admin-card">
        <h3>{{ tr('m_102824e51a9d') }}</h3>
        <p class="admin-hint">{{ tr('slim_ead7dd2418de') }}</p>
        <div class="admin-form">
          <input v-model="settings.passkey_rp_id" type="text" :placeholder="tr('m_9b9ca2069036')" class="vercel-input" />
          <input v-model="settings.passkey_origins" type="text" :placeholder="tr('m_15b1847e23f3')" class="vercel-input" />
          <button class="btn btn-primary" type="button" :disabled="busy" @click="saveSettings">{{ tr('m_a3030bf8f16d') }}</button>
        </div>
        <div class="setting-row">
          <span class="setting-label">{{ tr('m_3a027241650b') }}</span>
          <n-switch v-model:value="passwordLoginOff" size="small" :disabled="!settings.passkey_admin_ready" @update:value="saveSettings">
            <template #checked>{{ tr('m_bea6fbd73a4a') }}</template>
            <template #unchecked>{{ tr('m_3c54e6e2ef5d') }}</template>
          </n-switch>
          <small v-if="!settings.passkey_admin_ready" class="text-muted">{{ tr('slim_5a6b4ebb3db3') }}</small>
        </div>
      </div>

      <div class="admin-card">
        <h3>{{ tr('m_e4f6aa5a185d') }}</h3>
        <p class="admin-hint">{{ tr('slim_3ae11db137c8') }}</p>
        <div class="setting-row">
          <span class="setting-label">{{ tr('m_b05703226c5e') }}</span>
          <n-switch v-model:value="turnstile.enabled" size="small" />
        </div>
        <div class="admin-form">
          <input v-model="turnstile.site_key" type="text" placeholder="Site Key（0x4A…）" class="vercel-input" />
          <input v-model="turnstileSecret" type="password" :placeholder="tr('m_96656dd2d25f')" class="vercel-input" autocomplete="off" />
          <button class="btn btn-primary" type="button" :disabled="busy" @click="saveTurnstile">{{ tr('m_a3030bf8f16d') }}</button>
          <span class="tag" :class="turnstileHasSecret ? 'tag-ok' : 'tag-down'">{{ turnstileHasSecret ? tr('m_c0ee21a90d7b') : tr('m_5061a4ea571c') }}</span>
        </div>
      </div>

      <div class="admin-card">
        <h3>{{ tr('m_a44454e9d95a') }}</h3>
        <p class="admin-hint">{{ tr('m_c32b4ee1a818') }}</p>
        <p class="admin-hint">{{ tr('m_7a9d0bac635c') }}</p>
        <div class="admin-form">
          <input v-model="oauthCfg.client_id" type="text" placeholder="Client ID" class="vercel-input" />
          <input v-model="oauthCfg.client_secret" type="password" :placeholder="tr('m_be5f73c5b889')" class="vercel-input" />
          <input v-model="oauthCfg.redirect_uri" type="text" :placeholder="tr('m_9df3fe96b710')" class="vercel-input" />
          <input v-model="oauthCfg.scopes" type="text" :placeholder="tr('m_b813dbaced7d')" class="vercel-input" />
          <button class="btn btn-primary" type="button" :disabled="busy" @click="saveOAuth">{{ tr('m_a3030bf8f16d') }}</button>
          <span class="tag" :class="oauthHasSecret ? 'tag-ok' : 'tag-down'">{{ oauthHasSecret ? tr('m_c0ee21a90d7b') : tr('m_5061a4ea571c') }}</span>
        </div>
      </div>

      <div class="admin-card">
        <h3>{{ tr('m_b145d5c1a57b') }}</h3>
        <p class="admin-hint">{{ tr('m_8e24fd96b54f') }}</p>
        <div class="setting-row">
          <span class="setting-label">{{ tr('m_06566a8c40f2') }}</span>
          <n-switch v-model:value="experimentalFeatures" size="small" :disabled="busy" @update:value="saveSettings">
            <template #checked>{{ tr('m_8da97ddda990') }}</template>
            <template #unchecked>{{ tr('m_3fd47edce45b') }}</template>
          </n-switch>
        </div>
        <label class="setting-row">
          <span class="setting-label">{{ tr('m_8b57f87fb23a') }}</span>
          <input v-model.number="settings.lab_daily_request_limit" class="vercel-input narrow" type="number" min="1" max="10000000" step="1" :disabled="busy" />
        </label>
        <label class="setting-row">
          <span class="setting-label">{{ tr('m_b8f8cb247e4d') }}</span>
          <input v-model.number="settings.lab_requests_per_second" class="vercel-input narrow" type="number" min="1" max="1000" step="1" :disabled="busy" />
        </label>
        <label class="setting-row">
          <span class="setting-label">{{ tr('m_652a6f861b6d') }}</span>
          <input v-model.number="settings.lab_max_workers" class="vercel-input narrow" type="number" min="1" max="256" step="1" :disabled="busy" />
        </label>
        <button class="btn btn-primary" type="button" :disabled="busy" @click="saveLabLimits">{{ tr('m_eedfd7ecc753') }}</button>
      </div>

      <div class="admin-card">
        <h3>{{ tr('m_dbf3b43ce49a') }}</h3>
        <p class="admin-hint">{{ tr('m_39a974380a04') }}</p>
        <div class="admin-form">
          <input v-model="encKeyInput" type="text" :placeholder="tr('m_6171d4f4500d')" class="vercel-input" />
          <button class="btn btn-primary" type="button" :disabled="busy || !encKeyInput" @click="saveEncKey">{{ tr('m_a334bb468aba') }}</button>
          <span class="tag">{{ tr('m_1f45e225299d') }}{{ encKeySource === 'env' ? tr('m_ae27b474ea4d') : encKeySource === 'stored' ? tr('m_263200d7d529') : tr('m_2f5f1d6fbfb0') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { t as tr } from '../i18n'
import { computed, onMounted, reactive, ref } from 'vue'
import { NSwitch, useMessage } from 'naive-ui'
import {
  setCNAMEPresets,
  setFallbackOrigin,
  setPreferredCNAME,
  setSiteSettings,
  type CNAMEPreset,
} from '../api'
import {
  getAppSettings, updateAppSettings,
  getSMTP, updateSMTP, testSMTP,
  getOAuthConfig, updateOAuthConfig,
  getEncryptionKeyStatus, saveEncryptionKey,
  type AppSettings,
} from '../api/admin'
import CnamePicker from '../components/CNAMEPicker.vue'
import { useConfigStore } from '../stores/config'
import { validateLabLimits } from '../utils/lab'
const message = useMessage()
const store = useConfigStore()
const config = store.config
const visible = ref(false)

// ---- 系统与安全（原管理后台的系统设置 / 邮件服务，单用户面板下并入全局设置）----
const busy = ref(false)
const settings = ref<AppSettings>({})
const turnstile = ref({ enabled: false, site_key: '' })
const turnstileSecret = ref('')
const turnstileHasSecret = ref(false)
const smtp = ref({ host: '', port: 587, username: '', password: '', from: '', tlsMode: 'ssl' })
const testMailTo = ref('')
const testingMail = ref(false)
const oauthCfg = ref({ client_id: '', client_secret: '', redirect_uri: '', scopes: '' })
const oauthHasSecret = ref(false)
const encKeyInput = ref('')
const encKeySource = ref('none')
const passwordLoginOff = computed({
  get: () => !!settings.value.password_login_disabled,
  set: (v: boolean) => { settings.value.password_login_disabled = v },
})
const experimentalFeatures = computed({
  get: () => !!settings.value.experimental_features_enabled,
  set: (v: boolean) => { settings.value.experimental_features_enabled = v },
})

async function loadSecurity() {
  busy.value = true
  try {
    const { data } = await getAppSettings()
    settings.value = { ...data,
      lab_daily_request_limit: data.lab_daily_request_limit ?? 100000,
      lab_requests_per_second: data.lab_requests_per_second ?? 10,
      lab_max_workers: data.lab_max_workers ?? 32 }
    turnstile.value.enabled = !!data.turnstile_enabled
    turnstile.value.site_key = data.turnstile_site_key || ''
    turnstileHasSecret.value = !!data.turnstile_has_secret
    turnstileSecret.value = ''
  } catch (e: any) {
    message.error(tr('slim_442d0d594e16') + (e.response?.data?.error || e.message))
  } finally {
    busy.value = false
  }
}

async function saveSettings() {
  busy.value = true
  try {
    const { data } = await updateAppSettings(settings.value)
    settings.value = data
    store.setExperimentalFeatures(!!settings.value.experimental_features_enabled)
    message.success(tr('m_bd03c0aad701'))
  } catch (e: any) {
    message.error(e.response?.data?.error || tr('m_6309a3bb5ba4'))
  } finally {
    busy.value = false
  }
}

function saveLabLimits() {
  const error = validateLabLimits(settings.value)
  if (error) return message.error(error)
  void saveSettings()
}

async function saveTurnstile() {
  // 带上完整设置：只保存人机验证时不能把通行密钥、限流等字段清空。
  const payload: Partial<AppSettings> & { turnstile_secret?: string } = {
    ...settings.value,
    turnstile_enabled: turnstile.value.enabled,
    turnstile_site_key: turnstile.value.site_key.trim(),
  }
  if (turnstileSecret.value) payload.turnstile_secret = turnstileSecret.value
  busy.value = true
  try {
    await updateAppSettings(payload)
    turnstileHasSecret.value = turnstileHasSecret.value || !!turnstileSecret.value
    turnstileSecret.value = ''
    message.success(tr('m_763363acbbeb'))
  } catch (e: any) {
    message.error(e.response?.data?.error || tr('m_6309a3bb5ba4'))
  } finally {
    busy.value = false
  }
}

async function saveOAuth() {
  if (busy.value) return
  busy.value = true
  try {
    await updateOAuthConfig({
      client_id: oauthCfg.value.client_id.trim(),
      client_secret: oauthCfg.value.client_secret || undefined,
      redirect_uri: oauthCfg.value.redirect_uri.trim(),
      scopes: oauthCfg.value.scopes.trim(),
    })
    oauthCfg.value.client_secret = ''
    const cfg = await getOAuthConfig()
    oauthHasSecret.value = cfg.data.has_client_secret
    message.success(tr('m_12c3bd5c0eeb'))
  } catch (e: any) {
    message.error(e.response?.data?.error || tr('m_6309a3bb5ba4'))
  } finally {
    busy.value = false
  }
}

async function saveEncKey() {
  if (busy.value) return
  busy.value = true
  try {
    const { data } = await saveEncryptionKey(encKeyInput.value.trim())
    message.success(data.message || tr('m_1bd91a7d0c53'))
    encKeyInput.value = ''
    const status = await getEncryptionKeyStatus()
    encKeySource.value = status.data.source
  } catch (e: any) {
    message.error(e.response?.data?.error || tr('m_6309a3bb5ba4'))
  } finally {
    busy.value = false
  }
}

async function saveSmtp() {
  if (busy.value) return
  busy.value = true
  try {
    const { data } = await updateSMTP({
      host: smtp.value.host.trim(),
      port: smtp.value.port,
      username: smtp.value.username.trim(),
      password: smtp.value.password || undefined,
      from: smtp.value.from.trim(),
      tls_mode: smtp.value.tlsMode,
    })
    message.success(data.configured ? tr('m_dddcb690d105') : tr('m_7afbd323b652'))
    smtp.value.password = ''
  } catch (e: any) {
    message.error(e.response?.data?.error || tr('m_6309a3bb5ba4'))
  } finally {
    busy.value = false
  }
}

async function sendTestMail() {
  if (!testMailTo.value.trim()) {
    message.error(tr('m_e3f39ac6f6c9'))
    return
  }
  testingMail.value = true
  try {
    const { data } = await testSMTP(testMailTo.value.trim())
    message.success(data.message || tr('m_cfb7989b7a73'))
  } catch (e: any) {
    message.error(e.response?.data?.error || tr('m_ac77953a1e06'))
  } finally {
    testingMail.value = false
  }
}
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
  void store.fetchConfig().then(syncFormFromConfig)
  requestAnimationFrame(() => { visible.value = true })
  void loadSecurity()
  getSMTP().then(({ data }) => {
    smtp.value.host = data.host
    smtp.value.port = data.port || 587
    smtp.value.username = data.username
    smtp.value.from = data.from
    smtp.value.tlsMode = data.tls_mode === 'plain' ? 'plain' : 'ssl'
  }).catch(() => {})
  getOAuthConfig().then(({ data }) => {
    oauthCfg.value.client_id = data.client_id
    oauthCfg.value.redirect_uri = data.redirect_uri
    oauthCfg.value.scopes = data.scopes
    oauthHasSecret.value = data.has_client_secret
  }).catch(() => {})
  getEncryptionKeyStatus().then(({ data }) => {
    encKeySource.value = data.source
  }).catch(() => {})
})
</script>
<style scoped>
.appearance-options { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px; }
.appearance-options label { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 16px; border: 1px solid var(--color-hairline); border-radius: 14px; color: var(--color-body); font-size: 13px; }
@media (max-width: 640px) { .appearance-options { grid-template-columns: 1fr; gap: 10px; } }
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

/* 系统与安全区块（原管理后台卡片样式） */
.security-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--spacing-lg);
}
.admin-card {
  min-width: 0;
  background: var(--color-canvas-raised);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
}
.admin-card h3 { margin: 0 0 var(--spacing-sm); font-size: 15px; color: var(--color-ink); }
.admin-hint { font-size: 12px; color: var(--color-mute); margin: 0 0 var(--spacing-sm); line-height: 1.6; }
.admin-form { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; padding: var(--spacing-sm) 0; }
.admin-form .vercel-input { max-width: 260px; }
.admin-form .btn { padding: 5px 14px; font-size: 12px; }
.smtp-form { border-bottom: 1px dashed var(--color-hairline); margin-bottom: var(--spacing-sm); }
.setting-row { display: flex; align-items: center; gap: 14px; padding: var(--spacing-sm) 0; flex-wrap: wrap; }
.setting-label { font-size: 13px; color: var(--color-ink); min-width: 96px; }
.vercel-input.narrow { max-width: 240px; }
.field-label2 { display: flex; flex-direction: column; gap: 2px; font-size: 12px; color: var(--color-body); width: 100%; }
.field-label2 small { font-size: 11px; color: var(--color-mute); }
.text-muted { font-size: 12px; color: var(--color-mute); }
.tag {
  display: inline-flex;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  border: 1px solid var(--color-hairline);
  color: var(--color-mute);
}
.tag-ok { color: var(--color-status-ok, #10b981); border-color: currentColor; }
.tag-down { color: var(--color-error); border-color: currentColor; }
@media (max-width: 1024px) { .security-grid { grid-template-columns: 1fr; } }
</style>
