import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useSessionStore } from './session.store'

describe('session store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('restores the authenticated principal only in memory', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({
        id: 'owner-1',
        email: 'owner@taller.pe',
        role: 'OWNER',
        workshop: { id: 'workshop-1', name: 'Taller San Martín' },
        mustChangePassword: false,
      }), { status: 200, headers: { 'Content-Type': 'application/json' } }),
    )
    vi.stubGlobal('fetch', fetchMock)
    const session = useSessionStore()

    await session.restore()

    expect(session.principal?.workshop.name).toBe('Taller San Martín')
    expect(session.status).toBe('authenticated')
    expect(session.restored).toBe(true)
    expect(localStorage.length).toBe(0)
  })
})
