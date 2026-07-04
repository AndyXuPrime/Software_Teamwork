import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
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

function getButtonByText(pattern: RegExp) {
  const button = screen.getAllByRole('button').find((item) => pattern.test(item.textContent ?? ''))

  if (!button) {
    throw new Error(`Unable to find button matching ${pattern}`)
  }

  return button
}

function createTemplatesPageFetchMock() {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const request = input instanceof Request ? input : new Request(input, init)
    const url = new URL(request.url)

    if (request.method === 'POST' && url.pathname.endsWith('/report-materials')) {
      return jsonResponse({
        data: {
          category: '煤库存',
          createdAt: '2026-07-03T00:00:00Z',
          enabled: true,
          filename: 'inventory.pdf',
          id: 'mat-uploaded',
          materialName: '上传素材',
          materialType: 'technical_doc',
        },
        requestId: 'req-material-upload',
      })
    }
    if (request.method === 'POST' && url.pathname.endsWith('/report-templates')) {
      return jsonResponse({
        data: {
          createdAt: '2026-07-03T00:00:00Z',
          enabled: true,
          filename: 'uploaded.docx',
          id: 'tpl-uploaded',
          reportType: 'summer_peak_inspection',
          templateName: '上传模板',
          version: 1,
        },
        requestId: 'req-upload',
      })
    }
    if (url.pathname.endsWith('/report-types')) {
      return jsonResponse({
        data: [{ code: 'summer_peak_inspection', name: '迎峰度夏巡检' }],
        requestId: 'req-types',
      })
    }
    if (url.pathname.endsWith('/report-templates')) {
      return pageResponse([])
    }
    if (url.pathname.endsWith('/report-materials')) {
      return pageResponse([])
    }
    if (url.pathname.endsWith('/report-statistics/overview')) {
      return jsonResponse({
        data: { materialCount: 0, reportCount: 0, templateCount: 0 },
        requestId: 'req-overview',
      })
    }
    if (url.pathname.endsWith('/report-statistics/daily')) {
      return jsonResponse({ data: [], requestId: 'req-daily' })
    }

    return jsonResponse({ data: [], requestId: 'req-default' })
  })
}

function createTemplatesPageListFetchMock({
  materials = [],
  templates = [],
}: {
  materials?: unknown[]
  templates?: unknown[]
} = {}) {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const request = input instanceof Request ? input : new Request(input, init)
    const url = new URL(request.url)

    if (url.pathname.endsWith('/report-types')) {
      return jsonResponse({
        data: [{ code: 'summer_peak_inspection', name: '迎峰度夏巡检' }],
        requestId: 'req-types',
      })
    }
    if (url.pathname.endsWith('/report-templates')) {
      return pageResponse(templates)
    }
    if (url.pathname.endsWith('/report-materials')) {
      return pageResponse(materials)
    }
    if (url.pathname.endsWith('/report-statistics/overview')) {
      return jsonResponse({
        data: {
          materialCount: materials.length,
          reportCount: 0,
          templateCount: templates.length,
        },
        requestId: 'req-overview',
      })
    }
    if (url.pathname.endsWith('/report-statistics/daily')) {
      return jsonResponse({ data: [], requestId: 'req-daily' })
    }

    return jsonResponse({ data: [], requestId: 'req-default' })
  })
}

async function openUploadDialog(user: ReturnType<typeof userEvent.setup>) {
  renderWithProviders(<ReportTemplatesPage />)

  await screen.findByText('报告模板与素材')
  await user.click(screen.getByRole('button', { name: /上传模板/ }))
  return await screen.findByRole('dialog', { name: '上传报告模板' })
}

async function openMaterialUploadDialog(user: ReturnType<typeof userEvent.setup>) {
  renderWithProviders(<ReportTemplatesPage />)

  await screen.findByText('报告模板与素材')
  await user.click(screen.getByRole('button', { name: /上传素材/ }))
  return await screen.findByRole('dialog', { name: '上传专业素材' })
}

describe('ReportTemplatesPage', () => {
  it('shows upload actions inside both template and material sections', async () => {
    const user = userEvent.setup()
    vi.stubGlobal('fetch', createTemplatesPageListFetchMock())

    renderWithProviders(<ReportTemplatesPage />)

    await screen.findByText('报告模板与素材')
    const templateSection = screen.getByRole('heading', { name: '模板列表' }).closest('section')
    const materialSection = screen.getByRole('heading', { name: '专业素材' }).closest('section')

    expect(templateSection).not.toBeNull()
    expect(materialSection).not.toBeNull()

    const templateUpload = within(templateSection as HTMLElement).getByRole('button', {
      name: /上传模板/,
    })
    const materialUpload = within(materialSection as HTMLElement).getByRole('button', {
      name: /上传素材/,
    })

    expect(templateUpload).toBeVisible()
    expect(materialUpload).toBeVisible()
    expect(templateUpload).toHaveClass('bg-primary')
    expect(materialUpload).toHaveClass('bg-primary')

    await user.click(templateUpload)
    expect(await screen.findByRole('dialog', { name: '上传报告模板' })).toBeVisible()
    await user.keyboard('{Escape}')
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())

    await user.click(materialUpload)
    expect(await screen.findByRole('dialog', { name: '上传专业素材' })).toBeVisible()
  })

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

  it('opens the template structure dialog with Enter and closes it with Escape', async () => {
    const keyboard = userEvent.setup()
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates/tpl-a11y/structure')) {
        return jsonResponse({
          data: { outlineSchema: [], styleConfig: { density: 'compact' } },
          requestId: 'req-structure',
        })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([
          {
            createdAt: '2026-07-02T09:00:00Z',
            enabled: true,
            filename: 'a11y-template.docx',
            id: 'tpl-a11y',
            reportType: 'summer_peak_inspection',
            templateName: 'A11y template',
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

    expect(await screen.findByText('A11y template')).toBeVisible()
    const openStructureButton = getButtonByText(/结构|缁撴瀯/)
    openStructureButton.focus()
    expect(openStructureButton).toHaveFocus()
    await keyboard.keyboard('{Enter}')

    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveAccessibleName(/A11y template/)

    await keyboard.keyboard('{Escape}')
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
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

  it('hides template technical metadata from list rows', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init)
        const url = new URL(request.url)

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
              reportType: 'coal_inventory_audit',
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
      }),
    )

    renderWithProviders(<ReportTemplatesPage />)

    expect(await screen.findByText('真实模板')).toBeVisible()
    expect(
      screen.queryByText('coal_inventory_audit · v1 · real-template.docx'),
    ).not.toBeInTheDocument()
  })

  it('hides material technical metadata from list rows', async () => {
    vi.stubGlobal(
      'fetch',
      createTemplatesPageListFetchMock({
        materials: [
          {
            category: 'local-demo',
            createdAt: '2026-06-30T00:00:00Z',
            enabled: true,
            filename: 'coal-inventory.txt',
            id: 'mat-local-demo',
            materialName: '煤场盘点素材',
            materialType: 'text',
          },
        ],
      }),
    )

    renderWithProviders(<ReportTemplatesPage />)

    expect(await screen.findByText('煤场盘点素材')).toBeVisible()
    expect(screen.queryByText('local-demo · text')).not.toBeInTheDocument()
  })

  it('opens a material detail dialog with realistic seeded material copy', async () => {
    const user = userEvent.setup()
    vi.stubGlobal(
      'fetch',
      createTemplatesPageListFetchMock({
        materials: [
          {
            category: 'local-demo',
            createdAt: '2026-06-30T00:00:00Z',
            description: '用于本地联调的安全占位素材，不包含真实文件引用或生产内容。',
            enabled: true,
            filename: 'local-demo-inspection-notes.md',
            id: '22222222-2222-4222-8222-222222222201',
            materialName: '本地演示检查记录',
            materialType: 'text',
            tags: ['本地演示', '种子数据', '无文件引用'],
          },
        ],
      }),
    )

    renderWithProviders(<ReportTemplatesPage />)

    expect(await screen.findByText('煤场库存盘点工作底稿')).toBeVisible()
    expect(screen.queryByText('本地演示检查记录')).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '查看素材' }))

    const dialog = await screen.findByRole('dialog', { name: /素材详情/ })
    expect(within(dialog).getByText('煤场库存盘点工作底稿')).toBeVisible()
    expect(within(dialog).getByText('文件名')).toBeVisible()
    expect(within(dialog).getByText('煤场库存盘点工作底稿.md')).toBeVisible()
    expect(within(dialog).getByText('分类')).toBeVisible()
    expect(within(dialog).getByText('煤库存审计')).toBeVisible()
    expect(within(dialog).getByText('类型')).toBeVisible()
    expect(within(dialog).getByText('审计底稿')).toBeVisible()
    expect(within(dialog).getByText('描述')).toBeVisible()
    expect(within(dialog).getByText(/记录2024年12月31日煤场库存盘点口径/)).toBeVisible()
    expect(within(dialog).getByText('标签')).toBeVisible()
    expect(within(dialog).getByText('煤场库存、盘点差异、热值折算、保供风险')).toBeVisible()
    expect(within(dialog).queryByText(/local-demo|本地演示|无文件引用/)).not.toBeInTheDocument()
  })

  it('uploads professional materials as multipart form data', async () => {
    const user = userEvent.setup()
    const fetchMock = createTemplatesPageFetchMock()
    vi.stubGlobal('fetch', fetchMock)

    const dialog = await openMaterialUploadDialog(user)
    await user.type(within(dialog).getByLabelText('素材名称'), '煤场盘点素材')
    await user.type(within(dialog).getByLabelText('素材类型'), 'technical_doc')
    await user.type(within(dialog).getByLabelText('分类'), '煤库存')
    await user.type(within(dialog).getByLabelText('标签'), '煤场, 盘点')
    await user.upload(
      within(dialog).getByLabelText('素材文件'),
      new File(['material'], 'inventory.pdf', { type: 'application/pdf' }),
    )
    await user.click(within(dialog).getByRole('button', { name: '上传' }))

    await waitFor(() =>
      expect(
        fetchMock.mock.calls.some((call) => {
          const request = call[0] instanceof Request ? call[0] : new Request(call[0], call[1])
          return (
            request.method === 'POST' && new URL(request.url).pathname.endsWith('/report-materials')
          )
        }),
      ).toBe(true),
    )
    expect(await screen.findByText('素材上传成功，列表已刷新。')).toBeVisible()
  })

  it('deletes professional materials through the gateway material endpoint', async () => {
    const deletedPaths: string[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (request.method === 'DELETE' && url.pathname.endsWith('/report-materials/mat-real')) {
        deletedPaths.push(url.pathname)
        return new Response(null, { status: 204 })
      }
      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([
          {
            category: '煤库存',
            createdAt: '2026-06-30T00:00:00Z',
            enabled: true,
            filename: 'inventory.pdf',
            id: 'mat-real',
            materialName: '真实素材',
            materialType: 'technical_doc',
          },
        ])
      }
      if (url.pathname.endsWith('/report-statistics/overview')) {
        return jsonResponse({
          data: { materialCount: 1, reportCount: 0, templateCount: 0 },
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

    expect(await screen.findByText('真实素材')).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: '删除素材' }))
    fireEvent.click(screen.getByRole('button', { name: '确认删除' }))

    await waitFor(() => expect(deletedPaths).toEqual(['/api/v1/report-materials/mat-real']))
  })

  it('blocks legacy DOC template uploads before posting to the backend', async () => {
    const user = userEvent.setup()
    const fetchMock = createTemplatesPageFetchMock()
    vi.stubGlobal('fetch', fetchMock)

    const dialog = await openUploadDialog(user)
    const fileInput = within(dialog).getByLabelText('模板文件')

    expect(fileInput).toHaveAttribute(
      'accept',
      '.docx,application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    )

    await user.type(within(dialog).getByLabelText('模板名称'), '旧版模板')
    fireEvent.change(fileInput, {
      target: { files: [new File(['legacy'], 'legacy.doc', { type: 'application/msword' })] },
    })
    await user.click(within(dialog).getByRole('button', { name: '上传' }))

    expect(await within(dialog).findByText('仅支持上传 DOCX 模板文件。')).toBeVisible()
    expect(
      fetchMock.mock.calls.some((call) => {
        const request = call[0] instanceof Request ? call[0] : new Request(call[0], call[1])
        return (
          request.method === 'POST' && new URL(request.url).pathname.endsWith('/report-templates')
        )
      }),
    ).toBe(false)
  })

  it('blocks DOCX template uploads above the backend 32 MiB limit before posting', async () => {
    const user = userEvent.setup()
    const fetchMock = createTemplatesPageFetchMock()
    vi.stubGlobal('fetch', fetchMock)

    const dialog = await openUploadDialog(user)
    const oversizedFile = new File(['template'], 'large.docx', {
      type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    })
    Object.defineProperty(oversizedFile, 'size', { value: 32 * 1024 * 1024 + 1 })

    await user.type(within(dialog).getByLabelText('模板名称'), '超大模板')
    await user.upload(within(dialog).getByLabelText('模板文件'), oversizedFile)
    await user.click(within(dialog).getByRole('button', { name: '上传' }))

    expect(await within(dialog).findByText('模板文件不能超过 32 MiB。')).toBeVisible()
    expect(
      fetchMock.mock.calls.some((call) => {
        const request = call[0] instanceof Request ? call[0] : new Request(call[0], call[1])
        return (
          request.method === 'POST' && new URL(request.url).pathname.endsWith('/report-templates')
        )
      }),
    ).toBe(false)
  })
})
