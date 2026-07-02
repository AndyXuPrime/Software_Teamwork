import type {
  AdminUser,
  CreateAdminPasswordResetRequest,
  CreateAdminUserRequest,
  CreatePasswordChangeRequest,
  CreateSessionRequest,
  CreateUserRequest,
  ManagedUserRole,
  SessionSummary,
  UpdateAdminUserRequest,
  UpdateUserProfileRequest,
  UserProfile,
  UserStatus,
  UserSummary,
} from '@/lib/types'

import {
  buildQuery,
  type GatewayPage,
  gatewayPageRequest,
  gatewayRequest,
  requestVoid,
} from './client'

export type AuthSessionResult = {
  user: UserSummary
  session: SessionSummary
}

export function createSession(body: CreateSessionRequest): Promise<AuthSessionResult> {
  return gatewayRequest<AuthSessionResult>('/sessions', {
    method: 'POST',
    body,
    token: null,
  })
}

export function createUserSession(body: CreateUserRequest): Promise<AuthSessionResult> {
  return gatewayRequest<AuthSessionResult>('/users', {
    method: 'POST',
    body,
    token: null,
  })
}

export function getCurrentUser(): Promise<UserSummary> {
  return gatewayRequest<UserSummary>('/users/me')
}

export function getCurrentUserProfile(): Promise<UserProfile> {
  return gatewayRequest<UserProfile>('/users/me/profile')
}

export function updateCurrentUserProfile(body: UpdateUserProfileRequest): Promise<UserProfile> {
  return gatewayRequest<UserProfile>('/users/me/profile', {
    method: 'PATCH',
    body,
  })
}

export function changeCurrentUserPassword(body: CreatePasswordChangeRequest): Promise<UserProfile> {
  return gatewayRequest<UserProfile>('/users/me/password-changes', {
    method: 'POST',
    body,
  })
}

export function deleteCurrentSession(): Promise<void> {
  return requestVoid('/sessions/current', { method: 'DELETE' })
}

export type ListAdminUsersParams = {
  page?: number
  pageSize?: number
  role?: ManagedUserRole | ''
  status?: UserStatus | ''
  username?: string
}

export type AdminUsersPage = {
  items: AdminUser[]
  page: GatewayPage
}

export function listAdminUsers(params: ListAdminUsersParams = {}): Promise<AdminUsersPage> {
  return gatewayPageRequest<AdminUser>(
    `/admin/users${buildQuery({
      page: params.page,
      pageSize: params.pageSize,
      role: params.role || undefined,
      status: params.status || undefined,
      username: params.username?.trim() || undefined,
    })}`,
  )
}

export function createAdminUser(body: CreateAdminUserRequest): Promise<AdminUser> {
  return gatewayRequest<AdminUser>('/admin/users', {
    method: 'POST',
    body,
  })
}

export function updateAdminUser(userId: string, body: UpdateAdminUserRequest): Promise<AdminUser> {
  return gatewayRequest<AdminUser>(`/admin/users/${encodeURIComponent(userId)}`, {
    method: 'PATCH',
    body,
  })
}

export function resetAdminUserPassword(
  userId: string,
  body: CreateAdminPasswordResetRequest,
): Promise<AdminUser> {
  return gatewayRequest<AdminUser>(`/admin/users/${encodeURIComponent(userId)}/password-resets`, {
    method: 'POST',
    body,
  })
}
