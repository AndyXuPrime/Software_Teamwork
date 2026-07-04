import { act, fireEvent, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createJSONStorage } from 'zustand/middleware'

import type { ChatStreamHandlers } from '@/api/chat'
import { streamChat } from '@/api/chat'
import type { QAMessage, QASession } from '@/lib/types'
import { useChatStore } from '@/stores/chat-store'
import { renderWithProviders } from '@/test/render'

import { ChatPage } from './page'

type ChatInputProps = {
  onSend: (text: string) => void | Promise<void>
  onStop?: () => void
  streaming?: boolean
}

type KnowledgeBaseMultiSelectProps = {
  onChange: (ids: string[]) => void
  value: string[]
}

const qaHookState = vi.hoisted(() => ({
  createSession: vi.fn(),
  deleteSession: vi.fn(),
  refetchSessions: vi.fn(),
  renameSession: vi.fn(),
  sessionsData: { items: [] as QASession[] },
}))

vi.mock('@/api/chat', () => ({
  replayEvents: vi.fn(),
  streamChat: vi.fn(),
}))

vi.mock('@/api/conversations', () => ({
  deleteSessionAttachment: vi.fn(),
  getSessionAttachment: vi.fn(),
  listSessions: vi.fn(async () => ({ items: [], page: { total: 0 } })),
  listSessionAttachments: vi.fn(async () => ({ items: [] })),
}))

vi.mock('@/components/chat', () => ({
  AttachmentList: () => null,
  AttachmentUploadStatus: () => null,
  ChatInput: ({ onSend, onStop, streaming = false }: ChatInputProps) => (
    <div>
      <button type="button" onClick={() => void onSend('background question')}>
        send
      </button>
      {streaming && (
        <button type="button" onClick={onStop}>
          stop
        </button>
      )}
    </div>
  ),
  ChatMessages: ({ messages }: { messages: QAMessage[] }) => (
    <div data-testid="messages">
      {messages.map((message) => (
        <div data-status={message.status} key={message.id}>
          {message.content}
        </div>
      ))}
    </div>
  ),
  ChatSidebar: () => <aside />,
  useAttachmentUpload: () => ({
    dismissUpload: () => undefined,
    uploadFile: () => undefined,
    uploadSessionId: null,
    uploadState: { phase: 'idle' },
  }),
}))

vi.mock('@/components/common', () => ({
  ConfirmDialog: () => null,
}))

vi.mock('@/features/knowledge', () => ({
  KnowledgeBaseMultiSelect: ({ onChange, value }: KnowledgeBaseMultiSelectProps) => (
    <div>
      <button type="button" onClick={() => onChange(['kb-session-a'])}>
        select knowledge base
      </button>
      <span data-testid="selected-knowledge-bases">
        {value.length > 0 ? value.join(',') : 'default'}
      </span>
    </div>
  ),
}))

vi.mock('@/features/qa', () => ({
  useCreateSession: () => ({
    isPending: false,
    mutateAsync: qaHookState.createSession,
  }),
  useDeleteSession: () => ({
    mutateAsync: qaHookState.deleteSession,
  }),
  useRenameSession: () => ({
    mutateAsync: qaHookState.renameSession,
  }),
  useSessionMessages: () => ({
    data: { items: [] },
    isError: false,
  }),
  useSessions: () => ({
    data: qaHookState.sessionsData,
    isError: false,
    isLoading: false,
    refetch: qaHookState.refetchSessions,
  }),
}))

const mockedStreamChat = vi.mocked(streamChat)
const now = '2026-07-04T00:00:00.000Z'

function makeSession(overrides: Partial<QASession> = {}): QASession {
  return {
    id: 'session-1',
    title: 'Background stream',
    status: 'active',
    messageCount: 0,
    lastMessagePreview: '',
    createdAt: now,
    updatedAt: now,
    ...overrides,
  }
}

function seedChatState(session = makeSession(), sessions = [session]) {
  useChatStore.setState({
    activeId: session.id,
    activeStream: null,
    attachmentsBySession: {},
    error: null,
    excludedAttachmentIds: {},
    lastFailedMsg: null,
    messagesBySession: {},
    qaChatVisible: true,
    qaUnreadCompletion: null,
    sessionIds: sessions.map((item) => item.id),
    sessions,
    streaming: false,
  })
  qaHookState.sessionsData = { items: sessions }
}

describe('ChatPage background streaming lifecycle', () => {
  let abortSpy: ReturnType<typeof vi.fn<() => void>>
  let abortStream: () => void
  let handlers: ChatStreamHandlers | null

  beforeEach(() => {
    useChatStore.persist.setOptions({
      storage: createJSONStorage(() => window.localStorage),
    })
    useChatStore.getState().reset()
    qaHookState.createSession.mockReset()
    qaHookState.deleteSession.mockReset()
    qaHookState.refetchSessions.mockReset()
    qaHookState.renameSession.mockReset()
    handlers = null
    abortSpy = vi.fn<() => void>()
    abortStream = () => {
      abortSpy()
      handlers?.onAbort?.()
    }
    mockedStreamChat.mockReset()
    mockedStreamChat.mockImplementation((_sessionId, _message, nextHandlers) => {
      handlers = nextHandlers
      return { abort: abortStream }
    })
    let frameNow = 0
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      frameNow += 40
      callback(frameNow)
      return 1
    })
    vi.stubGlobal('cancelAnimationFrame', vi.fn())
    seedChatState()
  })

  it('keeps streaming after the chat page unmounts and marks background completion unread', async () => {
    const view = renderWithProviders(<ChatPage />)

    fireEvent.click(screen.getByRole('button', { name: 'send' }))
    await waitFor(() => expect(mockedStreamChat).toHaveBeenCalledTimes(1))

    act(() => {
      useChatStore.getState().setQaChatVisible(false)
    })
    view.unmount()

    expect(abortSpy).not.toHaveBeenCalled()

    act(() => {
      handlers?.onAnswerDelta?.({ content: 'finished answer', seq: 1 })
      handlers?.onAnswerCompleted?.({ messageId: 'assistant-server', seq: 2 })
    })

    await waitFor(() => expect(useChatStore.getState().streaming).toBe(false))
    const state = useChatStore.getState()
    const messages = state.messagesBySession['session-1'] ?? []
    expect(messages.at(-1)).toMatchObject({
      content: 'finished answer',
      id: 'assistant-server',
      status: 'completed',
    })
    expect(state.qaUnreadCompletion).toMatchObject({
      messageId: 'assistant-server',
      sessionId: 'session-1',
    })
  })

  it('keeps the active stream stoppable after leaving and returning to the chat page', async () => {
    const firstView = renderWithProviders(<ChatPage />)

    fireEvent.click(screen.getByRole('button', { name: 'send' }))
    await waitFor(() => expect(useChatStore.getState().streaming).toBe(true))

    act(() => {
      useChatStore.getState().setQaChatVisible(false)
    })
    firstView.unmount()
    expect(abortSpy).not.toHaveBeenCalled()

    act(() => {
      useChatStore.getState().setQaChatVisible(true)
    })
    renderWithProviders(<ChatPage />)

    fireEvent.click(screen.getByRole('button', { name: 'stop' }))

    expect(abortSpy).toHaveBeenCalledTimes(1)
    await waitFor(() => expect(useChatStore.getState().streaming).toBe(false))
    expect(useChatStore.getState().messagesBySession['session-1']?.at(-1)).toMatchObject({
      status: 'stopped',
    })
  })

  it('does not reuse selected knowledge bases after switching sessions', async () => {
    const firstSession = makeSession({ id: 'session-1', title: 'Session one' })
    const secondSession = makeSession({ id: 'session-2', title: 'Session two' })
    seedChatState(firstSession, [firstSession, secondSession])

    renderWithProviders(<ChatPage />)

    fireEvent.click(screen.getByRole('button', { name: 'select knowledge base' }))
    expect(screen.getByTestId('selected-knowledge-bases')).toHaveTextContent('kb-session-a')

    act(() => {
      useChatStore.getState().setActiveId(secondSession.id)
    })

    await waitFor(() =>
      expect(screen.getByTestId('selected-knowledge-bases')).toHaveTextContent('default'),
    )

    fireEvent.click(screen.getByRole('button', { name: 'send' }))

    await waitFor(() => expect(mockedStreamChat).toHaveBeenCalledTimes(1))
    expect(mockedStreamChat.mock.calls[0]?.[0]).toBe(secondSession.id)
    expect(mockedStreamChat.mock.calls[0]?.[5]).toBeUndefined()
  })
})
