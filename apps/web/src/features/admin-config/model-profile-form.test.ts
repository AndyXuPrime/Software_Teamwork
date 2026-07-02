import { describe, expect, it } from 'vitest'

import { ApiError } from '@/api/client'

import {
  buildCreateModelProfileRequest,
  buildUpdateModelProfileRequest,
  formatModelProfileError,
  type ModelProfileFormValues,
  validateCreateModelProfileForm,
  validateUpdateModelProfileForm,
} from './model-profile-form'

const baseForm: ModelProfileFormValues = {
  apiKey: 'sk-local-test',
  baseUrl: 'https://api.example.invalid/v1',
  dimension: 0,
  enabled: true,
  isDefault: false,
  maxTokens: 0,
  model: 'test-model',
  name: 'test profile',
  provider: 'openai_compatible',
  purpose: 'chat',
  supportsStreaming: true,
  timeoutMs: 60000,
  topN: 0,
}

describe('model profile form helpers', () => {
  it('omits max_tokens from create and update payloads by default', () => {
    expect(buildCreateModelProfileRequest(baseForm)).not.toHaveProperty('defaultParameters')
    expect(buildUpdateModelProfileRequest(baseForm)).not.toHaveProperty('defaultParameters')
  })

  it('passes status and default controls through create and update payloads', () => {
    const form: ModelProfileFormValues = { ...baseForm, enabled: false, isDefault: true }

    expect(buildCreateModelProfileRequest(form)).toMatchObject({
      enabled: false,
      isDefault: true,
    })
    expect(buildUpdateModelProfileRequest(form)).toMatchObject({
      enabled: false,
      isDefault: true,
    })
  })

  it('does not emit max_tokens even when the local input is positive', () => {
    const form = { ...baseForm, maxTokens: 512 }

    expect(buildCreateModelProfileRequest(form)).not.toHaveProperty('defaultParameters')
    expect(buildUpdateModelProfileRequest(form)).not.toHaveProperty('defaultParameters')
  })

  it('requires dimensions for embedding profiles before create or update submit', () => {
    const form: ModelProfileFormValues = { ...baseForm, purpose: 'embedding' }

    expect(validateCreateModelProfileForm(form)).toEqual({
      isValid: false,
      message: '请填写向量维度',
    })
    expect(validateUpdateModelProfileForm(form)).toEqual({
      isValid: false,
      message: '请填写向量维度',
    })
  })

  it('requires topN for rerank profiles before create or update submit', () => {
    const form: ModelProfileFormValues = { ...baseForm, purpose: 'rerank' }

    expect(validateCreateModelProfileForm(form)).toEqual({
      isValid: false,
      message: '请填写默认 TopN',
    })
    expect(validateUpdateModelProfileForm(form)).toEqual({
      isValid: false,
      message: '请填写默认 TopN',
    })
  })

  it('includes ApiError field details and request id in user-visible failures', () => {
    const error = new ApiError({
      code: 'validation_error',
      fields: { defaultParameters: 'must not contain sensitive keys' },
      message: 'request validation failed',
      requestId: 'req-445',
      status: 400,
    })

    expect(formatModelProfileError(error, '创建失败')).toBe(
      '创建失败: request validation failed；defaultParameters: must not contain sensitive keys（requestId: req-445）',
    )
  })
})
