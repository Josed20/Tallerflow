import { fireEvent, render, screen, within } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createAppRouter } from '../../app/router'
import { useSessionStore } from '../session/session.store'
import { ANALYTICS_DISABLED_KEY, ANALYTICS_KEY, ANALYTICS_PREFERENCE_EVENT } from '../../native/analytics'
import * as platform from '../../native/platform'
import AccountView from './AccountView.vue'
import AppShell from '../session/AppShell.vue'
async function setup(role: 'OWNER' | 'ADMIN' | 'SUPERVISOR' | 'OPERATOR' = 'OWNER', authenticated = true) {
  const pinia = createPinia()
  const session = useSessionStore(pinia)
  session.restored = true
  session.csrfToken = authenticated ? 'csrf' : null
  session.principal = authenticated ? { id: 'u1', displayName: 'Ana', email: 'ana@example.test', role, workshop: { id: 'w1', name: 'Taller Demo' }, mustChangePassword: false } : null
  const router = createAppRouter({ pinia, history: createMemoryHistory() })
  await router.push('/app/account')
  await router.isReady()
  return { pinia, router }
}
beforeEach(() => { localStorage.clear() })
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); localStorage.clear() })

describe('account access and privacy navigation', () => {
  beforeEach(() => localStorage.clear())
  it.each(['OWNER', 'ADMIN', 'SUPERVISOR', 'OPERATOR'] as const)('opens account and public deletion for %s', async role => {
    const { pinia, router } = await setup(role)
    expect(router.currentRoute.value.name).toBe('account')
    expect(document.title).toBe('Mi cuenta · TallerFlow')
    render(AccountView, { global: { plugins: [pinia, router] } })
    expect(screen.getByText('ana@example.test')).toBeTruthy()
    expect(screen.getByText({ OWNER: 'Propietario', ADMIN: 'Administrador', SUPERVISOR: 'Supervisor', OPERATOR: 'Operador' }[role])).toBeTruthy()
    await fireEvent.click(screen.getByRole('link', { name: /Solicitar eliminación de cuenta/ }))
    await vi.waitFor(() => expect(router.currentRoute.value.name).toBe('account-delete'))
    expect(screen.queryByRole('button', { name: 'Cerrar información' })).toBeNull()
  })
  it.each(['OWNER', 'OPERATOR'] as const)('shows Android account actions for %s', async role => {
    vi.spyOn(platform, 'isNativeApp').mockReturnValue(true)
    const { pinia, router } = await setup(role)
    const logout = vi.spyOn(useSessionStore(pinia), 'logout').mockImplementation(async () => {
      useSessionStore(pinia).clearSession()
    })
    render(AccountView, { global: { plugins: [pinia, router] } })
    for (const name of ['Equipo', 'Ajustes del taller']) {
      expect(Boolean(screen.queryByRole('link', { name }))).toBe(role === 'OWNER')
    }
    await fireEvent.click(screen.getByRole('button', { name: 'Cerrar sesión' }))
    expect(logout).toHaveBeenCalledOnce()
    await vi.waitFor(() => expect(router.currentRoute.value.name).toBe('login'))
  })
  it('uses a short readable fallback for an unknown role', async () => {
    const { pinia, router } = await setup()
    useSessionStore(pinia).principal!.role = 'UNKNOWN_ROLE'
    render(AccountView, { global: { plugins: [pinia, router] } })
    expect(screen.getByText('Miembro')).toBeTruthy()
    expect(screen.queryByText('UNKNOWN_ROLE')).toBeNull()
  })
  it('requires authentication and keeps public instructions', async () => {
    const { router } = await setup('OWNER', false)
    expect(router.currentRoute.value.name).toBe('login')
    expect(router.currentRoute.value.query.redirect).toBe('/app/account')
    await router.push('/eliminar-cuenta')
    expect(router.currentRoute.value.name).toBe('delete-account')
  })
  it('puts account in the user disclosure and removes the global deletion footer', async () => {
    const { pinia, router } = await setup('OPERATOR')
    const { container } = render(AppShell, { slots: { default: '<p>Contenido</p>' }, global: { plugins: [pinia, router] } })
    expect(container.querySelector('.account-help')).toBeNull()
    expect(screen.getByRole('link', { name: 'Mi cuenta' }).getAttribute('href')).toBe('/app/account')
    expect(container.querySelector('.user-menu')).toBeNull()
  })
  it('turns analytics off immediately through the runtime setter and can enable it again', async () => {
    localStorage.setItem(ANALYTICS_KEY, '{"queue":["pending"]}')
    const { pinia, router } = await setup()
    render(AccountView, { global: { plugins: [pinia, router] } })
    const checkbox = screen.getByRole('checkbox', { name: /compartir estadísticas/i }) as HTMLInputElement
    expect(checkbox.checked).toBe(true)
    const preference = vi.fn()
    window.addEventListener(ANALYTICS_PREFERENCE_EVENT, preference)
    try {
      await fireEvent.click(checkbox)
      expect(checkbox.checked).toBe(false)
      expect(localStorage.getItem(ANALYTICS_DISABLED_KEY)).toBe('true')
      expect(localStorage.getItem(ANALYTICS_KEY)).toBeNull()
      expect(preference).toHaveBeenCalledOnce()
      expect((preference.mock.calls[0]?.[0] as CustomEvent).detail.disabled).toBe(true)
      expect(screen.getByRole('status').textContent).toMatch(/se detuvo el envío/i)
      await fireEvent.click(checkbox)
      expect(checkbox.checked).toBe(true)
      expect(localStorage.getItem(ANALYTICS_DISABLED_KEY)).toBeNull()
    } finally { window.removeEventListener(ANALYTICS_PREFERENCE_EVENT, preference) }
  })
  it('loads the saved opt-out', async () => {
    localStorage.setItem(ANALYTICS_DISABLED_KEY, 'true')
    const { pinia, router } = await setup()
    render(AccountView, { global: { plugins: [pinia, router] } })
    expect((screen.getByRole('checkbox') as HTMLInputElement).checked).toBe(false)
  })
  it.each([{ doNotTrack: '1' }, { globalPrivacyControl: true }])('respects browser privacy signals %j', async signal => {
    vi.stubGlobal('navigator', { ...navigator, ...signal })
    const { pinia, router } = await setup()
    render(AccountView, { global: { plugins: [pinia, router] } })
    const checkbox = screen.getByRole('checkbox') as HTMLInputElement
    expect(checkbox.checked).toBe(false)
    expect(checkbox.disabled).toBe(true)
    expect(screen.getByText(/tu navegador solicita/i)).toBeTruthy()
  })
  it('announces storage failure and restores the checkbox', async () => {
    const { pinia, router } = await setup()
    render(AccountView, { global: { plugins: [pinia, router] } })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('Storage unavailable') })
    const checkbox = screen.getByRole('checkbox') as HTMLInputElement
    await fireEvent.click(checkbox)
    expect(checkbox.checked).toBe(true)
    expect(screen.getByRole('alert').textContent).toMatch(/no se pudo guardar/i)
  })

})

it('persists optional metrics opt-out and deletes only telemetry data', async () => {
  localStorage.clear()
  localStorage.setItem('unrelated-preference', 'keep')
  localStorage.setItem('tallerflow.analytics.v1', JSON.stringify({ installation: 'random', opened: true, queue: [] }))
  const { pinia, router } = await setup()
  render(AccountView, { global: { plugins: [pinia, router] } })
  const checkbox = screen.getByRole('checkbox', { name: 'Compartir estadísticas de uso en este dispositivo' }) as HTMLInputElement
  expect(checkbox.checked).toBe(true)
  const listener = vi.fn()
  window.addEventListener('tallerflow:analytics-preference', listener)
  await fireEvent.click(checkbox)
  expect(localStorage.getItem('tallerflow.analytics.disabled')).toBe('true')
  expect(localStorage.getItem('tallerflow.analytics.v1')).toBeNull()
  expect(localStorage.getItem('unrelated-preference')).toBe('keep')
  expect(checkbox.checked).toBe(false)
  expect(listener).toHaveBeenCalledTimes(1)
  expect(screen.getByRole('status').textContent).toContain('Estadísticas de uso desactivadas')
  await fireEvent.click(checkbox)
  expect(localStorage.getItem('tallerflow.analytics.disabled')).toBeNull()
  expect(checkbox.checked).toBe(true)
  window.removeEventListener('tallerflow:analytics-preference', listener)
})
