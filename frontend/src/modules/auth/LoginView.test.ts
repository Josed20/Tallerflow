import { fireEvent, render, screen } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it, vi } from 'vitest'
import LoginView from './LoginView.vue'
import { useSessionStore } from '../session/session.store'

describe('LoginView', () => {
  it('shows accessible validation messages before submitting incomplete credentials', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: LoginView },
        { path: '/forgot-password', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    await router.isReady()
    render(LoginView, { global: { plugins: [createPinia(), router] } })

    await fireEvent.click(screen.getByRole('button', { name: 'Iniciar sesión' }))

    expect(screen.getByText('Ingresa un correo válido.')).toBeTruthy()
    expect(screen.getByText('La contraseña es obligatoria.')).toBeTruthy()
  })

  it('connects labels, prevents duplicate submits, and announces a generic server error', async () => {
    const pinia = createPinia()
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: LoginView },
        { path: '/forgot-password', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    await router.isReady()
    const session = useSessionStore(pinia)
    let rejectLogin!: (reason: unknown) => void
    const pending = new Promise<void>((_, reject) => { rejectLogin = reject })
    const login = vi.spyOn(session, 'login').mockReturnValue(pending)
    render(LoginView, { global: { plugins: [pinia, router] } })

    await fireEvent.update(screen.getByLabelText('Correo electrónico'), 'owner@example.com')
    await fireEvent.update(screen.getByLabelText('Contraseña'), 'secret')
    const button = screen.getByRole('button', { name: 'Iniciar sesión' })
    const form = button.closest('form') as HTMLFormElement
    await fireEvent.submit(form)
    await fireEvent.submit(form)

    expect(login).toHaveBeenCalledTimes(1)
    expect((button as HTMLButtonElement).disabled).toBe(true)
    rejectLogin(new Error('server unavailable'))
    await pending.catch(() => undefined)
    const alert = await screen.findByRole('alert')
    expect(alert.getAttribute('aria-live')).toBe('assertive')
    expect(alert.textContent).toContain('No pudimos iniciar sesión')
  })
})
