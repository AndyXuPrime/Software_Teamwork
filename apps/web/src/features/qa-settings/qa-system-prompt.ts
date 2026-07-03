import type { CreateQAConfigVersionRequest, QAConfigVersion } from './qa-settings.types'

export const QA_SYSTEM_PROMPT_MAX_BYTES = 20_000

const encoder = new TextEncoder()

export type PromptValidationResult =
  { ok: true; bytes: number } | { ok: false; bytes: number; message: string }

export function countUtf8Bytes(value: string): number {
  return encoder.encode(value).byteLength
}

export function validateSystemPrompt(value: string): PromptValidationResult {
  const bytes = countUtf8Bytes(value)

  if (value.trim().length === 0) {
    return { ok: false, bytes, message: '系统提示词不能为空' }
  }

  if (bytes > QA_SYSTEM_PROMPT_MAX_BYTES) {
    return {
      ok: false,
      bytes,
      message: `系统提示词不能超过 ${QA_SYSTEM_PROMPT_MAX_BYTES} UTF-8 bytes`,
    }
  }

  return { ok: true, bytes }
}

export function buildSystemPromptPayload(
  current: QAConfigVersion,
  systemPrompt: string,
): CreateQAConfigVersionRequest {
  return {
    defaultKnowledgeBaseIds: current.defaultKnowledgeBaseIds,
    knowledgeBases: current.knowledgeBases,
    retrieval: current.retrieval,
    maxIterations: current.maxIterations,
    toolTimeoutSeconds: current.toolTimeoutSeconds,
    modelTimeoutSeconds: current.modelTimeoutSeconds,
    overallTimeoutSeconds: current.overallTimeoutSeconds,
    enabledToolNames: current.enabledToolNames,
    agent: current.agent,
    systemPrompt,
    activate: true,
  }
}
