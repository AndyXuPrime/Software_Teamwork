import { requestJson } from './client'
import type { components } from './generated/gateway'

type QACitationDetail = components['schemas']['QACitationDetail']

export interface CitationDetail {
  chunk_id: string
  doc_id: string
  doc_name: string
  text: string
  context_before: string
  context_after: string
  page_number: number
  score: number
}

interface BatchCitationsRequest {
  chunk_ids: string[]
}

function toCitationDetail(citation: QACitationDetail): CitationDetail {
  return {
    chunk_id: citation.chunkId ?? '',
    doc_id: citation.documentId ?? citation.docId ?? '',
    doc_name: citation.documentName ?? citation.docName ?? '',
    text: citation.content ?? citation.text ?? citation.contentPreview ?? '',
    context_before: citation.context ?? '',
    context_after: '',
    page_number: citation.pageNumber ?? 0,
    score: citation.score ?? 0,
  }
}

export async function getCitation(chunkId: string): Promise<CitationDetail> {
  const citation = await requestJson<QACitationDetail>(`/citations/${encodeURIComponent(chunkId)}`)
  return toCitationDetail(citation)
}

export async function batchGetCitations(chunkIds: string[]): Promise<CitationDetail[]> {
  const citations = await requestJson<QACitationDetail[]>('/citation-lookups', {
    method: 'POST',
    body: {
      citationIds: chunkIds,
    } satisfies components['schemas']['CreateQACitationLookupRequest'],
  })
  return citations.map(toCitationDetail)
}

export type { BatchCitationsRequest }
