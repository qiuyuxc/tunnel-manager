<template>
  <section class="settings-card oidc-account" aria-labelledby="oidc-account-title">
    <div class="settings-card-header">
      <div><h3 id="oidc-account-title" class="settings-card-title">{{ tr('m_d1eb3be9948a') }}</h3>
      <p class="settings-card-desc">{{ tr('m_3ff70496ec7a') }}</p></div>
    </div>
    <p v-if="loading" class="field-help" role="status">{{ tr('m_8d80ae2393c8') }}</p>
    <p v-if="error" class="oidc-error" role="alert">{{ error }}</p>
    <button v-if="error && !account" type="button" class="btn btn-secondary" :disabled="busy || loading" @click="load">{{ tr('m_b8784c8dd563') }}</button>
    <p v-if="notice" class="field-help" role="status">{{ notice }}</p>
    <template v-if="account">
      <p v-if="!account.has_password" class="oidc-hint">{{ tr('m_75239914cb85') }}</p>
      <ul v-if="account.bindings.length" class="oidc-bindings">
        <li v-for="binding in account.bindings" :key="binding.provider_id">
          <div class="oidc-identity"><strong>{{ binding.name }}</strong><span>{{ binding.email || tr('m_40b94aa7b1c2') }}</span><small>{{ binding.enabled ? tr('m_af891bc406e5') : tr('m_c7eed2143af4') }}</small></div>
          <div class="oidc-actions">
            <button v-if="!account.has_password && binding.enabled" class="btn btn-secondary" type="button" :disabled="busy || !!passwordTicket" @click="start(binding.provider_id, 'password')">{{ tr('m_dcccc7b72be5') }}</button>
            <button class="btn btn-ghost danger-outline" type="button" :disabled="busy || !!passwordTicket" @click="confirmUnlink(binding)">{{ tr('m_3ba0a291df80') }}</button>
          </div>
        </li>
      </ul>
      <p v-else class="field-help">{{ tr('m_d9e3c6d08951') }}</p>
      <form v-if="passwordTicket" class="oidc-password" @submit.prevent="setPassword">
        <h4>{{ tr('m_f9597d80c822') }}</h4>
        <label class="field"><span class="field-label">{{ tr('m_515e9c7cf7b2') }}</span><input ref="passwordInput" v-model="password" class="vercel-input" type="password" autocomplete="new-password" minlength="6" maxlength="1024" required /></label>
        <label class="field"><span class="field-label">{{ tr('m_81b17bc0c362') }}</span><input v-model="passwordConfirm" class="vercel-input" type="password" autocomplete="new-password" minlength="6" maxlength="1024" required /></label>
        <div class="oidc-actions"><button class="btn btn-primary" type="submit" :disabled="busy || password.length < 6 || password !== passwordConfirm">{{ busy ? tr('m_917b1c1f18d0') : tr('m_0139e54b9887') }}</button><button class="btn btn-ghost" type="button" :disabled="busy" @click="cancelPassword">{{ tr('m_2cd0f3be8738') }}</button></div>
      </form>
      <label v-if="availableProviders.length && account.has_password" class="field oidc-bind-password"><span class="field-label">{{ tr('m_8d6b4cd237f4') }}</span><input v-model="bindingPassword" class="vercel-input" type="password" autocomplete="current-password" maxlength="1024" :disabled="busy" /></label>
      <div v-if="availableProviders.length" class="oidc-actions oidc-add" :aria-label="tr('m_a500a7ddaaf8')">
        <button v-for="provider in availableProviders" :key="provider.id" class="btn btn-secondary" type="button" :disabled="busy || !!passwordTicket || !account.has_password || !bindingPassword" @click="start(provider.id, 'bind')">{{ tr('m_3dd97f4d2766') }} {{ provider.name }}</button>
      </div>
      <p v-else-if="!account.bindings.length" class="field-help">{{ tr('m_52c452e7f83d') }}</p>
    </template>
  </section>
</template>

<script setup lang="ts">
import { t as tr } from '../i18n'

import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDialog } from 'naive-ui'
import { completeOIDC, getOIDCAccount, listOIDCProviders, setupOIDCPassword, startOIDCAccount, unlinkOIDC, type OIDCAccount, type OIDCBinding, type OIDCPublicProvider } from '../api/oidc'

const emit = defineEmits<{ passwordStatus: [hasPassword: boolean] }>()
const route = useRoute()
const router = useRouter()
const dialog = useDialog()
const account = ref<OIDCAccount | null>(null)
const providers = ref<OIDCPublicProvider[]>([])
const loading = ref(false)
const busy = ref(false)
const error = ref('')
const notice = ref('')
const passwordTicket = ref('')
const password = ref('')
const passwordConfirm = ref('')
const bindingPassword = ref('')
const passwordInput = ref<HTMLInputElement | null>(null)
let disposed = false
const availableProviders = computed(() => providers.value.filter(provider => !account.value?.bindings.some(binding => binding.provider_id === provider.id)))
const apiError = (e: any, fallback: string) => e.response?.data?.error || fallback

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [status, available] = await Promise.all([getOIDCAccount(), listOIDCProviders()])
    if (disposed) return
    account.value = status.data
    providers.value = available.data
    emit('passwordStatus', status.data.has_password)
  } catch (e: any) { if (!disposed) error.value = apiError(e, tr('m_6ec173b408c9')) }
  finally { if (!disposed) loading.value = false }
}

async function start(id: string, action: 'bind' | 'password') {
  if (busy.value) return
  busy.value = true
  error.value = ''
  notice.value = ''
  try { const { data } = await startOIDCAccount(id, action, action === 'bind' ? bindingPassword.value : ''); bindingPassword.value = ''; if (!disposed) window.location.assign(data.authorization_url) }
  catch (e: any) { if (!disposed) { error.value = apiError(e, tr('m_e3c9ccd519b4')); busy.value = false } }
}

function confirmUnlink(binding: OIDCBinding) {
  dialog.warning({ title: tr('m_0baea21e93bb'), content: tr('m_bf79777bd606', { p0: binding.name }), positiveText: tr('m_3ba0a291df80'), negativeText: tr('m_2cd0f3be8738'), positiveButtonProps: { type: 'error' }, onPositiveClick: async () => {
    busy.value = true
    error.value = ''
    notice.value = ''
    try { await unlinkOIDC(binding.provider_id); await load(); if (!disposed) notice.value = tr('m_29341d7210b9') }
    catch (e: any) { if (!disposed) error.value = apiError(e, tr('m_61dbace46644')) }
    finally { if (!disposed) busy.value = false }
  } })
}

function cancelPassword() { passwordTicket.value = ''; password.value = ''; passwordConfirm.value = '' }
async function setPassword() {
  if (busy.value || password.value !== passwordConfirm.value) return
  busy.value = true
  error.value = ''
  try {
    await setupOIDCPassword(passwordTicket.value, password.value)
    cancelPassword()
    await load()
    if (!disposed) notice.value = tr('m_c7370ce7f52b')
  } catch (e: any) { if (!disposed) error.value = apiError(e, tr('m_a4d5442c09d3')) }
  finally { if (!disposed) busy.value = false }
}

onMounted(async () => {
  const ticket = typeof route.query.oidc_ticket === 'string' ? route.query.oidc_ticket : ''
  const callbackError = typeof route.query.oidc_error === 'string' ? route.query.oidc_error : ''
  if (ticket || callbackError) {
    const query = { ...route.query }; delete query.oidc_ticket; delete query.oidc_error
    await router.replace({ path: route.path, query })
  }
  await load()
  if (disposed) return
  if (callbackError) error.value = callbackError
  if (!ticket) return
  busy.value = true
  try {
    const { data } = await completeOIDC(ticket)
    if (disposed) return
    if ('password_setup_required' in data) { passwordTicket.value = data.ticket; await nextTick(); passwordInput.value?.focus() }
    else if ('status' in data && data.status === 'ok') { await load(); if (!disposed) notice.value = tr('m_ac3adc200aae') }
    else error.value = tr('m_e2c65f6f95ba')
  } catch (e: any) { if (!disposed) error.value = apiError(e, tr('m_fc3abe4d0986')) }
  finally { if (!disposed) busy.value = false }
})
onBeforeUnmount(() => { disposed = true; bindingPassword.value = ''; cancelPassword() })
</script>

<style scoped>
.oidc-account { min-width: 0; }
.oidc-bindings { list-style: none; margin: 0; padding: 0; }
.oidc-bindings li { display: flex; align-items: center; justify-content: space-between; gap: var(--spacing-md); padding: var(--spacing-md) 0; border-top: 1px solid var(--color-hairline); }
.oidc-identity { display: flex; flex-direction: column; min-width: 0; gap: var(--spacing-xs); overflow-wrap: anywhere; }
.oidc-identity span, .oidc-identity small, .oidc-hint { color: var(--color-mute); }
.oidc-actions { display: flex; flex-wrap: wrap; gap: var(--spacing-sm); }
.oidc-actions .btn { white-space: normal; overflow-wrap: anywhere; }
.oidc-add { padding-top: var(--spacing-md); }
.oidc-bind-password { max-width: 480px; padding-top: var(--spacing-md); }
.oidc-password { display: flex; flex-direction: column; gap: var(--spacing-md); padding-top: var(--spacing-md); border-top: 1px solid var(--color-hairline); max-width: 480px; }
.oidc-password h4 { margin: 0; }
.field { display: flex; flex-direction: column; gap: var(--spacing-xs); }
.vercel-input { width: 100%; min-width: 0; box-sizing: border-box; }
.oidc-error { color: var(--color-error); overflow-wrap: anywhere; }
@media (max-width: 640px) { .oidc-bindings li { align-items: stretch; flex-direction: column; } }
</style>
