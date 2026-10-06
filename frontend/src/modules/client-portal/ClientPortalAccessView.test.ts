import { fireEvent, render, screen } from '@testing-library/vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ClientPortalAccessView from './ClientPortalAccessView.vue'
import { clientPortalApi } from './api'

vi.mock('./api', () => ({ clientPortalApi: { requestAccess: vi.fn(), consumeAccessLink: vi.fn() } }))

describe('ClientPortalAccessView', () => {
  beforeEach(() => vi.clearAllMocks())

  it('confirms an access-link request without revealing whether the email exists', async () => {
    vi.mocked(clientPortalApi.requestAccess).mockResolvedValue(undefined)
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/portal/access', component: ClientPortalAccessView }] })
    await router.push('/portal/access')
    await router.isReady()
    render(ClientPortalAccessView, { global: { plugins: [router] } })

    await fireEvent.update(screen.getByLabelText(/correo electrónico/i), 'cliente@example.test')
    await fireEvent.click(screen.getByRole('button', { name: /enviar enlace/i }))

    expect(await screen.findByText(/si el correo está asociado/i)).toBeTruthy()
    expect(screen.queryByText(/cliente@example.test/)).toBeNull()
  })
})
