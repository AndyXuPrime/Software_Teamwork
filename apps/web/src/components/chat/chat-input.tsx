import { Send } from 'lucide-react'
import { type KeyboardEvent, useCallback, useEffect, useRef } from 'react'

import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'

type ChatInputProps = {
  onSend: (text: string) => void
  disabled: boolean
  value: string
  onChange: (value: string) => void
  size?: 'normal' | 'large'
  className?: string
}

export default function ChatInput({
  onSend,
  disabled,
  value,
  onChange,
  size = 'normal',
  className,
}: ChatInputProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  // Auto-resize on text change
  useEffect(() => {
    const el = textareaRef.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = `${Math.min(el.scrollHeight, 160)}px`
  }, [value])

  const handleSend = useCallback(() => {
    const trimmed = value.trim()
    if (!trimmed || disabled) return
    onSend(trimmed)
    onChange('')
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto'
    }
  }, [value, disabled, onSend, onChange])

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const canSend = value.trim().length > 0 && !disabled

  const isLarge = size === 'large'

  return (
    <div
      className={cn(
        isLarge
          ? 'rounded-2xl border border-border/50 bg-card shadow-[0_4px_24px_-2px_rgba(0,0,0,0.10),0_1px_4px_-1px_rgba(0,0,0,0.05)] px-5 py-4 focus-within:border-primary/50 focus-within:shadow-[0_4px_24px_-2px_rgba(0,0,0,0.14),0_1px_4px_-1px_rgba(0,0,0,0.07)] focus-within:ring-2 focus-within:ring-primary/10'
          : 'rounded-xl border border-border/40 bg-card shadow-[0_2px_12px_-1px_rgba(0,0,0,0.07),0_1px_3px_-1px_rgba(0,0,0,0.04)] px-4 py-3 focus-within:border-primary/40 focus-within:shadow-[0_2px_12px_-1px_rgba(0,0,0,0.10),0_1px_3px_-1px_rgba(0,0,0,0.06)] focus-within:ring-2 focus-within:ring-primary/10',
        'shrink-0',
        className,
      )}
    >
      <div className="flex items-end gap-2">
        <Textarea
          ref={textareaRef}
          className={cn(
            'min-h-0 flex-1 resize-none border-0 bg-transparent p-0 placeholder:text-muted-foreground focus-visible:ring-0 disabled:cursor-not-allowed disabled:opacity-60 md:text-sm',
            isLarge ? 'py-2 text-lg' : 'py-1 text-base',
          )}
          placeholder={
            isLarge ? '输入问题，Enter 发送…' : '输入您的问题… (Enter 发送，Shift+Enter 换行)'
          }
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={disabled}
          rows={1}
        />
        <Button
          size="icon"
          onClick={handleSend}
          disabled={!canSend}
          className="shrink-0 rounded-full bg-primary text-primary-foreground transition-all duration-200 hover:bg-primary/90 hover:scale-110 hover:shadow-md active:scale-90"
          aria-label="发送消息"
        >
          <Send className="size-4" aria-hidden="true" />
        </Button>
      </div>
    </div>
  )
}
