import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { ChatStreamHandlers } from '@/api/chat'
import { streamChat } from '@/api/chat'

import { useStreamChat } from './use-stream-chat'

vi.mock('@/api/chat', () => ({
  streamChat: vi.fn(() => ({ abort: vi.fn() })),
}))

const mockedStreamChat = vi.mocked(streamChat)

afterEach(() => {
  vi.clearAllMocks()
})

describe('useStreamChat', () => {
  it('does not abort an active stream just because the component unmounts', () => {
    const abort = vi.fn()
    mockedStreamChat.mockReturnValue({ abort })

    const { result, unmount } = renderHook(() => useStreamChat({}))

    act(() => {
      result.current.sendMessage('session-1', 'hello')
    })
    unmount()

    expect(abort).not.toHaveBeenCalled()
  })

  it('still aborts when the user explicitly stops streaming', () => {
    const abort = vi.fn()
    mockedStreamChat.mockReturnValue({ abort })

    const { result } = renderHook(() => useStreamChat({}))

    act(() => {
      result.current.sendMessage('session-1', 'hello')
    })
    act(() => {
      result.current.abort()
    })

    expect(abort).toHaveBeenCalledTimes(1)
  })

  it('forwards reasoning delta events from streamChat', () => {
    const onReasoningDelta = vi.fn()
    const { result } = renderHook(() => useStreamChat({ onReasoningDelta }))

    act(() => {
      result.current.sendMessage('session-1', 'question')
    })

    const handlers = mockedStreamChat.mock.calls[0]?.[2] as ChatStreamHandlers
    const payload = { messageId: 'message-1', text: 'checked source', seq: 3 }

    act(() => {
      handlers.onReasoningDelta?.(payload)
    })

    expect(onReasoningDelta).toHaveBeenCalledWith(payload)
  })
})
