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
import { useEffect, useRef, useState } from 'react'

import { SEPAY_POLL_INTERVAL_MS } from '../constants'
import { getSePayOrderStatus, isApiSuccess } from '../api'
import type { SePayOrderStatus } from '../types'

// ============================================================================
// SePay Order-Status Polling Hook
// ============================================================================

interface UseSePayOrderPollingArgs {
  /** Order being polled. When null (panel closed) polling stops. */
  tradeNo: string | null
  /** Called exactly once when polling observes a successful order. */
  onSuccess?: (status: SePayOrderStatus) => void
}

interface UseSePayOrderPollingReturn {
  /** Latest status the server reported, or null until the first response. */
  status: SePayOrderStatus | null
  /** Whether the panel is still actively polling (pending). */
  polling: boolean
}

/**
 * Poll `GET /api/user/sepay/order/:trade_no` at a fixed interval while the
 * panel is open. Stops on `success`, `expired`, or when the trade number is
 * cleared or the component unmounts — matching the D9 decision that the client
 * learns the outcome by polling, not by redirect.
 */
export function useSePayOrderPolling(
  args: UseSePayOrderPollingArgs
): UseSePayOrderPollingReturn {
  const tradeNo = args.tradeNo
  const onSuccess = args.onSuccess

  const [status, setStatus] = useState<SePayOrderStatus | null>(null)
  const stoppedRef = useRef(false)
  const firedSuccessRef = useRef(false)
  const onSuccessRef = useRef(onSuccess)
  onSuccessRef.current = onSuccess

  useEffect(() => {
    setStatus(null)
    stoppedRef.current = false
    firedSuccessRef.current = false

    if (!tradeNo) {
      return
    }

    let cancelled = false
    let timer: ReturnType<typeof setInterval> | null = null

    const tick = async () => {
      if (cancelled || stoppedRef.current) {
        return
      }
      try {
        const response = await getSePayOrderStatus(tradeNo)
        if (cancelled || stoppedRef.current) {
          return
        }
        if (!isApiSuccess(response) || !response.data?.status) {
          return
        }
        const next = response.data.status
        setStatus(next)
        if (next === 'success' || next === 'expired') {
          stoppedRef.current = true
          if (timer) {
            clearInterval(timer)
          }
          if (next === 'success' && !firedSuccessRef.current) {
            firedSuccessRef.current = true
            onSuccessRef.current?.(next)
          }
        }
      } catch {
        // Transient poll errors are ignored; the next tick retries.
      }
    }

    timer = setInterval(() => {
      void tick()
    }, SEPAY_POLL_INTERVAL_MS)
    // Kick off an immediate poll so the first status lands without waiting
    // a full interval.
    void tick()

    return () => {
      cancelled = true
      if (timer) {
        clearInterval(timer)
      }
    }
  }, [tradeNo])

  const polling = status !== 'success' && status !== 'expired'

  return { status, polling }
}
