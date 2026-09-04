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

import { buildGoogleOAuthUrl } from '../oauth'

describe('buildGoogleOAuthUrl', () => {
  test('targets the Google authorization endpoint with the minimum required scopes', () => {
    const url = new URL(buildGoogleOAuthUrl('client-123', 'state-abc'))

    expect(url.origin + url.pathname).toBe(
      'https://accounts.google.com/o/oauth2/v2/auth'
    )
    expect(url.searchParams.get('client_id')).toBe('client-123')
    expect(url.searchParams.get('state')).toBe('state-abc')
    expect(url.searchParams.get('response_type')).toBe('code')
    expect(url.searchParams.get('scope')).toBe('openid email profile')
  })

  test('points redirect_uri at the current origin oauth/google callback', () => {
    const url = new URL(buildGoogleOAuthUrl('client-123', 'state-abc'))

    expect(url.searchParams.get('redirect_uri')).toBe(
      `${window.location.origin}/oauth/google`
    )
  })

  test('does not request offline access or a refresh token', () => {
    const url = new URL(buildGoogleOAuthUrl('client-123', 'state-abc'))

    expect(url.searchParams.get('access_type')).toBeNull()
    expect(url.searchParams.get('prompt')).toBeNull()
  })
})
