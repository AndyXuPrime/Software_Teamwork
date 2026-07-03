import { describe, expect, it } from 'vitest'

import { ApiError } from '@/api/client'

import { formatReportGatewayError, getReportGatewayErrorDetails } from './report-generation.errors'

describe('report generation gateway error helpers', () => {
  it('preserves gateway message and request id for user-visible diagnostics', () => {
    const error = new ApiError({
      code: 'dependency_error',
      message: 'Document service unavailable',
      requestId: 'req-report-1',
      status: 503,
    })

    expect(getReportGatewayErrorDetails(error)).toEqual({
      code: 'dependency_error',
      isCapabilityUnavailable: true,
      message: 'Document service unavailable',
      requestId: 'req-report-1',
      status: 503,
    })
    expect(formatReportGatewayError(error)).toBe(
      'Document service unavailable（requestId: req-report-1）',
    )
  })

  it('maps missing model configuration errors to a friendly setup hint', () => {
    const error = new ApiError({
      code: 'dependency_error',
      message: 'ai gateway chat client is not configured',
      requestId: 'req-model-missing',
      status: 502,
    })

    expect(formatReportGatewayError(error)).toBe(
      '请先配置模型：请在模型管理中新增并启用聊天模型，并在问答或报告配置中发布生效。（requestId: req-model-missing）',
    )
  })
})
