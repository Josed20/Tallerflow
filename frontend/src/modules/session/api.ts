import type { APIErrorEnvelope } from './types'

type UnauthorizedHandler = (reason: 'expired' | 'suspended') => void | Promise<void>

type ApiTransport =
  | { kind: 'web'; onUnauthorized?: UnauthorizedHandler }
  | {
      kind: 'mobile'
      getToken: () => string | null | Promise<string | null>
      onUnauthorized?: UnauthorizedHandler
    }

let transport: ApiTransport = { kind: 'web' }

export function configureApiTransport(next: ApiTransport): void {
  transport = next
}

function apiUrl(path: string, kind: ApiTransport['kind']): string {
  const baseUrl = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/+$/, '')
  if (kind === 'mobile' && !baseUrl) {
    throw new Error('VITE_API_BASE_URL is required for mobile requests')
  }
  return `${baseUrl}${path}`
}

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
  const activeTransport = transport
  const headers: Record<string, string> = {
    Accept: 'application/json',
    ...(init.headers as Record<string, string> | undefined),
  }
  if (activeTransport.kind === 'mobile') {
    delete headers['X-CSRF-Token']
    const token = path === '/api/v1/auth/login' ? null : await activeTransport.getToken()
    if (token) headers.Authorization = `Bearer ${token}`
  }

  const response = await fetch(apiUrl(path, activeTransport.kind), {
    ...init,
    credentials: activeTransport.kind === 'mobile' ? 'omit' : 'include',
    headers,
  })

  if (!response.ok) {
    const payload = await response.json().catch(() => null) as APIErrorEnvelope | null
    if (path !== '/api/v1/auth/login') {
      const suspended = response.status === 403 && payload?.error?.code === 'WORKSHOP_SUSPENDED'
      // A suspended workshop signs out both clients. An expired web session is
      // left to the router guard, which already sends it to the login.
      if (suspended || (activeTransport.kind === 'mobile' && response.status === 401)) {
        await activeTransport.onUnauthorized?.(suspended ? 'suspended' : 'expired')
      }
    }
    throw new ApiError(response.status, payload?.error?.code, payload?.error?.request_id)
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: 'GET' }),
  post: <T>(path: string, body?: unknown, csrfToken?: string, headers?: Record<string, string>) =>
    request<T>(path, {
      method: 'POST',
      headers: {
        ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
        ...(csrfToken === undefined ? {} : { 'X-CSRF-Token': csrfToken }),
        ...(headers ?? {}),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
    }),
  patch: <T>(path: string, body?: unknown, csrfToken?: string, headers?: Record<string, string>) =>
    request<T>(path, {
      method: 'PATCH',
      headers: {
        ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
        ...(csrfToken === undefined ? {} : { 'X-CSRF-Token': csrfToken }),
        ...(headers ?? {}),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
    }),
  delete: <T>(path: string, csrfToken?: string) =>
    request<T>(path, {
      method: 'DELETE',
      headers: csrfToken === undefined ? {} : { 'X-CSRF-Token': csrfToken },
    }),
}
