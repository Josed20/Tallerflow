import { fireEvent, render, screen, waitFor } from '@testing-library/vue'
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

const member = {
  id: 'member-1',
  user_id: 'user-1',
  email: 'operator@example.test',
  display_name: 'Operador Demo',
  role: 'OPERATOR' as const,
  status: 'ACTIVE' as const,
}

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
  await screen.findByText('Operador Demo')
  return { router }
}

describe('TeamView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(teamApi.list).mockResolvedValue({ data: { members: [member], invitations: [invitation] } })
  })

  it('reveals a replacement link inside the invitation after the system confirmation', async () => {
    vi.mocked(teamApi.regenerateInvitation).mockResolvedValue({
      data: {
        invitation: { ...invitation, expires_at: '2026-10-07T12:00:00Z' },
        join_url: 'http://localhost:8080/join?token=new-token',
        token: 'new-token',
      },
    })
    await renderTeam()

    await fireEvent.click(screen.getByRole('tab', { name: /Invitaciones 1/ }))
    await fireEvent.click(await screen.findByRole('button', { name: 'Generar enlace nuevo' }))

    expect(screen.getByRole('dialog')).toBeTruthy()
    await fireEvent.click(screen.getByRole('button', { name: 'Generar enlace' }))

    expect((await screen.findByLabelText('Enlace nuevo listo para copiar') as HTMLInputElement).value).toBe('http://localhost:8080/join?token=new-token')
    expect(teamApi.regenerateInvitation).toHaveBeenCalledWith('invitation-1', 'csrf-token')
  })

  it('updates a member locally after changing access instead of reloading the directory', async () => {
    vi.mocked(teamApi.updateMember).mockResolvedValue({ data: { ...member, status: 'INACTIVE' } })
    await renderTeam()

    await fireEvent.click(screen.getByRole('button', { name: 'Desactivar' }))

    await waitFor(() => expect(screen.getByText('Desactivado')).toBeTruthy())
    expect(teamApi.list).toHaveBeenCalledTimes(1)
  })
})
