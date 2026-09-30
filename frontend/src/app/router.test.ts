import { createPinia } from 'pinia'
import { createMemoryHistory } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createAppRouter } from './router'
import { useOnboardingStore } from '../modules/onboarding/onboarding.store'
import { useSessionStore } from '../modules/session/session.store'

describe('application router guards', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('sends an anonymous visitor from login to onboarding while the install is available', async () => {
    const { router, onboarding } = buildAnonymousRouter()
    vi.spyOn(onboarding, 'loadStatus').mockResolvedValue(true)

    await router.push('/login')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/onboarding')
    expect(onboarding.loadStatus).toHaveBeenCalled()
  })

  it('sends onboarding back to login once the installation is claimed', async () => {
    const { router, onboarding } = buildAnonymousRouter()
    vi.spyOn(onboarding, 'loadStatus').mockResolvedValue(false)

    await router.push('/onboarding')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/login')
  })

  it('routes a first anonymous private visit through the available onboarding flow', async () => {
    const { router, onboarding } = buildAnonymousRouter()
    vi.spyOn(onboarding, 'loadStatus').mockResolvedValue(true)

    await router.push('/app')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/onboarding')
  })

  it('keeps login usable without a redirect loop when the public status check fails', async () => {
    const { router, onboarding } = buildAnonymousRouter()
    vi.spyOn(onboarding, 'loadStatus').mockRejectedValue(new Error('network unavailable'))

    await router.push('/login')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/login')
  })

  it('resolves an unknown route as not-found for an anonymous visitor', async () => {
    const { router } = buildAnonymousRouter()

    await router.push('/ruta-rara')
    await router.isReady()

    expect(router.currentRoute.value.name).toBe('not-found')
    expect(router.currentRoute.value.path).toBe('/ruta-rara')
  })

  it('redirects an anonymous private visit to login when onboarding is unavailable', async () => {
    const { router, onboarding } = buildAnonymousRouter()
    vi.spyOn(onboarding, 'loadStatus').mockResolvedValue(false)

    await router.push('/app')
    await router.isReady()

    expect(router.currentRoute.value.name).toBe('login')
    expect(router.currentRoute.value.query.redirect).toBe('/app')
  })

  it('allows anonymous navigation to password recovery routes', async () => {
    const { router } = buildAnonymousRouter()

    await router.push('/forgot-password')
    await router.isReady()
    expect(router.currentRoute.value.name).toBe('forgot-password')

    await router.push('/reset-password?token=sample-123')
    expect(router.currentRoute.value.name).toBe('reset-password')
    expect(router.currentRoute.value.query.token).toBe('sample-123')
  })

  it('exposes the public invitation and protected team management routes', async () => {
    const { router } = buildAnonymousRouter()

    await router.push('/join?token=invite-token')
    await router.isReady()
    expect(router.currentRoute.value.name).toBe('join')
    expect(router.currentRoute.value.query.token).toBe('invite-token')

    await router.push('/app/team')
    expect(router.currentRoute.value.name).toBe('login')
    expect(router.currentRoute.value.query.redirect).toBe('/app/team')
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
    const pinia = createPinia()
    const router = createAppRouter({ pinia, history: createMemoryHistory() })

    await router.push('/reset-password?token=sample-authenticated-token')
    await router.isReady()

    expect(router.currentRoute.value.name).toBe('reset-password')
    expect(router.currentRoute.value.query.token).toBe('sample-authenticated-token')
  })
})

function buildAnonymousRouter() {
  const pinia = createPinia()
  const session = useSessionStore(pinia)
  const onboarding = useOnboardingStore(pinia)
  vi.spyOn(session, 'restore').mockResolvedValue()
  const router = createAppRouter({ pinia, history: createMemoryHistory() })
  return { router, onboarding }
}
