import { createPinia } from 'pinia'
import { createMemoryHistory } from 'vue-router'
import { describe, expect, it } from 'vitest'
import { createAppRouter, pageTitle, routes } from './router'
import { useSessionStore } from '../modules/session/session.store'

describe('page titles', () => {
  it('names the current screen in the browser tab', async () => {
    const pinia = createPinia()
    const session = useSessionStore(pinia)
    session.restore = async () => undefined
    const router = createAppRouter({ pinia, history: createMemoryHistory() })

    await router.push('/terminos')
    await router.isReady()

    expect(document.title).toBe('Términos de servicio · TallerFlow')
  })

  it('keeps the product tagline on the landing page and unnamed routes', () => {
    expect(pageTitle('landing')).toBe('TallerFlow — Trazabilidad para talleres de confección')
    expect(pageTitle(undefined)).toBe('TallerFlow — Trazabilidad para talleres de confección')
  })

  it('has a title for every named screen except the landing page', () => {
    const untitled = routes
      .map((route) => route.name)
      .filter((name): name is string => typeof name === 'string' && name !== 'landing')
      .filter((name) => !pageTitle(name).endsWith(' · TallerFlow'))

    expect(untitled).toEqual([])
  })
})
