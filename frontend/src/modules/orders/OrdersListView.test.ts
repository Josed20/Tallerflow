import { render, screen } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OrdersListView from './OrdersListView.vue'
import { ordersApi } from './api'
import { useSessionStore } from '../session/session.store'

vi.mock('./api', () => ({ ordersApi: { list: vi.fn() } }))

describe('OrdersListView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(ordersApi.list).mockResolvedValue({ data: [] })
  })

  it('does not show new-order controls to an operator', async () => {
    const pinia = createPinia()
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/app', component: { template: '<main />' } }, { path: '/app/orders', component: OrdersListView }],
    })
    await router.push('/app/orders')
    await router.isReady()
    const session = useSessionStore(pinia)
    session.principal = {
      id: 'operator-1', email: 'operator@example.test', displayName: 'Operator', role: 'OPERATOR',
      workshop: { id: 'workshop-1', name: 'Taller Demo' }, mustChangePassword: false,
    }
    render(OrdersListView, { global: { plugins: [pinia, router] } })

    await screen.findByText(/no hay órdenes/i)
    expect(screen.queryByRole('button', { name: /nueva orden/i })).toBeNull()
    expect(screen.queryByRole('button', { name: /crear la primera orden/i })).toBeNull()
  })
})
