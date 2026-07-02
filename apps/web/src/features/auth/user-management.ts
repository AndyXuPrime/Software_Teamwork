import { z } from 'zod'

import { ApiError } from '@/api/client'
import { hasAuthority } from '@/lib/permissions'
import type {
  AdminUser,
  CreateAdminPasswordResetRequest,
  CreateAdminUserRequest,
  CreatePasswordChangeRequest,
  ManagedUserRole,
  UpdateAdminUserRequest,
  UpdateUserProfileRequest,
  UserStatus,
  UserSummary,
} from '@/lib/types'

export type AdminUsersSearch = {
  page: number
  pageSize: number
  role?: ManagedUserRole
  status?: UserStatus
  username?: string
}

export type AdminUserCreateForm = {
  displayName: string
  email: string
  phone: string
  role: ManagedUserRole
  temporaryPassword: string
  username: string
}

export type AdminPasswordResetForm = {
  temporaryPassword: string
}

export type AdminRoleChangeForm = {
  role: ManagedUserRole
}

export type ProfileForm = {
  displayName: string
  email: string
  phone: string
}

export type PasswordChangeForm = {
  currentPassword: string
  newPassword: string
  newPasswordConfirmation: string
}

export const MANAGED_ROLE_OPTIONS: Array<{ label: string; value: ManagedUserRole }> = [
  { label: '普通用户', value: 'standard' },
  { label: '管理员', value: 'admin' },
]

export const USER_STATUS_OPTIONS: Array<{ label: string; value: UserStatus }> = [
  { label: '启用', value: 'active' },
  { label: '禁用', value: 'disabled' },
  { label: '锁定', value: 'locked' },
]

export const DEFAULT_ADMIN_USERS_SEARCH: AdminUsersSearch = {
  page: 1,
  pageSize: 20,
}

const PASSWORD_MIN_LENGTH = 8
const PASSWORD_MAX_LENGTH = 1024

const optionalEmailSchema = z
  .string()
  .trim()
  .refine((value) => value === '' || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value), {
    message: '请输入有效邮箱',
  })

const profileSchema = z.object({
  displayName: z.string().trim().max(100, '显示名不能超过 100 个字符'),
  email: optionalEmailSchema,
  phone: z.string().trim().max(32, '手机号不能超过 32 个字符'),
})

const passwordSchema = z
  .string()
  .min(PASSWORD_MIN_LENGTH, `密码不能少于 ${PASSWORD_MIN_LENGTH} 个字符`)
  .max(PASSWORD_MAX_LENGTH, `密码不能超过 ${PASSWORD_MAX_LENGTH} 个字符`)

const createAdminUserSchema = profileSchema.extend({
  role: z.enum(['standard', 'admin']),
  temporaryPassword: passwordSchema,
  username: z.string().trim().min(1, '请输入用户名').max(64, '用户名不能超过 64 个字符'),
})

const resetPasswordSchema = z.object({
  temporaryPassword: passwordSchema,
})

const roleChangeSchema = z.object({
  role: z.enum(['standard', 'admin']),
})

const passwordChangeSchema = z
  .object({
    currentPassword: z.string().min(1, '请输入当前临时密码'),
    newPassword: passwordSchema,
    newPasswordConfirmation: z.string().min(1, '请再次输入新密码'),
  })
  .refine((value) => value.newPassword === value.newPasswordConfirmation, {
    message: '两次输入的新密码不一致',
    path: ['newPasswordConfirmation'],
  })

export function normalizeAdminUsersSearch(search: Record<string, unknown>): AdminUsersSearch {
  const page = parsePositiveInt(search.page, DEFAULT_ADMIN_USERS_SEARCH.page)
  const pageSize = parsePageSize(search.pageSize, DEFAULT_ADMIN_USERS_SEARCH.pageSize)
  const role = parseManagedRole(search.role)
  const status = parseUserStatus(search.status)
  const username = typeof search.username === 'string' ? search.username.trim() : ''

  return {
    page,
    pageSize,
    ...(role ? { role } : {}),
    ...(status ? { status } : {}),
    ...(username ? { username } : {}),
  }
}

export function getAssignableRoles(user: UserSummary | null): ManagedUserRole[] {
  if (!user) return []
  if (hasAuthority(user, 'super_admin')) {
    return ['standard', 'admin']
  }
  if (hasAuthority(user, 'admin')) return ['standard']
  return []
}

export function canManageAdminUsers(user: UserSummary | null): boolean {
  return getAssignableRoles(user).length > 0
}

export function primaryManagedRole(user: AdminUser): ManagedUserRole | undefined {
  if (user.roles.some((role) => role.toLowerCase() === 'admin')) return 'admin'
  if (user.roles.some((role) => role.toLowerCase() === 'standard')) return 'standard'
  return undefined
}

export function roleLabel(role: string): string {
  if (role === 'admin') return '管理员'
  if (role === 'standard') return '普通用户'
  if (role === 'super_admin') return '超级管理员'
  return role
}

export function statusLabel(status?: string): string {
  if (status === 'active') return '启用'
  if (status === 'disabled') return '禁用'
  if (status === 'locked') return '锁定'
  return '未知'
}

export function validateCreateAdminUserForm(
  form: AdminUserCreateForm,
  allowedRoles: readonly ManagedUserRole[],
): { message?: string; request?: CreateAdminUserRequest; valid: boolean } {
  const parsed = createAdminUserSchema.safeParse(form)
  if (!parsed.success) return invalid(parsed.error.issues[0]?.message)
  if (!allowedRoles.includes(parsed.data.role)) return invalid('当前账号不能创建该角色')

  return {
    valid: true,
    request: {
      username: parsed.data.username.trim(),
      temporaryPassword: parsed.data.temporaryPassword,
      role: parsed.data.role,
      displayName: parsed.data.displayName.trim(),
      email: toNullableString(parsed.data.email),
      phone: toNullableString(parsed.data.phone),
    },
  }
}

export function validateProfileForm(form: ProfileForm): {
  message?: string
  request?: UpdateUserProfileRequest
  valid: boolean
} {
  const parsed = profileSchema.safeParse(form)
  if (!parsed.success) return invalid(parsed.error.issues[0]?.message)

  return {
    valid: true,
    request: {
      displayName: parsed.data.displayName.trim(),
      email: toNullableString(parsed.data.email),
      phone: toNullableString(parsed.data.phone),
    },
  }
}

export function validatePasswordChangeForm(form: PasswordChangeForm): {
  message?: string
  request?: CreatePasswordChangeRequest
  valid: boolean
} {
  const parsed = passwordChangeSchema.safeParse(form)
  if (!parsed.success) return invalid(parsed.error.issues[0]?.message)
  return { valid: true, request: parsed.data }
}

export function validatePasswordResetForm(form: AdminPasswordResetForm): {
  message?: string
  request?: CreateAdminPasswordResetRequest
  valid: boolean
} {
  const parsed = resetPasswordSchema.safeParse(form)
  if (!parsed.success) return invalid(parsed.error.issues[0]?.message)
  return { valid: true, request: parsed.data }
}

export function validateRoleChangeForm(
  form: AdminRoleChangeForm,
  allowedRoles: readonly ManagedUserRole[],
): { message?: string; request?: UpdateAdminUserRequest; valid: boolean } {
  const parsed = roleChangeSchema.safeParse(form)
  if (!parsed.success) return invalid(parsed.error.issues[0]?.message)
  if (!allowedRoles.includes(parsed.data.role)) return invalid('当前账号不能设置该角色')
  return { valid: true, request: { role: parsed.data.role } }
}

export function formatUserManagementError(error: unknown, fallback: string): string {
  if (error instanceof ApiError) {
    const details = error.fields
      ? Object.entries(error.fields).map(([field, message]) => `${field}: ${message}`)
      : []
    const requestId = error.requestId ? `（requestId: ${error.requestId}）` : ''
    return `${fallback}: ${[error.message, ...details].join('；')}${requestId}`
  }

  if (error instanceof Error) return `${fallback}: ${error.message}`
  return fallback
}

function invalid(message = '表单校验失败') {
  return { message, valid: false as const }
}

function toNullableString(value: string): string | null {
  const trimmed = value.trim()
  return trimmed ? trimmed : null
}

function parsePositiveInt(value: unknown, fallback: number): number {
  const parsed = typeof value === 'string' ? Number.parseInt(value, 10) : Number(value)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback
}

function parsePageSize(value: unknown, fallback: number): number {
  const parsed = parsePositiveInt(value, fallback)
  return [10, 20, 50].includes(parsed) ? parsed : fallback
}

function parseManagedRole(value: unknown): ManagedUserRole | undefined {
  return value === 'standard' || value === 'admin' ? value : undefined
}

function parseUserStatus(value: unknown): UserStatus | undefined {
  return value === 'active' || value === 'disabled' || value === 'locked' ? value : undefined
}
