import { describe, expect, it } from 'vitest'

import type { QAConfigVersion } from './qa-settings.types'
import {
  buildSystemPromptPayload,
  countUtf8Bytes,
  QA_SYSTEM_PROMPT_MAX_BYTES,
  validateSystemPrompt,
} from './qa-system-prompt'

const currentConfig: QAConfigVersion = {
  id: 'qa-config-1',
  versionNo: 8,
  defaultKnowledgeBaseIds: ['kb-1'],
  knowledgeBases: [{ id: 'kb-1', displayName: '主库', sortOrder: 1, type: 'default' }],
  retrieval: { enableRerank: true, rerankTopN: 4, topK: 6 },
  maxIterations: 5,
  toolTimeoutSeconds: 11,
  modelTimeoutSeconds: 61,
  overallTimeoutSeconds: 121,
  enabledToolNames: ['knowledge.search', 'report.create'],
  llm: {
    id: 'llm-1',
    versionNo: 3,
    provider: 'ai-gateway',
    profileId: 'profile-chat',
    modelName: 'gpt-5.5',
    timeoutSeconds: 60,
    temperature: 0.2,
    maxTokens: 4096,
    isActive: true,
    createdAt: '2026-07-02T08:00:00Z',
  },
  agent: {
    maxIterations: 5,
    toolTimeoutSeconds: 11,
    modelTimeoutSeconds: 61,
    overallTimeoutSeconds: 121,
    enabledToolNames: ['knowledge.search', 'report.create'],
  },
  systemPrompt: '旧提示词',
  isActive: true,
  createdAt: '2026-07-02T08:10:00Z',
}

describe('qa-system-prompt helpers', () => {
  it('counts UTF-8 bytes and validates empty and oversized prompts', () => {
    expect(countUtf8Bytes('abc')).toBe(3)
    expect(countUtf8Bytes('电力')).toBe(6)
    expect(validateSystemPrompt('   ')).toMatchObject({
      ok: false,
      message: '系统提示词不能为空',
    })
    expect(validateSystemPrompt('a'.repeat(QA_SYSTEM_PROMPT_MAX_BYTES + 1))).toMatchObject({
      ok: false,
      bytes: QA_SYSTEM_PROMPT_MAX_BYTES + 1,
    })
    expect(validateSystemPrompt('新的全局提示词')).toMatchObject({ ok: true, bytes: 21 })
  })

  it('builds a new QA config payload while preserving non-prompt fields', () => {
    const payload = buildSystemPromptPayload(currentConfig, '新的全局提示词')

    expect(payload).toEqual({
      defaultKnowledgeBaseIds: currentConfig.defaultKnowledgeBaseIds,
      knowledgeBases: currentConfig.knowledgeBases,
      retrieval: currentConfig.retrieval,
      maxIterations: currentConfig.maxIterations,
      toolTimeoutSeconds: currentConfig.toolTimeoutSeconds,
      modelTimeoutSeconds: currentConfig.modelTimeoutSeconds,
      overallTimeoutSeconds: currentConfig.overallTimeoutSeconds,
      enabledToolNames: currentConfig.enabledToolNames,
      agent: currentConfig.agent,
      systemPrompt: '新的全局提示词',
      activate: true,
    })
    expect(payload).not.toHaveProperty('llm')
  })
})
