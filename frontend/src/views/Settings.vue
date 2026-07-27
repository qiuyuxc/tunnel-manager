<template>
  <div class="page-container" style="padding-top: 0;">
    <div class="page-header">
      <router-link to="/" class="back-link">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>
        返回控制面板
      </router-link>
      <h2>全局设置</h2>
      <p>管理全局配置参数</p>
    </div>

    <div class="settings-list section">
      <div class="settings-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.08s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">全局优选 CNAME</div>
          <div class="settings-card-desc">该 CNAME 将用于主域名的 DNS 解析指向，以实现优选 IP 加速。</div>
        </div>
        <div class="settings-input-row">
          <div class="input-wrapper">
            <input v-model="preferredCNAME" placeholder="cf.090227.xyz" class="vercel-input" />
          </div>
          <button class="btn btn-primary" :disabled="savingCNAME" @click="savePreferredCNAME">
            {{ savingCNAME ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>

      <div class="settings-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.16s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">回退源设置</div>
          <div class="settings-card-desc">设置 Custom Hostnames 的回退源（Fallback Origin），用于 SaaS 模块。</div>
        </div>
        <div class="settings-input-row">
          <div class="input-wrapper">
            <input v-model="fallbackDomain" placeholder="例如: fallback.169977.xyz" class="vercel-input" />
          </div>
          <button class="btn btn-secondary" :disabled="savingFallback" @click="saveFallback">
            {{ savingFallback ? '设置中...' : '设置' }}
          </button>
        </div>
      </div>

      <div class="settings-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.24s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">隧道 ID</div>
          <div class="settings-card-desc">当前锁定的 Cloudflare Tunnel ID。</div>
        </div>
        <div class="settings-input-row">
          <div class="input-wrapper">
            <input v-model="tunnelID" placeholder="隧道 ID" class="vercel-input" />
          </div>
          <button class="btn btn-primary" :disabled="savingTunnel" @click="saveTunnelID">
            {{ savingTunnel ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { setPreferredCNAME, setFallbackOrigin, setTunnelID } from '../api'
import { useConfigStore } from '../stores/config'

const message = useMessage()
const store = useConfigStore()
const config = store.config
const visible = ref(false)

const preferredCNAME = ref(config.preferred_cname)
const fallbackDomain = ref('')
const tunnelID = ref(config.tunnel_id)

const savingCNAME = ref(false)
const savingFallback = ref(false)
const savingTunnel = ref(false)

async function savePreferredCNAME() {
  savingCNAME.value = true
  try {
    await setPreferredCNAME(preferredCNAME.value)
    config.preferred_cname = preferredCNAME.value
    message.success('优选 CNAME 已更新')
  } catch (e: any) {
    message.error('保存失败: ' + (e.response?.data?.error || e.message))
  } finally {
    savingCNAME.value = false
  }
}

async function saveFallback() {
  savingFallback.value = true
  try {
    const { data } = await setFallbackOrigin(fallbackDomain.value)
    message.success(data.message || '回退源已设置')
  } catch (e: any) {
    message.error('设置失败: ' + (e.response?.data?.error || e.message))
  } finally {
    savingFallback.value = false
  }
}

async function saveTunnelID() {
  savingTunnel.value = true
  try {
    await setTunnelID(tunnelID.value)
    config.tunnel_id = tunnelID.value
    message.success('隧道 ID 已更新')
  } catch (e: any) {
    message.error('保存失败: ' + (e.response?.data?.error || e.message))
  } finally {
    savingTunnel.value = false
  }
}

onMounted(() => {
  store.fetchConfig()
  requestAnimationFrame(() => { visible.value = true })
})
</script>

<style scoped>
.page-header { margin-bottom: var(--spacing-lg); }
.section { margin-bottom: 0; }

.settings-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.settings-card {
  background: var(--color-canvas);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
  box-shadow: 0 1px 2px rgba(58, 47, 34, 0.05);
}
.settings-card:hover {
  border-color: var(--color-hairline-strong);
  box-shadow: 0 12px 28px rgba(58, 47, 34, 0.08);
}

.settings-card-header { margin-bottom: var(--spacing-md); }
.settings-card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-ink);
  margin-bottom: 4px;
}
.settings-card-desc {
  font-size: 14px;
  color: var(--color-mute);
  line-height: 1.65;
}

.settings-input-row {
  display: flex;
  gap: var(--spacing-sm);
  align-items: flex-start;
}

.input-wrapper { flex: 1; min-width: 0; }

@media (max-width: 768px) {
  .settings-input-row {
    flex-direction: column;
  }
  .settings-input-row .btn {
    width: 100%;
    max-width: 100%;
    justify-content: center;
  }
}
</style>