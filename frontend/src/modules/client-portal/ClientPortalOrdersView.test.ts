import { render, screen } from '@testing-library/vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ClientPortalOrdersView from './ClientPortalOrdersView.vue'
import { clientPortalApi } from './api'

vi.mock('./api', () => ({ clientPortalApi: { listOrders: vi.fn(), logout: vi.fn() } }))

describe('ClientPortalOrdersView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(clientPortalApi.listOrders).mockResolvedValue({ data: [] })
  })

  it('renders an empty client order state without internal navigation', async () => {
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/portal/orders', component: ClientPortalOrdersView }] })
    await router.push('/portal/orders')
    await router.isReady()
    render(ClientPortalOrdersView, { global: { plugins: [router] } })

    expect(await screen.findByText(/aún no tienes pedidos/i)).toBeTruthy()
    expect(screen.queryByText(/equipo|clientes|nueva orden/i)).toBeNull()
  })
})
