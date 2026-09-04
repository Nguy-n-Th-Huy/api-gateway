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
import { existsSync } from 'node:fs'
import path from 'node:path'

import { afterEach, describe, expect, test } from 'vitest'

import { getCookie } from '@/lib/cookies'
import { clearLegacyAppearanceCookies } from '@/lib/legacy-appearance-cleanup'

const LEGACY_APPEARANCE_COOKIES = [
  'theme_preset',
  'theme_font',
  'theme_radius',
  'theme_scale',
  'theme_content_layout',
]

function writeCookie(name: string, value: string): void {
  document.cookie = `${name}=${value}; path=/; max-age=3600`
}

afterEach(() => {
  document.body.removeAttribute('data-theme-preset')
  document.body.removeAttribute('data-theme-font')
  document.body.removeAttribute('data-theme-radius')
  document.body.removeAttribute('data-theme-scale')
  document.body.removeAttribute('data-theme-content-layout')
})

describe('single application theme (web-theming)', () => {
  test('boot cleanup expires every legacy appearance cookie', () => {
    for (const name of LEGACY_APPEARANCE_COOKIES) {
      writeCookie(name, 'anthropic')
    }
    clearLegacyAppearanceCookies()
    for (const name of LEGACY_APPEARANCE_COOKIES) {
      expect(getCookie(name)).toBeUndefined()
    }
  })

  test('stale appearance cookies leave no data-theme attributes on the body', () => {
    for (const name of LEGACY_APPEARANCE_COOKIES) {
      writeCookie(name, 'rose-garden')
    }
    clearLegacyAppearanceCookies()

    expect(document.body.hasAttribute('data-theme-preset')).toBe(false)
    expect(document.body.hasAttribute('data-theme-font')).toBe(false)
    expect(document.body.hasAttribute('data-theme-radius')).toBe(false)
    expect(document.body.hasAttribute('data-theme-scale')).toBe(false)
    expect(document.body.hasAttribute('data-theme-content-layout')).toBe(false)
  })

  test('the theme-customization module no longer exists on disk', () => {
    const root = path.resolve(__dirname, '..', '..')
    expect(existsSync(path.join(root, 'lib', 'theme-customization.ts'))).toBe(
      false
    )
    expect(
      existsSync(path.join(root, 'context', 'theme-customization-provider.tsx'))
    ).toBe(false)
  })

  test('the appearance config drawer and quick switcher no longer exist on disk', () => {
    const root = path.resolve(__dirname, '..', '..')
    expect(existsSync(path.join(root, 'components', 'config-drawer.tsx'))).toBe(
      false
    )
    expect(
      existsSync(path.join(root, 'components', 'theme-quick-switcher.tsx'))
    ).toBe(false)
  })
})
