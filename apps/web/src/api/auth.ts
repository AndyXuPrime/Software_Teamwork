import type {
  CreateSessionRequest,
  CreateUserRequest,
  SessionSummary,
  UserSummary,
} from '@/lib/types'

import { gatewayRequest, requestVoid } from './client'

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

export function deleteCurrentSession(): Promise<void> {
  return requestVoid('/sessions/current', { method: 'DELETE' })
}
