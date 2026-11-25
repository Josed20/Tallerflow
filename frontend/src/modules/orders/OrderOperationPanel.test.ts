import { render, screen } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { expect, it, vi } from 'vitest'
import OrderOperationPanel from './OrderOperationPanel.vue'
import { useSessionStore } from '../session/session.store'
import { teamApi } from '../team/api'
import type { Order } from './api'
vi.mock('../team/api', () => ({ teamApi: { list: vi.fn() } }))

it('shows scoped responsible and actor names to operators without fetching the team', () => {
  const pinia = createPinia()
  useSessionStore(pinia).principal = { id: 'operator', role: 'OPERATOR', displayName: 'Operador', email: '', workshop: { id: 'rr', name: 'R&R' }, mustChangePassword: false }
  render(OrderOperationPanel, { props: { order: {
    id: 'order', responsible_user_id: 'hidden-user-id', responsible_name: 'Ana Pérez', block_reason: 'Falta tela', status: 'ACTIVE',
    workshop_id: 'rr', client_id: 'client', client_name: 'Cliente R&R', order_number: 1,
    code: 'OT-000001', product: 'Polos', specification: '', quantity: 600, unit: 'UNITS',
    promised_at: '2026-12-01T18:00:00Z', version: 1, traffic_light: 'ON_TIME',
    created_at: '2026-11-24T12:00:00Z', updated_at: '2026-11-24T12:00:00Z',
    changes: [{ id: 'change', actor_user_id: 'hidden-actor-id', actor_name: 'Luis Torres', created_at: '2026-01-01T12:00:00Z', changes: { responsible_user_id: { from: null, to: 'hidden-user-id' } } }],
  } satisfies Order }, global: { plugins: [pinia] } })
  expect(screen.getByText('Responsable: Ana Pérez')).toBeTruthy()
  expect(screen.getByText(/Luis Torres/)).toBeTruthy()
  expect(screen.queryByText(/hidden-user-id|hidden-actor-id/)).toBeNull()
  expect(teamApi.list).not.toHaveBeenCalled()
  expect(screen.queryByRole('button', { name: /guardar/i })).toBeNull()
})
