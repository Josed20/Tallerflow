import { api } from '../session/api'
import type { AuthSessionPayload } from '../session/types'

export interface OnboardingInput {
  workshop_name: string
  owner_name: string
  email: string
  password: string
  password_confirmation: string
}

interface OnboardingStatusPayload {
  data: {
    available: boolean
  }
}

export const onboardingApi = {
  status: () => api.get<OnboardingStatusPayload>('/api/v1/onboarding/status'),
  create: (input: OnboardingInput) => api.post<AuthSessionPayload>('/api/v1/onboarding/workshop', input),
}
