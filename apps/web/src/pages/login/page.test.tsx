import { fireEvent, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { apiClient } from '@/api/client'
import { useAuthStore } from '@/stores/auth-store'
import { renderWithProviders } from '@/test/render'

import { LoginPage } from './page'

const navigate = vi.fn()

vi.mock('@tanstack/react-router', () => ({
  useRouter: () => ({ navigate }),
}))

function mockSessionResponse() {
  vi.stubGlobal(
    'fetch',
    vi.fn<typeof fetch>().mockResolvedValue(
      new Response(
        JSON.stringify({
          data: {
            session: {
              accessToken: 'opaque-access-token',
              createdAt: '2026-06-30T00:00:00Z',
              expiresAt: '2026-07-01T00:00:00Z',
              id: 'session-1',
              userId: 'user-1',
            },
            user: {
              id: 'user-1',
              permissions: ['qa:use'],
              roles: ['operator'],
              username: 'operator',
            },
          },
          requestId: 'req-login',
        }),
        { headers: { 'Content-Type': 'application/json' }, status: 200 },
      ),
    ),
  )
}

describe('LoginPage', () => {
  beforeEach(() => {
    navigate.mockReset()
  })

  it('submits credentials through the gateway session API and stores only the opaque token', async () => {
    mockSessionResponse()
    const { container } = renderWithProviders(<LoginPage />)

    fireEvent.change(container.querySelector<HTMLInputElement>('#username')!, {
      target: { value: 'operator' },
    })
    fireEvent.change(container.querySelector<HTMLInputElement>('#password')!, {
      target: { value: 'secret' },
    })
    fireEvent.click(container.querySelector<HTMLButtonElement>('button[type="submit"]')!)

    await waitFor(() => expect(navigate).toHaveBeenCalledWith({ to: '/' }))

    const fetchRequest = (fetch as unknown as ReturnType<typeof vi.fn>).mock
      .calls[0]?.[0] as Request
    expect(fetchRequest.url).toContain('/api/v1/sessions')
    expect(fetchRequest.headers.get('Authorization')).toBeNull()
    expect(await fetchRequest.json()).toEqual({ password: 'secret', username: 'operator' })
    expect(apiClient.getToken()).toBe('opaque-access-token')
    expect(useAuthStore.getState().userName).toBe('operator')
  })

  it('blocks empty credentials before hitting the network', async () => {
    const fetchMock = vi.fn<typeof fetch>()
    vi.stubGlobal('fetch', fetchMock)
    const { container } = renderWithProviders(<LoginPage />)

    fireEvent.click(container.querySelector<HTMLButtonElement>('button[type="submit"]')!)

    expect(fetchMock).not.toHaveBeenCalled()
    expect(await screen.findByText(/输入|杈撳叆/)).toBeVisible()
  })
})
