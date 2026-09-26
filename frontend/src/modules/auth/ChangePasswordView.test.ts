import { fireEvent, render, screen } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it, vi } from 'vitest'
import ChangePasswordView from './ChangePasswordView.vue'
import { useSessionStore } from '../session/session.store'

describe('ChangePasswordView', () => {
  it('associates policy help and disables duplicate password submissions', async () => {
    const pinia = createPinia()
    const router = createRouter({ history: createMemoryHistory(), routes: [
      { path: '/', component: ChangePasswordView },
      { path: '/app', component: { template: '<div />' } },
    ] })
    await router.push('/')
    await router.isReady()
    const session = useSessionStore(pinia)
    let resolveChange!: () => void
    const pending = new Promise<void>((resolve) => { resolveChange = resolve })
    const changePassword = vi.spyOn(session, 'changePassword').mockReturnValue(pending)
    render(ChangePasswordView, { global: { plugins: [pinia, router] } })

    const current = screen.getByLabelText('Contraseña actual')
    const next = screen.getByLabelText('Nueva contraseña')
    expect(next.getAttribute('aria-describedby')).toContain('new-password-description')
    expect(screen.getByText('Usa al menos 12 caracteres.')).toBeTruthy()
    await fireEvent.update(current, 'Temporary password 9!')
    await fireEvent.update(next, 'Replacement password 10!')
    const button = screen.getByRole('button', { name: 'Actualizar contraseña' })
    const form = button.closest('form') as HTMLFormElement
    await fireEvent.submit(form)
    await fireEvent.submit(form)

    expect(changePassword).toHaveBeenCalledTimes(1)
    expect((button as HTMLButtonElement).disabled).toBe(true)
    resolveChange()
    await pending
  })

  it('announces server failures through a live alert', async () => {
    const pinia = createPinia()
    const router = createRouter({ history: createMemoryHistory(), routes: [
      { path: '/', component: ChangePasswordView },
      { path: '/app', component: { template: '<div />' } },
    ] })
    await router.push('/')
    await router.isReady()
    vi.spyOn(useSessionStore(pinia), 'changePassword').mockRejectedValue(new Error('failure'))
    render(ChangePasswordView, { global: { plugins: [pinia, router] } })
    await fireEvent.update(screen.getByLabelText('Contraseña actual'), 'Temporary password 9!')
    await fireEvent.update(screen.getByLabelText('Nueva contraseña'), 'Replacement password 10!')

    await fireEvent.submit(screen.getByRole('button', { name: 'Actualizar contraseña' }).closest('form') as HTMLFormElement)

    const alert = await screen.findByRole('alert')
    expect(alert.getAttribute('aria-live')).toBe('assertive')
  })
})
