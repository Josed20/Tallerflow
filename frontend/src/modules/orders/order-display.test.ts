import { describe, expect, it } from 'vitest'
import { formatOrderQuantity } from './order-display'

describe('order quantity display', () => {
  it('uses workshop-friendly units', () => {
    expect(formatOrderQuantity(1, 'UNITS')).toBe('1 unidad')
    expect(formatOrderQuantity(48, 'UNITS')).toBe('48 unidades')
    expect(formatOrderQuantity(2, 'DOZENS')).toBe('2 docenas')
    expect(formatOrderQuantity(2.5, 'KILOGRAMS')).toContain('kg')
  })
})
