import { fireEvent, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { useAuthStore } from '@/stores/auth-store'
import { renderWithProviders } from '@/test/render'

import { ReportRecordsPage } from './page'

vi.mock('@tanstack/react-router', () => ({
  Link: ({ children }: { children?: ReactNode }) => <a href="/reports/generate">{children}</a>,
}))

function gatewayError(code: string, message: string, requestId: string, status = 503) {
  return new Response(JSON.stringify({ error: { code, message, requestId } }), {
    headers: { 'Content-Type': 'application/json' },
    status,
  })
}

describe('ReportRecordsPage', () => {
  it('shows gateway errors instead of local fallback report records', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn<typeof fetch>()
        .mockResolvedValue(
          gatewayError('dependency_error', 'Document reports unavailable', 'req-records'),
        ),
    )

    renderWithProviders(<ReportRecordsPage />)

    expect((await screen.findAllByText(/Document reports unavailable/))[0]).toBeVisible()
    expect(screen.getAllByText(/req-records/).length).toBeGreaterThan(0)
    expect(screen.queryByText('2026年迎峰度夏检查报告')).not.toBeInTheDocument()
  })

  it('keeps delete context visible and shows request id when report deletion fails', async () => {
    useAuthStore.setState({
      status: 'authenticated',
      user: {
        id: 'user-1',
        username: 'tester',
        roles: [],
        permissions: ['report:write'],
      },
      userName: 'tester',
    })
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (request.method === 'DELETE' && url.pathname.endsWith('/reports/rpt-real')) {
        return gatewayError('dependency_error', 'Delete dependency down', 'req-delete')
      }

      return new Response(
        JSON.stringify({
          data: [
            {
              createdAt: '2026-06-30T00:00:00Z',
              id: 'rpt-real',
              name: '真实报告记录',
              reportType: 'summer_peak_inspection',
              status: 'draft',
              updatedAt: '2026-06-30T00:00:00Z',
              year: 2026,
            },
          ],
          page: { page: 1, pageSize: 20, total: 1 },
          requestId: 'req-record-list',
        }),
        { headers: { 'Content-Type': 'application/json' } },
      )
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportRecordsPage />)

    expect(await screen.findByText('真实报告记录')).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: '删除报告' }))
    fireEvent.click(screen.getByRole('button', { name: '确认删除' }))

    expect(await screen.findByText(/Delete dependency down/)).toBeVisible()
    expect(screen.getByText(/req-delete/)).toBeVisible()
    expect(screen.getByText(/即将删除报告"真实报告记录"/)).toBeVisible()
  })
})
