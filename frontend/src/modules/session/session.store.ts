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
        return
      }
      await loadPrincipal()
    } catch {
      clearSession()
    } finally {
      restored.value = true
    }
  }

  async function login(email: string, password: string) {
    const result = await api.post<AuthSessionPayload>('/api/v1/auth/login', { email, password })
    applySession(result)
    restored.value = true
    if (!mustChangePassword.value) await loadPrincipal()
  }

  async function changePassword(currentPassword: string, newPassword: string) {
    const result = await api.post<AuthSessionPayload>('/api/v1/auth/change-password', {
      current_password: currentPassword,
      new_password: newPassword,
    }, csrfToken.value ?? undefined)
    applySession(result)
    restored.value = true
    if (!mustChangePassword.value) await loadPrincipal()
  }

  async function acceptSession(session: AuthSessionPayload) {
    applySession(session)
    restored.value = true
    if (!mustChangePassword.value) {
      try {
        await loadPrincipal()
      } catch (error) {
        clearSession()
        restored.value = true
        throw error
      }
    }
  }

  async function logout() {
    await api.post('/api/v1/auth/logout', undefined, csrfToken.value ?? undefined)
    clearSession()
    restored.value = true
  }

  function applySession(session: AuthSessionPayload) {
    csrfToken.value = session.data.csrf_token
    mustChangePassword.value = session.data.must_change_password
    status.value = mustChangePassword.value ? 'password-change-required' : 'authenticated'
  }

  async function loadPrincipal() {
    const [apiPrincipal, access] = await Promise.all([
      api.get<ApiPrincipal>('/api/v1/me'),
      api.get<WorkshopAccess>('/api/v1/workshops/current'),
    ])
    principal.value = {
      id: apiPrincipal.data.userId,
      email: apiPrincipal.data.email,
      displayName: apiPrincipal.data.displayName,
      role: access.data.role,
      workshop: { id: access.data.workshop.id, name: access.data.workshop.name },
      mustChangePassword: false,
    }
    status.value = 'authenticated'
  }

  function clearSession() {
    principal.value = null
    csrfToken.value = null
    mustChangePassword.value = false
    status.value = 'anonymous'
  }

  return { principal, status, restored, csrfToken, isAuthenticated, requiresPasswordChange, restore, login, changePassword, acceptSession, logout }
})
