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

// Mirrors model.NormalizeTelegramHandle / model.ValidateTelegramHandle
// (model/telegram_handle.go) so the sign-up form and profile settings can
// reject an invalid handle before it ever reaches the server. The server
// remains the source of truth; this is client-side convenience only.

export const TELEGRAM_HANDLE_MIN_LENGTH = 5
export const TELEGRAM_HANDLE_MAX_LENGTH = 32

const TELEGRAM_HANDLE_PATTERN = /^[a-z][a-z0-9_]*$/

/**
 * Trims surrounding whitespace, removes a leading "@", and lower-cases the
 * result. Never throws: an input that normalizes to the empty string means
 * no handle was declared.
 */
export function normalizeTelegramHandle(raw: string): string {
  let value = raw.trim()
  if (value.startsWith('@')) {
    value = value.slice(1)
  }
  return value.toLowerCase()
}

/**
 * Accepts an already-normalized handle only when it is 5 to 32 characters
 * long, contains only letters, digits, and underscores, and begins with a
 * letter.
 */
export function isValidTelegramHandle(normalized: string): boolean {
  return (
    normalized.length >= TELEGRAM_HANDLE_MIN_LENGTH &&
    normalized.length <= TELEGRAM_HANDLE_MAX_LENGTH &&
    TELEGRAM_HANDLE_PATTERN.test(normalized)
  )
}
