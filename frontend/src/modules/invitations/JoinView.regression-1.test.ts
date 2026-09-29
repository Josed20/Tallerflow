import { fireEvent, render, screen, waitFor } from '@testing-library/vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import JoinView from './JoinView.vue'
import { teamApi } from '../team/api'

vi.mock('../team/api', () => ({
  teamApi: { consume: vi.fn() },
}))

describe('JoinView invitation completion', () => {
  beforeEach(() => vi.clearAllMocks())

  it('confirms the account email and carries it to login', async () => {
    // Regression: ISSUE-001 — successful registration did not tell the invited user how to continue.
    // Found by /qa on 2026-09-29
    // Report: .gstack/qa-reports/qa-report-localhost-2026-09-29.md
    vi.mocked(teamApi.consume).mockResolvedValue({
      data: { email: 'new.member@example.test', role: 'OPERATOR' },
    })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/join', component: JoinView },
        { path: '/login', component: { template: '<p>Login</p>' } },
      ],
    })
    await router.push('/join?token=fresh-token')
    await router.isReady()
    render(JoinView, { global: { plugins: [router] } })

    await fireEvent.update(screen.getByLabelText('Nombre'), 'Nuevo Miembro')
    await fireEvent.update(screen.getByLabelText('Clave'), 'secure-password')
    await fireEvent.click(screen.getByRole('button', { name: 'Aceptar invitación' }))

    expect(await screen.findByText('Cuenta creada correctamente')).toBeTruthy()
    expect(screen.getByText('new.member@example.test')).toBeTruthy()
    await fireEvent.click(screen.getByRole('button', { name: 'Iniciar sesión' }))
    await waitFor(() => {
      expect(router.currentRoute.value.fullPath).toBe('/login?email=new.member@example.test')
    })
  })

  it('offers login recovery when an invitation cannot be consumed', async () => {
    vi.mocked(teamApi.consume).mockRejectedValue(new Error('used token'))
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/join', component: JoinView },
        { path: '/login', component: { template: '<p>Login</p>' } },
      ],
    })
    await router.push('/join?token=used-token')
    await router.isReady()
    render(JoinView, { global: { plugins: [router] } })

    await fireEvent.update(screen.getByLabelText('Nombre'), 'Miembro Existente')
    await fireEvent.update(screen.getByLabelText('Clave'), 'secure-password')
    await fireEvent.click(screen.getByRole('button', { name: 'Aceptar invitación' }))

    expect(await screen.findByText('La invitación no está disponible o ya fue usada.')).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Ir a iniciar sesión' })).toBeTruthy()
    expect((screen.getByLabelText('Clave') as HTMLInputElement).value).toBe('')
  })
})
