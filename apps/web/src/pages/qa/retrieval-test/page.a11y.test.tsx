import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { renderWithProviders } from '@/test/render'

import { QARetrievalTestPage } from './page'

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('QARetrievalTestPage accessibility smoke', () => {
  it('submits the retrieval form with labelled controls using only the keyboard', async () => {
    const keyboard = userEvent.setup()
    const submittedPayloads: unknown[] = []
    const fetchMock = vi.fn<typeof fetch>(async (input, init) => {
      const request = input instanceof Request ? input.clone() : new Request(input, init)
      const url = new URL(request.url)

      if (request.method === 'GET' && url.pathname.endsWith('/knowledge-bases')) {
        return jsonResponse({
          data: [
            {
              createdAt: '2026-07-02T08:00:00Z',
              description: 'A11Y test knowledge base',
              documentCount: 1,
              id: 'kb-a11y',
              name: 'A11Y 知识库',
              retrievalStrategy: { mode: 'semantic' },
              status: 'ready',
              updatedAt: '2026-07-02T08:00:00Z',
            },
          ],
          page: { page: 1, pageSize: 100, total: 1 },
          requestId: 'req-knowledge-bases',
        })
      }

      if (request.method === 'POST' && url.pathname.endsWith('/retrieval-test-runs')) {
        submittedPayloads.push(await request.json())
        return jsonResponse({
          data: {
            createdAt: '2026-07-02T09:00:00Z',
            finishedAt: '2026-07-02T09:00:01Z',
            id: 'retrieval-run-1',
            results: [],
            status: 'completed',
          },
          requestId: 'req-retrieval-create',
        })
      }

      if (
        request.method === 'GET' &&
        url.pathname.endsWith('/retrieval-test-runs/retrieval-run-1')
      ) {
        return jsonResponse({
          data: {
            createdAt: '2026-07-02T09:00:00Z',
            finishedAt: '2026-07-02T09:00:01Z',
            id: 'retrieval-run-1',
            results: [],
            status: 'completed',
          },
          requestId: 'req-retrieval-run',
        })
      }

      return new Response(JSON.stringify({ error: { code: 'unexpected_request' } }), {
        headers: { 'Content-Type': 'application/json' },
        status: 500,
      })
    })
    vi.stubGlobal('fetch', fetchMock)

    renderWithProviders(<QARetrievalTestPage />)

    const queryInput = screen.getByLabelText('Query')
    const knowledgeSearchInput = await screen.findByLabelText('知识库范围搜索')
    const knowledgeIdInput = screen.getByLabelText('知识库范围ID')
    const topKInput = screen.getByLabelText('Top K')
    const rerankCheckbox = screen.getByRole('checkbox', { name: /rerank/i })

    expect(queryInput).toHaveAccessibleName('Query')
    expect(knowledgeSearchInput).toHaveAccessibleName('知识库范围搜索')
    expect(knowledgeIdInput).toHaveAccessibleName('知识库范围ID')
    expect(topKInput).toHaveAccessibleName('Top K')
    expect(rerankCheckbox).toHaveAccessibleName(/rerank/i)

    await keyboard.tab()
    expect(queryInput).toHaveFocus()
    await keyboard.keyboard('transformer oil temperature')
    await keyboard.tab()
    expect(knowledgeSearchInput).toHaveFocus()
    await keyboard.keyboard('a11y')
    await keyboard.tab()
    expect(knowledgeIdInput).toHaveFocus()
    await keyboard.tab()
    const knowledgeButton = screen.getByRole('button', { name: /A11Y 知识库/ })
    expect(knowledgeButton).toHaveFocus()
    await keyboard.keyboard('{Enter}')
    expect(knowledgeButton).toHaveAttribute('aria-pressed', 'true')
    await keyboard.tab()
    expect(topKInput).toHaveFocus()
    await keyboard.tab()
    await keyboard.tab()
    expect(rerankCheckbox).toHaveFocus()
    await keyboard.keyboard(' ')
    expect(rerankCheckbox).not.toBeChecked()
    await keyboard.tab()
    await keyboard.tab()
    await keyboard.tab()

    const buttons = screen.getAllByRole('button')
    const submitButton = buttons[buttons.length - 1]
    expect(submitButton).toHaveFocus()
    await keyboard.keyboard('{Enter}')

    await waitFor(() => expect(submittedPayloads).toHaveLength(1))
    expect(submittedPayloads[0]).toMatchObject({
      knowledgeBaseIds: ['kb-a11y'],
      question: 'transformer oil temperature',
      retrieval: {
        enableRerank: false,
        topK: 5,
      },
    })
  })
})
