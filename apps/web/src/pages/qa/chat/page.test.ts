import { describe, expect, it } from 'vitest'

import {
  finalizeThinkingStepsOnAnswerCompleted,
  sanitizeThinkingStep,
  type ToolThinkingStep,
  upsertReasoningStep,
} from './page'

describe('ChatPage stream thinking helpers', () => {
  it('keeps same-type reasoning steps separated across iterations', () => {
    const first = sanitizeThinkingStep({
      detail: '第 1 轮生成',
      iterationNo: 1,
      label: '生成回答',
      status: 'done',
      type: 'generation',
    })
    const second = sanitizeThinkingStep({
      detail: '第 2 轮生成',
      iterationNo: 2,
      label: '生成回答',
      status: 'done',
      type: 'generation',
    })

    const steps = upsertReasoningStep(upsertReasoningStep([], first), second)

    expect(steps).toHaveLength(2)
    expect(steps.map((step) => step.iterationNo)).toEqual([1, 2])
    expect(steps.map((step) => step.detail)).toEqual(['第 1 轮生成', '第 2 轮生成'])
  })

  it('updates only a matching stable reasoning step in the same iteration', () => {
    const started = sanitizeThinkingStep({
      id: 'reason-1',
      iterationNo: 1,
      label: '校验答案',
      status: 'running',
      type: 'verify',
    })
    const completed = sanitizeThinkingStep({
      id: 'reason-1',
      iterationNo: 1,
      label: '校验答案',
      status: 'done',
      type: 'verify',
    })
    const nextIteration = sanitizeThinkingStep({
      id: 'reason-1',
      iterationNo: 2,
      label: '校验答案',
      status: 'running',
      type: 'verify',
    })

    const updated = upsertReasoningStep(upsertReasoningStep([], started), completed)
    const withNextIteration = upsertReasoningStep(updated, nextIteration)

    expect(updated).toHaveLength(1)
    expect(updated[0]).toMatchObject({ iterationNo: 1, status: 'done' })
    expect(withNextIteration).toHaveLength(2)
    expect(withNextIteration[1]).toMatchObject({ iterationNo: 2, status: 'running' })
  })

  it('does not mark running tool calls as done when the answer completes', () => {
    const steps: ToolThinkingStep[] = [
      {
        iterationNo: 1,
        label: 'Agent 迭代 1',
        status: 'running',
        type: 'agent_iteration',
      },
      {
        iterationNo: 1,
        label: '调用: report_generator',
        status: 'running',
        toolCallId: 'tool-1',
        type: 'tool_call',
      },
    ]

    const finalized = finalizeThinkingStepsOnAnswerCompleted(steps)

    expect(finalized[0]).toMatchObject({ status: 'done', type: 'agent_iteration' })
    expect(finalized[1]).toMatchObject({ status: 'running', type: 'tool_call' })
  })
})
