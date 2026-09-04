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
export const INTERFACE_LANGUAGE_OPTIONS = [
  { code: 'en', label: 'English' },
  { code: 'vi', label: 'Tiếng Việt' },
] as const

export type InterfaceLanguageCode =
  (typeof INTERFACE_LANGUAGE_OPTIONS)[number]['code']

export function normalizeInterfaceLanguage(value?: string | null): string {
  if (!value) return 'en'

  const normalized = value.trim().replaceAll('_', '-').toLowerCase()
  const vietnamese = normalized === 'vi' || normalized.startsWith('vi-')

  return vietnamese ? 'vi' : 'en'
}

/**
 * Map a browser-detected locale onto the interface language codes this
 * project uses with i18next (`en` / `vi`).
 *
 * Browsers report standard BCP-47 tags (`vi-VN`, `en-US`, ...); any Vietnamese
 * variant resolves to `vi`. Every other value is returned unchanged so
 * i18next's own `supportedLngs` matching applies and — since only `en`/`vi`
 * are supported — falls to `fallbackLng` (`en`).
 */
export function convertDetectedLanguage(value: string): string {
  const lower = value.trim().replaceAll('_', '-').toLowerCase()
  if (lower === 'vi' || lower.startsWith('vi-')) return 'vi'
  return value
}

/**
 * Convert an interface language code (`en` / `vi`) into a valid BCP-47 locale
 * tag that the `Intl.*` APIs accept.
 *
 * Any locale derived from `i18n.language` / `i18n.resolvedLanguage` MUST be
 * run through this before it reaches an `Intl` constructor, which throws on
 * an unrecognized tag. Unknown values fall back to `undefined`, which makes
 * `Intl` use the runtime default locale.
 */
export function toIntlLocale(value?: string | null): string | undefined {
  if (!value) return undefined
  try {
    return Intl.getCanonicalLocales(value)[0]
  } catch {
    return undefined
  }
}
