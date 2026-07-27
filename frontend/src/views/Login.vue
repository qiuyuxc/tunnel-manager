<template>
  <div class="login-page">
    <div class="login-card" :class="{ 'login-card-enter': mounted, 'login-card-shake': shaking }">
      <div class="login-logo">
          <span class="logo-mark" aria-hidden="true">
            <svg width="18" height="18" viewBox="0 0 76 76" fill="none">
              <path d="M49 26H27v24l22-24z" fill="currentColor"/>
              <path d="M38 38L27 50h22L38 38z" fill="currentColor" fill-opacity="0.42"/>
            </svg>
          </span>
      </div>
      <h1 class="login-title">Tunnel Manager</h1>
      <p class="login-subtitle">登录以继续</p>

      <form class="login-form" @submit.prevent="handleLogin">
        <div class="field">
          <label class="field-label">用户名</label>
          <input
            v-model="form.username"
            type="text"
            placeholder="admin"
            class="vercel-input"
            :class="{ 'input-error': error }"
            autocomplete="username"
          />
        </div>
        <div class="field">
          <label class="field-label">密码</label>
          <input
            v-model="form.password"
            type="password"
            placeholder="••••••••"
            class="vercel-input"
            :class="{ 'input-error': error }"
            autocomplete="current-password"
          />
        </div>

        <div v-if="error" class="login-error">{{ error }}</div>

        <button type="submit" class="btn btn-primary login-btn" :disabled="loading">
          <span v-if="loading" class="spinner"></span>
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { login as loginApi } from '../api'
import { useConfigStore } from '../stores/config'

const router = useRouter()
const store = useConfigStore()

const form = reactive({ username: '', password: '' })
const loading = ref(false)
const error = ref('')
const mounted = ref(false)
const shaking = ref(false)

onMounted(() => {
  requestAnimationFrame(() => { mounted.value = true })
})

async function handleLogin() {
  if (!form.username || !form.password) {
    error.value = '请输入用户名和密码'
    triggerShake()
    return
  }
  loading.value = true
  error.value = ''
  try {
    const { data } = await loginApi(form.username, form.password)
    store.setAuth(data.token, data.username)
    router.push('/')
  } catch (e: any) {
    if (e.response?.status === 401) {
      error.value = '用户名或密码错误'
    } else {
      error.value = '登录失败: ' + (e.response?.data?.error || e.message)
    }
    triggerShake()
  } finally {
    loading.value = false
  }
}

function triggerShake() {
  shaking.value = true
  setTimeout(() => { shaking.value = false }, 500)
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-lg);
  background:
    radial-gradient(circle at 18% 12%, color-mix(in srgb, var(--color-link) 12%, transparent), transparent 28%),
    linear-gradient(135deg, var(--color-canvas-soft), var(--color-canvas));
  box-sizing: border-box;
}

.login-card {
  width: 100%;
  max-width: 392px;
  background: var(--color-canvas);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  padding: var(--spacing-2xl);
  text-align: center;
  box-shadow: 0 24px 60px rgba(58, 47, 34, 0.12);
  opacity: 0;
  transform: translateY(16px) scale(0.98);
  transition: opacity 420ms ease-out, transform 420ms cubic-bezier(0.16, 1, 0.3, 1);
}

.login-card-enter {
  opacity: 1;
  transform: translateY(0) scale(1);
}

.login-card-shake { animation: shake 0.45s ease-out; }

.login-logo {
  margin-bottom: var(--spacing-md);
  display: flex;
  justify-content: center;
}

.logo-mark {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-lg);
  background: var(--color-ink);
  color: var(--color-canvas);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.login-title {
  font-family: var(--font-display);
  font-size: 25px;
  font-weight: 600;
  line-height: 1.15;
  color: var(--color-ink);
  margin: 0 0 6px;
}

.login-subtitle {
  font-size: 14px;
  color: var(--color-mute);
  margin: 0 0 var(--spacing-xl) 0;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
  text-align: left;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field-label {
  font-size: 13px;
  font-weight: 700;
  color: var(--color-ink);
}

.login-error {
  font-size: 14px;
  color: var(--color-result-error-text);
  text-align: center;
  padding: 9px 10px;
  background: var(--color-result-error-bg);
  border: 1px solid var(--color-result-error-border);
  border-radius: var(--radius-md);
  animation: fadeIn 180ms ease-out;
}

.login-btn {
  width: 100%;
  justify-content: center;
  margin-top: var(--spacing-xs);
}

.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid color-mix(in srgb, currentColor 26%, transparent);
  border-top-color: currentColor;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin { to { transform: rotate(360deg); } }
@keyframes shake {
  0%, 100% { transform: translateX(0); }
  20% { transform: translateX(-4px); }
  40% { transform: translateX(4px); }
  60% { transform: translateX(-3px); }
  80% { transform: translateX(3px); }
}
@keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }

@media (max-width: 480px) {
  .login-card { padding: var(--spacing-xl) var(--spacing-lg); }
}
</style>