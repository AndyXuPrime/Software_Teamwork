import { screen, waitFor } from '@testing-library/react'
import type { CSSProperties } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { renderWithProviders } from '@/test/render'

import { StatsOverviewPage } from './stats-overview'

vi.mock('@/components/ui/echarts', () => ({
  EChartsWrapper: ({ style }: { style?: CSSProperties }) => (
    <div data-testid="echarts-wrapper" style={style} />
  ),
}))

function jsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    status: init?.status ?? 200,
    statusText: init?.statusText,
  })
}

describe('StatsOverviewPage', () => {
  it('renders cross-service admin overview cards from Gateway admin overview', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/admin/overview')) {
        return jsonResponse({
          data: {
            totals: {
              chunkCount: 7358,
              documentCount: 62,
              knowledgeBaseCount: 2,
              qaCount: 23,
              reportRecordCount: 0,
              reportTemplateCount: 7,
              userCount: 1,
            },
            updatedAt: '2026-07-05T00:00:00Z',
          },
          requestId: 'req-admin-overview',
        })
      }

      if (url.pathname.endsWith('/qa-metrics/overview')) {
        return jsonResponse({
          data: {
            activeUsersToday: 1,
            avgLatencyMs: 8098,
            conversationCount: 19,
            documentCount: 62,
            knowledgeBaseCount: 2,
            todayQaCount: 23,
            totalQaCount: 23,
            totalQuestionCount: 23,
          },
          requestId: 'req-qa-overview',
        })
      }

      if (url.pathname.endsWith('/qa-metrics/trend')) {
        return jsonResponse({
          data: {
            days: 30,
            points: [{ count: 23, date: '2026-07-05' }],
          },
          requestId: 'req-qa-trend',
        })
      }

      if (url.pathname.endsWith('/qa-metrics/top-queries')) {
        return jsonResponse({ data: [], requestId: 'req-top-queries' })
      }

      if (url.pathname.endsWith('/qa-metrics/intent-distribution')) {
        return jsonResponse({ data: [], requestId: 'req-intents' })
      }

      return jsonResponse({ error: { code: 'not_found', message: 'not found' } }, { status: 404 })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<StatsOverviewPage />)

    expect(await screen.findByText('切片数')).toBeVisible()
    expect(screen.getByText('7358')).toBeVisible()
    expect(screen.getByText('报告模板')).toBeVisible()
    expect(screen.getByText('7')).toBeVisible()
    expect(screen.getByText('报告生成')).toBeVisible()
    expect(screen.getByText('0')).toBeVisible()
    expect(screen.getByText('用户数')).toBeVisible()
    expect(screen.getByText('知识库')).toBeVisible()
    expect(screen.getAllByText('2')[0]).toBeVisible()

    await waitFor(() => {
      const requestedPaths = fetchMock.mock.calls.map((call) => {
        const input = call[0]
        const url = input instanceof Request ? input.url : String(input)
        return new URL(url).pathname
      })
      expect(requestedPaths).toContain('/api/v1/admin/overview')
    })
  })
})
