import { afterEach, expect, it, vi } from 'vitest'

const state = vi.hoisted(() => ({ native: false, events: [] as string[] }))
vi.mock('vue', () => ({ createApp: () => ({ use() { return this }, mount: vi.fn() }) }))
vi.mock('@tanstack/vue-query', () => ({ VueQueryPlugin: {} }))
vi.mock('./App.vue', () => ({ default: {} }))
vi.mock('./app/pinia', () => ({ pinia: {} }))
vi.mock('./app/router', () => ({ router: { replace: vi.fn(), back: vi.fn() } }))
vi.mock('./native/platform', () => ({ isNativeApp: () => state.native, usesNativeToken: () => state.native }))
vi.mock('./native/mobile-transport', () => ({ configureMobileTransport: () => { state.events.push('mobile-ready') } }))
vi.mock('./modules/session/api', () => ({ configureApiTransport: () => { state.events.push('web-ready') } }))
vi.mock('./modules/session/session.store', () => ({ useSessionStore: () => ({ clearSession: vi.fn() }) }))
vi.mock('@capacitor/app', () => ({ App: { addListener: vi.fn().mockResolvedValue({ remove: vi.fn() }), exitApp: vi.fn() } }))

afterEach(() => { vi.doUnmock('./native/analytics-bootstrap'); vi.resetModules() })

it.each([false, true])('the real entry point loads analytics after configuring transport (Android=%s)', async native => {
  vi.resetModules(); state.native = native; state.events = []
  vi.doMock('./native/analytics-bootstrap', () => {
    // This factory runs only when main.ts actually imports the bootstrap module.
    expect(state.events).toEqual([native ? 'mobile-ready' : 'web-ready'])
    state.events.push('analytics-loaded')
    return {}
  })
  await import('./main')
  await vi.waitFor(() => expect(state.events).toEqual([native ? 'mobile-ready' : 'web-ready', 'analytics-loaded']))
})
