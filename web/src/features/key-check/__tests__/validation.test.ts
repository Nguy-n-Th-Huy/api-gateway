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
import type { TFunction } from 'i18next'
import { describe, expect, test } from 'vitest'

import { getKeyCheckFormSchema } from '../lib/validation'

const t = ((key: string, options?: Record<string, unknown>) => {
  if (options?.min !== undefined) {
    return key.replace('{{min}}', String(options.min))
  }
  return key
}) as TFunction

describe('key-check form schema', () => {
  test('rejects an empty key with the required-field message', () => {
    const result = getKeyCheckFormSchema(t).safeParse({ key: '' })

    expect(result.success).toBe(false)
    expect(result.error?.issues[0]?.message).toBe('Please enter an API key')
  })

  test('rejects a whitespace-only key with the required-field message', () => {
    const result = getKeyCheckFormSchema(t).safeParse({ key: '   ' })

    expect(result.success).toBe(false)
    expect(result.error?.issues[0]?.message).toBe('Please enter an API key')
  })

  test('rejects a key shorter than 8 characters after trimming with the minimum-length message', () => {
    const result = getKeyCheckFormSchema(t).safeParse({ key: '  abc12  ' })

    expect(result.success).toBe(false)
    expect(result.error?.issues[0]?.message).toBe(
      'The key must be at least 8 characters'
    )
  })

  test('accepts a valid key and trims it', () => {
    const result = getKeyCheckFormSchema(t).safeParse({
      key: '  sk-abc123def456  ',
    })

    expect(result.success).toBe(true)
    expect(result.data?.key).toBe('sk-abc123def456')
  })

  test('accepts a key exactly at the minimum length', () => {
    const result = getKeyCheckFormSchema(t).safeParse({ key: '12345678' })

    expect(result.success).toBe(true)
  })
})
