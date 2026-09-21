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
        <h1 class="setup-title">安装引导</h1>
        <p class="setup-subtitle">选择面板使用的数据库，并设置管理员账户</p>
      </header>

      <div v-if="loading" class="setup-loading">
        <span class="spinner"></span>
        <span>正在读取安装状态…</span>
      </div>

      <template v-else-if="status">
        <ol class="setup-steps">
          <li :class="{ active: step === 1, done: step > 1 }"><em>1</em>数据库</li>
          <li :class="{ active: step === 2 }"><em>2</em>管理员账户</li>
        </ol>

        <section v-if="step === 1" class="setup-section">
          <div v-if="status.env_locked" class="setup-note">
            <strong>数据库由环境变量提供</strong>
            <p>本次启动由环境变量（<code>DATABASE_URL</code> / <code>STORE_PATH</code>）指定存储，安装引导不再询问数据库类型，只创建管理员账户。</p>
            <div class="setup-facts">
              <div><span>类型</span><b>{{ driverLabel(status.storage.driver) }}</b></div>
              <div v-if="status.storage.driver === 'sqlite'"><span>文件</span><b>{{ status.storage.path || '—' }}</b></div>
              <template v-else>
                <div><span>主机</span><b>{{ status.storage.postgres?.host || status.storage.url || '—' }}</b></div>
                <div v-if="status.storage.postgres?.database"><span>数据库</span><b>{{ status.storage.postgres.database }}</b></div>
                <div v-if="status.storage.postgres?.user"><span>用户</span><b>{{ status.storage.postgres.user }}</b></div>
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
                <span class="driver-desc">单文件数据库，零依赖，适合单机部署</span>
              </button>
              <button
                type="button"
                class="driver-card"
                :class="{ selected: driver === 'postgres' }"
                @click="chooseDriver('postgres')"
              >
                <span class="driver-name">PostgreSQL</span>
                <span class="driver-desc">外部数据库，适合多实例或已有数据库运维</span>
              </button>
            </div>

            <div v-if="driver === 'sqlite'" class="setup-form">
              <div class="field">
                <label class="field-label" for="setup-sqlite">数据库文件路径</label>
                <input id="setup-sqlite" v-model="sqlitePath" class="vercel-input mono" placeholder="data/tunnel-manager.db" spellcheck="false" />
                <small class="setup-hint">相对于服务工作目录，目录会自动创建。</small>
              </div>
            </div>

            <div v-else class="setup-form">
              <div class="field-row">
                <div class="field grow">
                  <label class="field-label" for="setup-pg-host">主机</label>
                  <input id="setup-pg-host" v-model="postgres.host" class="vercel-input mono" placeholder="127.0.0.1" spellcheck="false" />
                </div>
                <div class="field port">
                  <label class="field-label" for="setup-pg-port">端口</label>
                  <input id="setup-pg-port" v-model.number="postgres.port" class="vercel-input mono" inputmode="numeric" placeholder="5432" />
                </div>
              </div>
              <div class="field">
                <label class="field-label" for="setup-pg-db">数据库名</label>
                <input id="setup-pg-db" v-model="postgres.database" class="vercel-input mono" placeholder="tunnel_manager" spellcheck="false" />
              </div>
              <div class="field-row">
                <div class="field grow">
                  <label class="field-label" for="setup-pg-user">用户名</label>
                  <input id="setup-pg-user" v-model="postgres.user" class="vercel-input mono" placeholder="postgres" spellcheck="false" />
                </div>
                <div class="field grow">
                  <label class="field-label" for="setup-pg-pass">密码</label>
                  <input
                    id="setup-pg-pass"
                    v-model="postgres.password"
                    type="password"
                    class="vercel-input mono"
                    :placeholder="status.has_secret ? '留空沿用已保存的密码' : '数据库密码'"
                    autocomplete="new-password"
                  />
                </div>
              </div>
              <div class="field">
                <label class="field-label" for="setup-pg-ssl">SSL 模式</label>
                <select id="setup-pg-ssl" v-model="postgres.ssl_mode" class="vercel-input">
                  <option v-for="mode in sslModes" :key="mode" :value="mode">{{ mode }}</option>
                </select>
              </div>
              <small class="setup-hint">主机可填域名或 IP；与数据库同机部署时也可填 Unix socket 目录（如 <code>/var/run/postgresql</code>）。目标数据库需已存在，且账户有建表权限。</small>
            </div>

            <div v-if="testMessage" class="setup-result" :class="testedKey === payloadKey ? 'ok' : 'bad'">{{ testMessage }}</div>
          </template>

          <div class="setup-actions">
            <button v-if="!status.env_locked" type="button" class="btn btn-secondary" :disabled="testing || submitting" @click="handleTest">
              <span v-if="testing" class="spinner"></span>
              {{ testing ? '测试中…' : '测试连接' }}
            </button>
            <button type="button" class="btn btn-primary" :disabled="testing || submitting" @click="goAdminStep">
              下一步
            </button>
          </div>
        </section>

        <section v-else class="setup-section">
          <div class="setup-form">
            <div class="field">
              <label class="field-label" for="setup-admin">管理员用户名</label>
              <input id="setup-admin" v-model="adminUsername" class="vercel-input" placeholder="admin" autocomplete="username" spellcheck="false" />
            </div>
            <div class="field">
              <label class="field-label" for="setup-pass">管理员密码</label>
              <input id="setup-pass" v-model="adminPassword" type="password" class="vercel-input" placeholder="至少 6 位" autocomplete="new-password" />
            </div>
            <div class="field">
              <label class="field-label" for="setup-pass2">确认密码</label>
              <input id="setup-pass2" v-model="adminConfirm" type="password" class="vercel-input" placeholder="再输入一次" autocomplete="new-password" />
            </div>
            <small class="setup-hint">密码只保存在本地数据库，不会打印到日志，请自行妥善保管。</small>
          </div>

          <div v-if="error" class="setup-result bad" role="alert">{{ error }}</div>

          <div class="setup-actions">
            <button type="button" class="btn btn-secondary" :disabled="submitting" @click="step = 1">上一步</button>
            <button type="button" class="btn btn-primary" :disabled="submitting" @click="handleComplete">
              <span v-if="submitting" class="spinner"></span>
              {{ submitting ? '安装中…' : '完成安装' }}
            </button>
          </div>
        </section>
      </template>

      <div v-else-if="!restarting" class="setup-loading">
        <span>无法连接后端服务。</span>
        <button type="button" class="btn btn-secondary" @click="load()">重试</button>
      </div>
    </div>

    <div v-if="restarting" class="setup-page setup-overlay">
      <div class="setup-card">
        <div class="setup-loading">
          <span v-if="!restartFailed" class="spinner"></span>
          <strong>安装完成，正在启动面板…</strong>
          <p class="setup-subtitle">
            {{ restartFailed ? '已安装，但没等到面板响应。可以在日志里确认启动情况，或直接前往登录页。' : slowRestart ? '启动比平时慢一些，仍在等待服务响应…' : '服务重启后会自动跳转到登录页。' }}
          </p>
          <button v-if="restartFailed" type="button" class="btn btn-primary" @click="router.replace('/login')">前往登录页</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
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
    testMessage.value = res.ok ? '连接成功' : res.error || '连接失败'
  } catch (e: any) {
    testedKey.value = ''
    testMessage.value = e?.response?.data?.error || e?.message || '连接失败'
  } finally {
    testing.value = false
  }
}

function goAdminStep() {
  error.value = ''
  if (!envLocked.value && testedKey.value !== payloadKey.value) {
    testMessage.value = '请先测试连接'
    return
  }
  step.value = 2
}

async function handleComplete() {
  if (submitting.value) return
  if (!adminUsername.value.trim()) {
    error.value = '请填写管理员用户名'
    return
  }
  if (adminPassword.value.length < 6) {
    error.value = '密码至少 6 位'
    return
  }
  if (adminPassword.value !== adminConfirm.value) {
    error.value = '两次输入的密码不一致'
    return
  }
  submitting.value = true
  error.value = ''
  try {
    const res = await completeSetup(payload(), adminUsername.value.trim(), adminPassword.value)
    if (!res.ok) {
      error.value = res.error || '安装失败'
      return
    }
    await startPanel()
  } catch (e: any) {
    error.value = e?.response?.data?.error || e?.message || '安装失败'
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
