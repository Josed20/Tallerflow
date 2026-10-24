import { api } from '../session/api'
import type { AuthSessionPayload } from '../session/types'

export interface SignupInput {
  workshop_name: string
  owner_name: string
  email: string
  password: string
  accept_terms: boolean
}

export const signupApi = {
  register: (input: SignupInput) => api.post<{ data: Record<string, never> }>('/api/v1/signup', input),
  verify: (token: string, password: string) => api.post<AuthSessionPayload>('/api/v1/auth/email-verifications/consume', { token, password }),
  resend: (email: string) => api.post<{ data: Record<string, never> }>('/api/v1/auth/email-verifications/resend', { email }),
}
