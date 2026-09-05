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
  buildGroupPricingRows,
  groupPricingSignature,
  serializeGroupPricingRows,
  sourceGroupPricingSignature,
} from '../group-pricing-model'

const GROUP_RATIO = '{"default":1,"vip":1,"secret":1}'
const USABLE = '{"default":"D","vip":"V","secret":"S"}'
const TOPUP = '{"default":1}'

function rowByName(
  rows: ReturnType<typeof buildGroupPricingRows>,
  name: string
) {
  const found = rows.find((row) => row.name === name)
  if (!found) throw new Error(`row ${name} not built`)
  return found
}

describe('admin-only flag round-trip in the group editor', () => {
  test('buildGroupPricingRows reads the flag for names already in the registry', () => {
    const rows = buildGroupPricingRows(
      GROUP_RATIO,
      USABLE,
      TOPUP,
      '["secret"]'
    )

    expect(rowByName(rows, 'secret').adminOnly).toBe(true)
    expect(rowByName(rows, 'default').adminOnly).toBe(false)
    expect(rowByName(rows, 'vip').adminOnly).toBe(false)
    // The flag is independent of the other settings — secret is still usable.
    expect(rowByName(rows, 'secret').selectable).toBe(true)
    expect(rowByName(rows, 'secret').ratio).toBe('1')
  })

  test('an admin-only name absent from the registry does not create a row', () => {
    const rows = buildGroupPricingRows(
      GROUP_RATIO,
      USABLE,
      TOPUP,
      '["ghost"]'
    )

    expect(rows.map((row) => row.name)).toEqual(['default', 'vip', 'secret'])
    expect(rows.some((row) => row.name === 'ghost')).toBe(false)
  })

  test('serializeGroupPricingRows writes only the marked names as a JSON array', () => {
    const rows = buildGroupPricingRows(
      GROUP_RATIO,
      USABLE,
      TOPUP,
      '["secret","vip"]'
    )
    const serialized = serializeGroupPricingRows(rows)

    expect(JSON.parse(serialized.AdminOnlyGroups)).toEqual(['vip', 'secret'])
  })

  test('a marked group deleted from the registry is cleaned up on the next save', () => {
    // "ghost" was marked but no longer exists as a row, so serializing drops it.
    const rows = buildGroupPricingRows(GROUP_RATIO, USABLE, TOPUP, '["ghost"]')
    const serialized = serializeGroupPricingRows(rows)

    expect(JSON.parse(serialized.AdminOnlyGroups)).toEqual([])
  })

  test('build then serialize then build preserves the flag', () => {
    const initial = buildGroupPricingRows(
      GROUP_RATIO,
      USABLE,
      TOPUP,
      '["secret"]'
    )
    const serialized = serializeGroupPricingRows(initial)
    const rebuilt = buildGroupPricingRows(
      serialized.GroupRatio,
      serialized.UserUsableGroups,
      serialized.TopupGroupRatio,
      serialized.AdminOnlyGroups
    )

    expect(rowByName(rebuilt, 'secret').adminOnly).toBe(true)
    expect(rowByName(rebuilt, 'default').adminOnly).toBe(false)
  })

  test('the unsaved-changes signature reacts to toggling the flag', () => {
    const rows = buildGroupPricingRows(
      GROUP_RATIO,
      USABLE,
      TOPUP,
      '["secret"]'
    )
    const inSync = sourceGroupPricingSignature(
      GROUP_RATIO,
      USABLE,
      TOPUP,
      '["secret"]'
    )
    expect(groupPricingSignature(rows)).toBe(inSync)

    const toggled = rows.map((row) =>
      row.name === 'default' ? { ...row, adminOnly: true } : row
    )
    expect(groupPricingSignature(toggled)).not.toBe(inSync)
  })

  test('a marked name absent from the registry keeps the guard clean', () => {
    // "ghost" is marked but has no row, and no rebuild can ever produce a row for
    // it. The source signature must compare only names a row can represent, or the
    // editor rebuilds the rows with fresh ids on every render until the first edit.
    const rows = buildGroupPricingRows(
      GROUP_RATIO,
      USABLE,
      TOPUP,
      '["secret","ghost"]'
    )
    const inSync = sourceGroupPricingSignature(
      GROUP_RATIO,
      USABLE,
      TOPUP,
      '["secret","ghost"]'
    )

    expect(rows.map((row) => row.name)).toEqual(['default', 'vip', 'secret'])
    expect(groupPricingSignature(rows)).toBe(inSync)
  })

  test('the signature is independent of admin-only list order', () => {
    const rows = buildGroupPricingRows(
      GROUP_RATIO,
      USABLE,
      TOPUP,
      '["secret","vip"]'
    )
    const sourceReordered = sourceGroupPricingSignature(
      GROUP_RATIO,
      USABLE,
      TOPUP,
      '["vip","secret"]'
    )

    expect(groupPricingSignature(rows)).toBe(sourceReordered)
  })

  test('an empty admin-only list keeps resolution byte-identical across roles', () => {
    const empty = buildGroupPricingRows(GROUP_RATIO, USABLE, TOPUP, '[]')
    const sourceEmpty = sourceGroupPricingSignature(
      GROUP_RATIO,
      USABLE,
      TOPUP,
      '[]'
    )

    expect(empty.every((row) => row.adminOnly === false)).toBe(true)
    expect(groupPricingSignature(empty)).toBe(sourceEmpty)
  })
})
