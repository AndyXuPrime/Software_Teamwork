import type {
  KnowledgeBaseConfig,
  LLMConfig,
  RAGDefaults,
  RAGSearchResult,
  StatsOverview,
  TopQuery,
  TrendPoint,
} from '@/lib/types'

import { ApiError, requestJson, requestPaginated, requestVoid } from './client'
import type { components } from './generated/gateway'

type KnowledgeBaseSummary = components['schemas']['KnowledgeBaseSummary']
type ModelProfile = components['schemas']['ModelProfile']
type QALLMConfigVersion = components['schemas']['QALLMConfigVersion']
type QALLMConnectionTest = components['schemas']['QALLMConnectionTest']
type QAConfigVersion = components['schemas']['QAConfigVersion']
type QAMetricsOverview = components['schemas']['QAMetricsOverview']
type QAMetricsTrend = components['schemas']['QAMetricsTrend']
type QATopQuery = components['schemas']['QATopQuery']
type QAIntentDistributionItem = components['schemas']['QAIntentDistributionItem']
type QARetrievalTestRun = components['schemas']['QARetrievalTestRun']
type QARetrievalTestResult = components['schemas']['QARetrievalTestResult']

export interface LLMConfigResponse extends LLMConfig {
  extra_headers: Record<string, string>
  updated_at: string
  profile_id: string
}

export interface LLMTestRequest {
  api_url: string
  api_key: string
  model_name: string
}

export interface LLMTestResponse {
  success: boolean
  latency_ms: number
  model: string
  tested_at: string
}

export interface KnowledgeConfigResponse {
  knowledge_bases: KnowledgeBaseConfig[]
  defaults: RAGDefaults
}

export interface RAGTestRequest {
  query: string
  knowledge_bases_override?: string[] | null
  top_k_override?: number | null
  similarity_threshold_override?: number | null
}

export interface RAGTestResponse {
  query: string
  mode: string
  results: RAGSearchResult[]
  total_hits: number
  took_ms: number
}

export interface StatsTrendResponse {
  days: number
  points: TrendPoint[]
}

export interface TopQueriesResponse {
  items: TopQuery[]
}

export interface IntentDistributionItem {
  intent: string
  label: string
  count: number
  percent: number
}

export interface IntentDistributionResponse {
  items: IntentDistributionItem[]
}

export interface CreateKnowledgeBaseRequest {
  name: string
  description?: string
}

export interface AdminUser {
  id: string
  username: string
  role: string
  created_at: string
}

export interface AdminRole {
  id: string
  name: string
  permissions: string[]
}

function unsupportedAdminContract(): never {
  throw new ApiError({
    code: 'unsupported_mode',
    message: 'This admin contract is not active in the gateway OpenAPI yet.',
    status: 0,
  })
}

function toKnowledgeBaseConfig(kb: KnowledgeBaseSummary): KnowledgeBaseConfig {
  return {
    id: kb.id,
    name: kb.name,
    doc_count: kb.documentCount,
    status: 'active',
  }
}

function toLLMConfig(config?: QALLMConfigVersion, profile?: ModelProfile): LLMConfigResponse {
  return {
    api_url: profile?.baseUrl ?? '',
    api_key: profile?.apiKeyConfigured ? '********' : '',
    model_name: config?.modelName ?? profile?.model ?? '',
    timeout: config?.timeoutSeconds ?? Math.round((profile?.timeoutMs ?? 30000) / 1000),
    temperature: config?.temperature ?? 0.7,
    max_tokens: config?.maxTokens ?? 2048,
    extra_headers: {},
    updated_at: config?.createdAt ?? profile?.updatedAt ?? '',
    profile_id: config?.profileId ?? profile?.id ?? '',
  }
}

function toStatsOverview(data: QAMetricsOverview): StatsOverview {
  return {
    total_qa_count: data.totalQaCount ?? data.totalQuestionCount ?? 0,
    today_qa_count: data.todayQaCount ?? 0,
    avg_latency_ms: data.avgLatencyMs ?? 0,
    active_users_today: data.activeUsersToday ?? 0,
    knowledge_base_count: data.knowledgeBaseCount ?? 0,
    document_count: data.documentCount ?? 0,
  }
}

function toRagResult(result: QARetrievalTestResult): RAGSearchResult {
  return {
    rank: result.rankNo,
    chunk_id: result.chunkId ?? '',
    doc_id: result.documentId ?? result.docId ?? '',
    doc_name: result.documentName ?? result.docName ?? '',
    text: result.contentPreview ?? result.text ?? '',
    vector_score: result.vectorScore ?? result.score ?? 0,
    rerank_score: result.rerankScore,
  }
}

async function getDefaultChatProfile(): Promise<ModelProfile | undefined> {
  const profiles = await requestJson<ModelProfile[]>(
    '/admin/model-profiles?purpose=chat&enabled=true',
  )
  return profiles.find((profile) => profile.isDefault) ?? profiles[0]
}

export async function getLLMConfig(): Promise<LLMConfigResponse> {
  const [llmConfig, profile] = await Promise.all([
    requestJson<QALLMConfigVersion>('/llm-config-versions/current').catch(() => undefined),
    getDefaultChatProfile().catch(() => undefined),
  ])
  return toLLMConfig(llmConfig, profile)
}

export async function updateLLMConfig(config: Partial<LLMConfig>): Promise<LLMConfigResponse> {
  const current = await getLLMConfig()
  const profileId = current.profile_id
  if (!profileId) {
    throw new ApiError({
      code: 'validation_error',
      message: 'No active chat model profile is available.',
      status: 400,
    })
  }

  const next = await requestJson<QALLMConfigVersion>('/llm-config-versions', {
    method: 'POST',
    body: {
      provider: 'ai-gateway',
      profileId,
      modelName: config.model_name ?? current.model_name,
      timeoutSeconds: config.timeout ?? current.timeout,
      temperature: config.temperature ?? current.temperature,
      maxTokens: config.max_tokens ?? current.max_tokens,
      activate: true,
    } satisfies components['schemas']['CreateQALLMConfigVersionRequest'],
  })

  return toLLMConfig(next, await getDefaultChatProfile().catch(() => undefined))
}

export async function testLLMConnection(params: LLMTestRequest): Promise<LLMTestResponse> {
  const profile = await getDefaultChatProfile()
  if (!profile) {
    throw new ApiError({
      code: 'validation_error',
      message: 'No active chat model profile is available.',
      status: 400,
    })
  }

  const result = await requestJson<QALLMConnectionTest>('/llm-connection-tests', {
    method: 'POST',
    body: {
      provider: 'ai-gateway',
      profileId: profile.id,
      modelName: params.model_name || profile.model,
    } satisfies components['schemas']['CreateQALLMConnectionTestRequest'],
  })

  return {
    success: result.success,
    latency_ms: result.latencyMs ?? 0,
    model: result.modelName ?? params.model_name,
    tested_at: result.testedAt,
  }
}

export async function getKnowledgeConfig(): Promise<KnowledgeConfigResponse> {
  const [knowledgeBases, qaConfig] = await Promise.all([
    listKnowledgeBases(),
    requestJson<QAConfigVersion>('/qa-config-versions/current').catch(() => undefined),
  ])
  const retrieval = qaConfig?.retrieval

  return {
    knowledge_bases: knowledgeBases,
    defaults: {
      knowledge_bases: qaConfig?.defaultKnowledgeBaseIds ?? [],
      top_k: retrieval?.topK ?? 10,
      similarity_threshold: retrieval?.scoreThreshold ?? 0,
      use_rerank: retrieval?.enableRerank ?? retrieval?.useRerank ?? false,
      rerank_threshold: 0,
      rerank_top_n: retrieval?.rerankTopN ?? 0,
    },
  }
}

export async function updateKnowledgeConfig(
  defaults: Partial<RAGDefaults>,
): Promise<KnowledgeConfigResponse> {
  await requestJson<QAConfigVersion>('/qa-config-versions', {
    method: 'POST',
    body: {
      defaultKnowledgeBaseIds: defaults.knowledge_bases,
      retrieval: {
        topK: defaults.top_k,
        scoreThreshold: defaults.similarity_threshold,
        enableRerank: defaults.use_rerank,
        useRerank: defaults.use_rerank,
        rerankTopN: defaults.rerank_top_n,
      },
      activate: true,
    } satisfies components['schemas']['CreateQAConfigVersionRequest'],
  })
  return getKnowledgeConfig()
}

export async function ragTest(params: RAGTestRequest): Promise<RAGTestResponse> {
  const run = await requestJson<QARetrievalTestRun>('/retrieval-test-runs', {
    method: 'POST',
    body: {
      question: params.query,
      query: params.query,
      knowledgeBaseIds: params.knowledge_bases_override ?? undefined,
      retrieval: {
        topK: params.top_k_override ?? undefined,
        scoreThreshold: params.similarity_threshold_override ?? undefined,
      },
    } satisfies components['schemas']['CreateQARetrievalTestRunRequest'],
  })
  const results = run.results ?? []

  return {
    query: run.question ?? run.query ?? params.query,
    mode: 'retrieval_test',
    results: results.map(toRagResult),
    total_hits: results.length,
    took_ms: 0,
  }
}

export async function getStatsOverview(): Promise<StatsOverview> {
  return toStatsOverview(await requestJson<QAMetricsOverview>('/qa-metrics/overview'))
}

export async function getStatsTrend(days = 30): Promise<StatsTrendResponse> {
  const params = new URLSearchParams({ days: String(days) })
  const data = await requestJson<QAMetricsTrend>(`/qa-metrics/trend?${params}`)
  return {
    days: data.days,
    points: data.points.map((point) => ({
      date: point.date,
      count: point.count ?? point.questionCount ?? 0,
    })),
  }
}

export async function getTopQueries(limit = 10, daysParam = 7): Promise<TopQueriesResponse> {
  const params = new URLSearchParams({
    limit: String(limit),
    days: String(daysParam),
  })
  const data = await requestJson<QATopQuery[]>(`/qa-metrics/top-queries?${params}`)
  return {
    items: data.map((item) => ({
      query: item.query,
      count: item.count,
      avg_accuracy_score: 0,
      last_asked_at: item.lastAskedAt ?? '',
    })),
  }
}

export async function getIntentDistribution(daysParam = 7): Promise<IntentDistributionResponse> {
  const params = new URLSearchParams({ days: String(daysParam) })
  const data = await requestJson<QAIntentDistributionItem[]>(
    `/qa-metrics/intent-distribution?${params}`,
  )
  return {
    items: data.map((item) => ({
      intent: item.intent,
      label: item.label ?? item.intent,
      count: item.count,
      percent: item.percent ?? 0,
    })),
  }
}

export async function listKnowledgeBases(): Promise<KnowledgeBaseConfig[]> {
  const envelope = await requestPaginated<KnowledgeBaseSummary>(
    '/knowledge-bases?page=1&pageSize=100',
  )
  return envelope.data.map(toKnowledgeBaseConfig)
}

export async function createKnowledgeBase(
  params: CreateKnowledgeBaseRequest,
): Promise<KnowledgeBaseConfig> {
  const kb = await requestJson<KnowledgeBaseSummary>('/knowledge-bases', {
    method: 'POST',
    body: {
      name: params.name,
      description: params.description ?? '',
    },
  })
  return toKnowledgeBaseConfig(kb)
}

export async function deleteKnowledgeBase(id: string): Promise<void> {
  await requestVoid(`/knowledge-bases/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

export function listUsers(): Promise<AdminUser[]> {
  unsupportedAdminContract()
}

export function createUser(_body: Record<string, unknown>): Promise<AdminUser> {
  unsupportedAdminContract()
}

export function updateUser(_id: string, _body: Record<string, unknown>): Promise<AdminUser> {
  unsupportedAdminContract()
}

export function deleteUser(_id: string): Promise<void> {
  unsupportedAdminContract()
}

export function listRoles(): Promise<AdminRole[]> {
  unsupportedAdminContract()
}

export function createRole(_body: Record<string, unknown>): Promise<AdminRole> {
  unsupportedAdminContract()
}

export function updateRole(_id: string, _body: Record<string, unknown>): Promise<AdminRole> {
  unsupportedAdminContract()
}

export function deleteRole(_id: string): Promise<void> {
  unsupportedAdminContract()
}
