import { fireEvent, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { KnowledgeBaseSummary } from '@/api/knowledge'
import { getKnowledgeBase } from '@/api/knowledge'
import {
  formatGatewayCapabilityError,
  getGatewayCapabilityIssue,
  useDeleteDocument,
  useDocuments,
  useKnowledgeBases,
  useUpdateDocument,
  useUploadDocument,
} from '@/features/knowledge'
import type { DocumentSummary, UserSummary } from '@/lib/types'
import { useAuthStore } from '@/stores/auth-store'
import { renderWithProviders } from '@/test/render'

import { KnowledgeDocumentsPage } from './page'

vi.mock('@/api/knowledge', () => ({
  getDocumentContent: vi.fn(),
  getKnowledgeBase: vi.fn(),
}))

vi.mock('@/features/knowledge', () => ({
  formatGatewayCapabilityError: vi.fn(),
  getGatewayCapabilityIssue: vi.fn(),
  useDeleteDocument: vi.fn(),
  useDocuments: vi.fn(),
  useKnowledgeBases: vi.fn(),
  useUpdateDocument: vi.fn(),
  useUploadDocument: vi.fn(),
}))

function createDocument(overrides: Partial<DocumentSummary> = {}): DocumentSummary {
  return {
    chunkCount: 0,
    contentType: 'application/pdf',
    createdAt: '2026-07-01T00:00:00.000Z',
    createdBy: 'user-1',
    errorCode: null,
    errorMessage: null,
    id: 'doc-1',
    jobId: null,
    knowledgeBaseId: 'kb-1',
    name: 'Existing.pdf',
    parserBackend: null,
    sizeBytes: 2048,
    status: 'ready',
    tags: ['规程'],
    updatedAt: '2026-07-01T00:00:00.000Z',
    ...overrides,
  }
}

function createKnowledgeBase(overrides: Partial<KnowledgeBaseSummary> = {}): KnowledgeBaseSummary {
  return {
    chunkCount: 0,
    chunkStrategy: { chunkSize: 1600, overlap: 200, type: 'SEMANTIC_TEXT' },
    createdAt: '2026-07-01T00:00:00.000Z',
    createdBy: 'user-1',
    description: 'Operations manuals',
    docType: 'GENERAL',
    documentCount: 1,
    id: 'kb-1',
    name: 'Safety KB',
    retrievalStrategy: { mode: 'VECTOR', scoreThreshold: 0.35, topK: 10 },
    updatedAt: '2026-07-01T00:00:00.000Z',
    ...overrides,
  }
}

function createUser(permissions: string[]): UserSummary {
  return {
    id: 'user-1',
    permissions,
    roles: [],
    username: 'kevin',
  }
}

function getDialogContent(): HTMLElement {
  const content = document.querySelector('[data-slot="dialog-content"]')
  expect(content).toBeInstanceOf(HTMLElement)
  return content as HTMLElement
}

function getFileInput(): HTMLInputElement {
  const input = document.querySelector('input[type="file"]')
  expect(input).toBeInstanceOf(HTMLInputElement)
  return input as HTMLInputElement
}

function selectFile(file: File) {
  fireEvent.change(getFileInput(), { target: { files: [file] } })
}

function renderDocumentsPage({
  permissions = ['document:upload'],
  uploadMutate = vi.fn(),
  uploadPending = false,
}: {
  permissions?: string[]
  uploadMutate?: ReturnType<typeof vi.fn>
  uploadPending?: boolean
} = {}) {
  useAuthStore.setState({
    accessToken: 'token',
    error: null,
    status: 'authenticated',
    user: createUser(permissions),
    userName: 'kevin',
  })

  vi.mocked(useUploadDocument).mockReturnValue({
    isPending: uploadPending,
    mutate: uploadMutate,
  } as unknown as ReturnType<typeof useUploadDocument>)
  vi.mocked(useUpdateDocument).mockReturnValue({
    isPending: false,
    mutate: vi.fn(),
  } as unknown as ReturnType<typeof useUpdateDocument>)
  vi.mocked(useDeleteDocument).mockReturnValue({
    isPending: false,
    mutate: vi.fn(),
  } as unknown as ReturnType<typeof useDeleteDocument>)

  return {
    uploadMutate,
    user: userEvent.setup(),
    ...renderWithProviders(<KnowledgeDocumentsPage knowledgeBaseId="kb-1" />),
  }
}

beforeEach(() => {
  vi.mocked(getKnowledgeBase).mockResolvedValue(createKnowledgeBase())
  vi.mocked(formatGatewayCapabilityError).mockImplementation((error) => {
    const message = error instanceof Error ? error.message : 'unknown error'
    return `formatted-error: ${message}`
  })
  vi.mocked(getGatewayCapabilityIssue).mockReturnValue({
    description: 'mock description',
    kind: 'error',
    requestIdText: 'mock request id',
    title: 'mock title',
    variant: 'error',
  })
  vi.mocked(useDocuments).mockReturnValue({
    data: {
      items: [createDocument()],
      page: { page: 1, pageSize: 20, total: 1 },
    },
    error: null,
    isError: false,
    isLoading: false,
    refetch: vi.fn(),
  } as unknown as ReturnType<typeof useDocuments>)
  vi.mocked(useKnowledgeBases).mockReturnValue({
    data: {
      filteredLocally: false,
      items: [createKnowledgeBase()],
      page: { page: 1, pageSize: 100, total: 1 },
    },
    error: null,
    isError: false,
    isLoading: false,
    refetch: vi.fn(),
  } as unknown as ReturnType<typeof useKnowledgeBases>)
})

describe('KnowledgeDocumentsPage upload interactions', () => {
  it('hides the upload entry when the user lacks upload and write permissions', () => {
    renderDocumentsPage({ permissions: ['knowledge:read'] })

    expect(screen.queryByRole('button', { name: /上传文档/ })).not.toBeInTheDocument()
  })

  it('opens the upload dialog, accepts a valid file, and submits trimmed tags', async () => {
    const { uploadMutate, user } = renderDocumentsPage()

    await user.click(screen.getByRole('button', { name: /上传文档/ }))

    const dialog = getDialogContent()
    expect(within(dialog).getByRole('button', { name: /^上传$/ })).toBeDisabled()

    const file = new File(['manual content'], 'Manual.PDF', { type: 'application/pdf' })
    selectFile(file)

    expect(await within(dialog).findByText('Manual.PDF')).toBeInTheDocument()
    expect(within(dialog).getByText('14 B')).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: /^上传$/ })).toBeEnabled()

    await user.type(
      document.querySelector('#upload-tags') as HTMLInputElement,
      '规程, 安全, , 2024 ',
    )
    await user.click(within(dialog).getByRole('button', { name: /^上传$/ }))

    expect(uploadMutate).toHaveBeenCalledWith(
      {
        file,
        knowledgeBaseId: 'kb-1',
        tags: ['规程', '安全', '2024'],
      },
      expect.objectContaining({
        onError: expect.any(Function),
        onSuccess: expect.any(Function),
      }),
    )
  })

  it('uses only the first file when multiple files are dropped', async () => {
    const { user } = renderDocumentsPage()

    await user.click(screen.getByRole('button', { name: /上传文档/ }))
    const dialog = getDialogContent()
    const input = getFileInput()
    const dropZone = input.parentElement
    expect(dropZone).toBeInstanceOf(HTMLElement)

    const first = new File(['first'], 'first.pdf', { type: 'application/pdf' })
    const second = new File(['second'], 'second.pdf', { type: 'application/pdf' })
    fireEvent.drop(dropZone as HTMLElement, {
      dataTransfer: { files: [first, second] },
    })

    expect(await within(dialog).findByText('first.pdf')).toBeInTheDocument()
    expect(within(dialog).queryByText('second.pdf')).not.toBeInTheDocument()
  })

  it('shows an error for an unsupported extension and keeps the previous valid file selected', async () => {
    const { uploadMutate, user } = renderDocumentsPage()

    await user.click(screen.getByRole('button', { name: /上传文档/ }))
    const dialog = getDialogContent()

    selectFile(new File(['safe'], 'safe.pdf', { type: 'application/pdf' }))
    expect(await within(dialog).findByText('safe.pdf')).toBeInTheDocument()

    selectFile(new File(['bad'], 'virus.exe', { type: 'application/x-msdownload' }))

    expect(await screen.findByText(/\.exe/)).toBeVisible()
    expect(within(dialog).getByText('safe.pdf')).toBeInTheDocument()
    expect(within(dialog).queryByText('virus.exe')).not.toBeInTheDocument()
    expect(uploadMutate).not.toHaveBeenCalled()
  })

  it('keeps the dialog input state when upload mutation fails', async () => {
    const uploadMutate = vi.fn((_variables, options) => {
      options.onError(new Error('backend exploded'))
    })
    const { user } = renderDocumentsPage({ uploadMutate })

    await user.click(screen.getByRole('button', { name: /上传文档/ }))
    const dialog = getDialogContent()

    selectFile(new File(['manual'], 'Manual.PDF', { type: 'application/pdf' }))
    await user.type(document.querySelector('#upload-tags') as HTMLInputElement, '规程, 安全')
    await user.click(within(dialog).getByRole('button', { name: /^上传$/ }))

    expect(await screen.findByText('formatted-error: backend exploded')).toBeVisible()
    expect(within(dialog).getByText('Manual.PDF')).toBeInTheDocument()
    expect(document.querySelector('#upload-tags')).toHaveValue('规程, 安全')
  })

  it('disables upload and cancel controls while an upload is pending', async () => {
    const uploadMutate = vi.fn()
    const { user } = renderDocumentsPage({ uploadMutate, uploadPending: true })

    await user.click(screen.getByRole('button', { name: /上传文档/ }))
    const dialog = getDialogContent()

    selectFile(new File(['manual'], 'Manual.PDF', { type: 'application/pdf' }))

    const uploadButton = within(dialog).getByRole('button', { name: /^上传$/ })
    expect(uploadButton).toBeDisabled()
    expect(uploadButton.querySelector('.animate-spin')).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: /^取消$/ })).toBeDisabled()

    await user.click(uploadButton)
    expect(uploadMutate).not.toHaveBeenCalled()
  })
})
