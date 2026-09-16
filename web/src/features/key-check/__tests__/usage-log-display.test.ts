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
  getLogTypeLabel,
  getUsageLogPageCount,
  getUsageLogRows,
  getUsageLogTokens,
} from '../lib/log-display'
import type { KeyUsageLogEntry } from '../types'

const baseEntry: KeyUsageLogEntry = {
  created_at: 1700000000,
  type: 2,
  model_name: 'gpt-4o',
  quota: 500000,
  prompt_tokens: 1500,
  completion_tokens: 20,
  use_time: 1.5,
  is_stream: true,
  group: 'default',
  request_id: 'req-1',
}

describe('key-check usage log display — paging', () => {
  test('no entries still yields one page', () => {
    expect(getUsageLogPageCount(0, 10)).toBe(1)
  })

  test('an exact multiple yields one page per slice', () => {
    expect(getUsageLogPageCount(20, 10)).toBe(2)
  })

  test('a remainder yields one more page', () => {
    expect(getUsageLogPageCount(21, 10)).toBe(3)
  })
})

describe('key-check usage log display — tokens', () => {
  test('a request that recorded no tokens shows a single dash', () => {
    expect(
      getUsageLogTokens({ ...baseEntry, prompt_tokens: 0, completion_tokens: 0 })
    ).toBe('-')
  })

  test('prompt and completion tokens are shown side by side', () => {
    expect(getUsageLogTokens(baseEntry)).toBe('1.5K / 20')
  })

  test('a missing completion count is still shown as a dash', () => {
    expect(getUsageLogTokens({ ...baseEntry, completion_tokens: 0 })).toBe(
      '1.5K / -'
    )
  })
})

describe('key-check usage log display — rows', () => {
  test('one row is produced per entry, in payload order', () => {
    const rows = getUsageLogRows([
      baseEntry,
      { ...baseEntry, created_at: 1700000100, request_id: 'req-2' },
    ])

    expect(rows).toHaveLength(2)
    expect(rows[0].id).not.toBe(rows[1].id)
  })

  test('the timestamp is rendered from the log seconds', () => {
    const [row] = getUsageLogRows([baseEntry])

    expect(row.time).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/)
  })

  test('the type is labelled with the console log-type vocabulary', () => {
    expect(getLogTypeLabel(5)).toBe('Error')
    expect(getUsageLogRows([{ ...baseEntry, type: 6 }])[0].typeLabel).toBe(
      'Refund'
    )
  })

  test('an unrecognized type falls back to the vocabulary default', () => {
    expect(getLogTypeLabel(4242)).toBe('Unknown')
  })

  test('a request with no recorded duration shows a dash', () => {
    expect(getUsageLogRows([{ ...baseEntry, use_time: 0 }])[0].duration).toBe(
      '-'
    )
  })

  test('a request with a recorded duration shows it in seconds', () => {
    expect(getUsageLogRows([{ ...baseEntry, use_time: 1.5 }])[0].duration).toBe(
      '1.5s'
    )
  })

  test('an entry with no model name shows a dash instead of an empty cell', () => {
    expect(getUsageLogRows([{ ...baseEntry, model_name: '' }])[0].modelName).toBe(
      '-'
    )
  })
})