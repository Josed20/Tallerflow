import { describe, expect, it, vi } from 'vitest'
import { ApiError, api } from './api'

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
})
