import { fireEvent, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { ModelProfile } from '@/lib/types'
import { renderWithProviders } from '@/test/render'

import { ReportDocumentModelSettingsPage } from './report-document-model-settings'

function jsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    status: init?.status ?? 200,
    statusText: init?.statusText,
  })
}

function deferredResponse<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => {
    resolve = done
  })
  return { promise, resolve }
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

describe('ReportDocumentModelSettingsPage', () => {
  it('publishes the selected document generation model profile through report settings', async () => {
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
      }),
    )

    renderWithProviders(<ReportDocumentModelSettingsPage />)

    expect(await screen.findByText('old-report-profile')).toBeVisible()
    const modelTrigger = screen.getByLabelText('文档生成模型')
    fireEvent.click(modelTrigger)
    fireEvent.click(await screen.findByRole('option', { name: /报告生成模型/ }))
    fireEvent.click(screen.getByRole('button', { name: /发布文档模型配置/ }))

    await waitFor(() => expect(patchBodies).toHaveLength(1))
    expect(patchBodies[0]).toEqual({
      llm: { profileId: 'mp-chat-report', provider: 'ai-gateway' },
    })
    expect(patchBodies[0]).not.toHaveProperty('apiKey')
    expect(patchBodies[0]).not.toHaveProperty('baseUrl')
    expect(await screen.findByText(/文档生成模型配置已发布/)).toBeVisible()
  })

  it('disables publishing when no enabled chat profiles exist', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init)
        const url = new URL(request.url)

        if (request.method === 'GET' && url.pathname.endsWith('/report-settings')) {
          return jsonResponse({
            data: { llm: { provider: 'ai-gateway' } },
            requestId: 'req-report-settings',
          })
        }
        if (request.method === 'GET' && url.pathname.endsWith('/admin/model-profiles')) {
          return jsonResponse({ data: [], requestId: 'req-chat-profiles' })
        }

        return jsonResponse({ data: [], requestId: 'req-empty' })
      }),
    )

    renderWithProviders(<ReportDocumentModelSettingsPage />)

    const modelTrigger = await screen.findByLabelText('文档生成模型')
    const publishButton = screen.getByRole('button', { name: /发布文档模型配置/ })

    await waitFor(() => expect(modelTrigger).toBeDisabled())
    expect(modelTrigger).toHaveTextContent('请先创建聊天模型 Profile')
    expect(publishButton).toBeDisabled()
    expect(await screen.findByText(/请先在模型管理中新增并启用用途为 chat/)).toBeVisible()
  })

  it('waits for report settings before defaulting and publishing a document profile', async () => {
    const settings = deferredResponse<Response>()
    const patchBodies: unknown[] = []
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init)
        const url = new URL(request.url)

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
      }),
    )

    renderWithProviders(<ReportDocumentModelSettingsPage />)

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
})
