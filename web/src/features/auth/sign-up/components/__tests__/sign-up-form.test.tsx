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

import type { SystemStatus } from '@/features/auth/types'

import { SignUpForm } from '../sign-up-form'

// ---------------------------------------------------------------------------
// SignUpForm reaches a live status query, a router, and a toast bus. None of
// those are the behavior under test (whether the Telegram handle field is
// required per the status flag and rejects a malformed value before submit),
// so they are stubbed at the same boundaries used elsewhere in this repo
// (features/home/components/sections/__tests__/hero.test.tsx).
// ---------------------------------------------------------------------------

const mockRegister = vi.fn()
vi.mock('@/features/auth/api', () => ({
  register: (...args: unknown[]) => mockRegister(...args),
  wechatLoginByCode: vi.fn(),
  sendEmailVerification: vi.fn(),
}))

const mockUseStatus = vi.fn()
vi.mock('@/hooks/use-status', () => ({
  useStatus: () => mockUseStatus(),
}))

vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => vi.fn(),
}))

const mockToastError = vi.fn()
vi.mock('sonner', () => ({
  toast: {
    error: (...args: unknown[]) => mockToastError(...args),
    success: vi.fn(),
    info: vi.fn(),
  },
}))

function statusWithTelegramRequired(required: boolean): SystemStatus {
  return {
    telegram_handle_required: required,
    email_verification: false,
    user_agreement_enabled: false,
    privacy_policy_enabled: false,
    oauth_register_enabled: false,
    wechat_login: false,
    turnstile_check: false,
  }
}

function renderForm(required: boolean) {
  mockUseStatus.mockReturnValue({
    status: statusWithTelegramRequired(required),
    loading: false,
    error: null,
  })
  return render(<SignUpForm />)
}

async function fillRequiredAccountFields(handle?: string) {
  const user = userEvent.setup()
  await user.type(screen.getByLabelText('Username'), 'new-user')
  await user.type(screen.getByLabelText('Password'), 'password123')
  await user.type(screen.getByLabelText('Confirm password'), 'password123')
  if (handle !== undefined) {
    await user.type(screen.getByLabelText(/Telegram handle/), handle)
  }
  await user.click(screen.getByRole('button', { name: 'Create account' }))
}

describe('sign-up form — Telegram handle field', () => {
  afterEach(() => {
    mockRegister.mockReset()
    mockUseStatus.mockReset()
    mockToastError.mockReset()
  })

  test('labels the field as required and blocks submission when left blank', async () => {
    renderForm(true)

    expect(screen.getByText('Telegram handle')).toBeInTheDocument()

    await fillRequiredAccountFields()

    expect(mockToastError).toHaveBeenCalledWith(
      'Please enter your Telegram handle'
    )
    expect(mockRegister).not.toHaveBeenCalled()
  })

  test('labels the field as optional and submits when left blank', async () => {
    mockRegister.mockResolvedValue({ success: true })
    renderForm(false)

    expect(screen.getByText('Telegram handle (optional)')).toBeInTheDocument()

    await fillRequiredAccountFields()

    expect(mockRegister).toHaveBeenCalledTimes(1)
    expect(mockRegister.mock.calls[0][0]).toMatchObject({
      telegram_username: undefined,
    })
  })

  test('rejects a malformed handle before submitting', async () => {
    renderForm(false)

    await fillRequiredAccountFields('a1')

    expect(
      await screen.findByText(
        'Telegram handle must be 5-32 characters, start with a letter, and contain only letters, digits, and underscores'
      )
    ).toBeInTheDocument()
    expect(mockRegister).not.toHaveBeenCalled()
  })

  test('accepts a well-formed handle and submits it normalized', async () => {
    mockRegister.mockResolvedValue({ success: true })
    renderForm(true)

    await fillRequiredAccountFields('@John_Doe99')

    expect(mockRegister).toHaveBeenCalledTimes(1)
    expect(mockRegister.mock.calls[0][0]).toMatchObject({
      telegram_username: 'john_doe99',
    })
  })
})
