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
 * One-shot cleanup for appearance cookies written by the removed theme
 * customization layer (preset / font / radius / scale / content-layout).
 *
 * The single canonical theme no longer reads these values, but stale cookies
 * left in a viewer's browser would otherwise linger unread and can confuse
 * debugging. Expiring them once on boot keeps the stored state clean without
 * affecting the light/dark preference, which lives under a separate key.
 */
import { removeCookie } from '@/lib/cookies'

const LEGACY_APPEARANCE_COOKIE_KEYS = [
  'theme_preset',
  'theme_font',
  'theme_radius',
  'theme_scale',
  'theme_content_layout',
] as const

export function clearLegacyAppearanceCookies(): void {
  if (typeof document === 'undefined') return
  for (const key of LEGACY_APPEARANCE_COOKIE_KEYS) {
    removeCookie(key)
  }
}
