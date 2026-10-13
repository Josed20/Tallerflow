import { render, screen } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it } from 'vitest'
import AppShell from './AppShell.vue'
import { useSessionStore } from './session.store'

async function renderShell(role: 'OWNER' | 'OPERATOR') {
  const pinia = createPinia()
  const router = createRouter({
    history: createMemoryHistory(),
    routes: ['/app', '/app/orders', '/app/clients', '/app/orders/new', '/app/team'].map((path) => ({ path, component: { template: '<main />' } })),
  })
  await router.push('/app')
  await router.isReady()
  useSessionStore(pinia).principal = { id: 'user-1', email: 'user@example.test', displayName: 'User', role, workshop: { id: 'workshop-1', name: 'Taller' }, mustChangePassword: false }
  render(AppShell, { slots: { default: '<p>Contenido</p>' }, global: { plugins: [pinia, router] } })
}

describe('AppShell navigation', () => {
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
