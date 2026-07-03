import { Edit, FileCode, Loader2, Plus, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'

import { InlineNotice, StateBlock } from '@/components/common'
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
  useCreateParserConfig,
  useDeleteParserConfig,
  useParserConfigs,
  useUpdateParserConfig,
} from '@/features/admin-config'
import { formatGatewayCapabilityError, getGatewayCapabilityIssue } from '@/features/knowledge'
import type {
  CreateParserConfigRequest,
  ParserBackend,
  ParserConfig,
  UpdateParserConfigRequest,
} from '@/lib/types'

// ── Constants ──

const BACKEND_OPTIONS = [
  { value: 'builtin', label: '内置' },
  { value: 'tika', label: 'Apache Tika' },
  { value: 'unstructured', label: 'Unstructured' },
  { value: 'local_ocr', label: '本地 OCR' },
  { value: 'remote_compatible', label: '远程兼容' },
  { value: 'paddleocr_cloud', label: 'PaddleOCR 云端' },
] as const

const BACKEND_LABELS: Record<string, string> = {
  builtin: '内置',
  tika: 'Apache Tika',
  unstructured: 'Unstructured',
  local_ocr: '本地 OCR',
  remote_compatible: '远程兼容',
  paddleocr_cloud: 'PaddleOCR 云端',
}

const PADDLEOCR_BACKEND: ParserBackend = 'paddleocr_cloud'
const PADDLEOCR_DEFAULT_ALGORITHM = 'PaddleOCR-VL'

// ── Types ──

interface FormData {
  name: string
  backend: ParserBackend
  enabled: boolean
  isDefault: boolean
  concurrency: number
  fileTypes: string
  chunkSize: number
  chunkOverlap: number
  separators: string
  endpointUrl: string
  paddleocrBaseUrl: string
  paddleocrAccessToken: string
  paddleocrAlgorithm: string
}

type NotificationState = {
  type: 'success' | 'error'
  text: string
}

// ── Defaults ──

const EMPTY_FORM: FormData = {
  name: '',
  backend: 'builtin',
  enabled: true,
  isDefault: false,
  concurrency: 4,
  fileTypes: '',
  chunkSize: 512,
  chunkOverlap: 64,
  separators: '',
  endpointUrl: '',
  paddleocrBaseUrl: '',
  paddleocrAccessToken: '',
  paddleocrAlgorithm: PADDLEOCR_DEFAULT_ALGORITHM,
}

// ── Helpers ──

type UpdateFormField = <K extends keyof FormData>(field: K, value: FormData[K]) => void

function parserParameterString(
  parameters: ParserConfig['defaultParameters'] | undefined,
  key: string,
  fallback = '',
) {
  const value = parameters?.[key]
  return typeof value === 'string' ? value : fallback
}

function buildDefaultParameters(form: FormData, includePaddleOCRToken: boolean) {
  const defaultParameters: Record<string, unknown> = {
    chunk_size: form.chunkSize,
    chunk_overlap: form.chunkOverlap,
    separators: form.separators.trim()
      ? form.separators
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean)
      : [],
  }

  if (form.backend === PADDLEOCR_BACKEND) {
    defaultParameters.paddleocr_base_url = form.paddleocrBaseUrl.trim()
    defaultParameters.paddleocr_algorithm =
      form.paddleocrAlgorithm.trim() || PADDLEOCR_DEFAULT_ALGORITHM

    const token = form.paddleocrAccessToken.trim()
    if (includePaddleOCRToken && token) {
      defaultParameters.paddleocr_access_token = token
    }
  }

  return defaultParameters
}

function paddleOCRFormReady(form: FormData, mode: 'create' | 'edit', hasConfiguredToken = false) {
  if (form.backend !== PADDLEOCR_BACKEND) return true
  if (!form.paddleocrBaseUrl.trim()) return false
  if (mode === 'create' && !form.paddleocrAccessToken.trim()) return false
  if (mode === 'edit' && !hasConfiguredToken && !form.paddleocrAccessToken.trim()) return false
  return true
}

function formToCreateRequest(form: FormData): CreateParserConfigRequest {
  const params: CreateParserConfigRequest = {
    name: form.name,
    backend: form.backend,
    concurrency: form.concurrency,
    enabled: form.enabled,
    isDefault: form.isDefault,
    defaultParameters: buildDefaultParameters(form, true),
  }

  if (form.backend === 'remote_compatible' && form.endpointUrl.trim()) {
    params.endpointUrl = form.endpointUrl.trim()
  }

  if (form.fileTypes.trim()) {
    params.supportedContentTypes = form.fileTypes
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)
  }
  return params
}

function formToUpdateRequest(form: FormData): UpdateParserConfigRequest {
  const params: UpdateParserConfigRequest = {
    name: form.name,
    backend: form.backend,
    enabled: form.enabled,
    isDefault: form.isDefault,
    concurrency: form.concurrency,
    endpointUrl: form.backend === 'remote_compatible' ? form.endpointUrl.trim() || null : null,
    supportedContentTypes: form.fileTypes.trim()
      ? form.fileTypes
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean)
      : [],
    defaultParameters: buildDefaultParameters(form, form.paddleocrAccessToken.trim() !== ''),
  }
  return params
}

function PaddleOCRCloudFields({
  form,
  mode,
  tokenRequired,
  updateField,
}: {
  form: FormData
  mode: 'create' | 'edit'
  tokenRequired: boolean
  updateField: UpdateFormField
}) {
  return (
    <div className="grid gap-3 rounded-md border border-border/70 p-3 sm:grid-cols-2">
      <div className="sm:col-span-2">
        <label
          htmlFor={`pc-${mode}-paddleocr-base-url`}
          className="mb-1 block text-sm font-medium text-foreground"
        >
          API 地址 <span className="text-destructive">*</span>
        </label>
        <Input
          id={`pc-${mode}-paddleocr-base-url`}
          type="url"
          placeholder="https://paddleocr-api.example.com"
          value={form.paddleocrBaseUrl}
          onChange={(e) => updateField('paddleocrBaseUrl', e.target.value)}
        />
      </div>

      <div>
        <label
          htmlFor={`pc-${mode}-paddleocr-access-token`}
          className="mb-1 block text-sm font-medium text-foreground"
        >
          API Token {tokenRequired && <span className="text-destructive">*</span>}
        </label>
        <Input
          id={`pc-${mode}-paddleocr-access-token`}
          type="password"
          placeholder={mode === 'edit' ? '留空保持不变' : '请输入 API Token'}
          value={form.paddleocrAccessToken}
          onChange={(e) => updateField('paddleocrAccessToken', e.target.value)}
        />
      </div>

      <div>
        <label
          htmlFor={`pc-${mode}-paddleocr-algorithm`}
          className="mb-1 block text-sm font-medium text-foreground"
        >
          模型名称
        </label>
        <Input
          id={`pc-${mode}-paddleocr-algorithm`}
          type="text"
          placeholder={PADDLEOCR_DEFAULT_ALGORITHM}
          value={form.paddleocrAlgorithm}
          onChange={(e) => updateField('paddleocrAlgorithm', e.target.value)}
        />
      </div>
    </div>
  )
}

// ── Skeleton ──

function ParserConfigsSkeleton() {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="h-7 w-32 rounded skeleton-shimmer" />
        <div className="h-8 w-36 rounded skeleton-shimmer" />
      </div>
      <div className="rounded-lg border border-border bg-card">
        <div className="border-b border-border px-4 py-3">
          <div className="grid grid-cols-5 gap-4">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="h-4 rounded skeleton-shimmer" />
            ))}
          </div>
        </div>
        <div className="divide-y divide-border">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="grid grid-cols-5 gap-4 px-4 py-3">
              {Array.from({ length: 5 }).map((_, j) => (
                <div key={j} className="h-4 rounded skeleton-shimmer" />
              ))}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

// ── Main component ──

export function ParserConfigsPage() {
  // ── State ──
  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [editingConfig, setEditingConfig] = useState<ParserConfig | null>(null)
  const [deletingConfig, setDeletingConfig] = useState<ParserConfig | null>(null)

  const [form, setForm] = useState<FormData>(EMPTY_FORM)
  const [notification, setNotification] = useState<NotificationState | null>(null)

  // ── Queries & mutations ──

  const { data, isLoading, isError, error, refetch } = useParserConfigs()

  const createMutation = useCreateParserConfig()
  const updateMutation = useUpdateParserConfig()
  const deleteMutation = useDeleteParserConfig()

  const isMutating =
    createMutation.isPending || updateMutation.isPending || deleteMutation.isPending

  // ── Notification auto-dismiss ──

  useEffect(() => {
    if (!notification) return
    const timer = setTimeout(() => setNotification(null), 4000)
    return () => clearTimeout(timer)
  }, [notification])

  // ── Handlers ──

  const updateField = useCallback(<K extends keyof FormData>(field: K, value: FormData[K]) => {
    setForm((prev) => ({ ...prev, [field]: value }))
  }, [])

  const updateBackend = useCallback((backend: ParserBackend) => {
    setForm((prev) => ({
      ...prev,
      backend,
      endpointUrl: backend === 'remote_compatible' ? prev.endpointUrl : '',
      paddleocrBaseUrl: backend === PADDLEOCR_BACKEND ? prev.paddleocrBaseUrl : '',
      paddleocrAccessToken: '',
      paddleocrAlgorithm:
        backend === PADDLEOCR_BACKEND
          ? prev.paddleocrAlgorithm || PADDLEOCR_DEFAULT_ALGORITHM
          : PADDLEOCR_DEFAULT_ALGORITHM,
    }))
  }, [])

  const openCreate = useCallback(() => {
    setForm(EMPTY_FORM)
    setCreateOpen(true)
  }, [])

  const openEdit = useCallback((config: ParserConfig) => {
    setEditingConfig(config)
    setForm({
      name: config.name,
      backend: config.backend,
      enabled: config.enabled,
      isDefault: config.isDefault,
      concurrency: config.concurrency,
      fileTypes: config.supportedContentTypes?.join(', ') ?? '',
      chunkSize: (config.defaultParameters?.chunk_size as number) ?? 512,
      chunkOverlap: (config.defaultParameters?.chunk_overlap as number) ?? 64,
      separators:
        config.defaultParameters?.separators != null
          ? (config.defaultParameters.separators as string[]).join(', ')
          : '',
      endpointUrl: config.endpointUrl ?? '',
      paddleocrBaseUrl: parserParameterString(config.defaultParameters, 'paddleocr_base_url'),
      paddleocrAccessToken: '',
      paddleocrAlgorithm: parserParameterString(
        config.defaultParameters,
        'paddleocr_algorithm',
        PADDLEOCR_DEFAULT_ALGORITHM,
      ),
    })
    setEditOpen(true)
  }, [])

  const openDelete = useCallback((config: ParserConfig) => {
    setDeletingConfig(config)
    setDeleteOpen(true)
  }, [])

  const handleCreate = useCallback(() => {
    createMutation.mutate(
      formToCreateRequest(form) as Parameters<typeof createMutation.mutate>[0],
      {
        onSuccess: () => {
          setNotification({ type: 'success', text: '解析器配置创建成功' })
          setCreateOpen(false)
        },
        onError: (err: Error) => {
          setNotification({
            type: 'error',
            text: formatGatewayCapabilityError(err, '解析器配置创建'),
          })
        },
      },
    )
  }, [form, createMutation])

  const handleEdit = useCallback(() => {
    if (!editingConfig) return
    updateMutation.mutate(
      {
        id: editingConfig.id,
        ...formToUpdateRequest(form),
      } as Parameters<typeof updateMutation.mutate>[0],
      {
        onSuccess: () => {
          setNotification({ type: 'success', text: '解析器配置更新成功' })
          setEditOpen(false)
          setEditingConfig(null)
        },
        onError: (err: Error) => {
          setNotification({
            type: 'error',
            text: formatGatewayCapabilityError(err, '解析器配置更新'),
          })
        },
      },
    )
  }, [editingConfig, form, updateMutation])

  const handleDelete = useCallback(() => {
    if (!deletingConfig) return
    deleteMutation.mutate(deletingConfig.id, {
      onSuccess: () => {
        setNotification({ type: 'success', text: '解析器配置已删除' })
        setDeleteOpen(false)
        setDeletingConfig(null)
      },
      onError: (err: Error) => {
        setNotification({
          type: 'error',
          text: formatGatewayCapabilityError(err, '解析器配置删除'),
        })
      },
    })
  }, [deletingConfig, deleteMutation])

  // ── Derived ──

  const items = data ?? []
  const isEmpty = !isLoading && !isError && items.length === 0
  const parserConfigIssue = isError ? getGatewayCapabilityIssue(error, '解析器配置') : null
  const createDisabled = !form.name.trim() || !paddleOCRFormReady(form, 'create') || isMutating
  const editHasPaddleOCRToken =
    editingConfig?.backend === PADDLEOCR_BACKEND &&
    editingConfig.paddleocrAccessTokenConfigured === true
  const editTokenRequired = form.backend === PADDLEOCR_BACKEND && !editHasPaddleOCRToken
  const editDisabled =
    !form.name.trim() || !paddleOCRFormReady(form, 'edit', editHasPaddleOCRToken) || isMutating

  // ── Render ──

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h3 className="text-2xl font-semibold text-foreground">解析器配置</h3>
          <p className="mt-1 text-sm text-muted-foreground">
            管理文档解析器配置，支持新建、编辑、删除操作。
          </p>
        </div>
        <Button onClick={openCreate}>
          <Plus aria-hidden="true" className="mr-1 size-4" />
          新建配置
        </Button>
      </div>

      {/* Toast notification */}
      {notification && (
        <InlineNotice className="toast-enter mb-4" variant={notification.type}>
          {notification.text}
        </InlineNotice>
      )}

      {/* Loading state */}
      {isLoading && <ParserConfigsSkeleton />}

      {/* Error state */}
      {isError && !isLoading && (
        <StateBlock
          action={
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <Loader2 aria-hidden="true" className="mr-1.5 size-3.5" />
              重试
            </Button>
          }
          description={parserConfigIssue?.description ?? '未知错误'}
          size="compact"
          title={parserConfigIssue?.title ?? '加载解析器配置失败'}
          variant={
            parserConfigIssue?.kind === 'forbidden'
              ? 'forbidden'
              : (parserConfigIssue?.variant ?? 'error')
          }
        />
      )}

      {/* Empty state */}
      {isEmpty && (
        <div className="rounded-lg border border-dashed border-border p-12 text-center">
          <FileCode aria-hidden="true" className="mx-auto mb-3 size-10 text-muted-foreground/40" />
          <p className="text-sm text-muted-foreground">暂无解析器配置，点击新建配置开始</p>
          <Button variant="outline" size="sm" className="mt-3" onClick={openCreate}>
            <Plus aria-hidden="true" className="mr-1 size-3.5" />
            新建配置
          </Button>
        </div>
      )}

      {/* Table */}
      {!isLoading && !isError && items.length > 0 && (
        <div className="overflow-x-auto rounded-lg border border-border bg-card">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/30">
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">名称</th>
                <th className="hidden px-4 py-2.5 text-left font-medium text-muted-foreground sm:table-cell">
                  后端
                </th>
                <th className="hidden px-4 py-2.5 text-left font-medium text-muted-foreground md:table-cell">
                  文件类型
                </th>
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">
                  分块大小
                </th>
                <th className="hidden px-4 py-2.5 text-left font-medium text-muted-foreground sm:table-cell">
                  分块重叠
                </th>
                <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">状态</th>
                <th className="px-4 py-2.5 text-right font-medium text-muted-foreground">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {items.map((config) => {
                const chunkSize = (config.defaultParameters?.chunk_size as number) ?? '-'
                const chunkOverlap = (config.defaultParameters?.chunk_overlap as number) ?? '-'

                return (
                  <tr key={config.id} className="transition-colors duration-150 hover:bg-muted/30">
                    <td className="max-w-36 truncate px-4 py-2.5 font-medium text-foreground">
                      {config.name}
                    </td>
                    <td className="hidden px-4 py-2.5 sm:table-cell">
                      <Badge variant="secondary">
                        {BACKEND_LABELS[config.backend] ?? config.backend}
                      </Badge>
                    </td>
                    <td className="hidden max-w-40 truncate px-4 py-2.5 text-muted-foreground md:table-cell">
                      {config.supportedContentTypes?.join(', ') ?? '-'}
                    </td>
                    <td className="px-4 py-2.5 tabular-nums text-muted-foreground">
                      {typeof chunkSize === 'number' ? chunkSize : '-'}
                    </td>
                    <td className="hidden px-4 py-2.5 tabular-nums text-muted-foreground sm:table-cell">
                      {typeof chunkOverlap === 'number' ? chunkOverlap : '-'}
                    </td>
                    <td className="px-4 py-2.5">
                      <div className="flex flex-wrap gap-1">
                        {config.enabled ? (
                          <Badge variant="default">启用</Badge>
                        ) : (
                          <Badge variant="secondary">禁用</Badge>
                        )}
                        {config.isDefault && <Badge variant="outline">默认</Badge>}
                      </div>
                    </td>
                    <td className="px-4 py-2.5">
                      <div className="flex items-center justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => openEdit(config)}
                          aria-label={`编辑 ${config.name}`}
                        >
                          <Edit aria-hidden="true" className="size-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => openDelete(config)}
                          aria-label={`删除 ${config.name}`}
                          className="text-destructive hover:text-destructive"
                        >
                          <Trash2 aria-hidden="true" className="size-3.5" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* ── Create Dialog ── */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>新建解析器配置</DialogTitle>
            <DialogDescription>配置文档解析器的基本参数。</DialogDescription>
          </DialogHeader>

          <div className="space-y-3">
            {/* Name */}
            <div>
              <label
                htmlFor="pc-create-name"
                className="mb-1 block text-sm font-medium text-foreground"
              >
                名称 <span className="text-destructive">*</span>
              </label>
              <Input
                id="pc-create-name"
                type="text"
                placeholder="解析器配置名称"
                value={form.name}
                onChange={(e) => updateField('name', e.target.value)}
              />
            </div>

            {/* Backend */}
            <div>
              <label
                htmlFor="pc-create-backend"
                className="mb-1 block text-sm font-medium text-foreground"
              >
                解析后端 <span className="text-destructive">*</span>
              </label>
              <Select
                value={form.backend}
                onValueChange={(v) => updateBackend(String(v) as ParserBackend)}
              >
                <SelectTrigger id="pc-create-backend" className="h-8 w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {BACKEND_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="grid gap-2 sm:grid-cols-2">
              <label className="flex items-center gap-2 text-sm text-foreground">
                <input
                  type="checkbox"
                  checked={form.enabled}
                  onChange={(e) => updateField('enabled', e.target.checked)}
                />
                启用
              </label>
              <label className="flex items-center gap-2 text-sm text-foreground">
                <input
                  type="checkbox"
                  checked={form.isDefault}
                  onChange={(e) => updateField('isDefault', e.target.checked)}
                />
                设为默认配置
              </label>
            </div>

            {/* Conditional: endpointUrl (remote_compatible) */}
            {form.backend === 'remote_compatible' && (
              <div>
                <label
                  htmlFor="pc-create-endpointurl"
                  className="mb-1 block text-sm font-medium text-foreground"
                >
                  远程地址
                </label>
                <Input
                  id="pc-create-endpointurl"
                  type="url"
                  placeholder="https://parser-api.example.com/v1"
                  value={form.endpointUrl}
                  onChange={(e) => updateField('endpointUrl', e.target.value)}
                />
              </div>
            )}

            {form.backend === PADDLEOCR_BACKEND && (
              <PaddleOCRCloudFields
                form={form}
                mode="create"
                tokenRequired
                updateField={updateField}
              />
            )}

            {/* Concurrency */}
            <div>
              <label
                htmlFor="pc-create-concurrency"
                className="mb-1 block text-sm font-medium text-foreground"
              >
                并发数 <span className="text-destructive">*</span>
              </label>
              <Input
                id="pc-create-concurrency"
                type="number"
                placeholder="4"
                min={1}
                max={128}
                value={form.concurrency}
                onChange={(e) =>
                  updateField(
                    'concurrency',
                    Math.min(128, Math.max(1, Number(e.target.value) || 1)),
                  )
                }
              />
            </div>

            {/* File Types */}
            <div>
              <label
                htmlFor="pc-create-filetypes"
                className="mb-1 block text-sm font-medium text-foreground"
              >
                文件类型
              </label>
              <Input
                id="pc-create-filetypes"
                type="text"
                placeholder="application/pdf, text/plain (逗号分隔)"
                value={form.fileTypes}
                onChange={(e) => updateField('fileTypes', e.target.value)}
              />
              <p className="mt-1 text-xs text-muted-foreground">支持的文件 MIME 类型，逗号分隔。</p>
            </div>

            {/* Chunk Size */}
            <div>
              <label
                htmlFor="pc-create-chunksize"
                className="mb-1 block text-sm font-medium text-foreground"
              >
                分块大小
              </label>
              <Input
                id="pc-create-chunksize"
                type="number"
                placeholder="512"
                min={1}
                value={form.chunkSize}
                onChange={(e) => updateField('chunkSize', Math.max(1, Number(e.target.value) || 1))}
              />
            </div>

            {/* Chunk Overlap */}
            <div>
              <label
                htmlFor="pc-create-chunkoverlap"
                className="mb-1 block text-sm font-medium text-foreground"
              >
                分块重叠
              </label>
              <Input
                id="pc-create-chunkoverlap"
                type="number"
                placeholder="64"
                min={0}
                value={form.chunkOverlap}
                onChange={(e) =>
                  updateField('chunkOverlap', Math.max(0, Number(e.target.value) || 0))
                }
              />
            </div>

            {/* Separators */}
            <div>
              <label
                htmlFor="pc-create-separators"
                className="mb-1 block text-sm font-medium text-foreground"
              >
                分隔符（可选）
              </label>
              <Input
                id="pc-create-separators"
                type="text"
                placeholder="\n\n, \n, 。(逗号分隔)"
                value={form.separators}
                onChange={(e) => updateField('separators', e.target.value)}
              />
              <p className="mt-1 text-xs text-muted-foreground">
                文本分段使用的分隔符，逗号分隔。留空使用默认设置。
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)} disabled={isMutating}>
              取消
            </Button>
            <Button onClick={handleCreate} disabled={createDisabled}>
              {createMutation.isPending && (
                <Loader2 aria-hidden="true" className="mr-1.5 size-3.5 animate-spin" />
              )}
              创建
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ── Edit Dialog ── */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>编辑解析器配置</DialogTitle>
            <DialogDescription>修改 "{editingConfig?.name}" 的配置信息。</DialogDescription>
          </DialogHeader>

          <div className="space-y-3">
            {/* Name */}
            <div>
              <label
                htmlFor="pc-edit-name"
                className="mb-1 block text-sm font-medium text-foreground"
              >
                名称 <span className="text-destructive">*</span>
              </label>
              <Input
                id="pc-edit-name"
                type="text"
                placeholder="解析器配置名称"
                value={form.name}
                onChange={(e) => updateField('name', e.target.value)}
              />
            </div>

            {/* Backend */}
            <div>
              <label
                htmlFor="pc-edit-backend"
                className="mb-1 block text-sm font-medium text-foreground"
              >
                解析后端 <span className="text-destructive">*</span>
              </label>
              <Select
                value={form.backend}
                onValueChange={(v) => updateBackend(String(v) as ParserBackend)}
              >
                <SelectTrigger id="pc-edit-backend" className="h-8 w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {BACKEND_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="grid gap-2 sm:grid-cols-2">
              <label className="flex items-center gap-2 text-sm text-foreground">
                <input
                  type="checkbox"
                  checked={form.enabled}
                  onChange={(e) => updateField('enabled', e.target.checked)}
                />
                启用
              </label>
              <label className="flex items-center gap-2 text-sm text-foreground">
                <input
                  type="checkbox"
                  checked={form.isDefault}
                  onChange={(e) => updateField('isDefault', e.target.checked)}
                />
                设为默认配置
              </label>
            </div>

            {/* Conditional: endpointUrl (remote_compatible) */}
            {form.backend === 'remote_compatible' && (
              <div>
                <label
                  htmlFor="pc-edit-endpointurl"
                  className="mb-1 block text-sm font-medium text-foreground"
                >
                  远程地址
                </label>
                <Input
                  id="pc-edit-endpointurl"
                  type="url"
                  placeholder="https://parser-api.example.com/v1"
                  value={form.endpointUrl}
                  onChange={(e) => updateField('endpointUrl', e.target.value)}
                />
              </div>
            )}

            {form.backend === PADDLEOCR_BACKEND && (
              <PaddleOCRCloudFields
                form={form}
                mode="edit"
                tokenRequired={editTokenRequired}
                updateField={updateField}
              />
            )}

            {/* Concurrency */}
            <div>
              <label
                htmlFor="pc-edit-concurrency"
                className="mb-1 block text-sm font-medium text-foreground"
              >
                并发数 <span className="text-destructive">*</span>
              </label>
              <Input
                id="pc-edit-concurrency"
                type="number"
                placeholder="4"
                min={1}
                max={128}
                value={form.concurrency}
                onChange={(e) =>
                  updateField(
                    'concurrency',
                    Math.min(128, Math.max(1, Number(e.target.value) || 1)),
                  )
                }
              />
            </div>

            {/* File Types */}
            <div>
              <label
                htmlFor="pc-edit-filetypes"
                className="mb-1 block text-sm font-medium text-foreground"
              >
                文件类型
              </label>
              <Input
                id="pc-edit-filetypes"
                type="text"
                placeholder="application/pdf, text/plain (逗号分隔)"
                value={form.fileTypes}
                onChange={(e) => updateField('fileTypes', e.target.value)}
              />
              <p className="mt-1 text-xs text-muted-foreground">支持的文件 MIME 类型，逗号分隔。</p>
            </div>

            {/* Chunk Size */}
            <div>
              <label
                htmlFor="pc-edit-chunksize"
                className="mb-1 block text-sm font-medium text-foreground"
              >
                分块大小
              </label>
              <Input
                id="pc-edit-chunksize"
                type="number"
                placeholder="512"
                min={1}
                value={form.chunkSize}
                onChange={(e) => updateField('chunkSize', Math.max(1, Number(e.target.value) || 1))}
              />
            </div>

            {/* Chunk Overlap */}
            <div>
              <label
                htmlFor="pc-edit-chunkoverlap"
                className="mb-1 block text-sm font-medium text-foreground"
              >
                分块重叠
              </label>
              <Input
                id="pc-edit-chunkoverlap"
                type="number"
                placeholder="64"
                min={0}
                value={form.chunkOverlap}
                onChange={(e) =>
                  updateField('chunkOverlap', Math.max(0, Number(e.target.value) || 0))
                }
              />
            </div>

            {/* Separators */}
            <div>
              <label
                htmlFor="pc-edit-separators"
                className="mb-1 block text-sm font-medium text-foreground"
              >
                分隔符（可选）
              </label>
              <Input
                id="pc-edit-separators"
                type="text"
                placeholder="\n\n, \n, 。(逗号分隔)"
                value={form.separators}
                onChange={(e) => updateField('separators', e.target.value)}
              />
              <p className="mt-1 text-xs text-muted-foreground">
                文本分段使用的分隔符，逗号分隔。留空使用默认设置。
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setEditOpen(false)
                setEditingConfig(null)
              }}
              disabled={isMutating}
            >
              取消
            </Button>
            <Button onClick={handleEdit} disabled={editDisabled}>
              {updateMutation.isPending && (
                <Loader2 aria-hidden="true" className="mr-1.5 size-3.5 animate-spin" />
              )}
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ── Delete Confirmation Dialog ── */}
      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>
              确定要删除解析器配置 "{deletingConfig?.name}" 吗？此操作不可撤销。
            </DialogDescription>
          </DialogHeader>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setDeleteOpen(false)
                setDeletingConfig(null)
              }}
              disabled={isMutating}
            >
              取消
            </Button>
            <Button variant="destructive" onClick={handleDelete} disabled={isMutating}>
              {deleteMutation.isPending && (
                <Loader2 aria-hidden="true" className="mr-1.5 size-3.5 animate-spin" />
              )}
              确认删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
