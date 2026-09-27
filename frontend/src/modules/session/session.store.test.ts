import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useSessionStore } from './session.store'

describe('session store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.unstubAllGlobals()
    localStorage.clear()
    sessionStorage.clear()
  })

  it('restores the authenticated principal from the auth and workshops contracts', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({
        data: { expires_at: '2026-09-26T01:00:00Z', csrf_token: 'csrf-1', must_change_password: false },
      }))
      .mockResolvedValueOnce(jsonResponse({
        data: {
          userId: 'owner-1', email: 'owner@taller.pe', displayName: 'Lucero', workshopId: 'workshop-1',
          role: 'OWNER', passwordChangeRequired: false,
        },
      }))
      .mockResolvedValueOnce(jsonResponse({
        data: {
          workshop: { id: 'workshop-1', name: 'Taller San Martín', timezone: 'America/Lima' },
          role: 'OWNER',
        },
      }))
    vi.stubGlobal('fetch', fetchMock)
    const session = useSessionStore()

    await session.restore()

    expect(session.principal?.workshop.name).toBe('Taller San Martín')
    expect(session.principal?.id).toBe('owner-1')
    expect(session.principal?.displayName).toBe('Lucero')
    expect(session.status).toBe('authenticated')
    expect(session.restored).toBe(true)
    expect(session.requiresPasswordChange).toBe(false)
    expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
      '/api/v1/auth/session',
      '/api/v1/me',
      '/api/v1/workshops/current',
    ])
    expect(localStorage.length).toBe(0)
    expect(sessionStorage.length).toBe(0)
  })

  it('keeps a mandatory password-change session authenticated without loading protected profile data', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({
      data: { expires_at: '2026-09-26T01:00:00Z', csrf_token: 'csrf-1', must_change_password: true },
    }))
    vi.stubGlobal('fetch', fetchMock)
    const session = useSessionStore()

    await session.restore()

    expect(session.isAuthenticated).toBe(true)
    expect(session.requiresPasswordChange).toBe(true)
    expect(session.status).toBe('password-change-required')
    expect(session.principal).toBeNull()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/auth/session')
  })

  it('sends the current CSRF token and backend field names when changing a password', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({
        data: { expires_at: '2026-09-26T01:00:00Z', csrf_token: 'csrf-1', must_change_password: true },
      }))
      .mockResolvedValueOnce(jsonResponse({
        data: { expires_at: '2026-09-26T02:00:00Z', csrf_token: 'csrf-2', must_change_password: false },
      }))
      .mockResolvedValueOnce(jsonResponse({
        data: {
          userId: 'owner-1', email: 'owner@taller.pe', displayName: 'Lucero', workshopId: 'workshop-1',
          role: 'OWNER', passwordChangeRequired: false,
        },
      }))
      .mockResolvedValueOnce(jsonResponse({
        data: {
          workshop: { id: 'workshop-1', name: 'Taller San Martín', timezone: 'America/Lima' },
          role: 'OWNER',
        },
      }))
      .mockResolvedValueOnce(jsonResponse({
        data: {},
      }))
    vi.stubGlobal('fetch', fetchMock)
    const session = useSessionStore()
    await session.restore()

    await session.changePassword('Current password 123!', 'New password 456!')

    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/v1/auth/change-password', {
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
        'X-CSRF-Token': 'csrf-1',
      },
      method: 'POST',
      body: JSON.stringify({ current_password: 'Current password 123!', new_password: 'New password 456!' }),
    })
    expect(session.requiresPasswordChange).toBe(false)
    expect(session.principal?.workshop.name).toBe('Taller San Martín')
    await session.logout()
    expect(fetchMock).toHaveBeenNthCalledWith(5, '/api/v1/auth/logout', {
      credentials: 'include',
      headers: { Accept: 'application/json', 'X-CSRF-Token': 'csrf-2' },
      method: 'POST',
      body: undefined,
    })
  })

  it('sends the current CSRF token when logging out', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({
        data: { expires_at: '2026-09-26T01:00:00Z', csrf_token: 'csrf-1', must_change_password: false },
      }))
      .mockResolvedValueOnce(jsonResponse({
        data: {
          userId: 'owner-1', email: 'owner@taller.pe', displayName: 'Lucero', workshopId: 'workshop-1',
          role: 'OWNER', passwordChangeRequired: false,
        },
      }))
      .mockResolvedValueOnce(jsonResponse({
        data: {
          workshop: { id: 'workshop-1', name: 'Taller San Martín', timezone: 'America/Lima' },
          role: 'OWNER',
        },
      }))
      .mockResolvedValueOnce(jsonResponse({ data: {} }))
    vi.stubGlobal('fetch', fetchMock)
    const session = useSessionStore()
    await session.restore()

    await session.logout()

    expect(fetchMock).toHaveBeenNthCalledWith(4, '/api/v1/auth/logout', {
      credentials: 'include',
      headers: { Accept: 'application/json', 'X-CSRF-Token': 'csrf-1' },
      method: 'POST',
      body: undefined,
    })
    expect(session.isAuthenticated).toBe(false)
  })

  it('clears all in-memory state after a 401 restoration response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'SESSION_INVALID', message: 'expired', details: {}, request_id: 'request-1' },
    }), { status: 401, headers: { 'Content-Type': 'application/json' } })))
    const session = useSessionStore()

    await session.restore()

    expect(session.status).toBe('anonymous')
    expect(session.isAuthenticated).toBe(false)
    expect(session.principal).toBeNull()
    expect(localStorage.length).toBe(0)
    expect(sessionStorage.length).toBe(0)
  })

  it('accepts an onboarding session in memory and loads its principal', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({
        data: {
          userId: 'owner-web', email: 'owner@web.test', displayName: 'Owner Web', workshopId: 'workshop-web',
          role: 'OWNER', passwordChangeRequired: false,
        },
      }))
      .mockResolvedValueOnce(jsonResponse({
        data: { workshop: { id: 'workshop-web', name: 'Taller Web', timezone: 'America/Lima' }, role: 'OWNER' },
      }))
    vi.stubGlobal('fetch', fetchMock)
    const session = useSessionStore()

    await session.acceptSession({ data: { expires_at: '2026-09-28T01:00:00Z', csrf_token: 'csrf-web', must_change_password: false } })

    expect(session.isAuthenticated).toBe(true)
    expect(session.principal?.workshop.name).toBe('Taller Web')
    expect(session.restored).toBe(true)
    expect(localStorage.length).toBe(0)
    expect(sessionStorage.length).toBe(0)
  })
})

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })
}
