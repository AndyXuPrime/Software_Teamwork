import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { reportKeys, useReportJobQuery } from './report-generation.queries'

function jsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    status: init?.status ?? 200,
    statusText: init?.statusText,
  })
}

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  })
}

describe('report generation query hooks', () => {
  it('refreshes report data once when a polled job reaches a terminal status', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>().mockResolvedValue(
        jsonResponse({
          data: {
            createdAt: '2026-06-30T00:00:00Z',
            finishedAt: '2026-06-30T00:01:00Z',
            id: 'job-done',
            jobType: 'outline_generation',
            reportId: 'rpt-real',
            status: 'succeeded',
          },
          requestId: 'req-job',
        }),
      ),
    )

    const queryClient = createQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )

    const { result, rerender } = renderHook(() => useReportJobQuery('job-done'), { wrapper })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    await waitFor(() => expect(invalidateSpy).toHaveBeenCalledTimes(5))

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: reportKeys.outlines('rpt-real') })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: reportKeys.sections('rpt-real') })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: reportKeys.detail('rpt-real') })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: reportKeys.records() })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: reportKeys.events('rpt-real') })

    rerender()

    expect(invalidateSpy).toHaveBeenCalledTimes(5)
  })
})
