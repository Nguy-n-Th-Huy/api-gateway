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
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { OAuthSection } from '../oauth-section'

// ---------------------------------------------------------------------------
// OAuthSection persists through a react-query mutation and reads/blocks
// TanStack Router navigation for its unsaved-changes guard. Neither is the
// behavior under test (that a bot credential field never re-displays the
// value it just saved), so both are stubbed at the same module boundaries
// used elsewhere in this repo for router-dependent components
// (features/home/components/sections/__tests__/hero.test.tsx).
// ---------------------------------------------------------------------------

const mockMutateAsync = vi.fn()
vi.mock('../../hooks/use-update-option', () => ({
  useUpdateOption: () => ({
    mutateAsync: (...args: unknown[]) => mockMutateAsync(...args),
    isPending: false,
  }),
}))

vi.mock('@tanstack/react-router', () => ({
  useBlocker: () => ({ status: 'idle' }),
}))

const baseDefaultValues = {
  GitHubOAuthEnabled: false,
  GitHubClientId: '',
  GitHubClientSecret: '',
  GoogleOAuthEnabled: false,
  GoogleClientId: '',
  GoogleClientSecret: '',
  'discord.enabled': false,
  'discord.client_id': '',
  'discord.client_secret': '',
  'oidc.enabled': false,
  'oidc.display_name': '',
  'oidc.client_id': '',
  'oidc.client_secret': '',
  'oidc.well_known': '',
  'oidc.authorization_endpoint': '',
  'oidc.token_endpoint': '',
  'oidc.user_info_endpoint': '',
  TelegramOAuthEnabled: false,
  TelegramBotToken: '',
  TelegramBotName: '',
  TelegramBotIntegrationEnabled: true,
  TelegramBotServiceKey: '',
  TelegramBotCallbackURL: '',
  TelegramBotCallbackSecret: '',
  TelegramHandleRequired: false,
  LinuxDOOAuthEnabled: false,
  LinuxDOClientId: '',
  LinuxDOClientSecret: '',
  LinuxDOMinimumTrustLevel: '',
  WeChatAuthEnabled: false,
  WeChatServerAddress: '',
  WeChatServerToken: '',
  WeChatAccountQRCodeImageURL: '',
}

function renderTelegramTab() {
  const utils = render(
    <OAuthSection
      defaultValues={baseDefaultValues}
      serverAddress='https://gateway.example.com'
    />
  )
  fireEvent.click(screen.getByRole('tab', { name: 'Telegram' }))
  return utils
}

function submitForm(container: HTMLElement) {
  const form = container.querySelector('form')
  if (!form) throw new Error('Expected the settings form to be present')
  fireEvent.submit(form)
}

describe('OAuthSection — bot integration credential fields', () => {
  afterEach(() => {
    mockMutateAsync.mockReset()
  })

  test('never re-displays the bot service key or callback secret after a save', async () => {
    mockMutateAsync.mockResolvedValue({ success: true })
    const { container } = renderTelegramTab()
    const user = userEvent.setup()

    const serviceKeyInput = screen.getByLabelText('Bot Service Key')
    const callbackSecretInput = screen.getByLabelText('Event Callback Secret')

    await user.type(serviceKeyInput, 'brand-new-service-key')
    await user.type(callbackSecretInput, 'brand-new-callback-secret')

    submitForm(container)

    await waitFor(() => {
      expect(mockMutateAsync).toHaveBeenCalledWith({
        key: 'TelegramBotServiceKey',
        value: 'brand-new-service-key',
      })
    })
    await waitFor(() => {
      expect(mockMutateAsync).toHaveBeenCalledWith({
        key: 'TelegramBotCallbackSecret',
        value: 'brand-new-callback-secret',
      })
    })

    await waitFor(() => {
      expect(screen.getByLabelText('Bot Service Key')).toHaveValue('')
    })
    expect(screen.getByLabelText('Event Callback Secret')).toHaveValue('')
  })

  test('leaving the credential fields blank keeps the stored value unchanged', async () => {
    mockMutateAsync.mockResolvedValue({ success: true })
    const { container } = renderTelegramTab()

    // Change an unrelated field so the form has something to save, without
    // touching either write-only credential.
    fireEvent.click(
      screen.getByRole('switch', {
        name: 'Require Telegram Handle at Registration',
      })
    )

    submitForm(container)

    await waitFor(() => {
      expect(mockMutateAsync).toHaveBeenCalledWith({
        key: 'TelegramHandleRequired',
        value: true,
      })
    })
    expect(mockMutateAsync).not.toHaveBeenCalledWith(
      expect.objectContaining({ key: 'TelegramBotServiceKey' })
    )
    expect(mockMutateAsync).not.toHaveBeenCalledWith(
      expect.objectContaining({ key: 'TelegramBotCallbackSecret' })
    )
  })
})
