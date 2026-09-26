export type SessionStatus = 'idle' | 'restoring' | 'authenticated' | 'anonymous'

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
    csrf_token: string
    must_change_password: boolean
  }
}

export interface ApiPrincipal {
  userId: string
  email: string
  displayName: string
  workshopId: string
  role: string
}

export interface WorkshopAccess {
  workshop: {
    id: string
    name: string
  }
  role: string
}
