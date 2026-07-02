'use client'

import * as React from 'react'

import { cn } from '@/lib/utils'

type SwitchProps = Omit<React.ComponentProps<'input'>, 'type' | 'size'>

function Switch({
  className,
  disabled,
  id,
  checked,
  defaultChecked,
  onChange,
  ...props
}: SwitchProps) {
  const [internalChecked, setInternalChecked] = React.useState(defaultChecked ?? false)
  const isChecked = checked !== undefined ? checked : internalChecked

  const handleChange = React.useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setInternalChecked(e.target.checked)
      onChange?.(e)
    },
    [onChange],
  )

  return (
    <label
      className={cn(
        'relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full transition-colors duration-200',
        isChecked ? 'bg-primary' : 'bg-muted-foreground/20 hover:bg-muted-foreground/30',
        disabled && 'cursor-not-allowed opacity-50',
        className,
      )}
    >
      <input
        type="checkbox"
        id={id}
        disabled={disabled}
        checked={isChecked}
        onChange={handleChange}
        className="peer sr-only"
        {...props}
      />
      <span
        className={cn(
          'pointer-events-none block size-4 rounded-full bg-background shadow-sm ring-0 transition-all duration-200',
          isChecked ? 'translate-x-[18px]' : 'translate-x-[2px]',
        )}
      />
    </label>
  )
}

export { Switch }
