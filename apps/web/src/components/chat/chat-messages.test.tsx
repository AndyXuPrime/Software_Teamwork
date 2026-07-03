import { fireEvent, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { QAMessage, QAReportArtifact, QAThinkingStep } from '@/lib/types'
import { renderWithProviders } from '@/test/render'

import ChatMessages from './chat-messages'

type TestThinkingStep = QAThinkingStep & {
  argumentsSummary?: unknown
  completedAt?: number
  errorSummary?: string
  iterationNo?: number
  reportArtifact?: QAReportArtifact
  resultSummary?: unknown
  startedAt?: number
  toolCallId?: string
  toolName?: string
}

function assistantWithThinking(thinking: TestThinkingStep[], content = '回答正文'): QAMessage {
  return {
    content,
    createdAt: '2026-07-03T00:00:00.000Z',
    id: 'assistant-1',
    role: 'assistant',
    sessionId: 'session-1',
    status: 'streaming',
    thinking,
  }
}

function renderMessage(message: QAMessage, onArtifactDownload = vi.fn()) {
  return renderWithProviders(
    <ChatMessages
      messages={[message]}
      streaming={message.status === 'streaming'}
      error={null}
      onArtifactDownload={onArtifactDownload}
    />,
  )
}

describe('ChatMessages ThinkPanel', () => {
  it('groups a single iteration with one expandable tool call', () => {
    renderMessage(
      assistantWithThinking([
        { iterationNo: 1, label: 'Agent 迭代 1', status: 'running', type: 'agent_iteration' },
        {
          argumentsSummary: { query: '变压器巡检', topK: 5 },
          completedAt: 1700000000900,
          iterationNo: 1,
          label: 'search_knowledge 完成',
          resultSummary: {
            citations: [{ internalUrl: 'http://10.0.0.2/private' }, { title: '公开摘要' }],
            hitCount: 3,
          },
          startedAt: 1700000000000,
          status: 'done',
          toolCallId: 'tool-1',
          toolName: 'search_knowledge',
          type: 'tool_call',
        },
      ]),
    )

    expect(screen.getByText('第 1 轮')).toBeInTheDocument()
    expect(screen.getByText(/1 个工具调用/)).toBeInTheDocument()
    expect(screen.getByText('耗时 900ms')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /search_knowledge 完成/ }))

    expect(screen.getByText('参数')).toBeInTheDocument()
    expect(screen.getByText('查询词')).toBeInTheDocument()
    expect(screen.getByText('变压器巡检')).toBeInTheDocument()
    expect(screen.getByText('TopK')).toBeInTheDocument()
    expect(screen.getByText('结果')).toBeInTheDocument()
    expect(screen.getByText('命中数')).toBeInTheDocument()
    expect(screen.getByText('引用数')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.queryByText(/10\.0\.0\.2/)).not.toBeInTheDocument()
  })

  it('renders multiple tool calls in the same iteration as a parallel list', () => {
    renderMessage(
      assistantWithThinking([
        { iterationNo: 1, label: 'Agent 迭代 1', status: 'running', type: 'agent_iteration' },
        {
          iterationNo: 1,
          label: 'search_knowledge 执行中',
          startedAt: 1700000000000,
          status: 'running',
          toolCallId: 'tool-1',
          type: 'tool_call',
        },
        {
          iterationNo: 1,
          label: 'search_session_attachments 执行中',
          startedAt: 1700000000005,
          status: 'running',
          toolCallId: 'tool-2',
          type: 'tool_call',
        },
      ]),
    )

    expect(screen.getByText(/2 个工具调用/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /search_knowledge 执行中/ })).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: /search_session_attachments 执行中/ }),
    ).toBeInTheDocument()
    expect(screen.getAllByText('▊')).toHaveLength(2)
  })

  it('keeps multiple ReAct iterations visually separated', () => {
    renderMessage(
      assistantWithThinking([
        { iterationNo: 1, label: 'Agent 迭代 1', status: 'done', type: 'agent_iteration' },
        {
          completedAt: 1700000000300,
          iterationNo: 1,
          label: 'search_knowledge 完成',
          startedAt: 1700000000000,
          status: 'done',
          type: 'tool_call',
        },
        { iterationNo: 2, label: 'Agent 迭代 2', status: 'running', type: 'agent_iteration' },
        {
          completedAt: 1700000000900,
          iterationNo: 2,
          label: 'get_citation_source 完成',
          startedAt: 1700000000500,
          status: 'done',
          type: 'tool_call',
        },
      ]),
    )

    expect(screen.getByText('第 1 轮')).toBeInTheDocument()
    expect(screen.getByText('第 2 轮')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /search_knowledge 完成/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /get_citation_source 完成/ })).toBeInTheDocument()
  })

  it('highlights failed tool calls and shows a safe failure summary', () => {
    renderMessage(
      assistantWithThinking([
        { iterationNo: 1, label: 'Agent 迭代 1', status: 'running', type: 'agent_iteration' },
        {
          completedAt: 1700000000400,
          errorSummary: '依赖服务暂时不可用',
          iterationNo: 1,
          label: 'search_knowledge 失败',
          startedAt: 1700000000000,
          status: 'failed',
          toolCallId: 'tool-1',
          type: 'tool_call',
        },
      ]),
    )

    expect(screen.getByText(/有失败/)).toBeInTheDocument()
    const trigger = screen.getByRole('button', { name: /search_knowledge 失败/ })
    expect(within(trigger).getByText('失败')).toBeInTheDocument()

    fireEvent.click(trigger)

    expect(screen.getByText(/失败原因：依赖服务暂时不可用/)).toBeInTheDocument()
  })

  it('omits ThinkPanel when a pure text answer has no tool or reasoning steps', () => {
    renderMessage({
      content: '纯文本回答',
      createdAt: '2026-07-03T00:00:00.000Z',
      id: 'assistant-1',
      role: 'assistant',
      sessionId: 'session-1',
      status: 'completed',
      thinking: [],
    })

    expect(screen.queryByText(/思考过程/)).not.toBeInTheDocument()
    expect(screen.getByText('纯文本回答')).toBeInTheDocument()
  })

  it('keeps report artifacts visible at message level after completion', () => {
    const onArtifactDownload = vi.fn()
    const artifact: QAReportArtifact = {
      artifactType: 'report_generation',
      downloadPath: '/api/v1/report-files/file-1/content',
      fileStatus: 'succeeded',
      filename: '巡检报告.docx',
      jobStatus: 'succeeded',
      reportId: 'report-1',
      reportName: '巡检报告',
    }

    renderMessage(
      {
        ...assistantWithThinking(
          [
            {
              label: 'report_generator 完成',
              reportArtifact: artifact,
              status: 'done',
              type: 'tool_call',
            },
          ],
          '报告已生成',
        ),
        artifacts: [artifact],
        status: 'completed',
      } as QAMessage & { artifacts: QAReportArtifact[] },
      onArtifactDownload,
    )

    expect(screen.getByText('巡检报告')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /下载报告/ }))

    expect(onArtifactDownload).toHaveBeenCalledWith(
      '/api/v1/report-files/file-1/content',
      '巡检报告.docx',
    )
  })
})
