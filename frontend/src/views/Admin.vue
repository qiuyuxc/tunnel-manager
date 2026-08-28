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
              <tr><th>用户名</th><th>邮箱</th><th>角色</th><th>用户组</th><th>状态</th><th>最近登录</th><th>操作</th></tr>
            </thead>
            <tbody>
              <tr v-for="user in users" :key="user.id">
                <td class="mono">{{ user.username }}</td>
                <td>{{ user.email || '—' }}</td>
                <td><span class="tag" :class="user.role === 'admin' ? 'tag-admin' : ''">{{ user.role === 'admin' ? '管理员' : '用户' }}</span></td>
                <td>{{ groupName(user.group_id) }}</td>
                <td><span class="tag" :class="user.status === 'active' ? 'tag-ok' : 'tag-down'">{{ user.status === 'active' ? '正常' : '已禁用' }}</span></td>
                <td>{{ formatTime(user.last_login_at) }}</td>
                <td class="actions">
                  <button class="btn btn-ghost" type="button" @click="toggleUserStatus(user)">{{ user.status === 'active' ? '禁用' : '启用' }}</button>
                  <button class="btn btn-ghost" type="button" @click="changeGroup(user)">改组</button>
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
        <label class="admin-check setting-row">
          <input v-model="settings.registration_enabled" type="checkbox" @change="saveSettings" />
          开放邮箱注册
        </label>
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
    </section>

    <section v-show="activeTab === 'smtp'" class="admin-section">
      <div class="admin-card">
        <h3>SMTP 邮件服务</h3>
        <p class="admin-hint">用于注册验证码与监控告警邮件。密码留空表示保持原值不变。</p>
        <div class="admin-form smtp-form">
          <input v-model="smtp.host" type="text" placeholder="SMTP 主机，如 smtp.example.com" class="vercel-input" />
          <input v-model.number="smtp.port" type="number" placeholder="端口（587）" class="vercel-input narrow" />
          <input v-model="smtp.username" type="text" placeholder="用户名（通常为邮箱）" class="vercel-input" />
          <input v-model="smtp.password" type="password" placeholder="密码 / 授权码（留空保持不变）" class="vercel-input" />
          <input v-model="smtp.from" type="text" placeholder="发件人，如 panel@example.com" class="vercel-input" />
          <select v-model="smtp.tlsMode" class="vercel-input narrow">
            <option value="starttls">STARTTLS（587 常用）</option>
            <option value="none">不加密（25，不推荐）</option>
          </select>
          <button class="btn btn-primary" type="button" :disabled="busy" @click="saveSmtp">保存设置</button>
        </div>
        <div class="admin-form">
          <input v-model="testMailTo" type="email" placeholder="发送测试邮件到：你的邮箱" class="vercel-input" />
          <button class="btn btn-secondary" type="button" :disabled="busy || testingMail || !smtp.host" @click="sendTestMail">{{ testingMail ? '发送中…' : '发送测试邮件' }}</button>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useMessage } from 'naive-ui'
import {
  listUsers, createUser, setUserStatus, setUserGroup, resetUserPassword, deleteUser,
  listGroups, createGroup, updateGroup, deleteGroup,
  listInvites, createInvite, updateInvite, deleteInvite,
  getAppSettings, updateAppSettings,
  getSMTP, updateSMTP, testSMTP,
  ALL_PERMISSIONS, PERMISSION_LABELS,
  type UserView, type UserGroup, type Invite, type AppSettings,
} from '../api/admin'

const tabs = [
  { key: 'users', label: '用户' },
  { key: 'groups', label: '用户组' },
  { key: 'invites', label: '邀请码' },
  { key: 'settings', label: '系统设置' },
  { key: 'smtp', label: '邮件服务' },
]
const activeTab = ref('users')
const busy = ref(false)
const message = ref('')
const messageIsError = ref(false)
const toast = useMessage()

const users = ref<UserView[]>([])
const groups = ref<UserGroup[]>([])
const invites = ref<Invite[]>([])
const settings = ref<AppSettings>({ registration_enabled: true, invite_mode: 'off', default_group_id: '' })
const allPermissions = ALL_PERMISSIONS
const permissionLabels = PERMISSION_LABELS

const showCreateUser = ref(false)
const showCreateGroup = ref(false)
const showCreateInvite = ref(false)
const newUser = ref({ username: '', email: '', password: '', groupId: '', admin: false })
const newGroupName = ref('')
const newInvite = ref({ groupId: '', maxUses: 0, expireDays: 0 })
const smtp = ref({ host: '', port: 587, username: '', password: '', from: '', tlsMode: 'starttls' })
const testMailTo = ref('')
const testingMail = ref(false)

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

async function loadAll() {
  busy.value = true
  try {
    const [u, g, i, s] = await Promise.all([listUsers(), listGroups(), listInvites(), getAppSettings()])
    users.value = u.data.users
    groups.value = g.data.groups
    invites.value = i.data.invites
    settings.value = s.data
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
      max_uses: newInvite.value.maxUses > 0 ? newInvite.value.maxUses : 0,
      expires_at: newInvite.value.expireDays > 0 ? Math.floor(Date.now() / 1000) + newInvite.value.expireDays * 86400 : 0,
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
  void run(() => updateAppSettings(settings.value), '设置已保存')
}

onMounted(() => {
  void loadAll()
  getSMTP().then(({ data }) => {
    smtp.value.host = data.host
    smtp.value.port = data.port || 587
    smtp.value.username = data.username
    smtp.value.from = data.from
    smtp.value.tlsMode = data.tls_mode || 'starttls'
  }).catch(() => {})
})

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
.admin-hint { font-size: 12px; color: var(--color-mute); margin: 0 0 var(--spacing-sm); }
.smtp-form { border-bottom: none; margin-bottom: 0; }
</style>

