import { fireEvent, render, screen, waitFor, within } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { downloadOrdersCsv } from './order-csv'
import OrdersListView from './OrdersListView.vue'
import { ordersApi } from './api'
import { useSessionStore } from '../session/session.store'

vi.mock('./order-csv', () => ({ downloadOrdersCsv: vi.fn() }))
vi.mock('./api', () => ({ ordersApi: { list: vi.fn(), getByID: vi.fn(), updateStage: vi.fn() } }))
vi.mock('./production-api', () => ({ productionApi: { get: vi.fn().mockResolvedValue({ data: { stages: [{ name: 'MUESTRA' }, { name: 'ENTREGA' }] } }) }, defaultStages: [{ name: 'CORTE' }] }))

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
    vi.mocked(ordersApi.getByID).mockResolvedValue({ data: { ...baseOrder, stages: [{ position: 1, status: 'IN_PROGRESS', name: 'CORTE' }, { position: 2, status: 'PENDING', name: 'COSTURA' }] } } as never)
    vi.mocked(ordersApi.updateStage).mockResolvedValue({ data: { position: 1, status: 'COMPLETED' } } as never)
    render(OrdersListView, { global: { plugins: [pinia, router] } })

    const advanceButton = await screen.findByRole('button', { name: /completar etapa/i })
    await fireEvent.click(advanceButton)
    expect(ordersApi.updateStage).not.toHaveBeenCalled()
    await fireEvent.click(await screen.findByRole('button', { name: /seguir en esta etapa/i }))
    expect(ordersApi.updateStage).not.toHaveBeenCalled()
    await fireEvent.click(advanceButton)
    await fireEvent.click(await screen.findByRole('button', { name: /confirmar avance/i }))

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

async function renderBoard(path = '/app/orders?view=board') {
  const pinia = createPinia()
  useSessionStore(pinia).principal = { id: 'owner', email: 'owner@example.test', displayName: 'Owner', role: 'OWNER', workshop: { id: 'w1', name: 'Taller' }, mustChangePassword: false }
  const router = createRouter({ history: createMemoryHistory(), routes: ['/app/orders', '/app/orders/new', '/app/orders/:id'].map(path => ({ path, component: { template: '<main />' } })) })
  await router.push(path)
  render(OrdersListView, { global: { plugins: [pinia, router] } })
}

afterEach(() => { vi.restoreAllMocks(); vi.useRealTimers() })

it('shows cancelled cards and exports exactly the current board result', async () => {
  const base = { product: 'Polos', client_name: 'Cliente', quantity: 12, unit: 'UNITS', current_stage: 'MUESTRA', promised_at: '2026-12-01T18:00:00Z' }
  const orders = [
    { ...base, id: 'active', code: 'OT-A', status: 'ACTIVE', traffic_light: 'LATE' },
    { ...base, id: 'done', code: 'OT-D', status: 'COMPLETED', traffic_light: 'COMPLETED' },
    { ...base, id: 'cancelled', code: 'OT-C', status: 'CANCELLED', traffic_light: 'LATE' },
  ]
  vi.mocked(ordersApi.list).mockResolvedValue({ data: orders } as never)
  await renderBoard()
  const cancelledColumn = await screen.findByRole('region', { name: 'Canceladas, 1 órdenes' })
  expect(within(cancelledColumn).getByText('OT-C')).toBeTruthy()
  expect(within(cancelledColumn).getByText('Cancelada')).toBeTruthy()
  expect(within(cancelledColumn).queryByRole('button', { name: /completar etapa/i })).toBeNull()
  expect(screen.getAllByRole('article')).toHaveLength(3)
  expect(screen.getByRole('button', { name: /atrasadas 1/i })).toBeTruthy()
  await fireEvent.click(screen.getByRole('button', { name: /exportar CSV \(3\)/i }))
  expect(downloadOrdersCsv).toHaveBeenLastCalledWith(orders, 'OWNER')
  await fireEvent.click(screen.getByRole('button', { name: /canceladas 1/i }))
  expect(screen.getAllByRole('article')).toHaveLength(1)
  await fireEvent.click(screen.getByRole('button', { name: /exportar CSV \(1\)/i }))
  expect(downloadOrdersCsv).toHaveBeenLastCalledWith([orders[2]], 'OWNER')
  await fireEvent.click(screen.getByRole('button', { name: /atrasadas 1/i }))
  expect(screen.queryByText('OT-C')).toBeNull()
  expect(screen.getByText('OT-A')).toBeTruthy()
})

it('uses the Lima delivery day even when the device calendar is UTC', async () => {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(new Date('2026-11-25T02:00:00Z'))
  const original = Date.prototype.toLocaleDateString
  vi.spyOn(Date.prototype, 'toLocaleDateString').mockImplementation(function (this: Date, locales, options) {
    return original.call(this, locales, options ?? { timeZone: 'UTC' })
  })
  const base = { product: 'Polos', client_name: 'Cliente', quantity: 1, unit: 'UNITS', status: 'ACTIVE', traffic_light: 'ON_TIME', current_stage: 'MUESTRA' }
  vi.mocked(ordersApi.list).mockResolvedValue({ data: [
    { ...base, id: 'today', code: 'OT-LIMA-TODAY', promised_at: '2026-11-25T03:00:00Z' },
    { ...base, id: 'tomorrow', code: 'OT-LIMA-TOMORROW', promised_at: '2026-11-25T06:00:00Z' },
    { ...base, id: 'done-today', code: 'OT-DONE-TODAY', status: 'COMPLETED', traffic_light: 'COMPLETED', promised_at: '2026-11-25T03:00:00Z' },
    { ...base, id: 'cancelled-today', code: 'OT-CANCELLED-TODAY', status: 'CANCELLED', promised_at: '2026-11-25T03:00:00Z' },
  ] } as never)
  await renderBoard('/app/orders?view=list&due=today')
  expect(await screen.findByText('OT-LIMA-TODAY')).toBeTruthy()
  expect(screen.queryByText('OT-LIMA-TOMORROW')).toBeNull()
  expect(screen.queryByText('OT-DONE-TODAY')).toBeNull()
  expect(screen.queryByText('OT-CANCELLED-TODAY')).toBeNull()
})
