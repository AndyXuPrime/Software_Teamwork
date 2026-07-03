type StreamingTextControllerOptions = {
  fps?: number
  maxCharsPerFrame?: number
  minCharsPerFrame?: number
  onDone?: () => void
  onUpdate: (text: string) => void
  reducedMotion?: boolean
}

type AnimationFrameBatcherOptions<T> = {
  merge: (current: T, next: T) => T
  onFlush: (value: T) => void
}

function userPrefersReducedMotion(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

export class AnimationFrameBatcher<T> {
  private active = true
  private frameId: number | null = null
  private readonly options: AnimationFrameBatcherOptions<T>
  private pending: T | undefined

  constructor(options: AnimationFrameBatcherOptions<T>) {
    this.options = options
  }

  push(value: T): void {
    if (!this.active) return

    this.pending = this.pending === undefined ? value : this.options.merge(this.pending, value)

    if (this.frameId === null) {
      this.frameId = requestAnimationFrame(this.handleFrame)
    }
  }

  flush(): void {
    if (!this.active) return

    if (this.frameId !== null) {
      cancelAnimationFrame(this.frameId)
      this.frameId = null
    }

    const pending = this.pending
    this.pending = undefined
    if (pending !== undefined) this.options.onFlush(pending)
  }

  cancel(): void {
    this.active = false
    this.pending = undefined
    if (this.frameId !== null) cancelAnimationFrame(this.frameId)
    this.frameId = null
  }

  private readonly handleFrame = (): void => {
    this.frameId = null
    if (!this.active) return

    const pending = this.pending
    this.pending = undefined
    if (pending !== undefined) this.options.onFlush(pending)
  }
}

export class StreamingTextController {
  private readonly fps: number
  private readonly maxCharsPerFrame: number
  private readonly minCharsPerFrame: number
  private readonly onDone?: () => void
  private readonly onUpdate: (text: string) => void
  private readonly reducedMotion: boolean
  private readonly segmenter: Intl.Segmenter | null

  private frameId: number | null = null
  private finishingBatchSize = 0
  private head = 0
  private holdUntil: number | null = null
  private lastFrameTime: number | null = null
  private queue: string[] = []
  private state: 'active' | 'cancelled' | 'done' | 'finishing' = 'active'
  private visibleText = ''

  constructor(options: StreamingTextControllerOptions) {
    this.onUpdate = options.onUpdate
    this.onDone = options.onDone
    this.fps = Math.max(1, options.fps ?? 30)
    this.minCharsPerFrame = Math.max(1, options.minCharsPerFrame ?? 2)
    this.maxCharsPerFrame = Math.max(this.minCharsPerFrame, options.maxCharsPerFrame ?? 100)
    this.reducedMotion = options.reducedMotion ?? userPrefersReducedMotion()
    this.segmenter =
      typeof Intl !== 'undefined' && 'Segmenter' in Intl
        ? new Intl.Segmenter('zh', { granularity: 'grapheme' })
        : null
  }

  feed(text: string): void {
    if (!text || this.state === 'cancelled' || this.state === 'done') return

    this.queue.push(...this.splitGraphemes(text))
    if (this.state === 'finishing') {
      this.finishingBatchSize = Math.max(
        this.finishingBatchSize,
        this.maxCharsPerFrame,
        Math.ceil(this.remaining / 8),
      )
    }
    this.scheduleFrame()
  }

  finish(): void {
    if (this.isStopped()) return

    this.state = 'finishing'
    if (this.remaining === 0) {
      this.notifyDone()
      return
    }
    this.finishingBatchSize = Math.max(this.maxCharsPerFrame, Math.ceil(this.remaining / 8))
    this.scheduleFrame()
  }

  cancel(): void {
    if (this.isStopped()) return

    this.state = 'cancelled'
    this.queue = []
    this.head = 0
    if (this.frameId !== null) cancelAnimationFrame(this.frameId)
    this.frameId = null
  }

  delayStart(delayMs: number, now = performance.now()): void {
    if (delayMs <= 0 || this.isStopped() || this.visibleText.length > 0) return

    const until = now + delayMs
    this.holdUntil = this.holdUntil === null ? until : Math.max(this.holdUntil, until)
  }

  getVisibleText(): string {
    return this.visibleText
  }

  private get remaining(): number {
    return this.queue.length - this.head
  }

  private readonly handleFrame = (now: number): void => {
    this.frameId = null
    if (this.state === 'cancelled' || this.state === 'done') return

    if (this.holdUntil !== null) {
      if (now < this.holdUntil) {
        this.scheduleFrame()
        return
      }
      this.holdUntil = null
    }

    const frameInterval = 1000 / this.fps
    if (this.lastFrameTime !== null && now - this.lastFrameTime < frameInterval) {
      this.scheduleFrame()
      return
    }
    this.lastFrameTime = now

    const remaining = this.remaining
    if (remaining === 0) {
      if (this.state === 'finishing') this.notifyDone()
      return
    }

    const count = this.reducedMotion ? remaining : this.pickBatchSize(remaining)
    const end = Math.min(this.head + count, this.queue.length)
    this.visibleText += this.queue.slice(this.head, end).join('')
    this.head = end
    this.onUpdate(this.visibleText)

    if (this.isStopped()) return

    if (this.head > 4096) {
      this.queue = this.queue.slice(this.head)
      this.head = 0
    }

    if (this.remaining > 0) {
      this.scheduleFrame()
    } else if (this.state === 'finishing') {
      this.notifyDone()
    }
  }

  private notifyDone(): void {
    if (this.state === 'cancelled' || this.state === 'done') return

    this.state = 'done'
    if (this.frameId !== null) cancelAnimationFrame(this.frameId)
    this.frameId = null
    this.onDone?.()
  }

  private isStopped(): boolean {
    return this.state === 'cancelled' || this.state === 'done'
  }

  private pickBatchSize(remaining: number): number {
    if (this.state === 'finishing') {
      return Math.min(remaining, this.finishingBatchSize)
    }
    if (remaining < 40) return Math.min(remaining, this.minCharsPerFrame)
    if (remaining < 200) return Math.min(remaining, 6)
    if (remaining < 1000) return Math.min(remaining, 16)
    return Math.min(remaining, this.maxCharsPerFrame, Math.ceil(remaining / 20))
  }

  private scheduleFrame(): void {
    if (this.frameId !== null || this.state === 'cancelled' || this.state === 'done') return
    this.frameId = requestAnimationFrame(this.handleFrame)
  }

  private splitGraphemes(text: string): string[] {
    if (this.segmenter) {
      return Array.from(this.segmenter.segment(text), (item) => item.segment)
    }
    return Array.from(text)
  }
}
