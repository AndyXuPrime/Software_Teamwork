import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { renderWithProviders } from '@/test/render'

import { KnowledgeBaseMultiSelect } from './knowledge-base-multi-select'

function jsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    status: init?.status ?? 200,
    statusText: init?.statusText,
  })
}

function knowledgeBase(id: string, name: string) {
  return {
    createdAt: '2026-07-04T00:00:00Z',
    documentCount: 0,
    id,
    name,
    retrievalStrategy: { mode: 'semantic' },
    status: 'ready',
    updatedAt: '2026-07-04T00:00:00Z',
  }
}

function ControlledSelector({
  initialValue = [],
  onValueChange,
  variant = 'panel',
}: {
  initialValue?: string[]
  onValueChange: (value: string[]) => void
  variant?: 'panel' | 'compact'
}) {
  const [value, setValue] = useState<string[]>(initialValue)

  return (
    <KnowledgeBaseMultiSelect
      value={value}
      variant={variant}
      onChange={(nextValue) => {
        setValue(nextValue)
        onValueChange(nextValue)
      }}
    />
  )
}

describe('KnowledgeBaseMultiSelect', () => {
  it('opens the compact knowledge selector upward from a left aligned trigger', async () => {
    const user = userEvent.setup()
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>(async () =>
        jsonResponse({
          data: [knowledgeBase('kb-001', '运行规程库')],
          page: { page: 1, pageSize: 100, total: 1 },
          requestId: 'req-kb-page-1',
        }),
      ),
    )

    renderWithProviders(<ControlledSelector variant="compact" onValueChange={vi.fn()} />)

    expect(screen.getByRole('button', { name: /选择知识库/ })).toBeVisible()
    const closedPopover = screen.getByTestId('knowledge-base-selector-popover')
    expect(closedPopover).toHaveAttribute('aria-hidden', 'true')
    expect(closedPopover).toHaveClass('opacity-0', 'pointer-events-none')
    expect(screen.getByLabelText('知识库范围搜索')).toBeDisabled()

    await user.click(screen.getByRole('button', { name: /选择知识库/ }))

    const popover = screen.getByTestId('knowledge-base-selector-popover')
    expect(popover).toHaveAttribute('data-side', 'top')
    expect(popover).toHaveAttribute('aria-hidden', 'false')
    expect(popover).toHaveClass('opacity-100', 'pointer-events-auto')
    expect(popover).toHaveClass('w-[min(14.75rem,calc(100vw-2rem))]')
    expect(screen.getByLabelText('知识库范围搜索')).toBeVisible()
    expect(screen.getByLabelText('知识库范围搜索')).toBeEnabled()
    expect(screen.getByRole('checkbox', { name: /运行规程库 \(kb-001\)/ })).toBeVisible()
    expect(screen.queryByText('kb-001')).not.toBeInTheDocument()
  })

  it('selects multiple knowledge bases from compact checkbox items', async () => {
    const user = userEvent.setup()
    const values: string[][] = []
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>(async () =>
        jsonResponse({
          data: [knowledgeBase('kb-001', '运行规程库'), knowledgeBase('kb-002', '检修案例库')],
          page: { page: 1, pageSize: 100, total: 2 },
          requestId: 'req-kb-page-1',
        }),
      ),
    )

    renderWithProviders(
      <ControlledSelector
        variant="compact"
        onValueChange={(nextValue) => values.push(nextValue)}
      />,
    )

    await user.click(screen.getByRole('button', { name: /选择知识库/ }))
    await user.click(await screen.findByRole('checkbox', { name: /运行规程库 \(kb-001\)/ }))
    await user.click(screen.getByRole('checkbox', { name: /检修案例库 \(kb-002\)/ }))

    await waitFor(() => expect(values.at(-1)).toEqual(['kb-001', 'kb-002']))
    expect(screen.getByRole('button', { name: /已选 2 个知识库/ })).toBeVisible()
  })

  it('clears compact selections through the no knowledge base option', async () => {
    const user = userEvent.setup()
    const values: string[][] = []
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>(async () =>
        jsonResponse({
          data: [knowledgeBase('kb-001', '运行规程库'), knowledgeBase('kb-002', '检修案例库')],
          page: { page: 1, pageSize: 100, total: 2 },
          requestId: 'req-kb-page-1',
        }),
      ),
    )

    renderWithProviders(
      <ControlledSelector
        initialValue={['kb-001', 'kb-002']}
        variant="compact"
        onValueChange={(nextValue) => values.push(nextValue)}
      />,
    )

    await user.click(screen.getByRole('button', { name: /已选 2 个知识库/ }))
    await user.click(await screen.findByRole('checkbox', { name: '不选择知识库' }))

    await waitFor(() => expect(values.at(-1)).toEqual([]))
    expect(screen.getByRole('button', { name: /选择知识库/ })).toBeVisible()
  })

  it('keeps many compact knowledge bases inside a scrollable picker list', async () => {
    const user = userEvent.setup()
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>(async () =>
        jsonResponse({
          data: Array.from({ length: 12 }, (_, index) =>
            knowledgeBase(`kb-${String(index + 1).padStart(3, '0')}`, `知识库 ${index + 1}`),
          ),
          page: { page: 1, pageSize: 100, total: 12 },
          requestId: 'req-kb-page-1',
        }),
      ),
    )

    renderWithProviders(<ControlledSelector variant="compact" onValueChange={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: /选择知识库/ }))

    const list = await screen.findByTestId('knowledge-base-selector-list')
    expect(list).toHaveClass('max-h-64', 'overflow-y-auto')
    expect(within(list).getByRole('checkbox', { name: /知识库 12 \(kb-012\)/ })).toBeVisible()
  })

  it('can page beyond the first 100 knowledge bases and select later results', async () => {
    const user = userEvent.setup()
    const values: string[][] = []
    const requestedPages: string[] = []
    const fetchMock = vi.fn<typeof fetch>(async (input, init) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)
      requestedPages.push(`${url.searchParams.get('page')}:${url.searchParams.get('pageSize')}`)

      if (url.searchParams.get('page') === '2') {
        return jsonResponse({
          data: [knowledgeBase('kb-101', '第 101 个知识库')],
          page: { page: 2, pageSize: 100, total: 101 },
          requestId: 'req-kb-page-2',
        })
      }

      return jsonResponse({
        data: [knowledgeBase('kb-001', '第 1 个知识库')],
        page: { page: 1, pageSize: 100, total: 101 },
        requestId: 'req-kb-page-1',
      })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<ControlledSelector onValueChange={(value) => values.push(value)} />)

    expect(await screen.findByRole('button', { name: /第 1 个知识库/ })).toBeVisible()
    await user.click(screen.getByRole('button', { name: '下一页知识库' }))

    const secondPageKnowledgeBase = await screen.findByRole('button', { name: /第 101 个知识库/ })
    await user.click(secondPageKnowledgeBase)

    await waitFor(() => expect(values.at(-1)).toEqual(['kb-101']))
    expect(requestedPages).toContain('1:100')
    expect(requestedPages).toContain('2:100')
  })

  it('adds a knowledge base selected from the loaded Gateway list', async () => {
    const user = userEvent.setup()
    const values: string[][] = []
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>(async () =>
        jsonResponse({
          data: [
            knowledgeBase('kb-001', '第 1 个知识库'),
            knowledgeBase('kb-002', '第 2 个知识库'),
          ],
          page: { page: 1, pageSize: 100, total: 101 },
          requestId: 'req-kb-page-1',
        }),
      ),
    )

    renderWithProviders(<ControlledSelector onValueChange={(value) => values.push(value)} />)

    await screen.findByRole('button', { name: /第 1 个知识库/ })
    await user.click(screen.getByRole('combobox', { name: '知识库范围选择' }))
    await user.click(screen.getByRole('option', { name: /第 2 个知识库 \(kb-002\)/ }))
    await user.click(screen.getByRole('button', { name: '添加' }))

    await waitFor(() => expect(values.at(-1)).toEqual(['kb-002']))
    expect(screen.getByTitle('kb-002')).toHaveTextContent('第 2 个知识库')
    expect(screen.queryByLabelText('知识库范围ID')).not.toBeInTheDocument()
  })

  it('keeps manual id fallback for diagnostic retrieval', async () => {
    const user = userEvent.setup()
    const values: string[][] = []
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>(async () =>
        jsonResponse(
          {
            error: {
              code: 'dependency_error',
              message: 'knowledge service unavailable',
              requestId: 'req-kb-failed',
            },
          },
          { status: 502 },
        ),
      ),
    )

    renderWithProviders(<ControlledSelector onValueChange={(value) => values.push(value)} />)

    expect(await screen.findByText('知识库列表依赖失败')).toBeVisible()
    await user.type(screen.getByLabelText('知识库范围手动ID'), 'kb-known, kb-off-page')
    await user.click(screen.getByRole('button', { name: '添加ID' }))

    await waitFor(() => expect(values.at(-1)).toEqual(['kb-known', 'kb-off-page']))
    expect(screen.getByTitle('kb-known')).toBeVisible()
    expect(screen.getByTitle('kb-off-page')).toBeVisible()
  })

  it('keeps the search box editable when the knowledge base list fails', async () => {
    const user = userEvent.setup()
    vi.stubGlobal(
      'fetch',
      vi.fn<typeof fetch>(async () =>
        jsonResponse(
          {
            error: {
              code: 'dependency_error',
              message: 'knowledge service unavailable',
              requestId: 'req-kb-failed',
            },
          },
          { status: 502 },
        ),
      ),
    )

    renderWithProviders(<ControlledSelector onValueChange={vi.fn()} />)

    expect(await screen.findByText('知识库列表依赖失败')).toBeVisible()
    const searchInput = screen.getByLabelText('知识库范围搜索')
    expect(searchInput).not.toBeDisabled()

    await user.type(searchInput, '运行规程')

    expect(searchInput).toHaveValue('运行规程')
  })
})
