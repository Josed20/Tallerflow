import { api } from '../session/api'

export interface Client {
  id: string
  workshop_id: string
  name: string
  email: string | null
  phone: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreateClientPayload {
  name: string
  email?: string | null
  phone?: string | null
}

export interface UpdateClientPayload {
  name?: string
  email?: string | null
  phone?: string | null
  is_active?: boolean
}

export const clientsApi = {
  list: () => api.get<{ data: Client[] }>('/api/v1/clients'),
  create: (body: CreateClientPayload, csrfToken?: string) =>
    api.post<{ data: Client }>('/api/v1/clients', body, csrfToken),
  update: (id: string, body: UpdateClientPayload, csrfToken?: string) =>
    api.patch<{ data: Client }>(`/api/v1/clients/${id}`, body, csrfToken),
}
