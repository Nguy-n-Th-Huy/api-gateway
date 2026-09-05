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
  getExpiryDisplay,
  getModelRestrictionDisplay,
  getQuotaDisplay,
  getStatusEntry,
  isNeverExpires,
} from '../lib/report-display'
import type { KeyCheckReport } from '../types'

const baseReport: KeyCheckReport = {
  name: 'my-key',
  group: 'default',
  status: 1,
  unlimited_quota: false,
  total_granted: 2000000,
  total_used: 500000,
  total_available: 1500000,
  expires_at: -1,
  created_time: 1700000000,
  accessed_time: 1700000100,
  model_limits_enabled: false,
  model_limits: {},
  available_models: ['gpt-4o'],
}

describe('key-check report display — quota', () => {
  test('a limited-quota report yields three quota figures and a 25% used share', () => {
    const quota = getQuotaDisplay(baseReport)

    expect(quota.unlimited).toBe(false)
    expect(quota.usedSharePercent).toBe(25)
  })

  test('an unlimited-quota report has no usage share', () => {
    const quota = getQuotaDisplay({ ...baseReport, unlimited_quota: true })

    expect(quota.unlimited).toBe(true)
    expect(quota.usedSharePercent).toBeNull()
  })
})

describe('key-check report display — expiry', () => {
  test('expires_at === -1 means the key never expires', () => {
    expect(isNeverExpires(-1)).toBe(true)
    expect(getExpiryDisplay(-1)).toBeNull()
  })

  test('expires_at === 0 also means the key never expires', () => {
    expect(isNeverExpires(0)).toBe(true)
    expect(getExpiryDisplay(0)).toBeNull()
  })

  test('a positive expires_at renders a formatted date', () => {
    expect(isNeverExpires(1700000000)).toBe(false)
    expect(getExpiryDisplay(1700000000)).not.toBeNull()
  })
})

describe('key-check report display — model restriction', () => {
  test('model_limits_enabled false yields the all-models wording', () => {
    const restriction = getModelRestrictionDisplay({
      ...baseReport,
      model_limits_enabled: false,
    })

    expect(restriction.allAllowed).toBe(true)
    expect(restriction.models).toEqual([])
  })

  test('model_limits_enabled true lists the allowed models', () => {
    const restriction = getModelRestrictionDisplay({
      ...baseReport,
      model_limits_enabled: true,
      model_limits: { 'gpt-4o': true, 'gpt-4o-mini': true },
    })

    expect(restriction.allAllowed).toBe(false)
    expect(restriction.models).toEqual(['gpt-4o', 'gpt-4o-mini'])
  })
})

describe('key-check report display — status', () => {
  test.each([
    [1, 'Enabled', 'success'],
    [2, 'Disabled', 'neutral'],
    [3, 'Expired', 'warning'],
    [4, 'Exhausted', 'danger'],
  ] as const)('status %i maps to the %s status entry', (status, label, variant) => {
    const entry = getStatusEntry(status)

    expect(entry.label).toBe(label)
    expect(entry.variant).toBe(variant)
  })
})
