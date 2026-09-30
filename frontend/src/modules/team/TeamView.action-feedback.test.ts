import { fireEvent, render, screen, waitFor } from '@testing-library/vue'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import TeamView from './TeamView.vue'
import { useSessionStore } from '../session/session.store'
import { teamApi, type TeamMember } from './api'

vi.mock('./api', () => ({
  teamApi: {
    list: vi.fn(),
    invite: vi.fn(),
    regenerateInvitation: vi.fn(),
    cancelInvitation: vi.fn(),
    updateMember: vi.fn(),
  },
}))

const member: TeamMember = {
  id: 'member-1',
  user_id: 'user-1',
  email: 'operator@example.test',
  display_name: 'Operador Demo',
  role: 'OPERATOR',
  status: 'ACTIVE',
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
}

describe('TeamView action feedback', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(teamApi.list).mockResolvedValue({ data: { members: [member], invitations: [] } })
  })

  it('disables the member action while the access change is pending', async () => {
    let resolveUpdate: ((value: { data: typeof member }) => void) | undefined
    vi.mocked(teamApi.updateMember).mockImplementation(() => new Promise((resolve) => { resolveUpdate = resolve }))
    await renderTeam()

    const action = screen.getByRole('button', { name: 'Desactivar acceso' })
    await fireEvent.click(action)

    expect(action).toHaveProperty('disabled', true)
    expect(action.getAttribute('aria-busy')).toBe('true')

    resolveUpdate?.({ data: { ...member, status: 'INACTIVE' } })
    await waitFor(() => expect(screen.getByRole('button', { name: 'Activar acceso' })).toBeTruthy())
  })
})
