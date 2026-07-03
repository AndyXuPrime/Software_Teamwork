import {
  Check,
  ChevronLeft,
  ChevronRight,
  Edit3,
  MessageSquare,
  Plus,
  Search,
  Trash2,
  X,
} from 'lucide-react'
import { type KeyboardEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { ConfirmDialog, StateBlock } from '@/components/common'
import { Button } from '@/components/ui/button'
import type { QASessionListItem } from '@/lib/types'
import { cn } from '@/lib/utils'

type ChatSidebarProps = {
  sessions: QASessionListItem[]
  activeId: string
  isLoading: boolean
  fetchError: string | null
  onRetryFetch: () => void
  onSelect: (sessionId: string) => void
  onCreate: () => void
  onDelete: (sessionId: string) => void
  onRename: (sessionId: string, title: string) => void
  onPrepareClearAll?: () => Promise<number>
  onClearAll?: () => Promise<void> | void
}

export default function ChatSidebar({
  sessions,
  activeId,
  isLoading,
  fetchError,
  onRetryFetch,
  onSelect,
  onCreate,
  onDelete,
  onRename,
  onPrepareClearAll,
  onClearAll,
}: ChatSidebarProps) {
  const [collapsed, setCollapsed] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editTitle, setEditTitle] = useState('')
  const [deleteTargetId, setDeleteTargetId] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [showClearConfirm, setShowClearConfirm] = useState(false)
  const [clearAllPending, setClearAllPending] = useState(false)
  const [clearAllPreparing, setClearAllPreparing] = useState(false)
  const [clearAllCount, setClearAllCount] = useState<number | null>(null)
  const [clearAllError, setClearAllError] = useState<string | null>(null)
  const editInputRef = useRef<HTMLInputElement>(null)

  const trimmedSearchQuery = searchQuery.trim()
  const isSearching = trimmedSearchQuery.length > 0

  const filteredSessions = useMemo(() => {
    if (!isSearching) return sessions

    const normalizedQuery = trimmedSearchQuery.toLocaleLowerCase()
    return sessions.filter((session) => {
      const title = session.title?.trim()
      if (!title) return false
      return title.toLocaleLowerCase().includes(normalizedQuery)
    })
  }, [isSearching, sessions, trimmedSearchQuery])

  const visibleSessions = collapsed ? sessions : filteredSessions

  const searchEnterTarget = useMemo(() => {
    if (!isSearching) return undefined

    const normalizedQuery = trimmedSearchQuery.toLocaleLowerCase()
    const exactMatches = filteredSessions.filter(
      (session) => session.title?.trim().toLocaleLowerCase() === normalizedQuery,
    )

    if (exactMatches.length === 1) return exactMatches[0]
    if (filteredSessions.length === 1) return filteredSessions[0]
    return undefined
  }, [filteredSessions, isSearching, trimmedSearchQuery])

  // Focus and select the inline edit input when entering edit mode
  useEffect(() => {
    if (editingId) {
      editInputRef.current?.focus()
      editInputRef.current?.select()
    }
  }, [editingId])

  // ── Edit helpers ──

  const startEdit = useCallback((sessionId: string, title: string) => {
    setEditingId(sessionId)
    setEditTitle(title)
  }, [])

  const confirmEdit = useCallback(() => {
    const id = editingId
    const title = editTitle.trim()
    if (id && title) {
      onRename(id, title)
    }
    setEditingId(null)
  }, [editingId, editTitle, onRename])

  const cancelEdit = useCallback(() => {
    setEditingId(null)
  }, [])

  const handleEditKeyDown = useCallback(
    (e: KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter') {
        e.preventDefault()
        confirmEdit()
      } else if (e.key === 'Escape') {
        e.preventDefault()
        cancelEdit()
      }
    },
    [confirmEdit, cancelEdit],
  )

  const handleSearchKeyDown = useCallback(
    (e: KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter') {
        if (searchEnterTarget) {
          e.preventDefault()
          onSelect(searchEnterTarget.id)
        }
      } else if (e.key === 'Escape' && searchQuery) {
        e.preventDefault()
        setSearchQuery('')
      }
    },
    [onSelect, searchEnterTarget, searchQuery],
  )

  const deleteTarget = sessions.find((session) => session.id === deleteTargetId)

  const openClearAllConfirm = useCallback(async () => {
    if (!onClearAll) return
    setShowClearConfirm(true)
    setClearAllCount(null)
    setClearAllError(null)
    setClearAllPreparing(true)
    try {
      const count = onPrepareClearAll ? await onPrepareClearAll() : sessions.length
      setClearAllCount(count)
    } catch {
      setClearAllError('无法确认将要删除的对话总数，请稍后重试。')
    } finally {
      setClearAllPreparing(false)
    }
  }, [onClearAll, onPrepareClearAll, sessions.length])

  const handleConfirmClearAll = useCallback(async () => {
    if (!onClearAll || clearAllCount === null || clearAllCount <= 0) return
    setClearAllPending(true)
    try {
      await onClearAll()
      setShowClearConfirm(false)
    } finally {
      setClearAllPending(false)
    }
  }, [clearAllCount, onClearAll])

  const clearAllDescription = clearAllPreparing
    ? '正在确认将要删除的对话总数...'
    : clearAllError
      ? clearAllError
      : clearAllCount === null
        ? '请先确认将要删除的对话总数。'
        : clearAllCount <= 0
          ? '服务端没有可删除的对话。'
          : `即将删除全部 ${clearAllCount} 个对话。此操作不可撤销。`

  // ── Render ──

  return (
    <aside
      className={cn(
        'flex shrink-0 flex-col bg-card shadow-[1px_0_6px_rgba(0,0,0,0.04),3px_0_12px_rgba(0,0,0,0.03)] transition-[width] duration-300',
        collapsed ? 'w-14' : 'w-72',
      )}
    >
      {/* ── Toggle bar ── */}
      <div className="flex items-center border-b border-border/30">
        {!collapsed && (
          <h2 className="flex-1 truncate px-4 py-2.5 text-sm font-semibold">对话历史</h2>
        )}
        <button
          aria-label={collapsed ? '展开侧栏' : '折叠侧栏'}
          className={cn(
            'flex shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-foreground transition-all',
            collapsed ? 'mx-auto my-2 size-7' : 'mr-1 size-7',
          )}
          onClick={() => setCollapsed(!collapsed)}
        >
          {collapsed ? <ChevronRight className="size-4" /> : <ChevronLeft className="size-4" />}
        </button>
      </div>

      {/* ── New session button (sticky) ── */}
      <div className="p-2">
        {collapsed ? (
          <button
            onClick={onCreate}
            className="group mx-auto flex size-9 items-center justify-center rounded-full bg-primary text-primary-foreground transition-all hover:bg-primary/90 hover:scale-105 active:scale-95"
            title="新建对话"
          >
            <Plus className="size-4 transition-transform duration-300 group-hover:rotate-90" />
          </button>
        ) : (
          <Button
            onClick={onCreate}
            className="group w-full bg-primary text-primary-foreground transition-all duration-200 hover:bg-primary/90 hover:scale-[1.02] hover:shadow-lg active:scale-[0.98]"
          >
            <Plus className="size-4 transition-transform duration-300 group-hover:rotate-90" />
            新建对话
          </Button>
        )}
      </div>

      {!collapsed && (
        <div className="px-2 pb-2">
          <label className="relative block">
            <span className="sr-only">搜索对话标题</span>
            <Search className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <input
              aria-label="搜索对话标题"
              className="h-9 w-full rounded-md border border-input bg-background py-2 pl-8 pr-8 text-sm outline-none transition-colors placeholder:text-muted-foreground focus-visible:ring-1 focus-visible:ring-ring"
              placeholder="搜索对话标题"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyDown={handleSearchKeyDown}
            />
            {searchQuery && (
              <button
                aria-label="清空搜索"
                className="absolute right-1.5 top-1/2 flex size-6 -translate-y-1/2 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                onClick={() => setSearchQuery('')}
                type="button"
              >
                <X className="size-3.5" aria-hidden="true" />
              </button>
            )}
          </label>
        </div>
      )}

      {/* ── Clear all button ── */}
      {onClearAll && sessions.length > 0 && (
        <div className="border-t border-border/30 px-2 py-1.5">
          {collapsed ? (
            <button
              aria-label="清空全部对话"
              type="button"
              onClick={() => void openClearAllConfirm()}
              className="mx-auto flex size-8 items-center justify-center rounded-md text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors"
              title="清空全部对话"
            >
              <Trash2 className="size-4" />
            </button>
          ) : (
            <Button
              variant="ghost"
              size="sm"
              className="w-full justify-start gap-2 text-xs text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
              onClick={() => void openClearAllConfirm()}
            >
              <Trash2 className="size-3.5" />
              清空全部对话
            </Button>
          )}
        </div>
      )}

      {/* ── Session list ── */}
      <div
        className="flex-1 overflow-y-auto pr-1
        [&::-webkit-scrollbar]:w-2
        [&::-webkit-scrollbar-thumb]:rounded-full
        [&::-webkit-scrollbar-thumb]:bg-border
        [&::-webkit-scrollbar-thumb]:hover:bg-muted-foreground/30
        [&::-webkit-scrollbar-track]:bg-transparent"
      >
        <div className="flex flex-col gap-1 p-2">
          {/* Fetch error state — hidden when collapsed */}
          {!collapsed && fetchError && !isLoading && (
            <StateBlock
              action={
                <Button variant="outline" size="sm" onClick={onRetryFetch}>
                  重新加载
                </Button>
              }
              className="mx-2"
              description={fetchError}
              size="compact"
              title="会话列表加载失败"
              variant="error"
            />
          )}

          {/* Loading state — hidden when collapsed */}
          {!collapsed && !fetchError && isLoading && sessions.length === 0 && (
            <StateBlock className="mx-2" size="compact" title="加载会话列表..." variant="loading" />
          )}

          {/* Empty state — hidden when collapsed */}
          {!collapsed && !fetchError && !isLoading && sessions.length === 0 && (
            <StateBlock className="mx-2" size="compact" title="暂无对话记录" variant="empty" />
          )}

          {/* Search empty state — hidden when collapsed */}
          {!collapsed &&
            !fetchError &&
            !isLoading &&
            isSearching &&
            sessions.length > 0 &&
            filteredSessions.length === 0 && (
              <StateBlock className="mx-2" size="compact" title="未找到匹配对话" variant="empty" />
            )}

          {/* Session items */}
          {visibleSessions.map((sess, index) => {
            const isEditing = editingId === sess.id
            const isActive = sess.id === activeId

            return (
              <button
                key={sess.id}
                type="button"
                className={cn(
                  'group relative flex items-center rounded-md transition-all hover:bg-primary/5',
                  collapsed
                    ? 'justify-center px-0 py-2'
                    : 'w-full flex-col items-start gap-0.5 px-3 py-2.5',
                  isActive &&
                    !collapsed &&
                    'bg-primary/10 text-primary border-l-[3px] border-l-primary',
                  isActive && collapsed && 'bg-primary/10',
                )}
                onClick={() => onSelect(sess.id)}
                onDoubleClick={() => !collapsed && startEdit(sess.id, sess.title ?? '')}
                title={collapsed ? (sess.title ?? `对话 ${index + 1}`) : undefined}
              >
                {collapsed ? (
                  /* ── Collapsed: numbered circle ── */
                  <span
                    className={cn(
                      'flex size-7 items-center justify-center rounded-full text-xs font-medium transition-colors',
                      isActive
                        ? 'bg-primary text-primary-foreground shadow-[0_0_0_2px_var(--primary)_/_0.2]'
                        : 'bg-muted text-muted-foreground',
                    )}
                  >
                    {index + 1}
                  </span>
                ) : isEditing ? (
                  /* ── Inline rename ── */
                  <span
                    className="flex w-full items-center gap-1"
                    onClick={(e) => e.stopPropagation()}
                    onKeyDown={(e) => {
                      if (e.key === 'Escape') {
                        e.stopPropagation()
                        cancelEdit()
                      }
                    }}
                  >
                    <input
                      ref={editInputRef}
                      className="min-w-0 flex-1 rounded border border-input bg-background px-1.5 py-0.5 text-sm outline-none focus-visible:ring-1 focus-visible:ring-ring"
                      value={editTitle}
                      onChange={(e) => setEditTitle(e.target.value)}
                      onKeyDown={handleEditKeyDown}
                      onBlur={confirmEdit}
                    />
                    <span
                      className="flex size-4 shrink-0 cursor-pointer items-center justify-center rounded text-muted-foreground hover:text-foreground"
                      role="button"
                      title="确认"
                      onClick={confirmEdit}
                      tabIndex={0}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') confirmEdit()
                      }}
                    >
                      <Check className="size-3" />
                    </span>
                    <span
                      className="flex size-4 shrink-0 cursor-pointer items-center justify-center rounded text-muted-foreground hover:text-foreground"
                      role="button"
                      title="取消"
                      onClick={cancelEdit}
                      tabIndex={0}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') cancelEdit()
                      }}
                    >
                      <X className="size-3" />
                    </span>
                  </span>
                ) : (
                  /* ── Normal display ── */
                  <>
                    <span className="w-full truncate pr-14 text-sm font-medium">
                      {sess.title ?? '新对话'}
                    </span>
                    <span className="text-xs text-muted-foreground">
                      <MessageSquare className="mr-1 inline-block size-3" />
                      {sess.messageCount ?? 0} 条消息
                    </span>

                    {/* Action buttons — visible on row hover */}
                    <span className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
                      {/* Edit (rename) button */}
                      <span
                        className="flex size-5 items-center justify-center rounded text-muted-foreground hover:bg-accent hover:text-foreground"
                        role="button"
                        title="重命名"
                        onClick={(e) => {
                          e.stopPropagation()
                          startEdit(sess.id, sess.title ?? '')
                        }}
                        tabIndex={0}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ' ') {
                            e.stopPropagation()
                            startEdit(sess.id, sess.title ?? '')
                          }
                        }}
                      >
                        <Edit3 className="size-3" aria-hidden="true" />
                      </span>

                      {/* Delete button */}
                      <span
                        className="flex size-5 items-center justify-center rounded text-muted-foreground hover:bg-destructive hover:text-destructive-foreground"
                        role="button"
                        title="删除对话"
                        onClick={(e) => {
                          e.stopPropagation()
                          setDeleteTargetId(sess.id)
                        }}
                        tabIndex={0}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ' ') {
                            e.stopPropagation()
                            setDeleteTargetId(sess.id)
                          }
                        }}
                      >
                        <Trash2 className="size-3" aria-hidden="true" />
                      </span>
                    </span>
                  </>
                )}
              </button>
            )
          })}
        </div>
      </div>
      <ConfirmDialog
        cancelLabel="取消"
        confirmLabel="确认删除"
        description={
          deleteTarget?.title
            ? `即将删除会话"${deleteTarget.title}"。此操作不可撤销。`
            : '此操作不可撤销。'
        }
        onConfirm={() => {
          if (deleteTargetId) onDelete(deleteTargetId)
          setDeleteTargetId(null)
        }}
        onOpenChange={(open) => {
          if (!open) setDeleteTargetId(null)
        }}
        open={Boolean(deleteTargetId)}
        title="确定删除该会话？"
        variant="destructive"
      />
      <ConfirmDialog
        cancelLabel="取消"
        confirmLabel="全部删除"
        description={clearAllDescription}
        disabled={!onClearAll || clearAllPreparing || clearAllCount === null || clearAllCount <= 0}
        onConfirm={() => void handleConfirmClearAll()}
        onOpenChange={(open) => {
          if (clearAllPending) return
          if (!open) {
            setShowClearConfirm(false)
            setClearAllCount(null)
            setClearAllError(null)
          }
        }}
        open={showClearConfirm}
        pending={clearAllPreparing || clearAllPending}
        pendingLabel={clearAllPreparing ? '确认中...' : '删除中...'}
        title="清空全部对话？"
        variant="destructive"
      />
    </aside>
  )
}
