import { describe, expect, it } from 'vitest'

import type { QACitation } from '@/lib/types'

import { createCitationMap, parseCitationMarkers } from './citation-markers'

function citation(citationNo: number, documentId = 'doc-1'): QACitation {
  return {
    citationNo,
    documentId,
    documentName: `Document ${documentId}`,
    id: `cite-${citationNo}`,
    messageId: 'msg-1',
    text: `quote ${citationNo}`,
  }
}

function markerLabels(text: string, citations: QACitation[]) {
  return parseCitationMarkers(text, createCitationMap(citations)).map((token) =>
    token.kind === 'marker' ? token.label : token.text,
  )
}

describe('citation marker parsing', () => {
  it('renders single and discrete citation markers', () => {
    expect(markerLabels('Answer [1] and [3]', [citation(1), citation(3)])).toEqual([
      'Answer ',
      '[1]',
      ' and ',
      '[3]',
    ])
  })

  it('keeps unmatched citation markers as plain text', () => {
    expect(markerLabels('Answer [99]', [citation(1)])).toEqual(['Answer ', '[99]'])
  })

  it('keeps comma markers grouped without merging skipped numbers', () => {
    expect(markerLabels('Answer [1,3,5]', [citation(1), citation(3), citation(5)])).toEqual([
      'Answer ',
      '[1,3,5]',
    ])
  })

  it('expands explicit range markers when every citation exists', () => {
    expect(markerLabels('Answer [1-3]', [citation(1), citation(2), citation(3)])).toEqual([
      'Answer ',
      '[1-3]',
    ])
  })

  it('merges adjacent consecutive markers from the same document', () => {
    expect(markerLabels('Answer [1][2][3]', [citation(1), citation(2), citation(3)])).toEqual([
      'Answer ',
      '[1-3]',
    ])
  })

  it('does not merge adjacent markers from different documents', () => {
    expect(markerLabels('Answer [1][2]', [citation(1, 'doc-1'), citation(2, 'doc-2')])).toEqual([
      'Answer ',
      '[1]',
      '[2]',
    ])
  })
})
