import type { APIErrorEnvelope } from './types'

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code = 'REQUEST_FAILED',
    public readonly requestId = '',
  ) {
    super('No pudimos completar la solicitud.')
  }
}

async function request<T>(path: string, init: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    credentials: 'include',
    headers: { Accept: 'application/json', ...(init.headers ?? {}) },
  })

  if (!response.ok) {
    const payload = await response.json().catch(() => null) as APIErrorEnvelope | null
    throw new ApiError(response.status, payload?.error?.code, payload?.error?.request_id)
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: 'GET' }),
  post: <T>(path: string, body?: unknown, csrfToken?: string) =>
    request<T>(path, {
      method: 'POST',
      headers: {
        ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
        ...(csrfToken === undefined ? {} : { 'X-CSRF-Token': csrfToken }),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
    }),
  patch: <T>(path: string, body?: unknown, csrfToken?: string) =>
    request<T>(path, {
      method: 'PATCH',
      headers: {
        ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
        ...(csrfToken === undefined ? {} : { 'X-CSRF-Token': csrfToken }),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
    }),
}
