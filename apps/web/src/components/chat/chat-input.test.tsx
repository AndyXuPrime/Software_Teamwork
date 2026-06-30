import { fireEvent, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { renderWithProviders } from '@/test/render'

import ChatInput from './chat-input'

describe('ChatInput', () => {
  it('sends trimmed text and clears the draft', () => {
    const onSend = vi.fn()
    const onChange = vi.fn()

    renderWithProviders(
      <ChatInput onSend={onSend} disabled={false} value="  变压器巡检  " onChange={onChange} />,
    )

    fireEvent.click(screen.getByRole('button'))

    expect(onSend).toHaveBeenCalledWith('变压器巡检')
    expect(onChange).toHaveBeenCalledWith('')
  })

  it('keeps disabled or blank drafts from sending', () => {
    const onSend = vi.fn()
    const { rerender } = renderWithProviders(
      <ChatInput onSend={onSend} disabled={false} value="   " onChange={vi.fn()} />,
    )

    fireEvent.click(screen.getByRole('button'))
    expect(onSend).not.toHaveBeenCalled()

    rerender(<ChatInput onSend={onSend} disabled value="hello" onChange={vi.fn()} />)
    fireEvent.click(screen.getByRole('button'))
    expect(onSend).not.toHaveBeenCalled()
  })
})
