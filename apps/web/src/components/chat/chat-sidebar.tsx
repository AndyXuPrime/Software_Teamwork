import { MessageSquare,Plus, Trash2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import type { ConversationListItem } from '@/lib/types'
import { cn } from '@/lib/utils'

interface ChatSidebarProps {
  conversations: ConversationListItem[]
  activeId: string
  onSelect: (id: string) => void
  onCreate: () => void
  onDelete: (id: string) => void
}

export default function ChatSidebar({
  conversations,
  activeId,
  onSelect,
  onCreate,
  onDelete,
}: ChatSidebarProps) {
  return (
    <aside className="flex w-72 shrink-0 flex-col border-r border-border bg-card">
      {/* ── Header ── */}
      <div className="flex flex-col gap-2 border-b border-border p-4">
        <h2 className="text-lg font-semibold text-foreground">对话历史</h2>
        <Button onClick={onCreate} className="w-full">
          <Plus className="size-4" />
          新建对话
        </Button>
      </div>

      {/* ── Conversation list ── */}
      <ScrollArea className="flex-1">
        <div className="flex flex-col gap-1 p-2">
          {conversations.length === 0 && (
            <p className="px-4 py-8 text-center text-sm text-muted-foreground">
              暂无对话记录
            </p>
          )}

          {conversations.map((conv) => (
            <button
              key={conv.id}
              type="button"
              className={cn(
                'group relative flex w-full flex-col items-start gap-0.5 rounded-md px-3 py-2.5 text-left transition-colors hover:bg-muted',
                conv.id === activeId && 'bg-accent text-accent-foreground',
              )}
              onClick={() => onSelect(conv.id)}
            >
              <span className="w-full truncate pr-5 text-sm font-medium">
                {conv.title}
              </span>
              <span className="text-xs text-muted-foreground">
                <MessageSquare className="mr-1 inline-block size-3" />
                {conv.message_count} 条消息
              </span>

              {/* Delete button — visible on hover */}
              <span
                className="absolute right-2 top-1/2 -translate-y-1/2 flex size-5 items-center justify-center rounded text-muted-foreground opacity-0 transition-opacity hover:bg-destructive hover:text-destructive-foreground group-hover:opacity-100"
                role="button"
                title="删除对话"
                onClick={(e) => {
                  e.stopPropagation()
                  onDelete(conv.id)
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.stopPropagation()
                    onDelete(conv.id)
                  }
                }}
                tabIndex={0}
              >
                <Trash2 className="size-3" aria-hidden="true" />
              </span>
            </button>
          ))}
        </div>
      </ScrollArea>
    </aside>
  )
}
