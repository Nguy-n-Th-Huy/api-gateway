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
import type { PerfModelSummary } from '@/features/performance-metrics/types'

import { MODEL_STATUS_BAR_WINDOW, MODEL_STATUS_THRESHOLDS } from '../constants'

export type ModelHealthState = 'operational' | 'down' | 'no-data'
export type ModelBarColor = 'healthy' | 'degraded' | 'failing' | 'neutral'

export interface ModelStatusBar {
  /** `null` means the interval recorded no traffic. */
  successRate: number | null
  color: ModelBarColor
}

export interface ModelStatusEntry {
  modelName: string
  state: ModelHealthState
  /** `null` when the model has no metrics entry at all (`no-data`). */
  successRate: number | null
  bars: ModelStatusBar[]
}

/**
 * A model with no metrics entry (`summary` undefined) recorded no traffic in
 * the window — the summary endpoint omits models with zero requests entirely
 * (see `pkg/perf_metrics/metrics.go`'s `mergeModelTotals`), so absence from
 * the response, not a zero `success_rate`, is what "no traffic" means here.
 */
export function classifyModelState(
  summary: PerfModelSummary | undefined
): ModelHealthState {
  if (!summary) return 'no-data'
  return summary.success_rate >= MODEL_STATUS_THRESHOLDS.OPERATIONAL
    ? 'operational'
    : 'down'
}

/** Classifies one bar's colour. `null`/`undefined`/non-finite input is
 * treated as "no recorded traffic" and gets the neutral colour. */
export function classifyBarColor(rate: number | null | undefined): ModelBarColor {
  if (rate == null || !Number.isFinite(rate)) return 'neutral'
  if (rate >= MODEL_STATUS_THRESHOLDS.OPERATIONAL) return 'healthy'
  if (rate >= MODEL_STATUS_THRESHOLDS.DEGRADED) return 'degraded'
  return 'failing'
}

/**
 * Builds a fixed-size, oldest-to-newest window of interval rates. The
 * backend's `recent_success_rates` only contains intervals that had traffic
 * (missing intervals are dropped, not zero-filled), so any shortfall against
 * `windowSize` is padded with `null` ("no recorded traffic") on the oldest
 * side, keeping the most recent known rates rightmost.
 */
export function buildBarStrip(
  rates: number[] | undefined,
  windowSize: number = MODEL_STATUS_BAR_WINDOW
): (number | null)[] {
  const values = (rates ?? []).slice(-windowSize)
  const missing = Math.max(0, windowSize - values.length)
  const padding: null[] = Array.from({ length: missing }, () => null)
  return [...padding, ...values]
}

export function buildModelStatusEntry(
  modelName: string,
  summary: PerfModelSummary | undefined
): ModelStatusEntry {
  const bars = buildBarStrip(summary?.recent_success_rates).map((rate) => ({
    successRate: rate,
    color: classifyBarColor(rate),
  }))

  return {
    modelName,
    state: classifyModelState(summary),
    successRate: summary ? summary.success_rate : null,
    bars,
  }
}

/** Builds the display list for a fixed set of model names, looking each one
 * up in the metrics summary. A name absent from `summaries` is listed with
 * the `no-data` state rather than dropped. */
export function buildModelStatusList(
  modelNames: string[],
  summaries: PerfModelSummary[]
): ModelStatusEntry[] {
  const byName = new Map(summaries.map((s) => [s.model_name, s]))
  return modelNames.map((name) => buildModelStatusEntry(name, byName.get(name)))
}
