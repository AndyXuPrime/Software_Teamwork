/**
 * Chat SSE streaming + RAG search — API doc sections 3 & 4.
 *
 * Based on frontend/src/api/chat.ts SSE implementation.
 */

import type {
  ChatStreamRequest,
  RAGSearchRequest,
  RAGSearchResult,
  SSECitationData,
  SSEDoneData,
  SSEErrorData,
  SSEEventType,
  SSEIntentStatusData,
  SSEThinkingStepData,
  SSETokenData,
} from '@/lib/types'

import { apiClient, doRequest } from './client'

// ---------------------------------------------------------------------------
// SSE handlers (mirrors frontend/src/api/chat.ts SSEHandlers)
// ---------------------------------------------------------------------------

export interface SSEHandlers {
  onIntentStatus?: (data: SSEIntentStatusData) => void
  onThinkingStep?: (data: SSEThinkingStepData) => void
  onToken?: (data: SSETokenData) => void
  onCitation?: (data: SSECitationData) => void
  onDone?: (data: SSEDoneData) => void
  onError?: (data: SSEErrorData) => void
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

function dispatch(
  event: SSEEventType,
  data: unknown,
  handlers: SSEHandlers,
): void {
  switch (event) {
    case 'intent_status':
      handlers.onIntentStatus?.(data as SSEIntentStatusData)
      break
    case 'thinking_step':
      handlers.onThinkingStep?.(data as SSEThinkingStepData)
      break
    case 'token':
      handlers.onToken?.(data as SSETokenData)
      break
    case 'citation':
      handlers.onCitation?.(data as SSECitationData)
      break
    case 'done':
      handlers.onDone?.(data as SSEDoneData)
      break
    case 'error':
      handlers.onError?.(data as SSEErrorData)
      break
    // heartbeat — silently ignored
    default:
      break
  }
}

function anyAbort(...signals: AbortSignal[]): AbortSignal {
  const ctrl = new AbortController()
  for (const s of signals) {
    if (s.aborted) {
      ctrl.abort(s.reason)
      return ctrl.signal
    }
    s.addEventListener('abort', () => ctrl.abort(s.reason), { once: true })
  }
  return ctrl.signal
}

// ---------------------------------------------------------------------------
// 3.1  SSE streaming chat
// ---------------------------------------------------------------------------

/**
 * Initiate a streaming chat request via SSE.
 * Returns an `abort` function for cancellation.
 */
export function streamChat(
  params: ChatStreamRequest,
  handlers: SSEHandlers,
  signal?: AbortSignal,
): { abort: () => void } {
  const controller = new AbortController()
  const combinedSignal = signal
    ? anyAbort(signal, controller.signal)
    : controller.signal

  // Build request body — only include optional params when explicitly set
  const body: Record<string, unknown> = {
    conversation_id: params.conversation_id,
    message: params.message,
  }
  if (params.knowledge_bases?.length) {
    body.knowledge_bases = params.knowledge_bases
  }
  if (params.params) {
    const p: Record<string, unknown> = {}
    if (params.params.top_k != null) p.top_k = params.params.top_k
    if (params.params.similarity_threshold != null) {
      p.similarity_threshold = params.params.similarity_threshold
    }
    if (params.params.use_rerank != null) {
      p.use_rerank = params.params.use_rerank
    }
    if (params.params.rerank_threshold != null) {
      p.rerank_threshold = params.params.rerank_threshold
    }
    if (Object.keys(p).length) body.params = p
  }

  fetch(`${apiClient.baseUrl}/chat/stream`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    signal: combinedSignal,
  })
    .then(async (res) => {
      if (!res.ok) {
        const text = await res.text().catch(() => '')
        handlers.onError?.({
          code: res.status,
          message: text || '请求失败',
          fatal: true,
        })
        return
      }

      const reader = res.body?.getReader()
      if (!reader) {
        handlers.onError?.({
          code: 50000,
          message: '无法读取响应流',
          fatal: true,
        })
        return
      }

      const decoder = new TextDecoder()
      let buffer = ''

      const processLines = (chunk: string[]) => {
        let currentEvent: string | null = null
        for (const line of chunk) {
          if (line.startsWith('event: ')) {
            currentEvent = line.slice(7).trim()
          } else if (line.startsWith('data: ') && currentEvent) {
            try {
              const data: unknown = JSON.parse(line.slice(6))
              dispatch(currentEvent as SSEEventType, data, handlers)
            } catch {
              // ignore unparseable data lines
            }
            currentEvent = null
          }
        }
      }

      for (;;) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        processLines(lines)
      }

      // Flush decoder remainder + any buffered partial line
      buffer += decoder.decode()
      if (buffer.trim()) {
        processLines(buffer.split('\n'))
      }
    })
    .catch((err) => {
      if (err instanceof DOMException && err.name === 'AbortError') return
      handlers.onError?.({
        code: 0,
        message: err instanceof Error ? err.message : '网络异常，请检查连接',
        fatal: true,
      })
    })

  return { abort: () => controller.abort() }
}

// ---------------------------------------------------------------------------
// 4 / 5.1  RAG semantic search (no LLM)
// ---------------------------------------------------------------------------

export interface RAGSearchResponse {
  query: string
  mode: string
  results: RAGSearchResult[]
  total_hits: number
  took_ms: number
}

/**
 * RAG semantic search.
 * API doc 5.1 — debug/search endpoint, no LLM involved.
 */
export async function ragSearch(
  params: RAGSearchRequest,
): Promise<RAGSearchResponse> {
  const res = await fetch(`${apiClient.baseUrl}/rag/search`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(params),
  })
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new Error(text || 'RAG 检索失败')
  }
  const json: { code: number; message: string; data: RAGSearchResponse } =
    await res.json()
  if (json.code !== 0) throw new Error(json.message || 'RAG 检索失败')
  return json.data
}

// ---------------------------------------------------------------------------
// 5.3  RAG search compare — vector-only vs vector+rerank
// ---------------------------------------------------------------------------

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

/**
 * Compare vector-only search against vector+rerank search.
 * API doc 5.3 — /rag/search/compare endpoint.
 */
export async function ragSearchCompare(
  params: RAGSearchCompareRequest,
): Promise<RAGSearchCompareResponse> {
  const body: Record<string, unknown> = {
    query: params.query,
  }
  if (params.knowledge_bases?.length) {
    body.knowledge_bases = params.knowledge_bases
  }
  if (params.top_k != null) body.top_k = params.top_k
  if (params.threshold != null) body.threshold = params.threshold

  return doRequest<RAGSearchCompareResponse>('/rag/search/compare', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}
