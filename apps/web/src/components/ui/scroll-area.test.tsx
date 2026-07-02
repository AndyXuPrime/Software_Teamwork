import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { ScrollArea } from './scroll-area'

describe('ScrollArea', () => {
  it('applies viewport classes to the scrollable viewport', () => {
    const { container } = render(
      <ScrollArea viewportClassName="overscroll-y-contain">
        <div>Scrollable content</div>
      </ScrollArea>,
    )

    expect(container.querySelector('[data-slot="scroll-area-viewport"]')).toHaveClass(
      'overscroll-y-contain',
    )
  })
})
