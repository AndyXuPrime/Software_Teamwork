import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { streamChat } from '@/api/chat'

import { useStreamChat } from './use-stream-chat'

vi.mock('@/api/chat', () => ({
  streamChat: vi.fn(),
}))

afterEach(() => {
  vi.clearAllMocks()
})

describe('useStreamChat', () => {
  it('does not abort an active stream just because the component unmounts', () => {
    const abort = vi.fn()
    vi.mocked(streamChat).mockReturnValue({ abort })

    const { result, unmount } = renderHook(() => useStreamChat({}))

    act(() => {
      result.current.sendMessage('session-1', 'hello')
    })
    unmount()

    expect(abort).not.toHaveBeenCalled()
  })

  it('still aborts when the user explicitly stops streaming', () => {
    const abort = vi.fn()
    vi.mocked(streamChat).mockReturnValue({ abort })

    const { result } = renderHook(() => useStreamChat({}))

    act(() => {
      result.current.sendMessage('session-1', 'hello')
    })
    act(() => {
      result.current.abort()
    })

    expect(abort).toHaveBeenCalledTimes(1)
  })
})
