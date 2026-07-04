import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type * as ApiClientModule from '@/api/client'
import { streamGateway } from '@/api/client'

import { useReportEventStream } from './use-report-event-stream'

vi.mock('@/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof ApiClientModule>()
  return {
    ...actual,
    streamGateway: vi.fn(() => ({ abort: vi.fn(), signal: new AbortController().signal })),
  }
})

const mockedStreamGateway = vi.mocked(streamGateway)

type StreamOptions = Parameters<typeof streamGateway>[1]

function reportEventFrame(overrides: { eventType: string; jobId?: string; message?: string }): {
  data: string
  event: 'report.event'
} {
  return {
    event: 'report.event',
    data: JSON.stringify({
      createdAt: '2026-07-04T00:00:00Z',
      eventType: overrides.eventType,
      id: `evt-${overrides.eventType}`,
      jobId: overrides.jobId,
      message: overrides.message,
      reportId: 'report-1',
    }),
  }
}

afterEach(() => {
  vi.clearAllMocks()
  vi.unstubAllGlobals()
})

function installAnimationFrameHarness() {
  let nextId = 1
  const callbacks = new Map<number, FrameRequestCallback>()
  const request = vi.fn((callback: FrameRequestCallback) => {
    const id = nextId++
    callbacks.set(id, callback)
    return id
  })
  const cancel = vi.fn((id: number) => {
    callbacks.delete(id)
  })

  vi.stubGlobal('requestAnimationFrame', request)
  vi.stubGlobal('cancelAnimationFrame', cancel)

  return {
    step(now: number) {
      const current = [...callbacks.values()]
      callbacks.clear()
      for (const callback of current) callback(now)
    },
  }
}

function stepUntil(
  read: () => string,
  expected: string,
  raf: ReturnType<typeof installAnimationFrameHarness>,
) {
  for (let frame = 0; frame < 40 && read() !== expected; frame += 1) {
    act(() => {
      raf.step(frame * 34)
    })
  }
}

describe('useReportEventStream', () => {
  it('renders streamed outline JSON as readable titles without exposing raw JSON', () => {
    const raf = installAnimationFrameHarness()
    const { result } = renderHook(() =>
      useReportEventStream({ enabled: true, jobId: 'job-1', reportId: 'report-1' }),
    )

    const streamOptions = mockedStreamGateway.mock.calls[0]?.[1] as StreamOptions

    expect(mockedStreamGateway.mock.calls[0]?.[0]).toBe(
      '/reports/report-1/events/stream?jobId=job-1',
    )

    act(() => {
      streamOptions.onEvent(
        reportEventFrame({
          eventType: 'outline.delta',
          jobId: 'job-1',
          message: '{"sections":[{"title":"Overview"},{"title":"Risk analysis"}]}',
        }),
      )
    })

    stepUntil(() => result.current.outlineText, 'Overview\nRisk analysis', raf)

    expect(result.current.outlineText).toBe('Overview\nRisk analysis')
    expect(result.current.outlineText).not.toContain('{')
    expect(result.current.outlineText).not.toContain('sections')
  })

  it('renders streamed section JSON as body text without exposing raw JSON', () => {
    const raf = installAnimationFrameHarness()
    const { result } = renderHook(() =>
      useReportEventStream({ enabled: true, jobId: 'job-1', reportId: 'report-1' }),
    )

    const streamOptions = mockedStreamGateway.mock.calls[0]?.[1] as StreamOptions

    act(() => {
      streamOptions.onEvent(
        reportEventFrame({
          eventType: 'section.delta',
          jobId: 'job-1',
          message: JSON.stringify({
            sectionId: 'section-1',
            text: '{"content":"Generated body from KB","tables":[]}',
          }),
        }),
      )
    })

    stepUntil(
      () => result.current.sectionTextById['section-1'] ?? '',
      'Generated body from KB',
      raf,
    )

    expect(result.current.sectionTextById['section-1']).toBe('Generated body from KB')
    expect(result.current.sectionTextById['section-1']).not.toContain('{')
    expect(result.current.sectionTextById['section-1']).not.toContain('content')
  })

  it('does not append streamed section table JSON to the body textarea text', () => {
    const raf = installAnimationFrameHarness()
    const { result } = renderHook(() =>
      useReportEventStream({ enabled: true, jobId: 'job-1', reportId: 'report-1' }),
    )

    const streamOptions = mockedStreamGateway.mock.calls[0]?.[1] as StreamOptions

    act(() => {
      streamOptions.onEvent(
        reportEventFrame({
          eventType: 'section.delta',
          jobId: 'job-1',
          message: JSON.stringify({
            sectionId: 'section-1',
            text: JSON.stringify({
              content: 'Only narrative body',
              tables: [
                {
                  headers: ['Metric', 'Value'],
                  rows: [['Peak load', '102 MW']],
                  footnote: 'table-only evidence',
                },
              ],
            }),
          }),
        }),
      )
    })

    stepUntil(() => result.current.sectionTextById['section-1'] ?? '', 'Only narrative body', raf)

    expect(result.current.sectionTextById['section-1']).toBe('Only narrative body')
    expect(result.current.sectionTextById['section-1']).not.toContain('Metric')
    expect(result.current.sectionTextById['section-1']).not.toContain('Peak load')
    expect(result.current.sectionTextById['section-1']).not.toContain('table-only evidence')
  })

  it('continues typing readable body text from partial JSON chunks', () => {
    const raf = installAnimationFrameHarness()
    const { result } = renderHook(() =>
      useReportEventStream({ enabled: true, jobId: 'job-1', reportId: 'report-1' }),
    )

    const streamOptions = mockedStreamGateway.mock.calls[0]?.[1] as StreamOptions

    act(() => {
      streamOptions.onEvent(
        reportEventFrame({
          eventType: 'section.delta',
          jobId: 'job-1',
          message: JSON.stringify({
            sectionId: 'section-1',
            text: '{"content":"Generated ',
          }),
        }),
      )
    })

    stepUntil(() => result.current.sectionTextById['section-1'] ?? '', 'Generated', raf)
    expect(result.current.sectionTextById['section-1']).toBe('Generated')

    act(() => {
      streamOptions.onEvent(
        reportEventFrame({
          eventType: 'section.delta',
          jobId: 'job-1',
          message: JSON.stringify({
            sectionId: 'section-1',
            text: 'body text","tables":[]}',
          }),
        }),
      )
    })

    stepUntil(() => result.current.sectionTextById['section-1'] ?? '', 'Generated body text', raf)

    expect(result.current.sectionTextById['section-1']).toBe('Generated body text')
    expect(result.current.sectionTextById['section-1']).not.toContain('{')
  })

  it('types visible outline text through animation frames instead of showing it all at once', () => {
    const raf = installAnimationFrameHarness()
    const { result } = renderHook(() =>
      useReportEventStream({ enabled: true, jobId: 'job-1', reportId: 'report-1' }),
    )

    const streamOptions = mockedStreamGateway.mock.calls[0]?.[1] as StreamOptions

    act(() => {
      streamOptions.onEvent(
        reportEventFrame({
          eventType: 'outline.delta',
          jobId: 'job-1',
          message: 'abcdef',
        }),
      )
    })

    expect(result.current.outlineText).toBe('')

    act(() => {
      raf.step(0)
    })
    expect(result.current.outlineText).toBe('ab')

    act(() => {
      raf.step(34)
    })
    expect(result.current.outlineText).toBe('abcd')
  })

  it('ignores stale deltas after the active report job changes', () => {
    const abortFirst = vi.fn()
    const abortSecond = vi.fn()
    mockedStreamGateway
      .mockReturnValueOnce({ abort: abortFirst, signal: new AbortController().signal })
      .mockReturnValueOnce({ abort: abortSecond, signal: new AbortController().signal })

    const { result, rerender } = renderHook(
      ({ jobId }: { jobId: string }) =>
        useReportEventStream({ enabled: true, jobId, reportId: 'report-1' }),
      { initialProps: { jobId: 'job-1' } },
    )

    const firstStreamOptions = mockedStreamGateway.mock.calls[0]?.[1] as StreamOptions

    rerender({ jobId: 'job-2' })

    act(() => {
      firstStreamOptions.onEvent(
        reportEventFrame({
          eventType: 'section.delta',
          jobId: 'job-1',
          message: '{"sectionId":"section-1","text":"stale body"}',
        }),
      )
    })

    expect(abortFirst).toHaveBeenCalledTimes(1)
    expect(result.current.sectionText).toBe('')
  })

  it('keeps user-canceled streams from being marked done by late callbacks', () => {
    const abort = vi.fn()
    mockedStreamGateway.mockReturnValueOnce({
      abort,
      signal: new AbortController().signal,
    })

    const { result } = renderHook(() =>
      useReportEventStream({ enabled: true, jobId: 'job-1', reportId: 'report-1' }),
    )

    const streamOptions = mockedStreamGateway.mock.calls[0]?.[1] as StreamOptions

    act(() => {
      result.current.abort()
    })
    act(() => {
      streamOptions.onDone?.()
    })

    expect(abort).toHaveBeenCalledTimes(1)
    expect(result.current.status).not.toBe('done')
  })

  it('marks document SSE error events as errors and ignores the following EOF', () => {
    const abort = vi.fn()
    mockedStreamGateway.mockReturnValueOnce({
      abort,
      signal: new AbortController().signal,
    })

    const { result } = renderHook(() =>
      useReportEventStream({ enabled: true, jobId: 'job-1', reportId: 'report-1' }),
    )

    const streamOptions = mockedStreamGateway.mock.calls[0]?.[1] as StreamOptions

    act(() => {
      streamOptions.onEvent(
        reportEventFrame({
          eventType: 'section.delta',
          jobId: 'job-1',
          message: JSON.stringify({
            sectionId: 'section-1',
            text: '{"content":"partial body"}',
          }),
        }),
      )
    })
    act(() => {
      streamOptions.onEvent({
        data: JSON.stringify({
          code: 'dependency_error',
          message: 'event stream failed',
        }),
        event: 'error',
      })
    })
    act(() => {
      streamOptions.onDone?.()
    })

    expect(abort).toHaveBeenCalledTimes(1)
    expect(result.current.status).toBe('error')
    expect(result.current.error).toMatchObject({
      code: 'dependency_error',
      message: 'event stream failed',
    })
    expect(result.current.sectionTextById['section-1']).toBe('partial body')
  })
})
