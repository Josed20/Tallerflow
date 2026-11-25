import { fireEvent, render, screen, waitFor } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OrdersListView from './OrdersListView.vue'
import { ordersApi } from './api'
import { downloadOrdersCsv } from './order-csv'
import { useSessionStore } from '../session/session.store'
vi.mock('./api', () => ({ ordersApi: { list: vi.fn() } }))
vi.mock('./order-csv', () => ({ downloadOrdersCsv: vi.fn() }))
const order = { id: 'o1', code: 'OT-000001', product: 'Polos', client_name: 'Sur', quantity: 12, unit: 'UNITS', status: 'ACTIVE', traffic_light: 'ON_TIME', current_stage: 'CORTE', promised_at: '2026-12-01T18:00:00Z', responsible_user_id: 'u2', block_reason: 'Falta tela' }
async function setup(role: 'OWNER' | 'ADMIN' | 'OPERATOR') {
  const pinia = createPinia()
  useSessionStore(pinia).principal = { id: 'u1', email: 'user@example.test', displayName: 'User', role, workshop: { id: 'w1', name: 'Taller' }, mustChangePassword: false }
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/app/orders', component: OrdersListView }, { path: '/app/orders/new', component: { template: '<main />' } }, { path: '/app/orders/:id', component: { template: '<main />' } }] })
  await router.push('/app/orders?view=list')
  render(OrdersListView, { global: { plugins: [pinia, router] } })
}
beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(ordersApi.list).mockResolvedValue({ data: [order, { ...order, id: 'o2', code: 'OT-000002', product: 'Casacas', traffic_light: 'LATE' }] } as never)
})
describe('CSV export controls', () => {
  it.each(['OWNER', 'ADMIN'] as const)('exports only filtered orders for %s', async role => {
    await setup(role)
    await screen.findByText('Polos')
    await fireEvent.update(screen.getByRole('searchbox', { name: /buscar órdenes/i }), 'Polos')
    await fireEvent.click(screen.getByRole('button', { name: /exportar CSV \(1\)/i }))
    expect(downloadOrdersCsv).toHaveBeenCalledWith([order], role)
    await fireEvent.update(screen.getByRole('searchbox', { name: /buscar órdenes/i }), 'no existe')
    expect((screen.getByRole('button', { name: /exportar CSV \(0\)/i }) as HTMLButtonElement).disabled).toBe(true)
  })
  it('hides export from operators', async () => {
    await setup('OPERATOR')
    await screen.findByText('Polos')
    expect(screen.queryByRole('button', { name: /exportar CSV/i })).toBeNull()
  })
  it('disables export after a failed load', async () => {
    vi.mocked(ordersApi.list).mockRejectedValue(new Error('Sin conexión'))
    await setup('OWNER')
    await waitFor(() => expect(screen.getByText('Sin conexión')).toBeTruthy())
    expect((screen.getByRole('button', { name: /exportar CSV/i }) as HTMLButtonElement).disabled).toBe(true)
    expect(downloadOrdersCsv).not.toHaveBeenCalled()
  })
})
