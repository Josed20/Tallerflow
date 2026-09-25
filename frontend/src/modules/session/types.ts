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
