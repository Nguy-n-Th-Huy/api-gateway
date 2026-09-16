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
/**
 * Effective token status reported by `POST /api/token/check`.
 * Matches `common.TokenStatus*` on the backend and the frontend's
 * existing `API_KEY_STATUS` vocabulary (`@/features/keys/constants`).
 */
export type EffectiveKeyStatus = 1 | 2 | 3 | 4

/**
 * The report returned for an existing key. Field names and meanings mirror
 * `GetTokenUsage`'s response, plus `group` and `available_models` which are
 * unique to this endpoint. See `openspec/changes/add-public-key-check-page/
 * specs/public-key-check/spec.md`.
 */
export interface KeyCheckReport {
  name: string
  group: string
  status: EffectiveKeyStatus
  unlimited_quota: boolean
  total_granted: number
  total_used: number
  total_available: number
  /** Raw expiry timestamp (seconds). `-1` means the key never expires. */
  expires_at: number
  created_time: number
  accessed_time: number
  model_limits_enabled: boolean
  /** Map of allowed model name -> true. Only meaningful when
   * `model_limits_enabled` is true. Mirrors `Token.GetModelLimitsMap()`. */
  model_limits: Record<string, boolean>
  /** Models enabled for the key's effective group. Sole source for every
   * model choice offered on the page. */
  available_models: string[]
}

export interface KeyCheckResponse {
  success: boolean
  message?: string
  data?: KeyCheckReport
}

/**
 * One entry of the key's usage log, as returned by `POST /api/token/logs`.
 * Mirrors `service.PublicKeyLogEntry` — an explicit whitelist that carries no
 * account identity, client IP, or channel information. See
 * specs/public-key-check/spec.md — "Key usage log entries exclude account
 * identity and infrastructure fields".
 */
export interface KeyUsageLogEntry {
  /** Raw timestamp (seconds). */
  created_at: number
  /** `common.LogType*` on the backend, which decides the entry's outcome:
   * `consume` succeeded, `error` failed. */
  type: number
  model_name: string
  /** Quota units consumed by the request. */
  quota: number
  prompt_tokens: number
  completion_tokens: number
  /** Cache-read tokens, `0` when the entry recorded none. */
  cache_tokens: number
  /** Cache-write tokens, `0` when the entry recorded none. */
  cache_creation_tokens: number
  /** The relay endpoint the entry was recorded for, e.g. `/v1/chat/completions`.
   * Empty for an entry that has none (a refund, or metadata that could not be
   * read). */
  request_path: string
  /** Duration in seconds. */
  use_time: number
  is_stream: boolean
  group: string
  request_id: string
}

export interface KeyUsageLogsPage {
  page: number
  page_size: number
  total: number
  items: KeyUsageLogEntry[]
}

export interface KeyUsageLogsResponse {
  success: boolean
  message?: string
  data?: KeyUsageLogsPage
}
