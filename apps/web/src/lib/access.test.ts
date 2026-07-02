import { describe, expect, it } from 'vitest'

import { adminShellAccess } from './access'
import { canAccess } from './permissions'
import type { UserSummary } from './types'

const businessUser: UserSummary = {
  id: 'user-standard',
  username: 'standard',
  roles: ['standard'],
  permissions: ['qa:use', 'knowledge:read', 'document:upload'],
}

describe('shared access requirements', () => {
  it('does not grant the admin shell to unrelated standard business permissions', () => {
    expect(canAccess(businessUser, adminShellAccess)).toBe(false)
  })

  it('grants the admin shell to report authorities used by admin report routes', () => {
    expect(
      canAccess(
        {
          ...businessUser,
          permissions: ['report:read'],
        },
        adminShellAccess,
      ),
    ).toBe(true)
  })

  it('grants the admin shell to admin authorities', () => {
    expect(
      canAccess(
        {
          ...businessUser,
          roles: ['admin'],
          permissions: [],
        },
        adminShellAccess,
      ),
    ).toBe(true)
  })
})
