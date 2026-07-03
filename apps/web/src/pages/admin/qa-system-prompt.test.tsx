import { fireEvent, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { QAConfigVersion } from '@/features/qa-settings/qa-settings.types'
import { useAuthStore } from '@/stores/auth-store'
import { renderWithProviders } from '@/test/render'

import { QASystemPromptPage } from './qa-system-prompt'

function jsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    status: init?.status ?? 200,
    statusText: init?.statusText,
  })
}

const currentConfig: QAConfigVersion = {
  id: 'qa-current',
  versionNo: 4,
  defaultKnowledgeBaseIds: ['kb-main'],
  knowledgeBases: [{ id: 'kb-main', displayName: '主库', sortOrder: 0, type: 'default' }],
  retrieval: { enableRerank: true, rerankTopN: 3, topK: 5 },
  maxIterations: 4,
  toolTimeoutSeconds: 10,
  modelTimeoutSeconds: 60,
  overallTimeoutSeconds: 120,
  enabledToolNames: ['knowledge.search'],
  agent: {
    maxIterations: 4,
    toolTimeoutSeconds: 10,
    modelTimeoutSeconds: 60,
    overallTimeoutSeconds: 120,
    enabledToolNames: ['knowledge.search'],
  },
  systemPrompt: '旧全局提示词',
  isActive: true,
  createdBy: 'admin-user',
  createdAt: '2026-07-02T08:00:00Z',
}

function setUser(permissions: string[]) {
  useAuthStore.setState({
    accessToken: 'token',
    error: null,
    status: 'authenticated',
    user: { id: 'user-1', username: 'admin', roles: ['admin'], permissions },
    userName: 'admin',
  })
}

beforeEach(() => {
  setUser(['qa:settings:read', 'qa:settings:write'])
})

describe('QASystemPromptPage', () => {
  it('shows a read-only prompt editor without publish actions when write permission is absent', async () => {
    setUser(['qa:settings:read'])
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => jsonResponse({ data: currentConfig, requestId: 'req-current' })),
    )

    renderWithProviders(<QASystemPromptPage />)

    const editor = await screen.findByLabelText('全局 systemPrompt')
    expect(editor).toHaveValue('旧全局提示词')
    expect(editor).toHaveAttribute('readonly')
    expect(screen.getByText('admin-user')).toBeVisible()
    expect(screen.getByText('当前账号只有读取权限，页面为只读模式。')).toBeVisible()
    expect(screen.queryByRole('button', { name: '发布新版本' })).not.toBeInTheDocument()
  })

  it('blocks invalid prompts before submit', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => jsonResponse({ data: currentConfig, requestId: 'req-current' })),
    )
    renderWithProviders(<QASystemPromptPage />)

    const editor = await screen.findByLabelText('全局 systemPrompt')
    await userEvent.clear(editor)

    expect(screen.getByRole('button', { name: '发布新版本' })).toBeDisabled()
    expect(screen.getByText('系统提示词不能为空')).toBeVisible()

    fireEvent.change(editor, { target: { value: 'a'.repeat(20_001) } })
    expect(screen.getByText('系统提示词不能超过 20000 UTF-8 bytes')).toBeVisible()
    expect(screen.getByRole('button', { name: '发布新版本' })).toBeDisabled()
  })

  it('confirms and publishes a payload that preserves the current QA config fields', async () => {
    const postBodies: unknown[] = []
    const createdConfig = {
      ...currentConfig,
      id: 'qa-created',
      versionNo: 5,
      systemPrompt: '新提示词',
    }
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (request.method === 'GET' && url.pathname.endsWith('/qa-config-versions/current')) {
        return jsonResponse({ data: currentConfig, requestId: 'req-current' })
      }

      if (request.method === 'POST' && url.pathname.endsWith('/qa-config-versions')) {
        postBodies.push(await request.clone().json())
        return jsonResponse({ data: createdConfig, requestId: 'req-create' }, { status: 201 })
      }

      return jsonResponse({ data: null, requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<QASystemPromptPage />)

    const editor = await screen.findByLabelText('全局 systemPrompt')
    await userEvent.clear(editor)
    await userEvent.type(editor, '新提示词')
    await userEvent.click(screen.getByRole('button', { name: '发布新版本' }))
    expect(screen.getByText('当前版本：4')).toBeVisible()
    await userEvent.click(screen.getAllByRole('button', { name: '发布新版本' }).at(-1)!)

    await waitFor(() => expect(postBodies).toHaveLength(1))
    expect(postBodies[0]).toMatchObject({
      defaultKnowledgeBaseIds: ['kb-main'],
      knowledgeBases: currentConfig.knowledgeBases,
      retrieval: currentConfig.retrieval,
      maxIterations: 4,
      toolTimeoutSeconds: 10,
      modelTimeoutSeconds: 60,
      overallTimeoutSeconds: 120,
      enabledToolNames: ['knowledge.search'],
      agent: currentConfig.agent,
      systemPrompt: '新提示词',
      activate: true,
    })
    expect(await screen.findByText('Agent 提示词版本 5 已发布')).toBeVisible()
  })

  it('keeps the draft and displays requestId when publishing fails', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (request.method === 'GET' && url.pathname.endsWith('/qa-config-versions/current')) {
        return jsonResponse({ data: currentConfig, requestId: 'req-current' })
      }

      if (request.method === 'POST' && url.pathname.endsWith('/qa-config-versions')) {
        return jsonResponse(
          {
            error: {
              code: 'forbidden',
              message: 'permission denied',
              requestId: 'req-denied',
            },
          },
          { status: 403 },
        )
      }

      return jsonResponse({ data: null, requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<QASystemPromptPage />)

    const editor = await screen.findByLabelText('全局 systemPrompt')
    await userEvent.clear(editor)
    await userEvent.type(editor, '失败后保留的草稿')
    await userEvent.click(screen.getByRole('button', { name: '发布新版本' }))
    await userEvent.click(screen.getAllByRole('button', { name: '发布新版本' }).at(-1)!)

    expect(await screen.findByText(/requestId: req-denied/)).toBeVisible()
    expect(editor).toHaveValue('失败后保留的草稿')
  })

  it('keeps an unsaved draft when a concurrent refresh returns a newer server version', async () => {
    const refreshedConfig = {
      ...currentConfig,
      id: 'qa-refreshed',
      versionNo: 5,
      systemPrompt: '服务器新版本',
    }
    let getCount = 0
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => {
        getCount += 1
        return jsonResponse({
          data: getCount === 1 ? currentConfig : refreshedConfig,
          requestId: `req-current-${getCount}`,
        })
      }),
    )

    renderWithProviders(<QASystemPromptPage />)

    const editor = await screen.findByLabelText('全局 systemPrompt')
    await userEvent.clear(editor)
    await userEvent.type(editor, '本地未保存草稿')
    await userEvent.click(screen.getByRole('button', { name: '刷新' }))

    await screen.findByText('版本 5')
    expect(editor).toHaveValue('本地未保存草稿')
    expect(screen.getByText(/存在未保存变更/)).toBeVisible()
  })
})
