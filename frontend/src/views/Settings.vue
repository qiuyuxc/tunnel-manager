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

    <div class="page-header settings-heading section" style="margin-top: var(--spacing-xl);">
      <h2>系统与安全</h2>
      <p>邮件服务、登录保护、通行密钥、人机验证、Cloudflare 授权与加密密钥。</p>
    </div>
    <div class="security-grid section">
      <div class="admin-card">
        <h3>SMTP 邮件服务</h3>
        <p class="admin-hint">用于监控告警邮件与找回密码验证码。密码留空表示保持原值不变。</p>
        <div class="admin-form smtp-form">
          <input v-model="smtp.host" type="text" placeholder="SMTP 主机，如 smtp.example.com" class="vercel-input" />
          <input v-model.number="smtp.port" type="number" placeholder="端口（465 或 587）" class="vercel-input narrow" />
          <input v-model="smtp.username" type="text" placeholder="用户名（通常为邮箱）" class="vercel-input" />
          <input v-model="smtp.password" type="password" placeholder="密码 / 授权码（留空保持不变）" class="vercel-input" />
          <input v-model="smtp.from" type="text" placeholder="发件人，如 panel@example.com" class="vercel-input" />
          <select v-model="smtp.tlsMode" class="vercel-input narrow">
            <option value="ssl">加密</option>
            <option value="plain">不加密</option>
          </select>
          <button class="btn btn-primary" type="button" :disabled="busy" @click="saveSmtp">保存设置</button>
        </div>
        <div class="admin-form">
          <input v-model="testMailTo" type="email" placeholder="发送测试邮件到：你的邮箱" class="vercel-input" />
          <button class="btn btn-secondary" type="button" :disabled="busy || testingMail || !smtp.host" @click="sendTestMail">{{ testingMail ? '发送中…' : '发送测试邮件' }}</button>
        </div>
      </div>

      <div class="admin-card">
        <h3>登录保护</h3>
        <p class="admin-hint">登录、二次验证、验证码、找回与重置密码都受此限制。账号额度是真正的防线——换 IP 绕不过去；IP 额度用来挡住一台机器横扫。任一触发即返回 429 并附带解锁时间。</p>
        <div class="setting-row">
          <span class="setting-label">启用登录限流</span>
          <n-switch v-model:value="settings.rate_limit_enabled" size="small" @update:value="saveSettings" />
        </div>
        <div class="admin-form">
          <label class="field-label2">每账号失败次数<small>0 = 默认 5 次，填负数表示不限</small></label>
          <input v-model.number="settings.rate_limit_per_account" type="number" class="vercel-input" />
          <label class="field-label2">每 IP 失败次数<small>0 = 默认 20 次，填负数表示不限</small></label>
          <input v-model.number="settings.rate_limit_per_ip" type="number" class="vercel-input" />
          <label class="field-label2">统计窗口（分钟）<small>0 = 默认 15 分钟</small></label>
          <input v-model.number="settings.rate_limit_window_minutes" type="number" class="vercel-input" />
          <label class="field-label2">熟悉来源的额度倍数<small>90 天内登录过的网段给更宽的额度。0 = 默认 3 倍，1 = 不放宽</small></label>
          <input v-model.number="settings.rate_limit_familiar_multiplier" type="number" class="vercel-input" />
          <button class="btn btn-primary" type="button" :disabled="busy" @click="saveSettings">保存</button>
        </div>
        <div class="setting-row">
          <span class="setting-label">锁定时通知</span>
          <n-switch v-model:value="settings.rate_limit_notify" size="small" @update:value="saveSettings" />
          <small class="text-muted">走已配置的通知渠道（TG / 邮件），并沿用「登录通知」开关</small>
        </div>
      </div>

      <div class="admin-card">
        <h3>通行密钥</h3>
        <p class="admin-hint">通行密钥（WebAuthn）要求 HTTPS，且依赖方 ID 只能是访问域名本身或其父域。两个字段都留空时按访问域名自动推导，同一个域下的所有子域都能用。只有在需要限定来源时才填「允许的来源」，它必须与依赖方 ID 同域或为其子域——换域名后已绑定的通行密钥会失效，需要重新绑定。</p>
        <div class="admin-form">
          <input v-model="settings.passkey_rp_id" type="text" placeholder="依赖方 ID，如 panel.example.com" class="vercel-input" />
          <input v-model="settings.passkey_origins" type="text" placeholder="允许的来源，逗号分隔，如 https://panel.example.com" class="vercel-input" />
          <button class="btn btn-primary" type="button" :disabled="busy" @click="saveSettings">保存</button>
        </div>
        <div class="setting-row">
          <span class="setting-label">禁用密码登录</span>
          <n-switch v-model:value="passwordLoginOff" size="small" :disabled="!settings.passkey_admin_ready" @update:value="saveSettings">
            <template #checked>仅通行密钥</template>
            <template #unchecked>允许密码</template>
          </n-switch>
          <small v-if="!settings.passkey_admin_ready" class="text-muted">需先绑定至少一个通行密钥，避免面板被锁死</small>
        </div>
      </div>

      <div class="admin-card">
        <h3>人机验证（Cloudflare Turnstile）</h3>
        <p class="admin-hint">可选防护：开启后登录与找回密码需要完成 Cloudflare 人机验证。先在 Cloudflare 控制台创建 Turnstile widget（并把本站域名加入允许域名），再填写 Site Key 与 Secret Key。</p>
        <div class="setting-row">
          <span class="setting-label">启用人机验证</span>
          <n-switch v-model:value="turnstile.enabled" size="small" />
        </div>
        <div class="admin-form">
          <input v-model="turnstile.site_key" type="text" placeholder="Site Key（0x4A…）" class="vercel-input" />
          <input v-model="turnstileSecret" type="password" placeholder="Secret Key（留空保持不变）" class="vercel-input" autocomplete="off" />
          <button class="btn btn-primary" type="button" :disabled="busy" @click="saveTurnstile">保存</button>
          <span class="tag" :class="turnstileHasSecret ? 'tag-ok' : 'tag-down'">{{ turnstileHasSecret ? '密钥：已设置' : '密钥：未设置' }}</span>
        </div>
      </div>

      <div class="admin-card">
        <h3>Cloudflare OAuth 客户端</h3>
        <p class="admin-hint">留空时使用环境变量 CF_OAUTH_CLIENT_ID / CF_OAUTH_CLIENT_SECRET。此处填写后优先于环境变量；密钥留空表示保持不变。</p>
        <p class="admin-hint">OAuth 客户端需在 Cloudflare 控制台勾选权限：Account Settings:Read、Cloudflare Tunnel:Edit、Zone:Read、DNS:Edit、SSL and Certificates:Edit</p>
        <div class="admin-form">
          <input v-model="oauthCfg.client_id" type="text" placeholder="Client ID" class="vercel-input" />
          <input v-model="oauthCfg.client_secret" type="password" placeholder="Client Secret（留空保持不变）" class="vercel-input" />
          <input v-model="oauthCfg.redirect_uri" type="text" placeholder="回调地址（留空自动按访问域名推导）" class="vercel-input" />
          <input v-model="oauthCfg.scopes" type="text" placeholder="Scopes（留空用客户端已配置的）" class="vercel-input" />
          <button class="btn btn-primary" type="button" :disabled="busy" @click="saveOAuth">保存</button>
          <span class="tag" :class="oauthHasSecret ? 'tag-ok' : 'tag-down'">{{ oauthHasSecret ? '密钥：已设置' : '密钥：未设置' }}</span>
        </div>
      </div>

      <div class="admin-card">
        <h3>实验性功能</h3>
        <p class="admin-hint">开启后，侧边栏会显示“IP 优选实验室”。该功能允许直连指定 IP 段探测 Host/SNI 可用性与延迟，并可选择自动更新华为云 DNS。关闭后入口与 API 均不暴露。</p>
        <div class="setting-row">
          <span class="setting-label">开启实验性功能</span>
          <n-switch v-model:value="experimentalFeatures" size="small" @update:value="saveSettings">
            <template #checked>开启</template>
            <template #unchecked>关闭</template>
          </n-switch>
        </div>
      </div>

      <div class="admin-card">
        <h3>应用加密密钥</h3>
        <p class="admin-hint">用于加密 SMTP 密码、Cloudflare 授权令牌与 2FA 密文。未设置时自动生成并保存在数据库；环境变量 APP_ENCRYPTION_KEY 优先。更换后需重启服务，且会使已保存的密文失效。</p>
        <div class="admin-form">
          <input v-model="encKeyInput" type="text" placeholder="64 位十六进制密钥（留空保持不变）" class="vercel-input" />
          <button class="btn btn-primary" type="button" :disabled="busy || !encKeyInput" @click="saveEncKey">保存密钥</button>
          <span class="tag">当前来源：{{ encKeySource === 'env' ? '环境变量' : encKeySource === 'stored' ? '数据库' : '未设置' }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
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
    settings.value = data
    turnstile.value.enabled = !!data.turnstile_enabled
    turnstile.value.site_key = data.turnstile_site_key || ''
    turnstileHasSecret.value = !!data.turnstile_has_secret
    turnstileSecret.value = ''
  } catch (e: any) {
    message.error('加载安全设置失败: ' + (e.response?.data?.error || e.message))
  } finally {
    busy.value = false
  }
}

async function saveSettings() {
  busy.value = true
  try {
    await updateAppSettings(settings.value)
    store.setExperimentalFeatures(!!settings.value.experimental_features_enabled)
    message.success('设置已保存')
  } catch (e: any) {
    message.error(e.response?.data?.error || '保存失败')
  } finally {
    busy.value = false
  }
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
    message.success('人机验证设置已保存')
  } catch (e: any) {
    message.error(e.response?.data?.error || '保存失败')
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
    message.success('OAuth 客户端配置已保存')
  } catch (e: any) {
    message.error(e.response?.data?.error || '保存失败')
  } finally {
    busy.value = false
  }
}

async function saveEncKey() {
  if (busy.value) return
  busy.value = true
  try {
    const { data } = await saveEncryptionKey(encKeyInput.value.trim())
    message.success(data.message || '已保存')
    encKeyInput.value = ''
    const status = await getEncryptionKeyStatus()
    encKeySource.value = status.data.source
  } catch (e: any) {
    message.error(e.response?.data?.error || '保存失败')
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
    message.success(data.configured ? 'SMTP 设置已保存' : 'SMTP 设置已保存（尚未完整配置）')
    smtp.value.password = ''
  } catch (e: any) {
    message.error(e.response?.data?.error || '保存失败')
  } finally {
    busy.value = false
  }
}

async function sendTestMail() {
  if (!testMailTo.value.trim()) {
    message.error('请先填写收件邮箱')
    return
  }
  testingMail.value = true
  try {
    const { data } = await testSMTP(testMailTo.value.trim())
    message.success(data.message || '测试邮件已发送')
  } catch (e: any) {
    message.error(e.response?.data?.error || '发送失败')
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
