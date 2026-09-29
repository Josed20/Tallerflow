import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { router, routes } from './router'
import { pinia as applicationPinia } from './pinia'
import { useSessionStore } from '../modules/session/session.store'

describe('Router Guard & Route Resolution', () => {
  let pinia: ReturnType<typeof createPinia>

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
    vi.unstubAllGlobals()
  })

  function createTestRouter(sessionStore: ReturnType<typeof useSessionStore>) {
    const testRouter = createRouter({
      history: createMemoryHistory(),
      routes,
    })

    testRouter.beforeEach(async (to) => {
      await sessionStore.restore()

      if (to.name === 'not-found') {
        return
      }

      if (to.meta.public && sessionStore.isAuthenticated) {
        return sessionStore.requiresPasswordChange ? '/change-password' : '/app'
      }

      if (!to.meta.public && !sessionStore.isAuthenticated) {
        return { name: 'login', query: { redirect: to.fullPath } }
      }

      if (sessionStore.requiresPasswordChange && to.name !== 'change-password') {
        return '/change-password'
      }

      if (!sessionStore.requiresPasswordChange && to.name === 'change-password') {
        return '/app'
      }
    })

    return testRouter
  }

  it('allows unauthenticated navigation to /ruta-rara and resolves not-found without redirecting to login', async () => {
    // Mock anonymous session (401)
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('unauthenticated')))
    const session = useSessionStore()
    const testRouter = createTestRouter(session)

    await testRouter.push('/ruta-rara')
    await testRouter.isReady()

    expect(testRouter.currentRoute.value.name).toBe('not-found')
    expect(testRouter.currentRoute.value.path).toBe('/ruta-rara')
  })

  it('redirects unauthenticated access to protected route /app to /login with redirect query', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('unauthenticated')))
    const session = useSessionStore()
    const testRouter = createTestRouter(session)

    await testRouter.push('/app')
    await testRouter.isReady()

    expect(testRouter.currentRoute.value.name).toBe('login')
    expect(testRouter.currentRoute.value.query.redirect).toBe('/app')
  })

  it('allows unauthenticated navigation to /forgot-password', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('unauthenticated')))
    const session = useSessionStore()
    const testRouter = createTestRouter(session)

    await testRouter.push('/forgot-password')
    await testRouter.isReady()

    expect(testRouter.currentRoute.value.name).toBe('forgot-password')
    expect(testRouter.currentRoute.value.path).toBe('/forgot-password')
  })

  it('allows unauthenticated navigation to /reset-password', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('unauthenticated')))
    const session = useSessionStore()
    const testRouter = createTestRouter(session)

    await testRouter.push('/reset-password?token=sample-123')
    await testRouter.isReady()

    expect(testRouter.currentRoute.value.name).toBe('reset-password')
    expect(testRouter.currentRoute.value.query.token).toBe('sample-123')
  })

  it('allows an authenticated user to open a password reset link', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const path = input.toString()
      if (path.endsWith('/api/v1/auth/session')) {
        return new Response(JSON.stringify({
          data: { csrf_token: 'csrf-token', must_change_password: false },
        }), { status: 200 })
      }
      if (path.endsWith('/api/v1/me')) {
        return new Response(JSON.stringify({
          data: {
            userId: 'user-1',
            email: 'owner@tallerflow.pe',
            displayName: 'Owner',
            workshopId: 'workshop-1',
            role: 'OWNER',
            passwordChangeRequired: false,
          },
        }), { status: 200 })
      }
      return new Response(JSON.stringify({
        data: {
          workshop: { id: 'workshop-1', name: 'Taller', timezone: 'America/Lima' },
          role: 'OWNER',
        },
      }), { status: 200 })
    }))
    useSessionStore(applicationPinia)

    await router.push('/reset-password?token=sample-authenticated-token')
    await router.isReady()

    expect(router.currentRoute.value.name).toBe('reset-password')
    expect(router.currentRoute.value.query.token).toBe('sample-authenticated-token')
  })
})
