import {
  Check,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Loader2,
  Plus,
  RefreshCw,
  Search,
  X,
} from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'

import type { KnowledgeBaseSummary } from '@/api/knowledge'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectItemText,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'

import { getGatewayCapabilityIssue } from '../capability'
import { useKnowledgeBases } from '../hooks/use-knowledge-bases'

type KnowledgeBaseMultiSelectProps = {
  className?: string
  description?: string
  disabled?: boolean
  label?: string
  onChange: (ids: string[]) => void
  pageSize?: number
  value: string[]
  variant?: 'panel' | 'compact'
}

const emptyKnowledgeBases: KnowledgeBaseSummary[] = []

function toggleId(ids: string[], id: string): string[] {
  return ids.includes(id) ? ids.filter((item) => item !== id) : [...ids, id]
}

export function KnowledgeBaseMultiSelect({
  className,
  description,
  disabled = false,
  label = '知识库范围',
  onChange,
  pageSize = 100,
  value,
  variant = 'panel',
}: KnowledgeBaseMultiSelectProps) {
  const [keyword, setKeyword] = useState('')
  const [manualIdsText, setManualIdsText] = useState('')
  const [open, setOpen] = useState(false)
  const compactSelectorRef = useRef<HTMLDivElement>(null)
  const [showManualInput, setShowManualInput] = useState(false)
  const [selectedKnowledgeBaseId, setSelectedKnowledgeBaseId] = useState('')
  const [page, setPage] = useState(1)
  const query = useKnowledgeBases(page, pageSize)
  const items = query.data?.items ?? emptyKnowledgeBases
  const selectedItems = items.filter((item) => value.includes(item.id))
  const selectedUnknownIds = value.filter((id) => !items.some((item) => item.id === id))
  const normalizedKeyword = keyword.trim().toLowerCase()
  const filteredItems = useMemo(() => {
    if (!normalizedKeyword) return items
    return items.filter((item) =>
      `${item.name} ${item.description ?? ''} ${item.id}`.toLowerCase().includes(normalizedKeyword),
    )
  }, [items, normalizedKeyword])
  const addableItems = useMemo(
    () => filteredItems.filter((item) => !value.includes(item.id)),
    [filteredItems, value],
  )
  const selectedAddableItem = useMemo(
    () => addableItems.find((item) => item.id === selectedKnowledgeBaseId),
    [addableItems, selectedKnowledgeBaseId],
  )
  const issue = query.isError ? getGatewayCapabilityIssue(query.error, '知识库列表') : null
  const pageInfo = query.data?.page
  const currentPage = pageInfo?.page ?? page
  const totalItems = pageInfo?.total ?? items.length
  const effectivePageSize = pageInfo?.pageSize ?? pageSize
  const totalPages = Math.max(1, Math.ceil(totalItems / effectivePageSize))
  const canGoPrevious = currentPage > 1
  const canGoNext = currentPage < totalPages

  useEffect(() => {
    if (
      selectedKnowledgeBaseId &&
      !addableItems.some((item) => item.id === selectedKnowledgeBaseId)
    ) {
      setSelectedKnowledgeBaseId('')
    }
  }, [addableItems, selectedKnowledgeBaseId])

  useEffect(() => {
    if (!open || variant !== 'compact') return

    const handlePointerDown = (event: PointerEvent) => {
      const target = event.target
      if (!(target instanceof Node)) return
      if (!compactSelectorRef.current?.contains(target)) {
        setOpen(false)
      }
    }

    document.addEventListener('pointerdown', handlePointerDown)
    return () => document.removeEventListener('pointerdown', handlePointerDown)
  }, [open, variant])

  const addSelectedKnowledgeBase = () => {
    if (!selectedAddableItem) return
    onChange([...value, selectedAddableItem.id])
    setSelectedKnowledgeBaseId('')
  }

  const manualIds = useMemo(
    () =>
      manualIdsText
        .split(/[\s,，]+/)
        .map((id) => id.trim())
        .filter(Boolean),
    [manualIdsText],
  )
  const addableManualIds = useMemo(
    () => manualIds.filter((id, index) => manualIds.indexOf(id) === index && !value.includes(id)),
    [manualIds, value],
  )

  const addManualKnowledgeBaseIds = () => {
    if (addableManualIds.length === 0) return
    onChange([...value, ...addableManualIds])
    setManualIdsText('')
    setShowManualInput(false)
  }
  const shouldShowManualInput = showManualInput || query.isError
  const compactTriggerText = value.length === 0 ? '选择知识库' : `已选 ${value.length} 个知识库`
  const compactPanelDisabled = disabled || !open
  const compactCheckboxClass =
    'flex w-full items-start gap-2 rounded-md px-2.5 py-2 text-left text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50'
  const compactCheckMark = (checked: boolean) => (
    <span
      aria-hidden="true"
      className={cn(
        'mt-0.5 flex size-4 shrink-0 items-center justify-center rounded border',
        checked
          ? 'border-primary bg-primary text-primary-foreground'
          : 'border-input bg-background',
      )}
    >
      {checked && <Check className="size-3" />}
    </span>
  )

  if (variant === 'compact') {
    return (
      <div
        ref={compactSelectorRef}
        className={cn('relative flex justify-start text-sm', className)}
      >
        <button
          type="button"
          aria-label={compactTriggerText}
          aria-expanded={open}
          aria-haspopup="dialog"
          disabled={disabled}
          title={description}
          className={cn(
            'inline-flex h-8 max-w-full items-center gap-2 rounded-lg border border-input bg-background px-2.5 py-1 text-sm text-muted-foreground shadow-sm transition-colors outline-none',
            'hover:border-ring/60 hover:bg-muted hover:text-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50',
            'disabled:pointer-events-none disabled:bg-input/50 disabled:opacity-50',
            open && 'border-ring/60 bg-muted text-foreground',
          )}
          onClick={() => setOpen((current) => !current)}
        >
          <Search aria-hidden="true" className="size-3.5 shrink-0" />
          <span className="truncate">{compactTriggerText}</span>
          {query.isLoading ? (
            <Loader2 aria-hidden="true" className="size-3.5 shrink-0 animate-spin" />
          ) : (
            <ChevronDown
              aria-hidden="true"
              className={cn('size-3.5 shrink-0 transition-transform', open && 'rotate-180')}
            />
          )}
        </button>
        <div
          role="dialog"
          aria-hidden={!open}
          aria-label={`${label}选择`}
          data-side="top"
          data-testid="knowledge-base-selector-popover"
          className={cn(
            'absolute bottom-full left-0 z-50 mb-2 flex w-[min(14.75rem,calc(100vw-2rem))] origin-bottom-left flex-col gap-2 rounded-lg bg-popover p-2 text-sm text-popover-foreground shadow-md ring-1 ring-foreground/10 outline-hidden transform-gpu will-change-transform',
            'motion-safe:transition-[opacity,transform] motion-safe:duration-200 motion-safe:ease-out motion-reduce:transition-none',
            open
              ? 'pointer-events-auto translate-y-0 scale-100 opacity-100'
              : 'pointer-events-none translate-y-1 scale-95 opacity-0',
          )}
        >
          {description && <p className="sr-only">{description}</p>}
          <div className="relative">
            <Search
              aria-hidden="true"
              className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
            />
            <Input
              aria-label={`${label}搜索`}
              className="pl-8"
              placeholder="搜索知识库名称或 ID"
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              disabled={compactPanelDisabled}
            />
          </div>

          <button
            type="button"
            role="checkbox"
            aria-checked={value.length === 0}
            aria-label="不选择知识库"
            className={cn(
              compactCheckboxClass,
              value.length === 0
                ? 'bg-primary/10 text-primary'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground',
            )}
            onClick={() => onChange([])}
            disabled={compactPanelDisabled}
          >
            {compactCheckMark(value.length === 0)}
            <span className="min-w-0 flex-1">
              <span className="block font-medium">不选择知识库</span>
              <span className="block truncate text-xs opacity-80">使用默认范围</span>
            </span>
          </button>

          {query.isLoading ? (
            <div className="flex items-center gap-2 rounded-md bg-muted/40 px-3 py-2 text-sm text-muted-foreground">
              <Loader2 aria-hidden="true" className="size-4 animate-spin" />
              正在加载知识库列表...
            </div>
          ) : query.isError ? (
            <div className="space-y-2 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              <div className="font-medium">{issue?.title ?? '加载知识库失败'}</div>
              <p>{issue?.description ?? '请稍后重试。'}</p>
              <div className="grid gap-2">
                <Input
                  aria-label={`${label}手动ID`}
                  placeholder="输入已知 ID，多个用逗号或换行分隔"
                  value={manualIdsText}
                  onChange={(event) => setManualIdsText(event.target.value)}
                  disabled={compactPanelDisabled}
                />
                <div className="flex justify-end gap-2">
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={addManualKnowledgeBaseIds}
                    disabled={compactPanelDisabled || addableManualIds.length === 0}
                  >
                    <Plus className="size-3.5" />
                    添加ID
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => void query.refetch()}
                    disabled={compactPanelDisabled}
                  >
                    <RefreshCw className="size-3.5" />
                    重试
                  </Button>
                </div>
              </div>
            </div>
          ) : items.length === 0 ? (
            <div className="rounded-md bg-muted/40 px-3 py-2 text-sm text-muted-foreground">
              暂无可选择的知识库。
            </div>
          ) : filteredItems.length === 0 ? (
            <div className="rounded-md bg-muted/40 px-3 py-2 text-sm text-muted-foreground">
              未找到匹配的知识库。
            </div>
          ) : (
            <div
              data-testid="knowledge-base-selector-list"
              role="group"
              aria-label={label}
              className="max-h-64 space-y-1 overflow-y-auto rounded-md border border-border bg-background p-1"
            >
              {selectedUnknownIds.map((id) => (
                <button
                  key={id}
                  type="button"
                  role="checkbox"
                  aria-checked="true"
                  aria-label={`${id} (未加载知识库)`}
                  className={cn(compactCheckboxClass, 'bg-primary/10 text-primary')}
                  onClick={() => onChange(value.filter((item) => item !== id))}
                  disabled={compactPanelDisabled}
                >
                  {compactCheckMark(true)}
                  <span className="min-w-0 flex-1">
                    <span className="block truncate font-medium">{id}</span>
                    <span className="block truncate text-xs opacity-80">未加载知识库</span>
                  </span>
                </button>
              ))}
              {filteredItems.map((item) => {
                const checked = value.includes(item.id)
                return (
                  <button
                    key={item.id}
                    type="button"
                    role="checkbox"
                    aria-checked={checked}
                    aria-label={`${item.name} (${item.id})`}
                    className={cn(
                      compactCheckboxClass,
                      checked
                        ? 'bg-primary/10 text-primary'
                        : 'text-muted-foreground hover:bg-muted hover:text-foreground',
                    )}
                    onClick={() => onChange(toggleId(value, item.id))}
                    disabled={compactPanelDisabled}
                  >
                    {compactCheckMark(checked)}
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-medium">{item.name}</span>
                    </span>
                  </button>
                )
              })}
            </div>
          )}

          {totalPages > 1 && !query.isError && (
            <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
              <span>
                第 {currentPage} / {totalPages} 页
              </span>
              <div className="flex gap-1">
                <Button
                  type="button"
                  size="icon-sm"
                  variant="outline"
                  aria-label="上一页知识库"
                  onClick={() => setPage((current) => Math.max(1, current - 1))}
                  disabled={compactPanelDisabled || query.isFetching || !canGoPrevious}
                >
                  <ChevronLeft className="size-3.5" />
                </Button>
                <Button
                  type="button"
                  size="icon-sm"
                  variant="outline"
                  aria-label="下一页知识库"
                  onClick={() => setPage((current) => current + 1)}
                  disabled={compactPanelDisabled || query.isFetching || !canGoNext}
                >
                  <ChevronRight className="size-3.5" />
                </Button>
              </div>
            </div>
          )}
        </div>
      </div>
    )
  }

  return (
    <div
      className={cn(
        'space-y-2 rounded-lg border border-border bg-background p-3 text-sm',
        className,
      )}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <div className="font-medium text-foreground">{label}</div>
          {value.length === 0 ? (
            <Badge variant="outline" title={description}>
              默认范围
            </Badge>
          ) : (
            <span className="text-xs text-muted-foreground">已选 {value.length} 个</span>
          )}
          {description && <span className="sr-only">{description}</span>}
        </div>
        {value.length > 0 && (
          <Button
            type="button"
            size="sm"
            variant="ghost"
            onClick={() => onChange([])}
            disabled={disabled}
          >
            <X className="size-3.5" />
            清空
          </Button>
        )}
      </div>

      {(selectedItems.length > 0 || selectedUnknownIds.length > 0) && (
        <div className="flex min-h-7 flex-wrap gap-1.5 rounded-lg border border-border bg-muted/20 p-1.5">
          {selectedItems.map((item) => (
            <Badge key={item.id} variant="secondary" title={item.id}>
              {item.name}
              <button
                type="button"
                className="ml-0.5 rounded-full outline-none hover:text-destructive focus-visible:ring-2 focus-visible:ring-ring"
                aria-label={`移除知识库 ${item.name}`}
                onClick={() => onChange(value.filter((id) => id !== item.id))}
                disabled={disabled}
              >
                <X aria-hidden="true" className="size-3" />
              </button>
            </Badge>
          ))}
          {selectedUnknownIds.map((id) => (
            <Badge key={id} variant="outline" title={id}>
              {id}
              <button
                type="button"
                className="ml-0.5 rounded-full outline-none hover:text-destructive focus-visible:ring-2 focus-visible:ring-ring"
                aria-label={`移除知识库 ${id}`}
                onClick={() => onChange(value.filter((item) => item !== id))}
                disabled={disabled}
              >
                <X aria-hidden="true" className="size-3" />
              </button>
            </Badge>
          ))}
        </div>
      )}

      <div className="grid gap-2 md:grid-cols-[minmax(180px,1fr)_minmax(150px,0.55fr)_auto]">
        <div className="relative">
          <Search
            aria-hidden="true"
            className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
          />
          <Input
            aria-label={`${label}搜索`}
            className="pl-8"
            placeholder="搜索名称或 ID"
            value={keyword}
            onChange={(event) => setKeyword(event.target.value)}
            disabled={disabled}
          />
        </div>
        <Select
          value={selectedKnowledgeBaseId}
          onValueChange={(id) => setSelectedKnowledgeBaseId(String(id))}
          disabled={disabled || query.isLoading || query.isError || addableItems.length === 0}
        >
          <SelectTrigger aria-label={`${label}选择`}>
            <SelectValue
              placeholder={
                query.isLoading
                  ? '加载知识库...'
                  : addableItems.length === 0
                    ? '暂无可添加知识库'
                    : '选择知识库'
              }
            />
          </SelectTrigger>
          <SelectContent>
            {addableItems.map((item) => (
              <SelectItem key={item.id} value={item.id}>
                <SelectItemText>
                  {item.name} ({item.id})
                </SelectItemText>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          type="button"
          variant="outline"
          onClick={addSelectedKnowledgeBase}
          disabled={disabled || !selectedAddableItem}
        >
          <Plus className="size-3.5" />
          添加
        </Button>
      </div>

      {!shouldShowManualInput && (
        <div className="flex justify-end">
          <Button
            type="button"
            size="sm"
            variant="ghost"
            className="h-7 px-2 text-xs text-muted-foreground"
            onClick={() => setShowManualInput(true)}
            disabled={disabled}
          >
            手动添加已知 ID
          </Button>
        </div>
      )}

      {query.isLoading ? (
        <div className="flex items-center gap-2 rounded-lg bg-muted/40 px-3 py-2 text-sm text-muted-foreground">
          <Loader2 aria-hidden="true" className="size-4 animate-spin" />
          正在加载知识库列表...
        </div>
      ) : query.isError ? (
        <div className="space-y-2 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          <div className="font-medium">{issue?.title ?? '加载知识库失败'}</div>
          <p>{issue?.description ?? '请稍后重试。'}</p>
          <p className="text-xs text-destructive/80">可临时添加已知知识库 ID 继续诊断。</p>
          <div className="grid gap-2 md:grid-cols-[minmax(180px,1fr)_auto_auto]">
            <Input
              aria-label={`${label}手动ID`}
              placeholder="输入已知 ID，多个用逗号或换行分隔"
              value={manualIdsText}
              onChange={(event) => setManualIdsText(event.target.value)}
              disabled={disabled}
            />
            <Button
              type="button"
              variant="outline"
              onClick={addManualKnowledgeBaseIds}
              disabled={disabled || addableManualIds.length === 0}
            >
              <Plus className="size-3.5" />
              添加ID
            </Button>
            <Button type="button" variant="outline" onClick={() => void query.refetch()}>
              <RefreshCw className="size-3.5" />
              重试
            </Button>
          </div>
        </div>
      ) : items.length === 0 ? (
        <div className="rounded-lg bg-muted/40 px-3 py-2 text-sm text-muted-foreground">
          暂无可选择的知识库。
        </div>
      ) : filteredItems.length === 0 ? (
        <div className="rounded-lg bg-muted/40 px-3 py-2 text-sm text-muted-foreground">
          未找到匹配的知识库。
        </div>
      ) : (
        <>
          <div
            role="group"
            aria-label={label}
            className="max-h-40 space-y-1 overflow-y-auto rounded-lg border border-border bg-background p-1"
          >
            {filteredItems.map((item) => {
              const checked = value.includes(item.id)
              return (
                <button
                  key={item.id}
                  type="button"
                  className={cn(
                    'flex w-full items-start gap-2 rounded-md px-2.5 py-1.5 text-left transition-colors',
                    checked
                      ? 'bg-primary/10 text-primary'
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground',
                  )}
                  aria-pressed={checked}
                  onClick={() => onChange(toggleId(value, item.id))}
                  disabled={disabled}
                >
                  <span
                    aria-hidden="true"
                    className={cn(
                      'mt-0.5 flex size-4 shrink-0 items-center justify-center rounded border',
                      checked
                        ? 'border-primary bg-primary text-primary-foreground'
                        : 'border-input',
                    )}
                  >
                    {checked && <Check className="size-3" />}
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="block truncate font-medium">{item.name}</span>
                    <span className="block truncate text-xs opacity-80">{item.id}</span>
                  </span>
                </button>
              )
            })}
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
              <span>
                第 {currentPage} / {totalPages} 页
              </span>
              <div className="flex gap-1">
                <Button
                  type="button"
                  size="icon-sm"
                  variant="outline"
                  aria-label="上一页知识库"
                  onClick={() => setPage((current) => Math.max(1, current - 1))}
                  disabled={disabled || query.isFetching || !canGoPrevious}
                >
                  <ChevronLeft className="size-3.5" />
                </Button>
                <Button
                  type="button"
                  size="icon-sm"
                  variant="outline"
                  aria-label="下一页知识库"
                  onClick={() => setPage((current) => current + 1)}
                  disabled={disabled || query.isFetching || !canGoNext}
                >
                  <ChevronRight className="size-3.5" />
                </Button>
              </div>
            </div>
          )}
        </>
      )}

      {shouldShowManualInput && !query.isError && (
        <div className="grid gap-2 rounded-lg border border-dashed border-border bg-muted/20 p-2 md:grid-cols-[minmax(180px,1fr)_auto_auto]">
          <Input
            aria-label={`${label}手动ID`}
            placeholder="输入已知 ID，多个用逗号或换行分隔"
            value={manualIdsText}
            onChange={(event) => setManualIdsText(event.target.value)}
            disabled={disabled}
          />
          <Button
            type="button"
            variant="outline"
            onClick={addManualKnowledgeBaseIds}
            disabled={disabled || addableManualIds.length === 0}
          >
            <Plus className="size-3.5" />
            添加ID
          </Button>
          <Button
            type="button"
            variant="ghost"
            onClick={() => {
              setManualIdsText('')
              setShowManualInput(false)
            }}
          >
            收起
          </Button>
        </div>
      )}
    </div>
  )
}
