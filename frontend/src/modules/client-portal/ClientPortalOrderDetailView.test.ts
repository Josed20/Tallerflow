import { render, screen } from '@testing-library/vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ClientPortalOrderDetailView from './ClientPortalOrderDetailView.vue'
import { clientPortalApi } from './api'

vi.mock('./api', () => ({ clientPortalApi: { getOrder: vi.fn() } }))

describe('ClientPortalOrderDetailView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(clientPortalApi.getOrder).mockResolvedValue({
      data: { id: 'order-1', code: 'OT-000001', product: 'Polo Demo', quantity: 12, unit: 'UNITS', status: 'ACTIVE', traffic_light: 'ON_TIME', promised_at: '2026-10-12T18:00:00Z', updated_at: '2026-10-02T10:00:00Z', stages: [{ position: 1, name: 'Corte', status: 'IN_PROGRESS', completed_at: null }] },
    })
  })

  it('shows only the private order timeline', async () => {
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/portal/orders', component: { template: '<main />' } }, { path: '/portal/orders/:id', component: ClientPortalOrderDetailView }] })
    await router.push('/portal/orders/order-1')
    await router.isReady()
    render(ClientPortalOrderDetailView, { global: { plugins: [router] } })

    expect(await screen.findByText('Corte')).toBeTruthy()
    expect(screen.queryByText(/nota privada|operador|administrador/i)).toBeNull()
  })
})
