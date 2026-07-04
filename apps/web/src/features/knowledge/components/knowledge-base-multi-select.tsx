import { Check, Loader2, RefreshCw, Search, X } from 'lucide-react'
import { useMemo, useState } from 'react'

import type { KnowledgeBaseSummary } from '@/api/knowledge'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { getGatewayCapabilityIssue, useKnowledgeBases } from '@/features/knowledge'
import { cn } from '@/lib/utils'

type KnowledgeBaseMultiSelectProps = {
  className?: string
  description?: string
  disabled?: boolean
  label?: string
  onChange: (ids: string[]) => void
  pageSize?: number
  value: string[]
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
}: KnowledgeBaseMultiSelectProps) {
  const [keyword, setKeyword] = useState('')
  const query = useKnowledgeBases(1, pageSize)
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
  const issue = query.isError ? getGatewayCapabilityIssue(query.error, '知识库列表') : null

  return (
    <div className={cn('space-y-2 text-sm', className)}>
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <div className="font-medium text-foreground">{label}</div>
          {description && <p className="mt-1 text-xs text-muted-foreground">{description}</p>}
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

      <div className="flex min-h-8 flex-wrap gap-1.5 rounded-lg border border-border bg-muted/20 p-2">
        {selectedItems.length === 0 && selectedUnknownIds.length === 0 ? (
          <span className="text-xs text-muted-foreground">未选择时使用默认知识库范围</span>
        ) : (
          <>
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
          </>
        )}
      </div>

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
          disabled={disabled || query.isLoading || query.isError}
        />
      </div>

      {query.isLoading ? (
        <div className="flex items-center gap-2 rounded-lg border border-border bg-background p-3 text-sm text-muted-foreground">
          <Loader2 aria-hidden="true" className="size-4 animate-spin" />
          正在加载知识库列表...
        </div>
      ) : query.isError ? (
        <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
          <div className="font-medium">{issue?.title ?? '加载知识库失败'}</div>
          <p className="mt-1">{issue?.description ?? '请稍后重试。'}</p>
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="mt-2"
            onClick={() => void query.refetch()}
          >
            <RefreshCw className="size-3.5" />
            重试
          </Button>
        </div>
      ) : items.length === 0 ? (
        <div className="rounded-lg border border-border bg-background p-3 text-sm text-muted-foreground">
          暂无可选择的知识库。
        </div>
      ) : filteredItems.length === 0 ? (
        <div className="rounded-lg border border-border bg-background p-3 text-sm text-muted-foreground">
          未找到匹配的知识库。
        </div>
      ) : (
        <div
          role="group"
          aria-label={label}
          className="max-h-48 space-y-1 overflow-y-auto rounded-lg border border-border bg-background p-1"
        >
          {filteredItems.map((item) => {
            const checked = value.includes(item.id)
            return (
              <button
                key={item.id}
                type="button"
                className={cn(
                  'flex w-full items-start gap-2 rounded-md px-2.5 py-2 text-left transition-colors',
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
                    checked ? 'border-primary bg-primary text-primary-foreground' : 'border-input',
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
      )}
    </div>
  )
}
