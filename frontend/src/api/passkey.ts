// 通行密钥（WebAuthn）：注册、免密登录、第二因子与密码登录开关。
import { api } from './index'
import type { LoginResponse } from './index'
import type {
  PublicKeyCredentialCreationOptionsJSON,
  PublicKeyCredentialRequestOptionsJSON,
  RegistrationResponseJSON,
  AuthenticationResponseJSON,
} from '@simplewebauthn/browser'

export interface PasskeyView {
  id: string
  name: string
  created_at: number
  last_used_at: number
  /** 可同步（iCloud 钥匙串 / Google 密码管理器）的通行密钥 */
  backup_eligible: boolean
  /** 已经同步到其他设备 */
  backup_state: boolean
}

export interface PasskeySettings {
  passkeys: PasskeyView[]
  password_login_disabled: boolean
  password_login_disabled_globally: boolean
  rp_id: string
  origin: string
  /** 依赖方是否可解析；false 时无法执行 WebAuthn 流程 */
  available: boolean
}

/** 注册流程：begin 返回的 public_key 交给浏览器，credential 原样回传 finish。 */
export interface RegistrationCeremony {
  ceremony_token: string
  public_key: PublicKeyCredentialCreationOptionsJSON
}

export interface LoginCeremony {
  ceremony_token: string
  public_key: PublicKeyCredentialRequestOptionsJSON
}

export const getPasskeySettings = () => api.get<PasskeySettings>('/account/passkeys')

export const beginPasskeyRegistration = (password: string) =>
  api.post<RegistrationCeremony>('/account/passkeys/begin', { password })

export const finishPasskeyRegistration = (payload: {
  ceremony_token: string
  name: string
  credential: RegistrationResponseJSON
}) => api.post<PasskeyView>('/account/passkeys/finish', payload)

export const renamePasskey = (id: string, name: string) => api.put('/account/passkeys/' + id, { name })

export const deletePasskey = (id: string, password: string) =>
  api.delete('/account/passkeys/' + id, { data: { password } })

export const setPasswordLoginDisabled = (disabled: boolean, password: string) =>
  api.put<PasskeySettings>('/account/password-login', { disabled, password })

// 登录流程（无需会话）
export const beginPasskeyLogin = (account = '') =>
  api.post<LoginCeremony>('/auth/passkey/login/begin', { account })

export const finishPasskeyLogin = (payload: { ceremony_token: string; credential: AuthenticationResponseJSON }) =>
  api.post<LoginResponse>('/auth/passkey/login/finish', payload)

export const beginPasskeyTwoFactor = (challengeToken: string) =>
  api.post<LoginCeremony>('/admin/login/2fa/passkey/begin', { challenge_token: challengeToken })

export const finishPasskeyTwoFactor = (payload: {
  challenge_token: string
  ceremony_token: string
  credential: AuthenticationResponseJSON
}) => api.post<LoginResponse>('/admin/login/2fa/passkey/finish', payload)
