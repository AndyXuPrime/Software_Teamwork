import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import {
  changeCurrentUserPassword,
  createAdminUser,
  getCurrentUserProfile,
  listAdminUsers,
  type ListAdminUsersParams,
  resetAdminUserPassword,
  updateAdminUser,
  updateCurrentUserProfile,
} from '@/api/auth'
import type {
  CreateAdminPasswordResetRequest,
  CreateAdminUserRequest,
  CreatePasswordChangeRequest,
  UpdateAdminUserRequest,
  UpdateUserProfileRequest,
} from '@/lib/types'

export const userManagementKeys = {
  all: ['auth', 'users'] as const,
  adminLists: () => [...userManagementKeys.all, 'admin-list'] as const,
  adminList: (params: ListAdminUsersParams) =>
    [...userManagementKeys.adminLists(), params] as const,
  profile: () => [...userManagementKeys.all, 'profile'] as const,
}

export function useAdminUsers(params: ListAdminUsersParams) {
  return useQuery({
    queryKey: userManagementKeys.adminList(params),
    queryFn: () => listAdminUsers(params),
  })
}

export function useCurrentUserProfile() {
  return useQuery({
    queryKey: userManagementKeys.profile(),
    queryFn: getCurrentUserProfile,
  })
}

export function useUpdateCurrentUserProfile() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (body: UpdateUserProfileRequest) => updateCurrentUserProfile(body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: userManagementKeys.profile() })
    },
  })
}

export function useChangeCurrentUserPassword() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (body: CreatePasswordChangeRequest) => changeCurrentUserPassword(body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: userManagementKeys.profile() })
    },
  })
}

export function useCreateAdminUser() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (body: CreateAdminUserRequest) => createAdminUser(body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: userManagementKeys.adminLists() })
    },
  })
}

export function useUpdateAdminUser() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ userId, body }: { body: UpdateAdminUserRequest; userId: string }) =>
      updateAdminUser(userId, body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: userManagementKeys.adminLists() })
    },
  })
}

export function useResetAdminUserPassword() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ body, userId }: { body: CreateAdminPasswordResetRequest; userId: string }) =>
      resetAdminUserPassword(userId, body),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: userManagementKeys.adminLists() })
    },
  })
}
