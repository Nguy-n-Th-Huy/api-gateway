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
  getParameterControlValueText,
  normalizeParameterNumberValue,
  PLAYGROUND_PARAMETER_CONTROLS,
} from '../lib/parameters/playground-parameters'

// ---------------------------------------------------------------------------
// Parameter control bounds / state normalization
// The parameter rail clamps and normalizes every user entry through
// normalizeParameterNumberValue; these tests lock in that contract so a
// bounds change cannot silently let out-of-range values reach the request.
// They also exercise the display helpers callers show in the badge.
// ---------------------------------------------------------------------------

describe('playground parameter control metadata', () => {
  test('every control declares a complete, finite min/max/step contract', () => {
    for (const control of PLAYGROUND_PARAMETER_CONTROLS) {
      const c = control as { min: number; max: number; step: number }
      expect(Number.isFinite(c.min)).toBe(true)
      expect(Number.isFinite(c.max)).toBe(true)
      expect(Number.isFinite(c.step)).toBe(true)
      expect(c.max).toBeGreaterThanOrEqual(c.min)
      expect(c.step).toBeGreaterThan(0)
    }
  })

  test('controls cover the six exposed sampling parameters exactly once', () => {
    const keys = PLAYGROUND_PARAMETER_CONTROLS.map((control) => control.key)
    expect([...keys].sort()).toEqual(
      [
        'frequency_penalty',
        'max_tokens',
        'presence_penalty',
        'seed',
        'temperature',
        'top_p',
      ].sort()
    )
  })
})

describe('playground parameter value normalization', () => {
  test('values inside the range are preserved', () => {
    expect(normalizeParameterNumberValue('temperature', 0.5)).toBe(0.5)
    expect(normalizeParameterNumberValue('max_tokens', 1000)).toBe(1000)
  })

  test('values above max are clamped down to max', () => {
    expect(normalizeParameterNumberValue('temperature', 5)).toBe(1)
    expect(normalizeParameterNumberValue('top_p', 2)).toBe(1)
    expect(normalizeParameterNumberValue('frequency_penalty', 10)).toBe(2)
  })

  test('values below min are clamped up to min', () => {
    expect(normalizeParameterNumberValue('temperature', -1)).toBe(0.1)
    expect(normalizeParameterNumberValue('presence_penalty', -99)).toBe(-2)
  })

  test('decimal controls keep the step precision and drop float noise', () => {
    expect(normalizeParameterNumberValue('temperature', 0.30000000004)).toBe(0.3)
    expect(normalizeParameterNumberValue('top_p', '0.7')).toBe(0.7)
  })

  test('integer controls truncate fractional input', () => {
    expect(normalizeParameterNumberValue('max_tokens', 99.9)).toBe(99)
    expect(normalizeParameterNumberValue('seed', 12345.67)).toBe(12345)
  })

  test('empty input resets to zero for regular controls but to null for seed', () => {
    expect(normalizeParameterNumberValue('max_tokens', '')).toBe(0)
    expect(normalizeParameterNumberValue('temperature', '')).toBe(0)
    expect(normalizeParameterNumberValue('seed', '')).toBeNull()
  })

  test('non-numeric input falls back to the control default without throwing', () => {
    expect(normalizeParameterNumberValue('temperature', 'abc')).toBe(0)
    expect(normalizeParameterNumberValue('seed', 'not-a-number')).toBeNull()
  })

  test('string numerics are parsed and clamped the same as numbers', () => {
    expect(normalizeParameterNumberValue('temperature', '0.9')).toBe(0.9)
    expect(normalizeParameterNumberValue('max_tokens', '999999999')).toBe(200000)
  })
})

describe('playground parameter display text', () => {
  test('unset seed renders as the Not set placeholder', () => {
    expect(getParameterControlValueText('seed', null)).toBe('Not set')
  })

  test('concrete values render as their string form', () => {
    expect(getParameterControlValueText('temperature', 0.7)).toBe('0.7')
    expect(getParameterControlValueText('max_tokens', 4096)).toBe('4096')
    expect(getParameterControlValueText('seed', 42)).toBe('42')
  })
})
