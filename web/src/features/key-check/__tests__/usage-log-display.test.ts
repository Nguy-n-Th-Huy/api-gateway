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
  getRequestKindLabel,
  getUsageLogCacheTokens,
  getUsageLogOutcome,
  getUsageLogPageCount,
  getUsageLogRows,
} from '../lib/log-display'
import type { KeyUsageLogEntry } from '../types'

const baseEntry: KeyUsageLogEntry = {
  created_at: 1700000000,
  type: 2,
  model_name: 'gpt-4o',
  quota: 500000,
  prompt_tokens: 1500,
  completion_tokens: 20,
  cache_tokens: 0,
  cache_creation_tokens: 0,
  request_path: '/v1/chat/completions',
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

describe('key-check usage log display — request kind', () => {
  test('a chat completion is labelled as chat', () => {
    expect(getRequestKindLabel('/v1/chat/completions')).toBe('Chat')
  })

  test('the legacy completions endpoint does not fall into the chat kind', () => {
    expect(getRequestKindLabel('/v1/completions')).toBe('Completions')
  })

  test('a nested path keeps its parent kind', () => {
    expect(getRequestKindLabel('/v1/images/generations')).toBe('Image')
    expect(getRequestKindLabel('/v1/responses/compact')).toBe('Responses')
  })

  test('an unknown endpoint has no kind', () => {
    expect(getRequestKindLabel('/v1/unknown/thing')).toBeNull()
  })

  test('an entry with no endpoint has no kind', () => {
    expect(getRequestKindLabel('')).toBeNull()
  })
})

describe('key-check usage log display — outcome', () => {
  test('a consumed request is successful', () => {
    expect(getUsageLogOutcome(2)).toEqual({
      label: 'Success',
      variant: 'success',
    })
  })

  test('an errored request has failed', () => {
    expect(getUsageLogOutcome(5)).toEqual({
      label: 'Failed',
      variant: 'danger',
    })
  })

  test('a refund carries no outcome rather than claiming one', () => {
    expect(getUsageLogOutcome(6)).toBeNull()
  })
})

describe('key-check usage log display — cache tokens', () => {
  test('an entry with no cache usage shows a dash', () => {
    expect(getUsageLogCacheTokens(baseEntry)).toBe('-')
  })

  test('a cache read is shown with the downward marker', () => {
    const cell = getUsageLogCacheTokens({ ...baseEntry, cache_tokens: 86272 })

    expect(cell.startsWith('↓')).toBe(true)
    expect(cell.replaceAll(/\D/g, '')).toBe('86272')
  })

  test('a cache write is shown with the upward marker', () => {
    const cell = getUsageLogCacheTokens({
      ...baseEntry,
      cache_creation_tokens: 1024,
    })

    expect(cell.startsWith('↑')).toBe(true)
    expect(cell.replaceAll(/\D/g, '')).toBe('1024')
  })

  test('a read and a write are shown side by side', () => {
    const cell = getUsageLogCacheTokens({
      ...baseEntry,
      cache_tokens: 86272,
      cache_creation_tokens: 1024,
    })

    expect(cell.replaceAll(/[^\d↑↓]+/g, '')).toBe('↓86272↑1024')
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

  test('input and output tokens are separate grouped figures', () => {
    const [row] = getUsageLogRows([baseEntry])

    // The grouping separator follows the runtime locale, so the assertions
    // check the value and the marker, not the separator character.
    expect(row.inputTokens.replaceAll(/\D/g, '')).toBe('1500')
    expect(row.outputTokens).toBe('20')
  })

  test('a request that recorded no tokens shows a dash per cell', () => {
    const [row] = getUsageLogRows([
      { ...baseEntry, prompt_tokens: 0, completion_tokens: 0 },
    ])

    expect(row.inputTokens).toBe('-')
    expect(row.outputTokens).toBe('-')
  })

  test('the kind and the outcome come from the entry', () => {
    const [okRow] = getUsageLogRows([baseEntry])
    const [failedRow] = getUsageLogRows([{ ...baseEntry, type: 5 }])

    expect(okRow.typeLabel).toBe('Chat')
    expect(okRow.outcome?.label).toBe('Success')
    expect(failedRow.outcome?.label).toBe('Failed')
  })

  test('a refund falls back to the log-type wording and carries no outcome', () => {
    const [row] = getUsageLogRows([
      { ...baseEntry, type: 6, request_path: '' },
    ])

    expect(row.typeLabel).toBe('Refund')
    expect(row.outcome).toBeNull()
  })

  test('an unrecognized log type falls back to the vocabulary default', () => {
    const [row] = getUsageLogRows([
      { ...baseEntry, type: 4242, request_path: '' },
    ])

    expect(row.typeLabel).toBe('Unknown')
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
    expect(
      getUsageLogRows([{ ...baseEntry, model_name: '' }])[0].modelName
    ).toBe('-')
  })
})
