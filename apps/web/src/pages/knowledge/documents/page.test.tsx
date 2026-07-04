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
  useUploadDocumentBatch,
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
  useUploadDocumentBatch: vi.fn(),
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

function selectFiles(files: File[]) {
  fireEvent.change(getFileInput(), { target: { files } })
}

function renderDocumentsPage({
  permissions = ['knowledge:write'],
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

  vi.mocked(useUploadDocumentBatch).mockReturnValue({
    isPending: uploadPending,
    mutate: uploadMutate,
  } as unknown as ReturnType<typeof useUploadDocumentBatch>)
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

describe('KnowledgeDocumentsPage document type labels', () => {
  it('derives labels from filenames when legacy responses still contain broad doc content type', () => {
    vi.mocked(useDocuments).mockReturnValue({
      data: {
        items: [
          createDocument({ contentType: 'doc', id: 'doc-docx', name: 'manual.docx' }),
          createDocument({ contentType: 'doc', id: 'doc-pdf', name: 'policy.pdf' }),
          createDocument({ contentType: 'doc', id: 'doc-csv', name: 'records.csv' }),
          createDocument({ contentType: 'doc', id: 'doc-pptx', name: 'slides.pptx' }),
          createDocument({ contentType: 'doc', id: 'doc-png', name: 'photo.png' }),
        ],
        page: { page: 1, pageSize: 20, total: 5 },
      },
      error: null,
      isError: false,
      isLoading: false,
      refetch: vi.fn(),
    } as unknown as ReturnType<typeof useDocuments>)

    renderDocumentsPage()

    expect(screen.getByText('DOCX')).toBeVisible()
    expect(screen.getByText('PDF')).toBeVisible()
    expect(screen.getByText('CSV')).toBeVisible()
    expect(screen.getByText('PPTX')).toBeVisible()
    expect(screen.getByText('PNG')).toBeVisible()
    expect(screen.queryByText('DOC')).not.toBeInTheDocument()
  })

  it('normalizes MIME values before rendering the label', () => {
    vi.mocked(useDocuments).mockReturnValue({
      data: {
        items: [
          createDocument({
            contentType: 'Application/PDF; charset=binary',
            id: 'doc-mime',
            name: 'unknown.bin',
          }),
        ],
        page: { page: 1, pageSize: 20, total: 1 },
      },
      error: null,
      isError: false,
      isLoading: false,
      refetch: vi.fn(),
    } as unknown as ReturnType<typeof useDocuments>)

    renderDocumentsPage()

    expect(screen.getByText('PDF')).toBeVisible()
  })
})

describe('KnowledgeDocumentsPage knowledge base selection', () => {
  it('shows a create-first prompt instead of an empty dropdown when no knowledge bases exist', () => {
    useAuthStore.setState({
      accessToken: 'token',
      error: null,
      status: 'authenticated',
      user: createUser(['knowledge:read']),
      userName: 'kevin',
    })
    vi.mocked(useKnowledgeBases).mockReturnValue({
      data: {
        filteredLocally: false,
        items: [],
        page: { page: 1, pageSize: 100, total: 0 },
      },
      error: null,
      isError: false,
      isLoading: false,
      refetch: vi.fn(),
    } as unknown as ReturnType<typeof useKnowledgeBases>)

    renderWithProviders(<KnowledgeDocumentsPage />)

    expect(screen.getByText('请先创建知识库')).toBeVisible()
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
  })
})

describe('KnowledgeDocumentsPage upload interactions', () => {
  it('hides the upload entry when the user lacks upload and write permissions', () => {
    renderDocumentsPage({ permissions: ['knowledge:read'] })

    expect(screen.queryByRole('button', { name: /上传文档/ })).not.toBeInTheDocument()
  })

  it('hides the upload entry when the user only has the legacy document upload permission', () => {
    renderDocumentsPage({ permissions: ['document:upload'] })

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
        files: [file],
        knowledgeBaseId: 'kb-1',
        tags: ['规程', '安全', '2024'],
      },
      expect.objectContaining({
        onError: expect.any(Function),
        onSuccess: expect.any(Function),
      }),
    )
  })

  it('adds every valid dropped file and submits them in one batch', async () => {
    const uploadMutate = vi.fn((_variables, options) => {
      options.onSuccess({
        failedCount: 0,
        results: [
          { document: { id: 'doc-1' }, filename: 'first.pdf', status: 'uploaded' },
          { document: { id: 'doc-2' }, filename: 'second.pdf', status: 'uploaded' },
        ],
        successCount: 2,
        totalCount: 2,
      })
    })
    const { user } = renderDocumentsPage({ uploadMutate })

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
    expect(within(dialog).getByText('second.pdf')).toBeInTheDocument()

    await user.click(within(dialog).getByRole('button', { name: /^上传 2 个文件$/ }))

    expect(uploadMutate).toHaveBeenCalledWith(
      {
        files: [first, second],
        knowledgeBaseId: 'kb-1',
        tags: [],
      },
      expect.objectContaining({
        onError: expect.any(Function),
        onSuccess: expect.any(Function),
      }),
    )
  })

  it('accepts CSV files in a batch selection', async () => {
    const { user } = renderDocumentsPage()

    await user.click(screen.getByRole('button', { name: /上传文档/ }))
    const dialog = getDialogContent()

    const csvFile = new File(['id,name\n1,A'], 'records.csv', { type: 'text/csv' })
    selectFiles([csvFile])

    expect(await within(dialog).findByText('records.csv')).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: /^上传$/ })).toBeEnabled()
  })

  it('shortens long upload filenames without rendering the full visible name', async () => {
    const { user } = renderDocumentsPage()

    await user.click(screen.getByRole('button', { name: /上传文档/ }))
    const dialog = getDialogContent()

    const longName = 'DB31-767-2013-super-long-power-standard-document.pdf'
    selectFile(new File(['manual'], longName, { type: 'application/pdf' }))

    const shortenedName = await within(dialog).findByText(/…\.pdf$/)
    expect(shortenedName).toBeInTheDocument()
    expect(shortenedName).toHaveAttribute('title', longName)
    expect(within(dialog).queryByText(longName)).not.toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: /^上传$/ })).toBeEnabled()
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

  it('keeps failed batch items available for retry after a partial success', async () => {
    const uploadMutate = vi.fn((_variables, options) => {
      options.onSuccess({
        failedCount: 1,
        results: [
          { document: { id: 'doc-1' }, filename: 'good.pdf', status: 'uploaded' },
          {
            error: { code: 'validation_error', message: 'file must not be empty' },
            filename: 'empty.pdf',
            status: 'failed',
          },
        ],
        successCount: 1,
        totalCount: 2,
      })
    })
    const { user } = renderDocumentsPage({ uploadMutate })

    await user.click(screen.getByRole('button', { name: /上传文档/ }))
    const dialog = getDialogContent()
    const good = new File(['good'], 'good.pdf', { type: 'application/pdf' })
    const empty = new File([], 'empty.pdf', { type: 'application/pdf' })
    selectFiles([good, empty])

    await user.click(within(dialog).getByRole('button', { name: /^上传 2 个文件$/ }))

    expect(await screen.findByText('部分文档上传失败，请检查失败项后重试')).toBeVisible()
    expect(within(dialog).queryByText('good.pdf')).not.toBeInTheDocument()
    expect(within(dialog).getByText('empty.pdf')).toBeInTheDocument()
    expect(within(dialog).getByText(/file must not be empty/)).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: /^上传$/ })).toBeEnabled()
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
