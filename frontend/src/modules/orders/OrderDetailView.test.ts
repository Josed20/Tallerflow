import { fireEvent, render, screen, waitFor } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OrderDetailView from './OrderDetailView.vue'
import { ordersApi } from './api'
import { useSessionStore } from '../session/session.store'

vi.mock('./api', () => ({ ordersApi: { getByID: vi.fn(), update: vi.fn(), updateStage: vi.fn(), reopenStage: vi.fn(), addStagePhoto: vi.fn(), createTrackingLink: vi.fn() } }))

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
    expect(ordersApi.updateStage).not.toHaveBeenCalled()
    await fireEvent.click(screen.getByRole('button', { name: /seguir en esta etapa/i }))
    expect(ordersApi.updateStage).not.toHaveBeenCalled()
    await fireEvent.click(screen.getByRole('button', { name: /completar etapa/i }))
    await fireEvent.click(screen.getByRole('button', { name: /confirmar avance/i }))

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

  it('lets a manager correct only the last completed stage with a reason', async () => {
    const pinia = createPinia()
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/app/orders/:id', component: OrderDetailView }, { path: '/app/orders', component: { template: '<main />' } }] })
    await router.push('/app/orders/order-1')
    await router.isReady()
    const session = useSessionStore(pinia)
    session.csrfToken = 'csrf-correction'
    session.principal = { id: 'owner-1', email: 'owner@example.test', displayName: 'Owner', role: 'OWNER', workshop: { id: 'workshop-1', name: 'Taller' }, mustChangePassword: false }
    const advanced = { ...order, stages: [
      { ...order.stages[0], status: 'COMPLETED', completed_at: '2026-10-12T12:00:00Z' },
      { ...order.stages[1], status: 'IN_PROGRESS' },
    ] }
    vi.mocked(ordersApi.getByID).mockResolvedValue({ data: advanced } as never)
    vi.mocked(ordersApi.reopenStage).mockResolvedValue({ data: order.stages[0] } as never)
    render(OrderDetailView, { global: { plugins: [pinia, router] } })

    await fireEvent.click(await screen.findByRole('button', { name: /corregir avance/i }))
    await fireEvent.update(screen.getByLabelText(/motivo de la corrección/i), 'Se marcó por error')
    await fireEvent.click(screen.getByRole('button', { name: /confirmar corrección/i }))
    await waitFor(() => expect(ordersApi.reopenStage).toHaveBeenCalledWith('order-1', 1, 'Se marcó por error', 'csrf-correction'))
  })

  it('hides correction when the next stage already has evidence', async () => {
    const pinia = createPinia()
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/app/orders/:id', component: OrderDetailView }, { path: '/app/orders', component: { template: '<main />' } }] })
    await router.push('/app/orders/order-1')
    await router.isReady()
    useSessionStore(pinia).principal = { id: 'owner-1', email: 'owner@example.test', displayName: 'Owner', role: 'OWNER', workshop: { id: 'workshop-1', name: 'Taller' }, mustChangePassword: false }
    vi.mocked(ordersApi.getByID).mockResolvedValue({ data: {
      ...order,
      stages: [{ ...order.stages[0], status: 'COMPLETED' }, { ...order.stages[1], status: 'IN_PROGRESS' }],
      photos: [{ id: 'photo-1', stage_position: 2, photo_data: 'data:image/png;base64,AA', created_at: '2026-10-12T12:00:00Z' }],
    } } as never)
    render(OrderDetailView, { global: { plugins: [pinia, router] } })
    await screen.findByText('Costura')
    expect(screen.queryByRole('button', { name: /corregir avance/i })).toBeNull()
  })

  async function renderAs(role: 'OWNER' | 'OPERATOR') {
    const pinia = createPinia()
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/app/orders/:id', component: OrderDetailView }, { path: '/app/orders', component: { template: '<main />' } }],
    })
    await router.push('/app/orders/order-1')
    await router.isReady()
    const session = useSessionStore(pinia)
    session.csrfToken = 'csrf-share'
    session.principal = { id: 'user-1', email: 'user@example.test', displayName: 'User', role, workshop: { id: 'workshop-1', name: 'Taller' }, mustChangePassword: false }
    render(OrderDetailView, { global: { plugins: [pinia, router] } })
    await screen.findByText('Corte')
  }

  it('lets a manager generate a customer tracking link only after confirming', async () => {
    vi.mocked(ordersApi.createTrackingLink).mockResolvedValue({ data: { token: 'T'.repeat(43) } } as never)
    await renderAs('OWNER')

    await fireEvent.click(screen.getByRole('button', { name: /compartir con el cliente/i }))
    expect(ordersApi.createTrackingLink).not.toHaveBeenCalled()
    await fireEvent.click(screen.getByRole('button', { name: /generar enlace/i }))

    await waitFor(() => expect(ordersApi.createTrackingLink).toHaveBeenCalledWith('order-1', 'csrf-share'))
    expect((await screen.findByLabelText('Enlace privado') as HTMLInputElement).value).toBe(`${window.location.origin}/seguimiento/${'T'.repeat(43)}`)
  })

  it('does not offer customer sharing to an operator', async () => {
    await renderAs('OPERATOR')
    expect(screen.queryByRole('button', { name: /compartir con el cliente/i })).toBeNull()
  })
  it('lets a manager open edit modal and submit order updates', async () => {
    vi.mocked(ordersApi.update).mockResolvedValue({ data: { ...order, product: 'Polo Pique Actualizado', quantity: 24 } } as never)
    await renderAs('OWNER')

    const editBtn = await screen.findByRole('button', { name: /editar orden/i })
    expect(editBtn).toBeTruthy()
    await fireEvent.click(editBtn)

    expect(screen.getByRole('dialog', { name: /editar orden/i })).toBeTruthy()
    const productInput = screen.getByLabelText(/prenda \/ producto/i) as HTMLInputElement
    const quantityInput = screen.getByLabelText(/cantidad/i) as HTMLInputElement

    await fireEvent.update(productInput, 'Polo Pique Actualizado')
    await fireEvent.update(quantityInput, '24')
    await fireEvent.click(screen.getByRole('button', { name: /guardar cambios/i }))

    await waitFor(() => expect(ordersApi.update).toHaveBeenCalled())
  })

  it('does not offer order editing to an operator', async () => {
    await renderAs('OPERATOR')
    expect(screen.queryByRole('button', { name: /editar orden/i })).toBeNull()
  })
});
