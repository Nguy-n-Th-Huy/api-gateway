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
import { Landmark } from 'lucide-react'
import { type ReactNode } from 'react'

import { ReactIconByName } from '@/components/react-icon-by-name'

// ============================================================================
// UI Helper Functions (SePay — the single provider)
// ============================================================================

/**
 * Resolves a backend-provided image URL to https only. Rejects http:,
 * data:, blob:, file:, relative paths, and URLs with userinfo, which are unsafe
 * or ambiguous in <img src/>.
 */
function normalizeHttpIconUrl(raw: string | undefined | null): string | null {
  if (!raw) return null
  const s = raw.trim()
  if (!s) return null
  if (!/^https:\/\//i.test(s)) return null

  let url: URL
  try {
    url = new URL(s)
  } catch {
    return null
  }
  if (url.protocol !== 'https:') {
    return null
  }
  if (url.username || url.password) {
    return null
  }
  return url.toString()
}

/**
 * Get the payment icon component for the single provider (SePay, a domestic
 * bank transfer).
 *
 * When an icon is explicitly configured, render a safe http(s) image URL or
 * resolve it as a react-icons component name. Invalid configured icons
 * intentionally render nothing instead of falling back. Without a configured
 * icon, render the default bank/landmark icon.
 */
export function getPaymentIcon(
  className: string = 'h-4 w-4',
  icon?: string,
  altName?: string
): ReactNode {
  const iconValue = icon?.trim()
  const safeIconUrl = normalizeHttpIconUrl(iconValue)
  if (safeIconUrl) {
    return (
      <img
        src={safeIconUrl}
        alt={altName || 'SePay'}
        className={className}
        style={{ objectFit: 'contain' }}
        loading='lazy'
        decoding='async'
        referrerPolicy='no-referrer'
      />
    )
  }
  if (iconValue) {
    return (
      <ReactIconByName
        name={iconValue}
        className={className}
        title={altName || 'SePay'}
      />
    )
  }

  return <Landmark className={className} />
}
