import { fireEvent, render, screen, waitFor } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OrdersListView from './OrdersListView.vue'
import { ordersApi } from './api'
import { useSessionStore } from '../session/session.store'

vi.mock('./api', () => ({ ordersApi: { list: vi.fn(), getByID: vi.fn(), updateStage: vi.fn() } }))

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

  it('moves a card to the next station after the server completes the active stage', async () => {
    const pinia = createPinia()
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/app/orders', component: OrdersListView }, { path: '/app/orders/new', component: { template: '<main />' } }, { path: '/app/orders/:id', component: { template: '<main />' } }],
    })
    await router.push('/app/orders')
    await router.isReady()
    const session = useSessionStore(pinia)
    session.csrfToken = 'csrf-board'
    session.principal = {
      id: 'owner-1', email: 'owner@example.test', displayName: 'Owner', role: 'OWNER',
      workshop: { id: 'workshop-1', name: 'Taller Demo' }, mustChangePassword: false,
    }
    const baseOrder = {
      id: 'order-1', code: 'OT-000001', product: 'Polos', client_name: 'Cliente Demo', quantity: 12,
      unit: 'UNITS', status: 'ACTIVE', traffic_light: 'ON_TIME', current_stage: 'CORTE', promised_at: '2026-11-20T18:00:00Z',
    }
    vi.mocked(ordersApi.list)
      .mockResolvedValueOnce({ data: [baseOrder] } as never)
      .mockResolvedValueOnce({ data: [{ ...baseOrder, current_stage: 'COSTURA' }] } as never)
    vi.mocked(ordersApi.getByID).mockResolvedValue({ data: { ...baseOrder, stages: [{ position: 1, status: 'IN_PROGRESS', name: 'CORTE' }] } } as never)
    vi.mocked(ordersApi.updateStage).mockResolvedValue({ data: { position: 1, status: 'COMPLETED' } } as never)
    render(OrdersListView, { global: { plugins: [pinia, router] } })

    const advanceButton = await screen.findByRole('button', { name: /pasar a costura/i })
    await fireEvent.click(advanceButton)
    expect(ordersApi.updateStage).not.toHaveBeenCalled()
    await fireEvent.click(screen.getByRole('button', { name: /seguir en esta etapa/i }))
    expect(ordersApi.updateStage).not.toHaveBeenCalled()
    await fireEvent.click(advanceButton)
    await fireEvent.click(screen.getByRole('button', { name: /confirmar avance/i }))

    await waitFor(() => expect(ordersApi.updateStage).toHaveBeenCalledWith('order-1', 1, 'COMPLETED', 'csrf-board'))
    await screen.findByText(/pasó a costura/i)
    expect(screen.getByRole('region', { name: /costura, 1 órdenes/i })).toBeTruthy()
  })

  it('switches filters without retaining delivery-today and finds an order by number', async () => {
    const pinia = createPinia()
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/app/orders', component: OrdersListView }, { path: '/app/orders/:id', component: { template: '<main />' } }],
    })
    await router.push('/app/orders?view=list&due=today')
    await router.isReady()
    const session = useSessionStore(pinia)
    session.principal = { id: 'owner-1', email: 'owner@example.test', displayName: 'Owner', role: 'OWNER', workshop: { id: 'workshop-1', name: 'Taller' }, mustChangePassword: false }
    vi.mocked(ordersApi.list).mockResolvedValue({ data: [{
      id: 'order-3', order_number: 3, code: 'OT-000003', product: 'Polo', client_name: 'Cliente',
      quantity: 1, unit: 'UNITS', status: 'ACTIVE', traffic_light: 'LATE', current_stage: 'CORTE', promised_at: '2026-01-01T18:00:00Z',
    }] } as never)
    render(OrdersListView, { global: { plugins: [pinia, router] } })

    await screen.findByText(/no hay órdenes que coincidan/i)
    await fireEvent.click(screen.getByRole('button', { name: /todas 1/i }))
    expect(await screen.findByText('OT-000003')).toBeTruthy()
    expect(screen.getByRole('button', { name: /entrega hoy/i }).getAttribute('aria-pressed')).toBe('false')
    await fireEvent.update(screen.getByRole('searchbox', { name: /buscar órdenes/i }), '3')
    expect(screen.getByText('OT-000003')).toBeTruthy()
    await fireEvent.click(screen.getByRole('button', { name: /a tiempo 0/i }))
    await screen.findByText(/no hay órdenes que coincidan/i)
    await fireEvent.click(screen.getByRole('button', { name: /atrasadas 1/i }))
    expect(screen.getByText('OT-000003')).toBeTruthy()
  })
})
