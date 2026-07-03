import { describe, expect, it, vi } from 'vitest'

import { AnimationFrameBatcher, StreamingTextController } from './streaming-text'

function installAnimationFrameHarness() {
  let nextId = 1
  const callbacks = new Map<number, FrameRequestCallback>()
  const request = vi.fn((callback: FrameRequestCallback) => {
    const id = nextId++
    callbacks.set(id, callback)
    return id
  })
  const cancel = vi.fn((id: number) => {
    callbacks.delete(id)
  })

  vi.stubGlobal('requestAnimationFrame', request)
  vi.stubGlobal('cancelAnimationFrame', cancel)

  return {
    cancel,
    pending: () => callbacks.size,
    request,
    step(now: number) {
      const current = [...callbacks.values()]
      callbacks.clear()
      for (const callback of current) callback(now)
    },
  }
}

describe('StreamingTextController', () => {
  it('consumes short text incrementally at the configured frame rate', () => {
    const raf = installAnimationFrameHarness()
    const updates: string[] = []
    const controller = new StreamingTextController({
      fps: 30,
      minCharsPerFrame: 2,
      onUpdate: (text) => updates.push(text),
      reducedMotion: false,
    })

    controller.feed('abcdef')

    expect(updates).toEqual([])
    expect(raf.pending()).toBe(1)

    raf.step(0)
    expect(updates).toEqual(['ab'])

    raf.step(16)
    expect(updates).toEqual(['ab'])

    raf.step(34)
    expect(updates).toEqual(['ab', 'abcd'])
  })

  it('accelerates after finish and emits the exact complete text once', () => {
    const raf = installAnimationFrameHarness()
    const onDone = vi.fn()
    const updates: string[] = []
    const content = '长回答'.repeat(500)
    const controller = new StreamingTextController({
      maxCharsPerFrame: 100,
      onDone,
      onUpdate: (text) => updates.push(text),
      reducedMotion: false,
    })

    controller.feed(content)
    controller.finish()
    for (let frame = 0; frame < 10 && raf.pending() > 0; frame += 1) {
      raf.step(frame * 34)
    }

    expect(updates.at(-1)).toBe(content)
    expect(controller.getVisibleText()).toBe(content)
    expect(onDone).toHaveBeenCalledTimes(1)
    expect(updates.length).toBeLessThan(10)
  })

  it('cancels pending animation without later updates or completion', () => {
    const raf = installAnimationFrameHarness()
    const onDone = vi.fn()
    const onUpdate = vi.fn()
    const controller = new StreamingTextController({ onDone, onUpdate })

    controller.feed('不会显示')
    controller.finish()
    controller.cancel()
    raf.step(34)

    expect(onUpdate).not.toHaveBeenCalled()
    expect(onDone).not.toHaveBeenCalled()
    expect(raf.cancel).toHaveBeenCalledTimes(1)
  })

  it('coalesces all queued text per frame for reduced motion', () => {
    const raf = installAnimationFrameHarness()
    const onUpdate = vi.fn()
    const controller = new StreamingTextController({
      onUpdate,
      reducedMotion: true,
    })

    controller.feed('第一批')
    controller.feed('第二批')
    raf.step(0)

    expect(onUpdate).toHaveBeenCalledTimes(1)
    expect(onUpdate).toHaveBeenCalledWith('第一批第二批')
  })

  it('holds the first visible update until the configured start delay elapses', () => {
    const raf = installAnimationFrameHarness()
    const updates: string[] = []
    const controller = new StreamingTextController({
      minCharsPerFrame: 2,
      onUpdate: (text) => updates.push(text),
      reducedMotion: false,
    })

    controller.delayStart(1000, 0)
    controller.feed('abcdef')

    raf.step(0)
    raf.step(999)

    expect(updates).toEqual([])

    raf.step(1000)

    expect(updates).toEqual(['ab'])
  })

  it('does not split a Unicode surrogate pair', () => {
    const raf = installAnimationFrameHarness()
    const onUpdate = vi.fn()
    const controller = new StreamingTextController({
      minCharsPerFrame: 1,
      onUpdate,
      reducedMotion: false,
    })

    controller.feed('🙂a')
    raf.step(0)

    expect(onUpdate).toHaveBeenCalledWith('🙂')
  })
})

describe('AnimationFrameBatcher', () => {
  it('merges all values queued in the same frame', () => {
    const raf = installAnimationFrameHarness()
    const onFlush = vi.fn()
    const batcher = new AnimationFrameBatcher<Record<string, string>>({
      merge: (current, next) => ({ ...current, ...next }),
      onFlush,
    })

    batcher.push({ reasoning: '第一段' })
    batcher.push({ status: 'streaming' })

    expect(raf.request).toHaveBeenCalledTimes(1)
    raf.step(0)
    expect(onFlush).toHaveBeenCalledWith({ reasoning: '第一段', status: 'streaming' })
  })

  it('flushes pending work immediately and cancels its frame', () => {
    const raf = installAnimationFrameHarness()
    const onFlush = vi.fn()
    const batcher = new AnimationFrameBatcher<string>({
      merge: (_, next) => next,
      onFlush,
    })

    batcher.push('latest')
    batcher.flush()
    raf.step(0)

    expect(onFlush).toHaveBeenCalledTimes(1)
    expect(onFlush).toHaveBeenCalledWith('latest')
    expect(raf.cancel).toHaveBeenCalledTimes(1)
  })

  it('drops pending work after cancellation', () => {
    const raf = installAnimationFrameHarness()
    const onFlush = vi.fn()
    const batcher = new AnimationFrameBatcher<string>({
      merge: (_, next) => next,
      onFlush,
    })

    batcher.push('discarded')
    batcher.cancel()
    raf.step(0)

    expect(onFlush).not.toHaveBeenCalled()
  })
})
