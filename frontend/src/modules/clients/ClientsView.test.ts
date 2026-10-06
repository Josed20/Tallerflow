import { fireEvent, render, screen } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ClientsView from './ClientsView.vue'
import { clientsApi } from './api'
import { useSessionStore } from '../session/session.store'

vi.mock('./api', () => ({
  clientsApi: { list: vi.fn(), create: vi.fn(), update: vi.fn() },
}))

async function renderClients(role: 'OWNER' | 'ADMIN' | 'OPERATOR' = 'OWNER') {
  const pinia = createPinia()
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/app', component: { template: '<main />' } }, { path: '/app/clients', component: ClientsView }],
  })
  await router.push('/app/clients')
  await router.isReady()
  const session = useSessionStore(pinia)
  session.csrfToken = 'csrf-client'
  session.principal = {
    id: 'user-1', email: 'owner@example.test', displayName: 'Owner', role,
    workshop: { id: 'workshop-1', name: 'Taller Demo' }, mustChangePassword: false,
  }
  render(ClientsView, { global: { plugins: [pinia, router] } })
}

describe('ClientsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(clientsApi.list).mockResolvedValue({ data: [] })
    vi.mocked(clientsApi.create).mockResolvedValue({
      data: { id: 'client-1', workshop_id: 'workshop-1', name: 'Confecciones Demo', email: null, phone: null, is_active: true, created_at: '', updated_at: '' },
    })
  })

  it('creates a client with the current CSRF token', async () => {
    await renderClients()
    await fireEvent.click(await screen.findByRole('button', { name: /registrar el primer cliente/i }))
    await fireEvent.update(screen.getByLabelText(/nombre o razón social/i), 'Confecciones Demo')
    await fireEvent.click(screen.getByRole('button', { name: /guardar cliente/i }))

    expect(clientsApi.create).toHaveBeenCalledWith(
      { name: 'Confecciones Demo', phone: null, email: null },
      'csrf-client',
    )
  })

  it('does not show client creation controls to an operator', async () => {
    await renderClients('OPERATOR')

    await screen.findByText(/aún no hay clientes/i)
    expect(screen.queryByRole('button', { name: /nuevo cliente/i })).toBeNull()
    expect(screen.queryByRole('button', { name: /registrar el primer cliente/i })).toBeNull()
  })
})
