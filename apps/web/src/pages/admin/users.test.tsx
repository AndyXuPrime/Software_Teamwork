import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { AdminUser, UserSummary } from '@/lib/types'
import { useAuthStore } from '@/stores/auth-store'
import { renderWithProviders } from '@/test/render'

import { AdminUsersPage } from './users'

const routerMocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  search: { page: 1, pageSize: 20 } as Record<string, unknown>,
}))

vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => routerMocks.navigate,
  useSearch: () => routerMocks.search,
}))

function jsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    status: init?.status ?? 200,
    statusText: init?.statusText,
  })
}

const superUser: UserSummary = {
  id: 'super-1',
  permissions: [],
  roles: ['super_admin'],
  username: 'root',
}

const adminUser: UserSummary = {
  id: 'admin-1',
  permissions: [],
  roles: ['admin'],
  username: 'manager',
}

const managedStandardUser: AdminUser = {
  id: 'user-1',
  username: 'operator',
  displayName: 'Operator',
  email: null,
  phone: null,
  status: 'active',
  mustChangePassword: false,
  roles: ['standard'],
  permissions: ['qa:use'],
  createdAt: '2026-07-02T01:00:00Z',
  updatedAt: '2026-07-02T02:00:00Z',
  manageableRoles: ['standard', 'admin'],
  actions: {
    canChangeRole: true,
    canDisable: true,
    canEnable: false,
    canResetPassword: true,
  },
}

function seedAuth(user: UserSummary) {
  useAuthStore.setState({
    accessToken: 'opaque-token',
    error: null,
    status: 'authenticated',
    user,
    userName: user.username,
  })
}

describe('AdminUsersPage', () => {
  beforeEach(() => {
    routerMocks.navigate.mockReset()
    routerMocks.search = { page: 1, pageSize: 20 }
  })

  it('loads the list with Gateway query params and offers super-admin role options only', async () => {
    seedAuth(superUser)
    routerMocks.search = {
      page: 2,
      pageSize: 10,
      role: 'admin',
      status: 'active',
      username: 'lin',
    }
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)
      expect(url.pathname).toBe('/api/v1/admin/users')
      expect(url.searchParams.get('page')).toBe('2')
      expect(url.searchParams.get('pageSize')).toBe('10')
      expect(url.searchParams.get('role')).toBe('admin')
      expect(url.searchParams.get('status')).toBe('active')
      expect(url.searchParams.get('username')).toBe('lin')
      return jsonResponse({
        data: [managedStandardUser],
        page: { page: 2, pageSize: 10, total: 11 },
        requestId: 'req-users',
      })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<AdminUsersPage />)

    expect(await screen.findByText('operator')).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: '新建用户' }))

    const dialog = document.querySelector('[data-slot="dialog-content"]')
    expect(dialog).toBeTruthy()
    const roleSelect = within(dialog as HTMLElement).getByLabelText('角色') as HTMLSelectElement
    expect(Array.from(roleSelect.options).map((option) => option.value)).toEqual([
      'standard',
      'admin',
    ])
    expect(screen.queryByText('超级管理员')).not.toBeInTheDocument()
  })

  it('limits regular admins to standard-user creation and sends administrator temporary password', async () => {
    seedAuth(adminUser)
    const requestBodies: unknown[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)
      if (request.method === 'GET') {
        return jsonResponse({
          data: [managedStandardUser],
          page: { page: 1, pageSize: 20, total: 1 },
          requestId: 'req-users',
        })
      }
      expect(request.method).toBe('POST')
      expect(url.pathname).toBe('/api/v1/admin/users')
      requestBodies.push(await request.clone().json())
      return jsonResponse({ data: managedStandardUser, requestId: 'req-create' }, { status: 201 })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<AdminUsersPage />)
    expect(await screen.findByText('operator')).toBeVisible()

    fireEvent.click(screen.getByRole('button', { name: '新建用户' }))
    const dialog = document.querySelector('[data-slot="dialog-content"]') as HTMLElement
    const roleSelect = within(dialog).getByLabelText('角色') as HTMLSelectElement
    expect(Array.from(roleSelect.options).map((option) => option.value)).toEqual(['standard'])

    fireEvent.change(within(dialog).getByLabelText('用户名'), { target: { value: 'new-user' } })
    fireEvent.change(within(dialog).getByLabelText('临时密码'), {
      target: { value: 'temporary-password' },
    })
    fireEvent.click(within(dialog).getByRole('button', { name: '创建' }))

    await waitFor(() => expect(requestBodies).toHaveLength(1))
    expect(requestBodies[0]).toMatchObject({
      email: null,
      phone: null,
      role: 'standard',
      temporaryPassword: 'temporary-password',
      username: 'new-user',
    })
  })

  it('navigates with server-side filter params instead of client-side filtering', async () => {
    seedAuth(superUser)
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>().mockResolvedValue(
        jsonResponse({
          data: [],
          page: { page: 1, pageSize: 20, total: 0 },
          requestId: 'req-empty',
        }),
      ),
    )

    renderWithProviders(<AdminUsersPage />)
    expect(await screen.findByText('暂无可管理用户')).toBeVisible()

    fireEvent.change(screen.getByLabelText('用户名搜索'), { target: { value: 'lin' } })
    fireEvent.change(screen.getByLabelText('角色'), { target: { value: 'admin' } })
    fireEvent.change(screen.getByLabelText('状态'), { target: { value: 'disabled' } })
    fireEvent.click(screen.getByRole('button', { name: '查询' }))

    expect(routerMocks.navigate).toHaveBeenCalledWith({
      to: '/admin/users',
      search: {
        page: 1,
        pageSize: 20,
        role: 'admin',
        status: 'disabled',
        username: 'lin',
      },
    })
  })
})
