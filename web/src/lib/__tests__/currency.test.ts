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
import i18next from 'i18next'
import { afterEach, describe, expect, test } from 'vitest'

import {
  formatCurrencyFromUSD,
  getCurrencyDisplay,
  getCurrencyLabel,
} from '@/lib/currency'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
  type CurrencyConfig,
  type CurrencyDisplayType,
} from '@/stores/system-config-store'

function setCurrency(overrides: Partial<CurrencyConfig>): void {
  useSystemConfigStore.setState((state) => ({
    config: {
      ...state.config,
      currency: {
        ...DEFAULT_CURRENCY_CONFIG,
        ...overrides,
      },
    },
  }))
}

afterEach(async () => {
  await i18next.changeLanguage('en')
  setCurrency({})
})

describe('getDisplayMeta (via getCurrencyDisplay/formatCurrencyFromUSD)', () => {
  test('Vietnamese interface renders Dong at the status.price rate', async () => {
    await i18next.changeLanguage('vi')
    setCurrency({ quotaDisplayType: 'USD', dongPerUsd: 25000 })

    const { meta } = getCurrencyDisplay()
    expect(meta).toEqual({
      kind: 'currency',
      symbol: '₫',
      currencyCode: 'VND',
      exchangeRate: 25000,
    })
    expect(formatCurrencyFromUSD(1, { locale: 'en' })).toBe('₫25,000')
  })

  test.each([
    ['not yet loaded', 0],
    ['negative', -5],
  ])(
    'Vietnamese interface falls back to dollars when the rate is %s',
    async (_label, dongPerUsd) => {
      await i18next.changeLanguage('vi')
      setCurrency({ quotaDisplayType: 'USD', dongPerUsd })

      // Showing Dong at a stand-in rate would understate every price by three
      // orders of magnitude, so an unknown rate must not produce a Dong amount.
      const { meta } = getCurrencyDisplay()
      expect(meta).toEqual({
        kind: 'currency',
        symbol: '$',
        currencyCode: 'USD',
        exchangeRate: 1,
      })
      expect(formatCurrencyFromUSD(1, { locale: 'en' })).not.toContain('₫')
    }
  )

  test('English interface renders US dollars at rate 1 regardless of dongPerUsd', async () => {
    await i18next.changeLanguage('en')
    setCurrency({ quotaDisplayType: 'USD', dongPerUsd: 25000 })

    const { meta } = getCurrencyDisplay()
    expect(meta).toEqual({
      kind: 'currency',
      symbol: '$',
      currencyCode: 'USD',
      exchangeRate: 1,
    })
    expect(formatCurrencyFromUSD(10, { locale: 'en' })).toBe('$10')
  })

  test('TOKENS mode renders tokens regardless of the interface language', async () => {
    setCurrency({ quotaDisplayType: 'TOKENS', quotaPerUnit: 500000 })

    await i18next.changeLanguage('en')
    expect(getCurrencyDisplay().meta).toEqual({
      kind: 'tokens',
      quotaPerUnit: 500000,
    })

    await i18next.changeLanguage('vi')
    expect(getCurrencyDisplay().meta).toEqual({
      kind: 'tokens',
      quotaPerUnit: 500000,
    })
  })

  test.each<CurrencyDisplayType>(['CNY', 'CUSTOM'])(
    'legacy stored %s value resolves to currency mode following the interface language',
    async (legacyType) => {
      setCurrency({ quotaDisplayType: legacyType, dongPerUsd: 25000 })

      await i18next.changeLanguage('vi')
      expect(getCurrencyDisplay().meta).toEqual({
        kind: 'currency',
        symbol: '₫',
        currencyCode: 'VND',
        exchangeRate: 25000,
      })

      await i18next.changeLanguage('en')
      expect(getCurrencyDisplay().meta).toEqual({
        kind: 'currency',
        symbol: '$',
        currencyCode: 'USD',
        exchangeRate: 1,
      })
    }
  )

  test('getCurrencyLabel reflects the resolved currency, not the stored legacy value', async () => {
    setCurrency({ quotaDisplayType: 'CNY', dongPerUsd: 25000 })

    await i18next.changeLanguage('vi')
    expect(getCurrencyLabel()).toBe('VND')

    await i18next.changeLanguage('en')
    expect(getCurrencyLabel()).toBe('USD')
  })
})

describe('Dong amounts render in whole units', () => {
  test('a fractional Dong conversion rounds to the nearest whole ₫ with grouped digits', async () => {
    await i18next.changeLanguage('vi')
    setCurrency({ quotaDisplayType: 'USD', dongPerUsd: 24567.89 })

    // 1.23456 * 24567.89 = 30330.53..., a fractional Dong amount that must
    // round to the nearest whole ₫ rather than truncate or show a decimal.
    expect(formatCurrencyFromUSD(1.23456, { locale: 'en' })).toBe('₫30,331')
  })

  test('a Dong amount below one unit still rounds to a whole number, never a fraction', async () => {
    await i18next.changeLanguage('vi')
    setCurrency({ quotaDisplayType: 'USD', dongPerUsd: 25000 })

    expect(formatCurrencyFromUSD(0.00001, { locale: 'en' })).not.toMatch(
      /[.,]\d/
    )
  })
})
