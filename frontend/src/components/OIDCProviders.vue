<template>
  <div class="oidc-admin">
    <div class="settings-card">
      <div class="settings-card-header">
        <div><h3 class="settings-card-title">{{ tr('m_87619ee9ddbe') }}</h3><p class="settings-card-desc">{{ tr('m_2d8122d8db5b') }}</p></div>
        <button type="button" class="btn btn-secondary" :disabled="busy" @click="newProvider">{{ tr('m_7fac60538e4c') }}</button>
      </div>
      <p v-if="error" class="oidc-error" role="alert">{{ error }}</p>
      <p v-if="notice" class="field-help" role="status">{{ notice }}</p>
      <p v-if="loading" class="field-help" role="status">{{ tr('m_ed7eebd11ae8') }}</p>
      <p v-else-if="!providers.length" class="field-help">{{ tr('m_a912583729ca') }}</p>
      <ul v-else class="oidc-provider-list">
        <li v-for="provider in providers" :key="provider.id">
          <div class="oidc-provider-info"><strong>{{ provider.name }}</strong><code>{{ provider.issuer }}</code><span class="field-help">{{ provider.enabled ? tr('m_dfb802238b38') : tr('m_a8c3698b5b8c') }} {{ tr('m_7b57ee5fb610') }} {{ provider.order }}</span></div>
          <div class="oidc-actions"><button type="button" class="btn btn-secondary" :disabled="busy" @click="editProvider(provider)">{{ tr('m_051836569928') }}</button><button type="button" class="btn btn-ghost" :disabled="busy" @click="removeProvider(provider)">{{ tr('m_2f9daa828907') }}</button></div>
        </li>
      </ul>
    </div>
    <form v-if="editing" class="settings-card oidc-provider-form" @submit.prevent="save">
      <h3 class="settings-card-title">{{ editingID ? tr('m_1844ff89583e') : tr('m_7fac60538e4c') }}</h3>
      <div class="oidc-fields">
        <label class="field"><span class="field-label">{{ tr('m_a98585871c53') }}</span><input v-model="form.name" class="vercel-input" maxlength="80" required :placeholder="tr('m_e0ad94affff1')" /></label>
        <label class="field"><span class="field-label">Issuer URL</span><input v-model="form.issuer" class="vercel-input" type="url" required placeholder="https://id.example.com/realms/company" /><span class="field-help">{{ tr('m_982712ebed87') }}</span></label>
        <label class="field"><span class="field-label">Client ID</span><input v-model="form.client_id" class="vercel-input" required autocomplete="off" /></label>
        <label class="field"><span class="field-label">Client Secret</span><input v-model="form.client_secret" class="vercel-input" type="password" autocomplete="new-password" :placeholder="hasSecret ? tr('m_15b376320e0c') : tr('m_2b266f7e9eda')" /><span class="field-help">{{ tr('m_7d5ad81f26c5') }}</span></label>
        <label v-if="hasSecret" class="oidc-check"><input v-model="form.clear_secret" type="checkbox" /> {{ tr('m_2e76ff3f55a0') }}</label>
        <label class="field"><span class="field-label">{{ tr('m_9ca160d65985') }}</span><input v-model="form.redirect_uri" class="vercel-input" type="url" required /><span class="field-help">{{ tr('m_b7171daba293') }}</span></label>
        <label class="field"><span class="field-label">Scopes</span><input v-model="form.scopes" class="vercel-input" required /><span class="field-help">{{ tr('m_2e4e44079324') }}</span></label>
        <label class="field"><span class="field-label">{{ tr('m_a96c9a854190') }}</span><input v-model.number="form.order" class="vercel-input" type="number" min="0" max="9999" required /></label>
        <label class="oidc-check"><input v-model="form.enabled" type="checkbox" /> {{ tr('m_2cf816d1e4b5') }}</label>
      </div>
      <div class="form-actions"><button type="submit" class="btn btn-primary" :disabled="busy">{{ busy ? tr('m_917b1c1f18d0') : tr('m_a3030bf8f16d') }}</button><button type="button" class="btn btn-secondary" :disabled="busy" @click="copyCallback">{{ tr('m_1d9d7da8233f') }}</button><button type="button" class="btn btn-ghost" :disabled="busy" @click="editing = false">{{ tr('m_2cd0f3be8738') }}</button></div>
      <p class="field-help">{{ tr('m_14b74debf223') }}</p>
    </form>
  </div>
</template>

<script setup lang="ts">
import { t as tr } from '../i18n'

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
  catch (e: any) { error.value = e.response?.data?.error || tr('m_52c86532ad99') }
  finally { loading.value = false }
}
function newProvider() { Object.assign(form, defaults()); editingID.value = ''; hasSecret.value = false; editing.value = true; error.value = ''; notice.value = '' }
function editProvider(provider: OIDCProvider) { Object.assign(form, defaults(), provider, { client_secret: '', clear_secret: false }); editingID.value = provider.id; hasSecret.value = provider.has_client_secret; editing.value = true; error.value = ''; notice.value = '' }
async function save() {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    const { data } = await saveOIDCProvider(editingID.value, { name: form.name.trim(), issuer: form.issuer.trim(), client_id: form.client_id.trim(), client_secret: form.client_secret, redirect_uri: form.redirect_uri.trim(), scopes: form.scopes.trim(), enabled: form.enabled, order: form.order, clear_secret: form.clear_secret })
    form.client_secret = ''; editingID.value = data.id; hasSecret.value = data.has_client_secret
    await load(); editing.value = false; notice.value = tr('m_cd758cd9faeb')
  } catch (e: any) { error.value = e.response?.data?.error || tr('m_6309a3bb5ba4') }
  finally { busy.value = false }
}
function removeProvider(provider: OIDCProvider) {
  dialog.warning({ title: tr('m_a5e005fe5785'), content: tr('m_36f7c18ac9fd', { p0: provider.name }), positiveText: tr('m_2f9daa828907'), negativeText: tr('m_2cd0f3be8738'), onPositiveClick: async () => {
    busy.value = true; error.value = ''; notice.value = ''
    try { await deleteOIDCProvider(provider.id); if (editingID.value === provider.id) editing.value = false; await load(); notice.value = tr('m_6765098c31a5') }
    catch (e: any) { error.value = e.response?.data?.error || tr('m_c228558cf257') }
    finally { busy.value = false }
  } })
}
async function copyCallback() { try { await navigator.clipboard.writeText(form.redirect_uri); notice.value = tr('m_f36d6add2ed0') } catch { error.value = tr('m_4a6277670d51') } }
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
