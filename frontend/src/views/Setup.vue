<template>
  <div class="setup-page">
    <div class="setup-card">
      <header class="setup-head">
        <span class="logo-mark" aria-hidden="true">
          <svg width="18" height="18" viewBox="0 0 76 76" fill="none">
            <path d="M49 26H27v24l22-24z" fill="currentColor"/>
            <path d="M38 38L27 50h22L38 38z" fill="currentColor" fill-opacity="0.42"/>
          </svg>
        </span>
        <h1 class="setup-title">{{ tr('m_e03334f01c2f') }}</h1>
        <p class="setup-subtitle">{{ tr('m_5f7533982f55') }}</p>
      </header>

      <div v-if="loading" class="setup-loading">
        <span class="spinner"></span>
        <span>{{ tr('m_3a0b3e662e0b') }}</span>
      </div>

      <template v-else-if="status">
        <ol class="setup-steps">
          <li :class="{ active: step === 1, done: step > 1 }"><em>1</em>{{ tr('m_263200d7d529') }}</li>
          <li :class="{ active: step === 2 }"><em>2</em>{{ tr('m_aea56d3951f2') }}</li>
        </ol>

        <section v-if="step === 1" class="setup-section">
          <div v-if="status.env_locked" class="setup-note">
            <strong>{{ tr('m_36089d2ca3ff') }}</strong>
            <p>{{ tr('m_672b4d495cc6') }}<code>DATABASE_URL</code> / <code>STORE_PATH</code>{{ tr('m_3fe8ea8f9d28') }}</p>
            <div class="setup-facts">
              <div><span>{{ tr('m_ba40014ff496') }}</span><b>{{ driverLabel(status.storage.driver) }}</b></div>
              <div v-if="status.storage.driver === 'sqlite'"><span>{{ tr('m_39932f24fe11') }}</span><b>{{ status.storage.path || '—' }}</b></div>
              <template v-else>
                <div><span>{{ tr('m_e87d9f23a3f5') }}</span><b>{{ status.storage.postgres?.host || status.storage.url || '—' }}</b></div>
                <div v-if="status.storage.postgres?.database"><span>{{ tr('m_263200d7d529') }}</span><b>{{ status.storage.postgres.database }}</b></div>
                <div v-if="status.storage.postgres?.user"><span>{{ tr('m_0d0e1a86b3aa') }}</span><b>{{ status.storage.postgres.user }}</b></div>
              </template>
            </div>
          </div>

          <template v-else>
            <div class="driver-grid">
              <button
                type="button"
                class="driver-card"
                :class="{ selected: driver === 'sqlite' }"
                @click="chooseDriver('sqlite')"
              >
                <span class="driver-name">SQLite</span>
                <span class="driver-desc">{{ tr('m_70c9a7386f69') }}</span>
              </button>
              <button
                type="button"
                class="driver-card"
                :class="{ selected: driver === 'postgres' }"
                @click="chooseDriver('postgres')"
              >
                <span class="driver-name">PostgreSQL</span>
                <span class="driver-desc">{{ tr('m_9601abfcbd7e') }}</span>
              </button>
            </div>

            <div v-if="driver === 'sqlite'" class="setup-form">
              <div class="field">
                <label class="field-label" for="setup-sqlite">{{ tr('m_8997667013c2') }}</label>
                <input id="setup-sqlite" v-model="sqlitePath" class="vercel-input mono" placeholder="data/tunnel-manager.db" spellcheck="false" />
                <small class="setup-hint">{{ tr('m_9a9424715d8f') }}</small>
              </div>
            </div>

            <div v-else class="setup-form">
              <div class="field-row">
                <div class="field grow">
                  <label class="field-label" for="setup-pg-host">{{ tr('m_e87d9f23a3f5') }}</label>
                  <input id="setup-pg-host" v-model="postgres.host" class="vercel-input mono" placeholder="127.0.0.1" spellcheck="false" />
                </div>
                <div class="field port">
                  <label class="field-label" for="setup-pg-port">{{ tr('m_e71ac32b544b') }}</label>
                  <input id="setup-pg-port" v-model.number="postgres.port" class="vercel-input mono" inputmode="numeric" placeholder="5432" />
                </div>
              </div>
              <div class="field">
                <label class="field-label" for="setup-pg-db">{{ tr('m_acd97cee58a9') }}</label>
                <input id="setup-pg-db" v-model="postgres.database" class="vercel-input mono" placeholder="tunnel_manager" spellcheck="false" />
              </div>
              <div class="field-row">
                <div class="field grow">
                  <label class="field-label" for="setup-pg-user">{{ tr('m_1a3f0617d6de') }}</label>
                  <input id="setup-pg-user" v-model="postgres.user" class="vercel-input mono" placeholder="postgres" spellcheck="false" />
                </div>
                <div class="field grow">
                  <label class="field-label" for="setup-pg-pass">{{ tr('m_a621ab606db2') }}</label>
                  <input
                    id="setup-pg-pass"
                    v-model="postgres.password"
                    type="password"
                    class="vercel-input mono"
                    :placeholder="status.has_secret ? tr('m_645ce9c168df') : tr('m_073519025471')"
                    autocomplete="new-password"
                  />
                </div>
              </div>
              <div class="field">
                <label class="field-label" for="setup-pg-ssl">{{ tr('m_3a2408fd44b0') }}</label>
                <select id="setup-pg-ssl" v-model="postgres.ssl_mode" class="vercel-input">
                  <option v-for="mode in sslModes" :key="mode" :value="mode">{{ mode }}</option>
                </select>
              </div>
              <small class="setup-hint">{{ tr('m_9c2187c57ae0') }} <code>/var/run/postgresql</code>{{ tr('m_2bb2038bfd36') }}</small>
            </div>

            <div v-if="testMessage" class="setup-result" :class="testedKey === payloadKey ? 'ok' : 'bad'">{{ testMessage }}</div>
          </template>

          <div class="setup-actions">
            <button v-if="!status.env_locked" type="button" class="btn btn-secondary" :disabled="testing || submitting" @click="handleTest">
              <span v-if="testing" class="spinner"></span>
              {{ testing ? tr('m_105a9082c346') : tr('m_0a98a48eaaac') }}
            </button>
            <button type="button" class="btn btn-primary" :disabled="testing || submitting" @click="goAdminStep">
              {{ tr('m_acfc4e74a650') }}
            </button>
          </div>
        </section>

        <section v-else class="setup-section">
          <div class="setup-form">
            <div class="field">
              <label class="field-label" for="setup-admin">{{ tr('m_ef3ad87738fc') }}</label>
              <input id="setup-admin" v-model="adminUsername" class="vercel-input" placeholder="admin" autocomplete="username" spellcheck="false" />
            </div>
            <div class="field">
              <label class="field-label" for="setup-pass">{{ tr('m_c19091521d71') }}</label>
              <input id="setup-pass" v-model="adminPassword" type="password" class="vercel-input" :placeholder="tr('m_041568c2db7a')" autocomplete="new-password" />
            </div>
            <div class="field">
              <label class="field-label" for="setup-pass2">{{ tr('m_81b17bc0c362') }}</label>
              <input id="setup-pass2" v-model="adminConfirm" type="password" class="vercel-input" :placeholder="tr('m_f725268ab2d3')" autocomplete="new-password" />
            </div>
            <small class="setup-hint">{{ tr('m_6d6369cd0367') }}</small>
          </div>

          <div v-if="error" class="setup-result bad" role="alert">{{ error }}</div>

          <div class="setup-actions">
            <button type="button" class="btn btn-secondary" :disabled="submitting" @click="step = 1">{{ tr('m_da336fdc0dbd') }}</button>
            <button type="button" class="btn btn-primary" :disabled="submitting" @click="handleComplete">
              <span v-if="submitting" class="spinner"></span>
              {{ submitting ? tr('m_19658d9fa9aa') : tr('m_d505339b3422') }}
            </button>
          </div>
        </section>
      </template>

      <div v-else-if="!restarting" class="setup-loading">
        <span>{{ tr('m_06fcd85c7528') }}</span>
        <button type="button" class="btn btn-secondary" @click="load()">{{ tr('m_b8784c8dd563') }}</button>
      </div>
    </div>

    <div v-if="restarting" class="setup-page setup-overlay">
      <div class="setup-card">
        <div class="setup-loading">
          <span v-if="!restartFailed" class="spinner"></span>
          <strong>{{ tr('m_86fa46bbb30e') }}</strong>
          <p class="setup-subtitle">
            {{ restartFailed ? tr('m_672113e00a33') : slowRestart ? tr('m_41fd7067f4bb') : tr('m_f03d8379b597') }}
          </p>
          <button v-if="restartFailed" type="button" class="btn btn-primary" @click="router.replace('/login')">{{ tr('m_093d5fcb7597') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { t as tr } from '../i18n'

import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { completeSetup, setupStatus, testSetupStorage, waitForPanelRestart, type SetupDriver, type SetupStoragePayload } from '../api/setup'

const router = useRouter()

const status = ref<Awaited<ReturnType<typeof setupStatus>>>(null)
const loading = ref(true)
const step = ref(1)
const driver = ref<SetupDriver>('sqlite')
const sqlitePath = ref('')
const postgres = reactive({ host: '127.0.0.1', port: 5432, database: '', user: '', password: '', ssl_mode: 'disable' })
const adminUsername = ref('admin')
const adminPassword = ref('')
const adminConfirm = ref('')
const testing = ref(false)
// The connection is only proven for the values that were sent, so the key of
// the form is remembered and a later edit asks for another test.
const testedKey = ref('')
const testMessage = ref('')
const submitting = ref(false)
const error = ref('')
const restarting = ref(false)
const restartFailed = ref(false)
const slowRestart = ref(false)

const sslModes = ['disable', 'allow', 'prefer', 'require', 'verify-ca', 'verify-full']

// 环境变量接管时只创建管理员账户，表单被只读摘要取代。
const envLocked = computed(() => !!status.value?.env_locked)

function driverLabel(value: SetupDriver) {
  return value === 'postgres' ? 'PostgreSQL' : 'SQLite'
}

function chooseDriver(next: SetupDriver) {
  driver.value = next
  testMessage.value = ''
}

async function load() {
  loading.value = true
  const state = await setupStatus(true)
  status.value = state
  if (state) {
    driver.value = state.storage.driver
    sqlitePath.value = state.storage.path || ''
    const pg = state.storage.postgres
    if (pg) {
      postgres.host = pg.host || postgres.host
      postgres.port = pg.port || postgres.port
      postgres.database = pg.database || ''
      postgres.user = pg.user || ''
      postgres.ssl_mode = pg.ssl_mode || 'disable'
    }
  }
  loading.value = false
}

function payload(): SetupStoragePayload {
  if (driver.value === 'postgres') {
    return {
      driver: 'postgres',
      host: postgres.host.trim(),
      port: postgres.port || 5432,
      database: postgres.database.trim(),
      user: postgres.user.trim(),
      password: postgres.password,
      ssl_mode: postgres.ssl_mode,
    }
  }
  return { driver: 'sqlite', path: sqlitePath.value.trim() }
}

// payloadKey identifies the current form so a tested connection is only
// trusted while the fields stay untouched.
const payloadKey = computed(() => JSON.stringify(payload()))

async function handleTest() {
  if (testing.value) return
  testing.value = true
  testMessage.value = ''
  try {
    const res = await testSetupStorage(payload())
    testedKey.value = res.ok ? payloadKey.value : ''
    testMessage.value = res.ok ? tr('m_4a19cb8a31d7') : res.error || tr('m_89c4766afc85')
  } catch (e: any) {
    testedKey.value = ''
    testMessage.value = e?.response?.data?.error || e?.message || tr('m_89c4766afc85')
  } finally {
    testing.value = false
  }
}

function goAdminStep() {
  error.value = ''
  if (!envLocked.value && testedKey.value !== payloadKey.value) {
    testMessage.value = tr('m_7db039ac76c4')
    return
  }
  step.value = 2
}

async function handleComplete() {
  if (submitting.value) return
  if (!adminUsername.value.trim()) {
    error.value = tr('m_7a790a1d27f0')
    return
  }
  if (adminPassword.value.length < 6) {
    error.value = tr('m_ac31ae58b658')
    return
  }
  if (adminPassword.value !== adminConfirm.value) {
    error.value = tr('m_58807745708e')
    return
  }
  submitting.value = true
  error.value = ''
  try {
    const res = await completeSetup(payload(), adminUsername.value.trim(), adminPassword.value)
    if (!res.ok) {
      error.value = res.error || tr('m_a8581cc00ff3')
      return
    }
    await startPanel()
  } catch (e: any) {
    error.value = e?.response?.data?.error || e?.message || tr('m_a8581cc00ff3')
  } finally {
    submitting.value = false
  }
}

// 安装完成后服务端会重新拉起进程。以「安装接口消失」为准等它起来，再把
// 操作者送到登录页；等待期间顺手提示一下启动偏慢。
async function startPanel() {
  restarting.value = true
  restartFailed.value = false
  slowRestart.value = false
  const slowTimer = window.setTimeout(() => { slowRestart.value = true }, 10000)
  const alive = await waitForPanelRestart()
  window.clearTimeout(slowTimer)
  restartFailed.value = !alive
  router.replace('/login')
}

watch(payloadKey, () => { testMessage.value = '' })

onMounted(load)
</script>

<style scoped>
.setup-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-lg);
  background: var(--color-canvas-soft);
  box-sizing: border-box;
}

.setup-overlay { position: fixed; inset: 0; z-index: 40; }

.setup-card {
  width: 100%;
  max-width: 560px;
  background: var(--color-canvas-raised);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  padding: var(--spacing-2xl);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
}

.setup-head { text-align: center; margin-bottom: var(--spacing-xl); }

.logo-mark {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  background: var(--color-link);
  color: #fff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-bottom: var(--spacing-md);
}

.setup-title { font-size: 22px; font-weight: 600; color: var(--color-ink); margin: 0 0 6px; }
.setup-subtitle { font-size: 14px; color: var(--color-mute); margin: 0; }

.setup-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-xl) 0;
  color: var(--color-mute);
  font-size: 14px;
}

.setup-steps {
  display: flex;
  gap: var(--spacing-lg);
  list-style: none;
  margin: 0 0 var(--spacing-lg);
  padding: 0;
  font-size: 13px;
  color: var(--color-mute);
}

.setup-steps li { display: flex; align-items: center; gap: 8px; }
.setup-steps em {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 1px solid var(--color-hairline);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-style: normal;
  font-size: 12px;
}
.setup-steps li.active { color: var(--color-ink); font-weight: 600; }
.setup-steps li.active em, .setup-steps li.done em { background: var(--color-ink); border-color: var(--color-ink); color: var(--color-canvas-raised); }

.setup-section { display: flex; flex-direction: column; gap: var(--spacing-lg); }

.driver-grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--spacing-md); }

.driver-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
  text-align: left;
  padding: var(--spacing-md);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  background: var(--color-canvas-raised);
  cursor: pointer;
  transition: border-color 120ms ease, background-color 120ms ease;
}
.driver-card:hover { border-color: var(--color-hairline-strong); }
.driver-card.selected { border-color: var(--color-ink); background: var(--color-canvas-soft); }
.driver-name { font-size: 14px; font-weight: 600; color: var(--color-ink); font-family: var(--font-mono); }
.driver-desc { font-size: 12px; color: var(--color-mute); line-height: 1.5; }

.setup-form { display: flex; flex-direction: column; gap: var(--spacing-md); }
.field-row { display: flex; gap: var(--spacing-md); align-items: flex-end; }
.field.grow { flex: 1; }
.field.port { width: 110px; }
.mono { font-family: var(--font-mono); }
.setup-hint { font-size: 12px; color: var(--color-mute); }

.setup-note {
  padding: var(--spacing-md);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  background: var(--color-canvas-soft);
  font-size: 13px;
  color: var(--color-body);
}
.setup-note strong { display: block; color: var(--color-ink); margin-bottom: 4px; }
.setup-note p { margin: 0 0 var(--spacing-sm); line-height: 1.6; }
.setup-note code { font-family: var(--font-mono); font-size: 12px; }

.setup-facts { display: flex; flex-direction: column; gap: 6px; }
.setup-facts div { display: flex; gap: var(--spacing-sm); }
.setup-facts span { width: 60px; color: var(--color-mute); }
.setup-facts b { font-family: var(--font-mono); font-weight: 500; color: var(--color-ink); word-break: break-all; }

.setup-result {
  padding: 8px 10px;
  border-radius: var(--radius-md);
  font-size: 13px;
}
.setup-result.ok { color: var(--color-status-healthy-text); background: var(--color-status-healthy-bg); border: 1px solid var(--color-status-healthy-border); }
.setup-result.bad { color: var(--color-status-down-text); background: var(--color-status-down-bg); border: 1px solid var(--color-status-down-border); }

.setup-actions { display: flex; justify-content: flex-end; gap: var(--spacing-sm); }

.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid color-mix(in srgb, currentColor 26%, transparent);
  border-top-color: currentColor;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@media (max-width: 560px) {
  .driver-grid { grid-template-columns: 1fr; }
  .setup-card { padding: var(--spacing-xl); }
}
</style>
