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
import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'

import { getPerfMetricsSummary } from '@/features/performance-metrics/api'

import { buildModelStatusList, type ModelStatusEntry } from '../lib/model-status'

export interface UseModelStatusResult {
  entries: ModelStatusEntry[]
  isLoading: boolean
  isError: boolean
  refetch: () => void
}

/**
 * Loads `GET /api/perf-metrics/summary` independently of the key-check
 * form, so a metrics failure never blocks the lookup — see
 * specs/public-model-status/spec.md.
 *
 * @param availableModels `null` before any key is checked (list every model
 * the summary reports); the checked key's `available_models` afterwards, so
 * the section narrows to the key's group and a group model with no metrics
 * entry still appears as `No Data`.
 */
export function useModelStatus(
  availableModels: string[] | null
): UseModelStatusResult {
  const query = useQuery({
    queryKey: ['key-check', 'perf-metrics-summary'],
    queryFn: () => getPerfMetricsSummary(24),
  })

  const summaries = useMemo(
    () => query.data?.data?.models ?? [],
    [query.data]
  )

  const entries = useMemo(() => {
    const modelNames = availableModels ?? summaries.map((s) => s.model_name)
    return buildModelStatusList(modelNames, summaries)
  }, [availableModels, summaries])

  return {
    entries,
    isLoading: query.isLoading,
    isError: query.isError,
    refetch: () => {
      void query.refetch()
    },
  }
}
