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
import { API_KEY_STATUSES } from '@/features/keys/constants'
import { formatQuota, formatTimestampToDate } from '@/lib/format'

import type { EffectiveKeyStatus, KeyCheckReport } from '../types'

/** Maps the report's effective status to the console's own status vocabulary
 * (`@/features/keys/constants`), so status reads identically everywhere. */
export function getStatusEntry(status: EffectiveKeyStatus) {
  return API_KEY_STATUSES[status]
}

export interface QuotaDisplay {
  unlimited: boolean
  totalGranted: string
  totalUsed: string
  totalAvailable: string
  /** Percentage of granted quota already used, `null` for an unlimited key
   * (no remaining-quota figure or usage share is shown for those). */
  usedSharePercent: number | null
}

export function getQuotaDisplay(report: KeyCheckReport): QuotaDisplay {
  const totalGranted = formatQuota(report.total_granted)
  const totalUsed = formatQuota(report.total_used)
  const totalAvailable = formatQuota(report.total_available)

  if (report.unlimited_quota) {
    return {
      unlimited: true,
      totalGranted,
      totalUsed,
      totalAvailable,
      usedSharePercent: null,
    }
  }

  const usedSharePercent =
    report.total_granted > 0
      ? (report.total_used / report.total_granted) * 100
      : 0

  return {
    unlimited: false,
    totalGranted,
    totalUsed,
    totalAvailable,
    usedSharePercent,
  }
}

/** `-1` and `0` both mean "never expires" (matches `formatTimestampToDate`'s
 * own treatment of those values, and `GetTokenUsage`'s expiry convention). */
export function isNeverExpires(expiresAt: number): boolean {
  return expiresAt === -1 || expiresAt === 0
}

export function getExpiryDisplay(expiresAt: number): string | null {
  return isNeverExpires(expiresAt) ? null : formatTimestampToDate(expiresAt)
}

export function getCreatedTimeDisplay(createdTime: number): string {
  return formatTimestampToDate(createdTime)
}

export function getAccessedTimeDisplay(accessedTime: number): string {
  return formatTimestampToDate(accessedTime)
}

export interface ModelRestrictionDisplay {
  allAllowed: boolean
  models: string[]
}

export function getModelRestrictionDisplay(
  report: KeyCheckReport
): ModelRestrictionDisplay {
  if (!report.model_limits_enabled) {
    return { allAllowed: true, models: [] }
  }
  return {
    allAllowed: false,
    models: Object.keys(report.model_limits).sort(),
  }
}
