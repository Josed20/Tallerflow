import type { ClientPortalOrder } from './types'

export class ClientPortalApiError extends Error {
  constructor(public readonly status: number) {
    super('No pudimos completar la solicitud del portal.')
  }
}

async function request<T>(path: string, init: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    credentials: 'include',
    headers: { Accept: 'application/json', ...(init.headers ?? {}) },
  })
  if (!response.ok) throw new ClientPortalApiError(response.status)
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export const clientPortalApi = {
  requestAccess: (email: string) => request<void>('/api/v1/client-portal/access-links', {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ email }),
  }),
  consumeAccessLink: (token: string) => request<void>('/api/v1/client-portal/access-links/consume', {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ token }),
  }),
  listOrders: () => request<{ data: ClientPortalOrder[] }>('/api/v1/client-portal/orders', { method: 'GET' }),
  getOrder: (id: string) => request<{ data: ClientPortalOrder }>(`/api/v1/client-portal/orders/${id}`, { method: 'GET' }),
  logout: () => request<void>('/api/v1/client-portal/logout', { method: 'POST' }),
}
