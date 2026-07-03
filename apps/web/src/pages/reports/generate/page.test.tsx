import { fireEvent, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import type { ModelProfile, UserSummary } from '@/lib/types'
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

const chatProfile: ModelProfile = {
  apiKeyConfigured: true,
  baseUrl: 'https://api.example.com/v1',
  createdAt: '2026-07-03T00:00:00Z',
  defaultParameters: {},
  enabled: true,
  id: 'mp-chat-report',
  isDefault: false,
  model: 'gpt-report',
  name: '报告生成模型',
  provider: 'openai_compatible',
  purpose: 'chat',
  supportsStreaming: true,
  timeoutMs: 60000,
  updatedAt: '2026-07-03T00:00:00Z',
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
    fireEvent.click(screen.getByRole('combobox'))
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

  it('publishes the selected document generation model profile through report settings', async () => {
    setAuthenticatedUser(['report:write', 'admin:model-profile:write'])
    const patchBodies: unknown[] = []
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
      if (request.method === 'GET' && url.pathname.endsWith('/report-settings')) {
        return jsonResponse({
          data: {
            llm: {
              model: 'old-report-model',
              profileId: 'old-report-profile',
              provider: 'ai-gateway',
              timeoutSeconds: 60,
            },
          },
          requestId: 'req-report-settings',
        })
      }
      if (request.method === 'GET' && url.pathname.endsWith('/admin/model-profiles')) {
        expect(url.searchParams.get('purpose')).toBe('chat')
        expect(url.searchParams.get('enabled')).toBe('true')
        return jsonResponse({ data: [chatProfile], requestId: 'req-chat-profiles' })
      }
      if (request.method === 'PATCH' && url.pathname.endsWith('/report-settings')) {
        patchBodies.push(await request.clone().json())
        return jsonResponse({
          data: { updatedAt: '2026-07-03T08:00:00Z' },
          requestId: 'req-report-settings-update',
        })
      }

      return jsonResponse({ data: [], requestId: 'req-empty' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    const modelTrigger = screen.getByLabelText('文档生成模型')
    expect(await screen.findByText('old-report-profile')).toBeVisible()

    // Open the document model Select and pick the chat profile
    fireEvent.click(modelTrigger)
    const option = await screen.findByRole('option', { name: /报告生成模型/ })
    fireEvent.click(option)
    fireEvent.click(screen.getByRole('button', { name: /发布文档模型配置/ }))

    await waitFor(() => expect(patchBodies).toHaveLength(1))
    expect(patchBodies[0]).toEqual({
      llm: { profileId: 'mp-chat-report', provider: 'ai-gateway' },
    })
    expect(patchBodies[0]).not.toHaveProperty('apiKey')
    expect(patchBodies[0]).not.toHaveProperty('baseUrl')
    expect(await screen.findByText(/文档生成模型配置已发布/)).toBeVisible()
  })

  it('shows the current user model config to non-admin report writers without admin-only requests', async () => {
    setAuthenticatedUser(['report:write'])
    const paths: string[] = []
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
    fireEvent.click(trigger)
    await screen.findByRole('option', { name: '真实巡检报告' })
    fireEvent.click(screen.getByRole('option', { name: '真实巡检报告' }))
    expect(await screen.findByText('当前 LLM 配置')).toBeVisible()
    expect(screen.getByText('mp-user-chat')).toBeVisible()
    expect(screen.getByText('gpt-user')).toBeVisible()
    expect(screen.queryByRole('button', { name: /发布文档模型配置/ })).not.toBeInTheDocument()
    await waitFor(() => expect(screen.getByRole('button', { name: /创建草稿/ })).toBeEnabled())

    fireEvent.click(screen.getByRole('button', { name: /创建草稿/ }))

    expect(await screen.findByText('job-writer')).toBeVisible()
    expect(paths.some((path) => path.includes('/report-settings'))).toBe(false)
    expect(paths.some((path) => path.includes('/admin/model-profiles'))).toBe(false)
  })

  it('waits for report settings before defaulting and publishing a document profile', async () => {
    setAuthenticatedUser(['report:write', 'admin:model-profile:write'])
    const settings = deferredResponse<Response>()
    const patchBodies: unknown[] = []
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
      if (request.method === 'GET' && url.pathname.endsWith('/report-settings')) {
        return settings.promise
      }
      if (request.method === 'GET' && url.pathname.endsWith('/admin/model-profiles')) {
        return jsonResponse({ data: [chatProfile], requestId: 'req-chat-profiles' })
      }
      if (request.method === 'PATCH' && url.pathname.endsWith('/report-settings')) {
        patchBodies.push(await request.clone().json())
        return jsonResponse({
          data: { updatedAt: '2026-07-03T08:00:00Z' },
          requestId: 'req-report-settings-update',
        })
      }

      return jsonResponse({ data: [], requestId: 'req-empty' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ReportGeneratePage />)

    const publishButton = await screen.findByRole('button', { name: /发布文档模型配置/ })
    expect(publishButton).toBeDisabled()
    expect(screen.getByLabelText('文档生成模型')).toHaveTextContent('请选择聊天模型 Profile')

    settings.resolve(
      jsonResponse({
        data: {
          llm: {
            model: 'old-report-model',
            profileId: 'old-report-profile',
            provider: 'ai-gateway',
          },
        },
        requestId: 'req-report-settings',
      }),
    )

    await waitFor(() =>
      expect(screen.getByLabelText('文档生成模型')).toHaveTextContent(/old-report/),
    )
    fireEvent.click(publishButton)

    await waitFor(() => expect(patchBodies).toHaveLength(1))
    expect(patchBodies[0]).toEqual({
      llm: { profileId: 'old-report-profile', provider: 'ai-gateway' },
    })
  })

  it('renders job progress in the current report area without side task or event panels', async () => {
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
        return jsonResponse({
          data: {
            createdAt: '2026-07-03T00:00:00Z',
            id: 'job-1',
            jobType: 'outline_generation',
            progress: { completedSections: 1, percent: 50, totalSections: 2 },
            reportId: 'rpt-progress',
            resultSummary: '已生成大纲初稿',
            status: 'running',
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
            progress: { completedSections: 1, percent: 50, totalSections: 2 },
            reportId: 'rpt-progress',
            resultSummary: '已生成大纲初稿',
            status: 'running',
          },
          requestId: 'req-job-status',
        })
      }
      if (
        url.pathname.endsWith('/reports/rpt-progress/outlines') ||
        url.pathname.endsWith('/reports/rpt-progress/sections')
      ) {
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

    expect(await screen.findByText('job-1')).toBeVisible()
    expect(screen.getByText(/50%/)).toBeVisible()
    expect(screen.getByText('已生成大纲初稿')).toBeVisible()
    expect(screen.queryByText('任务状态')).not.toBeInTheDocument()
    expect(screen.queryByText('事件日志')).not.toBeInTheDocument()
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
    const reportTrigger = screen.getByText('请选择报告类型').closest('button')!
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
    const reportTrigger = screen.getByText('请选择报告类型').closest('button')!
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
})
