import { Settings } from 'lucide-react'
import { useCallback, useMemo, useRef, useState } from 'react'

import { streamChat } from '@/api/chat'
import { createConversation, deleteConversation as deleteConvApi } from '@/api/conversations'
import { ChatInput, ChatMessages, ChatSidebar } from '@/components/chat'
import type { Citation, Conversation, ConversationListItem, Message, ThinkingStep } from '@/lib/types'

// ══════════════════════════════════════════════════════════════════════════════
// Helpers
// ══════════════════════════════════════════════════════════════════════════════

const STORAGE_KEY = 'qa-chat-conversations'

function loadLocalIds(): string[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) return JSON.parse(raw) as string[]
  } catch {
    /* ignore corrupt data */
  }
  return []
}

function saveLocalIds(ids: string[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(ids))
}

function nextId(): string {
  return Date.now().toString(36) + Math.random().toString(36).slice(2)
}

function toConversationListItem(c: Conversation): ConversationListItem {
  const last = c.messages[c.messages.length - 1]
  return {
    id: c.id,
    title: c.title,
    message_count: c.messages.length,
    last_message_preview: last ? last.content.slice(0, 50) : '',
    created_at: c.created_at,
    updated_at: c.updated_at,
  }
}

const SUGGESTED_PROMPTS = [
  '变压器巡检有哪些要点？',
  '如何判断变压器油是否需要更换？',
  '电力安全工作规程中关于停电操作的规定是什么？',
]

// ══════════════════════════════════════════════════════════════════════════════
// Component
// ══════════════════════════════════════════════════════════════════════════════

export function ChatPage() {
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [activeId, setActiveId] = useState<string>('')
  const [streaming, setStreaming] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lastFailedMsg, setLastFailedMsg] = useState<string | null>(null)
  const [inputText, setInputText] = useState('')

  // Refs so SSE callbacks always read latest state
  const setConvsRef = useRef(setConversations)
  setConvsRef.current = setConversations
  const activeIdRef = useRef(activeId)
  activeIdRef.current = activeId
  const streamingRef = useRef(streaming)
  streamingRef.current = streaming

  const [localIds, setLocalIds] = useState<string[]>(loadLocalIds)
  const localIdsRef = useRef(localIds)
  localIdsRef.current = localIds

  const persistLocalIds = useCallback((ids: string[]) => {
    setLocalIds(ids)
    saveLocalIds(ids)
  }, [])

  // ── Derive sidebar items ──
  const sidebarItems: ConversationListItem[] = useMemo(
    () => conversations.map(toConversationListItem),
    [conversations],
  )

  // ── Create conversation ──
  const handleCreate = useCallback(async () => {
    try {
      const c = await createConversation('新对话')
      setConversations((p) => [c, ...p])
      setActiveId(c.id)
      persistLocalIds([c.id, ...localIdsRef.current])
    } catch {
      setError('创建会话失败，请检查网络连接')
    }
  }, [persistLocalIds])

  // ── Delete conversation ──
  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deleteConvApi(id)
      } catch {
        /* best-effort deletion */
      }
      setConversations((p) => p.filter((c) => c.id !== id))
      if (activeIdRef.current === id) {
        const remaining = conversations.filter((c) => c.id !== id)
        setActiveId(remaining.length > 0 ? remaining[0]!.id : '')
      }
      persistLocalIds(localIdsRef.current.filter((lid) => lid !== id))
    },
    [conversations, persistLocalIds],
  )

  // ── Send message (SSE streaming) ──
  const sendMessage = useCallback(
    async (text: string) => {
      const trimmed = text.trim()
      if (!trimmed || streamingRef.current) return

      setError(null)
      setLastFailedMsg(null)

      let targetId = activeIdRef.current

      // ① Auto-create conversation if none active
      if (!targetId) {
        try {
          const title =
            trimmed.slice(0, 30) + (trimmed.length > 30 ? '…' : '')
          const c = await createConversation(title)
          setConvsRef.current((p) => {
            if (p.some((x) => x.id === c.id)) return p
            return [c, ...p]
          })
          targetId = c.id
          setActiveId(c.id)
          persistLocalIds([c.id, ...localIdsRef.current])
        } catch {
          setError('创建会话失败，请检查网络连接')
          return
        }
      }

      const uid = targetId

      // ② Push user message + empty assistant message
      const userMsg: Message = {
        id: nextId(),
        role: 'user',
        content: trimmed,
        timestamp: new Date().toISOString(),
      }
      const asstMsg: Message = {
        id: nextId(),
        role: 'assistant',
        content: '',
        timestamp: new Date().toISOString(),
        thinking: [],
        citations: [],
      }

      setConvsRef.current((prev) =>
        prev.map((c) => {
          if (c.id !== uid) return c
          const isFirst = c.messages.length === 0
          return {
            ...c,
            title: isFirst
              ? trimmed.slice(0, 30) + (trimmed.length > 30 ? '…' : '')
              : c.title,
            messages: [...c.messages, userMsg, asstMsg],
            updated_at: new Date().toISOString(),
          }
        }),
      )

      setStreaming(true)

      // Accumulators for SSE events
      let content = ''
      const steps: ThinkingStep[] = []
      const cites: Citation[] = []

      /**
       * Patch the last assistant message in the active conversation.
       * Uses array spread to avoid noUncheckedIndexedAccess issues.
       */
      const patchAssistant = (patch: {
        content?: string
        thinking?: ThinkingStep[]
        citations?: Citation[]
      }) => {
        setConvsRef.current((prev) =>
          prev.map((c) => {
            if (c.id !== uid) return c
            const msgs = [...c.messages]
            const lastIdx = msgs.length - 1
            const last = msgs[lastIdx]
            if (!last || last.role !== 'assistant') return c
            msgs[lastIdx] = { ...last, ...patch }
            return { ...c, messages: msgs }
          }),
        )
      }

      // ③ Initiate SSE stream
      const { abort } = streamChat(
        { conversation_id: uid, message: trimmed },
        {
          onIntentStatus(data) {
            const ex = steps.find((s) => s.type === 'intent')
            if (data.status === 'started' && !ex) {
              steps.push({
                type: 'intent',
                label: data.label,
                status: 'running',
              })
            } else if (data.status === 'done' && ex) {
              ex.status = 'done'
              ex.label = data.label
            }
            patchAssistant({ thinking: [...steps] })
          },
          onThinkingStep(data) {
            const idx = steps.findIndex((s) => s.type === data.step.type)
            if (idx >= 0) {
              steps[idx] = data.step
            } else {
              steps.push(data.step)
            }
            patchAssistant({ thinking: [...steps] })
          },
          onToken(data) {
            content += data.text
            patchAssistant({ content })
          },
          onCitation(data) {
            cites.push(data.citation)
            patchAssistant({ citations: [...cites] })
          },
          onDone() {
            setStreaming(false)
            patchAssistant({
              content,
              thinking: [...steps],
              citations: [...cites],
            })
          },
          onError(sseErr) {
            if (sseErr.fatal) {
              setStreaming(false)
              setError(sseErr.message || '请求失败')
              setLastFailedMsg(trimmed)
              patchAssistant({
                content,
                thinking: [...steps],
                citations: [...cites],
              })
              abort()
            }
          },
        },
      )
    },
    [persistLocalIds],
  )

  // ── Retry on error ──
  const handleRetry = useCallback(() => {
    if (lastFailedMsg) {
      const msg = lastFailedMsg
      setError(null)
      setLastFailedMsg(null)
      sendMessage(msg)
    }
  }, [lastFailedMsg, sendMessage])

  // ── Suggested prompt click ──
  const handleSuggested = useCallback(
    (prompt: string) => {
      setInputText(prompt)
      // Defer send so React commits the input text update first
      setTimeout(() => {
        sendMessage(prompt)
        setInputText('')
      }, 0)
    },
    [sendMessage],
  )

  // ── Active conversation ──
  const activeConv = conversations.find((c) => c.id === activeId)

  // ══════════════════════════════════════════════════════════════════════════
  // Render
  // ══════════════════════════════════════════════════════════════════════════

  return (
    <div className="flex h-full">
      {/* Left: conversation sidebar */}
      <ChatSidebar
        conversations={sidebarItems}
        activeId={activeId}
        onSelect={setActiveId}
        onCreate={handleCreate}
        onDelete={handleDelete}
      />

      {/* Right: main chat area */}
      <div className="flex min-w-0 flex-1 flex-col">
        {/* Header */}
        <header className="flex shrink-0 items-center justify-between border-b border-border bg-muted/30 px-6 py-3">
          <h1 className="text-lg font-semibold text-foreground">
            {activeConv?.title ?? '智能问答'}
          </h1>
          <a
            href="/admin"
            className="flex size-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            title="管理面板"
            aria-label="管理面板"
          >
            <Settings className="size-5" />
          </a>
        </header>

        {/* Messages area */}
        <ChatMessages
          messages={activeConv?.messages ?? []}
          streaming={streaming}
          error={error}
          suggestedPrompts={SUGGESTED_PROMPTS}
          onSuggestedClick={handleSuggested}
          onRetry={handleRetry}
        />

        {/* Input area */}
        <ChatInput
          onSend={sendMessage}
          disabled={streaming}
          value={inputText}
          onChange={setInputText}
        />
      </div>
    </div>
  )
}
