import { fireEvent, render, screen } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import TeamView from './TeamView.vue'
import { useSessionStore } from '../session/session.store'
import { teamApi } from './api'

vi.mock('./api', () => ({
  teamApi: {
    list: vi.fn(),
    invite: vi.fn(),
    regenerateInvitation: vi.fn(),
    cancelInvitation: vi.fn(),
    updateMember: vi.fn(),
  },
}))

const invitation = {
  id: 'invitation-1',
  email: 'pending@example.test',
  role: 'OPERATOR' as const,
  expires_at: '2026-10-06T12:00:00Z',
}

async function renderTeam() {
  const pinia = createPinia()
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/app', component: { template: '<main />' } },
      { path: '/app/team', component: TeamView },
    ],
  })
  await router.push('/app/team')
  await router.isReady()
  const session = useSessionStore(pinia)
  session.csrfToken = 'csrf-token'
  session.principal = {
    id: 'owner-1',
    email: 'owner@example.test',
    displayName: 'Owner Demo',
    role: 'OWNER',
    workshop: { id: 'workshop-1', name: 'Taller Demo' },
    mustChangePassword: false,
  }
  render(TeamView, { global: { plugins: [pinia, router] } })
  await screen.findByRole('tab', { name: /Invitaciones 1/ })
}

describe('TeamView invitation link visibility', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(teamApi.list).mockResolvedValue({ data: { members: [], invitations: [invitation] } })
    vi.mocked(teamApi.regenerateInvitation).mockResolvedValue({
      data: {
        invitation,
        join_url: 'http://localhost:8080/join?token=new-token',
        token: 'new-token',
      },
    })
  })

  it('can reveal a generated link again after hiding it during the same visit', async () => {
    await renderTeam()
    await fireEvent.click(screen.getByRole('tab', { name: /Invitaciones 1/ }))
    await fireEvent.click(await screen.findByRole('button', { name: 'Generar enlace nuevo' }))
    await fireEvent.click(screen.getByRole('button', { name: 'Generar enlace' }))

    const link = await screen.findByLabelText('Enlace nuevo listo para copiar') as HTMLInputElement
    expect(link.value).toContain('new-token')

    await fireEvent.click(screen.getByRole('button', { name: 'Ocultar enlace' }))
    expect(screen.queryByLabelText('Enlace nuevo listo para copiar')).toBeNull()
    expect(screen.getByText('Enlace nuevo oculto')).toBeTruthy()

    await fireEvent.click(screen.getByRole('button', { name: 'Ver enlace' }))
    expect((screen.getByLabelText('Enlace nuevo listo para copiar') as HTMLInputElement).value).toContain('new-token')
  })
})
