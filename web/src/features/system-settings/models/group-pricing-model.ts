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
import { safeJsonParse } from '../utils/json-parser'

export type GroupPricingRow = {
  _id: string
  name: string
  ratio: string
  topupRatio: string
  selectable: boolean
  adminOnly: boolean
  description: string
}

export type RegistryEntry = {
  name: string
  ratio: number
}

let groupPricingIdCounter = 0
export function createGroupPricingId() {
  groupPricingIdCounter += 1
  return `gpr_${groupPricingIdCounter}`
}

export function normalizeRatio(value: unknown): number {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : 1
}

export function parseRatioMap(value: string): Record<string, number> {
  return safeJsonParse<Record<string, number>>(value, {
    fallback: {},
    silent: true,
  })
}

export function parseUsableMap(value: string): Record<string, string> {
  return safeJsonParse<Record<string, string>>(value, {
    fallback: {},
    silent: true,
  })
}

export function parseNestedRatioMap(
  value: string
): Record<string, Record<string, number>> {
  return safeJsonParse<Record<string, Record<string, number>>>(value, {
    fallback: {},
    silent: true,
  })
}

export function parseAdminOnlySet(value: string): Set<string> {
  const list = safeJsonParse<string[]>(value, {
    fallback: [],
    silent: true,
  })
  return new Set(Array.isArray(list) ? list : [])
}

// canonicalAdminOnlyNames returns a sorted, de-duplicated name list so the
// unsaved-changes signature is order-independent: buildGroupPricingRows reads
// the flag from a stored array while serializeGroupPricingRows writes it back in
// row order, so a naive compare would report a false change on every load.
function canonicalAdminOnlyNames(names: Iterable<string>): string[] {
  return [...new Set(names)].sort()
}

export function buildGroupPricingRows(
  groupRatio: string,
  userUsableGroups: string,
  topupGroupRatio: string,
  adminOnlyGroups: string
): GroupPricingRow[] {
  const ratioMap = parseRatioMap(groupRatio)
  const usableMap = parseUsableMap(userUsableGroups)
  const topupMap = parseRatioMap(topupGroupRatio)
  const adminOnlySet = parseAdminOnlySet(adminOnlyGroups)
  const names = new Set([
    ...Object.keys(ratioMap),
    ...Object.keys(usableMap),
    ...Object.keys(topupMap),
  ])

  return [...names].map((name) => ({
    _id: createGroupPricingId(),
    name,
    ratio: String(normalizeRatio(ratioMap[name])),
    topupRatio: Object.hasOwn(topupMap, name) ? String(topupMap[name]) : '',
    selectable: Object.hasOwn(usableMap, name),
    adminOnly: adminOnlySet.has(name),
    description: String(usableMap[name] ?? ''),
  }))
}

export function serializeGroupPricingRows(rows: GroupPricingRow[]) {
  const groupRatio: Record<string, number> = {}
  const userUsableGroups: Record<string, string> = {}
  const topupGroupRatio: Record<string, number> = {}
  const adminOnlyGroups: string[] = []

  for (const row of rows) {
    const name = row.name.trim()
    if (!name) continue
    groupRatio[name] = normalizeRatio(row.ratio)
    if (row.selectable) {
      userUsableGroups[name] = row.description
    }
    const topup = row.topupRatio.trim()
    if (topup !== '' && Number.isFinite(Number(topup))) {
      topupGroupRatio[name] = Number(topup)
    }
    if (row.adminOnly) {
      adminOnlyGroups.push(name)
    }
  }

  return {
    GroupRatio: JSON.stringify(groupRatio, null, 2),
    UserUsableGroups: JSON.stringify(userUsableGroups, null, 2),
    TopupGroupRatio: JSON.stringify(topupGroupRatio, null, 2),
    AdminOnlyGroups: JSON.stringify(adminOnlyGroups, null, 2),
  }
}

export function groupPricingSignature(rows: GroupPricingRow[]): string {
  const serialized = serializeGroupPricingRows(rows)
  return JSON.stringify({
    groupRatio: parseRatioMap(serialized.GroupRatio),
    userUsableGroups: parseUsableMap(serialized.UserUsableGroups),
    topupGroupRatio: parseRatioMap(serialized.TopupGroupRatio),
    adminOnlyGroups: canonicalAdminOnlyNames(
      parseAdminOnlySet(serialized.AdminOnlyGroups)
    ),
  })
}

export function sourceGroupPricingSignature(
  groupRatio: string,
  userUsableGroups: string,
  topupGroupRatio: string,
  adminOnlyGroups: string
): string {
  const ratioMap = parseRatioMap(groupRatio)
  const usableMap = parseUsableMap(userUsableGroups)
  const topupMap = parseRatioMap(topupGroupRatio)
  // Compare only what a row can represent. buildGroupPricingRows never creates a
  // row for a marked name that is absent from the registry, so including such an
  // inert name here would keep this signature permanently apart from
  // groupPricingSignature(rows) and rebuild the rows — with fresh ids — on every
  // render until the first edit.
  const registryNames = new Set([
    ...Object.keys(ratioMap),
    ...Object.keys(usableMap),
    ...Object.keys(topupMap),
  ])
  const markedNames = [...parseAdminOnlySet(adminOnlyGroups)].filter((name) =>
    registryNames.has(name)
  )

  return JSON.stringify({
    groupRatio: ratioMap,
    userUsableGroups: usableMap,
    topupGroupRatio: topupMap,
    adminOnlyGroups: canonicalAdminOnlyNames(markedNames),
  })
}
