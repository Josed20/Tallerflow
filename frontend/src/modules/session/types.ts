export type SessionStatus = 'idle' | 'restoring' | 'password-change-required' | 'authenticated' | 'anonymous'

export interface SessionPrincipal {
  id: string
  email: string
  displayName?: string
  role: string
  workshop: {
    id: string
    name: string
  }
  mustChangePassword: boolean
}

export interface AuthSessionPayload {
  data: {
    expires_at: string
    csrf_token: string
    must_change_password: boolean
  }
}

export interface ApiPrincipal {
  data: {
    userId: string
    email: string
    displayName: string
    workshopId: string
    role: string
    passwordChangeRequired: boolean
  }
}

export interface WorkshopAccess {
  data: {
    workshop: {
      id: string
      name: string
      timezone: string
    }
    role: string
  }
}

export interface APIErrorEnvelope {
  error: {
    code: string
    message: string
    details: Record<string, unknown>
    request_id: string
  }
}
