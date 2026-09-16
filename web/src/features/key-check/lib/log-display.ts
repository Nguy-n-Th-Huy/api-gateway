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
import { LOG_TYPES } from '@/features/usage-logs/constants'
import {
  formatLogQuota,
  formatTimestampToDate,
  formatTokens,
  formatUseTime,
} from '@/lib/format'

import type { KeyUsageLogEntry } from '../types'

/**
 * The console's own log-type vocabulary, looked up through its constants
 * module: that module's only imports are types, so the public page does not
 * pull the console's log API and table code into its bundle the way importing
 * `@/features/usage-logs/lib/utils` would.
 */
export function getLogTypeLabel(type: number): string {
  const config = LOG_TYPES.find((entry) => entry.value === type) ?? LOG_TYPES[0]
  return config.label
}

/** Prompt / completion tokens as shown on the console's own log table; a
 * request that recorded no token count shows a single dash. */
export function getUsageLogTokens(entry: KeyUsageLogEntry): string {
  if (entry.prompt_tokens === 0 && entry.completion_tokens === 0) return '-'
  return `${formatTokens(entry.prompt_tokens)} / ${formatTokens(entry.completion_tokens)}`
}

export interface UsageLogRow {
  /** Row identity for rendering; the public payload carries no row id. */
  id: string
  time: string
  /** An i18n key — the caller translates it. */
  typeLabel: string
  modelName: string
  tokens: string
  cost: string
  duration: string
}

/** One table row per entry, in the order the entries arrived (newest first). */
export function getUsageLogRows(entries: KeyUsageLogEntry[]): UsageLogRow[] {
  return entries.map((entry) => ({
    id: `${entry.created_at}-${entry.request_id}`,
    time: formatTimestampToDate(entry.created_at),
    typeLabel: getLogTypeLabel(entry.type),
    modelName: entry.model_name || '-',
    tokens: getUsageLogTokens(entry),
    cost: formatLogQuota(entry.quota),
    duration: entry.use_time > 0 ? formatUseTime(entry.use_time) : '-',
  }))
}

/** Total pages for a result set, never below one so a pager always has a
 * valid current page. */
export function getUsageLogPageCount(total: number, pageSize: number): number {
  if (pageSize <= 0) return 1
  return Math.max(1, Math.ceil(total / pageSize))
}
