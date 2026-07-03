import { Check, ChevronDown, ChevronRight } from 'lucide-react'
import { type ReactNode, useEffect, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'

import ReportArtifactCard from '@/components/chat/report-artifact-card'
import { InlineNotice } from '@/components/common'
import { Button } from '@/components/ui/button'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import type { QACitation, QAMessage, QAReportArtifact, QAThinkingStep } from '@/lib/types'
import { cn } from '@/lib/utils'

// ══════════════════════════════════════════════════════════════════════════════
// Sub-components
// ══════════════════════════════════════════════════════════════════════════════

/* ── Citation tooltip ── */
function CitationTooltip({ c }: { c: QACitation }) {
  const [open, setOpen] = useState(false)

  // Resolve display fields (id is always present; docId/docName are deprecated aliases)
  const displayId = c.citationNo != null ? `[${c.citationNo}]` : c.id
  const docName = c.documentName ?? c.docName ?? '未知文档'
  const text = c.text ?? c.contentPreview ?? ''
  const score = c.score ?? 0

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        className="inline-flex rounded-sm bg-accent px-2 py-0.5 text-xs text-primary transition-colors hover:bg-primary hover:text-primary-foreground"
        onClick={(e) => {
          e.stopPropagation()
        }}
      >
        {displayId}
      </PopoverTrigger>
      <PopoverContent className="w-72">
        <div className="text-sm font-medium">{docName}</div>
        <div className="mt-1 text-sm italic text-muted-foreground">「{text}」</div>
        <div className="mt-1 text-xs text-muted-foreground">相关度: {Math.round(score * 100)}%</div>
      </PopoverContent>
    </Popover>
  )
}

/* ── Thinking panel ── */
type ThinkPanelStep = QAThinkingStep & {
  argumentsSummary?: unknown
  completedAt?: number
  errorSummary?: string
  iterationNo?: number
  reportArtifact?: QAReportArtifact
  resultSummary?: unknown
  startedAt?: number
  toolCallId?: string
  toolName?: string
}

type IterationGroup = {
  durationMs?: number
  iterationNo: number
  reasoningSteps: ThinkPanelStep[]
  status: QAThinkingStep['status']
  steps: ThinkPanelStep[]
  title?: string
  toolSteps: ThinkPanelStep[]
}

const SUMMARY_LIMIT = 500
const SUMMARY_KEY_LABELS: Record<string, string> = {
  citationCount: '引用数',
  citations: '引用数',
  chunkCount: '片段数',
  hitCount: '命中数',
  hits: '命中数',
  knowledgeBaseCount: '知识库数',
  query: '查询词',
  queryCount: '查询数',
  queryText: '查询词',
  rerankTopN: '重排序 TopN',
  resultCount: '结果数',
  topK: 'TopK',
}
const SENSITIVE_SUMMARY_PATTERN =
  /https?:\/\/|s3:\/\/|gs:\/\/|minio:\/\/|localhost|127\.|10\.|172\.(1[6-9]|2\d|3[01])\.|192\.168\.|api[_-]?key|authorization|bearer\s|credential|object[_-]?key|password|prompt|secret|token/i

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === 'object' && !Array.isArray(value))
}

function safeSummaryValue(value: unknown): string | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) return String(value)
  if (typeof value === 'boolean') return value ? 'true' : 'false'
  if (typeof value !== 'string') return undefined
  const trimmed = value.trim()
  if (!trimmed || SENSITIVE_SUMMARY_PATTERN.test(trimmed)) return undefined
  return trimmed
}

function collectSummaryRows(value: unknown): Array<{ label: string; value: string }> {
  if (!isRecord(value)) return []
  const rows: Array<{ label: string; value: string }> = []
  for (const [key, entry] of Object.entries(value)) {
    const label = SUMMARY_KEY_LABELS[key]
    if (label) {
      const displayValue =
        Array.isArray(entry) && (key === 'citations' || key === 'hits')
          ? String(entry.length)
          : safeSummaryValue(entry)
      if (displayValue) rows.push({ label, value: displayValue })
      continue
    }
    if (isRecord(entry)) rows.push(...collectSummaryRows(entry))
  }
  return rows.slice(0, 8)
}

function SummarySection({ title, value }: { title: string; value: unknown }) {
  const [expanded, setExpanded] = useState(false)
  const rows = collectSummaryRows(value)

  if (rows.length === 0) return null

  return (
    <div className="space-y-1">
      <p className="text-xs font-medium text-muted-foreground">{title}</p>
      <dl className="space-y-1">
        {rows.map((row, index) => {
          const shouldTruncate = row.value.length > SUMMARY_LIMIT
          const display =
            shouldTruncate && !expanded ? row.value.slice(0, SUMMARY_LIMIT) : row.value
          return (
            <div key={`${row.label}-${index}`} className="grid grid-cols-[5rem_1fr] gap-2 text-xs">
              <dt className="text-muted-foreground">{row.label}</dt>
              <dd className="break-words text-foreground/90">
                {display}
                {shouldTruncate && !expanded && '...'}
              </dd>
            </div>
          )
        })}
      </dl>
      {rows.some((row) => row.value.length > SUMMARY_LIMIT) && (
        <button
          className="text-xs font-medium text-primary hover:underline"
          type="button"
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? '收起' : '展开更多'}
        </button>
      )}
    </div>
  )
}

function getStepIteration(step: ThinkPanelStep, fallback: number): number {
  return typeof step.iterationNo === 'number' && Number.isFinite(step.iterationNo)
    ? step.iterationNo
    : fallback
}

function groupThinkingSteps(steps: QAThinkingStep[]): IterationGroup[] {
  const groups = new Map<number, IterationGroup>()
  let currentIteration = 1

  for (const rawStep of steps as ThinkPanelStep[]) {
    if (rawStep.type === 'agent_iteration') {
      currentIteration = getStepIteration(rawStep, currentIteration)
    }
    const iterationNo = getStepIteration(rawStep, currentIteration)
    currentIteration = iterationNo
    const group =
      groups.get(iterationNo) ??
      ({
        iterationNo,
        reasoningSteps: [],
        status: 'done',
        steps: [],
        toolSteps: [],
      } satisfies IterationGroup)

    group.steps.push(rawStep)
    if (rawStep.type === 'agent_iteration') {
      group.title = rawStep.label
      group.status = rawStep.status
    }
    if (rawStep.type === 'tool_call') group.toolSteps.push(rawStep)
    if (rawStep.type !== 'agent_iteration' && rawStep.type !== 'tool_call') {
      group.reasoningSteps.push(rawStep)
    }
    groups.set(iterationNo, group)
  }

  return [...groups.values()].map((group) => {
    const starts = group.toolSteps
      .map((step) => step.startedAt)
      .filter((value): value is number => typeof value === 'number')
    const ends = group.toolSteps
      .map((step) => step.completedAt)
      .filter((value): value is number => typeof value === 'number')
    const durationMs =
      starts.length > 0 && ends.length > 0 ? Math.max(...ends) - Math.min(...starts) : undefined
    return { ...group, durationMs }
  })
}

function statusText(group: IterationGroup): string {
  if (group.steps.some((step) => step.status === 'failed')) return '有失败'
  if (group.steps.some((step) => step.status === 'running') || group.status === 'running') {
    return '执行中'
  }
  return '已完成'
}

function formatDuration(ms: number | undefined): string | undefined {
  if (ms == null || ms < 0) return undefined
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

function ToolCallStep({
  onArtifactDownload,
  step,
}: {
  onArtifactDownload?: (reportFileId: string, filename: string) => void
  step: ThinkPanelStep
}) {
  const [open, setOpen] = useState(false)
  const hasDetails =
    Boolean(step.argumentsSummary) ||
    Boolean(step.resultSummary) ||
    Boolean(step.errorSummary) ||
    Boolean(step.reportArtifact)

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger
        className={cn(
          'flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors hover:bg-background/80',
          step.status === 'failed' && 'bg-red-500/10 text-red-600 hover:bg-red-500/15',
        )}
      >
        {open ? (
          <ChevronDown className="size-3 shrink-0" />
        ) : (
          <ChevronRight className="size-3 shrink-0" />
        )}
        <span
          className={cn(
            'size-1.5 shrink-0 rounded-full',
            step.status === 'done' && 'bg-green-500',
            step.status === 'running' && 'bg-primary animate-pulse',
            step.status === 'pending' && 'bg-muted-foreground/40 animate-pulse',
            step.status === 'failed' && 'bg-red-500',
          )}
        />
        <span className="min-w-0 flex-1 truncate">{step.label ?? step.toolName ?? '工具调用'}</span>
        {step.status === 'done' && <Check className="size-3 shrink-0 text-green-500" />}
        {step.status === 'running' && <span className="animate-pulse text-xs text-primary">▊</span>}
        {step.status === 'failed' && <span className="shrink-0 text-xs text-red-600">失败</span>}
      </CollapsibleTrigger>
      {hasDetails && (
        <CollapsibleContent className="ml-5 space-y-3 border-l border-border/60 py-2 pl-3">
          <SummarySection title="参数" value={step.argumentsSummary} />
          <SummarySection title="结果" value={step.resultSummary} />
          {step.errorSummary && (
            <p className="text-xs leading-relaxed text-red-600">失败原因：{step.errorSummary}</p>
          )}
          {step.reportArtifact && (
            <ReportArtifactCard artifact={step.reportArtifact} onDownload={onArtifactDownload} />
          )}
        </CollapsibleContent>
      )}
    </Collapsible>
  )
}

function ReasoningStep({ step }: { step: ThinkPanelStep }) {
  return (
    <div className="flex items-start gap-2 rounded-md px-2 py-1.5 text-sm text-muted-foreground">
      <span
        className={cn(
          'mt-1.5 size-1.5 shrink-0 rounded-full',
          step.status === 'done' && 'bg-green-500',
          step.status === 'running' && 'bg-primary animate-pulse',
          step.status === 'pending' && 'bg-muted-foreground/40 animate-pulse',
          step.status === 'failed' && 'bg-red-500',
        )}
      />
      <span className="min-w-0 flex-1">
        <span className="text-foreground/90">{step.label ?? '思考步骤'}</span>
        {step.detail && <span className="ml-2 text-xs leading-relaxed">{step.detail}</span>}
      </span>
      {step.status === 'running' && <span className="animate-pulse text-xs text-primary">▊</span>}
      {step.status === 'failed' && <span className="shrink-0 text-xs text-red-600">失败</span>}
    </div>
  )
}

function ThinkPanel({
  done,
  onArtifactDownload,
  steps,
}: {
  done: boolean
  onArtifactDownload?: (reportFileId: string, filename: string) => void
  steps: QAThinkingStep[]
}) {
  const [open, setOpen] = useState(!done)
  const groups = groupThinkingSteps(steps)

  useEffect(() => {
    if (done) {
      const t = setTimeout(() => setOpen(false), 3000)
      return () => clearTimeout(t)
    }
    setOpen(true)
  }, [done])

  if (steps.length === 0) return null

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger className="flex w-full items-center gap-1 py-1 text-sm text-muted-foreground transition-colors hover:text-foreground">
        {open ? (
          <ChevronDown className="size-3 shrink-0" />
        ) : (
          <ChevronRight className="size-3 shrink-0" />
        )}
        <span>思考过程 ({steps.length} 步)</span>
        {done && <Check className="size-3 shrink-0 text-green-500" />}
      </CollapsibleTrigger>
      <CollapsibleContent className="mt-1 space-y-3 rounded-md border border-border/50 bg-muted/50 p-3">
        {groups.map((group) => (
          <section key={group.iterationNo} className="space-y-2">
            <div className="flex flex-wrap items-center gap-2 text-sm">
              <span className="font-medium text-foreground">第 {group.iterationNo} 轮</span>
              <span className="text-xs text-muted-foreground">
                {group.toolSteps.length} 个工具调用 · {statusText(group)}
              </span>
              {formatDuration(group.durationMs) && (
                <span className="text-xs text-muted-foreground">
                  耗时 {formatDuration(group.durationMs)}
                </span>
              )}
            </div>
            {group.reasoningSteps.length > 0 && (
              <div className="space-y-1">
                {group.reasoningSteps.map((step, index) => (
                  <ReasoningStep key={`${group.iterationNo}-${step.type}-${index}`} step={step} />
                ))}
              </div>
            )}
            {group.toolSteps.length > 0 && (
              <div className="space-y-1">
                {group.toolSteps.map((step, index) => (
                  <ToolCallStep
                    key={step.toolCallId ?? `${group.iterationNo}-${index}`}
                    onArtifactDownload={onArtifactDownload}
                    step={step}
                  />
                ))}
              </div>
            )}
            {group.reasoningSteps.length === 0 && group.toolSteps.length === 0 && (
              <div className="flex items-center gap-2 px-2 py-1.5 text-sm text-muted-foreground">
                <span
                  className={cn(
                    'size-1.5 shrink-0 rounded-full',
                    group.status === 'running' ? 'bg-primary animate-pulse' : 'bg-green-500',
                  )}
                />
                <span>{group.title ?? '直接生成回答'}</span>
                {group.status === 'running' && (
                  <span className="animate-pulse text-xs text-primary">▊</span>
                )}
              </div>
            )}
          </section>
        ))}
      </CollapsibleContent>
    </Collapsible>
  )
}

/* ── Markdown content ── */
const markdownComponents = {
  h1: ({ children, ...rest }: { children?: ReactNode } & Record<string, unknown>) => (
    <h1 className="mb-4 mt-6 text-xl font-bold text-foreground" {...rest}>
      {children}
    </h1>
  ),
  h2: ({ children, ...rest }: { children?: ReactNode } & Record<string, unknown>) => (
    <h2 className="mb-3 mt-5 text-lg font-semibold text-foreground" {...rest}>
      {children}
    </h2>
  ),
  h3: ({ children, ...rest }: { children?: ReactNode } & Record<string, unknown>) => (
    <h3 className="mb-2 mt-4 text-base font-semibold text-foreground" {...rest}>
      {children}
    </h3>
  ),
  p: ({ children, ...rest }: { children?: ReactNode } & Record<string, unknown>) => (
    <p className="my-2" {...rest}>
      {children}
    </p>
  ),
  ul: ({ children, ...rest }: { children?: ReactNode } & Record<string, unknown>) => (
    <ul className="my-2 list-disc pl-6" {...rest}>
      {children}
    </ul>
  ),
  ol: ({ children, ...rest }: { children?: ReactNode } & Record<string, unknown>) => (
    <ol className="my-2 list-decimal pl-6" {...rest}>
      {children}
    </ol>
  ),
  li: ({ children, ...rest }: { children?: ReactNode } & Record<string, unknown>) => (
    <li className="my-1" {...rest}>
      {children}
    </li>
  ),
  strong: ({ children, ...rest }: { children?: ReactNode } & Record<string, unknown>) => (
    <strong className="font-semibold text-foreground" {...rest}>
      {children}
    </strong>
  ),
  code: ({
    className: cls,
    children,
    ...rest
  }: { className?: string; children?: ReactNode } & Record<string, unknown>) => {
    const isInline = !cls?.includes('language-')
    if (isInline) {
      return (
        <code className="rounded bg-muted px-1.5 py-0.5 text-xs font-mono" {...rest}>
          {children}
        </code>
      )
    }
    return (
      <code className={cls} {...rest}>
        {children}
      </code>
    )
  },
  pre: ({ children, ...rest }: { children?: ReactNode } & Record<string, unknown>) => (
    <pre className="my-2 overflow-x-auto rounded-md bg-zinc-950 p-4 text-sm text-zinc-50" {...rest}>
      {children}
    </pre>
  ),
}

/* ── Status label for assistant messages ── */
function StatusLabel({ status }: { status: QAMessage['status'] }) {
  if (!status || status === 'completed') return null
  if (status === 'streaming') return null
  if (status === 'stopped' || status === 'cancelled') {
    return (
      <span className="ml-2 text-xs text-muted-foreground" aria-label="回复已停止">
        已停止
      </span>
    )
  }
  if (status === 'failed') {
    return (
      <span className="ml-2 text-xs text-destructive" aria-label="发送失败">
        发送失败
      </span>
    )
  }
  return null
}

/* ── Single message bubble ── */
function MessageBubble({
  msg,
  isStreaming,
  onArtifactDownload,
}: {
  msg: QAMessage
  isStreaming: boolean
  onArtifactDownload?: (reportFileId: string, filename: string) => void
}) {
  const isUser = msg.role === 'user'
  const hasThinking = msg.thinking && msg.thinking.length > 0
  const hasCitations = msg.citations && msg.citations.length > 0

  // Report artifacts stored as a dynamic property (not in the QAMessage schema yet)
  const artifacts = (msg as Record<string, unknown>).artifacts as QAReportArtifact[] | undefined
  const hasArtifacts = artifacts && artifacts.length > 0

  // Determine effective streaming state
  const effectiveStreaming = msg.status === 'streaming' || (!msg.status && isStreaming)

  // Determine thinking done state
  const thinkingDone =
    msg.status === 'completed' ||
    msg.status === 'stopped' ||
    msg.status === 'cancelled' ||
    msg.status === 'failed' ||
    (!msg.status && !isStreaming)

  return (
    <div className={cn('flex gap-2', isUser ? 'flex-row-reverse' : '')}>
      {/* Avatar */}
      {isUser ? (
        <div className="flex size-8 shrink-0 items-center justify-center rounded-md bg-primary/20 text-xs font-bold text-primary">
          我
        </div>
      ) : (
        <div className="flex size-8 shrink-0 items-center justify-center rounded-md bg-primary text-xs font-bold text-primary-foreground">
          电
        </div>
      )}

      {/* Bubble */}
      <div
        className={cn(
          'min-w-0 px-4 py-3',
          isUser
            ? 'rounded-lg rounded-br-sm bg-primary text-primary-foreground'
            : 'rounded-lg rounded-bl-sm border border-border bg-muted',
        )}
      >
        {/* Thinking steps (assistant only) */}
        {hasThinking && (
          <div className="mb-2">
            <ThinkPanel
              steps={msg.thinking!}
              done={thinkingDone}
              onArtifactDownload={onArtifactDownload}
            />
          </div>
        )}

        {/* Message content */}
        <div className="leading-relaxed">
          {isUser ? (
            <p className="whitespace-pre-wrap">{msg.content}</p>
          ) : msg.content ? (
            <span>
              {/* @ts-expect-error react-markdown Components type mismatch with React 19 */}
              <ReactMarkdown components={markdownComponents}>{msg.content}</ReactMarkdown>
              <StatusLabel status={msg.status} />
            </span>
          ) : effectiveStreaming ? (
            <span>
              <span className="animate-pulse text-primary" aria-label="助手正在回复中">
                ▊
              </span>
              <StatusLabel status={msg.status} />
            </span>
          ) : msg.status === 'stopped' || msg.status === 'cancelled' || msg.status === 'failed' ? (
            <span>
              <span className="italic text-muted-foreground">（无内容）</span>
              <StatusLabel status={msg.status} />
            </span>
          ) : (
            <span className="italic text-muted-foreground">（无内容）</span>
          )}
        </div>

        {/* Citations (assistant only) */}
        {hasCitations && (
          <div className="mt-4 border-t border-border/50 pt-2">
            <p className="mb-1 text-xs font-semibold text-muted-foreground">引用来源</p>
            <div className="flex flex-wrap gap-1">
              {msg.citations!.map((c) => (
                <CitationTooltip key={c.id} c={c} />
              ))}
            </div>
          </div>
        )}

        {/* Report artifacts (assistant only) */}
        {hasArtifacts &&
          artifacts!.map((artifact) => (
            <ReportArtifactCard
              key={artifact.reportId ?? artifact.jobId ?? artifact.reportName ?? 'artifact'}
              artifact={artifact}
              onDownload={onArtifactDownload}
            />
          ))}
      </div>
    </div>
  )
}

// ══════════════════════════════════════════════════════════════════════════════
// Main component
// ══════════════════════════════════════════════════════════════════════════════

type ChatMessagesProps = {
  messages: QAMessage[]
  streaming: boolean
  error: string | null
  onRetry?: () => void
  onArtifactDownload?: (reportFileId: string, filename: string) => void
}

export default function ChatMessages({
  messages,
  streaming,
  error,
  onRetry,
  onArtifactDownload,
}: ChatMessagesProps) {
  const scrollRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom when messages or streaming updates
  useEffect(() => {
    const element = scrollRef.current
    if (element) element.scrollTop = element.scrollHeight
  }, [messages, streaming])

  return (
    <div ref={scrollRef} className="flex flex-1 flex-col gap-6 overflow-y-auto px-3 py-6">
      {/* ── Message list ── */}
      {messages.map((msg, i) => {
        const isLast = i === messages.length - 1
        const isStreamingAsst = isLast && msg.role === 'assistant' && streaming
        return (
          <div
            key={msg.id}
            className={cn('msg-enter max-w-[85%]', msg.role === 'user' ? 'self-end' : 'self-start')}
            style={{ animationDelay: `${Math.min(i * 50, 600)}ms` }}
          >
            <MessageBubble
              msg={msg}
              isStreaming={isStreamingAsst}
              onArtifactDownload={onArtifactDownload}
            />
          </div>
        )
      })}

      {/* ── Error ── */}
      {error && (
        <InlineNotice
          action={
            onRetry ? (
              <Button variant="destructive" size="sm" onClick={onRetry}>
                重试
              </Button>
            ) : undefined
          }
          className="mx-4"
          variant="error"
        >
          {error}
        </InlineNotice>
      )}
    </div>
  )
}
