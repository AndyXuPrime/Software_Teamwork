import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { Button } from './button'

describe('Button', () => {
  it('renders an accessible disabled command button', () => {
    render(<Button disabled>保存配置</Button>)

    const button = screen.getByRole('button', { name: '保存配置' })
    expect(button).toBeDisabled()
  })
})
