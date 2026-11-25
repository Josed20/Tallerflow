import { fireEvent, render, screen, waitFor } from '@testing-library/vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import TrackingView from './TrackingView.vue'

const order = {
  workshop_name: 'Confecciones Gamarra',
  code: 'OT-000003',
  product: 'Polo jersey',
  quantity: 50,
  unit: 'DOZENS',
  status: 'ACTIVE',
  traffic_light: 'ON_TIME',
  promised_at: '2026-10-23T22:00:00Z',
  updated_at: '2026-10-13T15:00:00Z',
  stages: [
    { position: 1, name: 'Corte', status: 'COMPLETED', completed_at: '2026-10-12T15:00:00Z' },
    { position: 2, name: 'Costura', status: 'IN_PROGRESS', completed_at: null },
    { position: 3, name: 'Acabado', status: 'PENDING', completed_at: null },
  ],
  photos: [],
}

async function renderTracking(response: Response) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response))
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/seguimiento/:token', component: TrackingView }] })
  await router.push('/seguimiento/' + 'A'.repeat(43))
  await router.isReady()
  render(TrackingView, { global: { plugins: [router] } })
  return router
}

describe('TrackingView', () => {
  afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers() })

  it('shows the current stage and progress to the customer', async () => {
    await renderTracking(new Response(JSON.stringify({ data: order }), { status: 200 }))

    expect(await screen.findByRole('heading', { name: 'Polo jersey' })).toBeTruthy()
    expect(screen.getByText('Confecciones Gamarra')).toBeTruthy()
    expect(screen.getByText('En proceso ahora')).toBeTruthy()
    expect(screen.getByText('1 de 3 etapas completadas')).toBeTruthy()
    expect(vi.mocked(fetch).mock.calls[0][0]).toBe('/api/v1/tracking/' + 'A'.repeat(43))
  })

  it('explains when the link was replaced or is invalid', async () => {
    await renderTracking(new Response('{}', { status: 404 }))

    expect(await screen.findByRole('heading', { name: 'Este enlace ya no está activo' })).toBeTruthy()
  })

  it('shows the specification, custom stage description and actual stage count', async () => {
    const custom = { ...order, specification: 'Tela azul\nCuello redondo', stages: [
      { position: 1, name: 'Revisión del lote', client_description: 'Verificamos las medidas acordadas.', status: 'IN_PROGRESS', completed_at: null },
    ] }
    await renderTracking(Response.json({ data: custom }))
    expect(await screen.findByRole('heading', { name: 'Descripción del pedido' })).toBeTruthy()
    expect(screen.getByText(/Tela azul/)).toBeTruthy()
    expect(screen.getAllByText('Verificamos las medidas acordadas.')).toHaveLength(2)
    expect(screen.getByText('0 de 1 etapas completadas')).toBeTruthy()
  })

  it('explains finishing and does not claim a completed order was delivered', async () => {
    await renderTracking(Response.json({ data: { ...order, status: 'COMPLETED', traffic_light: 'COMPLETED' } }))
    expect(await screen.findByText(/Coordina la entrega/)).toBeTruthy()
    expect(screen.queryByText('Entregado')).toBeNull()
    expect(screen.getByText(/Se retiran hilos sobrantes/)).toBeTruthy()
  })

  it('keeps the order during refresh failures and gives loading and success feedback', async () => {
    await renderTracking(Response.json({ data: order }))
    await screen.findByRole('heading', { name: order.product })
    let resolve!: (response: Response) => void
    vi.mocked(fetch).mockImplementationOnce(() => new Promise<Response>((done) => { resolve = done }))
    await fireEvent.click(screen.getByRole('button', { name: 'Actualizar' }))
    expect((screen.getByRole('button', { name: 'Actualizando…' }) as HTMLButtonElement).disabled).toBe(true)
    resolve(Response.json({ data: order }))
    await screen.findByRole('button', { name: 'Actualizar' })
    expect(screen.getByRole('status').textContent).toContain('Estado actualizado.')
    vi.mocked(fetch).mockRejectedValueOnce(new Error('offline'))
    await fireEvent.click(screen.getByRole('button', { name: 'Actualizar' }))
    await screen.findByText(/No pudimos actualizar el estado/)
    expect(screen.getByRole('heading', { name: order.product })).toBeTruthy()
  })

  it('automatically refreshes every minute only while visible and stops on invalidation', async () => {
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] })
    await renderTracking(Response.json({ data: order }))
    await screen.findByRole('heading', { name: order.product })
    expect(screen.getByText(/automáticamente cada minuto/)).toBeTruthy()
    vi.mocked(fetch).mockImplementation(async () => Response.json({ data: order }))
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'hidden' })
    await vi.advanceTimersByTimeAsync(60_000)
    expect(fetch).toHaveBeenCalledTimes(1)
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
    await vi.advanceTimersByTimeAsync(60_000)
    expect(fetch).toHaveBeenCalledTimes(2)
    vi.mocked(fetch).mockResolvedValueOnce(new Response('{}', { status: 404 }))
    await vi.advanceTimersByTimeAsync(60_000)
    await screen.findByRole('heading', { name: 'Este enlace ya no está activo' })
    await vi.advanceTimersByTimeAsync(120_000)
    expect(fetch).toHaveBeenCalledTimes(3)
  })

  it('reloads when navigating to a different tracking token', async () => {
    const router = await renderTracking(Response.json({ data: order }))
    await screen.findByRole('heading', { name: order.product })
    vi.mocked(fetch).mockResolvedValueOnce(Response.json({ data: { ...order, product: 'Otro pedido' } }))
    await router.push('/seguimiento/' + 'B'.repeat(43))
    await screen.findByRole('heading', { name: 'Otro pedido' })
    await waitFor(() => expect(fetch).toHaveBeenLastCalledWith('/api/v1/tracking/' + 'B'.repeat(43), expect.any(Object)))
  })

  it('handles an order without registered stages', async () => {
    await renderTracking(Response.json({ data: { ...order, stages: [] } }))
    expect(await screen.findByText('El taller todavía no registró etapas para este pedido.')).toBeTruthy()
    expect(screen.getByText('0 de 0 etapas completadas')).toBeTruthy()
  })
})
