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

import { TelegramLinkCodeSection } from '../telegram-link-code-section'

const mockRedeemTelegramLinkCode = vi.fn()
vi.mock('../../../api', () => ({
  redeemTelegramLinkCode: (...args: unknown[]) =>
    mockRedeemTelegramLinkCode(...args),
}))

async function submitCode(code: string) {
  const user = userEvent.setup()
  await user.type(screen.getByLabelText('Have a link code from the bot?'), code)
  await user.click(screen.getByRole('button', { name: 'Link' }))
}

describe('TelegramLinkCodeSection — refusal messages', () => {
  afterEach(() => {
    mockRedeemTelegramLinkCode.mockReset()
  })

  test('blocks submission with a validation message when the code is empty', async () => {
    render(<TelegramLinkCodeSection onSuccess={vi.fn()} />)

    await userEvent.setup().click(screen.getByRole('button', { name: 'Link' }))

    expect(
      await screen.findByText('Please enter the link code')
    ).toBeInTheDocument()
    expect(mockRedeemTelegramLinkCode).not.toHaveBeenCalled()
  })

  test.each([
    ['unknown or expired code', 'This code is invalid or has expired'],
    [
      'identity already bound elsewhere',
      'This Telegram account is already linked to another account',
    ],
    [
      'account already linked',
      'Your account already has a linked Telegram account',
    ],
    ['account disabled', 'Your account is disabled or unavailable'],
    ['session no longer valid', 'Your session is no longer valid, please sign in again'],
  ])(
    'surfaces the server-provided message for %s distinctly',
    async (_case, serverMessage) => {
      mockRedeemTelegramLinkCode.mockResolvedValue({
        success: false,
        message: serverMessage,
      })
      const onSuccess = vi.fn()
      render(<TelegramLinkCodeSection onSuccess={onSuccess} />)

      await submitCode('AB3D9F2K')

      expect(await screen.findByText(serverMessage)).toBeInTheDocument()
      expect(onSuccess).not.toHaveBeenCalled()
    }
  )

  test('two different refusal reasons produce two different messages', async () => {
    mockRedeemTelegramLinkCode.mockResolvedValueOnce({
      success: false,
      message: 'This code is invalid or has expired',
    })
    render(<TelegramLinkCodeSection onSuccess={vi.fn()} />)

    await submitCode('AAAAAAAA')
    const firstMessage = await screen.findByText(
      'This code is invalid or has expired'
    )
    expect(firstMessage).toBeInTheDocument()

    mockRedeemTelegramLinkCode.mockResolvedValueOnce({
      success: false,
      message: 'Your account already has a linked Telegram account',
    })
    await submitCode('BBBBBBBB')

    expect(
      await screen.findByText(
        'Your account already has a linked Telegram account'
      )
    ).toBeInTheDocument()
    expect(
      screen.queryByText('This code is invalid or has expired')
    ).not.toBeInTheDocument()
  })

  test('reports success distinctly from a refusal and notifies the caller', async () => {
    mockRedeemTelegramLinkCode.mockResolvedValue({
      success: true,
      message: 'Telegram account linked',
    })
    const onSuccess = vi.fn()
    render(<TelegramLinkCodeSection onSuccess={onSuccess} />)

    await submitCode('AB3D9F2K')

    expect(
      await screen.findByText('Telegram account linked')
    ).toBeInTheDocument()
    expect(onSuccess).toHaveBeenCalledTimes(1)
  })
})
