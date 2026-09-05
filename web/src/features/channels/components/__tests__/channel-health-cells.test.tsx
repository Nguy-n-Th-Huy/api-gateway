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
import { render } from '@testing-library/react'
import { describe, expect, test } from 'vitest'

import type { ChannelHealthStat } from '../../types'

const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { getChannelHealthCellView } = await import('../../lib/channel-utils')
const { AvgLatencyCell, SuccessRateCell } = await import(
  '../channel-health-cells'
)

type ChannelHealthLookup = Map<number, ChannelHealthStat>

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Not tested': 'Not tested',
        '< 1s': '< 1s',
        '{{value}}ms': '{{value}}ms',
        '{{value}}s': '{{value}}s',
      },
    },
  },
})

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

/**
 * Renders the actual `SuccessRateCell` / `AvgLatencyCell` components used by
 * the real "success_rate" and "avg_latency_ms" table columns in
 * `channels-columns.tsx`, from the same `ChannelHealthCellView` those
 * columns compute via `getChannelHealthCellView`. This exercises the real
 * `!hasData` branch inside the rendered cells, not just the view-model
 * helper: if that branch were removed, these assertions would fail.
 */
function renderHealthCells(lookup: ChannelHealthLookup, channelId: number) {
  const view = getChannelHealthCellView(lookup, channelId)
  return render(
    <I18nextProvider i18n={i18n}>
      <span data-testid='success-rate-cell'>
        <SuccessRateCell view={view} />
      </span>
      <span data-testid='avg-latency-cell'>
        <AvgLatencyCell view={view} />
      </span>
    </I18nextProvider>
  )
}

describe('channel health table cells (rendered)', () => {
  test('channel present with traffic renders its actual success rate and latency', () => {
    const lookup = buildLookup([
      buildStat({ channel_id: 1, success_rate: 0.8, avg_latency_ms: 2000 }),
    ])

    const { getByTestId } = renderHealthCells(lookup, 1)

    expect(getByTestId('success-rate-cell')).toHaveTextContent('80%')
    expect(getByTestId('avg-latency-cell')).toHaveTextContent('2.00s')
  })

  test('channel absent from the lookup renders "-" in both columns, never 0%', () => {
    const lookup = buildLookup([buildStat({ channel_id: 1 })])

    const { getByTestId } = renderHealthCells(lookup, 999)

    expect(getByTestId('success-rate-cell')).toHaveTextContent('-')
    expect(getByTestId('success-rate-cell')).not.toHaveTextContent('0%')
    expect(getByTestId('avg-latency-cell')).toHaveTextContent('-')
  })

  test('an empty lookup (health query failed or unavailable) renders "-" in both columns', () => {
    const emptyLookup: ChannelHealthLookup = new Map()

    const { getByTestId } = renderHealthCells(emptyLookup, 1)

    expect(getByTestId('success-rate-cell')).toHaveTextContent('-')
    expect(getByTestId('avg-latency-cell')).toHaveTextContent('-')
  })

  test('channel present with avg_latency_ms 0 renders "< 1s", not the "Not tested" label', () => {
    const lookup = buildLookup([
      buildStat({ channel_id: 1, avg_latency_ms: 0 }),
    ])

    const { getByTestId } = renderHealthCells(lookup, 1)

    expect(getByTestId('avg-latency-cell')).toHaveTextContent('< 1s')
    expect(getByTestId('avg-latency-cell')).not.toHaveTextContent('Not tested')
  })
})
