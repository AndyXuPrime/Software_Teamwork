import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { AnchorHTMLAttributes, ReactNode, Ref } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { UserSummary } from '@/lib/types'
import { useAuthStore } from '@/stores/auth-store'
import { renderWithProviders } from '@/test/render'

import { AppLayout } from './app-layout'

const routerMocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  pathname: '/chat',
}))

vi.mock('@tanstack/react-router', () => ({
  Link: ({
    activeProps: _activeProps,
    children,
    inactiveProps: _inactiveProps,
    ref,
    to,
    ...props
  }: AnchorHTMLAttributes<HTMLAnchorElement> & {
    activeProps?: unknown
    children?: ReactNode
    inactiveProps?: unknown
    ref?: Ref<HTMLAnchorElement>
    to: string
  }) => (
    <a
      {...props}
      href={to}
      ref={ref}
      onClick={(event) => {
        event.preventDefault()
        routerMocks.navigate({ to })
      }}
    >
      {children}
    </a>
  ),
  useRouter: () => ({ navigate: routerMocks.navigate }),
  useRouterState: () => ({ location: { pathname: routerMocks.pathname } }),
}))

const user: UserSummary = {
  id: 'user-1',
  permissions: [
    'qa:use',
    'report:read',
    'report:write',
    'knowledge:read',
    'knowledge:write',
    'system:admin',
  ],
  roles: ['system:admin'],
  username: 'operator',
}

const standardUser: UserSummary = {
  id: 'user-2',
  permissions: ['qa:use', 'knowledge:read', 'document:upload'],
  roles: ['standard'],
  username: 'standard',
}

const reportUser: UserSummary = {
  ...standardUser,
  permissions: ['qa:use', 'knowledge:read', 'report:read'],
}

describe('AppLayout accessibility smoke', () => {
  beforeEach(() => {
    routerMocks.navigate.mockReset()
    routerMocks.pathname = '/chat'
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) =>
      window.setTimeout(() => callback(0), 0),
    )
    vi.stubGlobal('cancelAnimationFrame', (id: number) => window.clearTimeout(id))
    useAuthStore.setState({
      accessToken: 'opaque-test-token',
      error: null,
      status: 'authenticated',
      user,
      userName: user.username,
    })
  })

  it('keeps top navigation links reachable and activatable from the keyboard', async () => {
    const keyboard = userEvent.setup()

    renderWithProviders(
      <AppLayout>
        <section aria-label="workspace">Workspace</section>
      </AppLayout>,
    )

    const nav = screen.getByRole('navigation')
    const navLinks = within(nav).getAllByRole('link')
    const logoutButton = screen.getByRole('button')

    expect(navLinks).toHaveLength(3)
    navLinks.forEach((link) => {
      expect(link).toHaveAccessibleName(/.+/)
    })
    expect(logoutButton).toHaveAccessibleName(/.+/)

    await keyboard.tab()
    expect(navLinks[0]).toHaveFocus()
    await keyboard.tab()
    expect(navLinks[1]).toHaveFocus()
    await keyboard.keyboard('{Enter}')
    expect(routerMocks.navigate).toHaveBeenCalledWith({ to: '/reports' })
    await keyboard.tab()
    expect(navLinks[2]).toHaveFocus()
    await keyboard.tab()
    expect(screen.getByRole('link', { name: '打开个人资料' })).toHaveFocus()
    await keyboard.tab()
    expect(logoutButton).toHaveFocus()
  })

  it('does not expose the admin shell to standard users', () => {
    useAuthStore.setState({
      accessToken: 'opaque-test-token',
      error: null,
      status: 'authenticated',
      user: standardUser,
      userName: standardUser.username,
    })

    renderWithProviders(
      <AppLayout>
        <section aria-label="workspace">Workspace</section>
      </AppLayout>,
    )

    const nav = screen.getByRole('navigation')
    expect(within(nav).getByRole('link', { name: '问答' })).toBeVisible()
    expect(within(nav).queryByRole('link', { name: '管理' })).not.toBeInTheDocument()
  })

  it('exposes the admin shell to users with admin report routes', () => {
    useAuthStore.setState({
      accessToken: 'opaque-test-token',
      error: null,
      status: 'authenticated',
      user: reportUser,
      userName: reportUser.username,
    })

    renderWithProviders(
      <AppLayout>
        <section aria-label="workspace">Workspace</section>
      </AppLayout>,
    )

    const nav = screen.getByRole('navigation')
    expect(within(nav).getByRole('link', { name: '管理' })).toBeVisible()
  })
})
