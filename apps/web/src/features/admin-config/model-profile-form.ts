import { ApiError } from '@/api/client'
import type {
  CreateModelProfileRequest,
  ModelProvider,
  ModelPurpose,
  UpdateModelProfileRequest,
} from '@/lib/types'

export type ModelProfileFormValues = {
  name: string
  purpose: ModelPurpose
  provider: ModelProvider
  baseUrl: string
  model: string
  apiKey: string
  enabled: boolean
  isDefault: boolean
  timeoutMs: number
  maxTokens: number
  dimension: number
  topN: number
  supportsStreaming: boolean
}

type ValidationResult = { isValid: true } | { isValid: false; message: string }

function hasValue(value: string): boolean {
  return value.trim().length > 0
}

function normalizedTimeout(value: number): number {
  return Math.max(1000, value || 60000)
}

function validatePurposeFields(form: ModelProfileFormValues): ValidationResult {
  if (form.purpose === 'embedding' && form.dimension <= 0) {
    return { isValid: false, message: '请填写向量维度' }
  }
  if (form.purpose === 'rerank' && form.topN <= 0) {
    return { isValid: false, message: '请填写默认 TopN' }
  }
  return { isValid: true }
}

function addPurposeFields(
  request: CreateModelProfileRequest | UpdateModelProfileRequest,
  form: ModelProfileFormValues,
): void {
  if (form.purpose === 'embedding' && form.dimension > 0) {
    request.dimensions = form.dimension
  }
  if (form.purpose === 'rerank' && form.topN > 0) {
    request.topN = form.topN
  }
}

export function validateCreateModelProfileForm(form: ModelProfileFormValues): ValidationResult {
  if (
    !hasValue(form.name) ||
    !hasValue(form.purpose) ||
    !hasValue(form.provider) ||
    !hasValue(form.baseUrl) ||
    !hasValue(form.model)
  ) {
    return { isValid: false, message: '请填写名称、类型、服务商、地址和模型名称' }
  }
  if (!hasValue(form.apiKey)) {
    return { isValid: false, message: '请填写 API Key' }
  }
  return validatePurposeFields(form)
}

export function validateUpdateModelProfileForm(form: ModelProfileFormValues): ValidationResult {
  if (
    !hasValue(form.name) ||
    !hasValue(form.provider) ||
    !hasValue(form.baseUrl) ||
    !hasValue(form.model)
  ) {
    return { isValid: false, message: '请填写名称、服务商、地址和模型名称' }
  }
  return validatePurposeFields(form)
}

export function buildCreateModelProfileRequest(
  form: ModelProfileFormValues,
): CreateModelProfileRequest {
  const request: CreateModelProfileRequest = {
    name: form.name.trim(),
    purpose: form.purpose,
    provider: form.provider,
    baseUrl: form.baseUrl.trim(),
    model: form.model.trim(),
    apiKey: form.apiKey.trim(),
    enabled: form.enabled,
    isDefault: form.isDefault,
    timeoutMs: normalizedTimeout(form.timeoutMs),
    supportsStreaming: form.purpose === 'chat' ? form.supportsStreaming : false,
  }

  addPurposeFields(request, form)
  return request
}

export function buildUpdateModelProfileRequest(
  form: ModelProfileFormValues,
): UpdateModelProfileRequest {
  const request: UpdateModelProfileRequest = {
    name: form.name.trim(),
    provider: form.provider,
    baseUrl: form.baseUrl.trim(),
    model: form.model.trim(),
    enabled: form.enabled,
    isDefault: form.isDefault,
    timeoutMs: normalizedTimeout(form.timeoutMs),
    ...(form.purpose === 'chat' ? { supportsStreaming: form.supportsStreaming } : {}),
  }

  if (hasValue(form.apiKey)) {
    request.apiKey = form.apiKey.trim()
  }

  addPurposeFields(request, form)
  return request
}

export function formatModelProfileError(error: unknown, fallback: string): string {
  const message = error instanceof Error && hasValue(error.message) ? error.message : '未知错误'
  const detailParts =
    error instanceof ApiError && error.fields
      ? Object.entries(error.fields).map(([field, fieldMessage]) => `${field}: ${fieldMessage}`)
      : []
  const requestId =
    error instanceof ApiError && error.requestId ? `（requestId: ${error.requestId}）` : ''

  return `${fallback}: ${[message, ...detailParts].join('；')}${requestId}`
}
