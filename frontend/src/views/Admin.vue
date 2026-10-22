<template>
  <div class="page-container admin-page">
    <div class="page-header">
      <h2>{{ tr('m_87354adb1c31') }}</h2>
      <p>{{ tr('m_74420630cb1f') }}</p>
    </div>

    <div class="admin-tabs">
      <button v-for="tab in tabs" :key="tab.key" type="button" class="admin-tab" :class="{ active: activeTab === tab.key }" @click="activeTab = tab.key">
        {{ tab.label }}
      </button>
    </div>

    <p v-if="message" class="admin-message" :class="{ error: messageIsError }">{{ message }}</p>

    <section v-show="activeTab === 'users'" class="admin-section">
      <div class="admin-card">
        <div class="admin-card-head">
          <h3>{{ tr('m_a4660b1813b3') }}</h3>
          <button class="btn btn-secondary" type="button" @click="showCreateUser = !showCreateUser">{{ showCreateUser ? tr('m_afd4b783536b') : tr('m_3ae9a33489d7') }}</button>
        </div>
        <form v-if="showCreateUser" class="admin-form" @submit.prevent="submitCreateUser">
          <input v-model="newUser.username" type="text" :placeholder="tr('m_1a3f0617d6de')" class="vercel-input" required />
          <input v-model="newUser.email" type="email" :placeholder="tr('m_504ebd6e7929')" class="vercel-input" />
          <input v-model="newUser.password" type="password" :placeholder="tr('m_f155bd296642')" class="vercel-input" required />
          <select v-model="newUser.groupId" class="vercel-input">
            <option value="">{{ tr('m_e69c528d68a8') }}</option>
            <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
          </select>
          <label class="admin-check"><input v-model="newUser.admin" type="checkbox" /> {{ tr('m_e19796712f1c') }}</label>
          <button class="btn btn-primary" type="submit" :disabled="busy">{{ tr('m_cde2cd071d25') }}</button>
        </form>
        <div class="admin-table-wrap">
          <table class="admin-table">
            <thead>
              <tr><th>{{ tr('m_1a3f0617d6de') }}</th><th>{{ tr('m_73075237fd0f') }}</th><th>{{ tr('m_c47b54e84e79') }}</th><th>{{ tr('m_963ead08d78d') }}</th><th>{{ tr('m_6320b4a8722a') }}</th><th>{{ tr('m_80527b7d0f96') }}</th><th>{{ tr('m_c85b3a16f0c8') }}</th><th>{{ tr('m_ed31fbb483ee') }}</th></tr>
            </thead>
            <tbody>
              <tr v-for="user in users" :key="user.id">
                <td class="mono">{{ user.username }}</td>
                <td>{{ user.email || '—' }}</td>
                <td><span class="tag" :class="user.role === 'admin' ? 'tag-admin' : ''">{{ user.role === 'admin' ? tr('m_e19796712f1c') : tr('m_0d0e1a86b3aa') }}</span></td>
                <td>{{ groupName(user.group_id) }}</td>
                <td><span class="tag" :class="user.status === 'active' ? 'tag-ok' : 'tag-down'">{{ user.status === 'active' ? tr('m_296de0e31f8c') : tr('m_bc5a87a757a5') }}</span></td>
                <td>
                  <span v-if="user.has_password !== false" class="tag">{{ tr('m_a621ab606db2') }}</span>
                  <span v-if="user.passkeys > 0" class="tag tag-ok">{{ user.passkeys }} {{ tr('m_a02cb2dac784') }}</span>
                  <span v-if="user.oidc_identities" class="tag tag-ok">{{ user.oidc_identities }} {{ tr('m_5c38a4a7fe36') }}</span>
                  <span v-if="user.has_password === false && !user.passkeys && !user.oidc_identities" class="tag tag-down">{{ tr('m_046a2463d762') }}</span>
                  <span v-if="user.password_login_disabled" class="tag tag-down">{{ tr('m_b7a378e767eb') }}</span>
                </td>
                <td>{{ formatTime(user.last_login_at) }}</td>
                <td class="actions">
                  <button class="btn btn-ghost" type="button" @click="toggleUserStatus(user)">{{ user.status === 'active' ? tr('m_7df5c456c765') : tr('m_f4f0ead1116b') }}</button>
                  <button class="btn btn-ghost" type="button" @click="changeGroup(user)">{{ tr('m_8dc3036181be') }}</button>
                  <button v-if="user.password_login_disabled" class="btn btn-ghost" type="button" @click="restorePasswordLogin(user)">{{ tr('m_466c8603a96a') }}</button>
                  <button class="btn btn-ghost" type="button" @click="resetPassword(user)">{{ tr('m_d70928debc2e') }}</button>
                  <button class="btn btn-ghost danger" type="button" @click="removeUser(user)">{{ tr('m_2f9daa828907') }}</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </section>

    <section v-show="activeTab === 'groups'" class="admin-section">
      <div class="admin-card">
        <div class="admin-card-head">
          <h3>{{ tr('m_963ead08d78d') }}</h3>
          <button class="btn btn-secondary" type="button" @click="showCreateGroup = !showCreateGroup">{{ showCreateGroup ? tr('m_afd4b783536b') : tr('m_e05eb0ea9c94') }}</button>
        </div>
        <form v-if="showCreateGroup" class="admin-form" @submit.prevent="submitCreateGroup">
          <input v-model="newGroupName" type="text" :placeholder="tr('m_744e71a6afc1')" class="vercel-input" required />
          <button class="btn btn-primary" type="submit" :disabled="busy">{{ tr('m_cde2cd071d25') }}</button>
        </form>
        <div v-for="group in groups" :key="group.id" class="group-row">
          <div class="group-title">
            <strong>{{ group.name }}</strong>
            <span v-if="group.builtin" class="tag">{{ tr('m_95e35aabd9a9') }}</span>
            <span class="group-count">{{ groupMemberCount(group.id) }} {{ tr('m_6f2f43e7dace') }}</span>
          </div>
          <div class="perm-list">
            <label v-for="perm in allPermissions" :key="perm" class="admin-check">
              <input type="checkbox" :checked="group.permissions.includes(perm)" :disabled="busy" @change="toggleGroupPerm(group, perm, ($event.target as HTMLInputElement).checked)" />
              {{ permissionLabels[perm] }}
            </label>
          </div>
          <div class="group-actions">
            <button class="btn btn-ghost" type="button" @click="renameGroup(group)">{{ tr('m_0d0cbac2eee5') }}</button>
            <button v-if="!group.builtin" class="btn btn-ghost danger" type="button" @click="removeGroup(group)">{{ tr('m_2f9daa828907') }}</button>
          </div>
        </div>
      </div>
    </section>

    <section v-show="activeTab === 'invites'" class="admin-section">
      <div class="admin-card">
        <div class="admin-card-head">
          <h3>{{ tr('m_c8b56dca7aed') }}</h3>
          <button class="btn btn-secondary" type="button" @click="showCreateInvite = !showCreateInvite">{{ showCreateInvite ? tr('m_afd4b783536b') : tr('m_5f6a55e484cc') }}</button>
        </div>
        <form v-if="showCreateInvite" class="admin-form" @submit.prevent="submitCreateInvite">
          <select v-model="newInvite.groupId" class="vercel-input">
            <option value="">{{ tr('m_e69c528d68a8') }}</option>
            <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
          </select>
          <input v-model.number="newInvite.maxUses" type="number" min="0" :placeholder="tr('m_62162c91428a')" class="vercel-input" />
          <input v-model.number="newInvite.expireDays" type="number" min="0" :placeholder="tr('m_5a52b819b316')" class="vercel-input" />
          <button class="btn btn-primary" type="submit" :disabled="busy">{{ tr('m_1ad1463fe16f') }}</button>
        </form>
        <div class="admin-table-wrap">
          <table class="admin-table">
            <thead>
              <tr><th>{{ tr('m_c8b56dca7aed') }}</th><th>{{ tr('m_963ead08d78d') }}</th><th>{{ tr('m_1e1fa1cd1221') }}</th><th>{{ tr('m_a8e5f1716600') }}</th><th>{{ tr('m_6320b4a8722a') }}</th><th>{{ tr('m_ed31fbb483ee') }}</th></tr>
            </thead>
            <tbody>
              <tr v-for="invite in invites" :key="invite.code">
                <td class="mono">{{ invite.code }}</td>
                <td>{{ groupName(invite.group_id) || tr('m_844b8cc8dff7') }}</td>
                <td>{{ invite.used_count }} / {{ invite.max_uses > 0 ? invite.max_uses : '∞' }}</td>
                <td>{{ invite.expires_at > 0 ? formatTime(invite.expires_at) : tr('m_3e71ccc89a43') }}</td>
                <td><span class="tag" :class="invite.enabled ? 'tag-ok' : 'tag-down'">{{ invite.enabled ? tr('m_f4f0ead1116b') : tr('m_4e6fd0e28c55') }}</span></td>
                <td class="actions">
                  <button class="btn btn-ghost" type="button" @click="toggleInvite(invite)">{{ invite.enabled ? tr('m_4e6fd0e28c55') : tr('m_f4f0ead1116b') }}</button>
                  <button class="btn btn-ghost danger" type="button" @click="removeInvite(invite)">{{ tr('m_2f9daa828907') }}</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </section>

    <section v-show="activeTab === 'settings'" class="admin-section">
      <div class="admin-card">
        <h3>{{ tr('m_b145d5c1a57b') }}</h3>
        <div class="setting-row">
          <span class="setting-label">{{ tr('m_58476be189a1') }}</span>
          <n-switch v-model:value="settings.experimental_features_enabled" size="small" :disabled="busy" @update:value="saveSettings" />
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
        <p class="text-muted">{{ tr('m_8e24fd96b54f') }}</p>
        <button class="btn btn-primary" type="button" :disabled="busy" @click="saveLabLimits">{{ tr('m_eedfd7ecc753') }}</button>
      </div>
      <div class="admin-card">
        <h3>{{ tr('m_5c4ef9d52d09') }}</h3>
        <div class="setting-row">
          <span class="setting-label">{{ tr('m_04f83ab519b2') }}</span>
          <n-switch v-model:value="regOpen" size="small" @update:value="saveSettings">
            <template #checked>{{ tr('m_c14c915da5fb') }}</template>
            <template #unchecked>{{ tr('m_3fd47edce45b') }}</template>
          </n-switch>
        </div>
        <div class="setting-row">
          <span class="setting-label">{{ tr('m_25cece360c1c') }}</span>
          <n-switch v-model:value="emailVerify" size="small" :disabled="!smtpReady || !regOpen" @update:value="saveSettings">
            <template #checked>{{ tr('m_59a37a575fd7') }}</template>
            <template #unchecked>{{ tr('m_c6ca814dc1e4') }}</template>
          </n-switch>
          <small v-if="!smtpReady" class="text-muted">{{ tr('m_d63dc99a4520') }}</small>
        </div>
        <div class="setting-row">
          <span class="setting-label">{{ tr('m_c8b56dca7aed') }}</span>
          <label class="admin-check"><input v-model="settings.invite_mode" type="radio" value="off" @change="saveSettings" /> {{ tr('m_3fd47edce45b') }}</label>
          <label class="admin-check"><input v-model="settings.invite_mode" type="radio" value="optional" @change="saveSettings" /> {{ tr('m_ebdcbdcf9c99') }}</label>
          <label class="admin-check"><input v-model="settings.invite_mode" type="radio" value="required" @change="saveSettings" /> {{ tr('m_11da9dc44285') }}</label>
        </div>
        <div class="setting-row">
          <span class="setting-label">{{ tr('m_e69c528d68a8') }}</span>
          <select v-model="settings.default_group_id" class="vercel-input narrow" @change="saveSettings">
            <option value="">{{ tr('m_ccb91ce53df9') }}</option>
            <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
          </select>
        </div>
      </div>

      <div class="admin-card">
        <h3>{{ tr('m_a0f79e91f13f') }}</h3>
        <p class="admin-hint">{{ tr('m_b3e7709d53fd') }}</p>
        <div class="setting-row">
          <span class="setting-label">{{ tr('m_5c4c49a2ae6b') }}</span>
          <select v-model.number="settings.audit_retention_days" class="vercel-input narrow" @change="saveSettings">
            <option :value="30">{{ tr('m_84ad2952a308') }}</option>
            <option :value="90">{{ tr('m_cb82f419192b') }}</option>
            <option :value="180">{{ tr('m_81a524216893') }}</option>
            <option :value="365">{{ tr('m_ef95232cba01') }}</option>
            <option :value="-1">{{ tr('m_14b99ee953f8') }}</option>
          </select>
          <small class="text-muted">{{ tr('m_43b7c1573e9d') }}</small>
        </div>
      </div>

      <div class="admin-card">
        <h3>{{ tr('m_102824e51a9d') }}</h3>
        <p class="admin-hint">{{ tr('m_7cb7b8eda6aa') }}</p>
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
          <small v-if="!settings.passkey_admin_ready" class="text-muted">{{ tr('m_cbfc19c348ad') }}</small>
        </div>
      </div>

      <div class="admin-card">
        <h3>{{ tr('m_37c24b911b27') }}</h3>
        <p class="admin-hint">{{ tr('m_924475a68b09') }}</p>
        <div class="setting-row">
          <span class="setting-label">{{ tr('m_829140bedce5') }}</span>
          <n-switch v-model:value="settings.rate_limit_enabled" size="small" @update:value="saveSettings" />
        </div>
        <div class="admin-form">
          <label class="field-label">{{ tr('m_671a08f4e738') }}<small>{{ tr('m_6728255e4bdb') }}</small></label>
          <input v-model.number="settings.rate_limit_per_account" type="number" class="vercel-input" />
          <label class="field-label">{{ tr('m_cf5a28cff90f') }}<small>{{ tr('m_f1d1405b8d84') }}</small></label>
          <input v-model.number="settings.rate_limit_per_ip" type="number" class="vercel-input" />
          <label class="field-label">{{ tr('m_088598d743e5') }}<small>{{ tr('m_5f566691080a') }}</small></label>
          <input v-model.number="settings.rate_limit_window_minutes" type="number" class="vercel-input" />
          <label class="field-label">{{ tr('m_b9ca02da2aa9') }}<small>{{ tr('m_0fc65b562a9c') }}</small></label>
          <input v-model.number="settings.rate_limit_familiar_multiplier" type="number" class="vercel-input" />
          <button class="btn btn-primary" type="button" :disabled="busy" @click="saveSettings">{{ tr('m_a3030bf8f16d') }}</button>
        </div>
        <div class="setting-row">
          <span class="setting-label">{{ tr('m_ab5fdea0b36d') }}</span>
          <n-switch v-model:value="settings.rate_limit_notify" size="small" @update:value="saveSettings" />
          <small class="text-muted">{{ tr('m_832ce410f6e7') }}</small>
        </div>
      </div>

      <div class="admin-card">
        <h3>{{ tr('m_e4f6aa5a185d') }}</h3>
        <p class="admin-hint">{{ tr('m_7eb357ebb73a') }}</p>
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
        <h3>{{ tr('m_dbf3b43ce49a') }}</h3>
        <p class="admin-hint">{{ tr('m_39a974380a04') }}</p>
        <div class="admin-form">
          <input v-model="encKeyInput" type="text" :placeholder="tr('m_6171d4f4500d')" class="vercel-input" />
          <button class="btn btn-primary" type="button" :disabled="busy || !encKeyInput" @click="saveEncKey">{{ tr('m_a334bb468aba') }}</button>
          <span class="tag">{{ tr('m_1f45e225299d') }}{{ encKeySource === 'env' ? tr('m_ae27b474ea4d') : encKeySource === 'stored' ? tr('m_263200d7d529') : tr('m_2f5f1d6fbfb0') }}</span>
        </div>
      </div>
    </section>

    <section v-if="activeTab === 'oidc'" class="admin-section"><OIDCProviders /></section>
    <section v-show="activeTab === 'smtp'" class="admin-section">
      <div class="admin-card">
        <h3>{{ tr('m_dd15bdd9da5a') }}</h3>
        <p class="admin-hint">{{ tr('m_25ed71be1ba5') }}</p>
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
    </section>

    <section v-show="activeTab === 'audit'" class="admin-section">
      <div class="admin-card">
        <div class="admin-card-head">
          <h3>{{ tr('m_a0f79e91f13f') }}</h3>
          <button class="btn btn-secondary" type="button" :disabled="auditLoading" @click="loadAudit">{{ tr('m_aee887434131') }}</button>
        </div>
        <div class="audit-stats">
          <div class="audit-stat">
            <span class="audit-stat-value">{{ auditStats.total }}</span>
            <span class="audit-stat-label">{{ tr('m_fac4970dabd7') }}</span>
          </div>
          <div class="audit-stat">
            <span class="audit-stat-value" :class="{ danger: auditStats.failed > 0 }">{{ auditStats.failed }}</span>
            <span class="audit-stat-label">{{ tr('m_abcce0f9904a') }}</span>
          </div>
          <div class="audit-stat">
            <span class="audit-stat-value">{{ auditStats.today }}</span>
            <span class="audit-stat-label">{{ tr('m_39def77a5416') }}</span>
          </div>
          <div class="audit-stat">
            <span class="audit-stat-value">{{ auditStats.actors }}</span>
            <span class="audit-stat-label">{{ tr('m_2f4406f304fa') }}</span>
          </div>
        </div>
        <form class="admin-form audit-filters" @submit.prevent="applyAuditFilters">
          <input v-model="auditFilters.actor" type="text" :placeholder="tr('m_e412e5321b6d')" class="vercel-input" />
          <select v-model="auditFilters.action" class="vercel-input">
            <option value="">{{ tr('m_7e093a4e23d8') }}</option>
            <optgroup v-for="group in auditActionGroups" :key="group.category" :label="group.label">
              <option v-for="item in group.items" :key="item.action" :value="item.action">{{ item.label }}</option>
            </optgroup>
          </select>
          <input v-model="auditFilters.fromDate" type="date" class="vercel-input narrow" :aria-label="tr('m_760506491eef')" />
          <span class="text-muted">{{ tr('m_65edbf9f1c67') }}</span>
          <input v-model="auditFilters.toDate" type="date" class="vercel-input narrow" :aria-label="tr('m_895cd52fbbb6')" />
          <button class="btn btn-primary" type="submit" :disabled="auditLoading">{{ tr('m_bcd6771e08ec') }}</button>
          <button class="btn btn-secondary" type="button" :disabled="auditLoading" @click="resetAuditFilters">{{ tr('m_cb5d682bac3d') }}</button>
        </form>
        <div class="admin-table-wrap">
          <table class="admin-table">
            <thead>
              <tr><th>{{ tr('m_8b6ff498515b') }}</th><th>{{ tr('m_9689906e74a1') }}</th><th>{{ tr('m_ed31fbb483ee') }}</th><th>{{ tr('m_53f92c063963') }}</th><th>{{ tr('m_4bf0843a4518') }}</th><th>{{ tr('m_bb7ef73495dd') }}</th></tr>
            </thead>
            <tbody>
              <tr v-for="log in auditLogs" :key="log.id">
                <td>{{ formatTime(log.created_at) }}</td>
                <td class="mono">{{ log.actor_name || '—' }}</td>
                <td>{{ auditActionLabel(log) }}</td>
                <td>{{ log.target || '—' }}</td>
                <td class="mono">{{ log.ip || '—' }}</td>
                <td><span class="tag" :class="log.success ? 'tag-ok' : 'tag-down'">{{ log.success ? tr('m_053461ce86d2') : tr('m_28384d7afd2e') }}</span></td>
              </tr>
              <tr v-if="!auditLoading && !auditLogs.length">
                <td colspan="6" class="text-muted">{{ tr('m_da3a855ef2ea') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="audit-pager">
          <span class="text-muted">{{ tr('m_76e547a8fa54') }} {{ auditTotal }} {{ tr('m_f004f1d84cf9') }}</span>
          <button class="btn btn-ghost" type="button" :disabled="auditLoading || auditPage <= 1" @click="changeAuditPage(auditPage - 1)">{{ tr('m_c9b9ae7a6144') }}</button>
          <span class="text-muted">{{ tr('m_af50a0fe2a50') }} {{ auditPage }} / {{ auditPageCount }} {{ tr('m_d24d3c99460c') }}</span>
          <button class="btn btn-ghost" type="button" :disabled="auditLoading || auditPage >= auditPageCount" @click="changeAuditPage(auditPage + 1)">{{ tr('m_8a8542f69648') }}</button>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { usePanelDialogs } from '../composables/usePanelDialogs'
const panel = usePanelDialogs()
import { locale, t as tr, localized } from '../i18n'

import { computed, onMounted, ref, watch } from 'vue'
import { useMessage, NSwitch } from 'naive-ui'
import { useConfigStore } from '../stores/config'
import OIDCProviders from '../components/OIDCProviders.vue'
import { validateLabLimits } from '../utils/lab'
import {
  listUsers, createUser, setUserStatus, setUserGroup, resetUserPassword, deleteUser, setUserPasswordLogin,
  listGroups, createGroup, updateGroup, deleteGroup,
  listInvites, createInvite, updateInvite, deleteInvite,
  getAppSettings, updateAppSettings,
  getSMTP, updateSMTP, testSMTP,
  getOAuthConfig, updateOAuthConfig,
  getEncryptionKeyStatus, saveEncryptionKey,
  listAuditLogs, getAuditStats, auditActionLabel, AUDIT_CATEGORY_LABELS, AUDIT_ACTIONS,
  ALL_PERMISSIONS, PERMISSION_LABELS,
  type UserView, type UserGroup, type Invite, type AppSettings,
  type AuditLog, type AuditStats, type AuditQuery,
} from '../api/admin'

const tabs = localized(() => ([
  { key: 'users', label: tr('m_0d0e1a86b3aa') },
  { key: 'groups', label: tr('m_963ead08d78d') },
  { key: 'invites', label: tr('m_c8b56dca7aed') },
  { key: 'settings', label: tr('m_68ea5dd4d7af') },
  { key: 'oidc', label: tr('m_0896be102d49') },
  { key: 'smtp', label: tr('m_0ac53099a515') },
  { key: 'audit', label: tr('m_a0f79e91f13f') },
]))
const activeTab = ref('users')
const busy = ref(false)
const message = ref('')
const messageIsError = ref(false)
const toast = useMessage()
const configStore = useConfigStore()

const users = ref<UserView[]>([])
const groups = ref<UserGroup[]>([])
const invites = ref<Invite[]>([])
const settings = ref<AppSettings>({ registration_enabled: true, invite_mode: 'off', default_group_id: '' })
const turnstile = ref({ enabled: false, site_key: '' })
const turnstileSecret = ref('')
const turnstileHasSecret = ref(false)
const allPermissions = ALL_PERMISSIONS
const permissionLabels = PERMISSION_LABELS

const showCreateUser = ref(false)
const showCreateGroup = ref(false)
const showCreateInvite = ref(false)
const newUser = ref({ username: '', email: '', password: '', groupId: '', admin: false })
const newGroupName = ref('')
const newInvite = ref<{ groupId: string; maxUses: number | null; expireDays: number | null }>({ groupId: '', maxUses: null, expireDays: null })
const smtp = ref({ host: '', port: 587, username: '', password: '', from: '', tlsMode: 'starttls' })
const testMailTo = ref('')
const testingMail = ref(false)
const oauthCfg = ref({ client_id: '', client_secret: '', redirect_uri: '', scopes: '' })
const oauthHasSecret = ref(false)
const smtpReady = ref(false)
const encKeyInput = ref('')
const encKeySource = ref('none')
const regOpen = computed({
  get: () => settings.value.registration_enabled,
  set: (v: boolean) => { settings.value.registration_enabled = v },
})
const passwordLoginOff = computed({
  get: () => !!settings.value.password_login_disabled,
  set: (v: boolean) => { settings.value.password_login_disabled = v },
})
const emailVerify = computed({
  get: () => !settings.value.email_verify_disabled,
  set: (v: boolean) => { settings.value.email_verify_disabled = !v },
})

function notify(text: string, isError = false) {
  message.value = text
  messageIsError.value = isError
  if (!isError) window.setTimeout(() => { message.value = '' }, 2500)
}

function groupName(id: string) {
  const group = groups.value.find((g) => g.id === id)
  return group ? group.name : ''
}

function groupMemberCount(id: string) {
  return users.value.filter((u) => u.group_id === id).length
}

function formatTime(seconds: number) {
  if (!seconds) return '—'
  return new Date(seconds * 1000).toLocaleString(locale.value)
}

const auditLogs = ref<AuditLog[]>([])
const auditStats = ref<AuditStats>({ total: 0, failed: 0, today: 0, actors: 0 })
const auditTotal = ref(0)
const auditPage = ref(1)
const auditPageSize = 50
const auditLoading = ref(false)
const auditFilters = ref({ actor: '', action: '', fromDate: '', toDate: '' })
const auditPageCount = computed(() => Math.max(1, Math.ceil(auditTotal.value / auditPageSize)))
const auditActionGroups = computed(() =>
  Object.entries(AUDIT_CATEGORY_LABELS)
    .map(([category, label]) => ({ category, label, items: AUDIT_ACTIONS.filter((item) => item.category === category) }))
    .filter((group) => group.items.length > 0),
)

/** 日期输入（YYYY-MM-DD）转换为当天起止的 Unix 秒。 */
function auditDayStart(value: string) {
  return value ? Math.floor(new Date(value + 'T00:00:00').getTime() / 1000) : 0
}

function auditDayEnd(value: string) {
  return value ? Math.floor(new Date(value + 'T23:59:59').getTime() / 1000) : 0
}

async function loadAudit() {
  auditLoading.value = true
  try {
    const params: AuditQuery = { page: auditPage.value, page_size: auditPageSize }
    const actor = auditFilters.value.actor.trim()
    if (actor) params.actor = actor
    if (auditFilters.value.action) params.action = auditFilters.value.action
    const from = auditDayStart(auditFilters.value.fromDate)
    const to = auditDayEnd(auditFilters.value.toDate)
    if (from) params.from = from
    if (to) params.to = to
    const [page, stats] = await Promise.all([listAuditLogs(params), getAuditStats(7)])
    auditLogs.value = page.data.logs
    auditTotal.value = page.data.total
    auditPage.value = page.data.page
    auditStats.value = stats.data
  } catch (e: any) {
    notify(tr('m_df08a37fb547') + (e.response?.data?.error || e.message), true)
  } finally {
    auditLoading.value = false
  }
}

function applyAuditFilters() {
  auditPage.value = 1
  void loadAudit()
}

function resetAuditFilters() {
  auditFilters.value = { actor: '', action: '', fromDate: '', toDate: '' }
  auditPage.value = 1
  void loadAudit()
}

function changeAuditPage(page: number) {
  auditPage.value = Math.min(Math.max(1, page), auditPageCount.value)
  void loadAudit()
}

async function loadAll() {
  busy.value = true
  try {
    const [u, g, i, s] = await Promise.all([listUsers(), listGroups(), listInvites(), getAppSettings()])
    users.value = u.data.users
    groups.value = g.data.groups
    invites.value = i.data.invites
    // 0/未设置表示默认窗口（90 天），-1 表示永久保留。
    settings.value = { ...s.data, audit_retention_days: s.data.audit_retention_days || 90,
      lab_daily_request_limit: s.data.lab_daily_request_limit ?? 100000,
      lab_requests_per_second: s.data.lab_requests_per_second ?? 10,
      lab_max_workers: s.data.lab_max_workers ?? 32 }
    configStore.setExperimentalFeatures(!!s.data.experimental_features_enabled)
    turnstile.value.enabled = !!s.data.turnstile_enabled
    turnstile.value.site_key = s.data.turnstile_site_key || ''
    turnstileHasSecret.value = !!s.data.turnstile_has_secret
    turnstileSecret.value = ''
  } catch (e: any) {
    notify(tr('m_08417baa06d4') + (e.response?.data?.error || e.message), true)
  } finally {
    busy.value = false
  }
}

async function run(action: () => Promise<unknown>, success: string) {
  busy.value = true
  try {
    await action()
    notify(success)
    await loadAll()
  } catch (e: any) {
    notify(e.response?.data?.error || tr('m_0c3b4cf7aa25'), true)
  } finally {
    busy.value = false
  }
}

function submitCreateUser() {
  const payload = {
    username: newUser.value.username.trim(),
    email: newUser.value.email.trim() || undefined,
    password: newUser.value.password,
    role: newUser.value.admin ? 'admin' : 'user',
    group_id: newUser.value.groupId || undefined,
  }
  void run(async () => {
    await createUser(payload)
    newUser.value = { username: '', email: '', password: '', groupId: '', admin: false }
    showCreateUser.value = false
  }, tr('m_d4f7d0bf6262'))
}

function toggleUserStatus(user: UserView) {
  const target = user.status === 'active' ? 'disabled' : 'active'
  void run(() => setUserStatus(user.id, target), target === 'active' ? tr('m_dfb802238b38') : tr('m_bc5a87a757a5'))
}

async function changeGroup(user: UserView) {
  const options = groups.value.map((g) => g.name + ':' + g.id).join('  ')
  const picked = await panel.prompt(tr('m_1596494489dd') + options + tr('m_f7f694838c28'), user.group_id)
  if (picked === null) return
  void run(() => setUserGroup(user.id, picked.trim()), tr('m_9afb25dec846'))
}

function restorePasswordLogin(user: UserView) {
  void run(() => setUserPasswordLogin(user.id, false), tr('m_744b48e64720') + user.username + tr('m_32a8b6e621a5'))
}

async function resetPassword(user: UserView) {
  const password = await panel.prompt(tr('m_49c073f4fa15') + user.username + tr('m_be823aab4fc0'), '', true)
  if (!password) return
  void run(() => resetUserPassword(user.id, password), tr('m_4c5c1892de86'))
}

async function removeUser(user: UserView) {
  if (!await panel.confirm(tr('m_511770d654f5') + user.username + tr('m_26ff3d7383ae'))) return
  void run(() => deleteUser(user.id), tr('m_7ad39d9c27ab'))
}

function submitCreateGroup() {
  const name = newGroupName.value.trim()
  if (!name) return
  void run(async () => {
    await createGroup(name, [])
    newGroupName.value = ''
    showCreateGroup.value = false
  }, tr('m_a44fb45d0f39'))
}

function toggleGroupPerm(group: UserGroup, perm: string, on: boolean) {
  const next = on ? group.permissions.concat(perm) : group.permissions.filter((p) => p !== perm)
  void run(() => updateGroup(group.id, group.name, next), tr('m_d72fbc805af9'))
}

async function renameGroup(group: UserGroup) {
  if (group.builtin) {
    notify(tr('m_e29e29f7c5d3'), true)
    return
  }
  const name = await panel.prompt(tr('m_e08ae7e4a8bf'), group.name)
  if (!name || name === group.name) return
  void run(() => updateGroup(group.id, name, group.permissions), tr('m_06afd1f0352e'))
}

async function removeGroup(group: UserGroup) {
  if (!await panel.confirm(tr('m_5a398d79b24c') + group.name + '？')) return
  void run(() => deleteGroup(group.id), tr('m_2194537e8be2'))
}

function submitCreateInvite() {
  void run(async () => {
    await createInvite({
      group_id: newInvite.value.groupId || undefined,
      max_uses: Number(newInvite.value.maxUses) > 0 ? Number(newInvite.value.maxUses) : 0,
      expires_at: Number(newInvite.value.expireDays) > 0 ? Math.floor(Date.now() / 1000) + Number(newInvite.value.expireDays) * 86400 : 0,
    })
    showCreateInvite.value = false
  }, tr('m_67021bc43cbd'))
}

function toggleInvite(invite: Invite) {
  void run(() => updateInvite(invite.code, !invite.enabled), invite.enabled ? tr('m_a8c3698b5b8c') : tr('m_dfb802238b38'))
}

async function removeInvite(invite: Invite) {
  if (!await panel.confirm(tr('m_759aeb16d395') + invite.code + '？')) return
  void run(() => deleteInvite(invite.code), tr('m_c714402bd321'))
}

function saveSettings() {
  void run(async () => {
    await updateAppSettings(settings.value)
    configStore.setExperimentalFeatures(!!settings.value.experimental_features_enabled)
  }, tr('m_bd03c0aad701'))
}

function saveLabLimits() {
  const error = validateLabLimits(settings.value)
  if (error) return notify(error, true)
  saveSettings()
}

function saveTurnstile() {
  // 带上完整设置：只保存人机验证时不能把通行密钥、审计保留期等字段清空。
  const payload: Partial<AppSettings> & { turnstile_secret?: string } = {
    ...settings.value,
    registration_enabled: settings.value.registration_enabled,
    invite_mode: settings.value.invite_mode,
    default_group_id: settings.value.default_group_id || '',
    email_verify_disabled: !!settings.value.email_verify_disabled,
    turnstile_enabled: turnstile.value.enabled,
    turnstile_site_key: turnstile.value.site_key.trim(),
  }
  if (turnstileSecret.value) payload.turnstile_secret = turnstileSecret.value
  void run(() => updateAppSettings(payload), tr('m_763363acbbeb'))
}

// 审计日志按需加载：切到该标签页时才请求，避免每次进入后台都多两次查询。
watch(activeTab, (tab) => {
  if (tab === 'audit') void loadAudit()
})

onMounted(() => {
  void loadAll()
  getSMTP().then(({ data }) => {
    smtp.value.host = data.host
    smtp.value.port = data.port || 587
    smtp.value.username = data.username
    smtp.value.from = data.from
    smtp.value.tlsMode = data.tls_mode === 'plain' ? 'plain' : 'ssl'
    smtpReady.value = data.configured
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

async function saveEncKey() {
  if (busy.value) return
  busy.value = true
  try {
    const { data } = await saveEncryptionKey(encKeyInput.value.trim())
    toast.success(data.message || tr('m_1bd91a7d0c53'))
    encKeyInput.value = ''
    const status = await getEncryptionKeyStatus()
    encKeySource.value = status.data.source
  } catch (e: any) {
    toast.error(e.response?.data?.error || tr('m_6309a3bb5ba4'))
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
    toast.success(tr('m_12c3bd5c0eeb'))
  } catch (e: any) {
    toast.error(e.response?.data?.error || tr('m_6309a3bb5ba4'))
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
    toast.success(data.configured ? tr('m_dddcb690d105') : tr('m_7afbd323b652'))
    smtp.value.password = ''
  } catch (e: any) {
    toast.error(e.response?.data?.error || tr('m_6309a3bb5ba4'))
  } finally {
    busy.value = false
  }
}

async function sendTestMail() {
  if (!testMailTo.value.trim()) {
    toast.error(tr('m_e3f39ac6f6c9'))
    return
  }
  testingMail.value = true
  try {
    const { data } = await testSMTP(testMailTo.value.trim())
    toast.success(data.message || tr('m_cfb7989b7a73'))
  } catch (e: any) {
    toast.error(e.response?.data?.error || tr('m_ac77953a1e06'))
  } finally {
    testingMail.value = false
  }
}
</script>

<style scoped>
.admin-tabs { display: flex; gap: 6px; margin-bottom: var(--spacing-lg); flex-wrap: wrap; }
.admin-tab {
  padding: 8px 18px;
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-md);
  background: var(--color-canvas-raised);
  color: var(--color-mute);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}
.admin-tab.active { color: var(--color-ink); border-color: var(--color-link); font-weight: 600; }
.admin-message {
  padding: 8px 12px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-hairline);
  font-size: 13px;
  margin: 0 0 var(--spacing-md);
}
.admin-message.error { color: var(--color-error); }
.admin-card {
  background: var(--color-canvas-raised);
  border: 1px solid var(--color-hairline);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
  margin-bottom: var(--spacing-md);
}
.admin-card-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--spacing-md); }
.admin-card h3 { margin: 0; font-size: 15px; color: var(--color-ink); }
.admin-form { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; padding: var(--spacing-sm) 0 var(--spacing-md); border-bottom: 1px dashed var(--color-hairline); margin-bottom: var(--spacing-sm); }
.admin-form .vercel-input { max-width: 260px; }
.admin-check { display: inline-flex; align-items: center; gap: 6px; font-size: 13px; color: var(--color-ink); cursor: pointer; }
.admin-table-wrap { overflow-x: auto; }
.admin-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.admin-table th, .admin-table td { text-align: left; padding: 8px 10px; border-bottom: 1px solid var(--color-hairline); white-space: nowrap; }
.admin-table th { color: var(--color-mute); font-weight: 500; font-size: 12px; }
.admin-table td.mono { font-family: var(--font-mono); }
.admin-table td.actions { display: flex; gap: 4px; flex-wrap: wrap; }
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
.tag-admin { color: var(--color-link); border-color: currentColor; }
.group-row { padding: var(--spacing-sm) 0; border-bottom: 1px solid var(--color-hairline); }
.group-title { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.group-title strong { font-size: 14px; color: var(--color-ink); }
.group-count { font-size: 12px; color: var(--color-mute); }
.perm-list { display: flex; flex-wrap: wrap; gap: 12px; margin-bottom: 6px; }
.group-actions { display: flex; gap: 4px; }
.setting-row { display: flex; align-items: center; gap: 14px; padding: var(--spacing-sm) 0; }
.setting-label { font-size: 13px; color: var(--color-ink); min-width: 96px; }
.vercel-input.narrow { max-width: 240px; }
.btn.danger { color: var(--color-error); }
.admin-form .btn { padding: 5px 14px; font-size: 12px; }
.admin-form .vercel-input { height: auto; }
.admin-hint { font-size: 12px; color: var(--color-mute); margin: 0 0 var(--spacing-sm); }
/* 登录保护表单：数字输入上面各带一行说明。 */
.field-label { display: flex; flex-direction: column; gap: 2px; font-size: 12px; color: var(--color-body); }
.field-label small { font-size: 11px; color: var(--color-mute); }
.smtp-form { border-bottom: none; margin-bottom: 0; }
.audit-stats { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; margin-bottom: var(--spacing-md); }
.audit-stat { display: flex; flex-direction: column; gap: 2px; padding: 10px 12px; border: 1px solid var(--color-hairline); border-radius: var(--radius-md); background: var(--color-canvas-soft); }
.audit-stat-value { font-size: 20px; font-weight: 600; color: var(--color-ink); }
.audit-stat-value.danger { color: var(--color-error); }
.audit-stat-label { font-size: 12px; color: var(--color-mute); }
.audit-filters { align-items: center; }
.audit-pager { display: flex; align-items: center; gap: 10px; margin-top: var(--spacing-sm); flex-wrap: wrap; }
@media (max-width: 640px) { .audit-stats { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
