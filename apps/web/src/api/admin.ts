/**
 * Admin API — Gateway OpenAPI admin-runtime-config, qa-settings,
 * qa-retrieval-tests, qa-metrics, knowledge, auth paths.
 *
 * All functions use gatewayRequest / gatewayPageRequest from ./client.
 * Types imported from @/lib/types (camelCase, per OpenAPI).
 */

import type {
  CreateKnowledgeBaseRequest,
  CreateQAConfigVersionRequest,
  CreateQALLMConfigVersionRequest,
  CreateUserRequest,
  DocumentChunk,
  DocumentSummary,
  KnowledgeBaseSummary,
  KnowledgeQueryRequest,
  KnowledgeQuerySummary,
  QAConfigVersion,
  QAIntentDistributionItem,
  QALLMConfigVersion,
  QALLMConnectionTest,
  QALLMConnectionTestRequest,
  QAMetricsOverview,
  QAMetricsTrend,
  QARetrievalTestRun,
  QARetrievalTestRunRequest,
  QATopQuery,
  SessionSummary,
  UpdateDocumentRequest,
  UpdateKnowledgeBaseRequest,
} from '@/lib/types'

import { buildQuery, gatewayFileRequest, gatewayPageRequest, gatewayRequest } from './client'

export { createUserSession as createUser, getCurrentUser } from './auth'

// =========================================================================
// LLM Configuration
// =========================================================================

/** GET /llm-config-versions/current */
export function getCurrentLLMConfig(): Promise<QALLMConfigVersion> {
  return gatewayRequest<QALLMConfigVersion>('/llm-config-versions/current')
}

/** POST /llm-config-versions */
export async function createLLMConfigVersion(
  config: CreateQALLMConfigVersionRequest,
): Promise<QALLMConfigVersion> {
  return gatewayRequest<QALLMConfigVersion>('/llm-config-versions', {
    method: 'POST',
    body: config,
  })
}

/** POST /llm-connection-tests */
export async function testLLMConnection(
  params: QALLMConnectionTestRequest,
): Promise<QALLMConnectionTest> {
  return gatewayRequest<QALLMConnectionTest>('/llm-connection-tests', {
    method: 'POST',
    body: params,
  })
}

// =========================================================================
// QA Configuration
// =========================================================================

/** GET /qa-config-versions/current */
export function getCurrentQAConfig(): Promise<QAConfigVersion> {
  return gatewayRequest<QAConfigVersion>('/qa-config-versions/current')
}

/** POST /qa-config-versions */
export async function createQAConfigVersion(
  config: CreateQAConfigVersionRequest,
): Promise<QAConfigVersion> {
  return gatewayRequest<QAConfigVersion>('/qa-config-versions', {
    method: 'POST',
    body: config,
  })
}

// =========================================================================
// Retrieval Test
// =========================================================================

/** POST /retrieval-test-runs */
export async function runRetrievalTest(
  params: QARetrievalTestRunRequest,
): Promise<QARetrievalTestRun> {
  return gatewayRequest<QARetrievalTestRun>('/retrieval-test-runs', {
    method: 'POST',
    body: params,
  })
}

// =========================================================================
// QA Metrics
// =========================================================================

/** GET /qa-metrics/overview?days=N */
export function getQAMetricsOverview(days?: number): Promise<QAMetricsOverview> {
  return gatewayRequest<QAMetricsOverview>(`/qa-metrics/overview${buildQuery({ days })}`)
}

/** GET /qa-metrics/trend?days=N */
export function getQAMetricsTrend(days?: number): Promise<QAMetricsTrend> {
  return gatewayRequest<QAMetricsTrend>(`/qa-metrics/trend${buildQuery({ days: days ?? 30 })}`)
}

/** GET /qa-metrics/top-queries?limit=N&days=N */
export async function getQATopQueries(limit?: number, days?: number): Promise<QATopQuery[]> {
  return gatewayRequest<QATopQuery[]>(`/qa-metrics/top-queries${buildQuery({ limit, days })}`)
}

/** GET /qa-metrics/intent-distribution?days=N */
export async function getQAIntentDistribution(days?: number): Promise<QAIntentDistributionItem[]> {
  return gatewayRequest<QAIntentDistributionItem[]>(
    `/qa-metrics/intent-distribution${buildQuery({ days })}`,
  )
}

// =========================================================================
// Knowledge Bases
// =========================================================================

/** GET /knowledge-bases?page=&pageSize= */
export interface ListKnowledgeBasesParams {
  page?: number
  pageSize?: number
}

export async function listKnowledgeBases(params: ListKnowledgeBasesParams = {}): Promise<{
  items: KnowledgeBaseSummary[]
  page: { page: number; pageSize: number; total: number }
}> {
  return gatewayPageRequest<KnowledgeBaseSummary>(
    `/knowledge-bases${buildQuery({ page: params.page, pageSize: params.pageSize })}`,
  )
}

/** POST /knowledge-bases */
export async function createKnowledgeBase(
  params: CreateKnowledgeBaseRequest,
): Promise<KnowledgeBaseSummary> {
  return gatewayRequest<KnowledgeBaseSummary>('/knowledge-bases', {
    method: 'POST',
    body: params,
  })
}

/** GET /knowledge-bases/{knowledgeBaseId} */
export async function getKnowledgeBase(knowledgeBaseId: string): Promise<KnowledgeBaseSummary> {
  return gatewayRequest<KnowledgeBaseSummary>(
    `/knowledge-bases/${encodeURIComponent(knowledgeBaseId)}`,
  )
}

/** PATCH /knowledge-bases/{knowledgeBaseId} */
export async function updateKnowledgeBase(
  knowledgeBaseId: string,
  params: UpdateKnowledgeBaseRequest,
): Promise<KnowledgeBaseSummary> {
  return gatewayRequest<KnowledgeBaseSummary>(
    `/knowledge-bases/${encodeURIComponent(knowledgeBaseId)}`,
    {
      method: 'PATCH',
      body: params,
    },
  )
}

/** DELETE /knowledge-bases/{knowledgeBaseId} */
export async function deleteKnowledgeBase(id: string): Promise<void> {
  await gatewayRequest<void>(`/knowledge-bases/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

// =========================================================================
// User Management — gateway auth paths
//
// The gateway exposes only these auth resources through /api/v1:
//   POST   /users              — Create user (registration)
//   GET    /users/me           — Get current user
//   POST   /sessions           — Create session (login)
//   DELETE /sessions/current   — Delete current session (logout)
//
// There are no list-all-users, update-user, delete-user, or role CRUD
// endpoints in the current gateway contract.
// =========================================================================

/** POST /users — Create a new user (registration). Returns user + session (envelope unwrapped). */

/** GET /users/me — Get current authenticated user. */
export function getCurrentUser(): Promise<UserSummary> {
  return gatewayRequest<UserSummary>('/users/me')
}

// =========================================================================
// Documents
// =========================================================================

export interface ListDocumentsParams {
  page?: number
  pageSize?: number
  status?: string
}

/** GET /knowledge-bases/{knowledgeBaseId}/documents */
export async function listDocuments(
  knowledgeBaseId: string,
  params: ListDocumentsParams = {},
): Promise<{ items: DocumentSummary[]; page: { page: number; pageSize: number; total: number } }> {
  return gatewayPageRequest<DocumentSummary>(
    `/knowledge-bases/${encodeURIComponent(knowledgeBaseId)}/documents${buildQuery({
      page: params.page,
      pageSize: params.pageSize,
      status: params.status,
    })}`,
  )
}

/** POST /knowledge-bases/{knowledgeBaseId}/documents — upload (multipart/form-data) */
export async function uploadDocument(
  knowledgeBaseId: string,
  file: File,
  tags?: string[],
): Promise<DocumentSummary> {
  const formData = new FormData()
  formData.append('file', file)
  if (tags && tags.length > 0) {
    for (const tag of tags) {
      formData.append('tags', tag)
    }
  }
  return gatewayRequest<DocumentSummary>(
    `/knowledge-bases/${encodeURIComponent(knowledgeBaseId)}/documents`,
    {
      method: 'POST',
      body: formData,
    },
  )
}

/** GET /documents/{documentId} */
export async function getDocument(documentId: string): Promise<DocumentSummary> {
  return gatewayRequest<DocumentSummary>(`/documents/${encodeURIComponent(documentId)}`)
}

/** PATCH /documents/{documentId} */
export async function updateDocument(
  documentId: string,
  params: UpdateDocumentRequest,
): Promise<DocumentSummary> {
  return gatewayRequest<DocumentSummary>(`/documents/${encodeURIComponent(documentId)}`, {
    method: 'PATCH',
    body: params,
  })
}

/** DELETE /documents/{documentId} */
export async function deleteDocument(documentId: string): Promise<void> {
  await gatewayRequest<void>(`/documents/${encodeURIComponent(documentId)}`, { method: 'DELETE' })
}

// =========================================================================
// Document Chunks
// =========================================================================

export interface ListChunksParams {
  page?: number
  pageSize?: number
}

/** GET /documents/{documentId}/chunks */
export async function listChunks(
  documentId: string,
  params: ListChunksParams = {},
): Promise<{ items: DocumentChunk[]; page: { page: number; pageSize: number; total: number } }> {
  return gatewayPageRequest<DocumentChunk>(
    `/documents/${encodeURIComponent(documentId)}/chunks${buildQuery({
      page: params.page,
      pageSize: params.pageSize,
    })}`,
  )
}

/** GET /documents/{documentId}/content — stream original file as Blob */
export async function getDocumentContent(documentId: string): Promise<Blob> {
  return gatewayFileRequest(`/documents/${encodeURIComponent(documentId)}/content`)
}

// =========================================================================
// Knowledge Queries (Search)
// =========================================================================

/** POST /knowledge-queries */
export async function runKnowledgeQuery(
  params: KnowledgeQueryRequest,
): Promise<KnowledgeQuerySummary> {
  return gatewayRequest<KnowledgeQuerySummary>('/knowledge-queries', {
    method: 'POST',
    body: params,
  })
}
