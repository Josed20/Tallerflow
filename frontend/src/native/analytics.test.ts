import { beforeEach, describe, expect, it } from 'vitest'
import { ANALYTICS_EVENT_PREFIX, ANALYTICS_KEY, AnalyticsQueue, analyticsAllowed, getAnalyticsDisabled, setAnalyticsDisabled } from './analytics'

const actor = 'aaaabbbb-1111-4111-8111-111111111111/aaaabbbb-2222-4222-8222-222222222222'
const other = 'aaaabbbb-3333-4333-8333-333333333333/aaaabbbb-2222-4222-8222-222222222222'
let now: number
beforeEach(() => { localStorage.clear(); now = Date.parse('2026-11-24T12:00:00Z') })
const pendingEvents = () => Object.keys(localStorage).filter(key => key.startsWith(ANALYTICS_EVENT_PREFIX)).map(key => JSON.parse(localStorage.getItem(key)!))
const queue = () => new AnalyticsQueue(localStorage, () => now)

describe('minimal telemetry queue', () => {
  it('does not overwrite an event when another tab writes between a read and write', async () => {
    let interleave = false
    let b!: AnalyticsQueue
    const storage: Storage = {
      get length() { return localStorage.length },
      clear: () => localStorage.clear(), key: index => localStorage.key(index),
      getItem: key => localStorage.getItem(key), removeItem: key => localStorage.removeItem(key),
      setItem: (key, value) => {
        if (interleave) { interleave = false; b.enqueue('activity', 'web', '1.1.0', actor) }
        localStorage.setItem(key, value)
      },
    }
    const a = new AnalyticsQueue(storage, () => now); b = queue()
    interleave = true
    a.enqueue('activity', 'web', '1.1.0', actor)
    const sent: string[] = []
    await queue().flush(actor, async event => { sent.push(event.id); return 'ok' })
    expect(new Set(sent).size).toBe(2)
  })
  it('preserves another tab pending event when an idle tab flushes', async () => {
    const a = queue(); const b = queue()
    a.enqueue('activity', 'web', '1.1.0', actor)
    await b.flush(other, async () => 'ok')
    const sent: string[] = []
    await queue().flush(actor, async event => { sent.push(event.type); return 'ok' })
    expect(sent).toEqual(['activity'])
  })
  it('preserves writes from another tab during an in-flight delivery', async () => {
    const a = queue(); const b = queue()
    a.enqueue('first_open', 'web', '1.1.0', null)
    let resolve!: (result: 'ok') => void
    const sending = a.flush(null, () => new Promise(done => { resolve = done }))
    b.enqueue('activity', 'web', '1.1.0', actor)
    resolve('ok'); await sending
    const sent: string[] = []
    await queue().flush(actor, async event => { sent.push(event.type); return 'ok' })
    expect(sent).toEqual(['activity'])
  })
  it('keeps first-open and event IDs stable across network retries and restart', async () => {
    const first = queue(); first.enqueue('first_open', 'android', '3', null)
    const pending = pendingEvents()[0].event
    await first.flush(null, async event => { expect(event).toEqual(pending); return 'retry' })
    const restarted = queue(); restarted.enqueue('first_open', 'android', '3', null)
    await restarted.flush(null, async event => { expect(event).toEqual(pending); return 'ok' })
    expect(pendingEvents()).toEqual([])
    restarted.enqueue('first_open', 'android', '3', null)
    expect(pendingEvents()).toEqual([])
  })
  it('never sends queued authenticated events through another account', async () => {
    const q = queue(); q.enqueue('activity', 'web', 'unknown', actor)
    let sent = 0
    await q.flush(other, async () => { sent++; return 'ok' }); expect(sent).toBe(0)
    await q.flush(actor, async () => { sent++; return 'ok' }); expect(sent).toBe(1)
  })
  it('bounds stored data to 100 events and seven days and allows only event metadata', async () => {
    const q = queue(); for (let i = 0; i < 110; i++) q.enqueue('activity', 'web', 'unknown', actor)
    const stored = { queue: pendingEvents() }
    expect(stored.queue).toHaveLength(100)
    expect(Object.keys(stored.queue[0].event).sort()).toEqual(['channel', 'id', 'installation_id', 'occurred_at', 'type', 'version'])
    now += 7 * 86400000 + 1
    let sent = 0; await queue().flush(actor, async () => { sent++; return 'ok' }); expect(sent).toBe(0)
    expect(pendingEvents()).toEqual([])
  })
  it('honors disabled, DNT and GPC', () => {
    expect(analyticsAllowed(localStorage, { doNotTrack: '1' } as Navigator)).toBe(false)
    expect(analyticsAllowed(localStorage, { globalPrivacyControl: true } as unknown as Navigator)).toBe(false)
    localStorage.setItem('tallerflow.analytics.disabled', 'true')
    expect(analyticsAllowed(localStorage, {} as Navigator)).toBe(false)
  })
  it('disables and clears even while a delivery promise is pending', async () => {
    const q = queue(); q.enqueue('first_open', 'web', 'unknown', null)
    let resolve!: (result: 'ok') => void
    const sending = q.flush(null, () => new Promise(done => { resolve = done }))
    q.disable()
    expect(localStorage.getItem(ANALYTICS_KEY)).toBeNull()
    expect(pendingEvents()).toEqual([])
    resolve('ok'); await sending
    expect(localStorage.getItem(ANALYTICS_KEY)).toBeNull()
    q.enqueue('first_open', 'web', 'unknown', null)
    expect(localStorage.getItem(ANALYTICS_KEY)).toBeNull()
    q.enable(); q.enqueue('first_open', 'web', 'unknown', null)
    expect(pendingEvents()).toHaveLength(1)
  })
  it('preferences clear storage synchronously without bootstrapping analytics', () => {
    localStorage.setItem(ANALYTICS_KEY, '{}')
    let notified = false
    const listener = () => { notified = true; expect(localStorage.getItem(ANALYTICS_KEY)).toBeNull() }
    window.addEventListener('tallerflow:analytics-preference', listener)
    setAnalyticsDisabled(true)
    expect(getAnalyticsDisabled()).toBe(true); expect(notified).toBe(true)
    window.removeEventListener('tallerflow:analytics-preference', listener)
    setAnalyticsDisabled(false); expect(getAnalyticsDisabled()).toBe(false)
  })
})
