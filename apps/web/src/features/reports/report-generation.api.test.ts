import { describe, expect, it, vi } from 'vitest'

import { createReportJobAttempt, deleteReport, deleteReportTemplate } from './report-generation.api'

describe('report generation API wrappers', () => {
  it('treats report and template DELETE 204 responses as success', async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(deleteReport('rpt-real')).resolves.toBeUndefined()
    await expect(deleteReportTemplate('tpl-real')).resolves.toBeUndefined()

    const reportRequest = fetchMock.mock.calls[0]?.[0]
    const templateRequest = fetchMock.mock.calls[1]?.[0]
    expect(reportRequest).toBeInstanceOf(Request)
    expect(templateRequest).toBeInstanceOf(Request)
    expect(reportRequest instanceof Request ? reportRequest.method : undefined).toBe('DELETE')
    expect(templateRequest instanceof Request ? templateRequest.method : undefined).toBe('DELETE')
    expect(reportRequest instanceof Request ? new URL(reportRequest.url).pathname : undefined).toBe(
      '/api/v1/reports/rpt-real',
    )
    expect(
      templateRequest instanceof Request ? new URL(templateRequest.url).pathname : undefined,
    ).toBe('/api/v1/report-templates/tpl-real')
  })

  it('returns report job attempts from the retry endpoint', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>().mockResolvedValue(
        new Response(
          JSON.stringify({
            data: {
              attemptNumber: 2,
              createdAt: '2026-06-30T00:00:00Z',
              id: 'attempt-2',
              jobId: 'job-real',
              status: 'running',
            },
            requestId: 'req-attempt',
          }),
          {
            headers: { 'Content-Type': 'application/json' },
            status: 202,
          },
        ),
      ),
    )

    await expect(createReportJobAttempt('job-real')).resolves.toMatchObject({
      attemptNumber: 2,
      id: 'attempt-2',
      jobId: 'job-real',
      status: 'running',
    })
  })
})
