import { describe, expect, it } from 'vitest'

import { canAccess, hasPermission, hasRole } from './permissions'
import type { UserSummary } from './types'

const user: UserSummary = {
  id: 'user-1',
  username: 'operator',
  roles: ['System:Admin', 'Reviewer'],
  permissions: ['knowledge:read', ' report:write ', 'QA:USE'],
}

describe('permission helpers', () => {
  it('normalizes permission and role checks', () => {
    expect(hasPermission(user.permissions, 'qa:use')).toBe(true)
    expect(hasPermission(user.permissions, 'REPORT:WRITE')).toBe(true)
    expect(hasRole(user.roles, 'system:admin')).toBe(true)
  })

  it('requires an authenticated user when no explicit requirement is provided', () => {
    expect(canAccess(user)).toBe(true)
    expect(canAccess(null)).toBe(false)
  })

  it('combines role, all, and any requirements', () => {
    expect(
      canAccess(user, {
        all: ['knowledge:read'],
        any: ['document:upload', 'qa:use'],
        roles: ['system:admin'],
      }),
    ).toBe(true)

    expect(canAccess(user, { all: ['knowledge:read', 'knowledge:write'] })).toBe(false)
    expect(canAccess(user, { any: ['system:admin'] })).toBe(false)
    expect(canAccess(user, { roles: ['auditor'] })).toBe(false)
  })
})
