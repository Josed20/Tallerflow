import { fireEvent, render, screen } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OrderCreateView from './OrderCreateView.vue'
import { clientsApi } from '../clients/api'
import { ordersApi } from './api'
import { productionApi } from './production-api'
import { useSessionStore } from '../session/session.store'

vi.mock('../clients/api', () => ({ clientsApi: { list: vi.fn() } }))
vi.mock('./api', () => ({ ordersApi: { create: vi.fn() } }))
vi.mock('./production-api', () => ({ productionApi: { get: vi.fn() }, defaultStages: [{ name: 'Corte' }] }))

describe('OrderCreateView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(productionApi.get).mockResolvedValue({ data: { version: 1, stages: [{ name: 'MUESTRA', client_description: 'Preparamos tu muestra.' }] } })
    vi.mocked(clientsApi.list).mockResolvedValue({ data: [] })
  })

  it('guides a manager to create a client when the workshop has none', async () => {
    const pinia = createPinia()
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/app/clients', component: { template: '<main />' } },
        { path: '/app/orders/new', component: OrderCreateView },
      ],
    })
    await router.push('/app/orders/new')
    await router.isReady()
    render(OrderCreateView, { global: { plugins: [pinia, router] } })

    expect(await screen.findByText(/no tienes clientes registrados/i)).toBeTruthy()
    expect(screen.queryByRole('form')).toBeNull()
    expect(screen.getByRole('button', { name: /registrar cliente/i })).toBeTruthy()
  })

  it.each([{ quantity: 24, unit: 'UNITS' }, { quantity: 0.5, unit: 'KILOGRAMS' }, { quantity: 0.25, unit: 'METERS' }])('submits $quantity $unit and previews the actual first stage', async ({ quantity, unit }) => {
    vi.mocked(clientsApi.list).mockResolvedValue({
      data: [{ id: 'client-1', workshop_id: 'workshop-1', name: 'Cliente Demo', email: null, phone: null, is_active: true, created_at: '', updated_at: '' }],
    })
    vi.mocked(ordersApi.create).mockResolvedValue({ data: { id: 'order-1' } } as never)
    const pinia = createPinia()
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/app/orders/:id', component: { template: '<main />' } },
        { path: '/app/orders/new', component: OrderCreateView },
      ],
    })
    await router.push('/app/orders/new')
    await router.isReady()
    useSessionStore(pinia).csrfToken = 'csrf-order'
    render(OrderCreateView, { global: { plugins: [pinia, router] } })

    await screen.findByLabelText(/producto/i)
    await fireEvent.update(screen.getByLabelText(/producto/i), 'Polo demo')
    await fireEvent.update(screen.getByLabelText(/unidad/i), unit)
    const quantityInput = screen.getByLabelText(/cantidad/i) as HTMLInputElement
    await fireEvent.update(quantityInput, String(quantity))
    expect(quantityInput.validity.valid).toBe(true)
    expect(screen.getByText(/La orden comenzará en MUESTRA/)).toBeTruthy()
    await fireEvent.click(screen.getByRole('button', { name: /^crear orden$/i }))

    expect(ordersApi.create).toHaveBeenCalledWith(
      expect.objectContaining({ client_id: 'client-1', product: 'Polo demo', quantity, unit }),
      expect.any(String),
      'csrf-order',
    )
  })
})
