import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { api } from './api'
import type { ApiPrincipal, AuthSessionPayload, SessionPrincipal, SessionStatus, WorkshopAccess } from './types'

export const useSessionStore = defineStore('session', () => {
  const principal = ref<SessionPrincipal | null>(null)
  const status = ref<SessionStatus>('idle')
  const restored = ref(false)
  const csrfToken = ref<string | null>(null)
  const mustChangePassword = ref(false)

  const isAuthenticated = computed(() => csrfToken.value !== null)
  const requiresPasswordChange = computed(() => mustChangePassword.value)

  async function restore() {
    if (restored.value) return
    status.value = 'restoring'
    try {
      const session = await api.get<AuthSessionPayload>('/api/v1/auth/session')
      applySession(session)
      if (mustChangePassword.value) {
        principal.value = null
        status.value = 'authenticated'
        return
      }
      const [apiPrincipal, access] = await Promise.all([
        api.get<ApiPrincipal>('/api/v1/me'),
        api.get<WorkshopAccess>('/api/v1/workshops/current'),
      ])
      principal.value = {
        id: apiPrincipal.userId,
        email: apiPrincipal.email,
        displayName: apiPrincipal.displayName,
        role: access.role,
        workshop: { id: access.workshop.id, name: access.workshop.name },
        mustChangePassword: false,
      }
      status.value = 'authenticated'
    } catch {
      principal.value = null
      csrfToken.value = null
      mustChangePassword.value = false
      status.value = 'anonymous'
    } finally {
      restored.value = true
    }
  }

  async function login(email: string, password: string) {
    await api.post('/api/v1/auth/login', { email, password })
    restored.value = false
    await restore()
  }

  async function changePassword(currentPassword: string, newPassword: string) {
    await api.post('/api/v1/auth/change-password', {
      current_password: currentPassword,
      new_password: newPassword,
    }, csrfToken.value ?? undefined)
    restored.value = false
    await restore()
  }

  async function logout() {
    await api.post('/api/v1/auth/logout', undefined, csrfToken.value ?? undefined)
    principal.value = null
    csrfToken.value = null
    mustChangePassword.value = false
    status.value = 'anonymous'
    restored.value = true
  }

  function applySession(session: AuthSessionPayload) {
    csrfToken.value = session.data.csrf_token
    mustChangePassword.value = session.data.must_change_password
  }

  return { principal, status, restored, isAuthenticated, requiresPasswordChange, restore, login, changePassword, logout }
})
