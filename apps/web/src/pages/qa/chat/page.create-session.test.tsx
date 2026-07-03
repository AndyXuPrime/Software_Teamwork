import { fireEvent, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createJSONStorage } from 'zustand/middleware'

import type { QAMessage, QASession, SessionAttachmentSummary } from '@/lib/types'
import { useChatStore } from '@/stores/chat-store'
import { renderWithProviders } from '@/test/render'

import { ChatPage } from './page'

type MockSidebarSession = {
  id: string
  messageCount?: number
  title?: string
}

type MockSidebarProps = {
  activeId: string
  onCreate: () => void
  sessions: MockSidebarSession[]
}

const qaHookState = vi.hoisted(() => ({
  createPending: false,
  createSession: vi.fn(),
  deleteSession: vi.fn(),
  refetchSessions: vi.fn(),
  renameSession: vi.fn(),
  sessionsData: { items: [] as QASession[] },
  uploadSessionId: null as string | null,
  uploadState: { phase: 'idle' } as
    | { phase: 'idle' }
    | { filename: string; phase: 'uploading' }
    | { attachment: SessionAttachmentSummary; attempts: number; phase: 'polling' },
}))

vi.mock('@/api/chat', () => ({
  replayEvents: vi.fn(),
  streamChat: vi.fn(),
}))

vi.mock('@/api/conversations', () => ({
  deleteSessionAttachment: vi.fn(),
  getSessionAttachment: vi.fn(),
  listSessionAttachments: vi.fn(async () => ({ items: [] })),
}))

vi.mock('@/components/chat', () => ({
  AttachmentList: () => null,
  AttachmentUploadStatus: () => null,
  ChatInput: () => null,
  ChatMessages: () => null,
  ChatSidebar: ({ activeId, onCreate, sessions }: MockSidebarProps) => (
    <aside>
      <button type="button" onClick={onCreate}>
        新建对话
      </button>
      <div data-testid="active-session">{activeId}</div>
      <div data-testid="session-count">{sessions.length}</div>
      {sessions.map((session) => (
        <div data-testid="session-row" data-active={session.id === activeId} key={session.id}>
          {session.title ?? '新对话'}:{session.messageCount ?? 0}
        </div>
      ))}
    </aside>
  ),
  useAttachmentUpload: () => ({
    dismissUpload: () => undefined,
    uploadFile: () => undefined,
    uploadSessionId: qaHookState.uploadSessionId,
    uploadState: qaHookState.uploadState,
  }),
}))

vi.mock('@/components/common', () => ({
  ConfirmDialog: () => null,
}))

vi.mock('@/features/qa', () => ({
  useCreateSession: () => ({
    isPending: qaHookState.createPending,
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

const now = '2026-07-03T00:00:00.000Z'

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
    content: '已有消息',
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
    status: 'parsing',
    createdAt: now,
    ...overrides,
  }
}

function setChatState(state: {
  activeId: string | null
  messagesBySession?: Record<string, QAMessage[]>
  sessions: QASession[]
  streaming?: boolean
}) {
  useChatStore.setState({
    activeId: state.activeId,
    attachmentsBySession: {},
    error: null,
    excludedAttachmentIds: {},
    lastFailedMsg: null,
    messagesBySession: state.messagesBySession ?? {},
    sessionIds: state.sessions.map((session) => session.id),
    sessions: state.sessions,
    streaming: state.streaming ?? false,
  })
}

describe('ChatPage create conversation action', () => {
  beforeEach(() => {
    useChatStore.persist.setOptions({
      storage: createJSONStorage(() => window.localStorage),
    })
    qaHookState.createPending = false
    qaHookState.createSession.mockReset()
    qaHookState.deleteSession.mockReset()
    qaHookState.refetchSessions.mockReset()
    qaHookState.renameSession.mockReset()
    qaHookState.sessionsData = { items: [] }
    qaHookState.uploadSessionId = null
    qaHookState.uploadState = { phase: 'idle' }
    setChatState({ activeId: null, sessions: [] })
  })

  it('reuses the active empty new conversation without calling create session', () => {
    const session = makeSession()
    qaHookState.sessionsData = { items: [session] }
    setChatState({ activeId: session.id, sessions: [session] })

    renderWithProviders(<ChatPage />)

    fireEvent.click(screen.getByRole('button', { name: '新建对话' }))
    fireEvent.click(screen.getByRole('button', { name: '新建对话' }))

    expect(qaHookState.createSession).not.toHaveBeenCalled()
    expect(screen.getByTestId('session-count')).toHaveTextContent('1')
    expect(screen.getByTestId('active-session')).toHaveTextContent(session.id)
  })

  it('reuses the selected empty conversation while another conversation is streaming', () => {
    const emptySession = makeSession({ id: 'empty-session' })
    const streamingSession = makeSession({
      id: 'streaming-session',
      messageCount: 2,
      title: '你好',
    })
    qaHookState.sessionsData = { items: [emptySession, streamingSession] }
    setChatState({
      activeId: emptySession.id,
      messagesBySession: {
        [streamingSession.id]: [
          makeMessage({ id: 'streaming-user', sessionId: streamingSession.id }),
          makeMessage({
            content: '',
            id: 'streaming-assistant',
            role: 'assistant',
            sessionId: streamingSession.id,
            status: 'streaming',
          }),
        ],
      },
      sessions: [emptySession, streamingSession],
      streaming: true,
    })

    renderWithProviders(<ChatPage />)

    fireEvent.click(screen.getByRole('button', { name: '新建对话' }))
    fireEvent.click(screen.getByRole('button', { name: '新建对话' }))

    expect(qaHookState.createSession).not.toHaveBeenCalled()
    expect(screen.getByTestId('session-count')).toHaveTextContent('2')
    expect(screen.getByTestId('active-session')).toHaveTextContent(emptySession.id)
  })

  it('reuses the selected empty conversation while another conversation is uploading', () => {
    const emptySession = makeSession({ id: 'empty-session' })
    const uploadingSession = makeSession({
      id: 'uploading-session',
      title: '附件会话',
    })
    qaHookState.sessionsData = { items: [emptySession, uploadingSession] }
    qaHookState.uploadSessionId = uploadingSession.id
    qaHookState.uploadState = { filename: 'guide.pdf', phase: 'uploading' }
    setChatState({
      activeId: emptySession.id,
      sessions: [emptySession, uploadingSession],
    })

    renderWithProviders(<ChatPage />)

    fireEvent.click(screen.getByRole('button', { name: '新建对话' }))
    fireEvent.click(screen.getByRole('button', { name: '新建对话' }))

    expect(qaHookState.createSession).not.toHaveBeenCalled()
    expect(screen.getByTestId('session-count')).toHaveTextContent('2')
    expect(screen.getByTestId('active-session')).toHaveTextContent(emptySession.id)
  })

  it('reuses the selected empty conversation while another conversation is polling upload', () => {
    const emptySession = makeSession({ id: 'empty-session' })
    const pollingSession = makeSession({
      id: 'polling-session',
      title: '附件会话',
    })
    qaHookState.sessionsData = { items: [emptySession, pollingSession] }
    qaHookState.uploadSessionId = pollingSession.id
    qaHookState.uploadState = {
      attachment: makeAttachment({ sessionId: pollingSession.id }),
      attempts: 1,
      phase: 'polling',
    }
    setChatState({
      activeId: emptySession.id,
      sessions: [emptySession, pollingSession],
    })

    renderWithProviders(<ChatPage />)

    fireEvent.click(screen.getByRole('button', { name: '新建对话' }))
    fireEvent.click(screen.getByRole('button', { name: '新建对话' }))

    expect(qaHookState.createSession).not.toHaveBeenCalled()
    expect(screen.getByTestId('session-count')).toHaveTextContent('2')
    expect(screen.getByTestId('active-session')).toHaveTextContent(emptySession.id)
  })

  it('creates and selects a new conversation when the active conversation has messages', async () => {
    const current = makeSession({ messageCount: 1 })
    const next = makeSession({ id: 'session-2' })
    qaHookState.sessionsData = { items: [current] }
    qaHookState.createSession.mockResolvedValueOnce(next)
    setChatState({
      activeId: current.id,
      messagesBySession: { [current.id]: [makeMessage()] },
      sessions: [current],
    })

    renderWithProviders(<ChatPage />)

    fireEvent.click(screen.getByRole('button', { name: '新建对话' }))

    await waitFor(() => expect(qaHookState.createSession).toHaveBeenCalledWith('新对话'))
    await waitFor(() => expect(screen.getByTestId('active-session')).toHaveTextContent(next.id))
    expect(screen.getAllByTestId('session-row')).toHaveLength(2)
  })

  it('creates and selects a new conversation when there is no active conversation', async () => {
    const next = makeSession({ id: 'session-1' })
    qaHookState.createSession.mockResolvedValueOnce(next)
    setChatState({ activeId: null, sessions: [] })

    renderWithProviders(<ChatPage />)

    fireEvent.click(screen.getByRole('button', { name: '新建对话' }))

    await waitFor(() => expect(qaHookState.createSession).toHaveBeenCalledWith('新对话'))
    await waitFor(() => expect(screen.getByTestId('active-session')).toHaveTextContent(next.id))
    expect(screen.getByTestId('session-count')).toHaveTextContent('1')
  })
})
