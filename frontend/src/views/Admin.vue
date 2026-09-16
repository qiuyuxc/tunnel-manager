<template>
  <div class="page-container admin-page">
    <div class="page-header">
      <h2>管理后台</h2>
      <p>管理用户、用户组、邀请码与注册策略</p>
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
          <h3>用户列表</h3>
          <button class="btn btn-secondary" type="button" @click="showCreateUser = !showCreateUser">{{ showCreateUser ? '收起' : '新建用户' }}</button>
        </div>
        <form v-if="showCreateUser" class="admin-form" @submit.prevent="submitCreateUser">
          <input v-model="newUser.username" type="text" placeholder="用户名" class="vercel-input" required />
          <input v-model="newUser.email" type="email" placeholder="邮箱（可留空）" class="vercel-input" />
          <input v-model="newUser.password" type="password" placeholder="初始密码（≥6 位）" class="vercel-input" required />
          <select v-model="newUser.groupId" class="vercel-input">
            <option value="">默认用户组</option>
            <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
          </select>
          <label class="admin-check"><input v-model="newUser.admin" type="checkbox" /> 管理员</label>
          <button class="btn btn-primary" type="submit" :disabled="busy">创建</button>
        </form>
        <div class="admin-table-wrap">
          <table class="admin-table">
            <thead>
              <tr><th>用户名</th><th>邮箱</th><th>角色</th><th>用户组</th><th>状态</th><th>登录方式</th><th>最近登录</th><th>操作</th></tr>
            </thead>
            <tbody>
              <tr v-for="user in users" :key="user.id">
                <td class="mono">{{ user.username }}</td>
                <td>{{ user.email || '—' }}</td>
                <td><span class="tag" :class="user.role === 'admin' ? 'tag-admin' : ''">{{ user.role === 'admin' ? '管理员' : '用户' }}</span></td>
                <td>{{ groupName(user.group_id) }}</td>
                <td><span class="tag" :class="user.status === 'active' ? 'tag-ok' : 'tag-down'">{{ user.status === 'active' ? '正常' : '已禁用' }}</span></td>
                <td>
                  <span class="tag" :class="user.passkeys > 0 ? 'tag-ok' : ''">{{ user.passkeys > 0 ? user.passkeys + ' 个通行密钥' : '仅密码' }}</span>
                  <span v-if="user.password_login_disabled" class="tag tag-down">已禁用密码</span>
                </td>
                <td>{{ formatTime(user.last_login_at) }}</td>
                <td class="actions">
                  <button class="btn btn-ghost" type="button" @click="toggleUserStatus(user)">{{ user.status === 'active' ? '禁用' : '启用' }}</button>
                  <button class="btn btn-ghost" type="button" @click="changeGroup(user)">改组</button>
                  <button v-if="user.password_login_disabled" class="btn btn-ghost" type="button" @click="restorePasswordLogin(user)">恢复密码登录</button>
                  <button class="btn btn-ghost" type="button" @click="resetPassword(user)">重置密码</button>
                  <button class="btn btn-ghost danger" type="button" @click="removeUser(user)">删除</button>
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
          <h3>用户组</h3>
          <button class="btn btn-secondary" type="button" @click="showCreateGroup = !showCreateGroup">{{ showCreateGroup ? '收起' : '新建用户组' }}</button>
        </div>
        <form v-if="showCreateGroup" class="admin-form" @submit.prevent="submitCreateGroup">
          <input v-model="newGroupName" type="text" placeholder="用户组名称" class="vercel-input" required />
          <button class="btn btn-primary" type="submit" :disabled="busy">创建</button>
        </form>
        <div v-for="group in groups" :key="group.id" class="group-row">
          <div class="group-title">
            <strong>{{ group.name }}</strong>
            <span v-if="group.builtin" class="tag">内置</span>
            <span class="group-count">{{ groupMemberCount(group.id) }} 名成员</span>
          </div>
          <div class="perm-list">
            <label v-for="perm in allPermissions" :key="perm" class="admin-check">
              <input type="checkbox" :checked="group.permissions.includes(perm)" :disabled="busy" @change="toggleGroupPerm(group, perm, ($event.target as HTMLInputElement).checked)" />
              {{ permissionLabels[perm] }}
            </label>
          </div>
          <div class="group-actions">
            <button class="btn btn-ghost" type="button" @click="renameGroup(group)">重命名</button>
            <button v-if="!group.builtin" class="btn btn-ghost danger" type="button" @click="removeGroup(group)">删除</button>
          </div>
        </div>
      </div>
    </section>

    <section v-show="activeTab === 'invites'" class="admin-section">
      <div class="admin-card">
        <div class="admin-card-head">
          <h3>邀请码</h3>
          <button class="btn btn-secondary" type="button" @click="showCreateInvite = !showCreateInvite">{{ showCreateInvite ? '收起' : '生成邀请码' }}</button>
        </div>
        <form v-if="showCreateInvite" class="admin-form" @submit.prevent="submitCreateInvite">
          <select v-model="newInvite.groupId" class="vercel-input">
            <option value="">默认用户组</option>
            <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
          </select>
          <input v-model.number="newInvite.maxUses" type="number" min="0" placeholder="可用次数（0 = 不限）" class="vercel-input" />
          <input v-model.number="newInvite.expireDays" type="number" min="0" placeholder="有效天数（0 = 永久）" class="vercel-input" />
          <button class="btn btn-primary" type="submit" :disabled="busy">生成</button>
        </form>
        <div class="admin-table-wrap">
          <table class="admin-table">
            <thead>
              <tr><th>邀请码</th><th>用户组</th><th>已用 / 上限</th><th>过期时间</th><th>状态</th><th>操作</th></tr>
            </thead>
            <tbody>
              <tr v-for="invite in invites" :key="invite.code">
                <td class="mono">{{ invite.code }}</td>
                <td>{{ groupName(invite.group_id) || '默认' }}</td>
                <td>{{ invite.used_count }} / {{ invite.max_uses > 0 ? invite.max_uses : '∞' }}</td>
                <td>{{ invite.expires_at > 0 ? formatTime(invite.expires_at) : '永久' }}</td>
                <td><span class="tag" :class="invite.enabled ? 'tag-ok' : 'tag-down'">{{ invite.enabled ? '启用' : '停用' }}</span></td>
                <td class="actions">
                  <button class="btn btn-ghost" type="button" @click="toggleInvite(invite)">{{ invite.enabled ? '停用' : '启用' }}</button>
                  <button class="btn btn-ghost danger" type="button" @click="removeInvite(invite)">删除</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </section>

    <section v-show="activeTab === 'settings'" class="admin-section">
      <div class="admin-card">
        <h3>注册策略</h3>
        <div class="setting-row">
          <span class="setting-label">开放注册</span>
          <n-switch v-model:value="regOpen" size="small" @update:value="saveSettings">
            <template #checked>开放</template>
            <template #unchecked>关闭</template>
          </n-switch>
        </div>
        <div class="setting-row">
          <span class="setting-label">注册邮箱验证</span>
          <n-switch v-model:value="emailVerify" size="small" :disabled="!smtpReady || !regOpen" @update:value="saveSettings">
            <template #checked>需要验证码</template>
            <template #unchecked>不需要</template>
          </n-switch>
          <small v-if="!smtpReady" class="text-muted">未配置邮件服务，验证码无法发送</small>
        </div>
        <div class="setting-row">
          <span class="setting-label">邀请码</span>
          <label class="admin-check"><input v-model="settings.invite_mode" type="radio" value="off" @change="saveSettings" /> 关闭</label>
          <label class="admin-check"><input v-model="settings.invite_mode" type="radio" value="optional" @change="saveSettings" /> 选填</label>
          <label class="admin-check"><input v-model="settings.invite_mode" type="radio" value="required" @change="saveSettings" /> 必填</label>
        </div>
        <div class="setting-row">
          <span class="setting-label">默认用户组</span>
          <select v-model="settings.default_group_id" class="vercel-input narrow" @change="saveSettings">
            <option value="">（未设置）</option>
            <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
          </select>
        </div>
      </div>

      <div class="admin-card">
        <h3>实验性功能</h3>
        <p class="admin-hint">开启后，管理员侧边栏会显示“IP 优选实验室”。该功能允许直连指定 IP 段探测 Host/SNI 可用性与延迟，并可选择自动更新华为云 DNS。关闭后入口与 API 均不暴露。</p>
        <div class="setting-row">
          <span class="setting-label">开启实验性功能</span>
          <n-switch v-model:value="experimentalFeatures" size="small" @update:value="saveSettings">
            <template #checked>开启</template>
            <template #unchecked>关闭</template>
          </n-switch>
        </div>
      </div>

      <div class="admin-card">
        <h3>审计日志</h3>
        <p class="admin-hint">记录管理后台与业务变更操作：登录与登出、用户与用户组、邀请码、系统设置、隧道、域名绑定、DNS 记录与服务监控。只读浏览不计入，可在「审计日志」标签页按用户、操作类型与时间范围筛选。</p>
        <div class="setting-row">
          <span class="setting-label">保留时长</span>
          <select v-model.number="settings.audit_retention_days" class="vercel-input narrow" @change="saveSettings">
            <option :value="30">30 天</option>
            <option :value="90">90 天</option>
            <option :value="180">180 天</option>
            <option :value="365">365 天</option>
            <option :value="-1">永久保留</option>
          </select>
          <small class="text-muted">超期日志每小时自动清理一次</small>
        </div>
      </div>

      <div class="admin-card">
        <h3>通行密钥</h3>
        <p class="admin-hint">通行密钥（WebAuthn）要求 HTTPS，且依赖方 ID 必须与访问域名一致。留空时按访问域名自动推导（沿用面板域名）；反向代理或多域名场景可在此显式指定，来源需与依赖方 ID 同域或为其子域。</p>
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
          <small v-if="!settings.passkey_admin_ready" class="text-muted">需至少一名启用的管理员已绑定通行密钥，避免面板被锁死</small>
        </div>
      </div>

      <div class="admin-card">
        <h3>登录保护</h3>
        <p class="admin-hint">登录、二次验证、注册、验证码、找回与重置密码都受此限制。账号额度是真正的防线——换 IP 绕不过去；IP 额度用来挡住一台机器横扫多个账号。任一触发即返回 429 并附带解锁时间。</p>
        <div class="setting-row">
          <span class="setting-label">启用登录限流</span>
          <n-switch v-model:value="settings.rate_limit_enabled" size="small" @update:value="saveSettings" />
        </div>
        <div class="admin-form">
          <label class="field-label">每账号失败次数<small>0 = 默认 5 次，填负数表示不限</small></label>
          <input v-model.number="settings.rate_limit_per_account" type="number" class="vercel-input" />
          <label class="field-label">每 IP 失败次数<small>0 = 默认 20 次，填负数表示不限</small></label>
          <input v-model.number="settings.rate_limit_per_ip" type="number" class="vercel-input" />
          <label class="field-label">统计窗口（分钟）<small>0 = 默认 15 分钟</small></label>
          <input v-model.number="settings.rate_limit_window_minutes" type="number" class="vercel-input" />
          <label class="field-label">熟悉来源的额度倍数<small>90 天内登录过的网段给更宽的额度。0 = 默认 3 倍，1 = 不放宽</small></label>
          <input v-model.number="settings.rate_limit_familiar_multiplier" type="number" class="vercel-input" />
          <button class="btn btn-primary" type="button" :disabled="busy" @click="saveSettings">保存</button>
        </div>
        <div class="setting-row">
          <span class="setting-label">锁定时通知管理员</span>
          <n-switch v-model:value="settings.rate_limit_notify" size="small" @update:value="saveSettings" />
          <small class="text-muted">走各管理员自己配置的通知渠道（TG / 邮件），并沿用「登录通知」开关</small>
        </div>
      </div>

      <div class="admin-card">
        <h3>人机验证（Cloudflare Turnstile）</h3>
        <p class="admin-hint">可选防护：开启后登录、注册与找回密码需要完成 Cloudflare 人机验证。先在 Cloudflare 控制台创建 Turnstile widget（并把本站域名加入允许域名），再填写 Site Key 与 Secret Key。</p>
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
        <h3>应用加密密钥</h3>
        <p class="admin-hint">用于加密 SMTP 密码、Cloudflare 授权令牌与 2FA 密文。未设置时自动生成并保存在数据库；环境变量 APP_ENCRYPTION_KEY 优先。更换后需重启服务，且会使已保存的密文失效。</p>
        <div class="admin-form">
          <input v-model="encKeyInput" type="text" placeholder="64 位十六进制密钥（留空保持不变）" class="vercel-input" />
          <button class="btn btn-primary" type="button" :disabled="busy || !encKeyInput" @click="saveEncKey">保存密钥</button>
          <span class="tag">当前来源：{{ encKeySource === 'env' ? '环境变量' : encKeySource === 'stored' ? '数据库' : '未设置' }}</span>
        </div>
      </div>
    </section>

    <section v-show="activeTab === 'smtp'" class="admin-section">
      <div class="admin-card">
        <h3>SMTP 邮件服务</h3>
        <p class="admin-hint">用于注册验证码与监控告警邮件。密码留空表示保持原值不变。</p>
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
    </section>

    <section v-show="activeTab === 'audit'" class="admin-section">
      <div class="admin-card">
        <div class="admin-card-head">
          <h3>审计日志</h3>
          <button class="btn btn-secondary" type="button" :disabled="auditLoading" @click="loadAudit">刷新</button>
        </div>
        <div class="audit-stats">
          <div class="audit-stat">
            <span class="audit-stat-value">{{ auditStats.total }}</span>
            <span class="audit-stat-label">近 7 天操作</span>
          </div>
          <div class="audit-stat">
            <span class="audit-stat-value" :class="{ danger: auditStats.failed > 0 }">{{ auditStats.failed }}</span>
            <span class="audit-stat-label">近 7 天失败</span>
          </div>
          <div class="audit-stat">
            <span class="audit-stat-value">{{ auditStats.today }}</span>
            <span class="audit-stat-label">今日操作</span>
          </div>
          <div class="audit-stat">
            <span class="audit-stat-value">{{ auditStats.actors }}</span>
            <span class="audit-stat-label">近 7 天活跃账号</span>
          </div>
        </div>
        <form class="admin-form audit-filters" @submit.prevent="applyAuditFilters">
          <input v-model="auditFilters.actor" type="text" placeholder="搜索用户（用户名）" class="vercel-input" />
          <select v-model="auditFilters.action" class="vercel-input">
            <option value="">全部操作</option>
            <optgroup v-for="group in auditActionGroups" :key="group.category" :label="group.label">
              <option v-for="item in group.items" :key="item.action" :value="item.action">{{ item.label }}</option>
            </optgroup>
          </select>
          <input v-model="auditFilters.fromDate" type="date" class="vercel-input narrow" aria-label="开始日期" />
          <span class="text-muted">至</span>
          <input v-model="auditFilters.toDate" type="date" class="vercel-input narrow" aria-label="结束日期" />
          <button class="btn btn-primary" type="submit" :disabled="auditLoading">查询</button>
          <button class="btn btn-secondary" type="button" :disabled="auditLoading" @click="resetAuditFilters">重置</button>
        </form>
        <div class="admin-table-wrap">
          <table class="admin-table">
            <thead>
              <tr><th>时间</th><th>操作人</th><th>操作</th><th>对象</th><th>来源 IP</th><th>结果</th></tr>
            </thead>
            <tbody>
              <tr v-for="log in auditLogs" :key="log.id">
                <td>{{ formatTime(log.created_at) }}</td>
                <td class="mono">{{ log.actor_name || '—' }}</td>
                <td>{{ auditActionLabel(log) }}</td>
                <td>{{ log.target || '—' }}</td>
                <td class="mono">{{ log.ip || '—' }}</td>
                <td><span class="tag" :class="log.success ? 'tag-ok' : 'tag-down'">{{ log.success ? '成功' : '失败' }}</span></td>
              </tr>
              <tr v-if="!auditLoading && !auditLogs.length">
                <td colspan="6" class="text-muted">没有符合条件的日志</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="audit-pager">
          <span class="text-muted">共 {{ auditTotal }} 条</span>
          <button class="btn btn-ghost" type="button" :disabled="auditLoading || auditPage <= 1" @click="changeAuditPage(auditPage - 1)">上一页</button>
          <span class="text-muted">第 {{ auditPage }} / {{ auditPageCount }} 页</span>
          <button class="btn btn-ghost" type="button" :disabled="auditLoading || auditPage >= auditPageCount" @click="changeAuditPage(auditPage + 1)">下一页</button>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useMessage, NSwitch } from 'naive-ui'
import { useConfigStore } from '../stores/config'
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

const tabs = [
  { key: 'users', label: '用户' },
  { key: 'groups', label: '用户组' },
  { key: 'invites', label: '邀请码' },
  { key: 'settings', label: '系统设置' },
  { key: 'smtp', label: '邮件服务' },
  { key: 'audit', label: '审计日志' },
]
const activeTab = ref('users')
const busy = ref(false)
const message = ref('')
const messageIsError = ref(false)
const toast = useMessage()
const configStore = useConfigStore()

const users = ref<UserView[]>([])
const groups = ref<UserGroup[]>([])
const invites = ref<Invite[]>([])
const settings = ref<AppSettings>({ registration_enabled: true, invite_mode: 'off', default_group_id: '', experimental_features_enabled: false })
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
const experimentalFeatures = computed({
  get: () => !!settings.value.experimental_features_enabled,
  set: (v: boolean) => { settings.value.experimental_features_enabled = v },
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
  return new Date(seconds * 1000).toLocaleString()
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
    notify('加载审计日志失败: ' + (e.response?.data?.error || e.message), true)
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
    settings.value = { ...s.data, audit_retention_days: s.data.audit_retention_days || 90 }
    turnstile.value.enabled = !!s.data.turnstile_enabled
    turnstile.value.site_key = s.data.turnstile_site_key || ''
    turnstileHasSecret.value = !!s.data.turnstile_has_secret
    turnstileSecret.value = ''
  } catch (e: any) {
    notify('加载失败: ' + (e.response?.data?.error || e.message), true)
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
    notify(e.response?.data?.error || '操作失败', true)
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
  }, '用户已创建')
}

function toggleUserStatus(user: UserView) {
  const target = user.status === 'active' ? 'disabled' : 'active'
  void run(() => setUserStatus(user.id, target), target === 'active' ? '已启用' : '已禁用')
}

function changeGroup(user: UserView) {
  const options = groups.value.map((g) => g.name + ':' + g.id).join('  ')
  const picked = window.prompt('输入目标用户组的编号：\n' + options + '\n留空 = 默认用户组', user.group_id)
  if (picked === null) return
  void run(() => setUserGroup(user.id, picked.trim()), '用户组已更新')
}

function restorePasswordLogin(user: UserView) {
  void run(() => setUserPasswordLogin(user.id, false), '已恢复 ' + user.username + ' 的密码登录')
}

function resetPassword(user: UserView) {
  const password = window.prompt('为 ' + user.username + ' 设置新密码（≥6 位）')
  if (!password) return
  void run(() => resetUserPassword(user.id, password), '密码已重置，该用户已被登出')
}

function removeUser(user: UserView) {
  if (!window.confirm('确定删除用户 ' + user.username + '？该操作不可恢复。')) return
  void run(() => deleteUser(user.id), '用户已删除')
}

function submitCreateGroup() {
  const name = newGroupName.value.trim()
  if (!name) return
  void run(async () => {
    await createGroup(name, [])
    newGroupName.value = ''
    showCreateGroup.value = false
  }, '用户组已创建')
}

function toggleGroupPerm(group: UserGroup, perm: string, on: boolean) {
  const next = on ? group.permissions.concat(perm) : group.permissions.filter((p) => p !== perm)
  void run(() => updateGroup(group.id, group.name, next), '权限已更新')
}

function renameGroup(group: UserGroup) {
  if (group.builtin) {
    notify('内置用户组不可重命名', true)
    return
  }
  const name = window.prompt('新的用户组名称', group.name)
  if (!name || name === group.name) return
  void run(() => updateGroup(group.id, name, group.permissions), '名称已更新')
}

function removeGroup(group: UserGroup) {
  if (!window.confirm('确定删除用户组 ' + group.name + '？')) return
  void run(() => deleteGroup(group.id), '用户组已删除')
}

function submitCreateInvite() {
  void run(async () => {
    await createInvite({
      group_id: newInvite.value.groupId || undefined,
      max_uses: Number(newInvite.value.maxUses) > 0 ? Number(newInvite.value.maxUses) : 0,
      expires_at: Number(newInvite.value.expireDays) > 0 ? Math.floor(Date.now() / 1000) + Number(newInvite.value.expireDays) * 86400 : 0,
    })
    showCreateInvite.value = false
  }, '邀请码已生成')
}

function toggleInvite(invite: Invite) {
  void run(() => updateInvite(invite.code, !invite.enabled), invite.enabled ? '已停用' : '已启用')
}

function removeInvite(invite: Invite) {
  if (!window.confirm('确定删除邀请码 ' + invite.code + '？')) return
  void run(() => deleteInvite(invite.code), '邀请码已删除')
}

function saveSettings() {
  void run(async () => {
    await updateAppSettings(settings.value)
    configStore.setExperimentalFeatures(!!settings.value.experimental_features_enabled)
  }, '设置已保存')
}

function saveTurnstile() {
  // 带上完整设置：只保存人机验证时不能把通行密钥、审计保留期等字段清空。
  const payload: Partial<AppSettings> & { turnstile_secret?: string } = {
    ...settings.value,
    registration_enabled: settings.value.registration_enabled,
    invite_mode: settings.value.invite_mode,
    default_group_id: settings.value.default_group_id || '',
    email_verify_disabled: !!settings.value.email_verify_disabled,
    experimental_features_enabled: !!settings.value.experimental_features_enabled,
    turnstile_enabled: turnstile.value.enabled,
    turnstile_site_key: turnstile.value.site_key.trim(),
  }
  if (turnstileSecret.value) payload.turnstile_secret = turnstileSecret.value
  void run(() => updateAppSettings(payload), '人机验证设置已保存')
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
    toast.success(data.message || '已保存')
    encKeyInput.value = ''
    const status = await getEncryptionKeyStatus()
    encKeySource.value = status.data.source
  } catch (e: any) {
    toast.error(e.response?.data?.error || '保存失败')
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
    toast.success('OAuth 客户端配置已保存')
  } catch (e: any) {
    toast.error(e.response?.data?.error || '保存失败')
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
    toast.success(data.configured ? 'SMTP 设置已保存' : 'SMTP 设置已保存（尚未完整配置）')
    smtp.value.password = ''
  } catch (e: any) {
    toast.error(e.response?.data?.error || '保存失败')
  } finally {
    busy.value = false
  }
}

async function sendTestMail() {
  if (!testMailTo.value.trim()) {
    toast.error('请先填写收件邮箱')
    return
  }
  testingMail.value = true
  try {
    const { data } = await testSMTP(testMailTo.value.trim())
    toast.success(data.message || '测试邮件已发送')
  } catch (e: any) {
    toast.error(e.response?.data?.error || '发送失败')
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

