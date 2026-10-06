import { fireEvent, render, screen, waitFor } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OrderDetailView from './OrderDetailView.vue'
import { ordersApi } from './api'
import { useSessionStore } from '../session/session.store'

vi.mock('./api', () => ({ ordersApi: { getByID: vi.fn(), updateStage: vi.fn(), addStagePhoto: vi.fn() } }))

const order = {
  id: 'order-1', code: 'OT-000001', product: 'Polo Demo', client_name: 'Cliente Demo', quantity: 12, unit: 'UNITS', status: 'ACTIVE',
  specification: '', promised_at: '2026-10-12T18:00:00Z', created_at: '2026-10-02T10:00:00Z', traffic_light: 'ON_TIME',
  stages: [
    { id: 'stage-1', workshop_id: 'workshop-1', order_id: 'order-1', position: 1, name: 'Corte', status: 'IN_PROGRESS', completed_at: null },
    { id: 'stage-2', workshop_id: 'workshop-1', order_id: 'order-1', position: 2, name: 'Costura', status: 'PENDING', completed_at: null },
  ],
}

describe('OrderDetailView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(ordersApi.getByID).mockResolvedValue({ data: order } as never)
    vi.mocked(ordersApi.updateStage).mockResolvedValue({ data: { ...order.stages[0], status: 'COMPLETED' } } as never)
  })

  it('lets an operator complete only the current stage and sends CSRF', async () => {
    const pinia = createPinia()
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/app/orders/:id', component: OrderDetailView }, { path: '/app/orders', component: { template: '<main />' } }],
    })
    await router.push('/app/orders/order-1')
    await router.isReady()
    const session = useSessionStore(pinia)
    session.csrfToken = 'csrf-stage'
    session.principal = { id: 'operator-1', email: 'operator@example.test', displayName: 'Operator', role: 'OPERATOR', workshop: { id: 'workshop-1', name: 'Taller' }, mustChangePassword: false }
    render(OrderDetailView, { global: { plugins: [pinia, router] } })

    await screen.findByText('Corte')
    expect(screen.getByRole('button', { name: /completar etapa/i })).toBeTruthy()
    expect(screen.queryByRole('button', { name: /^iniciar$/i })).toBeNull()
    await fireEvent.click(screen.getByRole('button', { name: /completar etapa/i }))

    await waitFor(() => expect(ordersApi.updateStage).toHaveBeenCalledWith('order-1', 1, 'COMPLETED', 'csrf-stage'))
  })

  it('does not expose production photo upload to a supervisor', async () => {
    const pinia = createPinia()
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/app/orders/:id', component: OrderDetailView }, { path: '/app/orders', component: { template: '<main />' } }],
    })
    await router.push('/app/orders/order-1')
    await router.isReady()
    const session = useSessionStore(pinia)
    session.csrfToken = 'csrf-supervisor'
    session.principal = { id: 'supervisor-1', email: 'supervisor@example.test', displayName: 'Supervisor', role: 'SUPERVISOR', workshop: { id: 'workshop-1', name: 'Taller' }, mustChangePassword: false }
    render(OrderDetailView, { global: { plugins: [pinia, router] } })

    await screen.findByText('Corte')
    expect(screen.queryByTitle(/tomar foto de evidencia/i)).toBeNull()
  })

  it('does not advertise a QR code when the printable ticket has no standards-compliant QR generator', async () => {
    const pinia = createPinia()
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/app/orders/:id', component: OrderDetailView }, { path: '/app/orders', component: { template: '<main />' } }],
    })
    await router.push('/app/orders/order-1')
    await router.isReady()
    const session = useSessionStore(pinia)
    session.principal = { id: 'owner-1', email: 'owner@example.test', displayName: 'Owner', role: 'OWNER', workshop: { id: 'workshop-1', name: 'Taller' }, mustChangePassword: false }
    render(OrderDetailView, { global: { plugins: [pinia, router] } })

    const ticketButton = await screen.findByRole('button', { name: /imprimir ficha viajera/i })
    expect(ticketButton.textContent).not.toMatch(/qr/i)
  })
})
