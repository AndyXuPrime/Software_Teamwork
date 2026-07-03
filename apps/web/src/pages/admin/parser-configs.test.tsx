import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { ParserConfig } from '@/lib/types'
import { renderWithProviders } from '@/test/render'

import { ParserConfigsPage } from './parser-configs'

function jsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    status: init?.status ?? 200,
    statusText: init?.statusText,
  })
}

const paddleOCRConfig: ParserConfig = {
  backend: 'paddleocr_cloud',
  concurrency: 4,
  createdAt: '2026-07-03T00:00:00Z',
  defaultParameters: {
    chunk_overlap: 64,
    chunk_size: 512,
    paddleocr_algorithm: 'PaddleOCR-VL-1.6',
    paddleocr_base_url: 'https://paddleocr.example.com/api',
    separators: [],
  },
  enabled: true,
  endpointUrl: null,
  id: 'parser-paddleocr',
  isDefault: false,
  name: 'PaddleOCR Cloud',
  paddleocrAccessTokenConfigured: true,
  supportedContentTypes: ['application/pdf'],
  updatedAt: '2026-07-03T00:00:00Z',
}

const builtinConfig: ParserConfig = {
  backend: 'builtin',
  concurrency: 4,
  createdAt: '2026-07-03T00:00:00Z',
  defaultParameters: {
    chunk_overlap: 64,
    chunk_size: 512,
    separators: [],
  },
  enabled: true,
  endpointUrl: null,
  id: 'parser-builtin',
  isDefault: true,
  name: 'Builtin Parser',
  paddleocrAccessTokenConfigured: false,
  supportedContentTypes: ['application/pdf'],
  updatedAt: '2026-07-03T00:00:00Z',
}

async function openCreateDialog() {
  renderWithProviders(<ParserConfigsPage />)
  expect(await screen.findByText('暂无解析器配置，点击新建配置开始')).toBeVisible()
  const createButton = screen.getAllByRole('button', { name: '新建配置' })[0]
  if (!createButton) throw new Error('create button not found')
  fireEvent.click(createButton)
  expect(await screen.findByText('新建解析器配置')).toBeVisible()
}

function chooseBackend(label: string) {
  fireEvent.click(screen.getByRole('combobox', { name: /解析后端/ }))
  fireEvent.click(screen.getByRole('option', { name: label }))
}

describe('ParserConfigsPage', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('renders PaddleOCR cloud fields and sends create payload with write-only token', async () => {
    const postBodies: unknown[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (request.method === 'GET' && url.pathname.endsWith('/admin/parser-configs')) {
        return jsonResponse({ data: [], requestId: 'req-parser-list' })
      }

      if (request.method === 'POST' && url.pathname.endsWith('/admin/parser-configs')) {
        postBodies.push(await request.clone().json())
        return jsonResponse({
          data: {
            ...paddleOCRConfig,
            id: 'parser-created',
            name: 'PaddleOCR Cloud Create',
          },
          requestId: 'req-parser-create',
        })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    await openCreateDialog()

    fireEvent.change(screen.getByLabelText(/名称/), {
      target: { value: 'PaddleOCR Cloud Create' },
    })
    chooseBackend('PaddleOCR 云端')

    expect(screen.queryByLabelText('远程地址')).not.toBeInTheDocument()
    expect(screen.getByLabelText(/API 地址/)).toBeVisible()
    const tokenInput = screen.getByLabelText(/API Token/) as HTMLInputElement
    expect(tokenInput.type).toBe('password')
    expect(screen.getByRole('button', { name: '创建' })).toBeDisabled()

    fireEvent.change(screen.getByLabelText(/API 地址/), {
      target: { value: 'https://paddleocr.example.com/api' },
    })
    expect(screen.getByRole('button', { name: '创建' })).toBeDisabled()

    fireEvent.change(tokenInput, { target: { value: 'sk-create-secret' } })
    fireEvent.change(screen.getByLabelText('模型名称'), {
      target: { value: 'PaddleOCR-VL-1.6' },
    })
    fireEvent.click(screen.getByRole('button', { name: '创建' }))

    await waitFor(() => expect(postBodies).toHaveLength(1))
    expect(postBodies[0]).toMatchObject({
      backend: 'paddleocr_cloud',
      defaultParameters: {
        paddleocr_access_token: 'sk-create-secret',
        paddleocr_algorithm: 'PaddleOCR-VL-1.6',
        paddleocr_base_url: 'https://paddleocr.example.com/api',
      },
      name: 'PaddleOCR Cloud Create',
    })
    expect(postBodies[0]).not.toHaveProperty('endpointUrl')
  })

  it('keeps the existing PaddleOCR token when editing with an empty password field', async () => {
    const patchBodies: unknown[] = []
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (request.method === 'GET' && url.pathname.endsWith('/admin/parser-configs')) {
        return jsonResponse({ data: [paddleOCRConfig], requestId: 'req-parser-list' })
      }

      if (
        request.method === 'PATCH' &&
        url.pathname.endsWith('/admin/parser-configs/parser-paddleocr')
      ) {
        patchBodies.push(await request.clone().json())
        return jsonResponse({ data: paddleOCRConfig, requestId: 'req-parser-update' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ParserConfigsPage />)

    expect(await screen.findByText('PaddleOCR Cloud')).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: '编辑 PaddleOCR Cloud' }))

    expect(await screen.findByText('编辑解析器配置')).toBeVisible()
    const tokenInput = screen.getByLabelText('API Token') as HTMLInputElement
    expect(tokenInput.type).toBe('password')
    expect(tokenInput.placeholder).toBe('留空保持不变')
    expect(tokenInput.value).toBe('')

    fireEvent.click(screen.getByRole('button', { name: '保存' }))

    await waitFor(() => expect(patchBodies).toHaveLength(1))
    expect(patchBodies[0]).toMatchObject({
      backend: 'paddleocr_cloud',
      defaultParameters: {
        paddleocr_algorithm: 'PaddleOCR-VL-1.6',
        paddleocr_base_url: 'https://paddleocr.example.com/api',
      },
      endpointUrl: null,
      name: 'PaddleOCR Cloud',
    })
    expect(
      (patchBodies[0] as { defaultParameters?: Record<string, unknown> }).defaultParameters,
    ).not.toHaveProperty('paddleocr_access_token')
  })

  it('requires a token when editing another backend into PaddleOCR cloud', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)

      if (request.method === 'GET' && url.pathname.endsWith('/admin/parser-configs')) {
        return jsonResponse({ data: [builtinConfig], requestId: 'req-parser-list' })
      }

      return jsonResponse({ data: [], requestId: 'req-default' })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ParserConfigsPage />)

    expect(await screen.findByText('Builtin Parser')).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: '编辑 Builtin Parser' }))
    chooseBackend('PaddleOCR 云端')
    fireEvent.change(screen.getByLabelText(/API 地址/), {
      target: { value: 'https://paddleocr.example.com/api' },
    })

    const tokenInput = screen.getByLabelText(/API Token/) as HTMLInputElement
    expect(tokenInput.type).toBe('password')
    expect(screen.getByRole('button', { name: '保存' })).toBeDisabled()

    fireEvent.change(tokenInput, { target: { value: 'sk-rotate-secret' } })
    expect(screen.getByRole('button', { name: '保存' })).toBeEnabled()
  })
})
