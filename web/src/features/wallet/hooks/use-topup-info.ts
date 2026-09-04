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
import { useState, useEffect, useCallback } from 'react'

import { getTopupInfo } from '../api'
import {
  generatePresetAmounts,
  getMinTopupAmount,
  isSePayAvailable,
  mergePresetAmounts,
} from '../lib'
import type { PresetAmount, TopupInfo } from '../types'

// ============================================================================
// Topup Info Hook (SePay — the single provider)
// ============================================================================

function parseAmountOptions(data: unknown): number[] {
  if (!Array.isArray(data)) {
    return []
  }
  return data.map((item) => Number(item)).filter((item) => Number.isFinite(item) && item > 0)
}

function parseDiscountMap(data: unknown): Record<number, number> {
  if (!data || typeof data !== 'object' || Array.isArray(data)) {
    return {}
  }

  return Object.entries(data as Record<string, unknown>).reduce<
    Record<number, number>
  >((result, [key, value]) => {
    const numericKey = Number(key)
    const numericValue = Number(value)

    if (Number.isFinite(numericKey) && Number.isFinite(numericValue)) {
      result[numericKey] = numericValue
    }

    return result
  }, {})
}

function processTopupInfo(raw: TopupInfo): TopupInfo {
  return {
    ...raw,
    amount_options: parseAmountOptions(raw.amount_options),
    discount: parseDiscountMap(raw.discount),
  }
}

export function useTopupInfo() {
  const [topupInfo, setTopupInfo] = useState<TopupInfo | null>(null)
  const [presetAmounts, setPresetAmounts] = useState<PresetAmount[]>([])
  const [loading, setLoading] = useState(true)

  const fetchTopupInfo = useCallback(async () => {
    try {
      setLoading(true)

      const response = await getTopupInfo()

      if (!response.success || !response.data) {
        return
      }

      const processedData = processTopupInfo(response.data)
      setTopupInfo(processedData)

      if (processedData.amount_options.length > 0) {
        setPresetAmounts(
          mergePresetAmounts(
            processedData.amount_options,
            processedData.discount || {}
          )
        )
      } else {
        setPresetAmounts(generatePresetAmounts(getMinTopupAmount(processedData)))
      }
    } catch {
      // Leave the previous state in place on a failed fetch.
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void fetchTopupInfo()
  }, [fetchTopupInfo])

  return {
    topupInfo,
    presetAmounts,
    loading,
    sepayAvailable: isSePayAvailable(topupInfo),
    refetch: fetchTopupInfo,
  }
}
