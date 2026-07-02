import { describe, expect, it } from 'vitest'

import { ApiError } from '@/api/client'
import type { UserSummary } from '@/lib/types'

import {
  formatUserManagementError,
  getAssignableRoles,
  normalizeAdminUsersSearch,
  validateCreateAdminUserForm,
  validatePasswordChangeForm,
  validateProfileForm,
} from './user-management'

const adminUser: UserSummary = {
  id: 'admin-1',
  permissions: [],
  roles: ['admin'],
  username: 'admin',
}

const superAdminUser: UserSummary = {
  id: 'super-1',
  permissions: [],
  roles: ['super_admin'],
  username: 'super',
}

describe('user management helpers', () => {
  it('filters assignable roles by caller authority and never exposes super_admin', () => {
    expect(getAssignableRoles(adminUser)).toEqual(['standard'])
    expect(getAssignableRoles(superAdminUser)).toEqual(['standard', 'admin'])
    expect(
      getAssignableRoles({
        id: 'system-1',
        permissions: ['system:admin'],
        roles: [],
        username: 'system',
      }),
    ).toEqual([])
    expect(
      getAssignableRoles({
        id: 'admin-system-1',
        permissions: ['system:admin'],
        roles: ['admin'],
        username: 'admin-system',
      }),
    ).toEqual(['standard'])
  })

  it('builds server-backed admin-user search params from URL search', () => {
    expect(
      normalizeAdminUsersSearch({
        page: '3',
        pageSize: '50',
        role: 'admin',
        status: 'disabled',
        username: '  lin  ',
      }),
    ).toEqual({
      page: 3,
      pageSize: 50,
      role: 'admin',
      status: 'disabled',
      username: 'lin',
    })

    expect(normalizeAdminUsersSearch({ page: '-1', pageSize: '999', role: 'super_admin' })).toEqual(
      {
        page: 1,
        pageSize: 20,
      },
    )
  })

  it('requires administrator-entered temporary passwords and allowed role options', () => {
    const baseForm = {
      displayName: '',
      email: '',
      phone: '',
      role: 'admin' as const,
      temporaryPassword: 'short',
      username: 'new-user',
    }

    expect(validateCreateAdminUserForm(baseForm, ['standard'])).toMatchObject({
      valid: false,
    })

    const valid = validateCreateAdminUserForm(
      { ...baseForm, role: 'standard', temporaryPassword: 'temporary-password' },
      ['standard'],
    )
    expect(valid).toMatchObject({
      valid: true,
      request: {
        email: null,
        phone: null,
        role: 'standard',
        temporaryPassword: 'temporary-password',
        username: 'new-user',
      },
    })
  })

  it('validates profile optional contact fields and required password-change confirmation', () => {
    expect(validateProfileForm({ displayName: '', email: '', phone: '' })).toMatchObject({
      request: { displayName: '', email: null, phone: null },
      valid: true,
    })
    expect(validateProfileForm({ displayName: '', email: 'bad-email', phone: '' })).toMatchObject({
      valid: false,
    })
    expect(
      validatePasswordChangeForm({
        currentPassword: 'temporary-password',
        newPassword: 'new-pass-1',
        newPasswordConfirmation: 'new-pass-2',
      }),
    ).toMatchObject({ valid: false })
  })

  it('formats gateway validation fields and request id', () => {
    const error = new ApiError({
      code: 'validation_error',
      fields: { temporaryPassword: 'too short' },
      message: 'request validation failed',
      requestId: 'req-user-1',
      status: 400,
    })

    expect(formatUserManagementError(error, '创建用户失败')).toBe(
      '创建用户失败: request validation failed；temporaryPassword: too short（requestId: req-user-1）',
    )
  })
})
