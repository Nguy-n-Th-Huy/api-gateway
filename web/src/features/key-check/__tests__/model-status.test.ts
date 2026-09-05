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

import type { PerfModelSummary } from '@/features/performance-metrics/types'

import {
  buildBarStrip,
  buildModelStatusList,
  classifyBarColor,
  classifyModelState,
} from '../lib/model-status'

function summary(overrides: Partial<PerfModelSummary>): PerfModelSummary {
  return {
    model_name: 'gpt-4o',
    avg_latency_ms: 100,
    success_rate: 100,
    avg_tps: 10,
    ...overrides,
  }
}

describe('model status — health classification', () => {
  test('a 98% success rate is Operational', () => {
    expect(classifyModelState(summary({ success_rate: 98 }))).toBe(
      'operational'
    )
  })

  test('exactly 90% success rate is Operational (boundary)', () => {
    expect(classifyModelState(summary({ success_rate: 90 }))).toBe(
      'operational'
    )
  })

  test('a 20% success rate with recorded traffic is Down', () => {
    expect(classifyModelState(summary({ success_rate: 20 }))).toBe('down')
  })

  test('no metrics entry at all is No Data', () => {
    expect(classifyModelState(undefined)).toBe('no-data')
  })
})

describe('model status — bar colour bands', () => {
  test('healthy at or above 90%', () => {
    expect(classifyBarColor(90)).toBe('healthy')
    expect(classifyBarColor(100)).toBe('healthy')
  })

  test('degraded from 50 up to (not including) 90%', () => {
    expect(classifyBarColor(50)).toBe('degraded')
    expect(classifyBarColor(89.9)).toBe('degraded')
  })

  test('failing below 50%', () => {
    expect(classifyBarColor(49.9)).toBe('failing')
    expect(classifyBarColor(0)).toBe('failing')
  })

  test('neutral for an interval with no recorded traffic', () => {
    expect(classifyBarColor(null)).toBe('neutral')
    expect(classifyBarColor(undefined)).toBe('neutral')
  })
})

describe('model status — bar strip', () => {
  test('a full window renders one bar of each colour band, oldest to newest', () => {
    const bars = buildBarStrip([100, 70, 20]).map((rate) =>
      classifyBarColor(rate)
    )

    expect(bars).toEqual(['healthy', 'degraded', 'failing'])
  })

  test('a model with no recorded intervals renders a full row of neutral bars', () => {
    const bars = buildBarStrip(undefined, 3)

    expect(bars).toEqual([null, null, null])
    expect(bars.map((rate) => classifyBarColor(rate))).toEqual([
      'neutral',
      'neutral',
      'neutral',
    ])
  })

  test('fewer recorded intervals than the window pads the oldest side with neutral', () => {
    const bars = buildBarStrip([80], 3)

    expect(bars).toEqual([null, null, 80])
  })
})

describe('model status — list narrowing', () => {
  test('a group model with no metrics entry is still listed, as No Data', () => {
    const list = buildModelStatusList(
      ['gpt-4o', 'no-metrics-model'],
      [summary({ model_name: 'gpt-4o', success_rate: 95 })]
    )

    expect(list).toHaveLength(2)
    expect(list[0].state).toBe('operational')
    expect(list[1].modelName).toBe('no-metrics-model')
    expect(list[1].state).toBe('no-data')
  })

  test('switching keys leaves no model from the previous key', () => {
    const summaries = [
      summary({ model_name: 'gpt-4o', success_rate: 95 }),
      summary({ model_name: 'claude-3', success_rate: 95 }),
    ]

    const firstKeyList = buildModelStatusList(['gpt-4o'], summaries)
    const secondKeyList = buildModelStatusList(['claude-3'], summaries)

    expect(firstKeyList.map((entry) => entry.modelName)).toEqual(['gpt-4o'])
    expect(secondKeyList.map((entry) => entry.modelName)).toEqual([
      'claude-3',
    ])
  })
})
