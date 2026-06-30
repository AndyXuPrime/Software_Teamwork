import { describe, expect, it } from 'vitest'

import { ApiError } from '@/api/client'

import {
  formatGatewayCapabilityError,
  getGatewayCapabilityIssue,
  isCapabilityUnavailable,
} from './capability'

describe('knowledge capability error helpers', () => {
  it('classifies 501 and not_implemented as not ready with request id', () => {
    const issue = getGatewayCapabilityIssue(
      new ApiError({
        code: 'not_implemented',
        message: 'route is not implemented',
        requestId: 'req-501',
        status: 501,
      }),
      '知识检索',
    )

    expect(issue).toMatchObject({
      kind: 'not_ready',
      requestId: 'req-501',
      requestIdText: 'requestId: req-501',
      title: '知识检索暂未就绪',
      variant: 'warning',
    })
    expect(issue.description).toContain('route is not implemented')
    expect(
      isCapabilityUnavailable(new ApiError({ code: 'http_501', message: 'nope', status: 501 })),
    ).toBe(true)
  })

  it('classifies dependency_error separately and reports missing request id', () => {
    const issue = getGatewayCapabilityIssue(
      new ApiError({
        code: 'dependency_error',
        message: 'knowledge service unavailable',
        status: 502,
      }),
      '文档分块',
    )

    expect(issue.kind).toBe('dependency_failed')
    expect(issue.requestId).toBeUndefined()
    expect(issue.requestIdText).toContain('响应未包含 requestId')
    expect(issue.description).toContain('knowledge service unavailable')
  })

  it('keeps forbidden distinct from readiness failures', () => {
    const text = formatGatewayCapabilityError(
      new ApiError({
        code: 'forbidden',
        message: 'permission denied',
        requestId: 'req-denied',
        status: 403,
      }),
      '删除文档',
    )

    expect(text).toContain('权限不足')
    expect(text).toContain('requestId: req-denied')
  })
})
