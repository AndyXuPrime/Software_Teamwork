'use client'

import { Select as SelectPrimitive } from '@base-ui/react/select'
import { Check, ChevronDown, ChevronUp } from 'lucide-react'
import * as React from 'react'

import { cn } from '@/lib/utils'

function Select({ ...props }: React.ComponentProps<typeof SelectPrimitive.Root>) {
  return <SelectPrimitive.Root data-slot="select" {...props} />
}

function SelectTrigger({ className, children, ...props }: SelectPrimitive.Trigger.Props) {
  return (
    <SelectPrimitive.Trigger
      data-slot="select-trigger"
      className={cn(
        'flex h-8 w-full items-center justify-between gap-1 rounded-lg border border-input bg-transparent px-2.5 py-1 text-base transition-colors outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-input/50 disabled:opacity-50 aria-invalid:border-destructive aria-invalid:ring-3 aria-invalid:ring-destructive/20 md:text-sm dark:bg-input/30 dark:disabled:bg-input/80 dark:aria-invalid:border-destructive/50 dark:aria-invalid:ring-destructive/40 [&_svg]:pointer-events-none [&_svg]:shrink-0',
        className,
      )}
      {...props}
    >
      {children}
      <ChevronDown className="size-4 shrink-0 text-muted-foreground transition-transform duration-300 [&[data-popup-open]]:rotate-180" />
    </SelectPrimitive.Trigger>
  )
}

function SelectValue({ className, ...props }: SelectPrimitive.Value.Props) {
  return (
    <SelectPrimitive.Value
      data-slot="select-value"
      className={cn('truncate text-foreground data-[placeholder]:text-muted-foreground', className)}
      {...props}
    />
  )
}

function SelectContent({
  className,
  align = 'center',
  alignOffset = 0,
  side = 'bottom',
  sideOffset = 4,
  children,
  ...props
}: SelectPrimitive.Popup.Props &
  Pick<SelectPrimitive.Positioner.Props, 'align' | 'alignOffset' | 'side' | 'sideOffset'>) {
  return (
    <SelectPrimitive.Portal>
      <SelectPrimitive.Positioner
        align={align}
        alignOffset={alignOffset}
        side={side}
        sideOffset={sideOffset}
        className="isolate z-50"
      >
        <SelectPrimitive.Popup
          data-slot="select-content"
          className={cn(
            'z-50 flex max-h-96 min-w-[8rem] flex-col overflow-hidden rounded-lg bg-popover p-1 text-popover-foreground shadow-md ring-1 ring-foreground/10 outline-hidden duration-100 data-[side=bottom]:slide-in-from-top-2 data-[side=inline-end]:slide-in-from-left-2 data-[side=inline-start]:slide-in-from-right-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95',
            className,
          )}
          {...props}
        >
          <SelectPrimitive.ScrollUpArrow className="flex justify-center py-1">
            <ChevronUp className="size-3 text-muted-foreground" />
          </SelectPrimitive.ScrollUpArrow>
          <SelectContentInner>{children}</SelectContentInner>
          <SelectPrimitive.ScrollDownArrow className="flex justify-center py-1">
            <ChevronDown className="size-3 text-muted-foreground" />
          </SelectPrimitive.ScrollDownArrow>
        </SelectPrimitive.Popup>
      </SelectPrimitive.Positioner>
    </SelectPrimitive.Portal>
  )
}

/**
 * Inner wrapper that manages the sliding highlight background and floating
 * cursor icon. Extracted so state (refs, event handlers) lives outside the
 * Portal/Positioner tree.
 */
function SelectContentInner({ children }: { children: React.ReactNode }) {
  const listRef = React.useRef<HTMLDivElement>(null)
  const floatingRef = React.useRef<HTMLDivElement>(null)
  const rafRef = React.useRef<number>(0)

  const handleMouseEnterItem = React.useCallback((e: React.MouseEvent<HTMLElement>) => {
    const item = e.currentTarget
    const list = item.closest<HTMLElement>('[data-slot="select-content-inner"]')
    if (!list) return
    list.style.setProperty('--slider-offset', `${item.offsetTop}px`)
  }, [])

  const handleMouseMove = React.useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    if (rafRef.current) return
    rafRef.current = requestAnimationFrame(() => {
      rafRef.current = 0
      const floating = floatingRef.current
      if (!floating) return
      const list = listRef.current
      if (!list) return
      const rect = list.getBoundingClientRect()
      const x = e.clientX - rect.x
      const y = e.clientY - rect.y
      const size = floating.offsetWidth || 26
      floating.style.setProperty('--float-x', `${x - size / 2}px`)
      floating.style.setProperty('--float-y', `${y - size / 2}px`)
    })
  }, [])

  React.useEffect(() => {
    return () => {
      if (rafRef.current) cancelAnimationFrame(rafRef.current)
    }
  }, [])

  return (
    <div
      data-slot="select-content-inner"
      className="group relative"
      style={
        {
          '--slider-offset': '0px',
          '--float-x': '0px',
          '--float-y': '0px',
        } as React.CSSProperties
      }
      onMouseMove={handleMouseMove}
    >
      <SelectPrimitive.List
        ref={listRef}
        className="relative flex flex-col gap-0.5 overflow-auto py-1
          before:pointer-events-none before:absolute before:left-0 before:right-0
          before:z-0 before:h-8 before:rounded-md before:bg-accent
          before:opacity-0 before:transition-all before:duration-300
          before:ease-out hover:before:opacity-100
          hover:before:translate-y-[var(--slider-offset)]"
      >
        {React.Children.map(children, (child) => {
          if (!React.isValidElement(child)) return child
          return React.cloneElement(child, { onMouseEnterItem: handleMouseEnterItem } as Record<
            string,
            unknown
          >)
        })}
      </SelectPrimitive.List>
      <div
        ref={floatingRef}
        aria-hidden="true"
        className="pointer-events-none absolute z-20 flex size-7 items-center justify-center rounded-lg bg-foreground/10 opacity-0 transition-opacity duration-300 [.group:hover_&]:opacity-100"
        style={{
          left: 'var(--float-x)',
          top: 'var(--float-y)',
        }}
      />
    </div>
  )
}

function SelectItem({
  className,
  children,
  onMouseEnterItem,
  ...props
}: SelectPrimitive.Item.Props & { onMouseEnterItem?: React.MouseEventHandler<HTMLElement> }) {
  // Auto-wrap plain text children in SelectItemText so SelectValue
  // displays the label instead of the raw value (e.g. kb name vs kb id).
  const content =
    typeof children === 'string' ? (
      <SelectPrimitive.ItemText>{children}</SelectPrimitive.ItemText>
    ) : (
      children
    )
  return (
    <SelectPrimitive.Item
      data-slot="select-item"
      onMouseEnter={onMouseEnterItem}
      className={cn(
        'relative flex w-full cursor-default items-center gap-2 rounded-md py-1.5 pr-8 pl-2 text-sm outline-hidden select-none data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*="size-"])]:size-4',
        className,
      )}
      {...props}
    >
      <SelectPrimitive.ItemIndicator className="absolute right-2 flex size-4 items-center justify-center">
        <Check className="size-3.5" />
      </SelectPrimitive.ItemIndicator>
      {content}
    </SelectPrimitive.Item>
  )
}

function SelectItemText({ className, ...props }: SelectPrimitive.ItemText.Props) {
  return (
    <SelectPrimitive.ItemText
      data-slot="select-item-text"
      className={cn('truncate', className)}
      {...props}
    />
  )
}

function SelectLabel({ className, ...props }: SelectPrimitive.Label.Props) {
  return (
    <SelectPrimitive.Label
      data-slot="select-label"
      className={cn('mb-1 block text-sm font-medium text-foreground', className)}
      {...props}
    />
  )
}

function SelectGroup({ ...props }: SelectPrimitive.Group.Props) {
  return <SelectPrimitive.Group data-slot="select-group" {...props} />
}

function SelectGroupLabel({ className, ...props }: SelectPrimitive.GroupLabel.Props) {
  return (
    <SelectPrimitive.GroupLabel
      data-slot="select-group-label"
      className={cn('px-2 py-1.5 text-xs font-medium text-muted-foreground', className)}
      {...props}
    />
  )
}

function SelectSeparator({
  className,
  ...props
}: React.ComponentProps<typeof SelectPrimitive.Separator>) {
  return (
    <SelectPrimitive.Separator
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
