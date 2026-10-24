import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, api, configureApiTransport } from './api'

afterEach(() => {
  configureApiTransport({ kind: 'web' })
  vi.unstubAllEnvs()
  vi.unstubAllGlobals()
})

describe('same-origin API client', () => {
  it('sends session requests with browser cookies and no bearer token', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: 'user-1' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await api.get<{ id: string }>('/api/v1/me')

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/me', {
      credentials: 'include',
      headers: { Accept: 'application/json' },
      method: 'GET',
    })
  })

  it('preserves the backend error code and request id without exposing auth headers', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'AUTH_RATE_LIMITED', message: 'Too many attempts.', details: {}, request_id: 'request-7' },
    }), { status: 429, headers: { 'Content-Type': 'application/json' } })))

	await expect(api.post('/api/v1/auth/login', { email: 'owner@example.com', password: 'secret' }))
		.rejects.toMatchObject({ status: 429, code: 'AUTH_RATE_LIMITED', requestId: 'request-7' } satisfies Partial<ApiError>)
  })

  it('sends a CSRF token when canceling an invitation', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)

    await api.delete('/api/v1/team/invitations/invitation-1', 'csrf-1')

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/team/invitations/invitation-1', {
      credentials: 'include',
      headers: { Accept: 'application/json', 'X-CSRF-Token': 'csrf-1' },
      method: 'DELETE',
    })
  })

  it('prefixes API paths when an external base URL is configured', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://tallerflow.pe/')
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)

    await api.get('/api/v1/me')

    expect(fetchMock.mock.calls[0]?.[0]).toBe('https://tallerflow.pe/api/v1/me')
    expect(fetchMock.mock.calls[0]?.[1]).toMatchObject({ credentials: 'include' })
  })

  it('uses bearer without cookies or CSRF in mobile mode', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://tallerflow.pe')
    configureApiTransport({ kind: 'mobile', getToken: async () => 'mobile-session' })
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)

    await api.post('/api/v1/orders', { product: 'Polo' }, 'web-csrf', { 'Idempotency-Key': 'order-1' })

    expect(fetchMock).toHaveBeenCalledWith('https://tallerflow.pe/api/v1/orders', {
      method: 'POST',
      credentials: 'omit',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
        'Idempotency-Key': 'order-1',
        Authorization: 'Bearer mobile-session',
      },
      body: JSON.stringify({ product: 'Polo' }),
    })
  })

  it('allows mobile login without a token and handles a revoked token', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://tallerflow.pe')
    let token: string | null = 'stale-session'
    const onUnauthorized = vi.fn(() => { token = null })
    configureApiTransport({ kind: 'mobile', getToken: () => token, onUnauthorized })
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { session_token: 'new-session' } }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ error: { code: 'SESSION_EXPIRED', request_id: 'expired-1' } }), { status: 401 }))
    vi.stubGlobal('fetch', fetchMock)

    await api.post('/api/v1/auth/login', { client: 'mobile', email: 'test@example.com' })
    expect(fetchMock.mock.calls[0]?.[1]).toMatchObject({ credentials: 'omit' })
    expect(fetchMock.mock.calls[0]?.[1].headers).not.toHaveProperty('Authorization')

    token = 'expired-session'
    await expect(api.get('/api/v1/me')).rejects.toMatchObject({ status: 401, code: 'SESSION_EXPIRED' })
    expect(onUnauthorized).toHaveBeenCalledOnce()
    expect(token).toBeNull()
  })

  it('revokes a mobile token when the workshop is suspended', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://tallerflow.pe')
    const onUnauthorized = vi.fn()
    configureApiTransport({ kind: 'mobile', getToken: () => 'active-token', onUnauthorized })
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'WORKSHOP_SUSPENDED', request_id: 'suspended-1' },
    }), { status: 403 })))

    await expect(api.get('/api/v1/me')).rejects.toMatchObject({ status: 403, code: 'WORKSHOP_SUSPENDED' })
    expect(onUnauthorized).toHaveBeenCalledWith('suspended')
  })

  it('signs the browser out when the workshop is suspended, but not on a login attempt', async () => {
    const onUnauthorized = vi.fn()
    configureApiTransport({ kind: 'web', onUnauthorized })
    const suspended = () => new Response(JSON.stringify({
      error: { code: 'WORKSHOP_SUSPENDED', request_id: 'suspended-web' },
    }), { status: 403 })
    vi.stubGlobal('fetch', vi.fn().mockImplementation(() => Promise.resolve(suspended())))

    await expect(api.post('/api/v1/auth/login', { email: 'a@b.pe', password: 'x' })).rejects.toMatchObject({ code: 'WORKSHOP_SUSPENDED' })
    expect(onUnauthorized).not.toHaveBeenCalled()

    await expect(api.get('/api/v1/orders')).rejects.toMatchObject({ status: 403, code: 'WORKSHOP_SUSPENDED' })
    expect(onUnauthorized).toHaveBeenCalledWith('suspended')
  })

  it('leaves an expired browser session to the router guard', async () => {
    const onUnauthorized = vi.fn()
    configureApiTransport({ kind: 'web', onUnauthorized })
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'SESSION_INVALID', request_id: 'expired-web' },
    }), { status: 401 })))

    await expect(api.get('/api/v1/orders')).rejects.toMatchObject({ status: 401 })
    expect(onUnauthorized).not.toHaveBeenCalled()
  })

  it('rejects mobile requests without a configured API origin', async () => {
    vi.stubEnv('VITE_API_BASE_URL', '')
    configureApiTransport({ kind: 'mobile', getToken: () => null })
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    await expect(api.get('/api/v1/me')).rejects.toThrow('VITE_API_BASE_URL is required')
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
