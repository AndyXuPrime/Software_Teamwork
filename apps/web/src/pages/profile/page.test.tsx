import { fireEvent, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { UserSummary } from '@/lib/types'
import { useAuthStore } from '@/stores/auth-store'
import { useThemeStore } from '@/stores/theme-store'
import { renderWithProviders } from '@/test/render'

import { ProfilePage } from './page'

function jsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    status: init?.status ?? 200,
    statusText: init?.statusText,
  })
}

const user: UserSummary = {
  id: 'user-1',
  displayName: '旧显示名',
  email: 'old@example.com',
  phone: null,
  permissions: ['qa:use'],
  roles: ['standard'],
  status: 'active',
  username: 'operator',
}

describe('ProfilePage', () => {
  it('loads current profile and updates only editable profile fields', async () => {
    useAuthStore.setState({
      accessToken: 'opaque-token',
      error: null,
      status: 'authenticated',
      user,
      userName: user.username,
    })

    const patchBodies: unknown[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (request.method === 'GET' && url.pathname.endsWith('/users/me/profile')) {
        return jsonResponse({
          data: {
            ...user,
            createdAt: '2026-07-02T01:00:00Z',
            updatedAt: '2026-07-02T02:00:00Z',
          },
          requestId: 'req-profile',
        })
      }

      if (request.method === 'PATCH' && url.pathname.endsWith('/users/me/profile')) {
        patchBodies.push(await request.clone().json())
        return jsonResponse({
          data: {
            ...user,
            displayName: '新显示名',
            email: null,
            phone: '13800000000',
            updatedAt: '2026-07-02T03:00:00Z',
          },
          requestId: 'req-profile-update',
        })
      }

      return jsonResponse({ data: null, requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ProfilePage />)

    expect(await screen.findByDisplayValue('旧显示名')).toBeVisible()
    expect(screen.getByText('权限说明')).toBeVisible()
    expect(screen.getByText(/当前账号为普通用户/)).toBeVisible()
    expect(screen.queryByText('qa:use')).not.toBeInTheDocument()
    expect(screen.getByRole('heading', { name: '界面外观' })).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: '深色模式' }))
    expect(useThemeStore.getState().mode).toBe('dark')

    fireEvent.change(screen.getByLabelText('显示名'), { target: { value: '新显示名' } })
    fireEvent.change(screen.getByLabelText('邮箱'), { target: { value: '' } })
    fireEvent.change(screen.getByLabelText('电话'), { target: { value: '13800000000' } })
    fireEvent.click(screen.getByRole('button', { name: '保存资料' }))

    await waitFor(() => expect(patchBodies).toHaveLength(1))
    expect(patchBodies[0]).toEqual({
      displayName: '新显示名',
      email: null,
      phone: '13800000000',
    })
    expect(useAuthStore.getState().user?.displayName).toBe('新显示名')
  })

  it('summarizes admin roles without exposing permission keys', async () => {
    const adminUser: UserSummary = {
      ...user,
      id: 'user-admin',
      permissions: ['knowledge:read', 'report:write', 'system:admin'],
      roles: ['admin'],
      username: 'admin-user',
    }

    useAuthStore.setState({
      accessToken: 'opaque-token',
      error: null,
      status: 'authenticated',
      user: adminUser,
      userName: adminUser.username,
    })

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init)
        const url = new URL(request.url)

        if (request.method === 'GET' && url.pathname.endsWith('/users/me/profile')) {
          return jsonResponse({
            data: adminUser,
            requestId: 'req-profile-admin',
          })
        }

        return jsonResponse({ data: null, requestId: 'req-default' })
      }),
    )

    renderWithProviders(<ProfilePage />)

    expect(await screen.findByText(/当前账号为管理员/)).toBeVisible()
    expect(screen.queryByText('knowledge:read')).not.toBeInTheDocument()
    expect(screen.queryByText('report:write')).not.toBeInTheDocument()
    expect(screen.queryByText('system:admin')).not.toBeInTheDocument()
    expect(screen.queryByText(/当前账号为普通用户/)).not.toBeInTheDocument()
    expect(screen.queryByText(/当前账号为系统管理员/)).not.toBeInTheDocument()
  })

  it('uses normalized role values for the standard role summary', async () => {
    const standardUser: UserSummary = {
      ...user,
      id: 'user-standard-spaced',
      permissions: [' QA:USE ', ' reports:write '],
      roles: [' standard '],
      username: 'standard-spaced-user',
    }

    useAuthStore.setState({
      accessToken: 'opaque-token',
      error: null,
      status: 'authenticated',
      user: standardUser,
      userName: standardUser.username,
    })

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init)
        const url = new URL(request.url)

        if (request.method === 'GET' && url.pathname.endsWith('/users/me/profile')) {
          return jsonResponse({
            data: standardUser,
            requestId: 'req-profile-standard-spaced',
          })
        }

        return jsonResponse({ data: null, requestId: 'req-default' })
      }),
    )

    renderWithProviders(<ProfilePage />)

    expect(await screen.findByText(/当前账号为普通用户/)).toBeVisible()
    expect(screen.queryByText(' QA:USE ')).not.toBeInTheDocument()
    expect(screen.queryByText(' reports:write ')).not.toBeInTheDocument()
  })

  it('uses bare system admin permission for the system management summary', async () => {
    const systemAdminUser: UserSummary = {
      ...user,
      id: 'user-system-admin',
      permissions: [' system:admin '],
      roles: [],
      username: 'system-admin-user',
    }

    useAuthStore.setState({
      accessToken: 'opaque-token',
      error: null,
      status: 'authenticated',
      user: systemAdminUser,
      userName: systemAdminUser.username,
    })

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init)
        const url = new URL(request.url)

        if (request.method === 'GET' && url.pathname.endsWith('/users/me/profile')) {
          return jsonResponse({
            data: systemAdminUser,
            requestId: 'req-profile-system-admin',
          })
        }

        return jsonResponse({ data: null, requestId: 'req-default' })
      }),
    )

    renderWithProviders(<ProfilePage />)

    expect(await screen.findByText(/当前账号为系统管理员/)).toBeVisible()
  })

  it('keeps the admin summary when the admin role includes system admin permission', async () => {
    const adminUser: UserSummary = {
      ...user,
      id: 'user-admin-system-permission',
      permissions: ['system:admin'],
      roles: ['admin'],
      username: 'admin-system-permission-user',
    }

    useAuthStore.setState({
      accessToken: 'opaque-token',
      error: null,
      status: 'authenticated',
      user: adminUser,
      userName: adminUser.username,
    })

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init)
        const url = new URL(request.url)

        if (request.method === 'GET' && url.pathname.endsWith('/users/me/profile')) {
          return jsonResponse({
            data: adminUser,
            requestId: 'req-profile-admin-system-permission',
          })
        }

        return jsonResponse({ data: null, requestId: 'req-default' })
      }),
    )

    renderWithProviders(<ProfilePage />)

    expect(await screen.findByText(/当前账号为管理员/)).toBeVisible()
    expect(screen.queryByText('system:admin')).not.toBeInTheDocument()
    expect(screen.queryByText(/当前账号为系统管理员/)).not.toBeInTheDocument()
  })

  it('shows an unassigned role summary when no role is available', async () => {
    const unassignedUser: UserSummary = {
      ...user,
      id: 'user-unassigned',
      permissions: [],
      roles: [],
      username: 'unassigned-user',
    }

    useAuthStore.setState({
      accessToken: 'opaque-token',
      error: null,
      status: 'authenticated',
      user: unassignedUser,
      userName: unassignedUser.username,
    })

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init)
        const url = new URL(request.url)

        if (request.method === 'GET' && url.pathname.endsWith('/users/me/profile')) {
          return jsonResponse({
            data: unassignedUser,
            requestId: 'req-profile-unassigned',
          })
        }

        return jsonResponse({ data: null, requestId: 'req-default' })
      }),
    )

    renderWithProviders(<ProfilePage />)

    expect(await screen.findByText(/当前账号暂未分配可用角色/)).toBeVisible()
  })

  it('shows a distinct summary for super admins', async () => {
    const superAdminUser: UserSummary = {
      ...user,
      id: 'user-super-admin',
      permissions: [],
      roles: ['super_admin'],
      username: 'super-admin-user',
    }

    useAuthStore.setState({
      accessToken: 'opaque-token',
      error: null,
      status: 'authenticated',
      user: superAdminUser,
      userName: superAdminUser.username,
    })

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init)
        const url = new URL(request.url)

        if (request.method === 'GET' && url.pathname.endsWith('/users/me/profile')) {
          return jsonResponse({
            data: superAdminUser,
            requestId: 'req-profile-super-admin',
          })
        }

        return jsonResponse({ data: null, requestId: 'req-default' })
      }),
    )

    renderWithProviders(<ProfilePage />)

    expect(await screen.findByText(/当前账号为超级管理员/)).toBeVisible()
    expect(screen.queryByText(/当前账号为管理员/)).not.toBeInTheDocument()
  })
})
