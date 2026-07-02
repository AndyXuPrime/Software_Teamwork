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
})
