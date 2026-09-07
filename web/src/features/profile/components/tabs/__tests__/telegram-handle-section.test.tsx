/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { TelegramHandleSection } from '../telegram-handle-section'

const mockUpdateUserProfile = vi.fn()
vi.mock('../../../api', () => ({
  updateUserProfile: (...args: unknown[]) => mockUpdateUserProfile(...args),
}))

const mockToastSuccess = vi.fn()
vi.mock('sonner', () => ({
  toast: {
    success: (...args: unknown[]) => mockToastSuccess(...args),
    error: vi.fn(),
  },
}))

describe('TelegramHandleSection', () => {
  afterEach(() => {
    mockUpdateUserProfile.mockReset()
    mockToastSuccess.mockReset()
  })

  test('rejects a malformed handle before calling the update API', async () => {
    const onUpdate = vi.fn()
    const user = userEvent.setup()
    render(<TelegramHandleSection onUpdate={onUpdate} />)

    await user.click(screen.getByRole('button', { name: 'Edit' }))
    await user.type(screen.getByPlaceholderText('e.g. john_doe99'), '1abc')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(
      await screen.findByText(
        'Telegram handle must be 5-32 characters, start with a letter, and contain only letters, digits, and underscores'
      )
    ).toBeInTheDocument()
    expect(mockUpdateUserProfile).not.toHaveBeenCalled()
    expect(onUpdate).not.toHaveBeenCalled()
  })

  test('saves a normalized, well-formed handle', async () => {
    mockUpdateUserProfile.mockResolvedValue({ success: true })
    const onUpdate = vi.fn()
    const user = userEvent.setup()
    render(<TelegramHandleSection onUpdate={onUpdate} />)

    await user.click(screen.getByRole('button', { name: 'Edit' }))
    await user.type(screen.getByPlaceholderText('e.g. john_doe99'), '@Valid_Handle1')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(mockUpdateUserProfile).toHaveBeenCalledWith({
      telegram_username: 'valid_handle1',
    })
    await screen.findByRole('button', { name: 'Edit' })
    expect(onUpdate).toHaveBeenCalledTimes(1)
  })

  test('clearing the field to empty is accepted as "no handle declared"', async () => {
    mockUpdateUserProfile.mockResolvedValue({ success: true })
    const onUpdate = vi.fn()
    const user = userEvent.setup()
    render(
      <TelegramHandleSection currentHandle='existing' onUpdate={onUpdate} />
    )

    await user.click(screen.getByRole('button', { name: 'Edit' }))
    const input = screen.getByDisplayValue('existing')
    await user.clear(input)
    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(mockUpdateUserProfile).toHaveBeenCalledWith({
      telegram_username: '',
    })
  })

  // Regression guard: a 200 response only means the request was accepted,
  // not that the clear actually persisted. The success toast and the "Not
  // set" display must both wait on the parent re-reading the handle from
  // the server (the onUpdate callback) and passing the confirmed value back
  // down as currentHandle, never on the component's own optimistic belief
  // that its submitted draft took effect.
  test('does not display the handle as cleared until the parent confirms it via onUpdate', async () => {
    mockUpdateUserProfile.mockResolvedValue({ success: true })
    const onUpdate = vi.fn()
    const user = userEvent.setup()
    const { rerender } = render(
      <TelegramHandleSection currentHandle='existing' onUpdate={onUpdate} />
    )

    await user.click(screen.getByRole('button', { name: 'Edit' }))
    const input = screen.getByDisplayValue('existing')
    await user.clear(input)
    await user.click(screen.getByRole('button', { name: 'Save' }))

    // The API call resolved with success, but the parent has not yet
    // re-supplied a confirmed currentHandle: the stale handle must still be
    // shown rather than an optimistic "Not set".
    await screen.findByRole('button', { name: 'Edit' })
    expect(screen.getByText('@existing')).toBeInTheDocument()
    expect(onUpdate).toHaveBeenCalledTimes(1)
    expect(mockToastSuccess).toHaveBeenCalledTimes(1)

    // Only once the parent re-renders with the server-confirmed value
    // (what onUpdate's refetch would produce for an actually-cleared
    // handle) does the display change.
    rerender(
      <TelegramHandleSection currentHandle={undefined} onUpdate={onUpdate} />
    )
    expect(screen.getByText('Not set')).toBeInTheDocument()
  })
})
