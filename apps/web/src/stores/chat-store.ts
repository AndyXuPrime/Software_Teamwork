/**
 * Chat UI state — conversations cache, streaming flag, error tracking.
 *
 * Full conversation objects live in memory only (they come from the server).
 * Only conversation IDs are persisted to localStorage so the sidebar can
 * restore the session list across page reloads.
 */

import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import type { Conversation, Message } from '@/lib/types'

export interface ChatState {
  /** Full conversation objects (in-memory, fetched from server). */
  conversations: Conversation[]
  /** Conversation IDs persisted to localStorage for session recovery. */
  conversationIds: string[]
  /** Currently selected conversation. */
  activeId: string | null
  /** Whether an SSE stream is in progress. */
  streaming: boolean
  /** Last fatal error message for display. */
  error: string | null
  /** The user message that triggered a fatal error (for retry). */
  lastFailedMsg: string | null

  // ── Actions ──

  setConversations: (conversations: Conversation[]) => void
  setConversationIds: (ids: string[]) => void
  setActiveId: (id: string | null) => void
  addConversation: (conversation: Conversation) => void
  removeConversation: (id: string) => void
  updateConversationMessages: (id: string, messages: Message[]) => void
  setStreaming: (streaming: boolean) => void
  setError: (error: string | null) => void
  setLastFailedMsg: (msg: string | null) => void
  clearError: () => void
}

export const useChatStore = create<ChatState>()(
  persist(
    (set) => ({
      conversations: [],
      conversationIds: [],
      activeId: null,
      streaming: false,
      error: null,
      lastFailedMsg: null,

      setConversations: (conversations) => set({ conversations }),

      setConversationIds: (ids) => set({ conversationIds: ids }),

      setActiveId: (id) => set({ activeId: id }),

      addConversation: (conversation) =>
        set((state) => {
          if (state.conversations.some((c) => c.id === conversation.id)) {
            return state
          }
          return {
            conversations: [conversation, ...state.conversations],
            conversationIds: [
              conversation.id,
              ...state.conversationIds.filter((cid) => cid !== conversation.id),
            ],
          }
        }),

      removeConversation: (id) =>
        set((state) => ({
          conversations: state.conversations.filter((c) => c.id !== id),
          conversationIds: state.conversationIds.filter((cid) => cid !== id),
          activeId: state.activeId === id ? null : state.activeId,
        })),

      updateConversationMessages: (id, messages) =>
        set((state) => ({
          conversations: state.conversations.map((c) =>
            c.id === id ? { ...c, messages } : c,
          ),
        })),

      setStreaming: (streaming) => set({ streaming }),

      setError: (error) => set({ error }),

      setLastFailedMsg: (msg) => set({ lastFailedMsg: msg }),

      clearError: () => set({ error: null, lastFailedMsg: null }),
    }),
    {
      name: 'qa-chat-store-conversations',
      partialize: (state) => ({ conversationIds: state.conversationIds }),
    },
  ),
)
