import type {
  ChatStreamRequest,
  RAGSearchRequest,
  RAGSearchResult,
  SSECitationData,
  SSEDoneData,
  SSEErrorData,
  SSEIntentStatusData,
  SSEThinkingStepData,
  SSETokenData,
} from '@/lib/types'

import { requestJson, streamGateway } from './client'
import type { components } from './generated/gateway'

export type QASseEventType = components['schemas']['QASseEventType']
type CreateQAMessageRequest = components['schemas']['CreateQAMessageRequest']
type KnowledgeQueryRequest = components['schemas']['KnowledgeQueryRequest']
type KnowledgeQuerySummary = components['schemas']['KnowledgeQuerySummary']

type JsonRecord = Record<string, unknown>

type QAStreamPayload = JsonRecord & {
  eventSeq?: number
  seq?: number
  text?: string
  delta?: string
  content?: string
  messageId?: string
  message_id?: string
  citation?: unknown
  step?: unknown
  status?: string
  label?: string
  code?: string | number
  message?: string
  fatal?: boolean
}

export interface SSEHandlers {
  onIntentStatus?: (data: SSEIntentStatusData) => void
  onThinkingStep?: (data: SSEThinkingStepData) => void
  onToken?: (data: SSETokenData) => void
  onCitation?: (data: SSECitationData) => void
  onDone?: (data: SSEDoneData) => void
  onError?: (data: SSEErrorData) => void
  onAbort?: () => void
}

function toRecord(value: unknown): JsonRecord {
  return value && typeof value === 'object' ? (value as JsonRecord) : {}
}

function parsePayload(data: string): QAStreamPayload {
  try {
    return toRecord(JSON.parse(data)) as QAStreamPayload
  } catch {
    return { text: data }
  }
}

function sequence(payload: QAStreamPayload, fallback: number): number {
  const raw = payload.eventSeq ?? payload.seq
  return typeof raw === 'number' ? raw : fallback
}

function textDelta(payload: QAStreamPayload): string {
  return String(payload.delta ?? payload.text ?? payload.content ?? '')
}

function dispatch(
  event: QASseEventType | string,
  payload: QAStreamPayload,
  handlers: SSEHandlers,
  fallbackSeq: number,
): void {
  const seq = sequence(payload, fallbackSeq)

  switch (event) {
    case 'message.created':
      handlers.onIntentStatus?.({
        status: 'started',
        label: String(payload.label ?? '消息已创建'),
        seq,
      })
      break
    case 'agent.iteration.started':
      handlers.onThinkingStep?.({
        step: {
          type: 'generation',
          label: String(payload.label ?? '智能体处理中'),
          status: 'running',
          detail: typeof payload.message === 'string' ? payload.message : undefined,
        },
        seq,
      })
      break
    case 'reasoning.step': {
      const step = toRecord(payload.step ?? payload)
      handlers.onThinkingStep?.({
        step: {
          type: 'generation',
          label: String(step.label ?? payload.label ?? '处理步骤'),
          status:
            step.status === 'failed' ? 'pending' : step.status === 'done' ? 'done' : 'running',
          detail: typeof step.detail === 'string' ? step.detail : undefined,
        },
        seq,
      })
      break
    }
    case 'tool.started':
    case 'tool.completed':
    case 'tool.failed':
      handlers.onThinkingStep?.({
        step: {
          type: 'retrieval',
          label: String(
            payload.label ?? (event === 'tool.started' ? '工具调用中' : '工具调用完成'),
          ),
          status:
            event === 'tool.started' ? 'running' : event === 'tool.failed' ? 'pending' : 'done',
          detail: typeof payload.message === 'string' ? payload.message : undefined,
        },
        seq,
      })
      break
    case 'answer.delta':
      handlers.onToken?.({ text: textDelta(payload), index: seq, seq })
      break
    case 'citation.delta':
      handlers.onCitation?.({
        citation: toRecord(payload.citation ?? payload) as unknown as SSECitationData['citation'],
        seq,
      })
      break
    case 'answer.completed':
      handlers.onDone?.({
        message_id: String(payload.messageId ?? payload.message_id ?? ''),
        total_tokens: 0,
        prompt_tokens: 0,
        completion_tokens: 0,
        latency_ms: 0,
        seq,
      })
      break
    case 'error':
      handlers.onError?.({
        code: typeof payload.code === 'number' ? payload.code : 0,
        message: typeof payload.message === 'string' ? payload.message : '流式回答失败',
        fatal: payload.fatal !== false,
        seq,
      })
      break
    case 'heartbeat':
      break
    default:
      break
  }
}

function toQAMessageRequest(params: ChatStreamRequest): CreateQAMessageRequest {
  return {
    message: params.message,
    knowledgeBaseIds: params.knowledge_bases,
    retrieval: params.params
      ? {
          topK: params.params.top_k,
          scoreThreshold: params.params.similarity_threshold,
        }
      : undefined,
  }
}

export function streamChat(
  params: ChatStreamRequest,
  handlers: SSEHandlers,
  signal?: AbortSignal,
): { abort: () => void } {
  let fallbackSeq = 0
  const stream = streamGateway(
    `/qa-sessions/${encodeURIComponent(params.conversation_id)}/messages`,
    {
      method: 'POST',
      body: toQAMessageRequest(params),
      signal,
      onEvent(event) {
        fallbackSeq += 1
        dispatch(event.event, parsePayload(event.data), handlers, fallbackSeq)
      },
      onError(error) {
        handlers.onError?.({
          code: error.status,
          message: error.message,
          fatal: true,
          seq: fallbackSeq + 1,
        })
      },
    },
  )

  return {
    abort() {
      stream.abort()
      handlers.onAbort?.()
    },
  }
}

function toKnowledgeQueryRequest(params: RAGSearchRequest): KnowledgeQueryRequest {
  return {
    query: params.query,
    knowledgeBaseIds: params.knowledge_bases,
    topK: params.top_k ?? 10,
    scoreThreshold: params.similarity_threshold ?? 0,
    rerank: params.use_rerank ?? false,
    rerankTopN: params.rerank_top_n,
  }
}

function toRagSearchResult(
  result: components['schemas']['KnowledgeQueryResult'],
  index: number,
): RAGSearchResult {
  return {
    rank: index + 1,
    chunk_id: result.chunkId ?? result.pointId ?? '',
    doc_id: result.documentId ?? '',
    doc_name: result.documentName ?? '',
    text: result.contentPreview,
    vector_score: result.score,
    rerank_score: undefined,
    page_number: undefined,
    chunk_index: result.chunkIndex ?? undefined,
  }
}

export interface RAGSearchResponse {
  query: string
  mode: string
  results: RAGSearchResult[]
  total_hits: number
  took_ms: number
}

export async function ragSearch(params: RAGSearchRequest): Promise<RAGSearchResponse> {
  const data = await requestJson<KnowledgeQuerySummary>('/knowledge-queries', {
    method: 'POST',
    body: toKnowledgeQueryRequest(params),
  })
  return {
    query: data.query,
    mode: 'knowledge_query',
    results: data.results.map(toRagSearchResult),
    total_hits: data.results.length,
    took_ms: 0,
  }
}

export interface RAGSearchCompareRequest {
  query: string
  knowledge_bases?: string[]
  top_k?: number
  threshold?: number
}

export interface RAGSearchCompareResultSet {
  results: RAGSearchResult[]
  took_ms: number
}

export interface RAGSearchComparison {
  overlap_count: number
  vector_only_unique: number
  rerank_unique: number
}

export interface RAGSearchCompareResponse {
  vector_only: RAGSearchCompareResultSet
  vector_rerank: RAGSearchCompareResultSet
  comparison: RAGSearchComparison
}

export async function ragSearchCompare(
  params: RAGSearchCompareRequest,
): Promise<RAGSearchCompareResponse> {
  const [vectorOnly, vectorRerank] = await Promise.all([
    ragSearch({
      query: params.query,
      knowledge_bases: params.knowledge_bases,
      top_k: params.top_k,
      similarity_threshold: params.threshold,
      use_rerank: false,
    }),
    ragSearch({
      query: params.query,
      knowledge_bases: params.knowledge_bases,
      top_k: params.top_k,
      similarity_threshold: params.threshold,
      use_rerank: true,
    }),
  ])

  const vectorIds = new Set(vectorOnly.results.map((result) => result.chunk_id))
  const rerankIds = new Set(vectorRerank.results.map((result) => result.chunk_id))
  const overlap = [...vectorIds].filter((id) => rerankIds.has(id)).length

  return {
    vector_only: { results: vectorOnly.results, took_ms: vectorOnly.took_ms },
    vector_rerank: { results: vectorRerank.results, took_ms: vectorRerank.took_ms },
    comparison: {
      overlap_count: overlap,
      vector_only_unique: vectorIds.size - overlap,
      rerank_unique: rerankIds.size - overlap,
    },
  }
}
