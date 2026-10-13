import { createPinia } from 'pinia'
import { createMemoryHistory } from 'vue-router'
import { describe, expect, it, vi } from 'vitest'
import { createAppRouter } from './router'
import { useSessionStore } from '../modules/session/session.store'

describe('role navigation', () => {
  it('redirects an operator away from client management to orders', async () => {
    const pinia = createPinia()
    const session = useSessionStore(pinia)
    session.csrfToken = 'csrf-token'
    session.principal = { id: 'operator-1', email: 'operator@example.test', displayName: 'Operator', role: 'OPERATOR', workshop: { id: 'workshop-1', name: 'Taller' }, mustChangePassword: false }
    vi.spyOn(session, 'restore').mockResolvedValue()
    const router = createAppRouter({ pinia, history: createMemoryHistory() })

    await router.push('/app/clients')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/app/orders')
  })
})
