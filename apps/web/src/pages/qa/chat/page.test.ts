import { describe, expect, it } from 'vitest'

import type { UploadStateData } from '@/components/chat'
import type { QAMessage, QASession, SessionAttachmentSummary } from '@/lib/types'

import {
  finalizeThinkingStepsOnAnswerCompleted,
  isReusableEmptyNewSession,
  sanitizeCitation,
  sanitizeThinkingStep,
  type ToolThinkingStep,
  upsertReasoningStep,
} from './page'

const now = '2026-07-03T00:00:00.000Z'
const idleUploadState: UploadStateData = { phase: 'idle' }

function makeSession(overrides: Partial<QASession> = {}): QASession {
  return {
    id: 'session-1',
    title: '新对话',
    status: 'active',
    messageCount: 0,
    lastMessagePreview: '',
    createdAt: now,
    updatedAt: now,
    ...overrides,
  }
}

function makeMessage(overrides: Partial<QAMessage> = {}): QAMessage {
  return {
    id: 'message-1',
    sessionId: 'session-1',
    role: 'user',
    status: 'completed',
    content: '已有问题',
    createdAt: now,
    ...overrides,
  }
}

function makeAttachment(
  overrides: Partial<SessionAttachmentSummary> = {},
): SessionAttachmentSummary {
  return {
    id: 'attachment-1',
    sessionId: 'session-1',
    filename: 'guide.pdf',
    contentType: 'application/pdf',
    sizeBytes: 128,
    status: 'ready',
    createdAt: now,
    ...overrides,
  }
}

function reusableParams(
  overrides: Partial<Parameters<typeof isReusableEmptyNewSession>[0]> = {},
): Parameters<typeof isReusableEmptyNewSession>[0] {
  return {
    activeId: 'session-1',
    attachmentsBySession: {},
    inputText: '',
    messagesBySession: {},
    sessions: [makeSession()],
    uploadSessionId: null,
    uploadState: idleUploadState,
    ...overrides,
  }
}

describe('isReusableEmptyNewSession', () => {
  it('allows reusing the active empty default conversation', () => {
    expect(isReusableEmptyNewSession(reusableParams())).toBe(true)
  })

  it('rejects missing or renamed active conversations', () => {
    expect(isReusableEmptyNewSession(reusableParams({ activeId: null }))).toBe(false)
    expect(
      isReusableEmptyNewSession(
        reusableParams({
          sessions: [makeSession({ title: '变压器巡检' })],
        }),
      ),
    ).toBe(false)
  })

  it('rejects conversations with server or local messages', () => {
    expect(
      isReusableEmptyNewSession(
        reusableParams({
          sessions: [makeSession({ messageCount: 1 })],
        }),
      ),
    ).toBe(false)
    expect(
      isReusableEmptyNewSession(
        reusableParams({
          messagesBySession: { 'session-1': [makeMessage()] },
        }),
      ),
    ).toBe(false)
  })

  it('rejects visible attachments but ignores failed or purged attachments', () => {
    expect(
      isReusableEmptyNewSession(
        reusableParams({
          attachmentsBySession: { 'session-1': [makeAttachment()] },
        }),
      ),
    ).toBe(false)
    expect(
      isReusableEmptyNewSession(
        reusableParams({
          attachmentsBySession: {
            'session-1': [
              makeAttachment({ id: 'failed-1', status: 'failed' }),
              makeAttachment({ id: 'purged-1', status: 'purged' }),
            ],
          },
        }),
      ),
    ).toBe(true)
  })

  it('rejects draft text, current-session upload progress, and active streaming messages', () => {
    expect(isReusableEmptyNewSession(reusableParams({ inputText: '未发送草稿' }))).toBe(false)
    expect(
      isReusableEmptyNewSession(
        reusableParams({
          uploadSessionId: 'session-1',
          uploadState: { phase: 'uploading', filename: 'guide.pdf' },
        }),
      ),
    ).toBe(false)
    expect(
      isReusableEmptyNewSession(
        reusableParams({
          messagesBySession: {
            'session-1': [
              makeMessage({
                content: '',
                id: 'streaming-assistant',
                role: 'assistant',
                status: 'streaming',
              }),
            ],
          },
        }),
      ),
    ).toBe(false)
  })

  it('allows reuse while another session owns upload progress', () => {
    expect(
      isReusableEmptyNewSession(
        reusableParams({
          uploadSessionId: 'session-2',
          uploadState: { phase: 'uploading', filename: 'guide.pdf' },
        }),
      ),
    ).toBe(true)
    expect(
      isReusableEmptyNewSession(
        reusableParams({
          uploadState: {
            attachment: makeAttachment({ sessionId: 'session-2', status: 'parsing' }),
            attempts: 0,
            phase: 'polling',
          },
        }),
      ),
    ).toBe(true)
  })

  it('rejects current-session polling progress', () => {
    expect(
      isReusableEmptyNewSession(
        reusableParams({
          uploadState: {
            attachment: makeAttachment({ sessionId: 'session-1', status: 'parsing' }),
            attempts: 0,
            phase: 'polling',
          },
        }),
      ),
    ).toBe(false)
  })
})

describe('ChatPage stream thinking helpers', () => {
  it('keeps same-type reasoning steps separated across iterations', () => {
    const first = sanitizeThinkingStep({
      detail: '第 1 轮生成',
      iterationNo: 1,
      label: '生成回答',
      status: 'done',
      type: 'generation',
    })
    const second = sanitizeThinkingStep({
      detail: '第 2 轮生成',
      iterationNo: 2,
      label: '生成回答',
      status: 'done',
      type: 'generation',
    })

    const steps = upsertReasoningStep(upsertReasoningStep([], first), second)

    expect(steps).toHaveLength(2)
    expect(steps.map((step) => step.iterationNo)).toEqual([1, 2])
    expect(steps.map((step) => step.detail)).toEqual(['第 1 轮生成', '第 2 轮生成'])
  })

  it('updates only a matching stable reasoning step in the same iteration', () => {
    const started = sanitizeThinkingStep({
      id: 'reason-1',
      iterationNo: 1,
      label: '校验答案',
      status: 'running',
      type: 'verify',
    })
    const completed = sanitizeThinkingStep({
      id: 'reason-1',
      iterationNo: 1,
      label: '校验答案',
      status: 'done',
      type: 'verify',
    })
    const nextIteration = sanitizeThinkingStep({
      id: 'reason-1',
      iterationNo: 2,
      label: '校验答案',
      status: 'running',
      type: 'verify',
    })

    const updated = upsertReasoningStep(upsertReasoningStep([], started), completed)
    const withNextIteration = upsertReasoningStep(updated, nextIteration)

    expect(updated).toHaveLength(1)
    expect(updated[0]).toMatchObject({ iterationNo: 1, status: 'done' })
    expect(withNextIteration).toHaveLength(2)
    expect(withNextIteration[1]).toMatchObject({ iterationNo: 2, status: 'running' })
  })

  it('does not mark running tool calls as done when the answer completes', () => {
    const steps: ToolThinkingStep[] = [
      {
        iterationNo: 1,
        label: 'Agent 迭代 1',
        status: 'running',
        type: 'agent_iteration',
      },
      {
        iterationNo: 1,
        label: '调用: report_generator',
        status: 'running',
        toolCallId: 'tool-1',
        type: 'tool_call',
      },
    ]

    const finalized = finalizeThinkingStepsOnAnswerCompleted(steps)

    expect(finalized[0]).toMatchObject({ status: 'done', type: 'agent_iteration' })
    expect(finalized[1]).toMatchObject({ status: 'running', type: 'tool_call' })
  })
})

describe('QA chat citation sanitizing', () => {
  it('normalizes legacy citation document aliases', () => {
    const citation = sanitizeCitation({
      citationNo: 1,
      docId: 'doc-legacy',
      docName: 'Legacy Manual.pdf',
      id: 'cite-1',
      isSourceAvailable: true,
      messageId: 'msg-1',
      score: 0.91,
      text: 'quoted text',
    })

    expect(citation.documentId).toBe('doc-legacy')
    expect(citation.documentName).toBe('Legacy Manual.pdf')
    expect(citation.docId).toBe('doc-legacy')
    expect(citation.docName).toBe('Legacy Manual.pdf')
    expect(citation.isSourceAvailable).toBe(true)
  })

  it('keeps canonical citation document fields preferred over aliases', () => {
    const citation = sanitizeCitation({
      docId: 'doc-legacy',
      docName: 'Legacy Manual.pdf',
      documentId: 'doc-canonical',
      documentName: 'Canonical Manual.pdf',
      id: 'cite-1',
      messageId: 'msg-1',
    })

    expect(citation.documentId).toBe('doc-canonical')
    expect(citation.documentName).toBe('Canonical Manual.pdf')
    expect(citation.docId).toBe('doc-legacy')
    expect(citation.docName).toBe('Legacy Manual.pdf')
  })
})
