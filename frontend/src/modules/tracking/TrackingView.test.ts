import { render, screen } from '@testing-library/vue'
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
}

describe('TrackingView', () => {
  afterEach(() => vi.unstubAllGlobals())

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
})
