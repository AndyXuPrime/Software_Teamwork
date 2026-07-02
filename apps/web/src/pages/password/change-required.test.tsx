import { fireEvent, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { UserSummary } from '@/lib/types'
import { useAuthStore } from '@/stores/auth-store'
import { usePageTransitionStore } from '@/stores/page-transition-store'
import { renderWithProviders } from '@/test/render'

import { PasswordChangeRequiredPage } from './change-required'

const navigate = vi.fn()

vi.mock('@tanstack/react-router', () => ({
  useRouter: () => ({ navigate }),
}))

function jsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    status: init?.status ?? 200,
    statusText: init?.statusText,
  })
}

const mustChangeUser: UserSummary = {
  id: 'user-1',
  permissions: ['qa:use'],
  roles: ['standard'],
  mustChangePassword: true,
  username: 'operator',
}

describe('PasswordChangeRequiredPage', () => {
  beforeEach(() => {
    navigate.mockReset()
    useAuthStore.setState({
      accessToken: 'opaque-token',
      error: null,
      status: 'authenticated',
      user: mustChangeUser,
      userName: mustChangeUser.username,
    })
  })

  it('rejects short or mismatched new passwords before submitting', async () => {
    const fetchMock = vi.fn<typeof fetch>()
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<PasswordChangeRequiredPage />)

    fireEvent.change(screen.getByLabelText('当前临时密码'), {
      target: { value: 'temporary-password' },
    })
    fireEvent.change(screen.getByLabelText('新密码'), { target: { value: 'short' } })
    fireEvent.change(screen.getByLabelText('确认新密码'), { target: { value: 'different' } })
    fireEvent.click(screen.getByRole('button', { name: '修改密码' }))

    expect(fetchMock).not.toHaveBeenCalled()
    expect(await screen.findByRole('alert')).toHaveTextContent('密码不能少于')
  })

  it('reveals any pending page transition overlay when mounted directly', async () => {
    vi.useFakeTimers()
    usePageTransitionStore.setState({ visible: true, covering: true })

    renderWithProviders(<PasswordChangeRequiredPage />)

    await vi.advanceTimersByTimeAsync(500)
    expect(usePageTransitionStore.getState().visible).toBe(false)
    expect(usePageTransitionStore.getState().covering).toBe(false)
    vi.useRealTimers()
  })

  it('submits current temporary password and clears must-change state after success', async () => {
    const requestBodies: unknown[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)
      expect(request.method).toBe('POST')
      expect(url.pathname).toBe('/api/v1/users/me/password-changes')
      requestBodies.push(await request.clone().json())
      return jsonResponse({
        data: {
          ...mustChangeUser,
          mustChangePassword: false,
          status: 'active',
        },
        requestId: 'req-password-change',
      })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<PasswordChangeRequiredPage />)

    fireEvent.change(screen.getByLabelText('当前临时密码'), {
      target: { value: 'temporary-password' },
    })
    fireEvent.change(screen.getByLabelText('新密码'), { target: { value: 'new-password' } })
    fireEvent.change(screen.getByLabelText('确认新密码'), {
      target: { value: 'new-password' },
    })
    fireEvent.click(screen.getByRole('button', { name: '修改密码' }))

    await waitFor(() => expect(requestBodies).toHaveLength(1))
    expect(requestBodies[0]).toEqual({
      currentPassword: 'temporary-password',
      newPassword: 'new-password',
      newPasswordConfirmation: 'new-password',
    })
    expect(useAuthStore.getState().user?.mustChangePassword).toBe(false)
    expect(navigate).toHaveBeenCalledWith({ to: '/' })
  })

  it('lets a must-change-password user log out from the forced change screen', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)
      expect(request.method).toBe('DELETE')
      expect(url.pathname).toBe('/api/v1/sessions/current')
      return new Response(null, { status: 204 })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<PasswordChangeRequiredPage />)

    fireEvent.click(screen.getByRole('button', { name: '退出登录' }))

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))
    expect(useAuthStore.getState().status).toBe('anonymous')
    expect(useAuthStore.getState().accessToken).toBeNull()
    expect(navigate).toHaveBeenCalledWith({ to: '/login' })
  })

  it('clears local session and navigates to login even when logout is blocked', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined)
    const fetchMock = vi.fn(async () =>
      jsonResponse(
        {
          error: {
            code: 'forbidden',
            message: 'password change required',
            requestId: 'req-forced-change',
          },
        },
        { status: 403 },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<PasswordChangeRequiredPage />)

    fireEvent.click(screen.getByRole('button', { name: '退出登录' }))

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1))
    expect(useAuthStore.getState().status).toBe('anonymous')
    expect(useAuthStore.getState().accessToken).toBeNull()
    expect(navigate).toHaveBeenCalledWith({ to: '/login' })
    warn.mockRestore()
  })
})
