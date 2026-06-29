import type { UserSummary } from './types'

export type PermissionRequirement = {
  all?: string[]
  any?: string[]
  roles?: string[]
}

function normalize(value: string): string {
  return value.trim().toLowerCase()
}

export function hasPermission(permissions: readonly string[], permission: string): boolean {
  return permissions.map(normalize).includes(normalize(permission))
}

export function hasRole(roles: readonly string[], role: string): boolean {
  return roles.map(normalize).includes(normalize(role))
}

export function canAccess(user: UserSummary | null, requirement?: PermissionRequirement): boolean {
  if (!requirement) return Boolean(user)
  if (!user) return false

  const roleOk =
    !requirement.roles?.length || requirement.roles.some((role) => hasRole(user.roles, role))
  const allOk =
    !requirement.all?.length ||
    requirement.all.every((permission) => hasPermission(user.permissions, permission))
  const anyOk =
    !requirement.any?.length ||
    requirement.any.some((permission) => hasPermission(user.permissions, permission))

  return roleOk && allOk && anyOk
}
