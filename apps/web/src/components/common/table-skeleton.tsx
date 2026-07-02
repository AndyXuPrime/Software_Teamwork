import { cn } from '@/lib/utils'

type TableSkeletonProps = {
  className?: string
  columns?: number
  rows?: number
  showToolbar?: boolean
}

export function TableSkeleton({
  className,
  columns = 6,
  rows = 5,
  showToolbar = true,
}: TableSkeletonProps) {
  return (
    <div className={cn('space-y-4', className)}>
      {showToolbar && (
        <div className="flex items-center justify-between gap-3">
          <div className="h-7 w-40 rounded skeleton-shimmer" />
          <div className="h-8 w-24 rounded skeleton-shimmer" />
        </div>
      )}
      <div className="flex gap-2">
        <div className="h-8 flex-1 rounded skeleton-shimmer" />
        <div className="h-8 w-28 rounded skeleton-shimmer" />
      </div>
      <div className="overflow-hidden rounded-lg border border-border bg-card">
        <div className="border-b border-border px-4 py-3">
          <div className="grid gap-3" style={{ gridTemplateColumns: `repeat(${columns}, 1fr)` }}>
            {Array.from({ length: columns }).map((_, index) => (
              <div key={index} className="h-4 rounded skeleton-shimmer" />
            ))}
          </div>
        </div>
        <div className="divide-y divide-border">
          {Array.from({ length: rows }).map((_, rowIndex) => (
            <div
              key={rowIndex}
              className="grid gap-3 px-4 py-3"
              style={{ gridTemplateColumns: `repeat(${columns}, 1fr)` }}
            >
              {Array.from({ length: columns }).map((_, columnIndex) => (
                <div key={columnIndex} className="h-4 rounded skeleton-shimmer" />
              ))}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
