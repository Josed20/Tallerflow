import { describe, expect, it, vi } from 'vitest'
import { api } from './api'

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
})
