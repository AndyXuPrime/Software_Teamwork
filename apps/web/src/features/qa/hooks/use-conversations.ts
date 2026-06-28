/**
 * React Query hooks for conversation CRUD.
 *
 * Server state managed by TanStack Query; UI cache synchronised via
 * the Zustand chat store on mutation success.
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import {
  createConversation,
  deleteConversation,
  getConversation,
  listConversations,
} from '@/api/conversations'

// ── Query keys ──

export const conversationKeys = {
  all: ['conversations'] as const,
  lists: () => [...conversationKeys.all, 'list'] as const,
  list: (page: number, pageSize: number) =>
    [...conversationKeys.lists(), { page, pageSize }] as const,
  details: () => [...conversationKeys.all, 'detail'] as const,
  detail: (id: string) => [...conversationKeys.details(), id] as const,
}

// ── Queries ──

/** Paginated conversation list. */
export function useConversations(page = 1, pageSize = 20) {
  return useQuery({
    queryKey: conversationKeys.list(page, pageSize),
    queryFn: () => listConversations(page, pageSize),
    placeholderData: (prev) => prev,
  })
}

/** Single conversation detail (includes messages). */
export function useConversation(id: string) {
  return useQuery({
    queryKey: conversationKeys.detail(id),
    queryFn: () => getConversation(id),
    enabled: id.length > 0,
  })
}

// ── Mutations ──

/** Create a new conversation. */
export function useCreateConversation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (title?: string) => createConversation(title),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: conversationKeys.lists(),
      })
    },
  })
}

/** Delete a conversation. */
export function useDeleteConversation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => deleteConversation(id),
    onSuccess: (_data, id) => {
      void queryClient.invalidateQueries({
        queryKey: conversationKeys.lists(),
      })
      queryClient.removeQueries({
        queryKey: conversationKeys.detail(id),
      })
    },
  })
}
