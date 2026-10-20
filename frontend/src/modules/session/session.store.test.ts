import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useSessionStore } from './session.store'
import { configureApiTransport } from './api'

const mobile = vi.hoisted(() => ({ enabled: false, localPreview: false, token: null as string | null }))
vi.mock('../../native/platform', () => ({ usesNativeToken: () => mobile.enabled && !mobile.localPreview }))
vi.mock('../../native/session-token', () => ({
  readMobileToken: async () => mobile.token,
  writeMobileToken: async (value: string) => { mobile.token = value },
  clearMobileToken: async () => { mobile.token = null },
}))

describe('session store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.unstubAllGlobals()
    vi.unstubAllEnvs()
    localStorage.clear()
    sessionStorage.clear()
    mobile.enabled = false
    mobile.localPreview = false
    mobile.token = null
    configureApiTransport({ kind: 'web' })
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

  it('logs in and restores a mobile session using a stored bearer token', async () => {
    mobile.enabled = true
    vi.stubEnv('VITE_API_BASE_URL', 'https://tallerflow.test')
    configureApiTransport({ kind: 'mobile', getToken: () => mobile.token })
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ data: {
        expires_at: '2026-11-01T00:00:00Z', session_token: 'secure-token', must_change_password: false,
      } }))
      .mockResolvedValueOnce(jsonResponse({ data: {
        userId: 'operator-1', email: 'operator@test.pe', displayName: 'Operador', workshopId: 'workshop-1',
      } }))
      .mockResolvedValueOnce(jsonResponse({ data: {
        workshop: { id: 'workshop-1', name: 'Taller Lima', timezone: 'America/Lima' }, role: 'OPERATOR',
      } }))
      .mockResolvedValueOnce(jsonResponse({ data: {
        expires_at: '2026-11-01T00:00:00Z', must_change_password: false,
      } }))
      .mockResolvedValueOnce(jsonResponse({ data: {
        userId: 'operator-1', email: 'operator@test.pe', displayName: 'Operador', workshopId: 'workshop-1',
      } }))
      .mockResolvedValueOnce(jsonResponse({ data: {
        workshop: { id: 'workshop-1', name: 'Taller Lima', timezone: 'America/Lima' }, role: 'OPERATOR',
      } }))
    vi.stubGlobal('fetch', fetchMock)
    const session = useSessionStore()

    await session.login('operator@test.pe', 'correct-password')

    expect(mobile.token).toBe('secure-token')
    expect(session.isAuthenticated).toBe(true)
    expect(fetchMock.mock.calls[0]?.[1].body).toContain('"client":"mobile"')
    expect(fetchMock.mock.calls[1]?.[1].headers.Authorization).toBe('Bearer secure-token')
    expect(fetchMock.mock.calls[1]?.[1].credentials).toBe('omit')
    expect(fetchMock.mock.calls[1]?.[1].headers).not.toHaveProperty('X-CSRF-Token')

    setActivePinia(createPinia())
    const restored = useSessionStore()
    await restored.restore()
    expect(restored.principal?.role).toBe('OPERATOR')
    expect(fetchMock.mock.calls[3]?.[1].headers.Authorization).toBe('Bearer secure-token')
    expect(localStorage.length).toBe(0)
  })

  it('uses the local web session in the Android USB preview', async () => {
    mobile.enabled = true
    mobile.localPreview = true
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ data: {
        expires_at: '2026-11-01T00:00:00Z', csrf_token: 'local-csrf', must_change_password: false,
      } }))
      .mockResolvedValueOnce(jsonResponse({ data: {
        userId: 'operator-1', email: 'operator@test.pe', displayName: 'Operador', workshopId: 'workshop-1',
      } }))
      .mockResolvedValueOnce(jsonResponse({ data: {
        workshop: { id: 'workshop-1', name: 'Taller Lima', timezone: 'America/Lima' }, role: 'OPERATOR',
      } }))
    vi.stubGlobal('fetch', fetchMock)

    const session = useSessionStore()
    await session.login('operator@test.pe', 'correct-password')

    expect(session.isAuthenticated).toBe(true)
    expect(mobile.token).toBeNull()
    expect(fetchMock.mock.calls[0]?.[1].body).not.toContain('"client":"mobile"')
    expect(fetchMock.mock.calls[0]?.[1].credentials).toBe('include')
  })
})

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })
}
