import { afterEach, describe, expect, it, vi } from 'vitest'
import { manageTrackingLink, fetchTrackingOrder, TrackingNotFoundError } from './api'

describe('tracking API', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('uses authenticated CSRF-protected requests for recovery and explicit management', async () => {
    const fetchMock = vi.fn().mockImplementation(async () => Response.json({ data: { token: 'T'.repeat(43) } }))
    vi.stubGlobal('fetch', fetchMock)
    for (const action of ['recover', 'replace', 'revoke'] as const) {
      await manageTrackingLink('order/id', 'csrf', action)
      expect(fetchMock).toHaveBeenLastCalledWith(`/api/v1/orders/order%2Fid/tracking-link${action === 'recover' ? '' : '/' + action}`, expect.objectContaining({
        method: 'POST', credentials: 'include', headers: expect.objectContaining({ 'X-CSRF-Token': 'csrf' }),
      }))
    }
  })

  it('keeps public reads anonymous and maps expired links to the inactive state', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response('{}', { status: 404 }))
    vi.stubGlobal('fetch', fetchMock)
    await expect(fetchTrackingOrder('expired')).rejects.toBeInstanceOf(TrackingNotFoundError)
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/tracking/expired', expect.objectContaining({ credentials: 'omit' }))
  })
})
