import { fireEvent, render, screen } from '@testing-library/vue'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ResetPasswordView from './ResetPasswordView.vue'

describe('ResetPasswordView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.unstubAllGlobals()
  })

  it('renders invalid state immediately when token query parameter is missing', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/reset-password', component: ResetPasswordView }],
    })
    await router.push('/reset-password')
    await router.isReady()

    render(ResetPasswordView, { global: { plugins: [router] } })

    const heading = screen.getByRole('heading', { level: 1, name: 'Enlace inválido' })
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(heading).toBeTruthy()
    expect(document.activeElement).toBe(heading)
    expect(screen.getByText('Solicitar un nuevo enlace')).toBeTruthy()
  })

  it('validates password requirements and confirmation match', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/reset-password', component: ResetPasswordView }],
    })
    await router.push('/reset-password?token=valid-token-sample-1234567890123456')
    await router.isReady()

    render(ResetPasswordView, { global: { plugins: [router] } })

    const submitBtn = screen.getByRole('button', { name: 'Restablecer contraseña' })
    await fireEvent.click(submitBtn)

    expect(screen.getByText('La nueva contraseña es obligatoria.')).toBeTruthy()

    const newPassInput = screen.getByLabelText('Nueva contraseña', { selector: 'input' })
    const confirmPassInput = screen.getByLabelText('Confirmar nueva contraseña', { selector: 'input' })

    // Less than 12 chars
    await fireEvent.update(newPassInput, 'short123')
    await fireEvent.update(confirmPassInput, 'short123')
    await fireEvent.click(submitBtn)
    expect(screen.getByText('Usa al menos 12 caracteres.')).toBeTruthy()

    // Passwords do not match
    await fireEvent.update(newPassInput, 'NewPassSecure2026!')
    await fireEvent.update(confirmPassInput, 'DifferentPass2026!')
    await fireEvent.click(submitBtn)
    expect(screen.getByText('Las contraseñas no coinciden.')).toBeTruthy()
  })

  it('successfully submits and displays success state', async () => {
    let callCount = 0
    const fetchMock = vi.fn().mockImplementation(async () => {
      callCount++
      return new Response(JSON.stringify({
        data: { status: 'PASSWORD_RESET_COMPLETED', message: 'Ok' },
      }), { status: 200, headers: { 'Content-Type': 'application/json' } })
    })
    vi.stubGlobal('fetch', fetchMock)

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/reset-password', component: ResetPasswordView }],
    })
    await router.push('/reset-password?token=valid-token-sample-1234567890123456')
    await router.isReady()

    render(ResetPasswordView, { global: { plugins: [router] } })

    const newPassInput = screen.getByLabelText('Nueva contraseña', { selector: 'input' })
    const confirmPassInput = screen.getByLabelText('Confirmar nueva contraseña', { selector: 'input' })
    await fireEvent.update(newPassInput, 'NewPassSecure2026!')
    await fireEvent.update(confirmPassInput, 'NewPassSecure2026!')

    const submitBtn = screen.getByRole('button', { name: 'Restablecer contraseña' })
    // Double click
    await Promise.all([fireEvent.click(submitBtn), fireEvent.click(submitBtn)])

    const successHeading = await screen.findByRole('heading', { level: 1, name: 'Contraseña restablecida' })
    expect(successHeading).toBeTruthy()
    expect(callCount).toBe(1)
    expect(screen.getByRole('button', { name: 'Iniciar sesión' })).toBeTruthy()
  })

  it.each([
    ['TOKEN_INVALID', 'Enlace inválido'],
    ['TOKEN_EXPIRED', 'Enlace expirado'],
    ['TOKEN_ALREADY_USED', 'Enlace ya utilizado'],
  ])('renders the %s recovery state returned by the API', async (code, heading) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code, message: 'Token rejected', details: {}, request_id: 'request-id' },
    }), { status: 400, headers: { 'Content-Type': 'application/json' } })))

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/reset-password', component: ResetPasswordView }],
    })
    await router.push('/reset-password?token=valid-token-sample-1234567890123456')
    await router.isReady()

    render(ResetPasswordView, { global: { plugins: [router] } })
    await fireEvent.update(screen.getByLabelText('Nueva contraseña', { selector: 'input' }), 'NewPassSecure2026!')
    await fireEvent.update(screen.getByLabelText('Confirmar nueva contraseña'), 'NewPassSecure2026!')
    await fireEvent.click(screen.getByRole('button', { name: 'Restablecer contraseña' }))

    expect(await screen.findByRole('heading', { level: 1, name: heading })).toBeTruthy()
  })

  it('keeps the form available after a recoverable API error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'INTERNAL_ERROR', message: 'Unexpected error', details: {}, request_id: 'request-id' },
    }), { status: 500, headers: { 'Content-Type': 'application/json' } })))

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/reset-password', component: ResetPasswordView }],
    })
    await router.push('/reset-password?token=valid-token-sample-1234567890123456')
    await router.isReady()

    render(ResetPasswordView, { global: { plugins: [router] } })
    await fireEvent.update(screen.getByLabelText('Nueva contraseña', { selector: 'input' }), 'NewPassSecure2026!')
    await fireEvent.update(screen.getByLabelText('Confirmar nueva contraseña'), 'NewPassSecure2026!')
    await fireEvent.click(screen.getByRole('button', { name: 'Restablecer contraseña' }))

    expect((await screen.findByRole('alert')).textContent).toContain('No pudimos actualizar la contraseña')
    expect(screen.getByRole('button', { name: 'Restablecer contraseña' })).toBeTruthy()
  })
})
