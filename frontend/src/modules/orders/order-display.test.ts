import { describe, expect, it } from 'vitest'
import { completedStations, formatOrderQuantity } from './order-display'

describe('order quantity display', () => {
  it('uses workshop-friendly units', () => {
    expect(formatOrderQuantity(1, 'UNITS')).toBe('1 unidad')
    expect(formatOrderQuantity(48, 'UNITS')).toBe('48 unidades')
    expect(formatOrderQuantity(2, 'DOZENS')).toBe('2 docenas')
    expect(formatOrderQuantity(2.5, 'KILOGRAMS')).toContain('kg')
  })
})

describe('completed stations', () => {
  it('counts the stations before the current one', () => {
    expect(completedStations({ status: 'ACTIVE', current_stage: 'CORTE' })).toBe(0)
    expect(completedStations({ status: 'ACTIVE', current_stage: 'Control de calidad' })).toBe(3)
  })

  it('fills the whole seam once the order is completed', () => {
    expect(completedStations({ status: 'COMPLETED', current_stage: 'ENTREGA' })).toBe(7)
  })

  it('treats an unknown station as not started', () => {
    expect(completedStations({ status: 'ACTIVE', current_stage: null })).toBe(0)
  })
})
