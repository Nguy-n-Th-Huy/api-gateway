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
import {
  DEFAULT_PRESET_MULTIPLIERS,
  DEFAULT_MIN_TOPUP,
  SEPAY_MAX_TOPUP,
} from '../constants'
import type { PresetAmount, SePayOrder, TopupInfo } from '../types'

// ============================================================================
// Payment Processing Functions (SePay — the single provider)
// ============================================================================

/**
 * Check if SePay top-up is offered (compliance confirmed + gateway available).
 */
export function isSePayAvailable(topupInfo: TopupInfo | null): boolean {
  if (!topupInfo) {
    return false
  }
  if (topupInfo.payment_compliance_confirmed === false) {
    return false
  }
  return topupInfo.enable_sepay_topup === true || topupInfo.enable_online_topup === true
}

/**
 * Get minimum topup amount from topup info.
 */
export function getMinTopupAmount(topupInfo: TopupInfo | null): number {
  if (!topupInfo) {
    return DEFAULT_MIN_TOPUP
  }

  const minTopup = topupInfo.sepay_min_topup ?? topupInfo.min_topup
  return minTopup && minTopup > 0 ? minTopup : DEFAULT_MIN_TOPUP
}

/**
 * Maximum topup amount accepted for a single order.
 */
export function getMaxTopupAmount(): number {
  return SEPAY_MAX_TOPUP
}

/**
 * Generate preset amounts based on minimum topup.
 */
export function generatePresetAmounts(minAmount: number): PresetAmount[] {
  return DEFAULT_PRESET_MULTIPLIERS.map((multiplier) => ({
    value: minAmount * multiplier,
  }))
}

/**
 * Merge custom preset amounts with discounts.
 */
export function mergePresetAmounts(
  amountOptions: number[],
  discounts: Record<number, number>
): PresetAmount[] {
  if (!amountOptions || amountOptions.length === 0) {
    return []
  }

  return amountOptions.map((amount) => ({
    value: amount,
    discount: discounts[amount] || 1.0,
  }))
}

/**
 * Live preview of payable Dong for `amount` USD of balance, using the same
 * displayed calculation the backend applies when inserting an order (spec
 * "Currency conversion" and controller `sePayPayMoneyFromDecimal`):
 * `payable_vnd = round(amount × price × topup_group_ratio × discount)`.
 *
 * `price` is the VND-per-USD price from `TopupInfo.price`. `discount` is the
 * tiered preset discount for `amount`, default 1.0.
 *
 * The user's top-up group ratio is not exposed to the client (it is an admin
 * setting keyed by group), so the preview omits it and the authoritative
 * payable amount is the one returned by order creation. Returns null when the
 * inputs cannot produce a positive whole-Dong amount.
 */
export function calculatePayableVNDPreview(params: {
  amount: number
  price: number
  discount?: number
}): number | null {
  const { amount, price } = params
  if (!Number.isFinite(amount) || amount <= 0) return null
  if (!Number.isFinite(price) || price <= 0) return null
  const discount =
    params.discount && params.discount > 0 && params.discount <= 1
      ? params.discount
      : 1
  const payable = Math.round(amount * price * discount)
  return payable > 0 ? payable : null
}

/**
 * Build the per-order VietQR image URL carrying the order's own payable amount
 * and memo.
 *
 * Mirrors the backend's `buildSePayVietQRURL`:
 * `https://img.vietqr.io/image/{BANK_CODE}-{ACCOUNT_NO}-compact2.png?amount={PAYABLE_VND}&addInfo={MEMO}&accountName={HOLDER}`.
 */
export function buildSePayVietQRURL(order: SePayOrder): string {
  if (order.vietqr_url) {
    return order.vietqr_url
  }
  const enc = encodeURIComponent
  const params = new URLSearchParams({
    amount: String(order.payable_vnd ?? 0),
    addInfo: order.memo ?? order.trade_no ?? '',
    accountName: order.account_holder ?? '',
  })
  return `https://img.vietqr.io/image/${enc(order.bank_code ?? '')}-${enc(order.bank_account ?? '')}-compact2.png?${params.toString()}`
}
