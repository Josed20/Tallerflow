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
    routes: ['/app', '/app/orders', '/app/production', '/app/clients', '/app/orders/new', '/app/team'].map((path) => ({ path, component: { template: '<main />' } })),
  })
  await router.push('/app')
  await router.isReady()
  useSessionStore(pinia).principal = { id: 'user-1', email: 'user@example.test', displayName: 'User', role, workshop: { id: 'workshop-1', name: 'Taller' }, mustChangePassword: false }
  render(AppShell, { slots: { default: '<p>Contenido</p>' }, global: { plugins: [pinia, router] } })
}

describe('AppShell navigation', () => {
  it('shows management navigation to an owner but not the operator production queue', async () => {
    await renderShell('OWNER')
    expect(screen.getByRole('link', { name: 'Clientes' })).toBeTruthy()
    expect(screen.queryByRole('link', { name: 'Producción' })).toBeNull()
  })

  it('shows the production queue to an operator without management controls', async () => {
    await renderShell('OPERATOR')
    expect(screen.getByRole('link', { name: 'Producción' })).toBeTruthy()
    expect(screen.queryByRole('link', { name: 'Clientes' })).toBeNull()
    expect(screen.queryByRole('link', { name: 'Nueva orden' })).toBeNull()
  })
})
