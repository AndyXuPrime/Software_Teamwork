import { describe, expect, it, vi } from 'vitest'

import {
  cancelReportJob,
  createReportJobAttempt,
  createReportTemplate,
  deleteReport,
  deleteReportTemplate,
  getReportSettings,
  updateReportSettings,
} from './report-generation.api'

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

  it('uploads report templates as multipart form data', async () => {
    const appendSpy = vi.spyOn(FormData.prototype, 'append')
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      expect(request.method).toBe('POST')
      expect(url.pathname).toBe('/api/v1/report-templates')
      expect(request.headers.get('Content-Type')).not.toBe('application/json')
      expect(request.headers.get('Content-Type')).toContain('multipart/form-data')

      const form = new Map(appendSpy.mock.calls.map(([key, value]) => [key, value]))
      expect(form.get('templateName')).toBe('巡检模板')
      expect(form.get('reportType')).toBe('inspection')
      expect(form.get('description')).toBe('现场巡检')
      const file = form.get('file')
      expect(file).toMatchObject({
        type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
      })
      expect(typeof file === 'object' && file && 'size' in file ? file.size : 0).toBeGreaterThan(0)

      return new Response(
        JSON.stringify({
          data: {
            createdAt: '2026-07-03T08:00:00Z',
            enabled: true,
            filename: 'inspection.docx',
            id: 'tpl-uploaded',
            reportType: 'inspection',
            templateName: '巡检模板',
            version: 1,
          },
          requestId: 'req-template-upload',
        }),
        {
          headers: { 'Content-Type': 'application/json' },
          status: 201,
        },
      )
    })
    vi.stubGlobal('fetch', fetchMock)

    await expect(
      createReportTemplate({
        description: '现场巡检',
        file: new File(['template'], 'inspection.docx', {
          type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        }),
        reportType: 'inspection',
        templateName: '巡检模板',
      }),
    ).resolves.toMatchObject({
      id: 'tpl-uploaded',
      templateName: '巡检模板',
    })
  })

  it('cancels a report job through the gateway report job update contract', async () => {
    const bodies: unknown[] = []
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init)
        bodies.push(await request.clone().json())
        return new Response(
          JSON.stringify({
            data: {
              createdAt: '2026-07-03T08:00:00Z',
              finishedAt: '2026-07-03T08:01:00Z',
              id: 'job-real',
              jobType: 'content_generation',
              progress: { completed: 1, total: 4 },
              reportId: 'rpt-real',
              status: 'canceled',
            },
            requestId: 'req-cancel',
          }),
          {
            headers: { 'Content-Type': 'application/json' },
            status: 200,
          },
        )
      }),
    )

    await expect(cancelReportJob('job-real')).resolves.toMatchObject({
      id: 'job-real',
      reportId: 'rpt-real',
      status: 'canceled',
    })

    const fetchMock = fetch as unknown as ReturnType<typeof vi.fn>
    const request = fetchMock.mock.calls[0]?.[0]
    expect(request).toBeInstanceOf(Request)
    expect(request instanceof Request ? request.method : undefined).toBe('PATCH')
    expect(request instanceof Request ? new URL(request.url).pathname : undefined).toBe(
      '/api/v1/report-jobs/job-real',
    )
    expect(bodies).toEqual([{ status: 'canceled' }])
  })

  it('reads and updates report generation settings through the gateway contract', async () => {
    const patchBodies: unknown[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (request.method === 'GET' && url.pathname.endsWith('/report-settings')) {
        return new Response(
          JSON.stringify({
            data: {
              file: { defaultFormat: 'docx', defaultNumberingMode: 'global' },
              llm: {
                model: 'gpt-report',
                profileId: 'mp-current',
                provider: 'ai-gateway',
                timeoutSeconds: 60,
              },
            },
            requestId: 'req-settings',
          }),
          {
            headers: { 'Content-Type': 'application/json' },
            status: 200,
          },
        )
      }

      if (request.method === 'PATCH' && url.pathname.endsWith('/report-settings')) {
        patchBodies.push(await request.clone().json())
        return new Response(
          JSON.stringify({
            data: { updatedAt: '2026-07-03T08:00:00Z' },
            requestId: 'req-settings-update',
          }),
          {
            headers: { 'Content-Type': 'application/json' },
            status: 200,
          },
        )
      }

      return new Response(JSON.stringify({ error: { code: 'not_found', message: 'not found' } }), {
        headers: { 'Content-Type': 'application/json' },
        status: 404,
      })
    })
    vi.stubGlobal('fetch', fetchMock)

    await expect(getReportSettings()).resolves.toMatchObject({
      llm: {
        model: 'gpt-report',
        profileId: 'mp-current',
        provider: 'ai-gateway',
      },
    })

    await expect(
      updateReportSettings({ llm: { profileId: 'mp-chat', provider: 'ai-gateway' } }),
    ).resolves.toEqual({ updatedAt: '2026-07-03T08:00:00Z' })

    expect(patchBodies).toEqual([{ llm: { profileId: 'mp-chat', provider: 'ai-gateway' } }])
    expect(patchBodies[0]).not.toHaveProperty('apiKey')
    expect(patchBodies[0]).not.toHaveProperty('baseUrl')
  })
})
