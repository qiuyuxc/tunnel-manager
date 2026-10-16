import { api, type LoginResult } from './index'

export interface OIDCPublicProvider { id: string; name: string }
export interface OIDCProvider extends OIDCPublicProvider {
  issuer: string
  client_id: string
  redirect_uri: string
  scopes: string
  enabled: boolean
  order: number
  has_client_secret: boolean
}
export interface OIDCProviderInput {
  name: string
  issuer: string
  client_id: string
  client_secret?: string
  redirect_uri: string
  scopes: string
  enabled: boolean
  order: number
  clear_secret?: boolean
}
export interface OIDCEmailChallenge { email_verification_required: true; email: string; ticket: string }
export interface OIDCPasswordChallenge { password_setup_required: true; ticket: string }
export type OIDCResult = LoginResult | OIDCEmailChallenge | OIDCPasswordChallenge | { status: string }
export interface OIDCBinding { provider_id: string; name: string; enabled: boolean; email: string; created_at: number }
export interface OIDCAccount { has_password: boolean; bindings: OIDCBinding[] }

export const listOIDCProviders = () => api.get<OIDCPublicProvider[]>('/auth/oidc/providers')
export const listAdminOIDCProviders = () => api.get<OIDCProvider[]>('/admin/oidc/providers')
export function saveOIDCProvider(id: string, input: OIDCProviderInput) {
  return id ? api.put<OIDCProvider>(`/admin/oidc/providers/${encodeURIComponent(id)}`, input) : api.post<OIDCProvider>('/admin/oidc/providers', input)
}
export const deleteOIDCProvider = (id: string) => api.delete(`/admin/oidc/providers/${encodeURIComponent(id)}`)
export const startOIDCLogin = (id: string, turnstileToken = '') => api.post<{ authorization_url: string }>('/auth/oidc/start', { provider_id: id, cf_turnstile_response: turnstileToken })
export const completeOIDC = (ticket: string, code = '') => api.post<OIDCResult>('/auth/oidc/complete', { ticket, code }, { skipAuthInvalidation: true })
export const sendOIDCCode = (ticket: string) => api.post<{ message: string }>('/auth/oidc/send-code', { ticket })
export const getOIDCAccount = () => api.get<OIDCAccount>('/account/oidc')
export const startOIDCAccount = (id: string, action: 'bind' | 'password', password = '') => api.post<{ authorization_url: string }>('/account/oidc/start', { provider_id: id, action, password })
export const unlinkOIDC = (id: string) => api.delete(`/account/oidc/${encodeURIComponent(id)}`)
export const setupOIDCPassword = (ticket: string, password: string) => api.post('/account/oidc/password', { ticket, password }, { skipAuthInvalidation: true })
