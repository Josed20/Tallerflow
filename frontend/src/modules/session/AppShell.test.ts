import { render, screen, within } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it, vi } from 'vitest'
import { isNativeApp } from '../../native/platform'
import AppShell from './AppShell.vue'
import { useSessionStore } from './session.store'

vi.mock('../../native/platform', async (importOriginal) => ({
  ...await importOriginal<typeof import('../../native/platform')>(),
  isNativeApp: vi.fn(() => false),
}))

async function renderShell(role: 'OWNER' | 'OPERATOR') {
  const pinia = createPinia()
  const router = createRouter({
    history: createMemoryHistory(),
    routes: ['/app', '/app/orders', '/app/clients', '/app/orders/new', '/app/team', '/app/account'].map((path) => ({ path, component: { template: '<main />' } })),
  })
  await router.push('/app')
  await router.isReady()
  useSessionStore(pinia).principal = { id: 'user-1', email: 'user@example.test', displayName: 'User', role, workshop: { id: 'workshop-1', name: 'Taller' }, mustChangePassword: false }
  render(AppShell, { slots: { default: '<p>Contenido</p>' }, global: { plugins: [pinia, router] } })
}

describe('AppShell navigation', () => {
  it('keeps Cuenta in the Android navigation without a separate options menu', async () => {
    vi.mocked(isNativeApp).mockReturnValueOnce(true)
    await renderShell('OWNER')
    const navigation = screen.getByRole('navigation', { name: 'Navegación de Android' })
    expect(within(navigation).getByRole('link', { name: 'Cuenta' }).getAttribute('href')).toBe('/app/account')
    expect(screen.queryByRole('button', { name: 'Más opciones' })).toBeNull()
  })

  it('shows management navigation to an owner', async () => {
    await renderShell('OWNER')
    expect(screen.getByRole('link', { name: 'Panel' })).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Clientes' })).toBeTruthy()
  })

  it('shows only the orders section to an operator', async () => {
    await renderShell('OPERATOR')
    const sections = screen.getByRole('navigation', { name: 'Secciones internas' })
    expect([...sections.querySelectorAll('a')].map((link) => link.textContent?.trim())).toEqual(['Órdenes'])
  })
})
