import { nextTick } from 'vue'
import { fireEvent, render, screen } from '@testing-library/vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OrderScannerDialog from './OrderScannerDialog.vue'
import type { Order } from './api'

const native = vi.hoisted(() => ({ enabled: false, scan: vi.fn() }))
vi.mock('../../native/scanner', () => ({
  canScanNatively: () => native.enabled,
  scanOrderCode: native.scan,
}))

const orders = [{ id: 'order-1', code: 'OT-000001', order_number: 1 }] as Order[]

async function renderScanner() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/app/orders', component: { template: '<main />' } },
      { path: '/app/orders/:id', component: { template: '<main id="order-detail" />' } },
    ],
  })
  await router.push('/app/orders')
  await router.isReady()
  const view = render(OrderScannerDialog, { props: { orders }, global: { plugins: [router] } })
  return { router, view }
}

describe('OrderScannerDialog', () => {
  beforeEach(() => {
    native.enabled = false
    native.scan.mockReset()
  })

  it('opens an order read by the Android scanner', async () => {
    native.enabled = true
    native.scan.mockResolvedValue('OT-000001')
    const { router } = await renderScanner()
    await new Promise((resolve) => setTimeout(resolve, 10))
    expect(native.scan).toHaveBeenCalledOnce()
    expect(router.currentRoute.value.path).toBe('/app/orders/order-1')
  })

  it('opens an order from a manually entered short code', async () => {
    const { router } = await renderScanner()

    expect(await screen.findByRole('heading', { name: /escanear ficha viajera/i })).toBeTruthy()
    await fireEvent.update(screen.getByLabelText(/o ingresa el código o id de la orden/i), 'OT-1')
    await fireEvent.click(screen.getByRole('button', { name: /abrir orden/i }))
    await nextTick()
    await new Promise((resolve) => setTimeout(resolve, 10))

    expect(router.currentRoute.value.path).toBe('/app/orders/order-1')
  })

  it('explains when a code does not match any order', async () => {
    await renderScanner()

    await fireEvent.update(screen.getByLabelText(/o ingresa el código o id de la orden/i), 'OT-999')
    await fireEvent.click(screen.getByRole('button', { name: /abrir orden/i }))

    expect(await screen.findByText(/no se encontró la orden/i)).toBeTruthy()
  })
})
