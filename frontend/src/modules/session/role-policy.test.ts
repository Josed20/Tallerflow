import { describe, expect, it } from 'vitest'
import { canAdvanceProduction, canManageOperations } from './role-policy'

describe('role policy', () => {
  it('allows OWNER and ADMIN to manage operations', () => {
    expect(canManageOperations('OWNER')).toBe(true)
    expect(canManageOperations('ADMIN')).toBe(true)
    expect(canManageOperations('OPERATOR')).toBe(false)
  })

  it('allows internal production roles to advance the current stage', () => {
    expect(canAdvanceProduction('OWNER')).toBe(true)
    expect(canAdvanceProduction('ADMIN')).toBe(true)
    expect(canAdvanceProduction('OPERATOR')).toBe(true)
    expect(canAdvanceProduction(undefined)).toBe(false)
  })
})
