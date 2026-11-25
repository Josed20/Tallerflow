import { watch } from 'vue'
import { App } from '@capacitor/app'
import { pinia } from '../app/pinia'
import { useSessionStore } from '../modules/session/session.store'
import { isNativeApp, usesNativeToken } from './platform'
import { readMobileToken } from './session-token'
import { ANALYTICS_PREFERENCE_EVENT, AnalyticsQueue, analyticsAllowed, clearAnalytics } from './analytics'
import { router } from '../app/router'

// Dynamically loaded by main.ts after the web/mobile transport is configured.
// All failures are isolated from sign-in and product requests.
let started = false
async function start() {
  if (started) return
  if (import.meta.env.MODE === 'test' || location.hostname.startsWith('admin.') || location.pathname.startsWith('/consola') || location.pathname.startsWith('/seguimiento')) return
  const allowed = () => analyticsAllowed(localStorage, navigator)
  if (!allowed()) { clearAnalytics(localStorage); return }
  started = true
  const session = useSessionStore(pinia)
  const channel = isNativeApp() ? 'android' : 'web'
  // Android build = Google Play App Version Code, not versionName.
  const version = isNativeApp() ? (await App.getInfo()).build : (import.meta.env.VITE_APP_VERSION || 'unknown')
  const queue = new AnalyticsQueue(localStorage)
  const originalFetch = window.fetch.bind(window)
  const actor = () => session.principal ? `${session.principal.id}/${session.principal.workshop.id}` : null
  let delivery = new AbortController()
  let generation = 0
  let nextActivity = 0
  let pendingAuth: 'login' | 'registration' | null = null
  const flush = async () => {
    if (!allowed()) { clearAnalytics(localStorage); return }
    const currentActor = actor()
    const currentGeneration = generation
    await queue.flush(currentActor, async event => {
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      const anonymous = event.type === 'first_open'
      if (!anonymous) {
        if (!currentActor || actor() !== currentActor) return 'retry'
        if (usesNativeToken()) {
          const token = await readMobileToken()
          if (!token) return 'retry'
          headers.Authorization = `Bearer ${token}`
        } else if (session.csrfToken) headers['X-CSRF-Token'] = session.csrfToken
        else return 'retry'
      }
      const base = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/+$/, '')
      if (usesNativeToken() && !base) return 'retry'
      if (generation !== currentGeneration || !allowed() || (!anonymous && actor() !== currentActor)) return 'drop'
      const response = await originalFetch(`${base}/api/v1/analytics/${anonymous ? 'first-open' : 'events'}`, {
        signal: delivery.signal, method: 'POST', credentials: usesNativeToken() ? 'omit' : 'include', headers, body: JSON.stringify(event),
      })
      if (response.ok) return 'ok'
      return response.status === 429 || response.status >= 500 || response.status === 401 || response.status === 403 ? 'retry' : 'drop'
    })
  }
  const safeFlush = () => { void flush().catch(() => undefined) }
  queue.enqueue('first_open', channel, version, null)
  const preferenceChanged = () => {
    generation++
    if (!allowed()) { pendingAuth = null; queue.disable(); delivery.abort() }
    else { delivery = new AbortController(); queue.enable(); queue.enqueue('first_open', channel, version, null); safeFlush() }
  }
  window.addEventListener(ANALYTICS_PREFERENCE_EVENT, preferenceChanged)
  window.addEventListener('storage', event => { if (event.key === 'tallerflow.analytics.disabled' || event.key === null) preferenceChanged() })
  // Observe successful auth responses only; never inspect bodies, credentials,
  // URLs outside the auth allowlist, orders, or public tracking traffic.
  window.fetch = async (input, init) => {
    const response = await originalFetch(input, init)
    try {
      const url = new URL(typeof input === 'string' ? input : input instanceof URL ? input.href : input.url, location.href)
      const base = new URL(import.meta.env.VITE_API_BASE_URL || location.origin)
      const method = init?.method || (input instanceof Request ? input.method : 'GET')
      if (allowed() && url.origin === base.origin && method.toUpperCase() === 'POST' && response.ok) {
        if (url.pathname === '/api/v1/auth/login') pendingAuth = 'login'
        if (url.pathname === '/api/v1/auth/email-verifications/consume') pendingAuth = 'registration'
      }
    } catch { /* telemetry must never change auth behavior */ }
    return response
  }
  watch(() => session.principal, current => {
    if (current && pendingAuth) { queue.enqueue(pendingAuth, channel, version, actor()); pendingAuth = null }
    safeFlush()
  })
  const activity = (event: Event) => {
    if (!event.isTrusted || !allowed() || !actor() || document.visibilityState !== 'visible' || Date.now() < nextActivity || location.pathname.startsWith('/consola') || location.pathname.startsWith('/seguimiento')) return
    nextActivity = Date.now() + 60000
    try { queue.enqueue('activity', channel, version, actor()); safeFlush() } catch { /* storage can be unavailable */ }
  }
  document.addEventListener('pointerdown', activity, { passive: true })
  document.addEventListener('keydown', activity, { passive: true })
  window.addEventListener('online', safeFlush)
  document.addEventListener('visibilitychange', () => { if (document.visibilityState === 'visible') safeFlush() })
  window.setInterval(safeFlush, 60000)
  safeFlush()
}
const safeStart = () => { void start().catch(() => { started = false }) }
router.afterEach(safeStart)
window.addEventListener(ANALYTICS_PREFERENCE_EVENT, safeStart)
safeStart()
