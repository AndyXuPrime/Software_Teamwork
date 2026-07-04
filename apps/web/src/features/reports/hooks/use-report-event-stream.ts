import {
  type Dispatch,
  type SetStateAction,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'

import { ApiError, type SseEvent, streamGateway } from '@/api/client'
import { StreamingTextController } from '@/lib/streaming-text'

import type { ReportEvent } from '../report-generation.types'

type ReportEventStreamStatus = 'idle' | 'connecting' | 'streaming' | 'done' | 'error'

type UseReportEventStreamOptions = {
  enabled?: boolean
  jobId?: string | null
  reportId: string | null
  resetKey?: string | null
}

type ReportEventStreamState = {
  error: ApiError | null
  outlineText: string
  sectionTextById: Record<string, string>
  status: ReportEventStreamStatus
}

type SectionDeltaMessage = {
  sectionId: string
  text: string
}

type ReadableTextTarget = {
  controller: StreamingTextController
  rawText: string
  readableText: string
}

type ReportStreamControllers = {
  cancel: () => void
  commitVisible: () => void
  feedOutline: (text: string) => void
  feedSection: (sectionId: string, text: string) => void
  finish: () => void
}

const emptyStreamState: ReportEventStreamState = {
  error: null,
  outlineText: '',
  sectionTextById: {},
  status: 'idle',
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === 'object' && !Array.isArray(value))
}

function parseReportEvent(data: string): ReportEvent | null {
  try {
    const parsed: unknown = JSON.parse(data)
    if (!isRecord(parsed)) return null
    if (typeof parsed.id !== 'string') return null
    if (typeof parsed.reportId !== 'string') return null
    if (typeof parsed.eventType !== 'string') return null
    if (typeof parsed.createdAt !== 'string') return null
    if ('jobId' in parsed && parsed.jobId != null && typeof parsed.jobId !== 'string') return null
    if ('message' in parsed && parsed.message != null && typeof parsed.message !== 'string') {
      return null
    }

    return {
      createdAt: parsed.createdAt,
      eventType: parsed.eventType,
      id: parsed.id,
      jobId: typeof parsed.jobId === 'string' ? parsed.jobId : undefined,
      message: typeof parsed.message === 'string' ? parsed.message : undefined,
      payload: isRecord(parsed.payload) ? parsed.payload : undefined,
      reportId: parsed.reportId,
    }
  } catch {
    return null
  }
}

function parseSectionDeltaMessage(message?: string): SectionDeltaMessage | null {
  if (!message) return null

  try {
    const parsed: unknown = JSON.parse(message)
    if (
      isRecord(parsed) &&
      typeof parsed.sectionId === 'string' &&
      typeof parsed.text === 'string'
    ) {
      return { sectionId: parsed.sectionId, text: parsed.text }
    }
  } catch {
    // Plain-text section deltas are accepted as a compatibility fallback.
  }

  return { sectionId: '__report__', text: message }
}

function isJsonLike(value: string): boolean {
  const trimmed = value.trimStart()
  return trimmed.startsWith('{') || trimmed.startsWith('[')
}

function parseJsonSafely(value: string): unknown | null {
  try {
    return JSON.parse(value) as unknown
  } catch {
    return null
  }
}

function parseStringFields(value: unknown): Record<string, string> | undefined {
  if (!isRecord(value)) return undefined

  const fields = Object.entries(value).reduce<Record<string, string>>((result, [key, field]) => {
    if (typeof field === 'string') result[key] = field
    return result
  }, {})

  return Object.keys(fields).length > 0 ? fields : undefined
}

function parseReportStreamError(data: string): ApiError {
  const parsed = parseJsonSafely(data)
  const payload = isRecord(parsed) && isRecord(parsed.error) ? parsed.error : parsed
  const fallbackMessage = data.trim() || 'Report event stream failed'

  if (!isRecord(payload)) {
    return new ApiError({
      code: 'stream_error',
      message: fallbackMessage,
    })
  }

  const code =
    typeof payload.code === 'string' && payload.code.trim() ? payload.code.trim() : 'stream_error'
  const message =
    typeof payload.message === 'string' && payload.message.trim()
      ? payload.message.trim()
      : fallbackMessage
  const requestId =
    typeof payload.requestId === 'string' && payload.requestId.trim()
      ? payload.requestId.trim()
      : undefined

  return new ApiError({
    code,
    fields: parseStringFields(payload.fields),
    message,
    requestId,
  })
}

function decodeJsonStringAt(
  source: string,
  quoteIndex: number,
): { endIndex: number; value: string } {
  let value = ''
  let escaped = false

  for (let index = quoteIndex + 1; index < source.length; index += 1) {
    const char = source[index]
    if (char === undefined) break

    if (escaped) {
      escaped = false
      switch (char) {
        case '"':
        case '\\':
        case '/':
          value += char
          break
        case 'b':
          value += '\b'
          break
        case 'f':
          value += '\f'
          break
        case 'n':
          value += '\n'
          break
        case 'r':
          value += '\r'
          break
        case 't':
          value += '\t'
          break
        case 'u': {
          const hex = source.slice(index + 1, index + 5)
          if (/^[0-9a-fA-F]{4}$/.test(hex)) {
            value += String.fromCharCode(parseInt(hex, 16))
            index += 4
          }
          break
        }
        default:
          value += char
      }
      continue
    }

    if (char === '\\') {
      escaped = true
      continue
    }

    if (char === '"') {
      return { endIndex: index + 1, value }
    }

    value += char
  }

  return { endIndex: source.length, value }
}

function findJsonStringPropertyValues(source: string, propertyName: string): string[] {
  const values: string[] = []
  const needle = `"${propertyName}"`
  let index = 0

  while (index < source.length) {
    const propertyIndex = source.indexOf(needle, index)
    if (propertyIndex === -1) break

    let cursor = propertyIndex + needle.length
    while (/\s/.test(source[cursor] ?? '')) cursor += 1
    if (source[cursor] !== ':') {
      index = cursor
      continue
    }

    cursor += 1
    while (/\s/.test(source[cursor] ?? '')) cursor += 1
    if (source[cursor] !== '"') {
      index = cursor
      continue
    }

    const decoded = decodeJsonStringAt(source, cursor)
    const text = decoded.value.trim()
    if (text) values.push(text)
    index = Math.max(decoded.endIndex, cursor + 1)
  }

  return values
}

function collectTitleValues(value: unknown, result: string[] = []): string[] {
  if (Array.isArray(value)) {
    value.forEach((item) => collectTitleValues(item, result))
    return result
  }

  if (!isRecord(value)) return result

  if (typeof value.title === 'string' && value.title.trim()) {
    result.push(value.title.trim())
  }

  Object.values(value).forEach((item) => collectTitleValues(item, result))
  return result
}

function normalizeReadableLines(values: string[]): string {
  return values
    .map((value) => value.trim())
    .filter(Boolean)
    .join('\n')
}

function outlineDeltaToReadableText(rawText: string): string {
  if (!isJsonLike(rawText)) return rawText

  const parsed = parseJsonSafely(rawText)
  if (parsed !== null) {
    const titles = collectTitleValues(parsed)
    if (titles.length > 0) return normalizeReadableLines(titles)
  }

  return normalizeReadableLines(findJsonStringPropertyValues(rawText, 'title'))
}

function sectionDeltaToReadableText(rawText: string): string {
  if (!isJsonLike(rawText)) return rawText

  const parsed = parseJsonSafely(rawText)
  if (isRecord(parsed)) {
    return typeof parsed.content === 'string' ? parsed.content.trim() : ''
  }

  const content = findJsonStringPropertyValues(rawText, 'content')[0]
  if (content) return content

  const text = findJsonStringPropertyValues(rawText, 'text')[0]
  return text ?? ''
}

function feedReadableTarget(
  target: ReadableTextTarget,
  incomingText: string,
  toReadableText: (rawText: string) => string,
): void {
  if (!incomingText) return

  target.rawText += incomingText
  const nextReadableText = toReadableText(target.rawText)
  if (nextReadableText === target.readableText) return

  const textToType = nextReadableText.startsWith(target.readableText)
    ? nextReadableText.slice(target.readableText.length)
    : nextReadableText

  target.readableText = nextReadableText
  target.controller.feed(textToType)
}

function createReportStreamControllers(
  canUpdate: () => boolean,
  setState: Dispatch<SetStateAction<ReportEventStreamState>>,
): ReportStreamControllers {
  const makeTarget = (onUpdate: (text: string) => void): ReadableTextTarget => ({
    controller: new StreamingTextController({
      onUpdate: (text) => {
        if (canUpdate()) onUpdate(text)
      },
    }),
    rawText: '',
    readableText: '',
  })

  const outline = makeTarget((text) => {
    setState((prev) => ({ ...prev, outlineText: text, status: 'streaming' }))
  })
  const sections = new Map<string, ReadableTextTarget>()

  const getSectionTarget = (sectionId: string) => {
    let target = sections.get(sectionId)
    if (!target) {
      target = makeTarget((text) => {
        setState((prev) => ({
          ...prev,
          sectionTextById: {
            ...prev.sectionTextById,
            [sectionId]: text,
          },
          status: 'streaming',
        }))
      })
      sections.set(sectionId, target)
    }
    return target
  }

  return {
    cancel: () => {
      outline.controller.cancel()
      sections.forEach((target) => target.controller.cancel())
    },
    commitVisible: () => {
      setState((prev) => {
        const sectionTextById = { ...prev.sectionTextById }
        sections.forEach((target, sectionId) => {
          sectionTextById[sectionId] = target.readableText
        })
        return {
          ...prev,
          outlineText: outline.readableText,
          sectionTextById,
        }
      })
    },
    feedOutline: (text) => {
      feedReadableTarget(outline, text, outlineDeltaToReadableText)
    },
    feedSection: (sectionId, text) => {
      feedReadableTarget(getSectionTarget(sectionId), text, sectionDeltaToReadableText)
    },
    finish: () => {
      outline.controller.finish()
      sections.forEach((target) => target.controller.finish())
    },
  }
}

export function useReportEventStream({
  enabled = true,
  jobId,
  reportId,
  resetKey,
}: UseReportEventStreamOptions): ReportEventStreamState & {
  abort: () => void
  hasPreview: boolean
  reset: () => void
  sectionText: string
} {
  const [state, setState] = useState<ReportEventStreamState>(emptyStreamState)
  const abortRef = useRef<(() => void) | null>(null)
  const canceledRef = useRef(false)
  const controllersRef = useRef<ReportStreamControllers | null>(null)

  const reset = useCallback(() => {
    controllersRef.current?.cancel()
    controllersRef.current = null
    setState(emptyStreamState)
  }, [])

  const abort = useCallback(() => {
    controllersRef.current?.commitVisible()
    controllersRef.current?.cancel()
    controllersRef.current = null
    canceledRef.current = true
    abortRef.current?.()
    abortRef.current = null
    setState((prev) => ({ ...prev, status: 'idle' }))
  }, [])

  useEffect(() => {
    abortRef.current?.()
    abortRef.current = null
    controllersRef.current?.cancel()
    controllersRef.current = null
    canceledRef.current = false
    setState(emptyStreamState)

    if (!enabled || !reportId) return

    let active = true
    const controllers = createReportStreamControllers(
      () => active && !canceledRef.current,
      setState,
    )
    controllersRef.current = controllers
    setState({ ...emptyStreamState, status: 'connecting' })
    let streamFailed = false

    const streamPath = jobId
      ? `/reports/${encodeURIComponent(reportId)}/events/stream?jobId=${encodeURIComponent(jobId)}`
      : `/reports/${encodeURIComponent(reportId)}/events/stream`

    const { abort: stopStream } = streamGateway(streamPath, {
      method: 'GET',
      onDone: () => {
        if (!active || canceledRef.current || streamFailed) return
        controllers.finish()
        setState((prev) => ({ ...prev, status: 'done' }))
        abortRef.current = null
      },
      onError: (error) => {
        if (!active || canceledRef.current || streamFailed) return
        controllers.commitVisible()
        controllers.cancel()
        setState((prev) => ({ ...prev, error, status: 'error' }))
        abortRef.current = null
      },
      onEvent: (event: SseEvent) => {
        if (!active || canceledRef.current) return

        if (event.event === 'error') {
          streamFailed = true
          controllers.commitVisible()
          controllers.cancel()
          setState((prev) => ({
            ...prev,
            error: parseReportStreamError(event.data),
            status: 'error',
          }))
          abortRef.current?.()
          abortRef.current = null
          return
        }

        if (streamFailed) return
        if (event.event !== 'report.event') return
        const reportEvent = parseReportEvent(event.data)
        if (!reportEvent) return
        if (jobId && reportEvent.jobId && reportEvent.jobId !== jobId) return

        if (reportEvent.eventType === 'outline.delta') {
          controllers.feedOutline(reportEvent.message ?? '')
          return
        }

        if (reportEvent.eventType === 'section.delta') {
          const delta = parseSectionDeltaMessage(reportEvent.message)
          if (!delta) return
          controllers.feedSection(delta.sectionId, delta.text)
        }
      },
    })

    abortRef.current = stopStream
    return () => {
      active = false
      if (abortRef.current === stopStream) {
        abortRef.current = null
      }
      if (controllersRef.current === controllers) {
        controllersRef.current = null
      }
      controllers.cancel()
      stopStream()
    }
  }, [enabled, jobId, reportId, resetKey])

  const sectionText = useMemo(
    () => Object.values(state.sectionTextById).join(''),
    [state.sectionTextById],
  )
  const hasPreview = state.outlineText.length > 0 || sectionText.length > 0

  return { ...state, abort, hasPreview, reset, sectionText }
}
