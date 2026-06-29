import type { Conversation, ConversationListItem, Message } from '@/lib/types'

import { requestJson, requestPaginated, requestVoid } from './client'
import type { components } from './generated/gateway'

type QASession = components['schemas']['QASession']
type QAMessage = components['schemas']['QAMessage']

type Citation = components['schemas']['QACitation']

type ThinkingStep = components['schemas']['QAThinkingStep']

function toMessage(message: QAMessage): Message {
  return {
    id: message.id,
    role: message.role,
    content: message.content,
    timestamp: message.createdAt,
    status:
      message.status === 'cancelled'
        ? 'stopped'
        : message.status === 'queued'
          ? 'streaming'
          : message.status,
    thinking: message.thinking?.map((step: ThinkingStep) => ({
      type:
        step.type === 'verify' ? 'verify' : step.type === 'generation' ? 'generation' : 'retrieval',
      label: step.label ?? '处理步骤',
      status: step.status === 'failed' ? 'pending' : step.status,
      detail: step.detail,
    })),
    citations: message.citations?.map((citation: Citation) => ({
      id: citation.id,
      doc_id: citation.documentId ?? citation.docId ?? '',
      doc_name: citation.documentName ?? citation.docName ?? '',
      chunk_id: citation.chunkId ?? '',
      text: citation.text ?? citation.contentPreview ?? '',
      score: citation.score ?? 0,
    })),
  }
}

function toConversation(session: QASession, messages: Message[] = []): Conversation {
  return {
    id: session.id,
    title: session.title ?? '新对话',
    messages,
    created_at: session.createdAt,
    updated_at: session.updatedAt,
  }
}

function toConversationListItem(session: QASession): ConversationListItem {
  return {
    id: session.id,
    title: session.title ?? '新对话',
    message_count: session.messageCount ?? 0,
    last_message_preview: session.lastMessagePreview ?? '',
    created_at: session.createdAt,
    updated_at: session.updatedAt,
  }
}

export async function createSession(title = '新对话'): Promise<Conversation> {
  const session = await requestJson<QASession>('/qa-sessions', {
    method: 'POST',
    body: { title },
  })
  return toConversation(session)
}

export async function listSessions(
  page = 1,
  pageSize = 20,
): Promise<{ items: ConversationListItem[]; total: number }> {
  const params = new URLSearchParams({
    page: String(page),
    pageSize: String(pageSize),
    sort: '-updatedAt',
  })
  const envelope = await requestPaginated<QASession>(`/qa-sessions?${params}`)
  return {
    items: envelope.data.map(toConversationListItem),
    total: envelope.page.total,
  }
}

export async function getSession(id: string): Promise<Conversation> {
  const [session, messages] = await Promise.all([
    requestJson<QASession>(`/qa-sessions/${encodeURIComponent(id)}`),
    getSessionMessages(id),
  ])
  return toConversation(session, messages)
}

export async function renameSession(sessionId: string, title: string): Promise<Conversation> {
  const session = await requestJson<QASession>(`/qa-sessions/${encodeURIComponent(sessionId)}`, {
    method: 'PATCH',
    body: { title },
  })
  return toConversation(session)
}

export async function deleteSession(id: string): Promise<void> {
  await requestVoid(`/qa-sessions/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

export async function getSessionMessages(sessionId: string): Promise<Message[]> {
  const envelope = await requestPaginated<QAMessage>(
    `/qa-sessions/${encodeURIComponent(sessionId)}/messages?page=1&pageSize=100&includeThinking=true&includeCitations=true`,
  )
  return envelope.data.map(toMessage)
}
