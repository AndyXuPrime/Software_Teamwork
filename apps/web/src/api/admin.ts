/**
 * Admin API — Gateway OpenAPI admin-runtime-config, qa-settings,
 * qa-retrieval-tests, qa-metrics, knowledge, auth paths.
 *
 * All functions use gatewayRequest / gatewayPageRequest from ./client.
 * Types imported from @/lib/types (camelCase, per OpenAPI).
 */

import type {
  CreateModelProfileRequest,
  CreateParserConfigRequest,
  CreateQAConfigVersionRequest,
  CreateQALLMConfigVersionRequest,
  ModelProfile,
  ParserConfig,
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
  UpdateModelProfileRequest,
  UpdateParserConfigRequest,
} from '@/lib/types'

import { buildQuery, gatewayRequest, requestVoid } from './client'

export { createUserSession as createUser, getCurrentUser } from './auth'
export {
  createKnowledgeBase,
  deleteDocument,
  deleteKnowledgeBase,
  getDocument,
  getDocumentContent,
  getKnowledgeBase,
  listChunks,
  listDocuments,
  type ListDocumentsParams,
  listKnowledgeBases,
  type ListKnowledgeBasesParams,
  runKnowledgeQuery,
  updateDocument,
  updateKnowledgeBase,
  uploadDocument,
} from './knowledge'

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

// =========================================================================
// Model Profiles (admin-runtime-config, owner: ai-gateway)
// =========================================================================

/** GET /admin/model-profiles?purpose=&enabled= */
export async function listModelProfiles(params?: {
  purpose?: string
  enabled?: boolean
}): Promise<ModelProfile[]> {
  return gatewayRequest<ModelProfile[]>(
    `/admin/model-profiles${buildQuery({
      purpose: params?.purpose,
      enabled: params?.enabled,
    })}`,
  )
}

/** POST /admin/model-profiles */
export async function createModelProfile(params: CreateModelProfileRequest): Promise<ModelProfile> {
  return gatewayRequest<ModelProfile>('/admin/model-profiles', {
    method: 'POST',
    body: params,
  })
}

/** GET /admin/model-profiles/{profileId} */
export async function getModelProfile(profileId: string): Promise<ModelProfile> {
  return gatewayRequest<ModelProfile>(`/admin/model-profiles/${encodeURIComponent(profileId)}`)
}

/** PATCH /admin/model-profiles/{profileId} */
export async function updateModelProfile(
  profileId: string,
  params: UpdateModelProfileRequest,
): Promise<ModelProfile> {
  return gatewayRequest<ModelProfile>(`/admin/model-profiles/${encodeURIComponent(profileId)}`, {
    method: 'PATCH',
    body: params,
  })
}

/** DELETE /admin/model-profiles/{profileId} */
export async function deleteModelProfile(profileId: string): Promise<void> {
  await requestVoid(`/admin/model-profiles/${encodeURIComponent(profileId)}`, { method: 'DELETE' })
}

// =========================================================================
// Parser Configs (admin-runtime-config, owner: knowledge)
// =========================================================================

/** GET /admin/parser-configs?enabled= */
export async function listParserConfigs(params?: { enabled?: boolean }): Promise<ParserConfig[]> {
  return gatewayRequest<ParserConfig[]>(
    `/admin/parser-configs${buildQuery({ enabled: params?.enabled })}`,
  )
}

/** POST /admin/parser-configs */
export async function createParserConfig(params: CreateParserConfigRequest): Promise<ParserConfig> {
  return gatewayRequest<ParserConfig>('/admin/parser-configs', {
    method: 'POST',
    body: params,
  })
}

/** GET /admin/parser-configs/{parserConfigId} */
export async function getParserConfig(parserConfigId: string): Promise<ParserConfig> {
  return gatewayRequest<ParserConfig>(`/admin/parser-configs/${encodeURIComponent(parserConfigId)}`)
}

/** PATCH /admin/parser-configs/{parserConfigId} */
export async function updateParserConfig(
  parserConfigId: string,
  params: UpdateParserConfigRequest,
): Promise<ParserConfig> {
  return gatewayRequest<ParserConfig>(
    `/admin/parser-configs/${encodeURIComponent(parserConfigId)}`,
    { method: 'PATCH', body: params },
  )
}

/** DELETE /admin/parser-configs/{parserConfigId} */
export async function deleteParserConfig(parserConfigId: string): Promise<void> {
  await requestVoid(`/admin/parser-configs/${encodeURIComponent(parserConfigId)}`, {
    method: 'DELETE',
  })
}

// =========================================================================
// Auth (re-exported from ./auth)
// =========================================================================
