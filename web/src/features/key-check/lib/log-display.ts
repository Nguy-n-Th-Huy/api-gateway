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
import type { StatusBadgeProps } from '@/components/status-badge'
import { LOG_TYPES, LOG_TYPE_ENUM } from '@/features/usage-logs/constants'
import {
  formatLogQuota,
  formatTimestampToDate,
  formatUseTime,
} from '@/lib/format'

import type { KeyUsageLogEntry } from '../types'

/**
 * The relay endpoint each kind of request is served on, most specific first,
 * so `/v1/chat/completions` can never fall into `/v1/completions`. Every label
 * is an i18n key.
 */
const REQUEST_KINDS: ReadonlyArray<{ path: string; label: string }> = [
  { path: '/v1/chat/completions', label: 'Chat' },
  { path: '/v1/responses', label: 'Responses' },
  { path: '/v1/messages', label: 'Messages' },
  { path: '/v1/completions', label: 'Completions' },
  { path: '/v1/embeddings', label: 'Embeddings' },
  { path: '/v1/engines', label: 'Embeddings' },
  { path: '/v1/images', label: 'Image' },
  { path: '/v1/audio', label: 'Audio' },
  { path: '/v1/rerank', label: 'Rerank' },
  { path: '/v1/moderations', label: 'Moderation' },
  { path: '/v1/realtime', label: 'Realtime' },
  { path: '/v1/alpha/search', label: 'Search' },
  { path: '/pg', label: 'Playground' },
  { path: '/mj', label: 'Midjourney' },
  { path: '/v1beta', label: 'Gemini' },
]

/** The request kind of an entry, or `null` when its endpoint is unknown. */
export function getRequestKindLabel(requestPath: string): string | null {
  if (!requestPath) return null
  const kind =
    REQUEST_KINDS.find(
      (candidate) =>
        requestPath === candidate.path ||
        requestPath.startsWith(`${candidate.path}/`)
    ) ?? null
  return kind?.label ?? null
}

export interface UsageLogOutcome {
  label: string
  variant: StatusBadgeProps['variant']
}

/** A consumed request succeeded; an errored one failed; anything else — a
 * refund, an unknown type — carries no outcome at all, so the column stays
 * blank rather than claiming a result the entry does not describe. */
export function getUsageLogOutcome(type: number): UsageLogOutcome | null {
  if (type === LOG_TYPE_ENUM.CONSUME) {
    return { label: 'Success', variant: 'success' }
  }
  if (type === LOG_TYPE_ENUM.ERROR) {
    return { label: 'Failed', variant: 'danger' }
  }
  return null
}

/** The console's own cache convention: `↓` cache-read, `↑` cache-write. An
 * entry with neither shows a dash instead of two zeroes. */
export function getUsageLogCacheTokens(entry: KeyUsageLogEntry): string {
  const parts: string[] = []
  if (entry.cache_tokens > 0) {
    parts.push(`↓${entry.cache_tokens.toLocaleString()}`)
  }
  if (entry.cache_creation_tokens > 0) {
    parts.push(`↑${entry.cache_creation_tokens.toLocaleString()}`)
  }
  return parts.length > 0 ? parts.join(' ') : '-'
}

export interface UsageLogRow {
  /** Row identity for rendering; the public payload carries no row id. */
  id: string
  time: string
  modelName: string
  /** The request kind when the entry's endpoint is known, otherwise the
   * entry's log-type wording (Refund, Consume, …). An i18n key. */
  typeLabel: string
  outcome: UsageLogOutcome | null
  /** Grouped token counts; a count the entry did not record shows a dash. */
  inputTokens: string
  cacheTokens: string
  outputTokens: string
  cost: string
  duration: string
}

/** One table row per entry, in the order the entries arrived (newest first). */
export function getUsageLogRows(entries: KeyUsageLogEntry[]): UsageLogRow[] {
  return entries.map((entry) => {
    const logType =
      LOG_TYPES.find((candidate) => candidate.value === entry.type) ??
      LOG_TYPES[0]
    const inputTokens =
      entry.prompt_tokens > 0 ? entry.prompt_tokens.toLocaleString() : '-'
    const outputTokens =
      entry.completion_tokens > 0 ? entry.completion_tokens.toLocaleString() : '-'

    return {
      id: `${entry.created_at}-${entry.request_id}`,
      time: formatTimestampToDate(entry.created_at),
      modelName: entry.model_name || '-',
      typeLabel: getRequestKindLabel(entry.request_path) ?? logType.label,
      outcome: getUsageLogOutcome(entry.type),
      inputTokens,
      cacheTokens: getUsageLogCacheTokens(entry),
      outputTokens,
      cost: formatLogQuota(entry.quota),
      duration: entry.use_time > 0 ? formatUseTime(entry.use_time) : '-',
    }
  })
}

/** Total pages for a result set, never below one so a pager always has a
 * valid current page. */
export function getUsageLogPageCount(total: number, pageSize: number): number {
  if (pageSize <= 0) return 1
  return Math.max(1, Math.ceil(total / pageSize))
}
