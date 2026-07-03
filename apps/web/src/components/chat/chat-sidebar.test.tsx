import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import type { QASessionListItem } from '@/lib/types'

import ChatSidebar from './chat-sidebar'

function makeSession(id: string, title: string | undefined, messageCount = 0): QASessionListItem {
  return {
    id,
    title,
    status: 'active',
    messageCount,
    createdAt: `2026-07-03T0${messageCount}:00:00.000Z`,
    updatedAt: `2026-07-03T0${messageCount}:00:00.000Z`,
  } as QASessionListItem
}

function renderSidebar(options?: {
  activeId?: string
  onPrepareClearAll?: () => Promise<number>
  onClearAll?: () => Promise<void> | void
  sessions?: QASessionListItem[]
  onSelect?: (sessionId: string) => void
}) {
  const sessions = options?.sessions ?? [
    makeSession('s-1', '变压器巡检记录', 2),
    makeSession('s-2', '设备缺陷复盘', 1),
    makeSession('s-3', 'Transformer Oil Review', 3),
  ]

  return render(
    <ChatSidebar
      sessions={sessions}
      activeId={options?.activeId ?? 's-1'}
      isLoading={false}
      fetchError={null}
      onRetryFetch={vi.fn()}
      onSelect={options?.onSelect ?? vi.fn()}
      onCreate={vi.fn()}
      onDelete={vi.fn()}
      onRename={vi.fn()}
      onPrepareClearAll={options?.onPrepareClearAll}
      onClearAll={options?.onClearAll}
    />,
  )
}

describe('ChatSidebar title search', () => {
  it('filters loaded sessions by title keyword and clears back to the full list', async () => {
    const user = userEvent.setup()
    renderSidebar()

    await user.type(screen.getByRole('textbox', { name: '搜索对话标题' }), '巡检')

    expect(screen.getByText('变压器巡检记录')).toBeInTheDocument()
    expect(screen.queryByText('设备缺陷复盘')).not.toBeInTheDocument()
    expect(screen.queryByText('Transformer Oil Review')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '清空搜索' }))

    expect(screen.getByText('变压器巡检记录')).toBeInTheDocument()
    expect(screen.getByText('设备缺陷复盘')).toBeInTheDocument()
    expect(screen.getByText('Transformer Oil Review')).toBeInTheDocument()
  })

  it('matches English titles case-insensitively', async () => {
    const user = userEvent.setup()
    renderSidebar()

    await user.type(screen.getByRole('textbox', { name: '搜索对话标题' }), 'transformer')

    expect(screen.getByText('Transformer Oil Review')).toBeInTheDocument()
    expect(screen.queryByText('设备缺陷复盘')).not.toBeInTheDocument()
  })

  it('shows a search empty state without replacing it with a loading error', async () => {
    const user = userEvent.setup()
    renderSidebar()

    await user.type(screen.getByRole('textbox', { name: '搜索对话标题' }), '不存在')

    expect(screen.getByText('未找到匹配对话')).toBeInTheDocument()
    expect(screen.queryByText('会话列表加载失败')).not.toBeInTheDocument()
  })

  it('selects the clicked search result', async () => {
    const user = userEvent.setup()
    const onSelect = vi.fn()
    renderSidebar({ onSelect })

    await user.type(screen.getByRole('textbox', { name: '搜索对话标题' }), '缺陷')
    await user.click(screen.getByText('设备缺陷复盘'))

    expect(onSelect).toHaveBeenCalledWith('s-2')
  })

  it('selects the only fuzzy result when pressing Enter', async () => {
    const user = userEvent.setup()
    const onSelect = vi.fn()
    renderSidebar({ onSelect })

    const search = screen.getByRole('textbox', { name: '搜索对话标题' })
    await user.type(search, 'Oil')
    await user.keyboard('{Enter}')

    expect(onSelect).toHaveBeenCalledWith('s-3')
  })

  it('selects a unique exact title match on Enter even when other fuzzy matches exist', async () => {
    const user = userEvent.setup()
    const onSelect = vi.fn()
    renderSidebar({
      onSelect,
      sessions: [
        makeSession('s-1', '变压器巡检', 1),
        makeSession('s-2', '变压器巡检记录', 2),
        makeSession('s-3', '巡检计划', 3),
      ],
    })

    await user.type(screen.getByRole('textbox', { name: '搜索对话标题' }), '变压器巡检')
    await user.keyboard('{Enter}')

    expect(onSelect).toHaveBeenCalledWith('s-1')
  })

  it('does not switch sessions on Enter when there are multiple fuzzy matches', async () => {
    const user = userEvent.setup()
    const onSelect = vi.fn()
    renderSidebar({
      onSelect,
      sessions: [
        makeSession('s-1', '变压器巡检', 1),
        makeSession('s-2', '巡检计划', 2),
        makeSession('s-3', '缺陷复盘', 3),
      ],
    })

    await user.type(screen.getByRole('textbox', { name: '搜索对话标题' }), '巡检')
    await user.keyboard('{Enter}')

    expect(onSelect).not.toHaveBeenCalled()
  })

  it('does not match untitled sessions through the fallback display text', async () => {
    const user = userEvent.setup()
    renderSidebar({
      sessions: [makeSession('s-1', undefined, 0), makeSession('s-2', '业务对话', 1)],
    })

    expect(screen.getByText('新对话')).toBeInTheDocument()

    await user.type(screen.getByRole('textbox', { name: '搜索对话标题' }), '新对话')

    expect(screen.getByText('未找到匹配对话')).toBeInTheDocument()
    expect(screen.queryByText('新对话')).not.toBeInTheDocument()
  })

  it('confirms before clearing all sessions', async () => {
    const user = userEvent.setup()
    const onPrepareClearAll = vi.fn().mockResolvedValue(80)
    const onClearAll = vi.fn()
    renderSidebar({ onPrepareClearAll, onClearAll })

    await user.click(screen.getByRole('button', { name: '清空全部对话' }))

    expect(onClearAll).not.toHaveBeenCalled()
    expect(await screen.findByText('即将删除全部 80 个对话。此操作不可撤销。')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '全部删除' }))

    expect(onPrepareClearAll).toHaveBeenCalledTimes(1)
    expect(onClearAll).toHaveBeenCalledTimes(1)
  })
})
