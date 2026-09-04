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
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, test, vi } from 'vitest'

import { copyToClipboard } from '@/lib/copy-to-clipboard'

import { SePayPaymentPanel } from '../components/sepay-payment-panel'
import type { SePayOrder } from '../types'

vi.mock('../api', () => ({
  getSePayOrderStatus: vi.fn().mockResolvedValue({
    success: true,
    data: { status: 'pending' },
  }),
}))

vi.mock('@/lib/copy-to-clipboard', () => ({
  copyToClipboard: vi.fn().mockResolvedValue(true),
}))

const mockCopy = vi.mocked(copyToClipboard)

const order: SePayOrder = {
  trade_no: 'SPABCDEF12345678',
  memo: 'SPABCDEF12345678',
  payable_vnd: 1234500,
  bank_account: '987654321',
  bank_code: 'niclop',
  account_holder: 'NGUYEN VAN A',
  vietqr_url: 'https://img.vietqr.io/image/niclop-987654321-compact2.png',
  create_time: 1700000000,
  expire_time: 1700000000 + 600,
  status: 'pending',
}

function renderPanel() {
  return render(
    <SePayPaymentPanel order={order} nowProvider={() => 1700000000} />
  )
}

describe('SePayPaymentPanel — required fields', () => {
  test('renders the VietQR image with the order vietqr_url as src', () => {
    renderPanel()

    const qr = screen.getByRole('img', { name: /vietqr payment code/i })
    expect(qr).toHaveAttribute('src', order.vietqr_url)
  })

  test('renders the payable VND amount', () => {
    renderPanel()

    expect(screen.getByText('Amount to transfer')).toBeInTheDocument()
    const formatted = new Intl.NumberFormat(undefined, {
      maximumFractionDigits: 0,
    }).format(order.payable_vnd)
    expect(screen.getByText(formatted)).toBeInTheDocument()
    expect(screen.getByText('VND')).toBeInTheDocument()
  })

  test('renders bank account, bank code, and account holder', () => {
    renderPanel()

    expect(screen.getByText('Account number')).toBeInTheDocument()
    expect(screen.getByText(order.bank_account)).toBeInTheDocument()
    expect(screen.getByText('Bank code')).toBeInTheDocument()
    expect(screen.getByText(order.bank_code)).toBeInTheDocument()
    expect(screen.getByText('Account holder')).toBeInTheDocument()
    expect(screen.getByText(order.account_holder)).toBeInTheDocument()
  })

  test('renders the memo with its own copy button', () => {
    renderPanel()

    expect(screen.getByText('Transfer memo (required)')).toBeInTheDocument()
    expect(screen.getByText(order.memo)).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: /copy transfer memo/i })
    ).toBeInTheDocument()
  })

  test('renders a copy button for the account number', () => {
    renderPanel()

    expect(
      screen.getByRole('button', { name: /copy account number/i })
    ).toBeInTheDocument()
  })

  test('renders the countdown to expiry', () => {
    renderPanel()

    // expire_time - nowProvider = 600s -> "10:00"
    expect(screen.getByText('10:00')).toBeInTheDocument()
  })

  test('renders the memo-required instruction', () => {
    renderPanel()

    expect(
      screen.getByText(/must use the transfer memo below as the transfer description/i)
    ).toBeInTheDocument()
  })

  test('copying the memo button is wired to the memo value', async () => {
    renderPanel()
    await userEvent.setup().click(
      screen.getByRole('button', { name: /copy transfer memo/i })
    )

    expect(mockCopy).toHaveBeenCalledWith(order.memo)
  })

  test('expired countdown shows 00:00 and the expired instruction', () => {
    render(
      <SePayPaymentPanel
        order={order}
        nowProvider={() => order.expire_time + 1}
      />
    )

    expect(screen.getByText('00:00')).toBeInTheDocument()
    expect(within(screen.getByRole('alert')).getByText(/has expired/i)).toBeInTheDocument()
  })
})
