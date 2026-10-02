export type InternalRole = 'OWNER' | 'ADMIN' | 'OPERATOR'

export function canManageOperations(role: string | undefined): role is 'OWNER' | 'ADMIN' {
  return role === 'OWNER' || role === 'ADMIN'
}

export function canAdvanceProduction(role: string | undefined): role is InternalRole {
  return role === 'OWNER' || role === 'ADMIN' || role === 'OPERATOR'
}
