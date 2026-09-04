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
import i18next from 'i18next'
import { useCallback, useState } from 'react'
import { toast } from 'sonner'

import { requestSePayTopUp, isApiSuccess } from '../api'
import { isSePayAvailable } from '../lib'
import type { SePayOrder, TopupInfo } from '../types'

// ============================================================================
// Payment Hook (SePay — the single provider)
// ============================================================================

export interface UsePaymentReturn {
  /** The pending SePay order, or null before creation / after clear. */
  order: SePayOrder | null
  /** Whether an order-creation request is in flight. */
  processing: boolean
  /**
   * Validate the amount client-side and create a pending SePay top-up order.
   * Returns the created order, or null on validation failure or API error.
   */
  createOrder: (amount: number) => Promise<SePayOrder | null>
  /** Clear the held order (e.g. after a settled panel is dismissed). */
  clearOrder: () => void
}

/**
 * Manages the SePay top-up order lifecycle.
 *
 * The amount itself is owned by the caller (the wallet page) so a single
 * source of truth drives both the live payable preview and order creation.
 * This hook performs the client-side guards (availability + positive integer
 * amount) and then calls the backend, which independently re-validates the
 * amount against the configured minimum, the per-order bound, and the wallet
 * capacity limit.
 */
export function usePayment(topupInfo: TopupInfo | null): UsePaymentReturn {
  const [order, setOrder] = useState<SePayOrder | null>(null)
  const [processing, setProcessing] = useState(false)

  const clearOrder = useCallback(() => setOrder(null), [])

  const createOrder = useCallback(
    async (amount: number): Promise<SePayOrder | null> => {
      if (topupInfo && !isSePayAvailable(topupInfo)) {
        toast.error(i18next.t('Online top-up is not available'))
        return null
      }
      if (!Number.isInteger(amount) || amount <= 0) {
        toast.error(i18next.t('Please enter a valid whole-number amount'))
        return null
      }

      setProcessing(true)
      try {
        const response = await requestSePayTopUp(amount)
        if (!isApiSuccess(response) || !response.data) {
          const message =
            response.message && String(response.message).trim().length > 0
              ? String(response.message)
              : i18next.t('Failed to create payment order')
          toast.error(message)
          return null
        }
        setOrder(response.data)
        return response.data
      } catch {
        toast.error(i18next.t('Failed to create payment order'))
        return null
      } finally {
        setProcessing(false)
      }
    },
    [topupInfo]
  )

  return { order, processing, createOrder, clearOrder }
}
