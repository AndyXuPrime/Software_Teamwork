/**
 * Session (conversation) CRUD — API doc section 2.
 *
 * Based on frontend/src/api/chat.ts session section.
 */

import type { Conversation, ConversationListItem } from '@/lib/types'

import { apiClient,ApiError } from './client'

// ---------------------------------------------------------------------------
// 2.1  Create
// ---------------------------------------------------------------------------

export async function createConversation(
  title = '新对话',
): Promise<Conversation> {
  const res = await fetch(`${apiClient.baseUrl}/conversations`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title }),
  })
  if (!res.ok) throw new ApiError(res.status, '创建会话失败')
  const json: { code: number; message: string; data: Conversation } =
    await res.json()
  if (json.code !== 0) throw new ApiError(json.code, json.message)
  return json.data
}

// ---------------------------------------------------------------------------
// 2.2  List (paginated)
// ---------------------------------------------------------------------------

export async function listConversations(
  page = 1,
  pageSize = 20,
): Promise<{ items: ConversationListItem[]; total: number }> {
  const params = new URLSearchParams({
    page: String(page),
    page_size: String(pageSize),
    sort: 'updated_at_desc',
  })
  const res = await fetch(
    `${apiClient.baseUrl}/conversations?${params}`,
  )
  if (!res.ok) throw new ApiError(res.status, '获取会话列表失败')
  const json: {
    code: number
    message: string
    data: { items: ConversationListItem[]; total: number }
  } = await res.json()
  if (json.code !== 0) throw new ApiError(json.code, json.message)
  return json.data
}

// ---------------------------------------------------------------------------
// 2.3  Get detail (with messages)
// ---------------------------------------------------------------------------

export async function getConversation(
  id: string,
): Promise<Conversation> {
  const res = await fetch(
    `${apiClient.baseUrl}/conversations/${encodeURIComponent(id)}`,
  )
  if (!res.ok) throw new ApiError(res.status, '获取会话详情失败')
  const json: { code: number; message: string; data: Conversation } =
    await res.json()
  if (json.code !== 0) throw new ApiError(json.code, json.message)
  return json.data
}

// ---------------------------------------------------------------------------
// 2.4  Update (rename) — present in API doc, not in source
// ---------------------------------------------------------------------------

export async function updateConversation(
  id: string,
  title: string,
): Promise<Conversation> {
  const res = await fetch(
    `${apiClient.baseUrl}/conversations/${encodeURIComponent(id)}`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title }),
    },
  )
  if (!res.ok) throw new ApiError(res.status, '更新会话失败')
  const json: { code: number; message: string; data: Conversation } =
    await res.json()
  if (json.code !== 0) throw new ApiError(json.code, json.message)
  return json.data
}

// ---------------------------------------------------------------------------
// 2.5  Delete
// ---------------------------------------------------------------------------

export async function deleteConversation(id: string): Promise<void> {
  const res = await fetch(
    `${apiClient.baseUrl}/conversations/${encodeURIComponent(id)}`,
    { method: 'DELETE' },
  )
  if (!res.ok) throw new ApiError(res.status, '删除会话失败')
  const json: { code: number; message: string } = await res.json()
  if (json.code !== 0) throw new ApiError(json.code, json.message)
}
