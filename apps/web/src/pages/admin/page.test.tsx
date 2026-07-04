import { screen } from '@testing-library/react'
import type { AnchorHTMLAttributes, ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { UserSummary } from '@/lib/types'
import { useAuthStore } from '@/stores/auth-store'
import { useUiStore } from '@/stores/ui-store'
import { renderWithProviders } from '@/test/render'

import { AdminPage } from './page'

const routerMocks = vi.hoisted(() => ({
  pathname: '/admin/knowledge/search',
}))

vi.mock('@tanstack/react-router', () => ({
  Link: ({
    children,
    to,
    ...props
  }: AnchorHTMLAttributes<HTMLAnchorElement> & {
    children?: ReactNode
    to: string
  }) => (
    <a {...props} href={to}>
      {children}
    </a>
  ),
  Outlet: () => <section aria-label="admin workspace">Admin workspace</section>,
  useRouterState: (options?: {
    select?: (state: { location: { pathname: string } }) => unknown
  }) => {
    const state = { location: { pathname: routerMocks.pathname } }
    return options?.select ? options.select(state) : state
  },
}))

const adminUser: UserSummary = {
  id: 'admin-1',
  permissions: ['knowledge:read', 'system:admin'],
  roles: ['system:admin'],
  username: 'admin',
}

describe('AdminPage layout transitions', () => {
  beforeEach(() => {
    routerMocks.pathname = '/admin/knowledge/search'
    useAuthStore.setState({
      accessToken: 'opaque-test-token',
      error: null,
      status: 'authenticated',
      user: adminUser,
      userName: adminUser.username,
    })
    useUiStore.setState({ sidebarCollapsed: false })
  })

  it('keeps the management sidebar outside the page entrance animation', () => {
    renderWithProviders(<AdminPage />)

    const sidebar = screen.getByRole('heading', { name: '管理面板' }).closest('aside')
    const workspace = screen.getByRole('region', { name: 'admin workspace' })
    const contentMain = workspace.closest('main')

    expect(sidebar).toBeInstanceOf(HTMLElement)
    expect(contentMain).toBeInstanceOf(HTMLElement)
    expect(sidebar).not.toHaveClass('page-enter-right')
    expect(contentMain).toHaveClass('page-enter-right')
  })
})
