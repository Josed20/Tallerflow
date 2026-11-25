import { beforeEach, describe, expect, it } from 'vitest'
import { ANALYTICS_KEY, AnalyticsQueue, analyticsAllowed, getAnalyticsDisabled, setAnalyticsDisabled } from './analytics'

const actor = 'aaaabbbb-1111-4111-8111-111111111111/aaaabbbb-2222-4222-8222-222222222222'
const other = 'aaaabbbb-3333-4333-8333-333333333333/aaaabbbb-2222-4222-8222-222222222222'
let now: number
beforeEach(() => { localStorage.clear(); now = Date.parse('2026-11-24T12:00:00Z') })
const queue = () => new AnalyticsQueue(localStorage, () => now)

describe('minimal telemetry queue', () => {
  it('keeps first-open and event IDs stable across network retries and restart', async () => {
    const first = queue(); first.enqueue('first_open', 'android', '3', null)
    const pending = JSON.parse(localStorage.getItem(ANALYTICS_KEY)!).queue[0].event
    await first.flush(null, async event => { expect(event).toEqual(pending); return 'retry' })
    const restarted = queue(); restarted.enqueue('first_open', 'android', '3', null)
    await restarted.flush(null, async event => { expect(event).toEqual(pending); return 'ok' })
    expect(JSON.parse(localStorage.getItem(ANALYTICS_KEY)!).queue).toEqual([])
    restarted.enqueue('first_open', 'android', '3', null)
    expect(JSON.parse(localStorage.getItem(ANALYTICS_KEY)!).queue).toEqual([])
  })
  it('never sends queued authenticated events through another account', async () => {
    const q = queue(); q.enqueue('activity', 'web', 'unknown', actor)
    let sent = 0
    await q.flush(other, async () => { sent++; return 'ok' }); expect(sent).toBe(0)
    await q.flush(actor, async () => { sent++; return 'ok' }); expect(sent).toBe(1)
  })
  it('bounds stored data to 100 events and seven days and allows only event metadata', async () => {
    const q = queue(); for (let i = 0; i < 110; i++) q.enqueue('activity', 'web', 'unknown', actor)
    const stored = JSON.parse(localStorage.getItem(ANALYTICS_KEY)!)
    expect(stored.queue).toHaveLength(100)
    expect(Object.keys(stored.queue[0].event).sort()).toEqual(['channel', 'id', 'installation_id', 'occurred_at', 'type', 'version'])
    now += 7 * 86400000 + 1
    let sent = 0; await queue().flush(actor, async () => { sent++; return 'ok' }); expect(sent).toBe(0)
    expect(JSON.parse(localStorage.getItem(ANALYTICS_KEY)!).queue).toEqual([])
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
    resolve('ok'); await sending
    expect(localStorage.getItem(ANALYTICS_KEY)).toBeNull()
    q.enqueue('first_open', 'web', 'unknown', null)
    expect(localStorage.getItem(ANALYTICS_KEY)).toBeNull()
    q.enable(); q.enqueue('first_open', 'web', 'unknown', null)
    expect(JSON.parse(localStorage.getItem(ANALYTICS_KEY)!).queue).toHaveLength(1)
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
