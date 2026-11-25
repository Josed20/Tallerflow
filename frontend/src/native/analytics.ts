export const ANALYTICS_KEY = 'tallerflow.analytics.v1'
export const ANALYTICS_DISABLED_KEY = 'tallerflow.analytics.disabled'
export const ANALYTICS_PREFERENCE_EVENT = 'tallerflow:analytics-preference'

// Importing preferences never starts telemetry. Runtime listeners synchronously
// clear memory and abort pending delivery when this setter disables it.
export function setAnalyticsDisabled(disabled: boolean): void {
  if (disabled) {
    localStorage.setItem(ANALYTICS_DISABLED_KEY, 'true')
    localStorage.removeItem(ANALYTICS_KEY)
  } else localStorage.removeItem(ANALYTICS_DISABLED_KEY)
  window.dispatchEvent(new CustomEvent(ANALYTICS_PREFERENCE_EVENT, { detail: { disabled } }))
}
export function getAnalyticsDisabled(): boolean { try { return localStorage.getItem(ANALYTICS_DISABLED_KEY) === 'true' } catch { return true } }
export type AnalyticsEvent = {
  id: string; installation_id: string; type: 'first_open' | 'registration' | 'login' | 'activity'
  occurred_at: string; channel: 'web' | 'android'; version: string
}
type Pending = { event: AnalyticsEvent; actor: string | null }
type State = { installation: string; opened: boolean; queue: Pending[] }
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i
const types = ['first_open', 'registration', 'login', 'activity']
const WEEK = 7 * 86400000

export function analyticsAllowed(storage: Storage, nav: Navigator): boolean {
  return storage.getItem(ANALYTICS_DISABLED_KEY) !== 'true' && nav.doNotTrack !== '1' && !(nav as Navigator & { globalPrivacyControl?: boolean }).globalPrivacyControl
}

// This schema is also the complete allowlist of data persisted by telemetry.
function validPending(p: Pending, now: number): boolean {
  if (!p || !p.event || !types.includes(p.event.type)) return false
  const e = p.event
  return UUID.test(e.id) && UUID.test(e.installation_id) && ['web', 'android'].includes(e.channel)
    && typeof e.version === 'string' && /^[A-Za-z0-9][A-Za-z0-9._+-]{0,63}$/.test(e.version)
    && Number.isFinite(Date.parse(e.occurred_at)) && Date.parse(e.occurred_at) >= now - WEEK && Date.parse(e.occurred_at) <= now + 300000
    && (e.type === 'first_open' ? p.actor === null : typeof p.actor === 'string' && /^[0-9a-f-]{36}\/[0-9a-f-]{36}$/i.test(p.actor))
    && Object.keys(e).sort().join(',') === 'channel,id,installation_id,occurred_at,type,version'
    && Object.keys(p).sort().join(',') === 'actor,event'
}

export class AnalyticsQueue {
  private state: State
  private sending = false
  private enabled = true
  constructor(private storage: Storage, private now = () => Date.now(), private random = () => crypto.randomUUID()) {
    let stored: State | null = null
    try { stored = JSON.parse(storage.getItem(ANALYTICS_KEY) || 'null') } catch { /* corrupt storage starts a fresh installation */ }
    this.state = stored && UUID.test(stored.installation) && typeof stored.opened === 'boolean' && Array.isArray(stored.queue)
      ? { installation: stored.installation, opened: stored.opened, queue: stored.queue.filter(p => validPending(p, now())).slice(-100) }
      : { installation: random(), opened: false, queue: [] }
  }
  private save() { if (this.enabled) this.storage.setItem(ANALYTICS_KEY, JSON.stringify(this.state)) }
  disable(): void {
    this.enabled = false
    this.state = { installation: this.random(), opened: false, queue: [] }
    this.storage.removeItem(ANALYTICS_KEY)
  }
  enable(): void { this.enabled = true }
  enqueue(type: AnalyticsEvent['type'], channel: AnalyticsEvent['channel'], version: string, actor: string | null): void {
    if (!this.enabled) return
    if (type === 'first_open' && this.state.opened) return
    const pending: Pending = { event: { id: this.random(), installation_id: this.state.installation, type, occurred_at: new Date(this.now()).toISOString(), channel, version }, actor }
    if (!validPending(pending, this.now())) return
    this.state.queue = this.state.queue.filter(p => validPending(p, this.now())).slice(-99)
    this.state.queue.push(pending)
    if (type === 'first_open') this.state.opened = true
    this.save()
  }
  async flush(actor: string | null, send: (event: AnalyticsEvent) => Promise<'ok' | 'retry' | 'drop'>): Promise<void> {
    if (this.sending || !this.enabled) return
    this.sending = true
    try {
      this.state.queue = this.state.queue.filter(p => validPending(p, this.now()))
      for (const p of [...this.state.queue]) {
        // Never attribute another account's queued event to the current session.
        if (p.actor !== null && p.actor !== actor) continue
        const result = await send(p.event).catch(() => 'retry' as const)
        if (!this.enabled) return
        if (result === 'retry') break
        this.state.queue = this.state.queue.filter(item => item.event.id !== p.event.id)
        this.save()
      }
      this.save()
    } finally { this.sending = false }
  }
}
