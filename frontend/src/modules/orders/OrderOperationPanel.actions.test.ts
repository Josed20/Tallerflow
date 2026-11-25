import { fireEvent, render, screen, within } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { expect, it, vi } from 'vitest'
import OrderOperationPanel from './OrderOperationPanel.vue'
import { productionApi } from './production-api'
import { teamApi } from '../team/api'
import { useSessionStore } from '../session/session.store'
import type { Order } from './api'
vi.mock('./production-api', () => ({ productionApi: { operation: vi.fn() } }))
vi.mock('../team/api', () => ({ teamApi: { list: vi.fn() } }))

it('changes the coordinator independently and preserves that person when pausing and resuming', async () => {
  const pinia = createPinia()
  const session = useSessionStore(pinia)
  session.csrfToken = 'csrf-local'
  session.principal = { id: 'owner', role: 'OWNER', displayName: 'Ana', email: 'ana@example.test', workshop: { id: 'w1', name: 'Demo' }, mustChangePassword: false }
  const order = { id: 'o1', version: 1, status: 'ACTIVE', responsible_user_id: null, block_reason: null, changes: [] } as unknown as Order
  vi.mocked(teamApi.list).mockResolvedValue({ data: { members: [{ user_id: 'worker', display_name: 'Luis', status: 'ACTIVE' }], invitations: [] } } as never)
  vi.mocked(productionApi.operation).mockResolvedValue({ data: order })
  const view = render(OrderOperationPanel, { props: { order }, global: { plugins: [pinia] } })
  await screen.findByRole('option', { name: 'Luis' })
  await fireEvent.update(screen.getByLabelText('Asignar a un miembro del equipo'), 'worker')
  await fireEvent.click(screen.getByRole('button', { name: 'Guardar responsable' }))
  await vi.waitFor(() => expect(productionApi.operation).toHaveBeenCalledWith('o1', { version: 1, responsible_user_id: 'worker', block_reason: '' }, 'csrf-local'))
  const assigned = { ...order, version: 2, responsible_user_id: 'worker' }
  await view.rerender({ order: assigned })
  await fireEvent.click(screen.getByRole('button', { name: 'Pausar pedido' }))
  const dialog = screen.getByRole('dialog')
  const reason = within(dialog).getByLabelText('Motivo de la pausa')
  await fireEvent.click(within(dialog).getByRole('button', { name: 'Confirmar pausa' }))
  expect(within(dialog).getByRole('alert').textContent).toContain('Indica el motivo')
  expect(document.activeElement).toBe(reason)
  expect(productionApi.operation).toHaveBeenCalledTimes(1)
  await fireEvent.update(reason, 'Falta aprobación del diseño')
  expect(within(dialog).queryByRole('alert')).toBeNull()
  await fireEvent.click(screen.getByRole('button', { name: 'Confirmar pausa' }))
  await vi.waitFor(() => expect(productionApi.operation).toHaveBeenLastCalledWith('o1', { version: 2, responsible_user_id: 'worker', block_reason: 'Falta aprobación del diseño' }, 'csrf-local'))
  await view.rerender({ order: { ...assigned, version: 3, block_reason: 'Falta aprobación del diseño' } })
  await fireEvent.click(screen.getByRole('button', { name: 'Reanudar producción' }))
  await fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Reanudar producción' }))
  await vi.waitFor(() => expect(productionApi.operation).toHaveBeenLastCalledWith('o1', { version: 3, responsible_user_id: 'worker', block_reason: '' }, 'csrf-local'))
})
