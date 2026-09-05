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
import { AxiosError } from 'axios'
import { t } from 'i18next'

import { api } from '@/lib/api'

import { KEY_CHECK_ENDPOINT } from './constants'
import type { KeyCheckReport, KeyCheckResponse } from './types'

/**
 * Thrown when the key-check request completes but the backend rejects the
 * key (invalid, empty, or a server error) — `message` is already localized
 * by the backend, or falls back to a client-side localized message when the
 * backend gave none (e.g. a network failure).
 */
export class KeyCheckRequestError extends Error {}

/**
 * Calls `POST /api/token/check` with the key in the request body (never the
 * URL or query string). Resolves with the report on success; rejects with
 * `KeyCheckRequestError` otherwise, regardless of whether the backend
 * answered with an HTTP error status or a `200` carrying `success: false`.
 *
 * Uses `skipBusinessError`/`skipErrorHandler` because this feature shows the
 * server's message inline itself rather than relying solely on the global
 * toast interceptor.
 */
export async function checkToken(key: string): Promise<KeyCheckReport> {
  try {
    const res = await api.post<KeyCheckResponse>(
      KEY_CHECK_ENDPOINT,
      { key },
      { skipBusinessError: true, skipErrorHandler: true }
    )

    if (!res.data.success || !res.data.data) {
      throw new KeyCheckRequestError(
        res.data.message || t('Failed to check key')
      )
    }

    return res.data.data
  } catch (error) {
    if (error instanceof KeyCheckRequestError) throw error

    if (error instanceof AxiosError) {
      const message = error.response?.data?.message as string | undefined
      throw new KeyCheckRequestError(message || t('Failed to check key'))
    }

    throw new KeyCheckRequestError(t('Failed to check key'))
  }
}
