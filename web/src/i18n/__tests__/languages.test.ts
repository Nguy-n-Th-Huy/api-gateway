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

import {
  convertDetectedLanguage,
  INTERFACE_LANGUAGE_OPTIONS,
  normalizeInterfaceLanguage,
} from '../languages'

describe('INTERFACE_LANGUAGE_OPTIONS', () => {
  test('offers exactly English and Vietnamese', () => {
    expect(INTERFACE_LANGUAGE_OPTIONS.map((lang) => lang.code)).toEqual([
      'en',
      'vi',
    ])
  })
})

describe('normalizeInterfaceLanguage', () => {
  test('returns en for a removed language code', () => {
    expect(normalizeInterfaceLanguage('zhCN')).toBe('en')
  })

  test('returns en for a language never offered', () => {
    expect(normalizeInterfaceLanguage('fr')).toBe('en')
  })

  test('returns en for a missing preference', () => {
    expect(normalizeInterfaceLanguage(undefined)).toBe('en')
    expect(normalizeInterfaceLanguage(null)).toBe('en')
    expect(normalizeInterfaceLanguage('')).toBe('en')
  })

  test('resolves any Vietnamese regional variant to vi', () => {
    expect(normalizeInterfaceLanguage('vi')).toBe('vi')
    expect(normalizeInterfaceLanguage('vi-VN')).toBe('vi')
    expect(normalizeInterfaceLanguage('VI')).toBe('vi')
  })

  test('resolves en regional variants to en', () => {
    expect(normalizeInterfaceLanguage('en-US')).toBe('en')
  })
})

describe('convertDetectedLanguage', () => {
  test('resolves a Vietnamese browser tag to vi', () => {
    expect(convertDetectedLanguage('vi-VN')).toBe('vi')
    expect(convertDetectedLanguage('vi')).toBe('vi')
  })

  test('no longer maps a Chinese browser tag to a Chinese interface code', () => {
    expect(convertDetectedLanguage('zh-CN')).toBe('zh-CN')
    expect(convertDetectedLanguage('zh-TW')).toBe('zh-TW')
    expect(convertDetectedLanguage('zh')).toBe('zh')
  })

  test('passes through any other language unchanged for supportedLngs matching', () => {
    expect(convertDetectedLanguage('en-US')).toBe('en-US')
    expect(convertDetectedLanguage('fr')).toBe('fr')
  })
})
