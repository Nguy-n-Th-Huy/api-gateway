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
// ============================================================================
// Wallet Type Definitions
// ============================================================================

/**
 * Generic API response
 */
export interface ApiResponse<T = unknown> {
  success?: boolean
  message?: string
  data?: T
}

/**
 * SePay order lifecycle status. Mirrors the backend top-up statuses that the
 * order-status endpoint reports.
 */
export type SePayOrderStatus = 'pending' | 'success' | 'expired'

/**
 * SePay order payload returned by both order creation and order status.
 *
 * The creation endpoints (POST /api/user/sepay/pay and
 * POST /api/subscription/sepay/pay) return a pending order; the status
 * endpoint (GET /api/user/sepay/order/:trade_no) returns the same shape with
 * `status` reflecting the current lifecycle state. The payable amount and the
 * transfer memo are frozen at order creation.
 */
export interface SePayOrder {
  /** Unique trade number; also the transfer memo the user must include */
  trade_no: string
  /** Transfer memo (equals trade_no) required as the bank transfer note */
  memo: string
  /** Payable amount in whole Vietnamese Dong, frozen at creation */
  payable_vnd: number
  /** Destination bank account number */
  bank_account: string
  /** Destination bank code used to build the VietQR image */
  bank_code: string
  /** Destination account holder name */
  account_holder: string
  /** Per-order VietQR image URL carrying amount and memo */
  vietqr_url: string
  /** Creation timestamp (unix seconds) */
  create_time: number
  /** Expiry timestamp (unix seconds) */
  expire_time: number
  /** Current order status */
  status?: SePayOrderStatus
  /** Credited/display money amount */
  money?: number
}

/**
 * SePay top-up order creation request. The subscription endpoint instead
 * takes a plan id.
 */
export interface SePayPaymentRequest {
  amount: number
}

export type SePayPaymentResponse = ApiResponse<SePayOrder>
export type SePayOrderStatusResponse = ApiResponse<SePayOrder>

/**
 * Topup configuration information reported by the top-up info endpoint. After
 * the consolidation onto SePay only the SePay availability/settings plus the
 * shared pricing settings remain.
 */
export interface TopupInfo {
  /** Whether online (SePay) top-up is enabled */
  enable_online_topup?: boolean
  /** Whether SePay bank-transfer top-up is available */
  enable_sepay_topup?: boolean
  /** Destination bank account number for SePay transfers */
  sepay_bank_account?: string
  /** Destination bank code for SePay transfers */
  sepay_bank_code?: string
  /** Destination account holder name for SePay transfers */
  sepay_account_holder?: string
  /** SePay-specific minimum top-up amount */
  sepay_min_topup?: number
  /** How long a pending SePay order stays payable, in minutes */
  sepay_order_expiry_minutes?: number
  /** Minimum top-up amount */
  min_topup: number
  /** Local-currency (VND) price per one USD of credited balance */
  price?: number
  /** Preset amount options */
  amount_options: number[]
  /** Discount rates by amount */
  discount: Record<number, number>
  /** Optional topup link for purchasing codes */
  topup_link?: string
  /** Whether redemption code usage is enabled */
  enable_redemption?: boolean
  /** Whether compliance confirmation has been completed */
  payment_compliance_confirmed?: boolean
  /** Current compliance terms version */
  payment_compliance_terms_version?: string
}

/**
 * Standard API response types
 */
export type TopupInfoResponse = ApiResponse<TopupInfo>
export type RedemptionResponse = ApiResponse<number>
export type AffiliateCodeResponse = ApiResponse<string>
export type AffiliateTransferResponse = ApiResponse

/**
 * Preset amount option with optional discount
 */
export interface PresetAmount {
  /** Preset amount value */
  value: number
  /** Optional discount rate (0-1) */
  discount?: number
}

/**
 * Redemption code request
 */
export interface RedemptionRequest {
  /** Redemption code key */
  key: string
}

/**
 * Affiliate quota transfer request
 */
export interface AffiliateTransferRequest {
  /** Quota amount to transfer */
  quota: number
}

/**
 * User wallet data
 */
export interface UserWalletData {
  /** User ID */
  id: number
  /** Username */
  username: string
  /** Current quota balance */
  quota: number
  /** Total used quota */
  used_quota: number
  /** Total request count */
  request_count: number
  /** Affiliate quota (pending rewards) */
  aff_quota: number
  /** Total affiliate quota earned (historical) */
  aff_history_quota: number
  /** Number of successful affiliate invites */
  aff_count: number
  /** User group */
  group: string
}

/**
 * Topup record status
 */
export type TopupStatus = 'success' | 'pending' | 'expired'

/**
 * Topup billing record
 */
export interface TopupRecord {
  /** Record ID */
  id: number
  /** User ID */
  user_id: number
  /** Topup amount (quota) */
  amount: number
  /** Payment amount (actual money paid) */
  money: number
  /** Trade/order number */
  trade_no: string
  /** Payment method type */
  payment_method: string
  /** Creation timestamp */
  create_time: number
  /** Completion timestamp */
  complete_time?: number
  /** Payment status */
  status: TopupStatus
}

/**
 * Billing history response
 */
export interface BillingHistoryResponse {
  items: TopupRecord[]
  total: number
}

/**
 * Complete order request (admin only)
 */
export interface CompleteOrderRequest {
  trade_no: string
}
