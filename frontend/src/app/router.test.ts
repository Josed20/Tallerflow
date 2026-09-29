import { createPinia } from 'pinia'
import { createMemoryHistory } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createAppRouter } from './router'
import { useOnboardingStore } from '../modules/onboarding/onboarding.store'
import { useSessionStore } from '../modules/session/session.store'

describe('application router onboarding guard', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('sends an anonymous visitor from login to onboarding while the install is available', async () => {
    const { router, onboarding } = buildRouter()
    vi.spyOn(onboarding, 'loadStatus').mockResolvedValue(true)

    await router.push('/login')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/onboarding')
    expect(onboarding.loadStatus).toHaveBeenCalled()
  })

  it('sends onboarding back to login once the installation is claimed', async () => {
    const { router, onboarding } = buildRouter()
    vi.spyOn(onboarding, 'loadStatus').mockResolvedValue(false)

    await router.push('/onboarding')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/login')
  })

  it('routes a first anonymous private visit through the available onboarding flow', async () => {
    const { router, onboarding } = buildRouter()
    vi.spyOn(onboarding, 'loadStatus').mockResolvedValue(true)

    await router.push('/app')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/onboarding')
  })

  it('keeps login usable without a redirect loop when the public status check fails', async () => {
    const { router, onboarding } = buildRouter()
    vi.spyOn(onboarding, 'loadStatus').mockRejectedValue(new Error('network unavailable'))

    await router.push('/login')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/login')
  })
})

function buildRouter() {
  const pinia = createPinia()
  const session = useSessionStore(pinia)
  const onboarding = useOnboardingStore(pinia)
  vi.spyOn(session, 'restore').mockResolvedValue()
  const router = createAppRouter({ pinia, history: createMemoryHistory() })
  return { router, onboarding }
}
