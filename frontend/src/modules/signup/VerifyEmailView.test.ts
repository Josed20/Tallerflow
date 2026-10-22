import { fireEvent, render, screen, waitFor } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import VerifyEmailView from './VerifyEmailView.vue'
import { signupApi } from './api'
import { ApiError } from '../session/api'
import { useSessionStore } from '../session/session.store'

const token = 'A'.repeat(43)
const session = { data: { expires_at: '2026-10-22T20:00:00Z', csrf_token: 'csrf', must_change_password: false } }

async function renderVerify(path: string, beforeMount?: (store: ReturnType<typeof useSessionStore>) => void) {
  const pinia = createPinia()
  const store = useSessionStore(pinia)
  beforeMount?.(store)
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/verificar-correo', component: VerifyEmailView },
      { path: '/app', component: { template: '<p>panel</p>' } },
      { path: '/login', component: { template: '<div />' } },
    ],
  })
  await router.push(path)
  await router.isReady()
  render(VerifyEmailView, { global: { plugins: [pinia, router] } })
  return { router, store }
}

afterEach(() => vi.restoreAllMocks())

describe('VerifyEmailView', () => {
  it('verifies the token, accepts the session and opens the dashboard', async () => {
    const verify = vi.spyOn(signupApi, 'verify').mockResolvedValue(session)
    let accept!: ReturnType<typeof vi.fn>
    const { router } = await renderVerify(`/verificar-correo?token=${token}`, (store) => {
      accept = vi.spyOn(store, 'acceptSession').mockResolvedValue() as unknown as ReturnType<typeof vi.fn>
    })

    await waitFor(() => expect(router.currentRoute.value.path).toBe('/app'))

    expect(verify).toHaveBeenCalledWith(token)
    expect(accept).toHaveBeenCalledWith(session)
    expect(window.location.search).not.toContain(token)
  })

  it('offers a new link when the token is expired or used', async () => {
    vi.spyOn(signupApi, 'verify').mockRejectedValue(new ApiError(422, 'VERIFICATION_TOKEN_INVALID'))
    const resend = vi.spyOn(signupApi, 'resend').mockResolvedValue({ data: {} })
    await renderVerify(`/verificar-correo?token=${token}`)

    expect(await screen.findByRole('heading', { name: 'Este enlace ya no sirve' })).toBeTruthy()
    await fireEvent.update(screen.getByLabelText('Correo con el que te registraste'), 'lucho@taller.pe')
    await fireEvent.click(screen.getByRole('button', { name: 'Enviarme un enlace nuevo' }))

    expect(resend).toHaveBeenCalledWith('lucho@taller.pe')
    expect(await screen.findByText(/te llegará un enlace nuevo/)).toBeTruthy()
  })

  it('does not call the API without a token', async () => {
    const verify = vi.spyOn(signupApi, 'verify')
    await renderVerify('/verificar-correo')

    expect(await screen.findByRole('heading', { name: 'Este enlace ya no sirve' })).toBeTruthy()
    expect(verify).not.toHaveBeenCalled()
  })
})
