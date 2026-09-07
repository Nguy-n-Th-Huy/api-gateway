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
  isValidTelegramHandle,
  normalizeTelegramHandle,
} from '@/lib/telegram-handle'

describe('normalizeTelegramHandle', () => {
  test('trims surrounding whitespace, strips a leading @, and lower-cases', () => {
    expect(normalizeTelegramHandle('  @JohnDoe  ')).toBe('johndoe')
  })

  test('lower-cases mixed case without a leading @', () => {
    expect(normalizeTelegramHandle('JohnDoe99')).toBe('johndoe99')
  })

  test('an all-whitespace input normalizes to the empty string', () => {
    expect(normalizeTelegramHandle('   ')).toBe('')
  })
})

describe('isValidTelegramHandle', () => {
  test.each([
    ['john_doe99', true],
    ['abcd', false], // too short (< 5)
    ['a'.repeat(33), false], // too long (> 32)
    ['a'.repeat(32), true], // exactly the max length
    ['abcde', true], // exactly the min length
    ['john-doe', false], // illegal character: hyphen
    ['john.doe', false], // illegal character: dot
    ['john doe', false], // illegal character: space
    ['1johndoe', false], // starts with a digit
    ['_johndoe', false], // starts with an underscore
  ])('isValidTelegramHandle(%j) === %j', (input, expected) => {
    expect(isValidTelegramHandle(input)).toBe(expected)
  })
})
