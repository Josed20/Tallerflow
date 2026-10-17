import { fireEvent, render, screen, waitFor } from '@testing-library/vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import TrackingView from './TrackingView.vue'

const order = {
  workshop_name: 'Confecciones Gamarra', code: 'OT-000003', product: 'Polo jersey', quantity: 50, unit: 'DOZENS',
  status: 'ACTIVE', traffic_light: 'ON_TIME', promised_at: '2026-10-23T22:00:00Z', updated_at: '2026-10-13T15:00:00Z',
  stages: [{ position: 1, name: 'CORTE', status: 'COMPLETED', completed_at: '2026-10-12T15:00:00Z' }],
  photos: [{ stage_position: 1, photo_data: 'data:image/jpeg;base64,AAAA', caption: '', created_at: '2026-10-12T15:00:00Z' }],
}

describe('TrackingView evidence photos', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('opens a photo from the keyboard, focuses Cerrar and closes with Escape', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: order }), { status: 200 })))
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/seguimiento/:token', component: TrackingView }] })
    await router.push('/seguimiento/' + 'A'.repeat(43))
    await router.isReady()
    render(TrackingView, { global: { plugins: [router] } })

    const thumbnail = await screen.findByRole('button', { name: /ver foto de corte/i })
    thumbnail.focus()
    await fireEvent.click(thumbnail)

    const close = await screen.findByRole('button', { name: 'Cerrar' })
    expect(screen.getByRole('dialog', { name: 'Foto de evidencia' })).toBeTruthy()
    await waitFor(() => expect(document.activeElement).toBe(close))

    await fireEvent.keyDown(window, { key: 'Escape' })
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(document.activeElement).toBe(thumbnail)
  })
})
