import { afterEach, expect, it, vi } from 'vitest'
import { ANALYTICS_KEY, setAnalyticsDisabled } from './analytics'

const mocks = vi.hoisted(() => ({ afterEach: vi.fn(), session: { principal: null, csrfToken: null } }))
vi.mock('../app/router', () => ({ router: { afterEach: mocks.afterEach } }))
vi.mock('../app/pinia', () => ({ pinia: {} }))
vi.mock('../modules/session/session.store', () => ({ useSessionStore: () => mocks.session }))
vi.mock('./platform', () => ({ isNativeApp: () => false, usesNativeToken: () => false }))
vi.mock('./session-token', () => ({ readMobileToken: async () => null }))

afterEach(() => { vi.clearAllTimers(); vi.useRealTimers(); vi.unstubAllGlobals(); vi.unstubAllEnvs(); localStorage.clear(); history.replaceState({}, '', '/') })

it('starts after leaving public tracking in the SPA and opt-out aborts delivery without restoring storage', async () => {
  vi.useFakeTimers(); vi.stubEnv('MODE', 'development'); localStorage.clear()
  history.replaceState({}, '', '/seguimiento/public-token')
  let signal: AbortSignal | undefined
  const fetchMock = vi.fn((_input: RequestInfo | URL, init?: RequestInit) => new Promise<Response>((_resolve, reject) => {
    signal = init?.signal as AbortSignal
    signal?.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')))
  }))
  vi.stubGlobal('fetch', fetchMock)
  await import('./analytics-bootstrap')
  expect(fetchMock).not.toHaveBeenCalled()
  expect(localStorage.getItem(ANALYTICS_KEY)).toBeNull()
  history.replaceState({}, '', '/login')
  mocks.afterEach.mock.calls[0]![0]()
  await Promise.resolve()
  expect(fetchMock).toHaveBeenCalledTimes(1)
  expect(signal?.aborted).toBe(false)
  expect(localStorage.getItem(ANALYTICS_KEY)).not.toBeNull()
  setAnalyticsDisabled(true)
  expect(signal?.aborted).toBe(true)
  await Promise.resolve(); await Promise.resolve()
  expect(localStorage.getItem(ANALYTICS_KEY)).toBeNull()
})
