import { fireEvent, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { ModelProfile } from '@/lib/types'
import { renderWithProviders } from '@/test/render'

import { QASettings } from './qa-settings'

function jsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    status: init?.status ?? 200,
    statusText: init?.statusText,
  })
}

const chatProfile: ModelProfile = {
  apiKeyConfigured: true,
  baseUrl: 'https://api.example.com/v1',
  createdAt: '2026-07-02T00:00:00Z',
  defaultParameters: {},
  enabled: true,
  id: 'mp_a27b266bfc922ff8995f5935',
  isDefault: true,
  model: 'gpt-5.5',
  name: '主聊天模型',
  provider: 'openai_compatible',
  purpose: 'chat',
  supportsStreaming: true,
  timeoutMs: 60000,
  updatedAt: '2026-07-02T00:00:00Z',
}

describe('QASettings', () => {
  it('selects an enabled chat profile and sends sanitized LLM test and publish payloads', async () => {
    const postBodies: Record<string, unknown>[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (request.method === 'GET' && url.pathname.endsWith('/qa-config-versions/current')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-02T08:00:00Z',
            id: 'qa-current-internal-id',
            isActive: true,
            retrieval: { enableRerank: false, topK: 5 },
            versionNo: 3,
          },
          requestId: 'req-qa-current',
        })
      }

      if (request.method === 'GET' && url.pathname.endsWith('/llm-config-versions/current')) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-02T08:30:00Z',
            id: 'llm-current-internal-id',
            isActive: true,
            modelName: 'old-model',
            profileId: 'old-profile',
            provider: 'ai-gateway',
            timeoutSeconds: 60,
            versionNo: 6,
          },
          requestId: 'req-llm-current',
        })
      }

      if (request.method === 'GET' && url.pathname.endsWith('/admin/model-profiles')) {
        expect(url.searchParams.get('purpose')).toBe('chat')
        expect(url.searchParams.get('enabled')).toBe('true')
        return jsonResponse({ data: [chatProfile], requestId: 'req-chat-profiles' })
      }

      if (request.method === 'POST' && url.pathname.endsWith('/llm-connection-tests')) {
        postBodies.push(await request.clone().json())
        return jsonResponse(
          {
            data: {
              id: 'test-1',
              latencyMs: 42,
              modelName: 'gpt-5.5',
              success: true,
              testedAt: '2026-07-02T08:45:00Z',
            },
            requestId: 'req-llm-test',
          },
          { status: 201 },
        )
      }

      if (request.method === 'POST' && url.pathname.endsWith('/llm-config-versions')) {
        postBodies.push(await request.clone().json())
        return jsonResponse(
          {
            data: {
              createdAt: '2026-07-02T08:50:00Z',
              id: 'llm-created-id',
              isActive: true,
              modelName: 'gpt-5.5',
              profileId: 'mp_a27b266bfc922ff8995f5935',
              provider: 'ai-gateway',
              timeoutSeconds: 60,
              versionNo: 7,
            },
            requestId: 'req-llm-create',
          },
          { status: 201 },
        )
      }

      return jsonResponse({ data: null, requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<QASettings />)

    const modelSelect = await screen.findByLabelText('聊天模型')
    expect(screen.getAllByText('生效中')).toHaveLength(2)
    expect(screen.queryByText('llm-current-internal-id')).not.toBeInTheDocument()
    expect(screen.queryByText('版本 6')).not.toBeInTheDocument()

    fireEvent.change(modelSelect, { target: { value: 'mp_a27b266bfc922ff8995f5935' } })

    expect(screen.getByText('mp_a27b266bfc922ff8995f5935')).toBeVisible()
    expect(screen.getByDisplayValue('gpt-5.5')).toBeVisible()

    fireEvent.click(screen.getByRole('button', { name: '测试连接' }))
    fireEvent.click(screen.getByRole('button', { name: '发布配置' }))

    await waitFor(() => expect(postBodies).toHaveLength(2))
    expect(postBodies[0]).toEqual({
      modelName: 'gpt-5.5',
      profileId: 'mp_a27b266bfc922ff8995f5935',
      provider: 'ai-gateway',
      timeoutSeconds: 60,
    })
    expect(postBodies[1]).toMatchObject({
      activate: true,
      modelName: 'gpt-5.5',
      profileId: 'mp_a27b266bfc922ff8995f5935',
      provider: 'ai-gateway',
      timeoutSeconds: 60,
    })
    expect(postBodies[0]).not.toHaveProperty('apiKey')
    expect(postBodies[0]).not.toHaveProperty('baseUrl')
    expect(postBodies[1]).not.toHaveProperty('apiKey')
    expect(postBodies[1]).not.toHaveProperty('baseUrl')
  })
})
