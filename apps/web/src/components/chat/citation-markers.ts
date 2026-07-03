import type { QACitation } from '@/lib/types'

export type CitationMarkerToken =
  { kind: 'text'; text: string } | { citations: QACitation[]; kind: 'marker'; label: string }

type CitationMarker = Extract<CitationMarkerToken, { kind: 'marker' }>

const CITATION_MARKER_PATTERN = /\[(\d+(?:\s*,\s*\d+)*|\d+\s*-\s*\d+)\]/g

function citationNo(citation: QACitation): number | undefined {
  return typeof citation.citationNo === 'number' ? citation.citationNo : undefined
}

function citationDocumentId(citation: QACitation): string {
  return citation.documentId ?? citation.docId ?? ''
}

export function createCitationMap(citations: QACitation[]): Map<number, QACitation> {
  const map = new Map<number, QACitation>()
  for (const citation of citations) {
    const no = citationNo(citation)
    if (no != null) map.set(no, citation)
  }
  return map
}

function parseCitationNumbers(raw: string): number[] | undefined {
  if (raw.includes('-')) {
    const [startRaw, endRaw] = raw.split('-', 2)
    if (!startRaw || !endRaw) return undefined
    const start = Number(startRaw.trim())
    const end = Number(endRaw.trim())
    if (!Number.isInteger(start) || !Number.isInteger(end) || start <= 0 || end < start) {
      return undefined
    }
    return Array.from({ length: end - start + 1 }, (_, index) => start + index)
  }

  const numbers = raw.split(',').map((part) => Number(part.trim()))
  if (numbers.some((value) => !Number.isInteger(value) || value <= 0)) return undefined
  return numbers
}

function tokenFromMarker(
  rawNumbers: string,
  citationsByNo: Map<number, QACitation>,
): CitationMarker | undefined {
  const numbers = parseCitationNumbers(rawNumbers)
  if (!numbers || numbers.length === 0) return undefined

  const citations = numbers.map((number) => citationsByNo.get(number))
  if (citations.some((citation) => citation == null)) return undefined

  const label = rawNumbers.includes('-')
    ? `[${numbers[0]}-${numbers.at(-1)}]`
    : `[${numbers.join(',')}]`
  return { citations: citations as QACitation[], kind: 'marker' as const, label }
}

function canMergeSingleCitationMarkers(left: CitationMarker, right: CitationMarker) {
  if (left.citations.length < 1 || right.citations.length !== 1) return false

  const leftCitation = left.citations.at(-1)
  const firstCitation = left.citations[0]
  const rightCitation = right.citations[0]
  if (!leftCitation || !firstCitation || !rightCitation) return false

  const leftNo = citationNo(leftCitation)
  const rightNo = citationNo(rightCitation)
  const documentId = citationDocumentId(firstCitation)

  return (
    leftNo != null &&
    rightNo != null &&
    rightNo === leftNo + 1 &&
    documentId !== '' &&
    citationDocumentId(rightCitation) === documentId
  )
}

function mergeAdjacentCitationTokens(tokens: CitationMarkerToken[]): CitationMarkerToken[] {
  const merged: CitationMarkerToken[] = []

  for (const token of tokens) {
    const previous = merged[merged.length - 1]
    if (
      previous?.kind === 'marker' &&
      token.kind === 'marker' &&
      canMergeSingleCitationMarkers(previous, token)
    ) {
      const citations = [...previous.citations, ...token.citations]
      const firstCitation = citations[0]
      const lastCitation = citations.at(-1)
      const first = firstCitation ? citationNo(firstCitation) : undefined
      const last = lastCitation ? citationNo(lastCitation) : undefined
      merged[merged.length - 1] = {
        citations,
        kind: 'marker',
        label: first != null && last != null ? `[${first}-${last}]` : previous.label,
      }
    } else {
      merged.push(token)
    }
  }

  return merged
}

export function parseCitationMarkers(
  text: string,
  citationsByNo: Map<number, QACitation>,
): CitationMarkerToken[] {
  if (text.length === 0 || citationsByNo.size === 0) return [{ kind: 'text', text }]

  const tokens: CitationMarkerToken[] = []
  let lastIndex = 0

  for (const match of text.matchAll(CITATION_MARKER_PATTERN)) {
    const marker = match[0]
    const rawNumbers = match[1]
    const index = match.index ?? 0
    if (!rawNumbers) continue

    if (index > lastIndex) tokens.push({ kind: 'text', text: text.slice(lastIndex, index) })

    const markerToken = tokenFromMarker(rawNumbers, citationsByNo)
    tokens.push(markerToken ?? { kind: 'text', text: marker })
    lastIndex = index + marker.length
  }

  if (lastIndex < text.length) tokens.push({ kind: 'text', text: text.slice(lastIndex) })
  return mergeAdjacentCitationTokens(tokens)
}
