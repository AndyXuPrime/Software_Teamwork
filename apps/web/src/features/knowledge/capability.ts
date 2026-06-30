import { ApiError } from '@/api/client'

export type GatewayCapabilityIssueKind =
  'not_ready' | 'dependency_failed' | 'forbidden' | 'unauthorized' | 'error'

export type GatewayCapabilityIssue = {
  description: string
  kind: GatewayCapabilityIssueKind
  requestId?: string
  requestIdText: string
  title: string
  variant: 'error' | 'warning'
}

const MISSING_REQUEST_ID_TEXT = '响应未包含 requestId，无法关联后端日志'

function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : '未知错误'
}

function getRequestIdText(error: ApiError): string {
  return error.requestId ? `requestId: ${error.requestId}` : MISSING_REQUEST_ID_TEXT
}

function isNotImplementedError(error: ApiError): boolean {
  return (
    error.status === 501 ||
    error.code === 'not_implemented' ||
    error.code === 'not_implemented_error' ||
    error.code === 'http_501'
  )
}

function isDependencyError(error: ApiError): boolean {
  return error.status === 502 || error.code === 'dependency_error'
}

export function getGatewayCapabilityIssue(
  error: unknown,
  featureName: string,
): GatewayCapabilityIssue {
  if (error instanceof ApiError) {
    const requestIdText = getRequestIdText(error)

    if (isNotImplementedError(error)) {
      return {
        description: `Gateway 已暴露 ${featureName}，但后端工作流尚未就绪。${error.message}（${requestIdText}）`,
        kind: 'not_ready',
        requestId: error.requestId,
        requestIdText,
        title: `${featureName}暂未就绪`,
        variant: 'warning',
      }
    }

    if (isDependencyError(error)) {
      return {
        description: `${featureName}依赖的后端服务暂不可用。${error.message}（${requestIdText}）`,
        kind: 'dependency_failed',
        requestId: error.requestId,
        requestIdText,
        title: `${featureName}依赖失败`,
        variant: 'error',
      }
    }

    if (error.isForbidden()) {
      return {
        description: `当前账号没有执行 ${featureName} 的权限。${error.message}（${requestIdText}）`,
        kind: 'forbidden',
        requestId: error.requestId,
        requestIdText,
        title: '权限不足',
        variant: 'error',
      }
    }

    if (error.isUnauthorized()) {
      return {
        description: `登录状态已失效，请重新登录。${error.message}（${requestIdText}）`,
        kind: 'unauthorized',
        requestId: error.requestId,
        requestIdText,
        title: '认证失效',
        variant: 'error',
      }
    }

    return {
      description: `${error.message}（${requestIdText}）`,
      kind: 'error',
      requestId: error.requestId,
      requestIdText,
      title: `${featureName}失败`,
      variant: 'error',
    }
  }

  return {
    description: `${getErrorMessage(error)}（非 Gateway 错误，未包含 requestId）`,
    kind: 'error',
    requestIdText: '非 Gateway 错误，未包含 requestId',
    title: `${featureName}失败`,
    variant: 'error',
  }
}

export function formatGatewayCapabilityError(error: unknown, featureName: string): string {
  const issue = getGatewayCapabilityIssue(error, featureName)
  return `${issue.title}: ${issue.description}`
}

export function isCapabilityUnavailable(error: unknown): boolean {
  if (!(error instanceof ApiError)) return false
  return isNotImplementedError(error) || isDependencyError(error)
}
