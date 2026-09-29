import { fireEvent, render, screen } from '@testing-library/vue'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ForgotPasswordView from './ForgotPasswordView.vue'

describe('ForgotPasswordView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.unstubAllGlobals()
  })

  it('validates email before submitting', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/forgot-password', component: ForgotPasswordView },
        { path: '/login', component: { template: '<div />' } },
      ],
    })
    await router.push('/forgot-password')
    await router.isReady()

    render(ForgotPasswordView, { global: { plugins: [router] } })

    const submitBtn = screen.getByRole('button', { name: 'Enviar enlace de recuperación' })
    await fireEvent.click(submitBtn)

    expect(screen.getByText('El correo electrónico es obligatorio.')).toBeTruthy()

    const emailInput = screen.getByLabelText('Correo electrónico')
    expect(document.activeElement).toBe(emailInput)
    await fireEvent.update(emailInput, 'invalid-email')
    await fireEvent.click(submitBtn)

    expect(screen.getByText('Ingresa un correo válido.')).toBeTruthy()
  })

  it('shows confirmation state upon successful submission and prevents duplicate clicks', async () => {
    let callCount = 0
    const fetchMock = vi.fn().mockImplementation(async () => {
      callCount++
      return new Response(JSON.stringify({
        data: { status: 'REQUESTED', message: 'Enlace enviado.' },
      }), { status: 200, headers: { 'Content-Type': 'application/json' } })
    })
    vi.stubGlobal('fetch', fetchMock)

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/forgot-password', component: ForgotPasswordView },
        { path: '/login', component: { template: '<div />' } },
      ],
    })
    await router.push('/forgot-password')
    await router.isReady()

    render(ForgotPasswordView, { global: { plugins: [router] } })

    const emailInput = screen.getByLabelText('Correo electrónico')
    await fireEvent.update(emailInput, 'user@tallerflow.pe')

    const submitBtn = screen.getByRole('button', { name: 'Enviar enlace de recuperación' })
    // Double click
    await Promise.all([fireEvent.click(submitBtn), fireEvent.click(submitBtn)])

    // Wait for the async state to update
    const sentHeading = await screen.findByRole('heading', { level: 1, name: 'Revisa tu correo' })
    expect(sentHeading).toBeTruthy()

    // Should only have made 1 fetch call due to double click prevention
    expect(callCount).toBe(1)
  })
})
