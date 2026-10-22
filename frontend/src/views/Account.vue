<template>
  <div class="page-container account-page">
    <div class="page-header">
            <h2>{{ tr('m_a720b4000be6') }}</h2>
      <p>{{ tr('m_d86f5c72c33e') }}</p>
    </div>

    <div class="settings-list">
      <section class="settings-card profile-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0s' }" aria-labelledby="profile-title">
        <div class="profile-heading">
          <div>
            <p class="security-kicker">Personal profile</p>
            <h3 id="profile-title" class="settings-card-title">{{ tr('m_227eef9cd7b1') }}</h3>
            <p class="settings-card-desc">{{ tr('m_c9ce2f44a713') }}</p>
          </div>
        </div>

        <div class="profile-main">
          <div class="avatar-block">
            <div class="avatar-preview" aria-hidden="true">
              <img v-if="store.avatar" :src="store.avatar" :alt="tr('m_1841af00f45a')" />
              <span v-else>{{ avatarInitial }}</span>
            </div>
            <div class="avatar-actions">
              <button class="btn btn-secondary" type="button" :disabled="uploadingAvatar" @click="triggerAvatarUpload">
                {{ uploadingAvatar ? tr('m_593470c32d44') : tr('m_d7d2344d545d') }}
              </button>
              <button v-if="store.avatar" class="btn btn-ghost danger-outline" type="button" :disabled="uploadingAvatar" @click="removeAvatar">
                {{ tr('m_6135d4159e89') }}
              </button>
            </div>
            <input ref="avatarInput" type="file" accept="image/png,image/jpeg,image/gif,image/webp" class="avatar-input" @change="handleAvatarFile" />
            <span class="avatar-hint">{{ tr('m_78786baa366b') }}</span>
          </div>

          <div class="profile-fields">
            <div class="form-stack">
              <label class="field">
                <span class="field-label">{{ tr('m_bf3b4816f16e') }}</span>
                <input v-model="nickname" :placeholder="tr('m_9e366ced887c')" class="vercel-input" :maxlength="64" />
              </label>
              <p class="profile-meta">{{ tr('m_e4985caf0e6c') }} <strong>{{ store.username }}</strong></p>
              <div class="form-actions">
                <button class="btn btn-primary" :disabled="savingProfile" @click="saveProfile">
                  {{ savingProfile ? tr('m_917b1c1f18d0') : tr('m_c0a70fcb56f0') }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <div class="security-rule" aria-hidden="true"></div>

        <div class="email-section">
          <div class="email-heading">
            <div>
              <span class="email-status" :class="store.email ? 'bound' : 'unbound'">{{ store.email ? tr('m_f70c0007f30f') : tr('m_e026c6693dc5') }}</span>
              <h4 class="email-title">{{ tr('m_f8f1b032f460') }}</h4>
              <p v-if="store.email" class="email-current">{{ tr('m_cdd2259a6d69') }} <strong>{{ store.email }}</strong></p>
              <p v-else class="email-current">{{ tr('m_e6bddbeaa97b') }}</p>
            </div>
          </div>

          <form class="form-stack email-form" @submit.prevent="saveEmail">
            <div class="email-fields">
              <label class="field">
                <span class="field-label">{{ store.email ? tr('m_3067c39fa4b2') : tr('m_2e412e03837f') }}</span>
                <input v-model="newEmail" type="email" placeholder="name@example.com" class="vercel-input" />
              </label>
              <label class="field">
                <span class="field-label">{{ tr('m_ab48ee83149b') }}</span>
                <input v-model="emailPassword" type="password" :placeholder="tr('m_632da57acf53')" class="vercel-input" autocomplete="current-password" />
              </label>
            </div>
            <div class="form-actions">
              <button class="btn btn-secondary" type="submit" :disabled="savingEmail || !newEmail.trim() || !emailPassword">
                {{ savingEmail ? tr('m_917b1c1f18d0') : store.email ? tr('m_da0f5e37e88b') : tr('m_6d4b7dce1ee6') }}
              </button>
            </div>
          </form>
        </div>
      </section>

      <OIDCAccount @password-status="hasLocalPassword = $event" />

      <section class="settings-card security-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.04s' }" aria-labelledby="two-factor-title">
        <div class="security-heading">
          <div>
            <p class="security-kicker">Operations security</p>
            <h3 id="two-factor-title" class="settings-card-title">{{ tr('m_cb5e99d1a4d8') }}</h3>
            <p class="settings-card-desc">{{ tr('m_303aeceaf321') }}</p>
          </div>
          <div class="status-stamp" :class="`stamp-${statusStamp.tone}`" role="status" aria-live="polite">
            <span class="stamp-mark" aria-hidden="true"></span>
            <span>
              <small>2FA status</small>
              <strong>{{ statusStamp.label }}</strong>
            </span>
          </div>
        </div>

        <div class="security-rule" aria-hidden="true"></div>

        <div class="security-content" aria-live="polite">
          <div v-if="statusLoading" class="security-state compact-state">
            <span class="state-pulse" aria-hidden="true"></span>
            <div><strong>{{ tr('m_49823bd59ceb') }}</strong><p>{{ tr('m_b5ec8b23b9d0') }}</p></div>
          </div>

          <div v-else-if="statusError" class="security-state compact-state state-error">
            <div><strong>{{ tr('m_d38c903220bb') }}</strong><p>{{ statusError }}</p></div>
            <button class="btn btn-secondary" type="button" @click="loadTwoFactorStatus">{{ tr('m_b8784c8dd563') }}</button>
          </div>

          <div v-else-if="recoveryCodes.length" ref="recoveryPanel" class="recovery-panel" tabindex="-1" aria-labelledby="recovery-title">
            <div class="operation-header">
              <div>
                <span class="operation-index">{{ tr('m_265eb059dc03') }}</span>
                <h4 id="recovery-title">{{ tr('m_faa3e777003d') }}</h4>
              </div>
              <span class="operation-state state-critical">{{ tr('m_47687872a258') }}</span>
            </div>
            <p class="operation-copy">{{ tr('m_294245ce1fb4') }}</p>
            <div class="recovery-actions">
              <button class="btn btn-secondary" type="button" @click="copyRecoveryCodes">{{ tr('m_75c393923f87') }}</button>
              <button class="btn btn-secondary" type="button" @click="downloadRecoveryCodes">{{ tr('m_6b717f958200') }}</button>
            </div>
            <ol class="recovery-grid" :aria-label="tr('m_c5674f920ef6')">
              <li v-for="code in recoveryCodes" :key="code"><code>{{ code }}</code></li>
            </ol>
            <label class="completion-check">
              <input v-model="recoveryAcknowledged" type="checkbox" />
              <span>{{ tr('m_688057d713b7') }}</span>
            </label>
            <button class="btn btn-primary finish-button" type="button" :disabled="!recoveryAcknowledged" @click="finishRecovery">{{ tr('m_581139134404') }}</button>
          </div>

          <div v-else-if="setupSession" class="setup-panel">
            <div class="operation-header">
              <div>
                <span class="operation-index">Enrollment / 01</span>
                <h4>{{ tr('m_e1cea76538f3') }}</h4>
              </div>
              <span class="operation-state" :class="setupExpired ? 'state-critical' : 'state-live'">
                {{ setupExpired ? tr('m_7b1f68536f70') : tr('m_b8c53db09738', { p0: countdownLabel }) }}
              </span>
            </div>

            <div class="setup-grid">
              <div class="qr-column">
                <div class="qr-frame" :class="{ 'qr-error': qrError }">
                  <canvas ref="qrCanvas" role="img" :aria-label="tr('m_e2cb31d85549')">{{ tr('m_6b7b30837d07') }}</canvas>
                  <p v-if="qrError">{{ tr('m_88a88f2592bb') }}</p>
                </div>
                <span class="qr-caption">{{ tr('m_5c2483a4c94c') }}</span>
              </div>
              <div class="setup-instructions">
                <ol class="instruction-list">
                  <li><span>01</span><p>{{ tr('m_426b8f5bd163') }}</p></li>
                  <li><span>02</span><p>{{ tr('m_134aa8c4fa86') }}</p></li>
                </ol>
                <div class="secret-block">
                  <span class="field-label">{{ tr('m_b39ca453b31d') }}</span>
                  <div class="secret-row">
                    <code>{{ setupSession.secret }}</code>
                    <button class="text-button" type="button" @click="copySecret">{{ tr('m_d22d5fd54ca0') }}</button>
                  </div>
                </div>
                <form class="confirm-form" @submit.prevent="confirmSetup">
                  <label for="confirm-code" class="field-label">{{ tr('m_97b339ba13ec') }}</label>
                  <div class="confirm-row">
                    <input id="confirm-code" ref="confirmCodeInput" v-model="confirmCode" class="vercel-input code-input" inputmode="numeric" autocomplete="one-time-code" maxlength="6" pattern="[0-9]{6}" placeholder="000000" :disabled="confirming || setupExpired" @input="normalizeConfirmCode" />
                    <button class="btn btn-primary" type="submit" :disabled="confirming || setupExpired || confirmCode.length !== 6">
                      {{ confirming ? tr('m_6c128e0ad353') : tr('m_4e634e80bcff') }}
                    </button>
                  </div>
                </form>
              </div>
            </div>
            <button class="text-button cancel-action" type="button" @click="cancelSetup">{{ tr('m_40e037727419') }}</button>
          </div>

          <div v-else-if="disableOpen" class="disable-panel">
            <div class="operation-header">
              <div>
                <span class="operation-index">Security override</span>
                <h4>{{ tr('m_2d971783b5af') }}</h4>
              </div>
              <span class="operation-state state-critical">{{ tr('m_d224b38d6907') }}</span>
            </div>
            <p class="operation-copy">{{ tr('m_f8fa10e61725') }}</p>
            <form class="disable-form" @submit.prevent="submitDisable">
              <label>
                <span class="field-label">{{ tr('m_a114cfb687e6') }}</span>
                <input ref="disablePasswordInput" v-model="disablePassword" class="vercel-input" type="password" autocomplete="current-password" :placeholder="tr('m_e0499346b818')" />
              </label>
              <label>
                <span class="field-label">{{ tr('m_63b1e717d40a') }}</span>
                <input ref="disableCodeInput" v-model="disableCode" class="vercel-input" type="text" autocomplete="one-time-code" :placeholder="tr('m_181e416ac212')" />
              </label>
              <div class="form-actions">
                <button class="btn btn-secondary" type="button" :disabled="disabling" @click="cancelDisable">{{ tr('m_2cd0f3be8738') }}</button>
                <button class="btn btn-danger" type="submit" :disabled="disabling || !disablePassword || !disableCode.trim()">
                  {{ disabling ? tr('m_334f03fa0b8d') : tr('m_6030ce142189') }}
                </button>
              </div>
            </form>
          </div>

          <div v-else-if="twoFactorStatus?.enabled" class="security-state enabled-state">
            <div class="status-copy">
              <strong>{{ tr('m_9fd43dbdf1fc') }}</strong>
              <p>{{ tr('m_356c07078edb') }} <b>{{ twoFactorStatus.recovery_codes_remaining }}</b> {{ tr('m_b47e4a044c7b') }}</p>
            </div>
            <button class="btn btn-secondary danger-outline" type="button" @click="openDisable">{{ tr('m_411939586e8d') }}</button>
          </div>

          <div v-else-if="twoFactorStatus && !twoFactorStatus.setup_available" class="security-state unavailable-state">
            <div class="status-copy">
              <strong>{{ tr('m_6c9a1b789007') }}</strong>
              <p>{{ tr('m_89727b9c363f') }} <code>APP_ENCRYPTION_KEY</code>{{ tr('m_e1f51d1ee9a2') }}</p>
            </div>
          </div>

          <div v-else class="security-state disabled-state">
            <div class="status-copy">
              <strong>{{ tr('m_58e8e4793ce2') }}</strong>
              <p v-if="setupNotice" class="setup-notice" role="status">{{ setupNotice }}</p>
              <p v-else>{{ tr('m_88de4f3ef0b9') }}</p>
            </div>
            <button class="btn btn-primary" type="button" :disabled="startingSetup" @click="startSetup">
              {{ startingSetup ? tr('m_c68d88d6f41e') : tr('m_cb0ea9c6dfa1') }}
            </button>
          </div>
        </div>
      </section>

      <section class="settings-card security-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.06s' }" aria-labelledby="passkey-title">
        <div class="security-heading">
          <div>
            <p class="security-kicker">Passwordless sign-in</p>
            <h3 id="passkey-title" class="settings-card-title">{{ tr('m_102824e51a9d') }}</h3>
            <p class="settings-card-desc">{{ tr('m_49626291c59d') }}</p>
          </div>
          <div class="status-stamp" :class="'stamp-' + passkeyStamp.tone" role="status" aria-live="polite">
            <span class="stamp-mark" aria-hidden="true"></span>
            <span>
              <small>Passkeys</small>
              <strong>{{ passkeyStamp.label }}</strong>
            </span>
          </div>
        </div>

        <div class="security-rule" aria-hidden="true"></div>

        <div class="security-content">
          <div v-if="passkeyLoading" class="security-state compact-state">
            <span class="state-pulse" aria-hidden="true"></span>
            <div><strong>{{ tr('m_9887a9d3d7d2') }}</strong><p>{{ tr('m_6d5519d3ffad') }}</p></div>
          </div>

          <div v-else-if="!passkeys.available" class="security-state unavailable-state">
            <div class="status-copy">
              <strong>{{ tr('m_142d8abc422e') }}</strong>
              <p>{{ tr('m_e59ca37942c3') }}</p>
            </div>
          </div>

          <template v-else>
            <div v-if="!passkeys.passkeys.length" class="security-state disabled-state">
              <div class="status-copy">
                <strong>{{ tr('m_af592dc55215') }}</strong>
                <p>{{ tr('m_470297890c68') }}<code>{{ passkeys.rp_id }}</code>{{ tr('m_23fe5e96d758') }}</p>
              </div>
            </div>
            <ul v-else class="passkey-list">
              <li v-for="item in passkeys.passkeys" :key="item.id" class="passkey-row">
                <div class="passkey-info">
                  <span class="passkey-name">
                    <strong>{{ item.name }}</strong>
                    <span v-if="item.backup_eligible" class="conn-badge">{{ item.backup_state ? tr('m_04062edb583b') : tr('m_f450fddb8f2f') }}</span>
                  </span>
                  <span class="passkey-meta">
                    {{ tr('m_76d76ed548a8') }} {{ formatPasskeyDate(item.created_at) }}
                    <template v-if="item.last_used_at"> {{ tr('m_310f9beb85f1') }} {{ formatPasskeyDate(item.last_used_at) }}</template>
                  </span>
                </div>
                <div class="passkey-actions">
                  <button class="btn btn-ghost" type="button" :disabled="passkeyBusy" @click="renamePasskeyEntry(item)">{{ tr('m_0d0cbac2eee5') }}</button>
                  <button class="btn btn-ghost danger" type="button" :disabled="passkeyBusy" @click="removePasskeyEntry(item)">{{ tr('m_2f9daa828907') }}</button>
                </div>
              </li>
            </ul>

            <form class="passkey-form" @submit.prevent="addPasskey">
              <div class="field">
                <label class="field-label" for="passkey-name">{{ tr('m_d44e9b3d3b31') }}</label>
                <input id="passkey-name" v-model="passkeyName" type="text" class="vercel-input" :placeholder="tr('m_76cd75a6386c')" maxlength="60" />
              </div>
              <div class="field">
                <label class="field-label" for="passkey-password">{{ tr('m_a114cfb687e6') }}</label>
                <input
                  id="passkey-password"
                  v-model="passkeyPassword"
                  type="password"
                  class="vercel-input"
                  :placeholder="tr('m_32378523fc55')"
                  autocomplete="current-password"
                />
              </div>
              <button class="btn btn-primary" type="submit" :disabled="passkeyBusy || !passkeyPassword">
                {{ passkeyBusy ? tr('m_7c93e5eb917e') : tr('m_a24e4cc96fdc') }}
              </button>
            </form>

            <div class="passkey-switch">
              <div>
                <strong>{{ tr('m_3a027241650b') }}</strong>
                <p>{{ tr('m_315aff09e8b3') }}</p>
                <p v-if="passkeys.password_login_disabled_globally">{{ tr('m_9b7d052ac30d') }}</p>
              </div>
              <button
                class="btn btn-secondary"
                :class="{ 'danger-outline': !passkeys.password_login_disabled }"
                type="button"
                :disabled="passkeyBusy || !passkeyPassword"
                @click="togglePasswordLogin"
              >
                {{ passkeys.password_login_disabled ? tr('m_466c8603a96a') : tr('m_3a027241650b') }}
              </button>
            </div>
          </template>
        </div>
      </section>

      <section class="settings-card cloudflare-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.08s' }" aria-labelledby="cloudflare-title">
        <div class="cloudflare-heading">
          <div>
            <p class="security-kicker">Account connection</p>
            <h3 id="cloudflare-title" class="settings-card-title">{{ tr('m_c3ee7f86565f') }}</h3>
            <p class="settings-card-desc">{{ tr('m_d5db29a9a5de') }}</p>
          </div>
          <span class="connection-badge" :class="cloudflare.connected ? 'connected' : 'disconnected'">
            {{ loadingCloudflare ? tr('m_465c03b272ea') : cloudflare.connected ? tr('m_5be0323e8adc') : tr('m_3d52574ce150') }}
          </span>
        </div>

        <div v-if="loadingCloudflare" class="cloudflare-loading" role="status">
          <span class="state-pulse" aria-hidden="true"></span>
          <span>{{ tr('m_4b8405e94d1a') }}</span>
        </div>

        <div v-else class="cloudflare-panel">
          <div class="cloudflare-status">
            <div>
              <span class="cloudflare-label">{{ tr('m_46cab3bd6fb3') }}</span>
              <strong>{{ cloudflareSourceLabel }}</strong>
            </div>
            <div>
              <span class="cloudflare-label">{{ tr('m_2997f476d9bc') }}</span>
              <strong>{{ cloudflare.account_name || tr('m_c5768ca9726c') }}</strong>
              <code v-if="cloudflare.account_id">{{ cloudflare.account_id }}</code>
            </div>
            <div>
              <span class="cloudflare-label">{{ tr('m_a2eea8c8f84b') }}</span>
              <strong>{{ cloudflare.expires_at ? formatOAuthExpiry(cloudflare.expires_at) : tr('m_2746d995801b') }}</strong>
            </div>
          </div>

          <div v-if="connections.length" class="connection-list">
            <div v-for="conn in connections" :key="conn.id" class="connection-item" :class="{ active: conn.active }">
              <div class="connection-info">
                <strong>{{ conn.account_name || conn.label }}</strong>
                <code v-if="conn.account_id">{{ conn.account_id }}</code>
                <span v-if="conn.active" class="conn-badge">{{ tr('m_fa48e8938940') }}</span>
              </div>
              <div class="connection-actions">
                <button v-if="!conn.active" class="btn btn-ghost" type="button" :disabled="busyConnection === conn.id" @click="activateConnection(conn.id)">{{ tr('m_21c86745003a') }}</button>
                <button class="btn btn-ghost danger-outline" type="button" :disabled="busyConnection === conn.id" @click="removeConnection(conn)">{{ tr('m_2f9daa828907') }}</button>
              </div>
            </div>
          </div>
          <label v-if="cloudflare.source === 'oauth' && cloudflare.accounts?.length" class="account-selector">
            <span class="field-label">{{ tr('m_87dfec587589') }}</span>
            <select class="vercel-input" :value="cloudflare.account_id" :disabled="selectingAccount" @change="changeCloudflareAccount">
              <option v-for="account in cloudflare.accounts" :key="account.id" :value="account.id">
                {{ account.name }} · {{ account.id }}
              </option>
            </select>
          </label>

          <p v-if="cloudflareStatusError || cloudflare.error" class="cloudflare-error" role="alert">
            {{ cloudflareStatusError || cloudflare.error }}
          </p>

          <div v-if="!cloudflare.configured" class="oauth-setup-hint">
            <strong>{{ tr('m_53955d61e968') }}</strong>
            <span>{{ tr('m_402329d011fb') }} <code>CF_OAUTH_CLIENT_ID</code>、<code>CF_OAUTH_CLIENT_SECRET</code> {{ tr('m_4418fae73b15') }} <code>APP_ENCRYPTION_KEY</code>。</span>
          </div>

          <div class="cloudflare-actions">
            <button v-if="cloudflare.configured" class="btn btn-primary" type="button" :disabled="startingOAuth" @click="connectCloudflare">
              {{ startingOAuth ? tr('m_aec824ac0086') : cloudflare.source === 'oauth' ? tr('m_978fa225040c') : tr('m_70c33ec4f5aa') }}
            </button>
            <button v-if="cloudflare.source === 'oauth'" class="btn btn-secondary danger-outline" type="button" :disabled="disconnectingOAuth" @click="confirmDisconnectCloudflare">
              {{ disconnectingOAuth ? tr('m_68f43f735935') : tr('m_6d5e6ecdaf60') }}
            </button>
            <button v-if="cloudflareStatusError" class="btn btn-secondary" type="button" @click="loadCloudflareStatus">{{ tr('m_b8784c8dd563') }}</button>
            <span v-if="cloudflare.source === 'api_token'" class="legacy-note">{{ tr('m_822047b561c5') }}</span>
          </div>
        </div>
      </section>

      <div class="settings-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.12s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">{{ tr('m_2309fd06dbb1') }}</div>
          <div class="settings-card-desc">{{ tr('m_10b1d198de1b') }} <strong>{{ store.username }}</strong></div>
        </div>
        <div class="form-stack">
          <label class="field">
            <span class="field-label">{{ tr('m_0ccf18acc1c8') }}</span>
            <input v-model="newUsername" :placeholder="tr('m_87de734107bf')" class="vercel-input" />
          </label>
          <label class="field">
            <span class="field-label">{{ tr('m_ab48ee83149b') }}</span>
            <input v-model="usernamePassword" type="password" :placeholder="tr('m_632da57acf53')" class="vercel-input" />
          </label>
          <div class="form-actions">
            <button class="btn btn-secondary" :disabled="savingUsername" @click="saveUsername">
              {{ savingUsername ? tr('m_917b1c1f18d0') : tr('m_dbf86ad13caf') }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="hasLocalPassword !== false" class="settings-card card-transition" :class="{ 'stagger-item': visible }" :style="{ animationDelay: '0.16s' }">
        <div class="settings-card-header">
          <div class="settings-card-title">{{ tr('m_08d008062411') }}</div>
          <div class="settings-card-desc">{{ tr('m_33059d066a95') }}</div>
        </div>
        <div class="form-stack">
          <label class="field">
            <span class="field-label">{{ tr('m_a114cfb687e6') }}</span>
            <input v-model="currentPassword" type="password" :placeholder="tr('m_a114cfb687e6')" class="vercel-input" />
          </label>
          <label class="field">
            <span class="field-label">{{ tr('m_515e9c7cf7b2') }}</span>
            <input v-model="newPassword" type="password" :placeholder="tr('m_515e9c7cf7b2')" class="vercel-input" />
          </label>
          <div class="form-actions">
            <button class="btn btn-primary" :disabled="savingPassword" @click="savePassword">
              {{ savingPassword ? tr('m_917b1c1f18d0') : tr('m_828170843089') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { usePanelDialogs } from '../composables/usePanelDialogs'
const panel = usePanelDialogs()
import { locale, t as tr } from '../i18n'

import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { useDialog, useMessage } from 'naive-ui'
import QRCode from 'qrcode'
import {
  changeEmail,
  changePassword,
  changeProfile,
  changeUsername,
  confirmTwoFactor,
  uploadAvatar,
  disconnectCloudflareOAuth,
  disableTwoFactor,
  getCloudflareOAuthStatus,
  getTwoFactorStatus,
  selectCloudflareAccount,
  activateCloudflareConnection,
  setupTwoFactor,
  startCloudflareOAuth,
  type CloudflareOAuthStatus,
  type CFConnectionView,
  type TOTPSetupResponse,
  type TOTPStatusResponse,
} from '../api'
import { browserSupportsWebAuthn, startRegistration } from '@simplewebauthn/browser'
import {
  beginPasskeyRegistration,
  deletePasskey,
  finishPasskeyRegistration,
  getPasskeySettings,
  renamePasskey,
  setPasswordLoginDisabled,
  type PasskeySettings,
  type PasskeyView,
} from '../api/passkey'
import OIDCAccount from '../components/OIDCAccount.vue'
import { useConfigStore } from '../stores/config'

const hasLocalPassword = ref<boolean | null>(null)
const message = useMessage()
const dialog = useDialog()
const route = useRoute()
const router = useRouter()
const store = useConfigStore()
const visible = ref(false)

const nickname = ref('')
const savingProfile = ref(false)
const avatarInput = ref<HTMLInputElement | null>(null)
const uploadingAvatar = ref(false)
const newEmail = ref('')
const emailPassword = ref('')
const savingEmail = ref(false)
const newUsername = ref('')
const usernamePassword = ref('')
const currentPassword = ref('')
const newPassword = ref('')
const savingUsername = ref(false)
const savingPassword = ref(false)

const loadingCloudflare = ref(true)
const startingOAuth = ref(false)
const disconnectingOAuth = ref(false)
const selectingAccount = ref(false)
const cloudflareStatusError = ref('')
const cloudflare = reactive<CloudflareOAuthStatus>({
  configured: false,
  connected: false,
  source: 'none',
  account_id: '',
  account_name: '',
  accounts: [],
  redirect_uri: '',
})

const statusLoading = ref(true)
const statusError = ref('')
const twoFactorStatus = ref<TOTPStatusResponse | null>(null)
const startingSetup = ref(false)
const setupSession = ref<TOTPSetupResponse | null>(null)
const setupNotice = ref('')
const confirmCode = ref('')
const confirming = ref(false)
const qrCanvas = ref<HTMLCanvasElement | null>(null)
const qrError = ref(false)
const expiresAtMs = ref(0)
const nowMs = ref(Date.now())
const recoveryCodes = ref<string[]>([])
const recoveryAcknowledged = ref(false)
const recoveryPanel = ref<HTMLElement | null>(null)
const confirmCodeInput = ref<HTMLInputElement | null>(null)
const disablePasswordInput = ref<HTMLInputElement | null>(null)
const disableCodeInput = ref<HTMLInputElement | null>(null)
const disableOpen = ref(false)
const disablePassword = ref('')
const disableCode = ref('')
const disabling = ref(false)
let countdownTimer: ReturnType<typeof setInterval> | undefined
let visibleFrame: number | undefined
let disposed = false
let statusRequestGeneration = 0
let setupRequestGeneration = 0
let confirmRequestGeneration = 0
let oauthRequestGeneration = 0

const setupExpired = computed(() => Boolean(setupSession.value) && expiresAtMs.value <= nowMs.value)
const countdownLabel = computed(() => {
  const seconds = Math.max(0, Math.ceil((expiresAtMs.value - nowMs.value) / 1000))
  const minutes = Math.floor(seconds / 60).toString().padStart(2, '0')
  return `${minutes}:${(seconds % 60).toString().padStart(2, '0')}`
})

const avatarInitial = computed(() => (store.displayName || store.username || '?').trim().charAt(0).toUpperCase())

const statusStamp = computed(() => {
  if (statusLoading.value) return { label: tr('m_465c03b272ea'), tone: 'neutral' }
  if (statusError.value) return { label: tr('m_78c70751f7a5'), tone: 'error' }
  if (recoveryCodes.value.length) return { label: tr('m_a10cd63fe4f1'), tone: 'warning' }
  if (setupSession.value) return { label: setupExpired.value ? tr('m_7b1f68536f70') : tr('m_6be8ede5322b'), tone: setupExpired.value ? 'error' : 'warning' }
  if (disableOpen.value) return { label: tr('m_24ea84bdd9a3'), tone: 'error' }
  if (twoFactorStatus.value?.enabled) return { label: tr('m_dfb802238b38'), tone: 'success' }
  if (twoFactorStatus.value && !twoFactorStatus.value.setup_available) return { label: tr('m_474ed5cb87f5'), tone: 'neutral' }
  return { label: tr('m_f95ea7f4c063'), tone: 'neutral' }
})

const cloudflareSourceLabel = computed(() => {
  if (cloudflare.source === 'oauth') return 'Cloudflare OAuth 2.0'
  if (cloudflare.source === 'api_token') return tr('m_c57b367a8b76')
  return tr('m_80a57e03f071')
})

function apiError(error: any, fallback: string) {
  return error.response?.data?.error || error.message || fallback
}

async function loadCloudflareStatus() {
  const generation = ++oauthRequestGeneration
  loadingCloudflare.value = true
  cloudflareStatusError.value = ''
  try {
    const { data } = await getCloudflareOAuthStatus()
    if (disposed || generation !== oauthRequestGeneration) return
    Object.assign(cloudflare, {
      configured: false,
      connected: false,
      source: 'none',
      account_id: '',
      account_name: '',
      accounts: [],
      expires_at: undefined,
      redirect_uri: '',
      error: undefined,
    }, data, { accounts: data.accounts || [] })
  } catch (error: any) {
    if (disposed || generation !== oauthRequestGeneration) return
    cloudflareStatusError.value = apiError(error, tr('m_9ba4b48c2d18'))
    message.error(tr('m_d47d2b0631d0') + cloudflareStatusError.value)
  } finally {
    if (!disposed && generation === oauthRequestGeneration) loadingCloudflare.value = false
  }
}

async function connectCloudflare() {
  if (startingOAuth.value) return
  startingOAuth.value = true
  try {
    const { data } = await startCloudflareOAuth()
    window.location.assign(data.authorization_url)
  } catch (error: any) {
    startingOAuth.value = false
    message.error(tr('m_0cd9d6808d2b') + apiError(error, tr('m_9ba4b48c2d18')))
  }
}

async function disconnectCloudflare() {
  if (disconnectingOAuth.value) return
  disconnectingOAuth.value = true
  try {
    const { data } = await disconnectCloudflareOAuth()
    if (data.warning) message.warning(data.warning)
    else message.success(tr('m_c276889cfaa8'))
    await loadCloudflareStatus()
  } catch (error: any) {
    message.error(tr('m_a0c2de63babc') + apiError(error, tr('m_9ba4b48c2d18')))
  } finally {
    disconnectingOAuth.value = false
  }
}

const busyConnection = ref('')
const connections = computed(() => cloudflare.connections || [])

async function activateConnection(connectionID: string) {
  if (busyConnection.value) return
  busyConnection.value = connectionID
  try {
    const { data } = await activateCloudflareConnection(connectionID)
    message.success(tr('m_543b8d1935f0') + (data.account_name || data.account_id))
    await loadCloudflareStatus()
    await store.fetchConfig()
  } catch (error: any) {
    message.error(tr('m_59f11fdc3a28') + apiError(error, tr('m_9ba4b48c2d18')))
  } finally {
    busyConnection.value = ''
  }
}

async function removeConnection(conn: CFConnectionView) {
  if (busyConnection.value) return
  if (!await panel.confirm(tr('m_9a0ac433a9be') + (conn.account_name || conn.label) + tr('m_14f4ceb14005'))) return
  busyConnection.value = conn.id
  try {
    const { data } = await disconnectCloudflareOAuth(conn.id)
    if (data.warning) message.warning(data.warning)
    else message.success(tr('m_bcb4b0b8810c'))
    await loadCloudflareStatus()
  } catch (error: any) {
    message.error(tr('m_48477dd59e05') + apiError(error, tr('m_9ba4b48c2d18')))
  } finally {
    busyConnection.value = ''
  }
}

function confirmDisconnectCloudflare() {
  if (disconnectingOAuth.value) return
  dialog.warning({
    title: tr('m_d20e59ee1db3'),
    content: tr('m_5d19a4dcd36f'),
    positiveText: tr('m_c642f3322039'),
    negativeText: tr('m_2cd0f3be8738'),
    positiveButtonProps: { type: 'error' },
    onPositiveClick: disconnectCloudflare,
  })
}

async function changeCloudflareAccount(event: Event) {
  const accountID = (event.target as HTMLSelectElement).value
  if (!accountID || selectingAccount.value || accountID === cloudflare.account_id) return
  selectingAccount.value = true
  try {
    const { data } = await selectCloudflareAccount(accountID)
    cloudflare.account_id = data.id
    cloudflare.account_name = data.name
    message.success(tr('m_a1b61d14e015', { p0: data.name }))
  } catch (error: any) {
    message.error(tr('m_d4418cde2035') + apiError(error, tr('m_9ba4b48c2d18')))
    await loadCloudflareStatus()
  } finally {
    selectingAccount.value = false
  }
}

// 通行密钥时间戳为 Unix 秒。
function formatPasskeyDate(seconds: number) {
  if (!seconds) return '—'
  return new Date(seconds * 1000).toLocaleString(locale.value)
}

function formatOAuthExpiry(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? tr('m_98073e15fa03') : date.toLocaleString(locale.value)
}

async function handleCloudflareOAuthResult() {
  const result = typeof route.query.cloudflare_oauth === 'string' ? route.query.cloudflare_oauth : ''
  if (!result) return
  const oauthMessage = typeof route.query.message === 'string' ? route.query.message : ''
  const query = { ...route.query }
  delete query.cloudflare_oauth
  delete query.message
  await router.replace({ path: route.path, query })
  if (result === 'success') message.success(tr('m_28cf30fefcd2'))
  if (result === 'error') message.error(tr('m_9eb2c713323f') + (oauthMessage || tr('m_13a46616e16f')))
}

async function loadTwoFactorStatus() {
  const generation = ++statusRequestGeneration
  statusLoading.value = true
  statusError.value = ''
  try {
    const { data } = await getTwoFactorStatus()
    if (disposed || generation !== statusRequestGeneration) return
    twoFactorStatus.value = data
  } catch (error: any) {
    if (disposed || generation !== statusRequestGeneration) return
    statusError.value = apiError(error, tr('m_9ba4b48c2d18'))
  } finally {
    if (!disposed && generation === statusRequestGeneration) statusLoading.value = false
  }
}

function clearCountdown() {
  if (countdownTimer) clearInterval(countdownTimer)
  countdownTimer = undefined
}

function beginCountdown(expiresAt: number) {
  clearCountdown()
  expiresAtMs.value = expiresAt
  nowMs.value = Date.now()
  countdownTimer = setInterval(() => {
    if (disposed) return
    nowMs.value = Date.now()
    if (expiresAtMs.value <= nowMs.value) expireSetup()
  }, 1000)
}

function expireSetup() {
  ++setupRequestGeneration
  clearSetup()
  setupNotice.value = tr('m_1286f11f6137')
}

async function renderQr(generation: number) {
  qrError.value = false
  await nextTick()
  if (disposed || generation !== setupRequestGeneration || !qrCanvas.value || !setupSession.value) return
  try {
    await QRCode.toCanvas(qrCanvas.value, setupSession.value.otpauth_uri, {
      width: 220,
      margin: 2,
      errorCorrectionLevel: 'M',
      color: { dark: '#000000', light: '#ffffff' },
    })
  } catch {
    if (!disposed && generation === setupRequestGeneration) qrError.value = true
  }
}

async function startSetup() {
  const generation = ++setupRequestGeneration
  startingSetup.value = true
  setupNotice.value = ''
  try {
    const { data } = await setupTwoFactor()
    if (disposed || generation !== setupRequestGeneration) return
    const expiresAt = Date.parse(data.expires_at)
    if (!Number.isFinite(expiresAt) || expiresAt <= Date.now()) {
      clearSetup()
      setupNotice.value = tr('m_7170801d9d8a')
      message.error(setupNotice.value)
      return
    }
    setupSession.value = data
    confirmCode.value = ''
    beginCountdown(expiresAt)
    await renderQr(generation)
    await nextTick()
    if (!disposed && generation === setupRequestGeneration) confirmCodeInput.value?.focus()
  } catch (error: any) {
    if (disposed || generation !== setupRequestGeneration) return
    message.error(tr('m_c449c4aca730') + apiError(error, tr('m_9ba4b48c2d18')))
    await loadTwoFactorStatus()
  } finally {
    if (!disposed && generation === setupRequestGeneration) startingSetup.value = false
  }
}

function clearSetup() {
  clearCountdown()
  setupSession.value = null
  confirmCode.value = ''
  expiresAtMs.value = 0
  qrError.value = false
  if (qrCanvas.value) {
    const context = qrCanvas.value.getContext('2d')
    context?.clearRect(0, 0, qrCanvas.value.width, qrCanvas.value.height)
  }
}

function cancelSetup() {
  ++setupRequestGeneration
  clearSetup()
  setupNotice.value = ''
}

function normalizeConfirmCode() {
  confirmCode.value = confirmCode.value.replace(/\D/g, '').slice(0, 6)
}

async function confirmSetup() {
  if (!setupSession.value || !/^\d{6}$/.test(confirmCode.value) || setupExpired.value) return
  const generation = ++confirmRequestGeneration
  const setupToken = setupSession.value.setup_token
  const code = confirmCode.value
  confirming.value = true
  try {
    const { data } = await confirmTwoFactor(setupToken, code)
    if (disposed || generation !== confirmRequestGeneration) return
    recoveryCodes.value = [...data.recovery_codes]
    recoveryAcknowledged.value = false
    twoFactorStatus.value = { enabled: data.enabled, recovery_codes_remaining: data.recovery_codes.length, setup_available: false }
    clearSetup()
    store.clearAuth()
    await nextTick()
    if (disposed || generation !== confirmRequestGeneration) return
    recoveryPanel.value?.focus()
    message.success(tr('m_98169c468a21'))
  } catch (error: any) {
    if (disposed || generation !== confirmRequestGeneration) return
    message.error(tr('m_1f752a49011b') + apiError(error, tr('m_4331e04edb46')))
    confirmCodeInput.value?.focus()
    confirmCodeInput.value?.select()
  } finally {
    if (!disposed && generation === confirmRequestGeneration) confirming.value = false
  }
}

async function copyText(value: string, successText: string) {
  try {
    await navigator.clipboard.writeText(value)
    message.success(successText)
  } catch {
    message.error(tr('m_dfaefbdef784'))
  }
}

function copySecret() {
  if (setupSession.value) void copyText(setupSession.value.secret, tr('m_d6f84f446290'))
}

function copyRecoveryCodes() {
  void copyText(recoveryCodes.value.join('\n'), tr('m_6f47d1a74036'))
}

function downloadRecoveryCodes() {
  const blob = new Blob([recoveryCodes.value.join('\n') + '\n'], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'tunnel-manager-recovery-codes.txt'
  link.click()
  window.setTimeout(() => URL.revokeObjectURL(url), 0)
}

async function finishRecovery() {
  if (!recoveryAcknowledged.value) return
  recoveryCodes.value = []
  recoveryAcknowledged.value = false
  await router.replace('/')
}

function openDisable() {
  disableOpen.value = true
  disablePassword.value = ''
  disableCode.value = ''
  nextTick(() => disablePasswordInput.value?.focus())
}

function cancelDisable() {
  disableOpen.value = false
  disablePassword.value = ''
  disableCode.value = ''
}

async function submitDisable() {
  if (!disablePassword.value || !disableCode.value.trim()) return
  disabling.value = true
  try {
    await disableTwoFactor(disablePassword.value, disableCode.value.trim())
    cancelDisable()
    store.clearAuth()
    message.success(tr('m_f0ac006c0d44'))
    await router.replace('/')
  } catch (error: any) {
    message.error(tr('m_f800c50b531d') + apiError(error, tr('m_72590600175c')))
    if (error.response?.status === 401) {
      disableCode.value = ''
      await nextTick()
      disableCodeInput.value?.focus()
    }
  } finally {
    disabling.value = false
  }
}

function handleBeforeUnload(event: BeforeUnloadEvent) {
  if (!confirming.value && !recoveryCodes.value.length) return
  event.preventDefault()
  event.returnValue = ''
}

onBeforeRouteLeave(() => {
  if (confirming.value || recoveryCodes.value.length) return false
})

async function saveProfile() {
  const value = nickname.value.trim()
  if (value.length > 64) {
    message.error(tr('m_dae4662865e0'))
    return
  }
  savingProfile.value = true
  try {
    await changeProfile(value, store.avatar)
    await store.fetchMe()
    message.success(tr('m_b844ea2e13f3'))
  } catch (e: any) {
    message.error(tr('m_84bd3b614686') + (e.response?.data?.error || e.message))
  } finally {
    savingProfile.value = false
  }
}

function triggerAvatarUpload() {
  avatarInput.value?.click()
}

function handleAvatarFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  const allowed = ['image/png', 'image/jpeg', 'image/gif', 'image/webp']
  if (!allowed.includes(file.type)) {
    message.error(tr('m_24a3b2d5136c'))
    return
  }
  if (file.size > 4 * 1024 * 1024) {
    message.error(tr('m_9d0b8bbb7900'))
    return
  }
  uploadingAvatar.value = true
  uploadAvatar(file)
    .then(async ({ data }) => {
      await store.fetchMe()
      message.success(tr('m_a938ed15b3e5'))
    })
    .catch((e: any) => {
      message.error(tr('m_69167075edb5') + (e.response?.data?.error || e.message))
    })
    .finally(() => {
      uploadingAvatar.value = false
    })
}

async function removeAvatar() {
  savingProfile.value = true
  try {
    await changeProfile(nickname.value.trim(), '')
    await store.fetchMe()
    message.success(tr('m_eca726423f04'))
  } catch (e: any) {
    message.error(tr('m_d5711a729a98') + (e.response?.data?.error || e.message))
  } finally {
    savingProfile.value = false
  }
}

async function saveEmail() {
  const email = newEmail.value.trim()
  if (!email || !emailPassword.value) {
    message.error(tr('m_bdb626f8b359'))
    return
  }
  savingEmail.value = true
  try {
    await changeEmail(emailPassword.value, email)
    newEmail.value = ''
    emailPassword.value = ''
    await store.fetchMe()
    message.success(store.email ? tr('m_0870a627daa8') : tr('m_bb71c4d5fc40'))
  } catch (e: any) {
    message.error(tr('m_84bd3b614686') + (e.response?.data?.error || e.message))
  } finally {
    savingEmail.value = false
  }
}

async function saveUsername() {
  if (!newUsername.value || !usernamePassword.value) {
    message.error(tr('m_18de25f0d593'))
    return
  }
  savingUsername.value = true
  try {
    await changeUsername(usernamePassword.value, newUsername.value)
    store.clearAuth()
    message.success(tr('m_dc6cae76cd7a'))
    await router.replace('/')
  } catch (e: any) {
    message.error(tr('m_00eb7b374633') + (e.response?.data?.error || e.message))
  } finally {
    savingUsername.value = false
  }
}

async function savePassword() {
  if (!currentPassword.value || !newPassword.value) {
    message.error(tr('m_824cc55354e0'))
    return
  }
  if (newPassword.value.length < 6) {
    message.error(tr('m_56e015fc995a'))
    return
  }
  savingPassword.value = true
  try {
    await changePassword(currentPassword.value, newPassword.value)
    store.clearAuth()
    message.success(tr('m_e559ccc5f437'))
    await router.replace('/')
  } catch (e: any) {
    message.error(tr('m_00eb7b374633') + (e.response?.data?.error || e.message))
  } finally {
    savingPassword.value = false
  }
}


// ---------------------------------------------------------------------------
// 通行密钥

const passkeys = ref<PasskeySettings>({
  passkeys: [],
  password_login_disabled: false,
  password_login_disabled_globally: false,
  rp_id: '',
  origin: '',
  available: false,
})
const passkeyLoading = ref(false)
const passkeyBusy = ref(false)
const passkeyName = ref('')
const passkeyPassword = ref('')

const passkeyStamp = computed(() => {
  if (!passkeys.value.available) return { label: tr('m_460b3574e4bd'), tone: 'neutral' }
  if (!passkeys.value.passkeys.length) return { label: tr('m_e026c6693dc5'), tone: 'neutral' }
  return { label: tr('m_2937a5ea1250') + passkeys.value.passkeys.length + tr('m_7aaf96ad52e0'), tone: 'success' }
})

async function loadPasskeys() {
  passkeyLoading.value = true
  try {
    const { data } = await getPasskeySettings()
    if (disposed) return
    passkeys.value = data
  } catch (error: any) {
    if (!disposed) message.error(apiError(error, tr('m_9ee2f030db41')))
  } finally {
    if (!disposed) passkeyLoading.value = false
  }
}

async function addPasskey() {
  if (passkeyBusy.value) return
  if (!browserSupportsWebAuthn()) {
    message.error(tr('m_73253680f9f3'))
    return
  }
  passkeyBusy.value = true
  try {
    const { data } = await beginPasskeyRegistration(passkeyPassword.value)
    const credential = await startRegistration({ optionsJSON: data.public_key })
    await finishPasskeyRegistration({
      ceremony_token: data.ceremony_token,
      name: passkeyName.value.trim(),
      credential,
    })
    passkeyName.value = ''
    message.success(tr('m_716a1862dd06'))
    await loadPasskeys()
  } catch (error: any) {
    if (error?.name === 'NotAllowedError' || error?.name === 'AbortError') message.warning(tr('m_f951ae00348b'))
    else message.error(apiError(error, tr('m_21221a0f26a9')))
  } finally {
    passkeyBusy.value = false
  }
}

async function renamePasskeyEntry(item: PasskeyView) {
  const name = await panel.prompt(tr('m_e8e46cb47287'), item.name)
  if (!name || name.trim() === item.name) return
  void (async () => {
    try {
      await renamePasskey(item.id, name.trim())
      message.success(tr('m_06afd1f0352e'))
      await loadPasskeys()
    } catch (error: any) {
      message.error(apiError(error, tr('m_e367f53aeadf')))
    }
  })()
}

function removePasskeyEntry(item: PasskeyView) {
  if (!passkeyPassword.value) {
    message.warning(tr('m_05b1298143b6'))
    return
  }
  dialog.warning({
    title: tr('m_fc3f78b499e1'),
    content: tr('m_72f49a525798') + item.name + tr('m_139f37a3ae7c'),
    positiveText: tr('m_2f9daa828907'),
    negativeText: tr('m_2cd0f3be8738'),
    onPositiveClick: async () => {
      try {
        await deletePasskey(item.id, passkeyPassword.value)
        message.success(tr('m_dbef251d5058'))
        await loadPasskeys()
      } catch (error: any) {
        message.error(apiError(error, tr('m_c228558cf257')))
      }
    },
  })
}

async function togglePasswordLogin() {
  if (!passkeyPassword.value) {
    message.warning(tr('m_05b1298143b6'))
    return
  }
  const disabled = !passkeys.value.password_login_disabled
  passkeyBusy.value = true
  try {
    const { data } = await setPasswordLoginDisabled(disabled, passkeyPassword.value)
    passkeys.value = data
    message.success(disabled ? tr('m_7cda7e6a2511') : tr('m_40ea52edefce'))
  } catch (error: any) {
    message.error(apiError(error, tr('m_0c3b4cf7aa25')))
  } finally {
    passkeyBusy.value = false
  }
}

onMounted(async () => {
  nickname.value = store.nickname || ''
  newEmail.value = ''
  emailPassword.value = ''
  await store.fetchMe()
  nickname.value = store.nickname || ''
  window.addEventListener('beforeunload', handleBeforeUnload)
  visibleFrame = requestAnimationFrame(() => {
    visibleFrame = undefined
    if (!disposed) visible.value = true
  })
  void loadTwoFactorStatus()
  void loadCloudflareStatus()
  void loadPasskeys()
  void handleCloudflareOAuthResult()
})

onBeforeUnmount(() => {
  disposed = true
  ++statusRequestGeneration
  ++setupRequestGeneration
  ++oauthRequestGeneration
  window.removeEventListener('beforeunload', handleBeforeUnload)
  if (visibleFrame !== undefined) cancelAnimationFrame(visibleFrame)
  clearCountdown()
  clearSetup()
  recoveryCodes.value = []
  disablePassword.value = ''
  disableCode.value = ''
})
</script>

<style scoped>
.settings-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.settings-card {
  min-width: 0;
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
  margin: 0 0 4px;
}
.settings-card-desc {
  font-size: 14px;
  color: var(--color-mute);
  line-height: 1.65;
  margin: 0;
}

.profile-card .security-rule {
  margin: var(--spacing-lg) 0 var(--spacing-md);
}

.profile-main {
  display: flex;
  gap: var(--spacing-lg);
  align-items: flex-start;
}

.avatar-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  flex: 0 0 auto;
  width: 132px;
}

.avatar-preview {
  width: 104px;
  height: 104px;
  border-radius: 50%;
  border: 1px solid var(--color-hairline-strong);
  background: var(--color-canvas-soft);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  font-size: 40px;
  font-weight: 700;
  color: var(--color-link);
  flex: 0 0 auto;
}

.avatar-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-actions {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  justify-content: center;
}

.avatar-actions .btn {
  padding: 5px 12px;
  font-size: 12px;
}

.avatar-input {
  display: none;
}

.avatar-hint {
  color: var(--color-mute);
  font-size: 11px;
  text-align: center;
  line-height: 1.5;
}

.profile-fields {
  flex: 1;
  min-width: 0;
}

.profile-meta {
  margin: 0;
  color: var(--color-mute);
  font-size: 13px;
}

.profile-meta strong {
  color: var(--color-ink);
  font-weight: 600;
}

.email-heading {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-sm);
  margin-bottom: var(--spacing-sm);
}

.email-title {
  margin: 0 0 4px;
  font-size: 15px;
  font-weight: 600;
  color: var(--color-ink);
}

.email-status {
  display: inline-block;
  margin-bottom: 6px;
  padding: 2px 9px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
}

.email-status.bound {
  color: var(--color-status-up-text);
  border: 1px solid var(--color-status-up-border);
  background: var(--color-status-up-bg);
}

.email-status.unbound {
  color: var(--color-mute);
  border: 1px solid var(--color-hairline-strong);
  background: var(--color-canvas-soft);
}

.email-current {
  margin: 0;
  color: var(--color-mute);
  font-size: 13px;
  line-height: 1.6;
}

.email-current strong {
  color: var(--color-ink);
  font-weight: 600;
  overflow-wrap: anywhere;
}

.email-form {
  margin-top: var(--spacing-sm);
}

.email-fields {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--spacing-sm);
}

@media (max-width: 640px) {
  .profile-main {
    flex-direction: column;
    align-items: center;
  }
  .profile-fields {
    width: 100%;
  }
  .email-fields {
    grid-template-columns: 1fr;
  }
}

.cloudflare-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--spacing-lg);
  margin-bottom: var(--spacing-md);
}
.cloudflare-heading .settings-card-title { font-size: 19px; }
.cloudflare-heading .settings-card-desc { max-width: 680px; }
.connection-badge { display: inline-flex; flex: none; align-items: center; padding: 6px 10px; border-radius: 999px; font-size: 12px; font-weight: 700; }
.connection-badge.connected { color: var(--color-success); background: var(--color-result-success-bg); }
.connection-badge.disconnected { color: var(--color-mute); background: var(--color-canvas-soft); }
.cloudflare-loading { display: flex; align-items: center; gap: var(--spacing-sm); min-height: 72px; color: var(--color-mute); font-size: 14px; }
.cloudflare-panel { display: flex; min-width: 0; flex-direction: column; gap: var(--spacing-md); }
.cloudflare-status { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: var(--spacing-sm); }
.cloudflare-status > div { display: flex; min-width: 0; flex-direction: column; gap: 4px; padding: var(--spacing-md); border: 1px solid var(--color-hairline); border-radius: var(--radius-md); background: var(--color-canvas-soft); }
.cloudflare-label { color: var(--color-mute); font-family: var(--font-mono); font-size: 10px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; }
.cloudflare-status strong { color: var(--color-ink); font-size: 14px; overflow-wrap: anywhere; }
.cloudflare-status code, .oauth-setup-hint code { color: var(--color-mute); font: 12px/1.5 var(--font-mono); overflow-wrap: anywhere; word-break: break-all; }
.account-selector { display: flex; min-width: 0; max-width: 680px; flex-direction: column; gap: 5px; }
.account-selector select { width: 100%; min-width: 0; }
.cloudflare-error { margin: 0; padding: var(--spacing-sm) var(--spacing-md); color: var(--color-error); border-radius: var(--radius-md); background: var(--color-result-error-bg); font-size: 13px; line-height: 1.6; overflow-wrap: anywhere; }
.oauth-setup-hint { display: flex; min-width: 0; flex-direction: column; gap: 5px; padding: var(--spacing-md); border: 1px dashed var(--color-hairline-strong); border-radius: var(--radius-md); color: var(--color-mute); font-size: 13px; overflow-wrap: anywhere; }
.oauth-setup-hint strong { color: var(--color-ink); }
.cloudflare-actions { display: flex; align-items: center; flex-wrap: wrap; gap: var(--spacing-sm); }
.legacy-note { min-width: 0; color: var(--color-mute); font-size: 12px; line-height: 1.6; overflow-wrap: anywhere; }

.security-card {
  padding: 0;
  overflow: hidden;
  border-color: var(--color-hairline-strong);
}

.security-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--spacing-lg);
  padding: 20px var(--spacing-lg) 18px;
}

.security-kicker,
.operation-index,
.field-label,
.qr-caption {
  display: block;
  margin: 0 0 5px;
  color: var(--color-mute);
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  line-height: 1.4;
  letter-spacing: 0.09em;
  text-transform: uppercase;
}

.security-heading .settings-card-title { font-size: 19px; }
.security-heading .settings-card-desc { max-width: 620px; }
.security-rule { height: 1px; background: var(--color-hairline); }
.security-content { min-height: 92px; }
.security-content > * { animation: security-fade 160ms ease-out both; }

.status-stamp {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 9px;
  min-width: 150px;
  padding: 10px 12px;
  border: 1px solid var(--color-hairline-strong);
  border-radius: var(--radius-sm);
  background: var(--color-canvas-soft);
  transform: rotate(-0.6deg);
}
.status-stamp small,
.status-stamp strong { display: block; line-height: 1.2; }
.status-stamp small {
  margin-bottom: 3px;
  font-family: var(--font-mono);
  font-size: 9px;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  opacity: 0.72;
}
.status-stamp strong { font-size: 13px; }
.stamp-mark { width: 8px; height: 8px; border-radius: 50%; background: currentColor; box-shadow: 0 0 0 3px color-mix(in srgb, currentColor 16%, transparent); }
.stamp-success { color: var(--color-status-healthy-text); border-color: var(--color-status-healthy-border); background: var(--color-status-healthy-bg); }
.stamp-warning { color: var(--color-status-degraded-text); border-color: var(--color-status-degraded-border); background: var(--color-status-degraded-bg); }
.stamp-error { color: var(--color-status-down-text); border-color: var(--color-status-down-border); background: var(--color-status-down-bg); }
.stamp-neutral { color: var(--color-body); }

.security-state {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-lg);
  padding: var(--spacing-md) var(--spacing-lg);
}
.security-state strong { color: var(--color-ink); font-size: 14px; }
.security-state p,
.operation-copy { margin: 3px 0 0; color: var(--color-body); font-size: 13px; line-height: 1.65; }
.security-state b { color: var(--color-ink); font-family: var(--font-mono); }
.status-copy { min-width: 0; }
.compact-state { justify-content: flex-start; }
.compact-state .btn { margin-left: auto; }
.state-error strong { color: var(--color-error); }
.state-pulse { flex: 0 0 auto; width: 9px; height: 9px; border-radius: 50%; background: var(--color-warning); animation: pulse 1.4s ease-in-out infinite; }

.operation-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--spacing-md);
  margin-bottom: var(--spacing-sm);
}
.operation-header h4 { margin: 0; color: var(--color-ink); font-size: 17px; }
.operation-state {
  flex: 0 0 auto;
  padding: 4px 7px;
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}
.state-live { color: var(--color-status-healthy-text); border-color: var(--color-status-healthy-border); background: var(--color-status-healthy-bg); }
.state-critical { color: var(--color-status-down-text); border-color: var(--color-status-down-border); background: var(--color-status-down-bg); }

.setup-panel,
.disable-panel,
.recovery-panel { padding: 20px var(--spacing-lg) var(--spacing-lg); }
.setup-grid {
  display: grid;
  grid-template-columns: minmax(220px, 0.72fr) minmax(0, 1.6fr);
  gap: var(--spacing-lg);
  margin-top: var(--spacing-md);
}
.qr-column { display: flex; flex-direction: column; align-items: center; justify-content: flex-start; min-width: 0; }
.qr-frame {
  display: grid;
  place-items: center;
  width: min(100%, 220px);
  aspect-ratio: 1;
  overflow: hidden;
  border: 1px solid var(--color-hairline-strong);
  border-radius: var(--radius-md);
  background: #fff;
}
.qr-frame canvas { display: block; width: 100% !important; height: auto !important; max-width: 220px; }
.qr-frame p { padding: var(--spacing-sm); color: #7e271c; font-size: 12px; text-align: center; }
.qr-error canvas { display: none; }
.qr-caption { margin: 8px 0 0; text-align: center; text-transform: none; letter-spacing: 0.02em; }
.setup-instructions { min-width: 0; }
.instruction-list { display: grid; gap: 10px; margin: 0 0 var(--spacing-md); padding: 0; list-style: none; }
.instruction-list li { display: grid; grid-template-columns: 28px 1fr; align-items: start; gap: 9px; }
.instruction-list span { color: var(--color-link); font-family: var(--font-mono); font-size: 11px; font-weight: 700; line-height: 1.7; }
.instruction-list p { margin: 0; color: var(--color-body); font-size: 13px; line-height: 1.7; }
.secret-block { padding: 10px 12px; border: 1px solid var(--color-hairline); border-radius: var(--radius-md); background: var(--color-canvas-soft); }
.secret-row { display: flex; align-items: center; justify-content: space-between; gap: var(--spacing-sm); }
.secret-row code { min-width: 0; color: var(--color-ink); font-family: var(--font-mono); font-size: 13px; font-weight: 700; overflow-wrap: anywhere; word-break: break-all; }
.text-button { appearance: none; padding: 0; border: 0; background: transparent; color: var(--color-link); font-size: 12px; font-weight: 700; cursor: pointer; }
.text-button:hover { text-decoration: underline; }
.confirm-form { margin-top: var(--spacing-md); }
.confirm-row { display: flex; gap: var(--spacing-sm); align-items: stretch; }
.code-input { font-family: var(--font-mono); font-size: 16px; letter-spacing: 0.18em; }
.cancel-action { margin-top: var(--spacing-md); }

.disable-form { display: grid; grid-template-columns: 1fr 1fr; gap: var(--spacing-sm); margin-top: var(--spacing-md); }
.disable-form label { min-width: 0; }
.form-actions { display: flex; grid-column: 1 / -1; justify-content: flex-end; gap: var(--spacing-sm); margin-top: 4px; }
.btn-danger { color: var(--color-status-down-text); border: 1px solid var(--color-status-down-border); background: var(--color-status-down-bg); }
.btn-danger:hover:not(:disabled) { border-color: var(--color-error); }
.danger-outline { color: var(--color-error); }

.recovery-panel { outline: none; }
.recovery-panel:focus {
  outline: 3px solid color-mix(in srgb, var(--color-focus) 42%, transparent);
  outline-offset: -3px;
}
.recovery-actions { display: flex; gap: var(--spacing-xs); margin: var(--spacing-md) 0; }
.recovery-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 7px var(--spacing-sm);
  margin: 0;
  padding: var(--spacing-md) var(--spacing-md) var(--spacing-md) 42px;
  border: 1px solid var(--color-hairline-strong);
  border-radius: var(--radius-md);
  background: var(--color-canvas-soft);
}
.recovery-grid li { padding-left: 3px; color: var(--color-mute); font-family: var(--font-mono); font-size: 11px; }
.recovery-grid code { color: var(--color-ink); font-size: 13px; font-weight: 700; overflow-wrap: anywhere; }
.completion-check { display: flex; align-items: flex-start; gap: 9px; margin: var(--spacing-md) 0 var(--spacing-sm); color: var(--color-body); font-size: 13px; cursor: pointer; }
.completion-check input { flex: 0 0 auto; margin-top: 4px; accent-color: var(--color-focus); }
.finish-button { min-width: 210px; }

.settings-input-row {
  display: flex;
  gap: var(--spacing-sm);
  align-items: flex-start;
}
.settings-action-row { margin-top: var(--spacing-sm); }
.input-wrapper { flex: 1; min-width: 0; }

@keyframes pulse { 50% { opacity: 0.35; transform: scale(0.75); } }
@keyframes security-fade { from { opacity: 0; transform: translateY(3px); } to { opacity: 1; transform: translateY(0); } }

@media (prefers-reduced-motion: reduce) {
  .state-pulse { animation: none; }
  .status-stamp { transform: none; }
  .settings-card { transition-duration: 1ms; }
  .security-content > * { animation-duration: 1ms; }
}

@media (max-width: 768px) {
  .security-heading,
  .security-state,
  .cloudflare-heading { align-items: stretch; flex-direction: column; gap: var(--spacing-sm); }
  .status-stamp { align-self: flex-start; min-width: 0; }
  .connection-badge { align-self: flex-start; }
  .cloudflare-status { grid-template-columns: 1fr; }
  .cloudflare-actions { align-items: stretch; flex-direction: column; }
  .cloudflare-actions .btn { width: 100%; max-width: 100%; justify-content: center; }
  .security-state .btn,
  .compact-state .btn { width: 100%; max-width: 100%; margin-left: 0; justify-content: center; }
  .setup-grid { grid-template-columns: 1fr; }
  .confirm-row,
  .disable-form { grid-template-columns: 1fr; flex-direction: column; }
  .confirm-row .btn,
  .form-actions .btn,
  .recovery-actions .btn,
  .finish-button { width: 100%; max-width: 100%; justify-content: center; }
  .form-actions { flex-direction: column; }
  .recovery-actions { flex-direction: column; }
  .recovery-grid { grid-template-columns: 1fr; }
  .settings-input-row { flex-direction: column; }
  .settings-input-row .btn { width: 100%; max-width: 100%; justify-content: center; }
}

@media (max-width: 480px) {
  .account-page { padding-left: var(--spacing-md); padding-right: var(--spacing-md); }
  .security-heading,
  .security-state,
  .setup-panel,
  .disable-panel,
  .recovery-panel,
  .settings-card:not(.security-card) { padding-left: var(--spacing-md); padding-right: var(--spacing-md); }
  .operation-header { flex-direction: column; }
  .secret-row { align-items: flex-start; flex-direction: column; }
  .recovery-grid { padding-left: 34px; }
}

/* Cloudflare 多账户连接 */
.connection-list { display: flex; flex-direction: column; gap: 8px; margin-bottom: var(--spacing-md); }
.connection-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  background: var(--color-canvas-soft);
}
.connection-item.active { border-color: var(--color-link); }
.connection-info { display: flex; align-items: center; gap: 10px; font-size: 13px; color: var(--color-ink); flex-wrap: wrap; flex: 1 1 auto; min-width: 0; }
.connection-info strong { min-width: 0; overflow-wrap: anywhere; word-break: break-word; }
.connection-info code { font-size: 12px; color: var(--color-mute); overflow-wrap: anywhere; word-break: break-all; }
.connection-actions { display: flex; gap: 4px; flex-shrink: 0; }
.conn-badge {
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  color: var(--color-link);
  border: 1px solid currentColor;
}
.connection-actions { display: flex; gap: 4px; }
.passkey-list { list-style: none; margin: 0 0 var(--spacing-md); padding: 0; display: flex; flex-direction: column; gap: 8px; }
.passkey-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 10px 12px; border: 1px solid var(--color-hairline); border-radius: var(--radius-md); }
.passkey-info { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.passkey-name { display: flex; align-items: center; gap: 8px; font-size: 14px; color: var(--color-ink); }
.passkey-meta { font-size: 12px; color: var(--color-mute); }
.passkey-actions { display: flex; gap: 4px; flex-shrink: 0; }
.passkey-form { display: flex; flex-wrap: wrap; gap: 10px; align-items: flex-end; padding: var(--spacing-sm) 0; border-top: 1px dashed var(--color-hairline); }
.passkey-form .field { display: flex; flex-direction: column; gap: 4px; }
.passkey-form .vercel-input { min-width: 200px; }
.passkey-switch { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding-top: var(--spacing-sm); border-top: 1px dashed var(--color-hairline); }
.passkey-switch strong { font-size: 13px; color: var(--color-ink); }
.passkey-switch p { margin: 2px 0 0; font-size: 12px; color: var(--color-mute); }
@media (max-width: 640px) {
  .passkey-row, .passkey-switch { flex-direction: column; align-items: flex-start; }
}
</style>
