export const MODEL_CONFIGURATION_HINT =
  '请先配置模型：请在模型管理中新增并启用聊天模型，并在问答或报告配置中发布生效。'

type ModelConfigurationErrorLike = {
  code?: string
  fields?: Record<string, string>
  message?: string
  status?: number
}

const modelConfigCodes = new Set(['model_error'])

const modelConfigMessagePatterns = [
  /\bLLM configuration not found\b/i,
  /\bmodel profile not found\b/i,
  /\bmodel profile\b.*\bnot configured\b/i,
  /\bprofile validator is not configured\b/i,
  /\bai gateway chat client is not configured\b/i,
  /\bmodel\b.*\bnot configured\b/i,
  /\bprofileId\b.*\bdoes not exist\b/i,
  /\bprofileId\b.*\bmust reference\b/i,
  /模型.*未配置/,
  /请先配置.*模型/,
]

function fieldText(fields: Record<string, string> | undefined): string {
  if (!fields) return ''
  return Object.entries(fields)
    .map(([key, value]) => `${key}: ${value}`)
    .join('\n')
}

export function isModelConfigurationError(error: ModelConfigurationErrorLike): boolean {
  if (error.code && modelConfigCodes.has(error.code)) return true
  const searchable = [error.message, fieldText(error.fields)].filter(Boolean).join('\n')
  if (!searchable) return false
  return modelConfigMessagePatterns.some((pattern) => pattern.test(searchable))
}
