import { render, screen, fireEvent } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { expect, it, vi } from 'vitest'
import { ordersApi } from '../orders/api'
import SessionHomeView from './SessionHomeView.vue'
import { useSessionStore } from './session.store'

vi.mock('../orders/api', () => ({ ordersApi: {
  getDashboard: vi.fn().mockResolvedValue({ data: { active: 1, on_time: 0, late: 1, at_risk: 0, completed: 0 } }),
  list: vi.fn().mockResolvedValue({ data: [{ id: 'rr-order', code: 'OT-000001', product: 'Polos R&R', quantity: 600, unit: 'UNITS', current_stage: 'MUESTRA', stage_count: 6, completed_stage_count: 0, status: 'ACTIVE', traffic_light: 'LATE', promised_at: '2026-01-01T18:00:00Z', block_reason: 'Falta aprobar muestra' }] }),
} }))
vi.mock('../clients/api', () => ({ clientsApi: { list: vi.fn().mockResolvedValue({ data: [] }) } }))
vi.mock('../team/api', () => ({ teamApi: { list: vi.fn().mockResolvedValue({ data: { members: [], invitations: [] } }) } }))
vi.mock('../orders/ProductionSettingsPanel.vue', () => ({ default: { template: '<div />' } }))

it('counts 600 garments as one order, exposes custom stage load and actionable blocks', async () => {
  const pinia = createPinia()
  useSessionStore(pinia).principal = { id: 'owner', role: 'OWNER', displayName: 'Owner', email: '', workshop: { id: 'rr', name: 'R&R' }, mustChangePassword: false }
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: SessionHomeView }, { path: '/app/orders/:id', component: { template: '<main />' } }] })
  await router.push('/')
  render(SessionHomeView, { global: { plugins: [pinia, router] } })
  const stage = await screen.findByRole('button', { name: /1 pedido MUESTRA 600 unidades/ })
  await fireEvent.click(stage)
  expect(screen.getByText('Etapa seleccionada: MUESTRA')).toBeTruthy()
  expect(screen.getByText(/En pausa: Falta aprobar muestra/)).toBeTruthy()
  expect(screen.getByRole('button', { name: 'En pausa (1)' })).toBeTruthy()
  expect(screen.getByRole('button', { name: 'Sin responsable asignado (1)' })).toBeTruthy()
  expect(screen.queryByText(/600 pedidos/)).toBeNull()
})

async function renderOverview() {
  const pinia = createPinia()
  useSessionStore(pinia).principal = { id: 'owner', role: 'OWNER', displayName: 'Owner', email: '', workshop: { id: 'rr', name: 'R&R' }, mustChangePassword: false }
  const router = createRouter({ history: createMemoryHistory(), routes: ['/', '/app/orders', '/app/orders/new', '/app/clients', '/app/team', '/app/orders/:id'].map(path => ({ path, component: { template: '<main />' } })) })
  await router.push('/')
  return render(SessionHomeView, { global: { plugins: [pinia, router] } })
}

it.each(['getDashboard', 'list'] as const)('shows unknown state after %s fails and recovers on refresh', async endpoint => {
  vi.mocked(ordersApi[endpoint]).mockRejectedValueOnce(new Error('Sin conexión'))
  await renderOverview()
  await screen.findByText('Estado del taller no disponible')
  expect(screen.queryByText('TODO A TIEMPO')).toBeNull()
  expect(screen.queryByText('Aún no hay pedidos activos')).toBeNull()
  expect(screen.queryByText('No hay entregas programadas para hoy.')).toBeNull()
  expect(screen.queryByRole('heading', { name: 'Acciones de producción' })).toBeNull()
  await fireEvent.click(screen.getByRole('button', { name: 'Actualizar panel' }))
  expect(await screen.findByText('1 ATRASADA')).toBeTruthy()
  expect(screen.queryByText('Estado del taller no disponible')).toBeNull()
})

it('keeps order counts separate from mixed unit quantities', async () => {
  const base = { product: 'Prenda', client_name: 'Cliente', current_stage: 'MUESTRA', status: 'ACTIVE', traffic_light: 'ON_TIME', promised_at: '2026-12-01T18:00:00Z' }
  vi.mocked(ordersApi.list).mockResolvedValueOnce({ data: [
    { ...base, id: 'u', code: 'OT-U', quantity: 600, unit: 'UNITS' },
    { ...base, id: 'k', code: 'OT-K', quantity: 0.5, unit: 'KILOGRAMS' },
    { ...base, id: 'm', code: 'OT-M', quantity: 2, unit: 'METERS' },
  ] } as never)
  await renderOverview()
  const stage = await screen.findByRole('button', { name: /3 pedidos MUESTRA/ })
  expect(stage.textContent).toContain('600 unidades')
  expect(stage.textContent).toContain('0.5 kg')
  expect(stage.textContent).toContain('2 m')
  expect(stage.textContent).not.toContain('602.5')
})
