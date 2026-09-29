import { api } from '../session/api'

export type TeamRole = 'OWNER' | 'ADMIN' | 'OPERATOR'
export type TeamStatus = 'ACTIVE' | 'INACTIVE'

export interface TeamMember {
  id: string
  user_id: string
  email: string
  display_name: string
  role: TeamRole
  status: TeamStatus
}

export interface TeamInvitation {
  id: string
  email: string
  role: Exclude<TeamRole, 'OWNER'>
  expires_at: string
}

export interface TeamPayload {
  data: {
    members: TeamMember[]
    invitations: TeamInvitation[]
  }
}

export interface InvitationCreatedPayload {
  data: {
    invitation: TeamInvitation
    join_url: string
    token: string
  }
}

export interface InvitationConsumePayload {
  data: {
    email: string
    role: string
  }
}

export const teamApi = {
  list: () => api.get<TeamPayload>('/api/v1/team'),
  invite: (email: string, role: string, csrfToken: string) =>
    api.post<InvitationCreatedPayload>('/api/v1/team/invitations', { email, role }, csrfToken),
  updateMember: (membershipId: string, body: { role?: string; status?: string }, csrfToken: string) =>
    api.patch<{ data: TeamMember }>(`/api/v1/team/${membershipId}`, body, csrfToken),
  consume: (token: string, name: string, password: string) =>
    api.post<InvitationConsumePayload>('/api/v1/team/invitations/consume', { token, name, password }),
}
