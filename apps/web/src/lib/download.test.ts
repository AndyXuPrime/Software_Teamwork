import { describe, expect, it, vi } from 'vitest'

import { downloadFromUrl } from './download'

describe('downloadFromUrl', () => {
  it('creates a temporary anchor and triggers a browser download', () => {
    const click = vi.fn()
    const createElement = vi.spyOn(document, 'createElement')

    createElement.mockImplementation((tagName: string) => {
      const element = document.createElementNS('http://www.w3.org/1999/xhtml', tagName)
      if (tagName === 'a') {
        Object.defineProperty(element, 'click', { configurable: true, value: click })
      }
      return element as HTMLElement
    })

    downloadFromUrl('/api/v1/report-files/file-1/content', 'report.docx')

    const anchor = createElement.mock.results[0]?.value as HTMLAnchorElement
    expect(anchor.href).toContain('/api/v1/report-files/file-1/content')
    expect(anchor.download).toBe('report.docx')
    expect(click).toHaveBeenCalledTimes(1)
  })
})
