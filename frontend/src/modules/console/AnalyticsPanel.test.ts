import { fireEvent, render, screen } from '@testing-library/vue'
import { beforeEach, expect, it, vi } from 'vitest'
import ConsoleView from './ConsoleView.vue'
import AnalyticsPanel from './AnalyticsPanel.vue'

const api = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))
vi.mock('../session/api', () => ({ api }))
const day = (date: string, opens: number | null, registrations: number | null) => ({
  day: `${date}T00:00:00Z`, first_opens: opens, registrations, logins: 1, activities: 1,
  active_users: 1, active_workshops: 1, play_device_installs: null, play_user_installs: null,
  play_uninstalls: null, play_imported_at: null, coverage_started_at: '2026-01-01T00:00:00Z',
})
const result = (complete = true) => ({
  data: [day('2026-01-01', 2, 1), day('2026-01-02', 3, 2)],
  totals: { active_users: complete ? 1 : null, active_workshops: complete ? 1 : null, coverage_complete: complete },
})
beforeEach(() => { vi.resetAllMocks(); sessionStorage.clear() })

it('preserves all chosen filters when including demos and reclassifying a workshop through the console', async () => {
  const workshop = { id: 'workshop-1', name: 'Taller Demo', owner_email: 'owner@example.test', created_at: '2026-01-01T00:00:00Z', last_activity: null, members: 1, active_orders: 0, photo_bytes: 0, photo_count: 0, r2_photo_count: 0, is_demo: false, status: 'ACTIVE', deletion_scheduled_for: null }
  api.get.mockImplementation(async (path: string) => {
    if (path.startsWith('/api/v1/console/analytics?')) return result()
    if (path.startsWith('/api/v1/console/workshops?')) return { data: [workshop] }
    if (path === '/api/v1/console/audit') return { data: [] }
    return { data: {} }
  })
  api.post.mockResolvedValue(undefined)
  render(ConsoleView)
  await screen.findByRole('button', { name: 'Aplicar filtros' })
  await fireEvent.update(screen.getByLabelText('Desde'), '2026-01-01')
  await fireEvent.update(screen.getByLabelText('Hasta'), '2026-01-02')
  await fireEvent.update(screen.getByLabelText('Canal'), 'android')
  await fireEvent.update(screen.getByLabelText('Versión'), '3')
  await fireEvent.click(screen.getByRole('button', { name: 'Aplicar filtros' }))
  await vi.waitFor(() => expect(api.get).toHaveBeenCalledWith('/api/v1/console/analytics?start=2026-01-01&end=2026-01-02&channel=android&version=3&include_demo=false'))
  await screen.findByRole('button', { name: 'Aplicar filtros' })
  const previousRequests = api.get.mock.calls.filter(([path]) => path.startsWith('/api/v1/console/analytics?')).length
  await fireEvent.click(screen.getByRole('checkbox', { name: /Incluir talleres demo/ }))
  const expected = '/api/v1/console/analytics?start=2026-01-01&end=2026-01-02&channel=android&version=3&include_demo=true'
  await vi.waitFor(() => expect(api.get).toHaveBeenCalledWith(expected))
  await fireEvent.click(screen.getByRole('button', { name: 'Marcar demo' }))
  await vi.waitFor(() => expect(api.get.mock.calls.filter(([path]) => path === expected)).toHaveLength(2))
  expect(api.post).toHaveBeenCalledWith('/api/v1/console/workshops/workshop-1/demo', { is_demo: true }, '')
  expect((screen.getByLabelText('Desde') as HTMLInputElement).value).toBe('2026-01-01')
  expect((screen.getByLabelText('Hasta') as HTMLInputElement).value).toBe('2026-01-02')
  expect((screen.getByLabelText('Canal') as HTMLSelectElement).value).toBe('android')
  expect((screen.getByLabelText('Versión') as HTMLInputElement).value).toBe('3')
  expect(api.get.mock.calls.filter(([path]) => path.startsWith('/api/v1/console/analytics?')).length).toBe(previousRequests + 2)
})

it('sums observed events only with complete coverage and keeps server period distinct counts', async () => {
  api.get.mockResolvedValue(result())
  const view = render(AnalyticsPanel, { props: { csrf: 'csrf', includeDemo: false, revision: 0 } })
  const total = (label: string) => screen.getByText(label).closest('article')!.querySelector('strong')!
  await screen.findByText('Usuarios únicos activos en todo el período')
  expect(total('Primeras aperturas observadas en el período').textContent).toBe('5')
  expect(total('Registros verificados en el período').textContent).toBe('3')
  expect(total('Usuarios únicos activos en todo el período').textContent).toBe('1')
  api.get.mockResolvedValue(result(false))
  await view.rerender({ revision: 1 })
  await vi.waitFor(() => expect(total('Primeras aperturas observadas en el período').textContent).toBe('Sin datos'))
  expect(total('Registros verificados en el período').textContent).toBe('Sin datos')
  api.get.mockResolvedValue({ ...result(), data: [day('2026-01-01', null, null)] })
  await view.rerender({ revision: 2 })
  await vi.waitFor(() => expect(api.get).toHaveBeenCalledTimes(3))
  expect(total('Primeras aperturas observadas en el período').textContent).toBe('Sin datos')
})
