import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { api } from './api'
import type { SessionPrincipal, SessionStatus } from './types'

export const useSessionStore = defineStore('session', () => {
  const principal = ref<SessionPrincipal | null>(null)
  const status = ref<SessionStatus>('idle')
  const restored = ref(false)

  const isAuthenticated = computed(() => principal.value !== null)
  const requiresPasswordChange = computed(() => principal.value?.mustChangePassword === true)

  async function restore() {
    if (restored.value) return
    status.value = 'restoring'
    try {
      principal.value = await api.get<SessionPrincipal>('/api/v1/me')
      status.value = 'authenticated'
    } catch {
      principal.value = null
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
    await api.post('/api/v1/auth/change-password', { currentPassword, newPassword })
    restored.value = false
    await restore()
  }

  async function logout() {
    await api.post('/api/v1/auth/logout')
    principal.value = null
    status.value = 'anonymous'
    restored.value = true
  }

  return { principal, status, restored, isAuthenticated, requiresPasswordChange, restore, login, changePassword, logout }
})
