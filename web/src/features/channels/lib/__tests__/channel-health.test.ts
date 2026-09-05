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

For commercial licensing, please contact support@quantumnous.com
*/
import { describe, expect, test } from 'vitest'

import type { ChannelHealthStat } from '../../types'
import {
  formatSuccessRate,
  getChannelHealthCellView,
  getSuccessRateVariant,
  type ChannelHealthLookup,
} from '../channel-utils'

function buildStat(overrides: Partial<ChannelHealthStat>): ChannelHealthStat {
  return {
    channel_id: 1,
    total_requests: 10,
    success_requests: 8,
    failed_requests: 2,
    success_rate: 0.8,
    avg_latency_ms: 1500,
    ...overrides,
  }
}

function buildLookup(stats: ChannelHealthStat[]): ChannelHealthLookup {
  const lookup: ChannelHealthLookup = new Map()
  for (const stat of stats) {
    lookup.set(stat.channel_id, stat)
  }
  return lookup
}

describe('getChannelHealthCellView', () => {
  test('channel present in the lookup renders its success rate and latency', () => {
    const lookup = buildLookup([
      buildStat({ channel_id: 42, success_rate: 0.8, avg_latency_ms: 2000 }),
    ])

    const view = getChannelHealthCellView(lookup, 42)

    expect(view.hasData).toBe(true)
    expect(view.successRateLabel).toBe('80%')
    expect(view.successRateVariant).toBe('warning')
    expect(view.avgLatencyMs).toBe(2000)
  })

  test('channel absent from the lookup renders the no-data state, not 0%', () => {
    const lookup = buildLookup([buildStat({ channel_id: 42 })])

    const view = getChannelHealthCellView(lookup, 999)

    expect(view.hasData).toBe(false)
    expect(view.successRateLabel).not.toBe('0%')
  })

  test('an empty lookup (query failed or unavailable) renders the no-data state for every channel', () => {
    const emptyLookup: ChannelHealthLookup = new Map()

    const view = getChannelHealthCellView(emptyLookup, 1)

    expect(view.hasData).toBe(false)
  })

  test('a channel present with a genuine zero success rate renders 0%, since it had real traffic', () => {
    const lookup = buildLookup([
      buildStat({
        channel_id: 7,
        total_requests: 5,
        success_requests: 0,
        failed_requests: 5,
        success_rate: 0,
        avg_latency_ms: 500,
      }),
    ])

    const view = getChannelHealthCellView(lookup, 7)

    expect(view.hasData).toBe(true)
    expect(view.successRateLabel).toBe('0%')
    expect(view.successRateVariant).toBe('danger')
  })
})

describe('formatSuccessRate', () => {
  test.each([
    [0, '0%'],
    [0.8, '80%'],
    [1, '100%'],
    [0.995, '100%'],
  ])('formats %f as %s', (rate, expected) => {
    expect(formatSuccessRate(rate)).toBe(expected)
  })
})

describe('getSuccessRateVariant', () => {
  test.each([
    [1, 'success'],
    [0.9, 'success'],
    [0.89, 'warning'],
    [0.7, 'warning'],
    [0.69, 'danger'],
    [0, 'danger'],
  ] as const)('rates %f as %s', (rate, expected) => {
    expect(getSuccessRateVariant(rate)).toBe(expected)
  })
})
