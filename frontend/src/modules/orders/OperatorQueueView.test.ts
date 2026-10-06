import { nextTick } from 'vue'
import { fireEvent, render, screen } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OperatorQueueView from './OperatorQueueView.vue'
import { ordersApi } from './api'

vi.mock('./api', () => ({ ordersApi: { list: vi.fn() } }))

describe('OperatorQueueView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(ordersApi.list).mockResolvedValue({
      data: [
        {
          id: 'order-1',
          code: 'OT-000001',
          product: 'Polo Demo',
          client_name: 'Cliente Demo',
          quantity: 12,
          unit: 'UNITS',
          status: 'ACTIVE',
          traffic_light: 'ON_TIME',
          current_stage: 'Corte',
        },
      ],
    } as never)
  })

  it('shows active work with a direct detail link', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/app/production', component: OperatorQueueView },
        { path: '/app/orders/:id', component: { template: '<main id="order-detail" />' } },
      ],
    })
    await router.push('/app/production')
    await router.isReady()
    render(OperatorQueueView, { global: { plugins: [createPinia(), router] } })

    expect(await screen.findByText('OT-000001')).toBeTruthy()
    expect(screen.getByRole('link', { name: /abrir ot-000001/i }).getAttribute('href')).toBe('/app/orders/order-1')
  })

  it('opens the scanner modal and navigates when entering a code manually', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/app/production', component: OperatorQueueView },
        { path: '/app/orders/:id', component: { template: '<main id="order-detail" />' } },
      ],
    })
    await router.push('/app/production')
    await router.isReady()
    render(OperatorQueueView, { global: { plugins: [createPinia(), router] } })

    // Wait for initial orders load
    expect(await screen.findByText('OT-000001')).toBeTruthy()

    // Open scanner modal
    const scanBtn = await screen.findByRole('button', { name: /escanear ficha viajera/i })
    await fireEvent.click(scanBtn)

    expect(await screen.findByRole('heading', { name: /escanear ficha viajera/i })).toBeTruthy()

    // Type manual code "OT-000001" and submit
    const input = screen.getByLabelText(/o ingresa el código o id de la orden/i)
    await fireEvent.update(input, 'OT-000001')

    const submitBtn = screen.getByRole('button', { name: /abrir orden/i })
    await fireEvent.click(submitBtn)
    await nextTick()
    await new Promise((resolve) => setTimeout(resolve, 10))

    // Should navigate to order detail
    expect(router.currentRoute.value.path).toBe('/app/orders/order-1')
  })
})
