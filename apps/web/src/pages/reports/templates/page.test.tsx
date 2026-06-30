import { fireEvent, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { renderWithProviders } from '@/test/render'

import { ReportTemplatesPage } from './page'

function gatewayError(code: string, message: string, requestId: string, status = 503) {
  return new Response(JSON.stringify({ error: { code, message, requestId } }), {
    headers: { 'Content-Type': 'application/json' },
    status,
  })
}

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json' },
  })
}

function pageResponse(data: unknown[]) {
  return jsonResponse({
    data,
    page: { page: 1, pageSize: 20, total: data.length },
    requestId: 'req-page',
  })
}

describe('ReportTemplatesPage', () => {
  it('shows gateway errors instead of local fallback templates or materials', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn<typeof fetch>()
        .mockImplementation(async () =>
          gatewayError('dependency_error', 'Document templates unavailable', 'req-templates'),
        ),
    )

    renderWithProviders(<ReportTemplatesPage />)

    expect((await screen.findAllByText(/Document templates unavailable/))[0]).toBeVisible()
    expect(screen.getAllByText(/req-templates/).length).toBeGreaterThan(0)
    expect(screen.queryByText('迎峰度夏默认模板')).not.toBeInTheDocument()
    expect(screen.queryByText('设备运行台账与缺陷闭环记录')).not.toBeInTheDocument()
  })

  it('keeps delete context visible and shows request id when template deletion fails', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (request.method === 'DELETE' && url.pathname.endsWith('/report-templates/tpl-real')) {
        return gatewayError(
          'dependency_error',
          'Template delete dependency down',
          'req-template-delete',
        )
      }
      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([
          {
            createdAt: '2026-06-30T00:00:00Z',
            enabled: true,
            filename: 'real-template.docx',
            id: 'tpl-real',
            reportType: 'summer_peak_inspection',
            templateName: '真实模板',
            version: 1,
          },
        ])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([])
      }
      if (url.pathname.endsWith('/report-statistics/overview')) {
        return jsonResponse({
          data: { materialCount: 0, reportCount: 1, templateCount: 1 },
          requestId: 'req-overview',
        })
      }
      if (url.pathname.endsWith('/report-statistics/daily')) {
        return jsonResponse({ data: [], requestId: 'req-daily' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportTemplatesPage />)

    expect(await screen.findByText('真实模板')).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: '删除模板' }))
    fireEvent.click(screen.getByRole('button', { name: '确认删除' }))

    expect(await screen.findByText(/Template delete dependency down/)).toBeVisible()
    expect(screen.getByText(/req-template-delete/)).toBeVisible()
    expect(screen.getByText(/即将删除模板"真实模板"/)).toBeVisible()
  })
})
