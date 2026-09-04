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
import { describe, expect, test } from 'vitest'

import { getAvailableOAuthProviders, hasOAuthProviders } from '../oauth'
import type { SystemStatus } from '../../types'

describe('getAvailableOAuthProviders', () => {
  test('returns an empty list when status is null', () => {
    expect(getAvailableOAuthProviders(null)).toEqual([])
  })

  test('includes Google with its client id when google_oauth is enabled', () => {
    const status: SystemStatus = {
      google_oauth: true,
      google_client_id: 'google-client-id',
    }

    const providers = getAvailableOAuthProviders(status)

    expect(providers).toContainEqual({
      name: 'Google',
      type: 'google',
      enabled: true,
      clientId: 'google-client-id',
    })
  })

  test('omits Google when google_oauth is disabled', () => {
    const status: SystemStatus = {
      google_oauth: false,
      github_oauth: true,
      github_client_id: 'github-client-id',
    }

    const providers = getAvailableOAuthProviders(status)

    expect(providers.some((provider) => provider.type === 'google')).toBe(
      false
    )
  })
})

describe('hasOAuthProviders', () => {
  test('reports true when only Google is enabled', () => {
    const status: SystemStatus = { google_oauth: true }
    expect(hasOAuthProviders(status)).toBe(true)
  })

  test('reports false when no provider is enabled', () => {
    const status: SystemStatus = {}
    expect(hasOAuthProviders(status)).toBe(false)
  })
})
