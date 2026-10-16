<template>
  <div class="oidc-admin">
    <div class="settings-card">
      <div class="settings-card-header">
        <div><h3 class="settings-card-title">OIDC 身份认证</h3><p class="settings-card-desc">添加通用身份提供方。首次登录自动注册，已有账号按已验证邮箱关联。</p></div>
        <button type="button" class="btn btn-secondary" :disabled="busy" @click="newProvider">添加接入方</button>
      </div>
      <p v-if="error" class="oidc-error" role="alert">{{ error }}</p>
      <p v-if="notice" class="field-help" role="status">{{ notice }}</p>
      <p v-if="loading" class="field-help" role="status">正在读取接入方...</p>
      <p v-else-if="!providers.length" class="field-help">尚未添加 OIDC 接入方。</p>
      <ul v-else class="oidc-provider-list">
        <li v-for="provider in providers" :key="provider.id">
          <div class="oidc-provider-info"><strong>{{ provider.name }}</strong><code>{{ provider.issuer }}</code><span class="field-help">{{ provider.enabled ? '已启用' : '已停用' }} · 排序 {{ provider.order }}</span></div>
          <div class="oidc-actions"><button type="button" class="btn btn-secondary" :disabled="busy" @click="editProvider(provider)">编辑</button><button type="button" class="btn btn-ghost" :disabled="busy" @click="removeProvider(provider)">删除</button></div>
        </li>
      </ul>
    </div>
    <form v-if="editing" class="settings-card oidc-provider-form" @submit.prevent="save">
      <h3 class="settings-card-title">{{ editingID ? '编辑接入方' : '添加接入方' }}</h3>
      <div class="oidc-fields">
        <label class="field"><span class="field-label">显示名称</span><input v-model="form.name" class="vercel-input" maxlength="80" required placeholder="如：公司统一登录" /></label>
        <label class="field"><span class="field-label">Issuer URL</span><input v-model="form.issuer" class="vercel-input" type="url" required placeholder="https://id.example.com/realms/company" /><span class="field-help">填写发行者地址，系统自动发现认证端点。</span></label>
        <label class="field"><span class="field-label">Client ID</span><input v-model="form.client_id" class="vercel-input" required autocomplete="off" /></label>
        <label class="field"><span class="field-label">Client Secret</span><input v-model="form.client_secret" class="vercel-input" type="password" autocomplete="new-password" :placeholder="hasSecret ? '已设置，留空保持不变' : '机密客户端填写，公共客户端可留空'" /><span class="field-help">密钥加密保存，读取配置时不会返回。</span></label>
        <label v-if="hasSecret" class="oidc-check"><input v-model="form.clear_secret" type="checkbox" /> 清除原密钥，改用公共客户端</label>
        <label class="field"><span class="field-label">回调地址</span><input v-model="form.redirect_uri" class="vercel-input" type="url" required /><span class="field-help">复制此地址到身份提供方的允许回调地址配置。</span></label>
        <label class="field"><span class="field-label">Scopes</span><input v-model="form.scopes" class="vercel-input" required /><span class="field-help">必须包含 openid，按空格分隔。</span></label>
        <label class="field"><span class="field-label">排序</span><input v-model.number="form.order" class="vercel-input" type="number" min="0" max="9999" required /></label>
        <label class="oidc-check"><input v-model="form.enabled" type="checkbox" /> 启用此接入方</label>
      </div>
      <div class="form-actions"><button type="submit" class="btn btn-primary" :disabled="busy">{{ busy ? '保存中...' : '保存' }}</button><button type="button" class="btn btn-secondary" :disabled="busy" @click="copyCallback">复制回调地址</button><button type="button" class="btn btn-ghost" :disabled="busy" @click="editing = false">取消</button></div>
      <p class="field-help">启用后允许该接入方用户自动注册，不受密码注册开关和邀请码限制；新用户沿用本站邮箱验证设置。</p>
    </form>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useDialog } from 'naive-ui'
import { listAdminOIDCProviders, saveOIDCProvider, deleteOIDCProvider, type OIDCProvider, type OIDCProviderInput } from '../api/oidc'

const dialog = useDialog()
const providers = ref<OIDCProvider[]>([])
const loading = ref(false)
const busy = ref(false)
const editing = ref(false)
const editingID = ref('')
const hasSecret = ref(false)
const error = ref('')
const notice = ref('')
function defaults(): OIDCProviderInput { return { name: '', issuer: '', client_id: '', client_secret: '', redirect_uri: window.location.origin + '/api/auth/oidc/callback', scopes: 'openid profile email', enabled: false, order: 0, clear_secret: false } }
const form = reactive<OIDCProviderInput>(defaults())
async function load() {
  loading.value = true
  try { providers.value = (await listAdminOIDCProviders()).data }
  catch (e: any) { error.value = e.response?.data?.error || '接入方读取失败' }
  finally { loading.value = false }
}
function newProvider() { Object.assign(form, defaults()); editingID.value = ''; hasSecret.value = false; editing.value = true; error.value = ''; notice.value = '' }
function editProvider(provider: OIDCProvider) { Object.assign(form, defaults(), provider, { client_secret: '', clear_secret: false }); editingID.value = provider.id; hasSecret.value = provider.has_client_secret; editing.value = true; error.value = ''; notice.value = '' }
async function save() {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    const { data } = await saveOIDCProvider(editingID.value, { name: form.name.trim(), issuer: form.issuer.trim(), client_id: form.client_id.trim(), client_secret: form.client_secret, redirect_uri: form.redirect_uri.trim(), scopes: form.scopes.trim(), enabled: form.enabled, order: form.order, clear_secret: form.clear_secret })
    form.client_secret = ''; editingID.value = data.id; hasSecret.value = data.has_client_secret
    await load(); editing.value = false; notice.value = 'OIDC 接入方已保存'
  } catch (e: any) { error.value = e.response?.data?.error || '保存失败' }
  finally { busy.value = false }
}
function removeProvider(provider: OIDCProvider) {
  dialog.warning({ title: '删除 OIDC 接入方', content: `删除“${provider.name}”？仍有关联账号时需先停用并解绑。`, positiveText: '删除', negativeText: '取消', onPositiveClick: async () => {
    busy.value = true; error.value = ''; notice.value = ''
    try { await deleteOIDCProvider(provider.id); if (editingID.value === provider.id) editing.value = false; await load(); notice.value = '接入方已删除' }
    catch (e: any) { error.value = e.response?.data?.error || '删除失败' }
    finally { busy.value = false }
  } })
}
async function copyCallback() { try { await navigator.clipboard.writeText(form.redirect_uri); notice.value = '回调地址已复制' } catch { error.value = '复制失败，请从回调地址输入框手动复制' } }
onMounted(load)
</script>

<style scoped>
.oidc-admin { display: flex; min-width: 0; flex-direction: column; gap: var(--spacing-lg); }
.oidc-provider-list { padding: 0; margin: 0; list-style: none; }
.oidc-provider-list li { display: flex; align-items: center; justify-content: space-between; gap: var(--spacing-md); padding: var(--spacing-md) 0; border-top: 1px solid var(--color-hairline); }
.oidc-provider-info { display: flex; min-width: 0; flex-direction: column; gap: var(--spacing-xs); }
.oidc-provider-info code { font-family: var(--font-mono); font-size: 12px; color: var(--color-mute); overflow-wrap: anywhere; }
.oidc-actions { display: flex; gap: var(--spacing-sm); flex-shrink: 0; }
.oidc-fields { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: var(--spacing-lg); margin-top: var(--spacing-lg); }
.field { display: flex; min-width: 0; flex-direction: column; gap: var(--spacing-xs); }
.vercel-input { width: 100%; min-width: 0; box-sizing: border-box; }
.oidc-check { display: flex; align-items: center; gap: var(--spacing-sm); font-size: 14px; }
.oidc-error { color: var(--color-error); overflow-wrap: anywhere; }
@media (max-width: 640px) { .oidc-fields { grid-template-columns: 1fr; } .oidc-provider-list li { align-items: stretch; flex-direction: column; } .form-actions { flex-wrap: wrap; } }
</style>
