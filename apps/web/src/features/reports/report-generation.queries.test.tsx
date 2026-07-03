import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  getReportEventsRefetchInterval,
  getReportJobRefetchInterval,
  getReportSectionsRefetchInterval,
  reportKeys,
  useCancelReportJob,
  useReportJobQuery,
  useReportSettingsQuery,
  useRetryReportJobMutation,
  useUpdateReportOutlineMutation,
  useUpdateReportSettingsMutation,
} from './report-generation.queries'
import type { ReportEvent, ReportJob, ReportOutline } from './report-generation.types'

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

function reportJob(overrides: Partial<ReportJob>): ReportJob {
  return {
    createdAt: '2026-07-02T11:58:00Z',
    id: 'job-test',
    jobType: 'outline_generation',
    reportId: 'rpt-test',
    status: 'running',
    ...overrides,
  }
}

function reportEvent(overrides: Partial<ReportEvent>): ReportEvent {
  return {
    createdAt: '2026-07-02T11:58:00Z',
    eventType: 'job.running',
    id: 'evt-test',
    reportId: 'rpt-test',
    ...overrides,
  }
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('report generation query hooks', () => {
  it('bounds failed job polling to a retry grace window', () => {
    vi.spyOn(Date, 'now').mockReturnValue(Date.parse('2026-07-02T12:00:00Z'))

    expect(
      getReportJobRefetchInterval(
        reportJob({ status: 'failed', finishedAt: '2026-07-02T11:57:01Z' }),
      ),
    ).toBe(8000)
    expect(
      getReportJobRefetchInterval(
        reportJob({ status: 'failed', finishedAt: '2026-07-02T11:56:59Z' }),
      ),
    ).toBe(false)
  })

  it('stops polling failed events after the retry grace window', () => {
    vi.spyOn(Date, 'now').mockReturnValue(Date.parse('2026-07-02T12:00:00Z'))

    expect(
      getReportEventsRefetchInterval([
        reportEvent({ eventType: 'job.failed', createdAt: '2026-07-02T11:57:01Z' }),
      ]),
    ).toBe(8000)
    expect(
      getReportEventsRefetchInterval([
        reportEvent({ eventType: 'job.failed', createdAt: '2026-07-02T11:56:59Z' }),
      ]),
    ).toBe(false)
    expect(
      getReportEventsRefetchInterval([
        reportEvent({ eventType: 'job.succeeded', createdAt: '2026-07-02T11:59:00Z' }),
      ]),
    ).toBe(false)
  })

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

  it('refreshes section data while a polled content job is still running', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>().mockResolvedValue(
        jsonResponse({
          data: {
            createdAt: '2026-07-03T08:00:00Z',
            id: 'job-live',
            jobType: 'content_generation',
            progress: { completed: 1, total: 4 },
            reportId: 'rpt-live',
            status: 'running',
          },
          requestId: 'req-job-live',
        }),
      ),
    )

    const queryClient = createQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )

    const { result } = renderHook(() => useReportJobQuery('job-live'), { wrapper })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    await waitFor(() =>
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: reportKeys.sections('rpt-live') }),
    )
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: reportKeys.events('rpt-live') })
  })

  it('polls section data while a matching report job is active', () => {
    expect(
      getReportSectionsRefetchInterval(
        'rpt-live',
        reportJob({ jobType: 'content_generation', reportId: 'rpt-live', status: 'running' }),
      ),
    ).toBe(3000)
    expect(
      getReportSectionsRefetchInterval(
        'rpt-live',
        reportJob({ jobType: 'content_generation', reportId: 'rpt-live', status: 'pending' }),
      ),
    ).toBe(3000)
    expect(
      getReportSectionsRefetchInterval(
        'rpt-live',
        reportJob({ jobType: 'outline_generation', reportId: 'rpt-live', status: 'running' }),
      ),
    ).toBe(false)
    expect(
      getReportSectionsRefetchInterval(
        'rpt-live',
        reportJob({ jobType: 'content_generation', reportId: 'rpt-live', status: 'succeeded' }),
      ),
    ).toBe(false)
    expect(
      getReportSectionsRefetchInterval(
        'rpt-live',
        reportJob({ jobType: 'content_generation', reportId: 'other-report', status: 'running' }),
      ),
    ).toBe(false)
  })

  it('reads report settings and invalidates settings after publishing model config', async () => {
    const patchBodies: unknown[] = []
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init)
        const url = new URL(request.url)

        if (request.method === 'GET' && url.pathname.endsWith('/report-settings')) {
          return jsonResponse({
            data: {
              llm: {
                model: 'gpt-report-current',
                profileId: 'mp-current',
                provider: 'ai-gateway',
              },
            },
            requestId: 'req-settings',
          })
        }

        if (request.method === 'PATCH' && url.pathname.endsWith('/report-settings')) {
          patchBodies.push(await request.clone().json())
          return jsonResponse({
            data: { updatedAt: '2026-07-03T08:00:00Z' },
            requestId: 'req-settings-update',
          })
        }

        return jsonResponse({ error: { code: 'not_found', message: 'not found' } }, { status: 404 })
      }),
    )

    const queryClient = createQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )

    const settingsHook = renderHook(() => useReportSettingsQuery(), { wrapper })

    await waitFor(() => expect(settingsHook.result.current.isSuccess).toBe(true))
    expect(settingsHook.result.current.data?.llm?.profileId).toBe('mp-current')

    const mutationHook = renderHook(() => useUpdateReportSettingsMutation(), { wrapper })
    await mutationHook.result.current.mutateAsync({
      llm: { profileId: 'mp-chat', provider: 'ai-gateway' },
    })

    expect(patchBodies).toEqual([{ llm: { profileId: 'mp-chat', provider: 'ai-gateway' } }])
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: reportKeys.settings() })
  })

  it('moves a retried terminal job back to pending in the local cache', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init)
        const url = new URL(request.url)

        if (request.method === 'POST' && url.pathname.endsWith('/report-jobs/job-retry/attempts')) {
          return jsonResponse({
            data: {
              attemptNumber: 3,
              createdAt: '2026-07-03T08:00:00Z',
              id: 'attempt-retry',
              jobId: 'job-retry',
              status: 'pending',
            },
            requestId: 'req-retry',
          })
        }

        return jsonResponse({ error: { code: 'not_found', message: 'not found' } }, { status: 404 })
      }),
    )

    const queryClient = createQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    queryClient.setQueryData(
      reportKeys.job('job-retry'),
      reportJob({
        error: { message: 'section failed' },
        finishedAt: '2026-07-03T07:59:00Z',
        id: 'job-retry',
        jobType: 'content_generation',
        progress: { completed: 3, total: 3 },
        reportId: 'rpt-retry',
        status: 'partial_succeeded',
      }),
    )
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )

    const mutationHook = renderHook(() => useRetryReportJobMutation(), { wrapper })
    await mutationHook.result.current.mutateAsync({
      jobId: 'job-retry',
      reportId: 'rpt-retry',
    })

    expect(queryClient.getQueryData<ReportJob>(reportKeys.job('job-retry'))).toMatchObject({
      error: undefined,
      finishedAt: undefined,
      progress: {},
      status: 'pending',
    })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: reportKeys.job('job-retry') })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: reportKeys.events('rpt-retry') })
  })

  it('moves a canceled running job to canceled in the local cache', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init)
        const url = new URL(request.url)

        if (request.method === 'PATCH' && url.pathname.endsWith('/report-jobs/job-cancel')) {
          return jsonResponse({
            data: {
              createdAt: '2026-07-03T08:00:00Z',
              finishedAt: '2026-07-03T08:01:00Z',
              id: 'job-cancel',
              jobType: 'content_generation',
              progress: { completed: 1, total: 4 },
              reportId: 'rpt-cancel',
              status: 'canceled',
            },
            requestId: 'req-cancel',
          })
        }

        return jsonResponse({ error: { code: 'not_found', message: 'not found' } }, { status: 404 })
      }),
    )

    const queryClient = createQueryClient()
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    queryClient.setQueryData(
      reportKeys.job('job-cancel'),
      reportJob({
        id: 'job-cancel',
        jobType: 'content_generation',
        progress: { completed: 1, total: 4 },
        reportId: 'rpt-cancel',
        status: 'running',
      }),
    )
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )

    const mutationHook = renderHook(() => useCancelReportJob(), { wrapper })
    await mutationHook.result.current.mutateAsync('job-cancel')

    expect(queryClient.getQueryData<ReportJob>(reportKeys.job('job-cancel'))).toMatchObject({
      finishedAt: '2026-07-03T08:01:00Z',
      status: 'canceled',
    })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: reportKeys.job('job-cancel') })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: reportKeys.events('rpt-cancel') })
  })

  it('updates outline cache and refreshes sections after saving an edited outline', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init)
        const url = new URL(request.url)

        if (
          request.method === 'PATCH' &&
          url.pathname.endsWith('/reports/rpt-outline-save/outlines/outline-save')
        ) {
          return jsonResponse({
            data: {
              createdAt: '2026-07-03T00:00:00Z',
              id: 'outline-save',
              isCurrent: true,
              manualEdited: true,
              reportId: 'rpt-outline-save',
              sections: [
                { id: 'node-edited', level: 1, numbering: '1', title: 'Edited section' },
                { id: 'node-added', level: 1, numbering: '2', title: 'Added section' },
              ],
              updatedAt: '2026-07-03T00:01:00Z',
              version: 2,
            },
            requestId: 'req-save-outline',
          })
        }

        return jsonResponse({ data: [], requestId: 'req-default' })
      }),
    )

    const queryClient = createQueryClient()
    const oldOutline: ReportOutline = {
      createdAt: '2026-07-03T00:00:00Z',
      id: 'outline-save',
      manualEdited: false,
      reportId: 'rpt-outline-save',
      sections: [{ id: 'node-original', level: 1, numbering: '1', title: 'Original section' }],
      updatedAt: '2026-07-03T00:00:00Z',
      version: 1,
    }
    queryClient.setQueryData<ReportOutline[]>(reportKeys.outlines('rpt-outline-save'), [oldOutline])
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries')
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )

    const { result } = renderHook(() => useUpdateReportOutlineMutation('rpt-outline-save'), {
      wrapper,
    })

    result.current.mutate({
      outlineId: 'outline-save',
      sections: [{ id: 'node-edited', level: 1, numbering: '1', title: 'Edited section' }],
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(
      queryClient.getQueryData<ReportOutline[]>(reportKeys.outlines('rpt-outline-save')),
    ).toEqual([
      expect.objectContaining({
        sections: [
          expect.objectContaining({ id: 'node-edited', title: 'Edited section' }),
          expect.objectContaining({ id: 'node-added', title: 'Added section' }),
        ],
        version: 2,
      }),
    ])
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: reportKeys.sections('rpt-outline-save'),
    })
  })
})
