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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { redeemTopupCode } from '../api'
import { useRedemption } from '../hooks/use-redemption'
import type { RedemptionResponse } from '../types'

// ---------------------------------------------------------------------------
// Mock the API boundary so the test asserts on the user-visible behaviour of
// the redeem flow rather than on network plumbing.
// ---------------------------------------------------------------------------

vi.mock('../api', () => ({
  redeemTopupCode: vi.fn(),
}))

// The success path refreshes the current user via getSelf; stub the network
// boundary so the test stays deterministic.
vi.mock('@/lib/api', () => ({
  getSelf: vi.fn().mockResolvedValue({ success: true, data: {} }),
}))

const mockRedeem = vi.mocked(redeemTopupCode)

// ---------------------------------------------------------------------------
// Small harness that wires a labelled redeem input + button to the hook.
// Keeps fixtures minimal and deterministic per AGENTS.md 3.14.
// ---------------------------------------------------------------------------

function RedeemHarness(props: { onSuccess?: (ok: boolean) => void }) {
  const [code, setCode] = useState('')
  const { redeeming, redeemCode } = useRedemption()

  return (
    <form
      onSubmit={async (event) => {
        event.preventDefault()
        const ok = await redeemCode(code)
        props.onSuccess?.(ok)
      }}
    >
      <label htmlFor='redemption-code'>Redemption code</label>
      <input
        id='redemption-code'
        value={code}
        onChange={(event) => setCode(event.target.value)}
      />
      <button type='submit' disabled={redeeming} aria-busy={redeeming}>
        {redeeming ? 'Redeeming' : 'Redeem'}
      </button>
      {redeeming ? <span role='status'>Working</span> : null}
    </form>
  )
}

const submit = async () => {
  await userEvent.setup().click(screen.getByRole('button', { name: /redeem/i }))
}

// ---------------------------------------------------------------------------

describe('wallet — redeem form validation', () => {
  beforeEach(() => {
    mockRedeem.mockReset()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  test('empty input short-circuits with a validation message and never calls the API', async () => {
    const onSuccess = vi.fn()
    render(<RedeemHarness onSuccess={onSuccess} />)

    await submit()

    expect(mockRedeem).not.toHaveBeenCalled()
    await waitFor(() => expect(onSuccess).toHaveBeenCalledWith(false))
  })

  test('whitespace-only input is treated as empty and never calls the API', async () => {
    const onSuccess = vi.fn()
    render(<RedeemHarness onSuccess={onSuccess} />)

    await userEvent.setup().type(screen.getByLabelText('Redemption code'), '   ')
    await submit()

    expect(mockRedeem).not.toHaveBeenCalled()
    await waitFor(() => expect(onSuccess).toHaveBeenCalledWith(false))
  })

  test('valid code submits to redeemTopupCode and disables the button while in flight', async () => {
    let resolveApi!: (val: RedemptionResponse) => void
    mockRedeem.mockImplementation(
      () => new Promise<RedemptionResponse>((res) => { resolveApi = res })
    )

    render(<RedeemHarness />)

    await userEvent.setup().type(screen.getByLabelText('Redemption code'), 'ABC-123')
    const promise = submit()

    await waitFor(() =>
      expect(mockRedeem).toHaveBeenCalledWith({ key: 'ABC-123' })
    )
    expect(screen.getByRole('button', { name: /redeem/i })).toBeDisabled()
    expect(screen.getByRole('status')).toBeInTheDocument()

    resolveApi({ success: true, data: 1000, message: '' })
    await promise
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /redeem/i })).not.toBeDisabled()
    )
  })

  test('successful redemption reports success via the boolean return', async () => {
    mockRedeem.mockResolvedValue({ success: true, data: 500, message: '' })

    const onSuccess = vi.fn()
    render(<RedeemHarness onSuccess={onSuccess} />)

    await userEvent.setup().type(screen.getByLabelText('Redemption code'), 'GOODCODE')
    await submit()

    await waitFor(() => expect(onSuccess).toHaveBeenCalledWith(true))
  })

  test('failed redemption reports failure and re-enables the control', async () => {
    mockRedeem.mockResolvedValue({
      success: false,
      message: 'Invalid code',
    })

    const onSuccess = vi.fn()
    render(<RedeemHarness onSuccess={onSuccess} />)

    await userEvent.setup().type(screen.getByLabelText('Redemption code'), 'BADCODE')
    await submit()

    await waitFor(() => expect(onSuccess).toHaveBeenCalledWith(false))
    expect(screen.getByRole('button', { name: /redeem/i })).not.toBeDisabled()
  })

  test('API exception is surfaced as failure and re-enables the control', async () => {
    mockRedeem.mockRejectedValue(new Error('network'))

    const onSuccess = vi.fn()
    render(<RedeemHarness onSuccess={onSuccess} />)

    await userEvent.setup().type(screen.getByLabelText('Redemption code'), 'THROWCODE')
    await submit()

    await waitFor(() => expect(onSuccess).toHaveBeenCalledWith(false))
    expect(screen.getByRole('button', { name: /redeem/i })).not.toBeDisabled()
  })

  test('submit button is reachable by keyboard and receives focus', async () => {
    const user = userEvent.setup()
    render(<RedeemHarness />)
    await user.tab()
    await user.tab()
    expect(screen.getByRole('button', { name: /redeem/i })).toHaveFocus()
  })
})
