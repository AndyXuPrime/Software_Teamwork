import {
  Check,
  ChevronLeft,
  ChevronRight,
  Download,
  Edit,
  Eye,
  FileText,
  Loader2,
  Search,
  Trash2,
  Upload,
  X,
} from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'

import { getDocumentContent, getKnowledgeBase } from '@/api/knowledge'
import { ConfirmDialog, InlineNotice, StateBlock, TableSkeleton } from '@/components/common'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  formatGatewayCapabilityError,
  getGatewayCapabilityIssue,
  useDeleteDocument,
  useDocuments,
  useKnowledgeBases,
  useUpdateDocument,
  useUploadDocumentBatch,
} from '@/features/knowledge'
import { canAccess } from '@/lib/permissions'
import type { DocumentStatus, DocumentSummary } from '@/lib/types'
import { useAuthStore } from '@/stores/auth-store'

// ── Constants ──

const PAGE_SIZE = 20
const KB_NAME_CACHE: Record<string, string> = {}

const ALLOWED_EXTENSIONS = [
  '.pdf',
  '.doc',
  '.docx',
  '.ppt',
  '.pptx',
  '.xls',
  '.xlsx',
  '.csv',
  '.md',
  '.txt',
  '.png',
  '.jpg',
  '.jpeg',
  '.gif',
  '.bmp',
  '.webp',
]

const ALLOWED_MIME_TYPES = [
  'application/pdf',
  'application/msword',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  'application/vnd.ms-powerpoint',
  'application/vnd.openxmlformats-officedocument.presentationml.presentation',
  'application/vnd.ms-excel',
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  'text/csv',
  'text/markdown',
  'text/plain',
  'image/png',
  'image/jpeg',
  'image/gif',
  'image/bmp',
  'image/webp',
]

const FILE_ACCEPT_TYPES = [...ALLOWED_EXTENSIONS, ...ALLOWED_MIME_TYPES].join(',')
const SUPPORTED_FILE_TYPES_TEXT = 'PDF, DOC/DOCX, PPT/PPTX, XLS/XLSX, CSV, MD, TXT, 图片'
const UPLOAD_FILE_NAME_MAX_CHARS = 32

const FILE_TYPE_LABELS_BY_EXTENSION: Record<string, string> = {
  '.pdf': 'PDF',
  '.doc': 'DOC',
  '.docx': 'DOCX',
  '.ppt': 'PPT',
  '.pptx': 'PPTX',
  '.xls': 'XLS',
  '.xlsx': 'XLSX',
  '.csv': 'CSV',
  '.md': 'MD',
  '.markdown': 'MD',
  '.mdx': 'MD',
  '.txt': 'TXT',
  '.png': 'PNG',
  '.jpg': 'JPG',
  '.jpeg': 'JPG',
  '.gif': 'GIF',
  '.bmp': 'BMP',
  '.webp': 'WEBP',
}

const FILE_TYPE_LABELS_BY_MIME: Record<string, string> = {
  'application/pdf': 'PDF',
  'application/msword': 'DOC',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document': 'DOCX',
  'application/vnd.ms-powerpoint': 'PPT',
  'application/vnd.openxmlformats-officedocument.presentationml.presentation': 'PPTX',
  'application/vnd.ms-excel': 'XLS',
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': 'XLSX',
  'text/csv': 'CSV',
  'text/markdown': 'MD',
  'text/plain': 'TXT',
  'image/png': 'PNG',
  'image/jpeg': 'JPG',
  'image/gif': 'GIF',
  'image/bmp': 'BMP',
  'image/webp': 'WEBP',
}

const WEAK_CONTENT_TYPE_VALUES = new Set([
  'application/octet-stream',
  'doc',
  'visual',
  'aural',
  'other',
])

const STATUS_LABELS: Record<DocumentStatus, string> = {
  uploaded: '已上传',
  parsing: '解析中',
  chunking: '分块中',
  embedding: '向量化中',
  ready: '就绪',
  failed: '失败',
}

const STATUS_VARIANTS: Record<DocumentStatus, 'default' | 'secondary' | 'destructive' | 'outline'> =
  {
    uploaded: 'secondary',
    parsing: 'default',
    chunking: 'default',
    embedding: 'default',
    ready: 'outline',
    failed: 'destructive',
  }

/** Sorted statuses for the filter dropdown. */
const FILTERABLE_STATUSES: (DocumentStatus | '')[] = [
  '',
  'uploaded',
  'parsing',
  'chunking',
  'embedding',
  'ready',
  'failed',
]

const PROCESSING_STATUSES: DocumentStatus[] = ['parsing', 'chunking', 'embedding']

const MAX_BATCH_FILES = 10

type UploadItemStatus = 'queued' | 'uploaded' | 'failed'

type UploadItem = {
  id: string
  file: File
  status: UploadItemStatus
  message?: string
  documentId?: string
}

// ── Helpers ──

function formatSize(bytes?: number): string {
  if (!bytes || bytes <= 0) return '-'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatUploadFileName(name: string): string {
  const chars = Array.from(name)
  if (chars.length <= UPLOAD_FILE_NAME_MAX_CHARS) return name

  const extensionIndex = name.lastIndexOf('.')
  const extension = extensionIndex > 0 ? name.slice(extensionIndex) : ''
  const stem = extension ? name.slice(0, -extension.length) : name
  const suffix = extension || chars.slice(-8).join('')
  const prefixLength = Math.max(12, UPLOAD_FILE_NAME_MAX_CHARS - Array.from(suffix).length - 1)
  const prefix = Array.from(stem).slice(0, prefixLength).join('')

  return `${prefix}…${suffix}`
}

function formatDateTime(iso?: string | null): string {
  if (!iso) return '-'
  try {
    return new Date(iso).toLocaleDateString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return iso
  }
}

function normalizeContentType(value?: string | null): string {
  return (value ?? '').split(';', 1)[0]?.trim().toLowerCase() ?? ''
}

function fileExtensionFromName(filename?: string | null): string {
  const name = filename?.trim()
  if (!name) return ''
  const dotIndex = name.lastIndexOf('.')
  if (dotIndex < 0 || dotIndex === name.length - 1) return ''
  return name.slice(dotIndex).toLowerCase()
}

function documentTypeLabel(doc: Pick<DocumentSummary, 'contentType' | 'name'>): string {
  const contentType = normalizeContentType(doc.contentType)
  const extensionLabel = FILE_TYPE_LABELS_BY_EXTENSION[fileExtensionFromName(doc.name)]
  if (extensionLabel && (!contentType || WEAK_CONTENT_TYPE_VALUES.has(contentType))) {
    return extensionLabel
  }

  const mimeLabel = FILE_TYPE_LABELS_BY_MIME[contentType]
  if (mimeLabel) return mimeLabel
  if (extensionLabel) return extensionLabel
  if (!contentType || WEAK_CONTENT_TYPE_VALUES.has(contentType)) return '-'
  return doc.contentType?.trim() ?? '-'
}

function isProcessing(status: DocumentStatus): boolean {
  return PROCESSING_STATUSES.includes(status)
}

function newUploadItemId(file: File, index: number): string {
  const random =
    typeof crypto !== 'undefined' && 'randomUUID' in crypto
      ? crypto.randomUUID()
      : `${Date.now()}-${Math.random().toString(36).slice(2)}`
  return `${file.name}-${file.size}-${file.lastModified}-${index}-${random}`
}

// ── Main component ──

interface KnowledgeDocumentsPageProps {
  knowledgeBaseId?: string
  onNavigateChunks?: (documentId: string, knowledgeBaseId: string) => void
}

export function KnowledgeDocumentsPage({
  knowledgeBaseId: initialKbId,
  onNavigateChunks,
}: KnowledgeDocumentsPageProps) {
  // ── State ──
  const [keyword, setKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<DocumentStatus | ''>('')
  const [page, setPage] = useState(1)
  const [activeKbId, setActiveKbId] = useState(initialKbId ?? '')
  const knowledgeBaseId = activeKbId

  // Sync when URL param changes (navigating between knowledge bases)
  useEffect(() => {
    if (initialKbId && initialKbId !== activeKbId) {
      setActiveKbId(initialKbId)
      setPage(1)
      setKeyword('')
      setStatusFilter('')
    }
  }, [initialKbId]) // eslint-disable-line

  const [kbName, setKbName] = useState(
    initialKbId ? (KB_NAME_CACHE[initialKbId] ?? initialKbId) : '',
  )
  const [notification, setNotification] = useState<{
    type: 'success' | 'error'
    text: string
  } | null>(null)

  // Upload state
  const [uploadOpen, setUploadOpen] = useState(false)
  const [uploadItems, setUploadItems] = useState<UploadItem[]>([])
  const [uploadTags, setUploadTags] = useState('')
  const [dragOver, setDragOver] = useState(false)
  const [showSuccess, setShowSuccess] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Edit tags state
  const [editOpen, setEditOpen] = useState(false)
  const [editingDoc, setEditingDoc] = useState<DocumentSummary | null>(null)
  const [editTagsText, setEditTagsText] = useState('')

  // Delete state
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deletingDoc, setDeletingDoc] = useState<DocumentSummary | null>(null)

  // ── Queries & mutations ──

  const { data, isLoading, isError, error, refetch } = useDocuments(
    knowledgeBaseId,
    page,
    PAGE_SIZE,
    statusFilter || undefined,
  )

  const uploadMutation = useUploadDocumentBatch()
  const updateMutation = useUpdateDocument()
  const deleteMutation = useDeleteDocument()

  const {
    data: kbListData,
    error: kbListError,
    isError: isKbListError,
    isLoading: isKbListLoading,
    refetch: refetchKbList,
  } = useKnowledgeBases(1, 100)

  const isMutating =
    uploadMutation.isPending || updateMutation.isPending || deleteMutation.isPending

  // ── Permissions ──

  const user = useAuthStore((s) => s.user)
  const canUpload = canAccess(user, {
    any: ['knowledge:write', 'knowledge:admin', 'system:admin'],
  })
  const canEditTags = canAccess(user, { any: ['knowledge:write'] })
  const canDelete = canAccess(user, { any: ['knowledge:write'] })

  // ── Fetch KB name ──

  useEffect(() => {
    if (!knowledgeBaseId) return
    if (KB_NAME_CACHE[knowledgeBaseId]) {
      setKbName(KB_NAME_CACHE[knowledgeBaseId])
      return
    }
    let cancelled = false
    getKnowledgeBase(knowledgeBaseId)
      .then((kb) => {
        if (!cancelled) {
          KB_NAME_CACHE[knowledgeBaseId] = kb.name
          setKbName(kb.name)
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setNotification({
            type: 'error',
            text: formatGatewayCapabilityError(err, '知识库详情'),
          })
        }
      })
    return () => {
      cancelled = true
    }
  }, [knowledgeBaseId])

  // ── Notification auto-dismiss ──

  useEffect(() => {
    if (!notification) return
    const timer = setTimeout(() => setNotification(null), 4000)
    return () => clearTimeout(timer)
  }, [notification])

  // ── Upload success auto-dismiss ──

  useEffect(() => {
    if (!showSuccess) return
    const timer = setTimeout(() => setShowSuccess(false), 1500)
    return () => clearTimeout(timer)
  }, [showSuccess])

  // ── Clear success when upload dialog opens/closes ──

  useEffect(() => {
    setShowSuccess(false)
  }, [uploadOpen])

  // ── Derived ──

  const totalPages = data ? Math.max(1, Math.ceil(data.page.total / PAGE_SIZE)) : 1
  const showPagination = totalPages > 1
  const isEmpty = !isLoading && !isError && data && data.items.length === 0
  const availableKnowledgeBases = kbListData?.items ?? []
  const hasKnowledgeBases = availableKnowledgeBases.length > 0
  const documentListIssue = isError ? getGatewayCapabilityIssue(error, '文档列表') : null
  const knowledgeBaseListIssue =
    !knowledgeBaseId && isKbListError ? getGatewayCapabilityIssue(kbListError, '知识库列表') : null
  const submittableUploadItems = uploadItems.filter((item) => item.status !== 'uploaded')

  // ── Filtered items (client-side keyword search) ──

  const items = data?.items
  const filteredItems = keyword
    ? items?.filter(
        (d) =>
          d.name.toLowerCase().includes(keyword.toLowerCase()) ||
          (d.tags ?? []).some((t) => t.toLowerCase().includes(keyword.toLowerCase())),
      )
    : items

  // ── Handlers ──

  const validateFile = useCallback((file: File): string | null => {
    const ext = '.' + file.name.split('.').pop()?.toLowerCase()
    if (!ALLOWED_EXTENSIONS.includes(ext)) {
      return `不支持的文件类型 "${ext}"。支持: ${SUPPORTED_FILE_TYPES_TEXT}`
    }
    return null
  }, [])

  const handleFilesSelect = useCallback(
    (files: File[] | FileList) => {
      const incoming = Array.from(files)
      if (incoming.length === 0) return

      const validItems: UploadItem[] = []
      const rejected: string[] = []
      incoming.forEach((file, index) => {
        const err = validateFile(file)
        if (err) {
          rejected.push(`${file.name}: ${err}`)
          return
        }
        validItems.push({
          id: newUploadItemId(file, index),
          file,
          status: 'queued',
        })
      })

      if (validItems.length > 0) {
        setUploadItems((current) => {
          const availableSlots = Math.max(0, MAX_BATCH_FILES - current.length)
          const accepted = validItems.slice(0, availableSlots)
          if (accepted.length < validItems.length) {
            setNotification({
              type: 'error',
              text: `单次最多上传 ${MAX_BATCH_FILES} 个文件，已添加前 ${accepted.length} 个。`,
            })
          }
          return [...current, ...accepted]
        })
        setShowSuccess(true)
      }

      if (rejected.length > 0) {
        setNotification({
          type: 'error',
          text:
            rejected.length === 1 ? (rejected[0] ?? '') : `${rejected.length} 个文件类型不受支持`,
        })
      }
    },
    [validateFile],
  )

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault()
      setDragOver(false)
      const files = e.dataTransfer.files
      if (files?.length) handleFilesSelect(files)
    },
    [handleFilesSelect],
  )

  const handleUpload = useCallback(() => {
    if (submittableUploadItems.length === 0) return
    const tags = uploadTags
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)
    const submitted = submittableUploadItems

    uploadMutation.mutate(
      { knowledgeBaseId, files: submitted.map((item) => item.file), tags },
      {
        onSuccess: (summary) => {
          setUploadItems((current) => {
            const submittedByIndex = new Map(submitted.map((item, index) => [index, item.id]))
            const resultByItemId = new Map<string, (typeof summary.results)[number]>()
            summary.results.forEach((result, index) => {
              const itemId = submittedByIndex.get(index)
              if (itemId) resultByItemId.set(itemId, result)
            })
            return current
              .map((item) => {
                const result = resultByItemId.get(item.id)
                if (!result) return item
                if (result.status === 'uploaded') {
                  const uploadedItem: UploadItem = {
                    ...item,
                    status: 'uploaded',
                    message: undefined,
                    documentId: result.document?.id,
                  }
                  return uploadedItem
                }
                const failedItem: UploadItem = {
                  ...item,
                  status: 'failed',
                  message: result.error?.message ?? '上传失败',
                }
                return failedItem
              })
              .filter((item) => item.status !== 'uploaded')
          })
          setPage(1)
          if (summary.failedCount === 0) {
            setNotification({ type: 'success', text: '文档上传成功' })
            setUploadOpen(false)
            setUploadItems([])
            setUploadTags('')
          } else if (summary.successCount > 0) {
            setNotification({ type: 'error', text: '部分文档上传失败，请检查失败项后重试' })
          } else {
            setNotification({ type: 'error', text: '文档上传失败，请检查失败项后重试' })
          }
        },
        onError: (err: Error) => {
          setNotification({
            type: 'error',
            text: formatGatewayCapabilityError(err, '文档上传'),
          })
        },
      },
    )
  }, [submittableUploadItems, uploadTags, knowledgeBaseId, uploadMutation])

  const openEditTags = useCallback((doc: DocumentSummary) => {
    setEditingDoc(doc)
    setEditTagsText((doc.tags ?? []).join(', '))
    setEditOpen(true)
  }, [])

  const handleEditTags = useCallback(() => {
    if (!editingDoc) return
    const tags = editTagsText
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)

    updateMutation.mutate(
      { id: editingDoc.id, knowledgeBaseId: editingDoc.knowledgeBaseId, tags },
      {
        onSuccess: () => {
          setNotification({ type: 'success', text: '标签更新成功' })
          setEditOpen(false)
          setEditingDoc(null)
        },
        onError: (err: Error) => {
          setNotification({
            type: 'error',
            text: formatGatewayCapabilityError(err, '文档标签更新'),
          })
        },
      },
    )
  }, [editingDoc, editTagsText, updateMutation])

  const openDelete = useCallback((doc: DocumentSummary) => {
    setDeletingDoc(doc)
    setDeleteOpen(true)
  }, [])

  const handleDelete = useCallback(() => {
    if (!deletingDoc) return
    deleteMutation.mutate(
      { id: deletingDoc.id, knowledgeBaseId: deletingDoc.knowledgeBaseId },
      {
        onSuccess: () => {
          setNotification({ type: 'success', text: '文档已删除' })
          setDeleteOpen(false)
          setDeletingDoc(null)
        },
        onError: (err: Error) => {
          setNotification({
            type: 'error',
            text: formatGatewayCapabilityError(err, '文档删除'),
          })
        },
      },
    )
  }, [deletingDoc, deleteMutation])

  const handleDownload = useCallback((doc: DocumentSummary) => {
    getDocumentContent(doc.id, doc.knowledgeBaseId)
      .then((blob) => {
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = doc.name
        a.click()
        URL.revokeObjectURL(url)
      })
      .catch((err: unknown) => {
        setNotification({
          type: 'error',
          text: formatGatewayCapabilityError(err, '文档原文下载'),
        })
      })
  }, [])

  // ── Polling for processing documents ──

  const hasProcessingDocs = filteredItems?.some((d) => isProcessing(d.status))

  useEffect(() => {
    if (!hasProcessingDocs) return
    const interval = setInterval(() => {
      refetch()
    }, 3000)
    return () => clearInterval(interval)
  }, [hasProcessingDocs, refetch])

  // ── Render ──

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h3 className="text-2xl font-semibold text-foreground">文档管理</h3>
          {knowledgeBaseId ? (
            <p className="mt-1 text-sm text-muted-foreground">
              知识库「{kbName}」的文档列表，支持上传、标签编辑与删除。
            </p>
          ) : (
            <p className="mt-1 text-sm text-muted-foreground">
              请选择一个知识库以查看和管理其文档。
            </p>
          )}
        </div>
        {knowledgeBaseId && canUpload && (
          <Button onClick={() => setUploadOpen(true)}>
            <Upload aria-hidden="true" className="mr-1 size-4" />
            上传文档
          </Button>
        )}
      </div>

      {/* KB selector — shown when no KB is pre-selected */}
      {!knowledgeBaseId && knowledgeBaseListIssue && (
        <StateBlock
          action={
            <Button variant="outline" size="sm" onClick={() => void refetchKbList()}>
              <Loader2 aria-hidden="true" className="mr-1.5 size-3.5" />
              重试
            </Button>
          }
          className="mb-6"
          description={knowledgeBaseListIssue.description}
          size="compact"
          title={knowledgeBaseListIssue.title}
          variant={
            knowledgeBaseListIssue.kind === 'forbidden'
              ? 'forbidden'
              : knowledgeBaseListIssue.variant
          }
        />
      )}

      {!knowledgeBaseId && !knowledgeBaseListIssue && (
        <StateBlock
          action={
            !isKbListLoading &&
            hasKnowledgeBases && (
              <Select
                value={activeKbId || undefined}
                onValueChange={(value) => setActiveKbId(String(value))}
              >
                <SelectTrigger className="h-9 w-auto min-w-[180px]">
                  <SelectValue placeholder="选择知识库…" />
                </SelectTrigger>
                <SelectContent>
                  {availableKnowledgeBases.map((kb) => (
                    <SelectItem key={kb.id} value={kb.id}>
                      {kb.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )
          }
          className="mb-6"
          description={isKbListLoading ? '正在从 Gateway 获取知识库列表。' : undefined}
          icon={isKbListLoading ? undefined : FileText}
          size="compact"
          title={
            isKbListLoading
              ? '正在加载知识库'
              : hasKnowledgeBases
                ? '选择一个知识库以查看和管理其文档'
                : '请先创建知识库'
          }
          variant={isKbListLoading ? 'loading' : 'empty'}
        />
      )}

      {/* Toast notification */}
      {notification && (
        <InlineNotice className="toast-enter mb-4" variant={notification.type}>
          {notification.text}
        </InlineNotice>
      )}

      {/* Loading state */}
      {knowledgeBaseId && isLoading && <TableSkeleton columns={7} />}

      {/* Error state */}
      {knowledgeBaseId && isError && !isLoading && (
        <StateBlock
          action={
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <Loader2 aria-hidden="true" className="mr-1.5 size-3.5" />
              重试
            </Button>
          }
          description={documentListIssue?.description ?? '未知错误'}
          size="compact"
          title={documentListIssue?.title ?? '加载文档列表失败'}
          variant={
            documentListIssue?.kind === 'forbidden'
              ? 'forbidden'
              : (documentListIssue?.variant ?? 'error')
          }
        />
      )}

      {/* Data area */}
      {knowledgeBaseId && !isLoading && !isError && (
        <>
          {/* Search & filter bar */}
          <div className="mb-4 flex gap-2">
            <div className="search-expand relative flex-1">
              <Search
                aria-hidden="true"
                className="absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground"
              />
              <Input
                type="text"
                maxLength={200}
                placeholder="搜索文档名称或标签..."
                value={keyword}
                onChange={(e) => {
                  setKeyword(e.target.value)
                  setPage(1)
                }}
                className="pl-8"
              />
            </div>
            <Select
              value={statusFilter || undefined}
              onValueChange={(v) => {
                setStatusFilter(String(v) as DocumentStatus | '')
                setPage(1)
              }}
            >
              <SelectTrigger className="h-8 w-auto min-w-[110px]">
                <SelectValue placeholder="全部状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">全部状态</SelectItem>
                {(FILTERABLE_STATUSES.filter(Boolean) as DocumentStatus[]).map((s) => (
                  <SelectItem key={s} value={s}>
                    {STATUS_LABELS[s]}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Empty state */}
          {isEmpty && (
            <StateBlock
              action={
                !keyword &&
                !statusFilter &&
                canUpload && (
                  <Button variant="outline" size="sm" onClick={() => setUploadOpen(true)}>
                    <Upload aria-hidden="true" className="mr-1 size-3.5" />
                    上传文档
                  </Button>
                )
              }
              icon={FileText}
              title={
                keyword || statusFilter
                  ? '未找到匹配的文档，请调整筛选条件'
                  : '暂无文档，点击上传文档开始'
              }
              variant="empty"
            />
          )}

          {/* Table */}
          {filteredItems && filteredItems.length > 0 && (
            <>
              <div className="overflow-x-auto rounded-lg border border-border bg-card">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border bg-muted/30">
                      <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">
                        文件名
                      </th>
                      <th className="hidden px-4 py-2.5 text-left font-medium text-muted-foreground sm:table-cell">
                        类型
                      </th>
                      <th className="hidden px-4 py-2.5 text-right font-medium text-muted-foreground md:table-cell">
                        大小
                      </th>
                      <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">
                        状态
                      </th>
                      <th className="hidden px-4 py-2.5 text-left font-medium text-muted-foreground lg:table-cell">
                        标签
                      </th>
                      <th className="hidden px-4 py-2.5 text-left font-medium text-muted-foreground xl:table-cell">
                        上传时间
                      </th>
                      <th className="px-4 py-2.5 text-right font-medium text-muted-foreground">
                        操作
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {filteredItems.map((doc) => (
                      <tr key={doc.id} className="transition-colors duration-150 hover:bg-muted/30">
                        <td className="max-w-40 truncate px-4 py-2.5 font-medium text-foreground">
                          <span title={doc.name}>{doc.name}</span>
                          {isProcessing(doc.status) && (
                            <Loader2
                              aria-hidden="true"
                              className="ml-1.5 inline size-3 animate-spin text-muted-foreground"
                            />
                          )}
                        </td>
                        <td className="hidden whitespace-nowrap px-4 py-2.5 text-muted-foreground sm:table-cell">
                          {documentTypeLabel(doc)}
                        </td>
                        <td className="hidden whitespace-nowrap px-4 py-2.5 text-right tabular-nums text-muted-foreground md:table-cell">
                          {formatSize(doc.sizeBytes)}
                        </td>
                        <td className="px-4 py-2.5">
                          <Badge variant={STATUS_VARIANTS[doc.status] ?? 'secondary'}>
                            {STATUS_LABELS[doc.status] ?? doc.status}
                          </Badge>
                          {doc.status === 'failed' && doc.errorMessage && (
                            <p
                              className="mt-1 max-w-48 truncate text-xs text-destructive"
                              title={doc.errorMessage}
                            >
                              {doc.errorMessage}
                            </p>
                          )}
                        </td>
                        <td className="hidden px-4 py-2.5 lg:table-cell">
                          <div className="flex flex-wrap gap-0.5">
                            {(doc.tags ?? []).length === 0 ? (
                              <span className="text-muted-foreground">-</span>
                            ) : (
                              (doc.tags ?? []).map((tag) => (
                                <Badge key={tag} variant="secondary" className="text-xs">
                                  {tag}
                                </Badge>
                              ))
                            )}
                          </div>
                        </td>
                        <td className="hidden whitespace-nowrap px-4 py-2.5 text-muted-foreground xl:table-cell">
                          {formatDateTime(doc.createdAt)}
                        </td>
                        <td className="px-4 py-2.5">
                          <div className="flex items-center justify-end gap-1">
                            {/* View chunks */}
                            {onNavigateChunks && (
                              <Button
                                variant="ghost"
                                size="icon-sm"
                                onClick={() => onNavigateChunks(doc.id, doc.knowledgeBaseId)}
                                aria-label={`查看 ${doc.name} 分块`}
                                title="查看分块"
                              >
                                <Eye aria-hidden="true" className="size-3.5" />
                              </Button>
                            )}
                            {/* Edit tags */}
                            {canEditTags && (
                              <Button
                                variant="ghost"
                                size="icon-sm"
                                onClick={() => openEditTags(doc)}
                                aria-label={`编辑 ${doc.name} 标签`}
                                title="编辑标签"
                              >
                                <Edit aria-hidden="true" className="size-3.5" />
                              </Button>
                            )}
                            {/* Download content */}
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              onClick={() => handleDownload(doc)}
                              aria-label={`下载 ${doc.name}`}
                              title="下载原文"
                              disabled={doc.status === 'failed'}
                            >
                              <Download aria-hidden="true" className="size-3.5" />
                            </Button>
                            {/* Delete */}
                            {canDelete && (
                              <Button
                                variant="ghost"
                                size="icon-sm"
                                onClick={() => openDelete(doc)}
                                aria-label={`删除 ${doc.name}`}
                                className="text-destructive hover:text-destructive"
                              >
                                <Trash2 aria-hidden="true" className="size-3.5" />
                              </Button>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Pagination */}
              {showPagination && (
                <div className="mt-4 flex items-center justify-between text-sm text-muted-foreground">
                  <span>
                    共 {data?.page.total ?? 0} 条，第 {page} / {totalPages} 页
                  </span>
                  <div className="flex gap-1">
                    <Button
                      variant="outline"
                      size="icon-sm"
                      className="hover:scale-105 active:scale-95 transition-transform"
                      disabled={page <= 1}
                      onClick={() => setPage((p) => p - 1)}
                      aria-label="上一页"
                    >
                      <ChevronLeft aria-hidden="true" className="size-3.5" />
                    </Button>
                    <Button
                      variant="outline"
                      size="icon-sm"
                      className="hover:scale-105 active:scale-95 transition-transform"
                      disabled={page >= totalPages}
                      onClick={() => setPage((p) => p + 1)}
                      aria-label="下一页"
                    >
                      <ChevronRight aria-hidden="true" className="size-3.5" />
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </>
      )}

      {/* ── Upload Dialog ── */}
      <Dialog open={uploadOpen} onOpenChange={setUploadOpen}>
        <DialogContent className="min-w-0 overflow-hidden sm:max-w-2xl lg:max-w-3xl">
          <DialogHeader className="min-w-0">
            <DialogTitle>上传文档</DialogTitle>
            <DialogDescription className="break-words">
              选择文档文件上传到知识库「{kbName}」。支持 {SUPPORTED_FILE_TYPES_TEXT}。
            </DialogDescription>
          </DialogHeader>

          <div className="w-full min-w-0 max-w-full space-y-4 overflow-hidden">
            {/* Drag-and-drop zone */}
            <div
              className={`relative flex w-full min-w-0 max-w-full flex-col items-center justify-center overflow-hidden rounded-lg border-2 border-dashed p-8 transition-all duration-200 ${
                dragOver
                  ? 'border-primary bg-primary/5 scale-[1.02]'
                  : uploadItems.length > 0
                    ? 'border-emerald-500/50 bg-emerald-50 dark:bg-emerald-950/20'
                    : 'upload-zone border-border hover:border-muted-foreground/30'
              }`}
              onDragOver={(e) => {
                e.preventDefault()
                setDragOver(true)
              }}
              onDragLeave={() => setDragOver(false)}
              onDrop={handleDrop}
              onClick={() => fileInputRef.current?.click()}
            >
              {/* Success check overlay */}
              {showSuccess && (
                <div className="check-pop absolute inset-0 z-10 flex items-center justify-center rounded-lg bg-emerald-500/10">
                  <Check className="size-10 text-emerald-500" strokeWidth={2.5} />
                </div>
              )}
              {uploadItems.length > 0 ? (
                <div className="w-full min-w-0 max-w-full space-y-3 overflow-hidden">
                  <div className="text-center">
                    <FileText aria-hidden="true" className="mx-auto mb-2 size-8 text-emerald-500" />
                    <p className="text-sm font-medium text-foreground">
                      已选择 {uploadItems.length} 个文件
                    </p>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      单次最多 {MAX_BATCH_FILES} 个文件
                    </p>
                  </div>
                  <div className="max-h-44 w-full min-w-0 max-w-full space-y-2 overflow-x-hidden overflow-y-auto rounded-md border border-border bg-background/70 p-2">
                    {uploadItems.map((item) => (
                      <div
                        key={item.id}
                        className="grid min-h-12 w-full min-w-0 max-w-full grid-cols-[1rem_minmax(0,1fr)_auto_auto] items-center gap-2 overflow-hidden rounded-md px-2 py-1.5 text-left"
                      >
                        <FileText
                          aria-hidden="true"
                          className="size-4 shrink-0 text-muted-foreground"
                        />
                        <div className="min-w-0 flex-1 overflow-hidden">
                          <p
                            className="truncate text-sm font-medium text-foreground"
                            title={item.file.name}
                          >
                            {formatUploadFileName(item.file.name)}
                          </p>
                          <p className="truncate text-xs text-muted-foreground">
                            {formatSize(item.file.size)}
                            {item.message ? ` · ${item.message}` : ''}
                          </p>
                        </div>
                        <Badge
                          className="shrink-0"
                          variant={
                            item.status === 'failed'
                              ? 'destructive'
                              : item.status === 'uploaded'
                                ? 'outline'
                                : 'secondary'
                          }
                        >
                          {item.status === 'failed'
                            ? '失败'
                            : item.status === 'uploaded'
                              ? '已上传'
                              : '待上传'}
                        </Badge>
                        {item.status !== 'uploaded' && (
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            className="shrink-0"
                            onClick={(e) => {
                              e.stopPropagation()
                              setUploadItems((current) =>
                                current.filter((candidate) => candidate.id !== item.id),
                              )
                            }}
                            aria-label={`移除 ${item.file.name}`}
                            title="移除"
                          >
                            <X aria-hidden="true" className="size-3.5" />
                          </Button>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              ) : (
                <>
                  <Upload aria-hidden="true" className="mb-2 size-8 text-muted-foreground/50" />
                  <p className="text-sm text-muted-foreground">拖拽文件到此处，或点击选择文件</p>
                  <p className="mt-1 text-xs text-muted-foreground/60">
                    {SUPPORTED_FILE_TYPES_TEXT} (PNG, JPG, GIF, BMP, WebP)
                  </p>
                </>
              )}
              <input
                ref={fileInputRef}
                type="file"
                multiple
                accept={FILE_ACCEPT_TYPES}
                className="hidden"
                onChange={(e) => {
                  const files = e.target.files
                  if (files?.length) handleFilesSelect(files)
                  e.currentTarget.value = ''
                }}
              />
            </div>

            {/* Tags */}
            <div>
              <label
                htmlFor="upload-tags"
                className="mb-1 block text-sm font-medium text-foreground"
              >
                标签（可选，逗号分隔）
              </label>
              <Input
                id="upload-tags"
                type="text"
                maxLength={500}
                placeholder="例如: 规程, 安全, 2024"
                value={uploadTags}
                onChange={(e) => setUploadTags(e.target.value)}
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setUploadOpen(false)
                setUploadItems([])
                setUploadTags('')
              }}
              disabled={isMutating}
            >
              取消
            </Button>
            <Button
              onClick={handleUpload}
              disabled={submittableUploadItems.length === 0 || isMutating}
            >
              {uploadMutation.isPending && (
                <Loader2 aria-hidden="true" className="mr-1.5 size-3.5 animate-spin" />
              )}
              上传
              {submittableUploadItems.length > 1 ? ` ${submittableUploadItems.length} 个文件` : ''}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ── Edit Tags Dialog ── */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>编辑标签</DialogTitle>
            <DialogDescription>
              修改文档 "{editingDoc?.name}" 的标签。多个标签用逗号分隔。
            </DialogDescription>
          </DialogHeader>

          <div>
            <label
              htmlFor="edit-doc-tags"
              className="mb-1 block text-sm font-medium text-foreground"
            >
              标签
            </label>
            <Input
              id="edit-doc-tags"
              type="text"
              maxLength={500}
              placeholder="标签1, 标签2, 标签3"
              value={editTagsText}
              onChange={(e) => setEditTagsText(e.target.value)}
            />
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setEditOpen(false)
                setEditingDoc(null)
              }}
              disabled={isMutating}
            >
              取消
            </Button>
            <Button onClick={handleEditTags} disabled={isMutating}>
              {updateMutation.isPending && (
                <Loader2 aria-hidden="true" className="mr-1.5 size-3.5 animate-spin" />
              )}
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        cancelLabel="取消"
        confirmLabel="确认删除"
        description={
          <>
            确定要删除文档 "{deletingDoc?.name}"
            吗？此操作不可撤销，文档的所有分块和向量数据也将被删除。
          </>
        }
        onConfirm={handleDelete}
        onOpenChange={(open) => {
          setDeleteOpen(open)
          if (!open) setDeletingDoc(null)
        }}
        open={deleteOpen}
        pending={deleteMutation.isPending}
        pendingLabel="删除中..."
        title="确认删除"
        variant="destructive"
      />
    </div>
  )
}
