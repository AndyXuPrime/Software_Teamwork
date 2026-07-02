'use client'

import { Check, ChevronDown } from 'lucide-react'
import * as React from 'react'

import { cn } from '@/lib/utils'

// ── Context ──

type SelectContextValue = {
  value: string | undefined
  onValueChange: (value: string) => void
  open: boolean
  setOpen: (open: boolean) => void
  labelMap: React.MutableRefObject<Map<string, string>>
  triggerRef: React.RefObject<HTMLButtonElement | null>
  listRef: React.RefObject<HTMLDivElement | null>
  highlightedIndex: number
  setHighlightedIndex: (i: number) => void
  itemsRef: React.MutableRefObject<string[]>
  disabled?: boolean
}

const SelectContext = React.createContext<SelectContextValue | null>(null)

function useSelectContext() {
  const ctx = React.useContext(SelectContext)
  if (!ctx) throw new Error('Select compound components must be used within <Select>')
  return ctx
}

// ── Root ──

type SelectProps = {
  value?: string
  onValueChange?: (value: string) => void
  disabled?: boolean
  children: React.ReactNode
}

function Select({ value: controlledValue, onValueChange, disabled, children }: SelectProps) {
  const [internalValue, setInternalValue] = React.useState<string | undefined>(undefined)
  const [open, setOpen] = React.useState(false)
  const [highlightedIndex, setHighlightedIndex] = React.useState(-1)
  const labelMap = React.useRef<Map<string, string>>(new Map())
  const triggerRef = React.useRef<HTMLButtonElement | null>(null)
  const listRef = React.useRef<HTMLDivElement | null>(null)
  const itemsRef = React.useRef<string[]>([])
  const rootRef = React.useRef<HTMLDivElement | null>(null)

  const value = controlledValue ?? internalValue

  const handleValueChange = React.useCallback(
    (v: string) => {
      setInternalValue(v)
      onValueChange?.(v)
      setOpen(false)
    },
    [onValueChange],
  )

  // Click outside to close
  React.useEffect(() => {
    if (!open) return
    const handleClick = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [open])

  // Close on Escape
  React.useEffect(() => {
    if (!open) return
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setOpen(false)
        triggerRef.current?.focus()
      }
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [open])

  return (
    <SelectContext.Provider
      value={{
        value,
        onValueChange: handleValueChange,
        open,
        setOpen,
        labelMap,
        triggerRef,
        listRef,
        highlightedIndex,
        setHighlightedIndex,
        itemsRef,
        disabled,
      }}
    >
      <div ref={rootRef} data-slot="select" className="relative">
        {children}
      </div>
    </SelectContext.Provider>
  )
}

// ── Trigger ──

type SelectTriggerProps = React.ComponentProps<'button'> & {
  'aria-label'?: string
  id?: string
}

function SelectTrigger({ className, children, id, ...props }: SelectTriggerProps) {
  const { open, setOpen, disabled, triggerRef, setHighlightedIndex } = useSelectContext()

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault()
      setOpen(true)
      setHighlightedIndex(e.key === 'ArrowDown' ? 0 : -1)
      return
    }
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      setOpen(!open)
      return
    }
  }

  return (
    <button
      ref={triggerRef}
      type="button"
      data-slot="select-trigger"
      id={id}
      disabled={disabled}
      aria-expanded={open}
      aria-haspopup="listbox"
      onClick={() => {
        setOpen(!open)
        setHighlightedIndex(-1)
      }}
      onKeyDown={handleKeyDown}
      className={cn(
        'flex h-8 w-full items-center justify-between gap-1 rounded-lg border border-input bg-transparent px-2.5 py-1 text-base transition-colors outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-input/50 disabled:opacity-50 aria-invalid:border-destructive aria-invalid:ring-3 aria-invalid:ring-destructive/20 md:text-sm dark:bg-input/30 dark:disabled:bg-input/80 dark:aria-invalid:border-destructive/50 dark:aria-invalid:ring-destructive/40 [&_svg]:pointer-events-none [&_svg]:shrink-0',
        className,
      )}
      {...props}
    >
      {children}
      <ChevronDown
        className={cn(
          'size-4 shrink-0 text-muted-foreground transition-transform duration-300',
          open && 'rotate-180',
        )}
      />
    </button>
  )
}

// ── Value ──

type SelectValueProps = { placeholder?: string; className?: string }

function SelectValue({ placeholder, className }: SelectValueProps) {
  const { value, labelMap } = useSelectContext()
  const label = value ? (labelMap.current.get(value) ?? value) : undefined

  return (
    <span
      data-slot="select-value"
      className={cn('truncate', !label && 'text-muted-foreground', className)}
    >
      {label ?? placeholder}
    </span>
  )
}

// ── Content ──

type SelectContentProps = React.ComponentProps<'div'>

function SelectContent({ className, children, ...props }: SelectContentProps) {
  const { open, setHighlightedIndex } = useSelectContext()
  const innerRef = React.useRef<HTMLDivElement | null>(null)
  const [contentHeight, setContentHeight] = React.useState(0)

  React.useEffect(() => {
    if (open && innerRef.current) {
      setContentHeight(innerRef.current.scrollHeight)
    }
    if (!open) {
      setHighlightedIndex(-1)
    }
  }, [open, children, setHighlightedIndex])

  return (
    <div
      data-slot="select-content"
      className={cn(
        'absolute top-full left-0 z-50 w-full overflow-hidden transition-[max-height] duration-300 ease-out',
        className,
      )}
      style={{ maxHeight: open ? contentHeight : 0 }}
      role="listbox"
      {...props}
    >
      <div
        ref={innerRef}
        className="mt-1 rounded-lg border bg-popover p-1 text-popover-foreground shadow-md"
      >
        <SelectContentInner>{children}</SelectContentInner>
      </div>
    </div>
  )
}

// ── Inner (sliding highlight) ──

function SelectContentInner({ children }: { children: React.ReactNode }) {
  const { setHighlightedIndex } = useSelectContext()

  const handleMouseEnterItem = React.useCallback(
    (e: React.MouseEvent<HTMLElement>, index: number) => {
      const item = e.currentTarget
      const list = item.closest<HTMLElement>('[data-slot="select-content-inner"]')
      if (!list) return
      list.style.setProperty('--slider-offset', `${item.offsetTop}px`)
      setHighlightedIndex(index)
    },
    [setHighlightedIndex],
  )

  return (
    <div
      data-slot="select-content-inner"
      className="relative"
      style={{ '--slider-offset': '0px' } as React.CSSProperties}
    >
      <div
        className="relative flex flex-col gap-0.5 overflow-auto py-1
          before:pointer-events-none before:absolute before:left-0 before:right-0
          before:z-0 before:h-8 before:rounded-md before:bg-accent
          before:opacity-0 before:transition-all before:duration-300
          before:ease-out hover:before:opacity-100
          hover:before:translate-y-[var(--slider-offset)]"
        role="presentation"
      >
        {React.Children.map(children, (child, index) => {
          if (!React.isValidElement(child)) return child
          return React.cloneElement(child, {
            onMouseEnterItem: (e: React.MouseEvent<HTMLElement>) => handleMouseEnterItem(e, index),
            index,
          } as Record<string, unknown>)
        })}
      </div>
    </div>
  )
}

// ── Item ──

type SelectItemProps = React.ComponentProps<'div'> & {
  value: string
  disabled?: boolean
  onMouseEnterItem?: (e: React.MouseEvent<HTMLElement>) => void
  index?: number
}

function SelectItem({
  className,
  children,
  value,
  disabled,
  onMouseEnterItem,
  index = -1,
  ...props
}: SelectItemProps) {
  const {
    value: selectedValue,
    onValueChange,
    labelMap,
    highlightedIndex,
    itemsRef,
  } = useSelectContext()
  const isSelected = selectedValue === value

  // Register label
  React.useEffect(() => {
    if (typeof children === 'string') {
      labelMap.current.set(value, children)
    }
    // Register this item's value in itemsRef
    if (index >= 0 && value) {
      // itemsRef gets updated externally; let's use a different approach
    }
  }, [value, children, labelMap, index])

  // Register / unregister item value
  React.useEffect(() => {
    const items = itemsRef.current
    if (index >= 0) {
      items[index] = value
    }
    return () => {
      if (index >= 0) {
        delete items[index]
      }
    }
  }, [value, index, itemsRef])

  const content =
    typeof children === 'string' ? (
      <span data-slot="select-item-text" className="truncate">
        {children}
      </span>
    ) : (
      children
    )

  return (
    <div
      role="option"
      aria-selected={isSelected}
      data-slot="select-item"
      data-value={value}
      data-highlighted={highlightedIndex === index || undefined}
      onMouseEnter={onMouseEnterItem}
      onClick={() => {
        if (!disabled) {
          // Capture label before closing
          if (typeof children === 'string') {
            labelMap.current.set(value, children)
          }
          onValueChange(value)
        }
      }}
      className={cn(
        'relative flex w-full cursor-default items-center gap-2 rounded-md py-1.5 pr-8 pl-2 text-sm outline-hidden select-none',
        'data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground',
        disabled && 'pointer-events-none opacity-50',
        '[&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*="size-"])]:size-4',
        className,
      )}
      {...props}
    >
      {isSelected && (
        <span className="absolute right-2 flex size-4 items-center justify-center">
          <Check className="size-3.5" />
        </span>
      )}
      {content}
    </div>
  )
}

// ── ItemText ──

function SelectItemText({ className, ...props }: React.ComponentProps<'span'>) {
  return <span data-slot="select-item-text" className={cn('truncate', className)} {...props} />
}

// ── Stubs ──

function SelectLabel({ className, ...props }: React.ComponentProps<'span'>) {
  return (
    <span
      data-slot="select-label"
      className={cn('mb-1 block px-2 text-sm font-medium text-foreground', className)}
      {...props}
    />
  )
}

function SelectGroup({ className, ...props }: React.ComponentProps<'div'>) {
  return <div data-slot="select-group" className={cn(className)} {...props} />
}

function SelectGroupLabel({ className, ...props }: React.ComponentProps<'span'>) {
  return (
    <span
      data-slot="select-group-label"
      className={cn('px-2 py-1.5 text-xs font-medium text-muted-foreground', className)}
      {...props}
    />
  )
}

function SelectSeparator({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div
      data-slot="select-separator"
      className={cn('-mx-1 my-1 h-px bg-border', className)}
      {...props}
    />
  )
}

export {
  Select,
  SelectContent,
  SelectGroup,
  SelectGroupLabel,
  SelectItem,
  SelectItemText,
  SelectLabel,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
}
