import { act, fireEvent, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { reportKeys } from '@/features/reports'
import type { UserSummary } from '@/lib/types'
import { useAuthStore } from '@/stores/auth-store'
import { renderWithProviders } from '@/test/render'

import { ReportGeneratePage } from './page'

function jsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    status: init?.status ?? 200,
    statusText: init?.statusText,
  })
}

function gatewayError(code: string, message: string, requestId: string, status = 503) {
  return jsonResponse({ error: { code, message, requestId } }, { status })
}

function pageResponse(data: unknown[]) {
  return jsonResponse({
    data,
    page: { page: 1, pageSize: 20, total: data.length },
    requestId: 'req-page',
  })
}

function sseResponse(frames: string[]) {
  const encoder = new TextEncoder()
  return new Response(
    new ReadableStream<Uint8Array>({
      start(controller) {
        frames.forEach((frame) => controller.enqueue(encoder.encode(frame)))
        controller.close()
      },
    }),
    {
      headers: { 'Content-Type': 'text/event-stream' },
      status: 200,
    },
  )
}

function createUser(permissions: string[]): UserSummary {
  return {
    id: 'user-1',
    permissions,
    roles: permissions.includes('system:admin') ? ['system:admin'] : [],
    username: 'operator',
  }
}

function setAuthenticatedUser(permissions: string[]) {
  useAuthStore.setState({
    accessToken: 'token',
    error: null,
    status: 'authenticated',
    user: createUser(permissions),
    userName: 'operator',
  })
}

function deferredResponse<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => {
    resolve = done
  })
  return { promise, resolve }
}

const reportType = {
  code: 'summer_peak_inspection',
  defaultTemplateId: 'tpl-real',
  description: '真实服务返回的报告类型',
  enabled: true,
  name: '真实巡检报告',
}

const coalReportType = {
  code: 'coal_inventory_audit',
  defaultTemplateId: 'tpl-coal',
  description: '煤库存审计报告类型',
  enabled: true,
  name: '煤库存审计报告',
}

const reportTemplate = {
  createdAt: '2026-06-30T00:00:00Z',
  enabled: true,
  filename: 'real-template.docx',
  id: 'tpl-real',
  reportType: 'summer_peak_inspection',
  templateName: '真实模板',
  version: 1,
}

const coalReportTemplate = {
  ...reportTemplate,
  id: 'tpl-coal',
  reportType: 'coal_inventory_audit',
  templateName: '煤库存审计模板',
}

const reportMaterial = {
  category: '真实素材',
  createdAt: '2026-06-30T00:00:00Z',
  enabled: true,
  id: 'mat-real',
  materialName: '真实素材',
  materialType: 'technical_doc',
}

const legacySeedReportMaterial = {
  category: 'local-demo',
  createdAt: '2026-06-30T00:00:00Z',
  description: '用于本地联调的安全占位素材，不包含真实文件引用或生产内容。',
  enabled: true,
  filename: 'local-demo-inspection-notes.md',
  id: '22222222-2222-4222-8222-222222222201',
  materialName: '本地演示检查记录',
  materialType: 'text',
  tags: ['本地演示', '种子数据', '无文件引用'],
}

const operationsKnowledgeBase = {
  chunkCount: 12,
  chunkStrategy: { chunkSize: 1600, overlap: 200, type: 'SEMANTIC_TEXT' },
  createdAt: '2026-07-01T00:00:00Z',
  description: 'Operational procedures',
  docType: 'policy',
  documentCount: 3,
  id: 'kb-ops',
  name: 'Operations KB',
  retrievalStrategy: { mode: 'VECTOR', scoreThreshold: 0.35, topK: 10 },
  updatedAt: '2026-07-01T00:00:00Z',
}

const safetyKnowledgeBase = {
  ...operationsKnowledgeBase,
  description: 'Safety evidence',
  id: 'kb-safety',
  name: 'Safety KB',
}

async function createReportAndReachOutlineStep() {
  const trigger = screen.getAllByRole('combobox')[0]!
  await waitFor(() => expect(trigger).not.toBeDisabled())
  fireEvent.click(trigger)
  const option = await screen.findByRole('option', { name: reportType.name })
  fireEvent.click(option)
  await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())
  fireEvent.click(screen.getByRole('button', { name: /创建草稿/ }))
  await waitFor(() => expect(screen.getByRole('button', { name: /生成正文/ })).toBeEnabled())
}

function createKnowledgeSelectionFetchMock(options: {
  contentJobStatus?: 'running' | 'succeeded'
  contentBodies: Record<string, unknown>[]
  events?: unknown[]
  knowledgeBases?: unknown[]
  knowledgeError?: boolean
  outlineJobStatus?: 'running' | 'succeeded'
  outlines?: unknown[]
  sectionBodies?: Record<string, unknown>[]
  sections?: unknown[]
  streamFrames?: string[]
}) {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const request = input instanceof Request ? input : new Request(input, init)
    const url = new URL(request.url)
    const outlineJobStatus = options.outlineJobStatus ?? 'succeeded'

    if (url.pathname.endsWith('/report-types')) {
      return jsonResponse({ data: [reportType], requestId: 'req-types' })
    }
    if (url.pathname.endsWith('/report-templates')) {
      return pageResponse([reportTemplate])
    }
    if (url.pathname.endsWith('/report-materials')) {
      return pageResponse([reportMaterial])
    }
    if (url.pathname.endsWith('/knowledge-bases')) {
      if (options.knowledgeError) {
        return gatewayError('dependency_error', 'Knowledge list unavailable', 'req-knowledge')
      }
      return pageResponse(options.knowledgeBases ?? [operationsKnowledgeBase, safetyKnowledgeBase])
    }
    if (request.method === 'POST' && url.pathname.endsWith('/reports')) {
      return jsonResponse({
        data: {
          id: 'rpt-knowledge-selection',
          name: 'knowledge-selection-report',
          reportType: 'summer_peak_inspection',
          status: 'draft',
          templateId: 'tpl-real',
        },
        requestId: 'req-create-report',
      })
    }
    if (
      request.method === 'POST' &&
      url.pathname.endsWith('/reports/rpt-knowledge-selection/jobs')
    ) {
      const body = (await request.clone().json()) as Record<string, unknown>
      if (body.jobType === 'section_regeneration') {
        options.sectionBodies?.push(body)
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:03:00Z',
            id: 'job-section-knowledge',
            jobType: 'section_regeneration',
            progress: { completed: 0, total: 1 },
            reportId: 'rpt-knowledge-selection',
            status: 'running',
            targetId: (body.target as { sectionId?: string } | undefined)?.sectionId,
            targetType: 'section',
          },
          requestId: 'req-section-job',
        })
      }
      if (body.jobType === 'content_generation') {
        const contentJobStatus = options.contentJobStatus ?? 'running'
        options.contentBodies.push(body)
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:02:00Z',
            finishedAt: contentJobStatus === 'succeeded' ? '2026-07-03T00:02:30Z' : undefined,
            id: 'job-content-knowledge',
            jobType: 'content_generation',
            progress:
              contentJobStatus === 'succeeded'
                ? { completed: 1, total: 1 }
                : { completed: 0, total: 1 },
            reportId: 'rpt-knowledge-selection',
            status: contentJobStatus,
          },
          requestId: 'req-content-job',
        })
      }
      return jsonResponse({
        data: {
          createdAt: '2026-07-03T00:00:00Z',
          finishedAt: outlineJobStatus === 'succeeded' ? '2026-07-03T00:01:00Z' : undefined,
          id: 'job-outline-knowledge',
          jobType: 'outline_generation',
          progress:
            outlineJobStatus === 'succeeded'
              ? { completed: 1, total: 1 }
              : { completed: 0, total: 1 },
          reportId: 'rpt-knowledge-selection',
          status: outlineJobStatus,
        },
        requestId: 'req-outline-job',
      })
    }
    if (url.pathname.endsWith('/report-jobs/job-outline-knowledge')) {
      return jsonResponse({
        data: {
          createdAt: '2026-07-03T00:00:00Z',
          finishedAt: outlineJobStatus === 'succeeded' ? '2026-07-03T00:01:00Z' : undefined,
          id: 'job-outline-knowledge',
          jobType: 'outline_generation',
          progress:
            outlineJobStatus === 'succeeded'
              ? { completed: 1, total: 1 }
              : { completed: 0, total: 1 },
          reportId: 'rpt-knowledge-selection',
          status: outlineJobStatus,
        },
        requestId: 'req-outline-status',
      })
    }
    if (url.pathname.endsWith('/report-jobs/job-content-knowledge')) {
      const contentJobStatus = options.contentJobStatus ?? 'running'
      return jsonResponse({
        data: {
          createdAt: '2026-07-03T00:02:00Z',
          finishedAt: contentJobStatus === 'succeeded' ? '2026-07-03T00:02:30Z' : undefined,
          id: 'job-content-knowledge',
          jobType: 'content_generation',
          progress:
            contentJobStatus === 'succeeded'
              ? { completed: 1, total: 1 }
              : { completed: 0, total: 1 },
          reportId: 'rpt-knowledge-selection',
          status: contentJobStatus,
        },
        requestId: 'req-content-status',
      })
    }
    if (url.pathname.endsWith('/report-jobs/job-section-knowledge')) {
      return jsonResponse({
        data: {
          createdAt: '2026-07-03T00:03:00Z',
          id: 'job-section-knowledge',
          jobType: 'section_regeneration',
          progress: { completed: 0, total: 1 },
          reportId: 'rpt-knowledge-selection',
          status: 'running',
          targetId: 'section-knowledge',
          targetType: 'section',
        },
        requestId: 'req-section-status',
      })
    }
    if (url.pathname.endsWith('/reports/rpt-knowledge-selection/outlines')) {
      return jsonResponse({
        data: options.outlines ?? [
          {
            createdAt: '2026-07-03T00:00:00Z',
            id: 'outline-knowledge',
            isCurrent: true,
            reportId: 'rpt-knowledge-selection',
            sections: [
              { id: 'node-knowledge', level: 1, numbering: '1', title: 'Knowledge section' },
            ],
            source: 'ai',
            version: 1,
          },
        ],
        requestId: 'req-outlines',
      })
    }
    if (url.pathname.endsWith('/reports/rpt-knowledge-selection/sections')) {
      return jsonResponse({ data: options.sections ?? [], requestId: 'req-sections' })
    }
    if (url.pathname.endsWith('/reports/rpt-knowledge-selection/events/stream')) {
      return sseResponse(options.streamFrames ?? [])
    }
    if (url.pathname.endsWith('/reports/rpt-knowledge-selection/events')) {
      return jsonResponse({ data: options.events ?? [], requestId: 'req-events' })
    }

    return jsonResponse({ data: [], requestId: 'req-default' })
  })
}

describe('ReportGeneratePage', () => {
  it('aligns draft defaults with the selected report type without overwriting custom text', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [coalReportType, reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([
          url.searchParams.get('reportType') === 'summer_peak_inspection'
            ? reportTemplate
            : coalReportTemplate,
        ])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }
      if (request.method === 'GET' && url.pathname.endsWith('/report-settings')) {
        return jsonResponse({
          data: { llm: { provider: 'ai-gateway' } },
          requestId: 'req-settings',
        })
      }
      if (request.method === 'GET' && url.pathname.endsWith('/admin/model-profiles')) {
        return jsonResponse({ data: [], requestId: 'req-profiles' })
      }

      return jsonResponse({ data: [], requestId: 'req-empty' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    expect(await screen.findByDisplayValue('2026年煤库存审计报告')).toBeVisible()
    expect(screen.getByDisplayValue('煤场库存账实与保供风险审计')).toBeVisible()

    // Open report type Select and pick another type
    fireEvent.click(screen.getAllByRole('combobox')[0]!)
    const summerOption = await screen.findByRole('option', { name: '真实巡检报告' })
    fireEvent.click(summerOption)

    expect(await screen.findByDisplayValue('2026年迎峰度夏检查报告')).toBeVisible()
    expect(screen.getByDisplayValue('迎峰度夏设备安全检查')).toBeVisible()

    fireEvent.change(screen.getByLabelText('报告名称'), {
      target: { value: '自定义审计标题' },
    })
    // Switch back report type
    fireEvent.click(screen.getAllByRole('combobox')[0]!)
    const coalOption = await screen.findByRole('option', { name: /煤库存审计报告/ })
    fireEvent.click(coalOption)

    expect(screen.getByDisplayValue('自定义审计标题')).toBeVisible()
    expect(await screen.findByDisplayValue('煤场库存账实与保供风险审计')).toBeVisible()
  })

  it('does not render local bootstrap fallback data when gateway bootstrap queries fail', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn<typeof fetch>()
        .mockImplementation(async () =>
          gatewayError('dependency_error', 'Document dependency down', 'req-bootstrap'),
        ),
    )

    renderWithProviders(<ReportGeneratePage />)

    expect(screen.queryByText('能力边界')).not.toBeInTheDocument()
    expect((await screen.findAllByText(/Document dependency down/))[0]).toBeVisible()
    expect(screen.getAllByText(/req-bootstrap/).length).toBeGreaterThan(0)
    expect(screen.queryByRole('option', { name: '煤库存审计报告' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /创建草稿/ })).toBeDisabled()
  })

  it('shows realistic seeded material labels while submitting the real material id', async () => {
    const user = userEvent.setup()
    const jobBodies: unknown[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([legacySeedReportMaterial])
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports')) {
        return jsonResponse({
          data: {
            id: 'rpt-material-seed',
            name: '迎峰度夏报告',
            reportType: 'summer_peak_inspection',
            status: 'draft',
            templateId: 'tpl-real',
          },
          requestId: 'req-create-report',
        })
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports/rpt-material-seed/jobs')) {
        jobBodies.push(await request.clone().json())
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            id: 'job-material-seed',
            jobType: 'outline_generation',
            progress: { completed: 0, total: 1 },
            reportId: 'rpt-material-seed',
            status: 'running',
          },
          requestId: 'req-outline-job',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-material-seed')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            id: 'job-material-seed',
            jobType: 'outline_generation',
            progress: { completed: 0, total: 1 },
            reportId: 'rpt-material-seed',
            status: 'running',
          },
          requestId: 'req-outline-status',
        })
      }
      if (
        url.pathname.endsWith('/reports/rpt-material-seed/outlines') ||
        url.pathname.endsWith('/reports/rpt-material-seed/sections') ||
        url.pathname.endsWith('/reports/rpt-material-seed/events')
      ) {
        return jsonResponse({ data: [], requestId: 'req-empty' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    const materialButton = await screen.findByRole('button', {
      name: '煤场库存盘点工作底稿',
    })
    expect(materialButton).toBeVisible()
    expect(screen.queryByText('本地演示检查记录')).not.toBeInTheDocument()

    await user.click(materialButton)
    await user.click(screen.getAllByRole('combobox')[0]!)
    await user.click(await screen.findByRole('option', { name: '真实巡检报告' }))
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())
    await user.click(screen.getByRole('button', { name: /创建草稿/ }))

    await waitFor(() => expect(jobBodies).toHaveLength(1))
    expect(jobBodies[0]).toMatchObject({
      jobType: 'outline_generation',
      materialIds: ['22222222-2222-4222-8222-222222222201'],
    })
  })

  it('does not render admin document model settings on the report generation page', async () => {
    setAuthenticatedUser(['report:write', 'admin:model-profile:write'])
    const paths: string[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)
      paths.push(`${request.method} ${url.pathname}${url.search}`)

      if (request.method === 'GET' && url.pathname.endsWith('/llm-config-versions/current')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            id: 'llm-admin-current',
            isActive: true,
            modelName: 'gpt-admin',
            profileId: 'mp-admin-chat',
            provider: 'ai-gateway',
            versionNo: 4,
          },
          requestId: 'req-admin-llm',
        })
      }
      if (
        url.pathname.endsWith('/report-settings') ||
        url.pathname.endsWith('/admin/model-profiles')
      ) {
        return gatewayError('forbidden', 'admin settings moved', 'req-admin', 403)
      }
      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }

      return jsonResponse({ data: [], requestId: 'req-empty' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    expect(await screen.findByLabelText('生成模型')).toBeVisible()
    expect(screen.queryByText('当前文档生成模型')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /发布文档模型配置/ })).not.toBeInTheDocument()
    expect(screen.queryByText('mp-admin-chat')).not.toBeInTheDocument()
    expect(screen.queryByText('gpt-admin')).not.toBeInTheDocument()
    expect(paths.some((path) => path.includes('/report-settings'))).toBe(false)
    expect(paths.some((path) => path.includes('/admin/model-profiles'))).toBe(false)
  })

  it('shows a compact personal generation model selector without admin-only requests', async () => {
    const user = userEvent.setup()
    setAuthenticatedUser(['report:write'])
    const paths: string[] = []
    const reportBodies: unknown[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)
      paths.push(`${request.method} ${url.pathname}${url.search}`)

      if (request.method === 'GET' && url.pathname.endsWith('/llm-config-versions/current')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            id: 'llm-user-current',
            isActive: true,
            modelName: 'gpt-user',
            profileId: 'mp-user-chat',
            provider: 'ai-gateway',
            versionNo: 3,
          },
          requestId: 'req-user-llm',
        })
      }
      if (
        url.pathname.endsWith('/report-settings') ||
        url.pathname.endsWith('/admin/model-profiles')
      ) {
        return gatewayError('forbidden', 'admin only', 'req-admin', 403)
      }
      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports')) {
        reportBodies.push(await request.clone().json())
        return jsonResponse({
          data: {
            id: 'rpt-writer',
            name: '迎峰度夏报告',
            reportType: 'summer_peak_inspection',
            status: 'draft',
            templateId: 'tpl-real',
          },
          requestId: 'req-create-report',
        })
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports/rpt-writer/jobs')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            id: 'job-writer',
            jobType: 'outline_generation',
            progress: { percent: 20 },
            reportId: 'rpt-writer',
            status: 'running',
          },
          requestId: 'req-job',
        })
      }
      if (request.method === 'GET' && url.pathname.endsWith('/report-jobs/job-writer')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            id: 'job-writer',
            jobType: 'outline_generation',
            progress: { percent: 20 },
            reportId: 'rpt-writer',
            status: 'running',
          },
          requestId: 'req-job-detail',
        })
      }
      if (
        url.pathname.endsWith('/reports/rpt-writer/outlines') ||
        url.pathname.endsWith('/reports/rpt-writer/sections')
      ) {
        return jsonResponse({ data: [], requestId: 'req-empty' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    // Wait for bootstrap data to load, then open Select and pick the report type
    const trigger = screen.getAllByRole('combobox')[0]!
    await waitFor(() => expect(trigger).not.toBeDisabled())
    await user.click(trigger)
    await screen.findByRole('option', { name: '真实巡检报告' })
    await user.click(screen.getByRole('option', { name: '真实巡检报告' }))
    expect(await screen.findByLabelText('生成模型')).toBeVisible()
    expect(screen.getByLabelText('生成模型')).toHaveTextContent('个人默认配置')
    expect(screen.queryByText('当前文档生成模型')).not.toBeInTheDocument()
    expect(screen.queryByText('mp-user-chat')).not.toBeInTheDocument()
    expect(screen.queryByText('gpt-user')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /发布文档模型配置/ })).not.toBeInTheDocument()
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())

    await user.click(screen.getByRole('button', { name: /创建草稿/ }))

    expect(await screen.findByText('报告模板类型')).toBeVisible()
    expect(screen.queryByText('20%')).not.toBeInTheDocument()
    expect(screen.getByText(/\d+%/)).toBeVisible()
    expect(screen.queryByText('job-writer')).not.toBeInTheDocument()
    expect(paths.some((path) => path.includes('/report-settings'))).toBe(false)
    expect(paths.some((path) => path.includes('/admin/model-profiles'))).toBe(false)
    expect(reportBodies[0]).toMatchObject({
      name: '2026年迎峰度夏检查报告',
      reportType: 'summer_peak_inspection',
      source: 'frontend',
      templateId: 'tpl-real',
    })
    expect(reportBodies[0]).not.toHaveProperty('profileId')
    expect(reportBodies[0]).not.toHaveProperty('llm')
  })

  it('renders user-facing report progress from completed sections without internal ids', async () => {
    const cancelBodies: unknown[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports')) {
        return jsonResponse({
          data: {
            id: 'rpt-progress',
            name: '迎峰度夏报告',
            reportType: 'summer_peak_inspection',
            status: 'draft',
            templateId: 'tpl-real',
          },
          requestId: 'req-create-report',
        })
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports/rpt-progress/jobs')) {
        const body = (await request.clone().json()) as { jobType?: string }
        if (body.jobType === 'content_generation') {
          return jsonResponse({
            data: {
              createdAt: '2026-07-03T00:00:00Z',
              id: 'job-content',
              jobType: 'content_generation',
              progress: { completed: 2, total: 4 },
              reportId: 'rpt-progress',
              resultSummary: '已生成 2 / 4 个章节',
              status: 'running',
            },
            requestId: 'req-content-job',
          })
        }
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            id: 'job-1',
            jobType: 'outline_generation',
            progress: { completed: 1, total: 1 },
            reportId: 'rpt-progress',
            resultSummary: '已生成大纲初稿',
            status: 'succeeded',
          },
          requestId: 'req-job',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-1')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            id: 'job-1',
            jobType: 'outline_generation',
            progress: { completed: 1, total: 1 },
            reportId: 'rpt-progress',
            resultSummary: '已生成大纲初稿',
            status: 'succeeded',
          },
          requestId: 'req-job-status',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-content')) {
        if (request.method === 'PATCH') {
          cancelBodies.push(await request.clone().json())
          return jsonResponse({
            data: {
              createdAt: '2026-07-03T00:00:00Z',
              finishedAt: '2026-07-03T00:05:00Z',
              id: 'job-content',
              jobType: 'content_generation',
              progress: { completed: 2, total: 4 },
              reportId: 'rpt-progress',
              status: 'canceled',
            },
            requestId: 'req-cancel-job',
          })
        }
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            id: 'job-content',
            jobType: 'content_generation',
            progress: { completed: 2, total: 4 },
            reportId: 'rpt-progress',
            resultSummary: '已生成 2 / 4 个章节',
            status: 'running',
          },
          requestId: 'req-content-job-status',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-progress/outlines')) {
        return jsonResponse({
          data: [
            {
              createdAt: '2026-07-03T00:00:00Z',
              id: 'outline-progress',
              isCurrent: true,
              reportId: 'rpt-progress',
              sections: [
                { id: 'node-1', level: 1, numbering: '1', title: '总述' },
                { id: 'node-2', level: 1, numbering: '2', title: '风险分析' },
              ],
              source: 'ai',
              version: 1,
            },
          ],
          requestId: 'req-outlines',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-progress/sections')) {
        return jsonResponse({ data: [], requestId: 'req-empty' })
      }
      if (url.pathname.endsWith('/reports/rpt-progress/events')) {
        return jsonResponse({
          data: [
            {
              createdAt: '2026-07-03T00:00:00Z',
              eventType: 'job.started',
              id: 'event-1',
              jobId: 'job-1',
              message: '任务已开始',
              reportId: 'rpt-progress',
            },
          ],
          requestId: 'req-events',
        })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    // Open report type Select and pick the first option
    const trigger = screen.getAllByRole('combobox')[0]!
    await waitFor(() => expect(trigger).not.toBeDisabled())
    fireEvent.click(trigger)
    await screen.findByRole('option', { name: '真实巡检报告' })
    fireEvent.click(screen.getByRole('option', { name: '真实巡检报告' }))
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())
    fireEvent.click(screen.getByRole('button', { name: /创建草稿/ }))

    expect(await screen.findByText('报告模板类型')).toBeVisible()
    expect(screen.getByText('真实巡检报告')).toBeVisible()
    expect(screen.queryByText('reportId')).not.toBeInTheDocument()
    expect(screen.queryByText('jobId')).not.toBeInTheDocument()
    expect(screen.queryByText('任务类型')).not.toBeInTheDocument()
    expect(screen.queryByText('job-1')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /取消任务/ })).not.toBeInTheDocument()

    await waitFor(() => expect(screen.getByRole('button', { name: /生成正文/ })).toBeEnabled())
    fireEvent.click(screen.getByRole('button', { name: /^生成正文$/ }))

    expect(await screen.findByText(/60%/)).toBeVisible()
    expect(screen.getByText('已生成 2 / 4 个章节')).toBeVisible()
    expect(screen.queryByText('job-content')).not.toBeInTheDocument()
    const cancelButton = screen.getByRole('button', { name: /取消任务/ })
    expect(cancelButton).toBeVisible()
    expect(cancelButton).not.toHaveClass('bg-black')
    expect(cancelButton).not.toHaveClass('text-white')
    fireEvent.click(cancelButton)
    await waitFor(() => expect(cancelBodies).toEqual([{ status: 'canceled' }]))
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(screen.queryByText('任务状态')).not.toBeInTheDocument()
    expect(screen.queryByText('事件日志')).not.toBeInTheDocument()
    expect(screen.queryByText('当前报告')).not.toBeInTheDocument()
  })

  it('submits selected knowledge base ids for content generation', async () => {
    const contentBodies: Record<string, unknown>[] = []
    vi.stubGlobal('fetch', createKnowledgeSelectionFetchMock({ contentBodies }))

    renderWithProviders(<ReportGeneratePage />)

    await createReportAndReachOutlineStep()
    fireEvent.click(await screen.findByRole('button', { name: 'Operations KB' }))
    fireEvent.click(screen.getByRole('button', { name: 'Safety KB' }))
    fireEvent.click(screen.getByRole('button', { name: /^生成正文$/ }))

    await waitFor(() => expect(contentBodies).toHaveLength(1))
    expect((contentBodies[0]?.options as Record<string, unknown>).knowledgeBaseIds).toEqual([
      'kb-ops',
      'kb-safety',
    ])
  })

  it('omits knowledgeBaseIds when no knowledge base is selected', async () => {
    const contentBodies: Record<string, unknown>[] = []
    vi.stubGlobal('fetch', createKnowledgeSelectionFetchMock({ contentBodies }))

    renderWithProviders(<ReportGeneratePage />)

    await createReportAndReachOutlineStep()
    fireEvent.click(screen.getByRole('button', { name: /^生成正文$/ }))

    await waitFor(() => expect(contentBodies).toHaveLength(1))
    expect(contentBodies[0]?.options).toEqual({ preserveManualEdits: true, saveResult: true })
    expect(contentBodies[0]?.options).not.toHaveProperty('knowledgeBaseIds')
  })

  it('creates a section regeneration job from the active section action', async () => {
    const contentBodies: Record<string, unknown>[] = []
    const sectionBodies: Record<string, unknown>[] = []
    vi.stubGlobal(
      'fetch',
      createKnowledgeSelectionFetchMock({
        contentBodies,
        contentJobStatus: 'succeeded',
        sectionBodies,
        sections: [
          {
            content: '已生成正文',
            generatedAt: '2026-07-03T00:02:30Z',
            generationStatus: 'succeeded',
            id: 'section-knowledge',
            numbering: '1',
            reportId: 'rpt-knowledge-selection',
            sortOrder: 1,
            title: 'Knowledge section',
          },
          {
            content: '其他章节正文',
            generatedAt: '2026-07-03T00:02:30Z',
            generationStatus: 'succeeded',
            id: 'section-stable',
            numbering: '2',
            reportId: 'rpt-knowledge-selection',
            sortOrder: 2,
            title: 'Stable section',
          },
        ],
      }),
    )

    renderWithProviders(<ReportGeneratePage />)

    await createReportAndReachOutlineStep()
    fireEvent.click(screen.getByRole('button', { name: /^生成正文$/ }))

    const regenerateButton = await screen.findByRole('button', { name: /重新生成本章/ })
    await waitFor(() => expect(regenerateButton).toBeEnabled())
    fireEvent.click(regenerateButton)

    await waitFor(() => expect(sectionBodies).toHaveLength(1))
    expect(sectionBodies[0]).toMatchObject({
      jobType: 'section_regeneration',
      target: { scope: 'section', sectionId: 'section-knowledge' },
    })
    const sectionList = screen.getByLabelText('章节列表')
    await within(sectionList).findByRole('button', { name: /Knowledge section.*生成中/ })
    expect(
      within(sectionList).getByRole('button', { name: /Stable section.*已完成/ }),
    ).toBeVisible()
  })

  it('keeps content generation available when knowledge bases fail to load', async () => {
    const contentBodies: Record<string, unknown>[] = []
    vi.stubGlobal(
      'fetch',
      createKnowledgeSelectionFetchMock({ contentBodies, knowledgeError: true }),
    )

    renderWithProviders(<ReportGeneratePage />)

    await createReportAndReachOutlineStep()
    expect(await screen.findByText(/Knowledge list unavailable/)).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: /^生成正文$/ }))

    await waitFor(() => expect(contentBodies).toHaveLength(1))
    expect(contentBodies[0]?.options).toEqual({ preserveManualEdits: true, saveResult: true })
    expect(contentBodies[0]?.options).not.toHaveProperty('knowledgeBaseIds')
  })

  it('does not pop a degraded notice when knowledge retrieval is skipped during content generation', async () => {
    const contentBodies: Record<string, unknown>[] = []
    vi.stubGlobal(
      'fetch',
      createKnowledgeSelectionFetchMock({
        contentBodies,
        events: [
          {
            createdAt: '2026-07-03T00:02:30Z',
            eventType: 'knowledge.retrieval_degraded',
            id: 'evt-knowledge-degraded',
            jobId: 'job-content-knowledge',
            message: 'knowledge retrieval failed; generation continued without knowledge context',
            reportId: 'rpt-knowledge-selection',
          },
        ],
      }),
    )

    renderWithProviders(<ReportGeneratePage />)

    await createReportAndReachOutlineStep()
    fireEvent.click(screen.getByRole('button', { name: /^生成正文$/ }))

    await waitFor(() => expect(contentBodies).toHaveLength(1))
    expect(screen.queryByText(/本次生成未引用知识库/)).not.toBeInTheDocument()
    expect(screen.queryByText(/知识库检索失败/)).not.toBeInTheDocument()
  })

  it('ignores streamed outline chunks in the outline editor area', async () => {
    const contentBodies: Record<string, unknown>[] = []
    const fetchMock = createKnowledgeSelectionFetchMock({
      contentBodies,
      outlineJobStatus: 'running',
      outlines: [],
      streamFrames: [
        [
          'event: report.event',
          'data: {"id":"evt-outline-delta-1","reportId":"rpt-knowledge-selection","jobId":"job-outline-knowledge","eventType":"outline.delta","message":"{\\"sections\\":[{\\"title\\":\\"Outline from JSON\\"},{\\"title\\":\\"Knowledge risks\\"}]}","createdAt":"2026-07-03T00:00:10Z"}',
          '',
          '',
        ].join('\n'),
      ],
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    const trigger = screen.getAllByRole('combobox')[0]!
    await waitFor(() => expect(trigger).not.toBeDisabled())
    fireEvent.click(trigger)
    fireEvent.click(await screen.findByRole('option', { name: reportType.name }))
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())
    fireEvent.click(screen.getByRole('button', { name: /创建草稿/ }))

    await expect(
      screen.findByDisplayValue('Outline from JSON', {}, { timeout: 300 }),
    ).rejects.toThrow()
    const streamRequested = fetchMock.mock.calls.some(([input, init]) => {
      const request = input instanceof Request ? input : new Request(input, init)
      return new URL(request.url).pathname.endsWith('/events/stream')
    })
    expect(streamRequested).toBe(false)
    expect(screen.queryByDisplayValue('Knowledge risks')).not.toBeInTheDocument()
    expect(screen.queryByDisplayValue(/sections/)).not.toBeInTheDocument()
    expect(screen.queryByText(/实时大纲预览/)).not.toBeInTheDocument()
  })

  it('fades persisted outline rows when the generated outline appears', async () => {
    const contentBodies: Record<string, unknown>[] = []
    vi.stubGlobal(
      'fetch',
      createKnowledgeSelectionFetchMock({
        contentBodies,
        streamFrames: [],
      }),
    )

    renderWithProviders(<ReportGeneratePage />)

    await createReportAndReachOutlineStep()

    const outlineTitleInput = await screen.findByDisplayValue('Knowledge section')
    const outlineRow = outlineTitleInput.closest('div')
    expect(outlineRow).not.toBeNull()
    expect(outlineRow as HTMLElement).toHaveClass('animate-[fade-in-up_0.28s_ease-out_both]')
    expect(outlineRow as HTMLElement).toHaveClass('opacity-0')
  })

  it('types streamed report body chunks into the section editor', async () => {
    const contentBodies: Record<string, unknown>[] = []
    const previousPersistedBody = 'previous persisted audit scope'
    const streamedBody = 'streamed body from knowledge'
    vi.stubGlobal(
      'fetch',
      createKnowledgeSelectionFetchMock({
        contentBodies,
        sections: [
          {
            content: previousPersistedBody,
            generatedAt: null,
            generationStatus: 'running',
            id: 'section-knowledge',
            numbering: '1',
            reportId: 'rpt-knowledge-selection',
            sortOrder: 1,
            title: 'Knowledge section',
          },
        ],
        streamFrames: [
          [
            'event: report.event',
            `data: ${JSON.stringify({
              createdAt: '2026-07-03T00:02:10Z',
              eventType: 'section.delta',
              id: 'evt-section-delta-1',
              jobId: 'job-content-knowledge',
              message: JSON.stringify({
                sectionId: 'section-knowledge',
                text: JSON.stringify({
                  content: streamedBody,
                  tables: [
                    {
                      headers: ['Metric', 'Value'],
                      rows: [['Peak load', '102 MW']],
                    },
                  ],
                }),
              }),
              reportId: 'rpt-knowledge-selection',
            })}`,
            '',
            '',
          ].join('\n'),
        ],
      }),
    )

    renderWithProviders(<ReportGeneratePage />)

    await createReportAndReachOutlineStep()
    fireEvent.click(screen.getByRole('button', { name: /^生成正文$/ }))

    await waitFor(() => expect(contentBodies).toHaveLength(1))
    expect(await screen.findByDisplayValue(streamedBody)).toBeVisible()
    expect(screen.getByLabelText('章节正文')).toHaveValue(streamedBody)
    expect(screen.queryByDisplayValue(new RegExp(previousPersistedBody))).not.toBeInTheDocument()
    expect(screen.queryByDisplayValue(/content/)).not.toBeInTheDocument()
    expect(screen.queryByDisplayValue(/Metric/)).not.toBeInTheDocument()
    expect(screen.queryByDisplayValue(/Peak load/)).not.toBeInTheDocument()
    expect(screen.queryByText(/实时正文预览/)).not.toBeInTheDocument()
  })

  it('does not duplicate streamed section text when polling returns persisted content during a live job', async () => {
    const contentBodies: Record<string, unknown>[] = []
    const streamedBody = 'streamed draft body'
    vi.stubGlobal(
      'fetch',
      createKnowledgeSelectionFetchMock({
        contentBodies,
        sections: [
          {
            content: '',
            generatedAt: null,
            generationStatus: 'running',
            id: 'section-knowledge',
            numbering: '1',
            reportId: 'rpt-knowledge-selection',
            sortOrder: 1,
            title: 'Knowledge section',
          },
        ],
        streamFrames: [
          [
            'event: report.event',
            `data: ${JSON.stringify({
              createdAt: '2026-07-03T00:02:10Z',
              eventType: 'section.delta',
              id: 'evt-section-delta-1',
              jobId: 'job-content-knowledge',
              message: JSON.stringify({
                sectionId: 'section-knowledge',
                text: streamedBody,
              }),
              reportId: 'rpt-knowledge-selection',
            })}`,
            '',
            '',
          ].join('\n'),
        ],
      }),
    )

    const { queryClient } = renderWithProviders(<ReportGeneratePage />)

    await createReportAndReachOutlineStep()
    fireEvent.click(screen.getByRole('button', { name: /^生成正文$/ }))

    await waitFor(() => expect(contentBodies).toHaveLength(1))
    expect(await screen.findByDisplayValue(streamedBody)).toBeVisible()

    act(() => {
      queryClient.setQueryData(reportKeys.sections('rpt-knowledge-selection'), [
        {
          content: streamedBody,
          generatedAt: '2026-07-03T00:02:30Z',
          generationStatus: 'succeeded',
          id: 'section-knowledge',
          lastJobId: 'job-content-knowledge',
          numbering: '1',
          reportId: 'rpt-knowledge-selection',
          sortOrder: 1,
          title: 'Knowledge section',
        },
      ])
    })

    await act(async () => {
      await Promise.resolve()
      await Promise.resolve()
    })

    await waitFor(() => expect(screen.getByText('已完成')).toBeVisible())
    expect(screen.getByLabelText('章节正文')).toHaveValue(streamedBody)
    expect(screen.queryByDisplayValue(`${streamedBody}${streamedBody}`)).not.toBeInTheDocument()
  })

  it('restores an in-progress content job after navigating away and back', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports')) {
        return jsonResponse({
          data: {
            id: 'rpt-restore',
            name: 'restore-report',
            reportType: 'summer_peak_inspection',
            status: 'draft',
            templateId: 'tpl-real',
          },
          requestId: 'req-create-report',
        })
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports/rpt-restore/jobs')) {
        const body = (await request.clone().json()) as { jobType?: string }
        if (body.jobType === 'content_generation') {
          return jsonResponse({
            data: {
              createdAt: '2026-07-03T00:02:00Z',
              id: 'job-content-restore',
              jobType: 'content_generation',
              progress: { completed: 1, total: 2 },
              reportId: 'rpt-restore',
              resultSummary: 'restore progress 1 / 2',
              status: 'running',
            },
            requestId: 'req-content-job',
          })
        }
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:01:00Z',
            id: 'job-outline-restore',
            jobType: 'outline_generation',
            progress: { completed: 1, total: 1 },
            reportId: 'rpt-restore',
            status: 'succeeded',
          },
          requestId: 'req-outline-job',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-outline-restore')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:01:00Z',
            id: 'job-outline-restore',
            jobType: 'outline_generation',
            reportId: 'rpt-restore',
            status: 'succeeded',
          },
          requestId: 'req-outline-status',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-content-restore')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:02:00Z',
            id: 'job-content-restore',
            jobType: 'content_generation',
            progress: { completed: 1, total: 2 },
            reportId: 'rpt-restore',
            resultSummary: 'restore progress 1 / 2',
            status: 'running',
          },
          requestId: 'req-content-status',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-restore/outlines')) {
        return jsonResponse({
          data: [
            {
              createdAt: '2026-07-03T00:00:00Z',
              id: 'outline-restore',
              isCurrent: true,
              reportId: 'rpt-restore',
              sections: [
                { id: 'node-1', level: 1, numbering: '1', title: 'restore-section-1' },
                { id: 'node-2', level: 1, numbering: '2', title: 'restore-section-2' },
              ],
              source: 'ai',
              version: 1,
            },
          ],
          requestId: 'req-outlines',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-restore/sections')) {
        return jsonResponse({
          data: [
            {
              content: 'done',
              generationStatus: 'succeeded',
              id: 'section-1',
              numbering: '1',
              outlineNodeId: 'node-1',
              reportId: 'rpt-restore',
              title: 'restore-section-1',
            },
            {
              content: '',
              generationStatus: 'running',
              id: 'section-2',
              numbering: '2',
              outlineNodeId: 'node-2',
              reportId: 'rpt-restore',
              title: 'restore-section-2',
            },
          ],
          requestId: 'req-sections',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-restore/events')) {
        return jsonResponse({ data: [], requestId: 'req-events' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    const firstRender = renderWithProviders(<ReportGeneratePage />)

    const restoreTypeTrigger = screen.getAllByRole('combobox')[0]!
    await waitFor(() => expect(restoreTypeTrigger).not.toBeDisabled())
    fireEvent.click(restoreTypeTrigger)
    await screen.findByRole('option', { name: '真实巡检报告' })
    fireEvent.click(screen.getByRole('option', { name: '真实巡检报告' }))
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())
    fireEvent.click(screen.getByRole('button', { name: /创建草稿/ }))

    await waitFor(() => expect(screen.getByRole('button', { name: /生成正文/ })).toBeEnabled())
    fireEvent.click(screen.getByRole('button', { name: /^生成正文$/ }))

    expect(await screen.findByText(/restore progress 1 \/ 2/)).toBeVisible()
    expect((await screen.findAllByText('restore-section-1')).length).toBeGreaterThan(0)
    const sectionList = screen.getByLabelText('章节列表')
    expect(sectionList).not.toHaveClass('lg:max-h-[620px]')
    expect(sectionList).not.toHaveClass('lg:overflow-y-auto')
    const sectionScroller = sectionList.querySelector('.space-y-2')
    expect(sectionScroller).toBeInstanceOf(HTMLElement)
    expect(sectionScroller).toHaveClass('max-h-[28rem]')
    expect(sectionScroller).toHaveClass('overflow-y-auto')
    expect(sectionScroller).not.toHaveClass('max-h-64')

    firstRender.unmount()
    renderWithProviders(<ReportGeneratePage />)

    expect(await screen.findByText(/restore progress 1 \/ 2/)).toBeVisible()
    expect((await screen.findAllByText('restore-section-1')).length).toBeGreaterThan(0)
    expect(screen.getAllByRole('button', { name: /取消任务/ })).toHaveLength(1)
  })

  it('allows retrying a succeeded content generation job', async () => {
    const retryBodies: unknown[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports')) {
        return jsonResponse({
          data: {
            id: 'rpt-retry-success',
            name: '迎峰度夏报告',
            reportType: 'summer_peak_inspection',
            status: 'draft',
            templateId: 'tpl-real',
          },
          requestId: 'req-create-report',
        })
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports/rpt-retry-success/jobs')) {
        const body = (await request.clone().json()) as { jobType?: string }
        if (body.jobType === 'content_generation') {
          return jsonResponse({
            data: {
              createdAt: '2026-07-03T00:00:00Z',
              finishedAt: '2026-07-03T00:05:00Z',
              id: 'job-content-success',
              jobType: 'content_generation',
              progress: { completed: 2, total: 2 },
              reportId: 'rpt-retry-success',
              resultSummary: '已生成 2 / 2 个章节',
              status: 'succeeded',
            },
            requestId: 'req-content-job',
          })
        }
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:01:00Z',
            id: 'job-outline-success',
            jobType: 'outline_generation',
            progress: { completed: 1, total: 1 },
            reportId: 'rpt-retry-success',
            status: 'succeeded',
          },
          requestId: 'req-outline-job',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-outline-success')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:01:00Z',
            id: 'job-outline-success',
            jobType: 'outline_generation',
            reportId: 'rpt-retry-success',
            status: 'succeeded',
          },
          requestId: 'req-outline-status',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-content-success/attempts')) {
        retryBodies.push(await request.clone().json())
        return jsonResponse({
          data: {
            attemptNumber: 2,
            createdAt: '2026-07-03T00:06:00Z',
            id: 'attempt-retry-success',
            jobId: 'job-content-success',
            status: 'pending',
          },
          requestId: 'req-retry-success',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-content-success')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:05:00Z',
            id: 'job-content-success',
            jobType: 'content_generation',
            progress: { completed: 2, total: 2 },
            reportId: 'rpt-retry-success',
            resultSummary: '已生成 2 / 2 个章节',
            status: 'succeeded',
          },
          requestId: 'req-content-status',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-retry-success/outlines')) {
        return jsonResponse({
          data: [
            {
              createdAt: '2026-07-03T00:00:00Z',
              id: 'outline-retry-success',
              isCurrent: true,
              reportId: 'rpt-retry-success',
              sections: [{ id: 'node-1', level: 1, numbering: '1', title: '总览' }],
              source: 'ai',
              version: 1,
            },
          ],
          requestId: 'req-outlines',
        })
      }
      if (
        url.pathname.endsWith('/reports/rpt-retry-success/sections') ||
        url.pathname.endsWith('/reports/rpt-retry-success/events')
      ) {
        return jsonResponse({ data: [], requestId: 'req-empty' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    const retryTypeTrigger = screen.getAllByRole('combobox')[0]!
    await waitFor(() => expect(retryTypeTrigger).not.toBeDisabled())
    fireEvent.click(retryTypeTrigger)
    await screen.findByRole('option', { name: '真实巡检报告' })
    fireEvent.click(screen.getByRole('option', { name: '真实巡检报告' }))
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())
    fireEvent.click(screen.getByRole('button', { name: /创建草稿/ }))

    await waitFor(() => expect(screen.getByRole('button', { name: /生成正文/ })).toBeEnabled())
    fireEvent.click(screen.getByRole('button', { name: /生成正文/ }))

    await screen.findByText(/100%/)
    const retryButton = await waitFor(() => {
      const buttons = screen.getAllByRole('button', { name: /重试任务/ })
      const enabled = buttons.find((button) => !(button as HTMLButtonElement).disabled)
      expect(enabled).toBeDefined()
      return enabled as HTMLButtonElement
    })
    fireEvent.click(retryButton)

    await waitFor(() => expect(retryBodies).toEqual([{ reason: 'frontend_retry' }]))
  })

  it('smooths running outline progress instead of displaying backend percent jumps', async () => {
    const now = new Date().toISOString()
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports')) {
        return jsonResponse({
          data: {
            id: 'rpt-smooth-outline',
            name: '迎峰度夏报告',
            reportType: 'summer_peak_inspection',
            status: 'draft',
            templateId: 'tpl-real',
          },
          requestId: 'req-create-report',
        })
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports/rpt-smooth-outline/jobs')) {
        return jsonResponse({
          data: {
            createdAt: now,
            id: 'job-smooth-outline',
            jobType: 'outline_generation',
            progress: { percent: 20 },
            reportId: 'rpt-smooth-outline',
            status: 'running',
          },
          requestId: 'req-outline-job',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-smooth-outline')) {
        return jsonResponse({
          data: {
            createdAt: now,
            id: 'job-smooth-outline',
            jobType: 'outline_generation',
            progress: { percent: 20 },
            reportId: 'rpt-smooth-outline',
            status: 'running',
          },
          requestId: 'req-outline-status',
        })
      }
      if (
        url.pathname.endsWith('/reports/rpt-smooth-outline/outlines') ||
        url.pathname.endsWith('/reports/rpt-smooth-outline/sections') ||
        url.pathname.endsWith('/reports/rpt-smooth-outline/events')
      ) {
        return jsonResponse({ data: [], requestId: 'req-empty' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    const outlineTypeTrigger = screen.getAllByRole('combobox')[0]!
    await waitFor(() => expect(outlineTypeTrigger).not.toBeDisabled())
    fireEvent.click(outlineTypeTrigger)
    await screen.findByRole('option', { name: '真实巡检报告' })
    fireEvent.click(screen.getByRole('option', { name: '真实巡检报告' }))
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())
    fireEvent.click(screen.getByRole('button', { name: /创建草稿/ }))

    await screen.findByText('当前文档进度')
    expect(screen.queryByText('20%')).not.toBeInTheDocument()
  })

  it('clears old outline and section state immediately when reusing a draft for a new outline', async () => {
    const nextOutlineRefresh = deferredResponse<Response>()
    let reportJobCreates = 0
    let outlineListReads = 0
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports')) {
        return jsonResponse({
          data: {
            id: 'rpt-outline-refresh',
            name: '迎峰度夏报告',
            reportType: 'summer_peak_inspection',
            status: 'draft',
            templateId: 'tpl-real',
          },
          requestId: 'req-create-report',
        })
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports/rpt-outline-refresh/jobs')) {
        reportJobCreates += 1
        if (reportJobCreates === 1) {
          return jsonResponse({
            data: {
              createdAt: '2026-07-03T00:00:00Z',
              finishedAt: '2026-07-03T00:01:00Z',
              id: 'job-old-outline',
              jobType: 'outline_generation',
              progress: { completed: 1, total: 1 },
              reportId: 'rpt-outline-refresh',
              status: 'succeeded',
            },
            requestId: 'req-old-outline-job',
          })
        }
        return jsonResponse({
          data: {
            createdAt: new Date().toISOString(),
            id: 'job-new-outline',
            jobType: 'outline_generation',
            progress: { percent: 8 },
            reportId: 'rpt-outline-refresh',
            status: 'running',
          },
          requestId: 'req-new-outline-job',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-old-outline')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:01:00Z',
            id: 'job-old-outline',
            jobType: 'outline_generation',
            progress: { completed: 1, total: 1 },
            reportId: 'rpt-outline-refresh',
            status: 'succeeded',
          },
          requestId: 'req-old-outline-status',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-new-outline')) {
        return jsonResponse({
          data: {
            createdAt: new Date().toISOString(),
            id: 'job-new-outline',
            jobType: 'outline_generation',
            progress: { percent: 8 },
            reportId: 'rpt-outline-refresh',
            status: 'running',
          },
          requestId: 'req-new-outline-status',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-outline-refresh/outlines')) {
        outlineListReads += 1
        if (outlineListReads > 1) return nextOutlineRefresh.promise
        return jsonResponse({
          data: [
            {
              createdAt: '2026-07-03T00:00:00Z',
              id: 'outline-old',
              isCurrent: true,
              reportId: 'rpt-outline-refresh',
              sections: [{ id: 'old-node', level: 1, numbering: '1', title: '旧大纲章节' }],
              source: 'ai',
              version: 1,
            },
          ],
          requestId: 'req-old-outline-list',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-outline-refresh/sections')) {
        return jsonResponse({
          data: [
            {
              content: '旧正文内容',
              generationStatus: 'succeeded',
              id: 'old-section',
              outlineNodeId: 'old-node',
              reportId: 'rpt-outline-refresh',
              title: '旧正文章节',
            },
          ],
          requestId: 'req-old-sections',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-outline-refresh/events')) {
        return jsonResponse({ data: [], requestId: 'req-events' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    const outlineTypeTrigger = screen.getAllByRole('combobox')[0]!
    await waitFor(() => expect(outlineTypeTrigger).not.toBeDisabled())
    fireEvent.click(outlineTypeTrigger)
    await screen.findByRole('option', { name: '真实巡检报告' })
    fireEvent.click(screen.getByRole('option', { name: '真实巡检报告' }))
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())
    fireEvent.click(screen.getByRole('button', { name: /创建草稿/ }))

    expect(await screen.findByDisplayValue('旧大纲章节')).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: /生成正文/ }))
    expect((await screen.findAllByText('旧大纲章节')).length).toBeGreaterThan(0)
    expect(screen.queryByText('旧正文章节')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /草稿与大纲/ }))
    fireEvent.click(screen.getByRole('button', { name: /复用草稿生成大纲/ }))
    await waitFor(() => expect(reportJobCreates).toBe(2))

    expect(screen.queryByDisplayValue('旧大纲章节')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /正文生成/ }))
    expect(screen.queryByText('旧大纲章节')).not.toBeInTheDocument()
    expect(screen.queryByText('旧正文章节')).not.toBeInTheDocument()

    nextOutlineRefresh.resolve(jsonResponse({ data: [], requestId: 'req-new-outline-empty' }))
  })

  it('resets section row statuses after retrying a content generation job', async () => {
    const attempts: unknown[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports')) {
        return jsonResponse({
          data: {
            id: 'rpt-retry-sections',
            name: '迎峰度夏报告',
            reportType: 'summer_peak_inspection',
            status: 'draft',
            templateId: 'tpl-real',
          },
          requestId: 'req-create-report',
        })
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports/rpt-retry-sections/jobs')) {
        const body = (await request.clone().json()) as { jobType?: string }
        if (body.jobType === 'content_generation') {
          return jsonResponse({
            data: {
              createdAt: '2026-07-03T00:02:00Z',
              finishedAt: '2026-07-03T00:03:00Z',
              id: 'job-content-done',
              jobType: 'content_generation',
              progress: { completed: 1, total: 1 },
              reportId: 'rpt-retry-sections',
              status: 'succeeded',
            },
            requestId: 'req-content-job',
          })
        }
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:01:00Z',
            id: 'job-outline-done',
            jobType: 'outline_generation',
            progress: { completed: 1, total: 1 },
            reportId: 'rpt-retry-sections',
            status: 'succeeded',
          },
          requestId: 'req-outline-job',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-outline-done')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:01:00Z',
            id: 'job-outline-done',
            jobType: 'outline_generation',
            reportId: 'rpt-retry-sections',
            status: 'succeeded',
          },
          requestId: 'req-outline-status',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-content-done')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:02:00Z',
            finishedAt: '2026-07-03T00:03:00Z',
            id: 'job-content-done',
            jobType: 'content_generation',
            progress: { completed: 1, total: 1 },
            reportId: 'rpt-retry-sections',
            status: 'succeeded',
          },
          requestId: 'req-content-status',
        })
      }
      if (
        request.method === 'POST' &&
        url.pathname.endsWith('/report-jobs/job-content-done/attempts')
      ) {
        attempts.push(
          await request
            .clone()
            .json()
            .catch(() => null),
        )
        return jsonResponse({
          data: {
            attempt: 2,
            createdAt: '2026-07-03T00:04:00Z',
            id: 'attempt-2',
            jobId: 'job-content-done',
            status: 'pending',
          },
          requestId: 'req-retry',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-retry-sections/outlines')) {
        return jsonResponse({
          data: [
            {
              createdAt: '2026-07-03T00:00:00Z',
              id: 'outline-retry',
              isCurrent: true,
              reportId: 'rpt-retry-sections',
              sections: [{ id: 'node-retry', level: 1, numbering: '1', title: '重试章节' }],
              source: 'ai',
              version: 1,
            },
          ],
          requestId: 'req-outlines',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-retry-sections/sections')) {
        return jsonResponse({
          data: [
            {
              content: '已生成正文',
              generationStatus: 'succeeded',
              id: 'section-retry',
              lastJobId: 'job-content-done',
              outlineNodeId: 'node-retry',
              reportId: 'rpt-retry-sections',
              title: '重试章节',
            },
          ],
          requestId: 'req-sections',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-retry-sections/events')) {
        return jsonResponse({ data: [], requestId: 'req-events' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    const outlineTypeTrigger = screen.getAllByRole('combobox')[0]!
    await waitFor(() => expect(outlineTypeTrigger).not.toBeDisabled())
    fireEvent.click(outlineTypeTrigger)
    await screen.findByRole('option', { name: '真实巡检报告' })
    fireEvent.click(screen.getByRole('option', { name: '真实巡检报告' }))
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())
    fireEvent.click(screen.getByRole('button', { name: /创建草稿/ }))

    await screen.findByDisplayValue('重试章节')
    fireEvent.click(screen.getByRole('button', { name: /生成正文/ }))
    expect((await screen.findAllByText('重试章节')).length).toBeGreaterThan(0)
    expect(screen.getByText('已完成')).toBeVisible()

    fireEvent.click(screen.getAllByRole('button', { name: /重试任务/ })[0]!)

    await waitFor(() => expect(attempts).toHaveLength(1))
    expect(await within(screen.getByLabelText('章节列表')).findByText('等待中')).toBeVisible()
  })

  it('shows completed section rows from the active content job without navigation', async () => {
    let contentJobCreated = false
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports')) {
        return jsonResponse({
          data: {
            id: 'rpt-live-section-status',
            name: '迎峰度夏报告',
            reportType: 'summer_peak_inspection',
            status: 'draft',
            templateId: 'tpl-real',
          },
          requestId: 'req-create-report',
        })
      }
      if (
        request.method === 'POST' &&
        url.pathname.endsWith('/reports/rpt-live-section-status/jobs')
      ) {
        const body = (await request.clone().json()) as { jobType?: string }
        if (body.jobType === 'content_generation') {
          contentJobCreated = true
          return jsonResponse({
            data: {
              createdAt: '2026-07-03T00:02:00Z',
              id: 'job-live-content',
              jobType: 'content_generation',
              progress: { completed: 1, total: 2 },
              reportId: 'rpt-live-section-status',
              status: 'running',
            },
            requestId: 'req-content-job',
          })
        }
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:01:00Z',
            id: 'job-live-outline',
            jobType: 'outline_generation',
            progress: { completed: 1, total: 1 },
            reportId: 'rpt-live-section-status',
            status: 'succeeded',
          },
          requestId: 'req-outline-job',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-live-outline')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:01:00Z',
            id: 'job-live-outline',
            jobType: 'outline_generation',
            reportId: 'rpt-live-section-status',
            status: 'succeeded',
          },
          requestId: 'req-outline-status',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-live-content')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:02:00Z',
            id: 'job-live-content',
            jobType: 'content_generation',
            progress: { completed: 1, total: 2 },
            reportId: 'rpt-live-section-status',
            status: 'running',
          },
          requestId: 'req-content-status',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-live-section-status/outlines')) {
        return jsonResponse({
          data: [
            {
              createdAt: '2026-07-03T00:00:00Z',
              id: 'outline-live',
              isCurrent: true,
              reportId: 'rpt-live-section-status',
              sections: [{ id: 'node-live', level: 1, numbering: '1', title: 'Live section' }],
              source: 'ai',
              version: 1,
            },
          ],
          requestId: 'req-outlines',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-live-section-status/sections')) {
        return jsonResponse({
          data: [
            {
              generationStatus: contentJobCreated ? 'succeeded' : 'pending',
              id: 'section-live',
              lastJobId: contentJobCreated ? 'job-live-content' : undefined,
              outlineNodeId: 'node-live',
              reportId: 'rpt-live-section-status',
              title: 'Live section',
            },
          ],
          requestId: 'req-sections',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-live-section-status/events')) {
        return jsonResponse({ data: [], requestId: 'req-events' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    const outlineTypeTrigger = screen.getAllByRole('combobox')[0]!
    await waitFor(() => expect(outlineTypeTrigger).not.toBeDisabled())
    fireEvent.click(outlineTypeTrigger)
    fireEvent.click(await screen.findByRole('option', { name: reportType.name }))
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())
    fireEvent.click(screen.getByRole('button', { name: /创建草稿/ }))

    await screen.findByDisplayValue('Live section')
    fireEvent.click(screen.getByRole('button', { name: /^生成正文$/ }))

    const sectionList = await screen.findByLabelText('章节列表')
    expect(await within(sectionList).findByText('已完成')).toBeVisible()
  })

  it('auto-saves dirty outline before generating content and uses saved titles', async () => {
    let savedOutline = false
    let contentJobCreated = false
    const requestOrder: string[] = []
    const savedOutlines: unknown[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports')) {
        return jsonResponse({
          data: {
            id: 'rpt-saved-outline',
            name: 'Saved outline report',
            reportType: 'summer_peak_inspection',
            status: 'draft',
            templateId: 'tpl-real',
          },
          requestId: 'req-create-report',
        })
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports/rpt-saved-outline/jobs')) {
        const body = (await request.clone().json()) as { jobType?: string }
        if (body.jobType === 'content_generation') {
          contentJobCreated = true
          requestOrder.push('content-job')
          return jsonResponse({
            data: {
              createdAt: '2026-07-03T00:02:00Z',
              id: 'job-saved-content',
              jobType: 'content_generation',
              progress: { completed: 0, total: 1 },
              reportId: 'rpt-saved-outline',
              status: 'running',
            },
            requestId: 'req-content-job',
          })
        }
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:01:00Z',
            id: 'job-saved-outline',
            jobType: 'outline_generation',
            progress: { completed: 1, total: 1 },
            reportId: 'rpt-saved-outline',
            status: 'succeeded',
          },
          requestId: 'req-outline-job',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-saved-outline')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:01:00Z',
            id: 'job-saved-outline',
            jobType: 'outline_generation',
            reportId: 'rpt-saved-outline',
            status: 'succeeded',
          },
          requestId: 'req-outline-status',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-saved-content')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:02:00Z',
            id: 'job-saved-content',
            jobType: 'content_generation',
            progress: { completed: 0, total: 1 },
            reportId: 'rpt-saved-outline',
            status: 'running',
          },
          requestId: 'req-content-status',
        })
      }
      if (
        request.method === 'PATCH' &&
        url.pathname.endsWith('/reports/rpt-saved-outline/outlines/outline-saved')
      ) {
        savedOutline = true
        requestOrder.push('save-outline')
        savedOutlines.push(await request.clone().json())
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            id: 'outline-saved',
            isCurrent: true,
            manualEdited: true,
            reportId: 'rpt-saved-outline',
            sections: [{ id: 'node-saved', level: 1, numbering: '1', title: 'Saved edited title' }],
            updatedAt: '2026-07-03T00:01:30Z',
            version: 1,
          },
          requestId: 'req-save-outline',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-saved-outline/outlines')) {
        return jsonResponse({
          data: [
            {
              createdAt: '2026-07-03T00:00:00Z',
              id: 'outline-saved',
              isCurrent: true,
              manualEdited: savedOutline,
              reportId: 'rpt-saved-outline',
              sections: [
                {
                  id: 'node-saved',
                  level: 1,
                  numbering: '1',
                  title: savedOutline ? 'Saved edited title' : 'Initial outline title',
                },
              ],
              updatedAt: savedOutline ? '2026-07-03T00:01:30Z' : '2026-07-03T00:00:00Z',
              version: 1,
            },
          ],
          requestId: 'req-outlines',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-saved-outline/sections')) {
        return jsonResponse({
          data: [
            {
              generationStatus: contentJobCreated ? 'running' : 'pending',
              id: 'section-saved',
              lastJobId: contentJobCreated ? 'job-saved-content' : undefined,
              outlineNodeId: 'node-saved',
              reportId: 'rpt-saved-outline',
              title: 'Initial section title',
            },
          ],
          requestId: 'req-sections',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-saved-outline/events')) {
        return jsonResponse({ data: [], requestId: 'req-events' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    const outlineTypeTrigger = screen.getAllByRole('combobox')[0]!
    await waitFor(() => expect(outlineTypeTrigger).not.toBeDisabled())
    fireEvent.click(outlineTypeTrigger)
    await screen.findByRole('option', { name: '真实巡检报告' })
    fireEvent.click(screen.getByRole('option', { name: '真实巡检报告' }))
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())
    fireEvent.click(screen.getByRole('button', { name: /创建草稿/ }))

    const title = await screen.findByDisplayValue('Initial outline title')
    fireEvent.change(title, { target: { value: 'Saved edited title' } })
    fireEvent.click(screen.getByRole('button', { name: /^生成正文$/ }))

    await waitFor(() => expect(savedOutlines).toHaveLength(1))
    await waitFor(() => expect(requestOrder).toEqual(['save-outline', 'content-job']))
    const sectionList = await screen.findByLabelText('章节列表')
    expect(within(sectionList).getByText('Saved edited title')).toBeVisible()
    expect(within(sectionList).queryByText('Initial section title')).not.toBeInTheDocument()
  })

  it('filters stale sections to the replacement outline after reusing a draft', async () => {
    let outlineJobCreates = 0
    let contentJobCreated = false
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports')) {
        return jsonResponse({
          data: {
            id: 'rpt-reuse-filter',
            name: '迎峰度夏报告',
            reportType: 'summer_peak_inspection',
            status: 'draft',
            templateId: 'tpl-real',
          },
          requestId: 'req-create-report',
        })
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports/rpt-reuse-filter/jobs')) {
        const body = (await request.clone().json()) as { jobType?: string }
        if (body.jobType === 'content_generation') {
          contentJobCreated = true
          return jsonResponse({
            data: {
              createdAt: '2026-07-03T00:04:00Z',
              id: 'job-reuse-content',
              jobType: 'content_generation',
              progress: { completed: 0, total: 1 },
              reportId: 'rpt-reuse-filter',
              status: 'running',
            },
            requestId: 'req-content-job',
          })
        }
        outlineJobCreates += 1
        const isReplacement = outlineJobCreates > 1
        return jsonResponse({
          data: {
            createdAt: isReplacement ? '2026-07-03T00:02:00Z' : '2026-07-03T00:00:00Z',
            finishedAt: isReplacement ? '2026-07-03T00:03:00Z' : '2026-07-03T00:01:00Z',
            id: isReplacement ? 'job-new-outline-filter' : 'job-old-outline-filter',
            jobType: 'outline_generation',
            progress: { completed: 1, total: 1 },
            reportId: 'rpt-reuse-filter',
            status: 'succeeded',
          },
          requestId: 'req-outline-job',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-old-outline-filter')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:01:00Z',
            id: 'job-old-outline-filter',
            jobType: 'outline_generation',
            reportId: 'rpt-reuse-filter',
            status: 'succeeded',
          },
          requestId: 'req-old-outline-status',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-new-outline-filter')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:02:00Z',
            finishedAt: '2026-07-03T00:03:00Z',
            id: 'job-new-outline-filter',
            jobType: 'outline_generation',
            reportId: 'rpt-reuse-filter',
            status: 'succeeded',
          },
          requestId: 'req-new-outline-status',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-reuse-content')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:04:00Z',
            id: 'job-reuse-content',
            jobType: 'content_generation',
            progress: { completed: 0, total: 1 },
            reportId: 'rpt-reuse-filter',
            status: 'running',
          },
          requestId: 'req-content-status',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-reuse-filter/outlines')) {
        const isReplacement = outlineJobCreates > 1
        return jsonResponse({
          data: [
            {
              createdAt: isReplacement ? '2026-07-03T00:02:00Z' : '2026-07-03T00:00:00Z',
              id: isReplacement ? 'outline-new-filter' : 'outline-old-filter',
              isCurrent: true,
              reportId: 'rpt-reuse-filter',
              sections: [
                {
                  id: isReplacement ? 'node-new-filter' : 'node-old-filter',
                  level: 1,
                  numbering: '1',
                  title: isReplacement ? 'New replacement section' : 'Old stale section',
                },
              ],
              source: 'ai',
              version: isReplacement ? 2 : 1,
            },
          ],
          requestId: 'req-outlines',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-reuse-filter/sections')) {
        return jsonResponse({
          data: [
            {
              generationStatus: 'succeeded',
              id: 'section-old-filter',
              outlineNodeId: 'node-old-filter',
              reportId: 'rpt-reuse-filter',
              title: 'Old stale section',
            },
            {
              generationStatus: contentJobCreated ? 'running' : 'pending',
              id: 'section-new-filter',
              lastJobId: contentJobCreated ? 'job-reuse-content' : undefined,
              outlineNodeId: 'node-new-filter',
              reportId: 'rpt-reuse-filter',
              title: 'New replacement section',
            },
          ],
          requestId: 'req-sections',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-reuse-filter/events')) {
        return jsonResponse({ data: [], requestId: 'req-events' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    const outlineTypeTrigger = screen.getAllByRole('combobox')[0]!
    await waitFor(() => expect(outlineTypeTrigger).not.toBeDisabled())
    fireEvent.click(outlineTypeTrigger)
    fireEvent.click(await screen.findByRole('option', { name: reportType.name }))
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())
    fireEvent.click(screen.getByRole('button', { name: /创建草稿/ }))

    expect(await screen.findByDisplayValue('Old stale section')).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: /草稿与大纲/ }))
    fireEvent.click(screen.getByRole('button', { name: /复用草稿生成大纲/ }))
    expect(await screen.findByDisplayValue('New replacement section')).toBeVisible()

    fireEvent.click(screen.getByRole('button', { name: /^生成正文$/ }))

    const sectionList = await screen.findByLabelText('章节列表')
    expect(await within(sectionList).findByText('New replacement section')).toBeVisible()
    expect(within(sectionList).queryByText('Old stale section')).not.toBeInTheDocument()
  })

  it('edits report outlines with add, delete, undo, redo, and a bounded undo history', async () => {
    const savedOutlines: unknown[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports')) {
        return jsonResponse({
          data: {
            id: 'rpt-outline-edit',
            name: '迎峰度夏报告',
            reportType: 'summer_peak_inspection',
            status: 'draft',
            templateId: 'tpl-real',
          },
          requestId: 'req-create-report',
        })
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports/rpt-outline-edit/jobs')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:01:00Z',
            id: 'job-outline-edit',
            jobType: 'outline_generation',
            progress: { completed: 1, total: 1 },
            reportId: 'rpt-outline-edit',
            status: 'succeeded',
          },
          requestId: 'req-outline-job',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-outline-edit')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:01:00Z',
            id: 'job-outline-edit',
            jobType: 'outline_generation',
            reportId: 'rpt-outline-edit',
            status: 'succeeded',
          },
          requestId: 'req-outline-status',
        })
      }
      if (
        request.method === 'PATCH' &&
        url.pathname.endsWith('/reports/rpt-outline-edit/outlines/outline-edit')
      ) {
        savedOutlines.push(await request.clone().json())
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            id: 'outline-edit',
            isCurrent: true,
            reportId: 'rpt-outline-edit',
            sections: [],
            source: 'manual',
            version: 2,
          },
          requestId: 'req-save-outline',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-outline-edit/outlines')) {
        return jsonResponse({
          data: [
            {
              createdAt: '2026-07-03T00:00:00Z',
              id: 'outline-edit',
              isCurrent: true,
              reportId: 'rpt-outline-edit',
              sections: [
                {
                  id: 'node-1',
                  level: 1,
                  numbering: '1',
                  title: '总览',
                  children: [{ id: 'node-1-1', level: 2, numbering: '1.1', title: '范围' }],
                },
                { id: 'node-2', level: 1, numbering: '2', title: '风险分析' },
              ],
              source: 'ai',
              version: 1,
            },
          ],
          requestId: 'req-outlines',
        })
      }
      if (
        url.pathname.endsWith('/reports/rpt-outline-edit/sections') ||
        url.pathname.endsWith('/reports/rpt-outline-edit/events')
      ) {
        return jsonResponse({ data: [], requestId: 'req-empty' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    const outlineTypeTrigger = screen.getAllByRole('combobox')[0]!
    await waitFor(() => expect(outlineTypeTrigger).not.toBeDisabled())
    fireEvent.click(outlineTypeTrigger)
    await screen.findByRole('option', { name: '真实巡检报告' })
    fireEvent.click(screen.getByRole('option', { name: '真实巡检报告' }))
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())
    fireEvent.click(screen.getByRole('button', { name: /创建草稿/ }))

    const firstTitle = await screen.findByDisplayValue('总览')
    const outlineList = firstTitle.closest('.space-y-2')
    expect(outlineList).toBeInstanceOf(HTMLElement)
    expect(outlineList).toHaveAttribute('aria-label', '大纲章节列表')
    expect(outlineList).not.toHaveClass('max-h-80')
    expect(outlineList).not.toHaveClass('overflow-y-auto')
    fireEvent.change(firstTitle, { target: { value: '总览修订' } })
    fireEvent.click(screen.getAllByRole('button', { name: /在此章节后新增同级章节/ })[0]!)
    expect(screen.getByDisplayValue('新章节')).toBeVisible()

    fireEvent.click(screen.getByRole('button', { name: /删除章节：风险分析/ }))
    expect(screen.queryByDisplayValue('风险分析')).not.toBeInTheDocument()
    fireEvent.keyDown(document, { ctrlKey: true, key: 'z' })
    expect(screen.getByDisplayValue('风险分析')).toBeVisible()
    fireEvent.keyDown(document, { ctrlKey: true, key: 'y' })
    expect(screen.queryByDisplayValue('风险分析')).not.toBeInTheDocument()

    const editedTitle = screen.getByDisplayValue('总览修订')
    for (let index = 1; index <= 16; index += 1) {
      fireEvent.change(editedTitle, { target: { value: `总览修订 ${index}` } })
    }
    for (let index = 0; index < 16; index += 1) {
      fireEvent.keyDown(document, { ctrlKey: true, key: 'z' })
    }
    expect(screen.getByDisplayValue('总览修订 1')).toBeVisible()

    fireEvent.click(screen.getByRole('button', { name: /保存大纲/ }))

    await waitFor(() => expect(savedOutlines).toHaveLength(1))
    const saved = savedOutlines[0] as {
      manualEdited?: boolean
      sections?: Array<{
        children?: Array<{ level: number; numbering?: string; title: string }>
        level: number
        numbering?: string
        title: string
      }>
    }
    expect(saved.manualEdited).toBe(true)
    expect(saved.sections?.[0]).toMatchObject({
      children: [{ level: 2, numbering: '1.1', title: '范围' }],
      level: 1,
      numbering: '1',
      title: '总览修订 1',
    })
    expect(saved.sections?.[1]).toMatchObject({ level: 1, numbering: '2', title: '新章节' })
    expect(saved.sections?.some((section) => section.title === '风险分析')).toBe(false)
  })

  it('reorders same-level outline sections with mouse drag and renumbers saved sections', async () => {
    const savedOutlines: unknown[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports')) {
        return jsonResponse({
          data: {
            id: 'rpt-outline-drag',
            name: '迎峰度夏报告',
            reportType: 'summer_peak_inspection',
            status: 'draft',
            templateId: 'tpl-real',
          },
          requestId: 'req-create-report',
        })
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports/rpt-outline-drag/jobs')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:01:00Z',
            id: 'job-outline-drag',
            jobType: 'outline_generation',
            progress: { completed: 1, total: 1 },
            reportId: 'rpt-outline-drag',
            status: 'succeeded',
          },
          requestId: 'req-outline-job',
        })
      }
      if (url.pathname.endsWith('/report-jobs/job-outline-drag')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            finishedAt: '2026-07-03T00:01:00Z',
            id: 'job-outline-drag',
            jobType: 'outline_generation',
            reportId: 'rpt-outline-drag',
            status: 'succeeded',
          },
          requestId: 'req-outline-status',
        })
      }
      if (
        request.method === 'PATCH' &&
        url.pathname.endsWith('/reports/rpt-outline-drag/outlines/outline-drag')
      ) {
        savedOutlines.push(await request.clone().json())
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            id: 'outline-drag',
            isCurrent: true,
            reportId: 'rpt-outline-drag',
            sections: [],
            source: 'manual',
            version: 2,
          },
          requestId: 'req-save-outline',
        })
      }
      if (url.pathname.endsWith('/reports/rpt-outline-drag/outlines')) {
        return jsonResponse({
          data: [
            {
              createdAt: '2026-07-03T00:00:00Z',
              id: 'outline-drag',
              isCurrent: true,
              reportId: 'rpt-outline-drag',
              sections: [
                {
                  id: 'node-1',
                  level: 1,
                  numbering: '1',
                  title: '总览',
                  children: [{ id: 'node-1-1', level: 2, numbering: '1.1', title: '范围' }],
                },
                { id: 'node-2', level: 1, numbering: '2', title: '风险分析' },
              ],
              source: 'ai',
              version: 1,
            },
          ],
          requestId: 'req-outlines',
        })
      }
      if (
        url.pathname.endsWith('/reports/rpt-outline-drag/sections') ||
        url.pathname.endsWith('/reports/rpt-outline-drag/events')
      ) {
        return jsonResponse({ data: [], requestId: 'req-empty' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    const outlineTypeTrigger = screen.getAllByRole('combobox')[0]!
    await waitFor(() => expect(outlineTypeTrigger).not.toBeDisabled())
    fireEvent.click(outlineTypeTrigger)
    await screen.findByRole('option', { name: '真实巡检报告' })
    fireEvent.click(screen.getByRole('option', { name: '真实巡检报告' }))
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())
    fireEvent.click(screen.getByRole('button', { name: /创建草稿/ }))

    await screen.findByDisplayValue('总览')
    const dragHandle = screen.getByRole('button', { name: /拖动章节调整顺序：风险分析/ })
    const dropTarget = screen.getByLabelText('大纲章节：总览')
    const dataTransfer = {
      dropEffect: '',
      effectAllowed: '',
      getData: vi.fn(() => ''),
      setData: vi.fn(),
    }

    fireEvent.dragStart(dragHandle, { dataTransfer })
    fireEvent.dragOver(dropTarget, { dataTransfer })
    fireEvent.drop(dropTarget, { dataTransfer })
    fireEvent.dragEnd(dragHandle, { dataTransfer })
    fireEvent.click(screen.getByRole('button', { name: /保存大纲/ }))

    await waitFor(() => expect(savedOutlines).toHaveLength(1))
    const saved = savedOutlines[0] as {
      sections?: Array<{
        children?: Array<{ numbering?: string; title: string }>
        numbering?: string
        title: string
      }>
    }
    expect(saved.sections?.map((section) => section.title)).toEqual(['风险分析', '总览'])
    expect(saved.sections?.[0]).toMatchObject({ numbering: '1', title: '风险分析' })
    expect(saved.sections?.[1]).toMatchObject({
      children: [{ numbering: '2.1', title: '范围' }],
      numbering: '2',
      title: '总览',
    })
  })

  it('shows gateway request id and does not create a local report when draft creation is not implemented', async () => {
    const fetchMock = vi.fn(async (request: RequestInfo | URL) => {
      const url = new URL(request instanceof Request ? request.url : String(request))

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }
      if (url.pathname.endsWith('/reports')) {
        return gatewayError(
          'not_implemented',
          'Real report creation is not ready',
          'req-create-501',
          501,
        )
      }

      return jsonResponse({ data: [], requestId: 'req-empty' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)
    const user = userEvent.setup()

    // Open the report-type Select and pick the first option
    const reportTrigger = screen.getAllByText('请选择报告类型')[0]!.closest('button')!
    await user.click(reportTrigger)
    const option = await screen.findByRole('option', { name: '真实巡检报告' })
    await user.click(option)
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())

    await user.click(screen.getByRole('button', { name: /创建草稿/ }))

    expect(await screen.findByText(/Real report creation is not ready/)).toBeVisible()
    expect(screen.getByText(/req-create-501/)).toBeVisible()
    expect(screen.queryByText(/local-report/)).not.toBeInTheDocument()
    expect(screen.queryByText(/已进入本地原型流程/)).not.toBeInTheDocument()
  })

  it('reuses an existing draft when outline job creation fails and the user retries', async () => {
    const reportCreatePaths: string[] = []
    const jobCreatePaths: string[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports')) {
        reportCreatePaths.push(url.pathname)
        return jsonResponse({
          data: {
            id: 'rpt-real',
            name: '迎峰度夏报告',
            reportType: 'summer_peak_inspection',
            status: 'draft',
          },
          requestId: 'req-create-report',
        })
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports/rpt-real/jobs')) {
        jobCreatePaths.push(url.pathname)
        return gatewayError('dependency_error', 'Outline job dependency down', 'req-job')
      }
      if (
        url.pathname.endsWith('/reports/rpt-real/outlines') ||
        url.pathname.endsWith('/reports/rpt-real/sections') ||
        url.pathname.endsWith('/reports/rpt-real/events')
      ) {
        return jsonResponse({ data: [], requestId: 'req-empty' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)
    const user = userEvent.setup()

    // Open the report-type Select and pick the first option
    const reportTrigger = screen.getAllByText('请选择报告类型')[0]!.closest('button')!
    await user.click(reportTrigger)
    const reportOption = await screen.findByRole('option', { name: '真实巡检报告' })
    await user.click(reportOption)
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())

    await user.click(screen.getByRole('button', { name: /创建草稿/ }))

    expect(await screen.findByText(/Outline job dependency down/)).toBeVisible()
    expect(screen.getByText(/req-job/)).toBeVisible()
    expect(await screen.findByText(/已保留报告草稿/)).toBeVisible()
    expect(screen.getByRole('button', { name: /复用草稿生成大纲/ })).toBeEnabled()

    await user.click(screen.getByRole('button', { name: /复用草稿生成大纲/ }))

    await waitFor(() => expect(jobCreatePaths).toHaveLength(2))
    expect(reportCreatePaths).toHaveLength(1)
  })

  it('restarts from a kept draft and clears stale progress and outline state', async () => {
    const reportCreatePaths: string[] = []
    const jobCreatePaths: string[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (url.pathname.endsWith('/report-types')) {
        return jsonResponse({ data: [reportType], requestId: 'req-types' })
      }
      if (url.pathname.endsWith('/report-templates')) {
        return pageResponse([reportTemplate])
      }
      if (url.pathname.endsWith('/report-materials')) {
        return pageResponse([reportMaterial])
      }
      if (request.method === 'POST' && url.pathname.endsWith('/reports')) {
        reportCreatePaths.push(url.pathname)
        const index = reportCreatePaths.length
        return jsonResponse({
          data: {
            id: `rpt-restart-${index}`,
            name: `迎峰度夏报告 ${index}`,
            reportType: 'summer_peak_inspection',
            status: 'draft',
            templateId: 'tpl-real',
          },
          requestId: `req-create-report-${index}`,
        })
      }
      if (
        request.method === 'POST' &&
        /^\/api\/v1\/reports\/rpt-restart-\d+\/jobs$/.test(url.pathname)
      ) {
        jobCreatePaths.push(url.pathname)
        return gatewayError('dependency_error', 'Outline job dependency down', 'req-job')
      }
      if (url.pathname.endsWith('/reports/rpt-restart-1/outlines')) {
        return jsonResponse({
          data: [
            {
              createdAt: '2026-07-03T00:00:00Z',
              id: 'outline-stale',
              isCurrent: true,
              reportId: 'rpt-restart-1',
              sections: [{ id: 'node-stale', level: 1, numbering: '1', title: '旧大纲' }],
              source: 'ai',
              version: 1,
            },
          ],
          requestId: 'req-outlines-stale',
        })
      }
      if (
        url.pathname.endsWith('/reports/rpt-restart-1/sections') ||
        url.pathname.endsWith('/reports/rpt-restart-1/events') ||
        url.pathname.endsWith('/reports/rpt-restart-2/outlines') ||
        url.pathname.endsWith('/reports/rpt-restart-2/sections') ||
        url.pathname.endsWith('/reports/rpt-restart-2/events')
      ) {
        return jsonResponse({ data: [], requestId: 'req-empty' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)
    const user = userEvent.setup()

    const reportTrigger = screen.getAllByText('请选择报告类型')[0]!.closest('button')!
    await user.click(reportTrigger)
    const reportOption = await screen.findByRole('option', { name: '真实巡检报告' })
    await user.click(reportOption)
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())

    await user.click(screen.getByRole('button', { name: /创建草稿/ }))

    expect(await screen.findByText(/已保留报告草稿/)).toBeVisible()
    expect(screen.getByText('当前文档进度')).toBeVisible()
    await waitFor(() => expect(window.sessionStorage.length).toBeGreaterThan(0))

    await user.click(screen.getByRole('button', { name: /编辑大纲/ }))
    expect(await screen.findByDisplayValue('旧大纲')).toBeVisible()

    await user.click(screen.getByRole('button', { name: /草稿与大纲/ }))
    expect(screen.getByRole('button', { name: /复用草稿生成大纲/ })).toBeEnabled()
    await user.click(screen.getByRole('button', { name: /重新开始/ }))

    await waitFor(() => expect(screen.queryByText('当前文档进度')).not.toBeInTheDocument())
    expect(screen.queryByText(/已保留报告草稿/)).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /复用草稿生成大纲/ })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled()
    expect(window.sessionStorage.length).toBe(0)

    await user.click(screen.getByRole('button', { name: /编辑大纲/ }))
    expect(screen.queryByDisplayValue('旧大纲')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /草稿与大纲/ }))
    await user.click(screen.getByRole('button', { name: /创建草稿/ }))

    await waitFor(() => expect(reportCreatePaths).toHaveLength(2))
    expect(jobCreatePaths).toEqual([
      '/api/v1/reports/rpt-restart-1/jobs',
      '/api/v1/reports/rpt-restart-2/jobs',
    ])
  })
})
