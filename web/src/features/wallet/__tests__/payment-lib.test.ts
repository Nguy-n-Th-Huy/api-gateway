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
import { describe, expect, test } from 'vitest'

import {
  calculatePayableVNDPreview,
  getMinTopupAmount,
  isSePayAvailable,
} from '../lib/payment'

// ---------------------------------------------------------------------------
// These tests recreate the meaningful single-provider contracts formerly
// covered by the deleted multi-gateway use-payment.test.ts / payment.test.ts:
// availability gating, minimum amount resolution, and the payable-Dong preview
// math (spec: "Currency conversion for the payable amount").
// ---------------------------------------------------------------------------

describe('isSePayAvailable', () => {
  test('null info is unavailable', () => {
    expect(isSePayAvailable(null)).toBe(false)
  })

  test('unconfirmed compliance is unavailable even when enabled', () => {
    expect(
      isSePayAvailable({
        min_topup: 1,
        amount_options: [],
        discount: {},
        enable_sepay_topup: true,
        payment_compliance_confirmed: false,
      })
    ).toBe(false)
  })

  test('enable_sepay_topup true makes SePay available', () => {
    expect(
      isSePayAvailable({
        min_topup: 1,
        amount_options: [],
        discount: {},
        enable_sepay_topup: true,
      })
    ).toBe(true)
  })

  test('legacy enable_online_topup flag also counts as available', () => {
    expect(
      isSePayAvailable({
        min_topup: 1,
        amount_options: [],
        discount: {},
        enable_online_topup: true,
      })
    ).toBe(true)
  })

  test('no flags means unavailable', () => {
    expect(
      isSePayAvailable({ min_topup: 1, amount_options: [], discount: {} })
    ).toBe(false)
  })
})

describe('getMinTopupAmount', () => {
  test('missing info falls back to the default minimum', () => {
    expect(getMinTopupAmount(null)).toBe(1)
  })

  test('sepay_min_topup takes precedence over the shared min_topup', () => {
    expect(
      getMinTopupAmount({
        min_topup: 10,
        sepay_min_topup: 50,
        amount_options: [],
        discount: {},
      })
    ).toBe(50)
  })

  test('falls back to shared min_topup when sepay-specific value is absent', () => {
    expect(
      getMinTopupAmount({ min_topup: 20, amount_options: [], discount: {} })
    ).toBe(20)
  })

  test('non-positive values fall back to the default', () => {
    expect(
      getMinTopupAmount({ min_topup: 0, amount_options: [], discount: {} })
    ).toBe(1)
    expect(
      getMinTopupAmount({
        min_topup: 10,
        sepay_min_topup: -5,
        amount_options: [],
        discount: {},
      })
    ).toBe(1)
  })
})

describe('calculatePayableVNDPreview', () => {
  test('default price: $10 at 1000 VND/USD is 10,000 Dong', () => {
    expect(calculatePayableVNDPreview({ amount: 10, price: 1000 })).toBe(10000)
  })

  test('discount multiplies the preview', () => {
    expect(
      calculatePayableVNDPreview({ amount: 100, price: 1000, discount: 0.9 })
    ).toBe(90000)
  })

  test('rounds to whole Dong', () => {
    expect(
      calculatePayableVNDPreview({ amount: 3, price: 1000.5 })
    ).toBe(3002) // 3 * 1000.5 = 3001.5 -> 3002
    expect(
      calculatePayableVNDPreview({ amount: 7, price: 1000, discount: 0.33 })
    ).toBe(2310) // 7 * 1000 * 0.33 = 2310
  })

  test('non-positive or non-finite inputs return null', () => {
    expect(calculatePayableVNDPreview({ amount: 0, price: 1000 })).toBeNull()
    expect(calculatePayableVNDPreview({ amount: -5, price: 1000 })).toBeNull()
    expect(calculatePayableVNDPreview({ amount: 10, price: 0 })).toBeNull()
    expect(calculatePayableVNDPreview({ amount: 10, price: -1 })).toBeNull()
    expect(
      calculatePayableVNDPreview({ amount: Number.NaN, price: 1000 })
    ).toBeNull()
    expect(
      calculatePayableVNDPreview({
        amount: Number.POSITIVE_INFINITY,
        price: 1000,
      })
    ).toBeNull()
  })

  test('out-of-range discounts are ignored rather than inflating the preview', () => {
    expect(
      calculatePayableVNDPreview({ amount: 10, price: 1000, discount: 1.5 })
    ).toBe(10000)
    expect(
      calculatePayableVNDPreview({ amount: 10, price: 1000, discount: 0 })
    ).toBe(10000)
    expect(
      calculatePayableVNDPreview({ amount: 10, price: 1000, discount: -0.5 })
    ).toBe(10000)
  })

  test('conversion that rounds to zero Dong returns null', () => {
    expect(
      calculatePayableVNDPreview({ amount: 0.0001, price: 1 })
    ).toBeNull()
  })
})
