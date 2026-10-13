import { describe, expect, it } from 'vitest'
import { matchesOrderSearch, orderNumberFromQuery } from './order-search'
import type { Order } from './api'

const order = { order_number: 3, code: 'OT-000003', product: 'Polos', client_name: 'Confecciones Sur', current_stage: 'CORTE' } as Order

describe('order search', () => {
  it('accepts the full code and short numeric forms', () => {
    expect(orderNumberFromQuery('OT-000003')).toBe(3)
    expect(orderNumberFromQuery('OT-3')).toBe(3)
    expect(orderNumberFromQuery('3')).toBe(3)
    expect(matchesOrderSearch(order, '3')).toBe(true)
    expect(matchesOrderSearch(order, 'OT-000003')).toBe(true)
    expect(matchesOrderSearch(order, '4')).toBe(false)
  })

  it('keeps text search for product and client', () => {
    expect(matchesOrderSearch(order, 'polos')).toBe(true)
    expect(matchesOrderSearch(order, 'confecciones')).toBe(true)
    expect(matchesOrderSearch(order, 'costura')).toBe(false)
  })
})
