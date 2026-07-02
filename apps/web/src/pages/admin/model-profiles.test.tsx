import { fireEvent, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { ModelProfile } from '@/lib/types'
import { renderWithProviders } from '@/test/render'

import { ModelProfilesPage } from './model-profiles'

function jsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    status: init?.status ?? 200,
    statusText: init?.statusText,
  })
}

const modelProfile: ModelProfile = {
  apiKeyConfigured: true,
  baseUrl: 'https://api.example.com/v1',
  createdAt: '2026-06-30T00:00:00Z',
  defaultParameters: { max_tokens: 1024 },
  enabled: true,
  id: 'profile-chat',
  isDefault: true,
  model: 'gpt-4o-mini',
  name: 'Chat profile',
  provider: 'openai_compatible',
  purpose: 'chat',
  supportsStreaming: true,
  timeoutMs: 60000,
  updatedAt: '2026-06-30T00:00:00Z',
}

describe('ModelProfilesPage', () => {
  it('shows profile id, copies it, and explains the QA/LLM next step', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText },
    })

    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (request.method === 'GET' && url.pathname.endsWith('/admin/model-profiles')) {
        return jsonResponse({ data: [modelProfile], requestId: 'req-model-list' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ModelProfilesPage />)

    expect(await screen.findByText('Chat profile')).toBeVisible()
    expect(screen.getByText('Profile ID')).toBeVisible()
    expect(screen.getByText('profile-chat')).toBeVisible()
    expect(screen.getByText(/QA \/ LLM 配置/)).toBeVisible()

    fireEvent.click(screen.getByRole('button', { name: '复制 Profile ID profile-chat' }))

    await waitFor(() => expect(writeText).toHaveBeenCalledWith('profile-chat'))
    expect(await screen.findByText('Profile ID 已复制')).toBeVisible()
  })

  it('updates enabled and default status without sending an empty api key', async () => {
    const patchBodies: unknown[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (request.method === 'GET' && url.pathname.endsWith('/admin/model-profiles')) {
        return jsonResponse({ data: [modelProfile], requestId: 'req-model-list' })
      }

      if (
        request.method === 'PATCH' &&
        url.pathname.endsWith('/admin/model-profiles/profile-chat')
      ) {
        patchBodies.push(await request.clone().json())
        return jsonResponse({
          data: { ...modelProfile, enabled: false, isDefault: false },
          requestId: 'req-model-update',
        })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ModelProfilesPage />)

    expect(await screen.findByText('Chat profile')).toBeVisible()
    expect(screen.getByText('默认')).toBeVisible()

    fireEvent.click(screen.getByRole('button', { name: '编辑 Chat profile' }))

    fireEvent.click(screen.getByLabelText('启用'))
    fireEvent.click(screen.getByLabelText('设为默认模型'))
    fireEvent.click(screen.getByRole('button', { name: '保存' }))

    await waitFor(() => expect(patchBodies).toHaveLength(1))
    expect(patchBodies[0]).toMatchObject({
      enabled: false,
      isDefault: false,
      model: 'gpt-4o-mini',
      name: 'Chat profile',
    })
    expect(patchBodies[0]).not.toHaveProperty('apiKey')
  })
})
