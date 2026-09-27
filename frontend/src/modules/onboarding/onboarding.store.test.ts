import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useOnboardingStore } from './onboarding.store'
import { useSessionStore } from '../session/session.store'

describe('onboarding store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.unstubAllGlobals()
  })

  it('caches public availability until a forced refresh', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ data: { available: true } }))
      .mockResolvedValueOnce(jsonResponse({ data: { available: false } }))
    vi.stubGlobal('fetch', fetchMock)
    const onboarding = useOnboardingStore()

    expect(await onboarding.loadStatus()).toBe(true)
    expect(await onboarding.loadStatus()).toBe(true)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(await onboarding.loadStatus(true)).toBe(false)
    expect(onboarding.availability).toBe('claimed')
  })

  it('creates the owner and hands the returned session to the session store', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ data: { expires_at: '2026-09-28T01:00:00Z', csrf_token: 'csrf-web', must_change_password: false } }, 201))
      .mockResolvedValueOnce(jsonResponse({ data: { userId: 'owner-web', email: 'owner@web.test', displayName: 'Owner Web', workshopId: 'workshop-web', role: 'OWNER', passwordChangeRequired: false } }))
      .mockResolvedValueOnce(jsonResponse({ data: { workshop: { id: 'workshop-web', name: 'Taller Web', timezone: 'America/Lima' }, role: 'OWNER' } }))
    vi.stubGlobal('fetch', fetchMock)
    const onboarding = useOnboardingStore()
    const session = useSessionStore()

    await onboarding.create({
      workshop_name: 'Taller Web', owner_name: 'Owner Web', email: 'owner@web.test',
      password: 'Secure password 123!', password_confirmation: 'Secure password 123!',
    })

    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/v1/onboarding/workshop', expect.objectContaining({
      method: 'POST', credentials: 'include', body: expect.stringContaining('Taller Web'),
    }))
    expect(onboarding.availability).toBe('claimed')
    expect(session.isAuthenticated).toBe(true)
    expect(session.principal?.id).toBe('owner-web')
  })
})

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}
