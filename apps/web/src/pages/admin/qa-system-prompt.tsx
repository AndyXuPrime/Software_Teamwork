import { RefreshCw, Rocket } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'

import { ApiError } from '@/api/client'
import { ConfirmDialog, InlineNotice, StateBlock } from '@/components/common'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import {
  useCreateQAConfigVersionMutation,
  useCurrentQAConfigVersionQuery,
} from '@/features/qa-settings/qa-settings.queries'
import {
  buildSystemPromptPayload,
  countUtf8Bytes,
  QA_SYSTEM_PROMPT_MAX_BYTES,
  validateSystemPrompt,
} from '@/features/qa-settings/qa-system-prompt'
import { canAccess } from '@/lib/permissions'
import { useAuthStore } from '@/stores/auth-store'

function getErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return error.requestId ? `${error.message}（requestId: ${error.requestId}）` : error.message
  }

  return error instanceof Error ? error.message : '未知错误'
}

function formatDate(value?: string): string {
  return value ? new Date(value).toLocaleString() : '-'
}

export function QASystemPromptPage() {
  const user = useAuthStore((state) => state.user)
  const canWrite = canAccess(user, { any: ['qa:settings:write'] })
  const qaConfigQuery = useCurrentQAConfigVersionQuery()
  const createMutation = useCreateQAConfigVersionMutation()
  const [draft, setDraft] = useState('')
  const [lastSyncedVersionId, setLastSyncedVersionId] = useState<string | null>(null)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  const current = qaConfigQuery.data
  const isDirty = current ? draft !== current.systemPrompt : false
  const byteCount = useMemo(() => countUtf8Bytes(draft), [draft])
  const validation = useMemo(() => validateSystemPrompt(draft), [draft])
  const loadError = qaConfigQuery.error ? getErrorMessage(qaConfigQuery.error) : null

  useEffect(() => {
    if (!current) return

    if (lastSyncedVersionId === null || (current.id !== lastSyncedVersionId && !isDirty)) {
      setDraft(current.systemPrompt)
      setLastSyncedVersionId(current.id)
    }
  }, [current, isDirty, lastSyncedVersionId])

  useEffect(() => {
    if (!isDirty) return

    const handler = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      event.returnValue = ''
    }

    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [isDirty])

  const refreshCurrent = async () => {
    setError(null)
    setSuccess(null)
    const result = await qaConfigQuery.refetch()
    if (result.data && (!isDirty || result.data.id === lastSyncedVersionId)) {
      setDraft(result.data.systemPrompt)
      setLastSyncedVersionId(result.data.id)
    }
  }

  const requestPublish = () => {
    if (!current || !canWrite) return

    setError(null)
    setSuccess(null)

    const validated = validateSystemPrompt(draft)
    if (!validated.ok) {
      setError(validated.message)
      return
    }

    setConfirmOpen(true)
  }

  const publish = async () => {
    if (!current || !canWrite) return

    try {
      const created = await createMutation.mutateAsync(buildSystemPromptPayload(current, draft))
      setLastSyncedVersionId(created.id)
      setDraft(created.systemPrompt)
      setSuccess(`Agent 提示词版本 ${created.versionNo} 已发布`)
      setConfirmOpen(false)
    } catch (publishError) {
      setError(`发布失败：${getErrorMessage(publishError)}`)
      setConfirmOpen(false)
    }
  }

  return (
    <div className="mx-auto max-w-5xl space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="text-2xl font-semibold text-foreground">Agent 提示词</h3>
          <p className="mt-2 text-sm text-muted-foreground">
            编辑 QA Agent 的全局 systemPrompt，并通过 QA 配置版本发布。
          </p>
        </div>
        <Button
          type="button"
          variant="outline"
          onClick={() => void refreshCurrent()}
          disabled={qaConfigQuery.isFetching}
        >
          <RefreshCw aria-hidden="true" className="size-4" />
          刷新
        </Button>
      </div>

      <InlineNotice variant="warning">
        该提示词适用于所有用户；发布后从下一次提问生效；正在生成的回答不受影响。
      </InlineNotice>

      {!canWrite && (
        <InlineNotice variant="warning">当前账号只有读取权限，页面为只读模式。</InlineNotice>
      )}
      {loadError && current && (
        <InlineNotice variant="error">加载提示词失败：{loadError}</InlineNotice>
      )}
      {error && <InlineNotice variant="error">{error}</InlineNotice>}
      {success && <InlineNotice variant="success">{success}</InlineNotice>}
      {isDirty && (
        <InlineNotice variant="warning">
          存在未保存变更。刷新到新版本时会保留当前草稿，发布成功后才覆盖。
        </InlineNotice>
      )}

      {qaConfigQuery.isLoading ? (
        <StateBlock size="full" title="正在加载 Agent 提示词" variant="loading" />
      ) : loadError && !current ? (
        <StateBlock
          action={
            <Button type="button" variant="outline" onClick={() => void refreshCurrent()}>
              <RefreshCw aria-hidden="true" className="size-4" />
              重试
            </Button>
          }
          description={loadError}
          size="full"
          title={
            qaConfigQuery.error instanceof ApiError && qaConfigQuery.error.isForbidden()
              ? '没有读取 Agent 提示词的权限'
              : '加载 Agent 提示词失败'
          }
          variant={
            qaConfigQuery.error instanceof ApiError && qaConfigQuery.error.isForbidden()
              ? 'forbidden'
              : 'error'
          }
        />
      ) : (
        <section className="space-y-5 rounded-lg border border-border bg-card p-5">
          <div className="grid gap-3 md:grid-cols-4">
            <div className="rounded-lg border border-border bg-background p-3">
              <div className="text-xs text-muted-foreground">当前版本</div>
              <div className="mt-1 text-lg font-semibold text-foreground">
                {current ? `版本 ${current.versionNo}` : '-'}
              </div>
            </div>
            <div className="rounded-lg border border-border bg-background p-3">
              <div className="text-xs text-muted-foreground">创建时间</div>
              <div className="mt-1 text-sm font-medium text-foreground">
                {formatDate(current?.createdAt)}
              </div>
            </div>
            <div className="rounded-lg border border-border bg-background p-3">
              <div className="text-xs text-muted-foreground">创建人</div>
              <div className="mt-1 text-sm font-medium text-foreground">
                {current?.createdBy ?? '-'}
              </div>
            </div>
            <div className="rounded-lg border border-border bg-background p-3">
              <div className="text-xs text-muted-foreground">生效状态</div>
              <div className="mt-1 text-sm font-medium text-foreground">
                {current?.isActive ? '生效中' : '未生效'}
              </div>
            </div>
          </div>

          <label className="block space-y-2 text-sm">
            <span className="font-medium text-foreground">全局 systemPrompt</span>
            <Textarea
              aria-invalid={!validation.ok}
              className="min-h-[360px] resize-y font-mono text-sm leading-6"
              readOnly={!canWrite || !current}
              value={draft}
              onChange={(event) => {
                setDraft(event.target.value)
                setError(null)
                setSuccess(null)
              }}
            />
          </label>

          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="text-sm text-muted-foreground">
              {draft.length} 字符 / {byteCount} UTF-8 bytes（限制 1–
              {QA_SYSTEM_PROMPT_MAX_BYTES} bytes）
              {!validation.ok && (
                <span className="ml-2 text-destructive">{validation.message}</span>
              )}
            </div>
            {canWrite && (
              <Button
                type="button"
                onClick={requestPublish}
                disabled={!current || !isDirty || !validation.ok || createMutation.isPending}
              >
                <Rocket aria-hidden="true" className="size-4" />
                {createMutation.isPending ? '发布中...' : '发布新版本'}
              </Button>
            )}
          </div>
        </section>
      )}

      <ConfirmDialog
        confirmLabel="发布新版本"
        description={
          <span className="space-y-2">
            <span className="block">当前版本：{current ? current.versionNo : '-'}</span>
            <span className="block">
              影响范围：所有用户；发布后从下一次提问生效；正在生成的回答不受影响。
            </span>
          </span>
        }
        disabled={!current || !validation.ok}
        onConfirm={() => void publish()}
        onOpenChange={setConfirmOpen}
        open={confirmOpen}
        pending={createMutation.isPending}
        pendingLabel="发布中..."
        title="确认发布全局 Agent 系统提示词？"
      />
    </div>
  )
}
