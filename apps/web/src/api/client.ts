import { activeGatewayPathSet, type GatewayPath } from './active-paths'

const DEFAULT_GATEWAY_BASE_URL = '/api/v1'
const JSON_CONTENT_TYPE = 'application/json'
const SSE_CONTENT_TYPE = 'text/event-stream'

export type GatewayMethod = 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'

export type GatewayErrorBody = {
  code: string
  message: string
  requestId?: string
  fields?: Record<string, string>
}

export type GatewaySuccessEnvelope<T> = {
  data: T
  requestId: string
}

export type GatewayPaginatedEnvelope<T> = GatewaySuccessEnvelope<T[]> & {
  page: {
    page: number
    pageSize: number
    total: number
  }
}

type GatewayErrorEnvelope = {
  error: GatewayErrorBody
}

export class ApiError extends Error {
  code: string
  status: number
  requestId?: string
  fields?: Record<string, string>

  constructor(
    params:
      | {
          code: string
          message: string
          status: number
          requestId?: string
          fields?: Record<string, string>
        }
      | string
      | number,
    message?: string,
    options?: { requestId?: string; fields?: Record<string, string> },
  ) {
    const normalized =
      typeof params === 'object'
        ? params
        : {
            code: String(params),
            message: message ?? 'Request failed',
            status: typeof params === 'number' ? params : 0,
            requestId: options?.requestId,
            fields: options?.fields,
          }

    super(normalized.message)
    this.name = 'ApiError'
    this.code = normalized.code
    this.status = normalized.status
    this.requestId = normalized.requestId
    this.fields = normalized.fields
  }
}

type RequestBody = BodyInit | Record<string, unknown> | unknown[] | null

export type GatewayPage = GatewayPaginatedEnvelope<unknown>['page']

export type GatewayRequestOptions = Omit<RequestInit, 'body' | 'method'> & {
  body?: RequestBody
  method?: GatewayMethod
  requestId?: string
  token?: string | null
}

type GatewayStreamOptions = GatewayRequestOptions & {
  onEvent: (event: SseEvent) => void
  onError?: (error: ApiError) => void
  onDone?: () => void
}

export type SseEvent = {
  event: string
  data: string
  id?: string
  retry?: number
}

type MockHandler = (request: Request) => Response | Promise<Response>

type MockRoute = {
  method: GatewayMethod
  path: GatewayPath
  handler: MockHandler
}

let accessTokenProvider: (() => string | null | undefined) | undefined
let requestIdProvider: (() => string | undefined) | undefined
let mockRoutes: MockRoute[] = []

export const apiClient = {
  get baseUrl() {
    return getGatewayBaseUrl()
  },
  setAccessTokenProvider(provider: typeof accessTokenProvider) {
    accessTokenProvider = provider
  },
  setRequestIdProvider(provider: typeof requestIdProvider) {
    requestIdProvider = provider
  },
  setMockRoutes(routes: readonly MockRoute[]) {
    mockRoutes = routes.map((route) => {
      assertActiveGatewayPath(route.path)
      return route
    })
  },
  clearMockRoutes() {
    mockRoutes = []
  },
}

function getGatewayBaseUrl(): string {
  const configured = import.meta.env?.VITE_API_BASE_URL as string | undefined
  return stripTrailingSlash(configured || DEFAULT_GATEWAY_BASE_URL)
}

function stripTrailingSlash(value: string): string {
  return value.endsWith('/') ? value.slice(0, -1) : value
}

function joinUrl(
  path: string,
  query?: URLSearchParams | Record<string, string | number | boolean | null | undefined>,
): string {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  const url = `${getGatewayBaseUrl()}${normalizedPath}`
  const params = query instanceof URLSearchParams ? query : toSearchParams(query)
  const queryString = params?.toString()
  return queryString ? `${url}?${queryString}` : url
}

function toSearchParams(
  query?: Record<string, string | number | boolean | null | undefined>,
): URLSearchParams | undefined {
  if (!query) return undefined
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value == null) continue
    params.set(key, String(value))
  }
  return params
}

function buildHeaders(options: GatewayRequestOptions, hasJsonBody: boolean): Headers {
  const headers = new Headers(options.headers)
  const token = options.token ?? accessTokenProvider?.()
  const requestId = options.requestId ?? requestIdProvider?.()

  if (hasJsonBody && !headers.has('Content-Type')) {
    headers.set('Content-Type', JSON_CONTENT_TYPE)
  }
  if (!headers.has('Accept')) {
    headers.set('Accept', JSON_CONTENT_TYPE)
  }
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  if (requestId) {
    headers.set('X-Request-Id', requestId)
  }

  return headers
}

function prepareBody(body: RequestBody | undefined): { body?: BodyInit; hasJsonBody: boolean } {
  if (body == null) return { hasJsonBody: false }
  if (
    body instanceof FormData ||
    body instanceof Blob ||
    body instanceof ArrayBuffer ||
    body instanceof URLSearchParams ||
    typeof body === 'string'
  ) {
    return { body, hasJsonBody: false }
  }
  return { body: JSON.stringify(body), hasJsonBody: true }
}

function isGatewayErrorEnvelope(value: unknown): value is GatewayErrorEnvelope {
  return Boolean(
    value &&
    typeof value === 'object' &&
    'error' in value &&
    (value as { error?: unknown }).error &&
    typeof (value as { error: { message?: unknown } }).error.message === 'string',
  )
}

async function readJsonSafely(response: Response): Promise<unknown> {
  const text = await response.text()
  if (!text) return undefined
  try {
    return JSON.parse(text) as unknown
  } catch {
    return text
  }
}

async function toApiError(response: Response): Promise<ApiError> {
  const body = await readJsonSafely(response)
  if (isGatewayErrorEnvelope(body)) {
    return new ApiError({
      code: body.error.code,
      message: body.error.message,
      status: response.status,
      requestId: body.error.requestId ?? response.headers.get('X-Request-Id') ?? undefined,
      fields: body.error.fields,
    })
  }

  return new ApiError({
    code: response.status ? `http_${response.status}` : 'network_error',
    message: typeof body === 'string' && body ? body : response.statusText || 'Request failed',
    status: response.status,
    requestId: response.headers.get('X-Request-Id') ?? undefined,
  })
}

function assertActiveGatewayPath(path: GatewayPath): void {
  if (!activeGatewayPathSet.has(path)) {
    throw new Error(`Mock path is not an active gateway OpenAPI path: ${path}`)
  }
}

function matchMock(method: GatewayMethod, path: string): MockRoute | undefined {
  if (import.meta.env?.VITE_API_MOCKS !== 'true') return undefined
  return mockRoutes.find((route) => route.method === method && route.path === path)
}

async function fetchGateway(path: string, options: GatewayRequestOptions = {}): Promise<Response> {
  const method = options.method ?? 'GET'
  const mock = matchMock(method, path)
  const { body, hasJsonBody } = prepareBody(options.body)
  const headers = buildHeaders(options, hasJsonBody)
  const request = new Request(joinUrl(path), {
    ...options,
    method,
    headers,
    body,
  })

  if (mock) return mock.handler(request)
  return fetch(request)
}

export async function requestEnvelope<T>(
  path: string,
  options?: GatewayRequestOptions,
): Promise<GatewaySuccessEnvelope<T>> {
  const response = await fetchGateway(path, options)
  if (!response.ok) throw await toApiError(response)
  return (await response.json()) as GatewaySuccessEnvelope<T>
}

export async function requestPaginated<T>(
  path: string,
  options?: GatewayRequestOptions,
): Promise<GatewayPaginatedEnvelope<T>> {
  const response = await fetchGateway(path, options)
  if (!response.ok) throw await toApiError(response)
  return (await response.json()) as GatewayPaginatedEnvelope<T>
}

export async function requestJson<T>(path: string, options?: GatewayRequestOptions): Promise<T> {
  const envelope = await requestEnvelope<T>(path, options)
  return envelope.data
}

export async function requestVoid(path: string, options?: GatewayRequestOptions): Promise<void> {
  const response = await fetchGateway(path, options)
  if (!response.ok) throw await toApiError(response)
}

export async function requestBinary(path: string, options?: GatewayRequestOptions): Promise<Blob> {
  const response = await fetchGateway(path, {
    ...options,
    headers: {
      Accept: 'application/octet-stream',
      ...options?.headers,
    },
  })
  if (!response.ok) throw await toApiError(response)
  return response.blob()
}

export function buildQuery(
  params: Record<string, string | number | boolean | undefined | null>,
): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') {
      search.set(key, String(value))
    }
  }

  const query = search.toString()
  return query ? `?${query}` : ''
}

function withJsonHeaders(options: GatewayRequestOptions = {}): GatewayRequestOptions {
  const headers = new Headers(options.headers)
  if (options.body != null && !(options.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', JSON_CONTENT_TYPE)
  }
  return { ...options, headers }
}

export function gatewayRequest<T>(path: string, options?: GatewayRequestOptions): Promise<T> {
  return requestJson<T>(path, withJsonHeaders(options))
}

export async function gatewayPageRequest<T>(
  path: string,
  options?: GatewayRequestOptions,
): Promise<{ items: T[]; page: GatewayPage }> {
  const envelope = await requestPaginated<T>(path, withJsonHeaders(options))
  return { items: envelope.data, page: envelope.page }
}

export function gatewayFileRequest(path: string, options?: GatewayRequestOptions): Promise<Blob> {
  return requestBinary(path, {
    ...options,
    headers: {
      Accept:
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document, application/octet-stream',
      ...options?.headers,
    },
  })
}

export function streamGateway(
  path: string,
  options: GatewayStreamOptions,
): { abort: () => void; signal: AbortSignal } {
  const controller = new AbortController()
  const signal = mergeAbortSignals(controller.signal, options.signal)

  void (async () => {
    try {
      const response = await fetchGateway(path, {
        ...options,
        method: options.method ?? 'POST',
        signal,
        headers: {
          Accept: SSE_CONTENT_TYPE,
          ...options.headers,
        },
      })

      if (!response.ok) throw await toApiError(response)
      const contentType = response.headers.get('Content-Type') ?? ''
      if (!contentType.includes(SSE_CONTENT_TYPE)) {
        throw new ApiError({
          code: 'invalid_stream_response',
          message: 'Expected text/event-stream response',
          status: response.status,
          requestId: response.headers.get('X-Request-Id') ?? undefined,
        })
      }
      if (!response.body) {
        throw new ApiError({
          code: 'empty_stream_response',
          message: 'Response body is not readable',
          status: response.status,
          requestId: response.headers.get('X-Request-Id') ?? undefined,
        })
      }

      await readSseStream(response.body, options.onEvent, signal)
      options.onDone?.()
    } catch (error) {
      if (signal.aborted) return
      options.onError?.(
        error instanceof ApiError
          ? error
          : new ApiError({
              code: 'network_error',
              message: error instanceof Error ? error.message : 'Network error',
              status: 0,
            }),
      )
    }
  })()

  return { abort: () => controller.abort(), signal }
}

function mergeAbortSignals(primary: AbortSignal, secondary?: AbortSignal | null): AbortSignal {
  if (!secondary) return primary
  const controller = new AbortController()
  const abort = (signal: AbortSignal) => controller.abort(signal.reason)

  if (primary.aborted) {
    abort(primary)
    return controller.signal
  }
  if (secondary.aborted) {
    abort(secondary)
    return controller.signal
  }

  primary.addEventListener('abort', () => abort(primary), { once: true })
  secondary.addEventListener('abort', () => abort(secondary), { once: true })
  return controller.signal
}

async function readSseStream(
  body: ReadableStream<Uint8Array>,
  onEvent: (event: SseEvent) => void,
  signal: AbortSignal,
): Promise<void> {
  const reader = body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let eventName = 'message'
  let eventId: string | undefined
  let retry: number | undefined
  let dataLines: string[] = []

  const flush = () => {
    if (!dataLines.length) {
      eventName = 'message'
      eventId = undefined
      retry = undefined
      return
    }
    onEvent({
      event: eventName,
      data: dataLines.join('\n'),
      id: eventId,
      retry,
    })
    eventName = 'message'
    eventId = undefined
    retry = undefined
    dataLines = []
  }

  const processLine = (line: string) => {
    const normalized = line.endsWith('\r') ? line.slice(0, -1) : line
    if (normalized === '') {
      flush()
      return
    }
    if (normalized.startsWith(':')) return

    const colon = normalized.indexOf(':')
    const field = colon === -1 ? normalized : normalized.slice(0, colon)
    const value = colon === -1 ? '' : normalized.slice(colon + 1).replace(/^ /, '')

    switch (field) {
      case 'event':
        eventName = value || 'message'
        break
      case 'data':
        dataLines.push(value)
        break
      case 'id':
        eventId = value
        break
      case 'retry': {
        const parsed = Number(value)
        if (Number.isFinite(parsed)) retry = parsed
        break
      }
      default:
        break
    }
  }

  try {
    for (;;) {
      if (signal.aborted) throw new DOMException('Aborted', 'AbortError')
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() ?? ''
      for (const line of lines) processLine(line)
    }

    buffer += decoder.decode()
    if (buffer) processLine(buffer)
    flush()
  } finally {
    reader.releaseLock()
  }
}

// Compatibility alias for existing feature wrappers while they migrate.
export const doRequest = requestJson
