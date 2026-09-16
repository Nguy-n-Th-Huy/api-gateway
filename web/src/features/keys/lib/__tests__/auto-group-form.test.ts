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
import type { TFunction } from 'i18next'
import { describe, expect, test } from 'vitest'

import { apiKeySchema, type ApiKey } from '../../types'
import {
  getApiKeyFormDefaultValues,
  getApiKeyFormSchema,
  transformApiKeyToFormDefaults,
  transformFormDataToPayload,
} from '../api-key-form'

const t = ((key: string, options?: Record<string, unknown>) => {
  if (options?.max !== undefined) {
    return key.replace('{{max}}', String(options.max))
  }
  return key
}) as TFunction

const baseApiKey: ApiKey = {
  id: 1,
  name: 'test',
  key: 'sk-test',
  status: 1,
  remain_quota: 0,
  used_quota: 0,
  unlimited_quota: true,
  expired_time: -1,
  created_time: 1,
  accessed_time: 0,
  group: 'auto',
  auto_groups: null,
  cross_group_retry: true,
  model_limits_enabled: false,
  model_limits: '',
  allow_ips: '',
}

describe('API key group form mapping', () => {
  test('treats legacy token responses without auto_groups as the global Auto order', () => {
    const legacyApiKey: Record<string, unknown> = { ...baseApiKey }
    delete legacyApiKey.auto_groups

    expect(apiKeySchema.parse(legacyApiKey).auto_groups).toBe(null)
  })

  test('defaults a create to no groups and no global Auto order', () => {
    const defaults = getApiKeyFormDefaultValues(false)

    expect(defaults.groups).toEqual([])
    expect(defaults.use_global_auto).toBe(false)
    expect(defaults.cross_group_retry).toBe(false)
  })

  test('defaults a create to the global Auto order when the deployment asks for it', () => {
    const defaults = getApiKeyFormDefaultValues(true)

    expect(defaults.use_global_auto).toBe(true)
    expect(defaults.cross_group_retry).toBe(true)
    expect(defaults.groups).toEqual([])

    const payload = transformFormDataToPayload({ ...defaults, name: 'create' })
    expect(payload.group).toBe('auto')
    expect(payload.auto_groups).toEqual([])
  })

  test('creates a key with no group from an empty selection', () => {
    const payload = transformFormDataToPayload({
      ...getApiKeyFormDefaultValues(false),
      name: 'ungrouped',
    })

    expect(payload.group).toBe('')
    expect(payload.auto_groups).toEqual([])
    expect(payload.cross_group_retry).toBe(false)
  })

  test('creates a single-group key from one selected group', () => {
    const payload = transformFormDataToPayload({
      ...getApiKeyFormDefaultValues(false),
      name: 'single',
      groups: ['vip'],
      cross_group_retry: true,
    })

    expect(payload.group).toBe('vip')
    expect(payload.auto_groups).toEqual([])
    expect(payload.cross_group_retry).toBe(false)
  })

  test('creates a multi-group key as an ordered snapshot in the selected order', () => {
    const payload = transformFormDataToPayload({
      ...getApiKeyFormDefaultValues(false),
      name: 'multi',
      groups: ['vip', 'default', 'team'],
      cross_group_retry: false,
    })

    expect(payload.group).toBe('auto')
    expect(payload.auto_groups).toEqual(['vip', 'default', 'team'])
    expect(payload.cross_group_retry).toBe(false)
  })

  test('keeps the global Auto order snapshot-free even when the list still holds groups', () => {
    const payload = transformFormDataToPayload({
      ...getApiKeyFormDefaultValues(true),
      name: 'global-auto',
      groups: ['vip', 'default'],
    })

    expect(payload.group).toBe('auto')
    expect(payload.auto_groups).toEqual([])
  })

  test('maps a stored global Auto order onto the toggle, not onto a list', () => {
    const legacyApiKey: Record<string, unknown> = { ...baseApiKey }
    delete legacyApiKey.auto_groups
    const globalAutoKeys = [
      apiKeySchema.parse(legacyApiKey),
      baseApiKey,
      { ...baseApiKey, auto_groups: [] },
    ]

    for (const apiKey of globalAutoKeys) {
      const defaults = transformApiKeyToFormDefaults(
        apiKey,
        ['default', 'vip'],
        2
      )

      expect(defaults.use_global_auto).toBe(true)
      expect(defaults.groups).toEqual([])
      expect(defaults.cross_group_retry).toBe(true)
    }
  })

  test('maps a stored snapshot onto the explicit list in its stored order', () => {
    const defaults = transformApiKeyToFormDefaults(
      { ...baseApiKey, auto_groups: ['vip', 'default'] },
      ['default', 'vip'],
      5
    )

    expect(defaults.use_global_auto).toBe(false)
    expect(defaults.groups).toEqual(['vip', 'default'])
  })

  test('filters a stored snapshot before applying a lowered limit', () => {
    const defaults = transformApiKeyToFormDefaults(
      { ...baseApiKey, auto_groups: ['revoked', 'vip', 'default'] },
      ['default', 'vip'],
      2
    )

    expect(defaults.use_global_auto).toBe(false)
    expect(defaults.groups).toEqual(['vip', 'default'])
  })

  test('round-trips a stored snapshot inside the limit unchanged', () => {
    const stored = { ...baseApiKey, auto_groups: ['vip', 'default'] }

    const payload = transformFormDataToPayload(
      transformApiKeyToFormDefaults(stored, ['default', 'vip'], 5)
    )

    expect(payload.group).toBe('auto')
    expect(payload.auto_groups).toEqual(['vip', 'default'])
    expect(payload.cross_group_retry).toBe(true)
  })

  test('round-trips a stored single group unchanged', () => {
    const stored = {
      ...baseApiKey,
      group: 'vip',
      auto_groups: [],
      cross_group_retry: false,
    }

    const payload = transformFormDataToPayload(
      transformApiKeyToFormDefaults(stored, ['vip', 'default'], 5)
    )

    expect(payload.group).toBe('vip')
    expect(payload.auto_groups).toEqual([])
    expect(payload.cross_group_retry).toBe(false)
  })

  test('drops a stored single group the requester may no longer select', () => {
    const defaults = transformApiKeyToFormDefaults(
      { ...baseApiKey, group: 'revoked', auto_groups: [] },
      ['default', 'vip'],
      5
    )

    expect(defaults.groups).toEqual([])
    expect(defaults.use_global_auto).toBe(false)
    expect(transformFormDataToPayload(defaults).group).toBe('')
  })

  test('leaves a fully filtered snapshot as an empty selection instead of global Auto', () => {
    const defaults = transformApiKeyToFormDefaults(
      { ...baseApiKey, auto_groups: ['revoked'] },
      ['default'],
      2
    )

    expect(defaults.use_global_auto).toBe(false)
    expect(defaults.groups).toEqual([])

    const result = getApiKeyFormSchema(t, 2).safeParse(defaults)
    expect(result.success).toBe(true)

    const payload = transformFormDataToPayload(defaults)
    expect(payload.group).toBe('')
    expect(payload.auto_groups).toEqual([])
    expect(payload.cross_group_retry).toBe(false)
  })

  test('accepts an empty selection', () => {
    const result = getApiKeyFormSchema(t, 2).safeParse({
      ...getApiKeyFormDefaultValues(false),
      name: 'ungrouped',
    })

    expect(result.success).toBe(true)
  })

  test('rejects selections over the configured limit', () => {
    const result = getApiKeyFormSchema(t, 1).safeParse({
      ...getApiKeyFormDefaultValues(false),
      name: 'limited token',
      groups: ['default', 'vip'],
    })

    expect(result.success).toBe(false)
    if (result.success) return
    expect(result.error.issues[0]?.path[0]).toBe('groups')
    expect(result.error.issues[0]?.message).toBe('Select at most 1 groups')
  })

  test('rejects duplicate groups', () => {
    const result = getApiKeyFormSchema(t).safeParse({
      ...getApiKeyFormDefaultValues(false),
      name: 'duplicate token',
      groups: ['vip', 'vip'],
    })

    expect(result.success).toBe(false)
    if (result.success) return
    expect(result.error.issues[0]?.path[0]).toBe('groups')
    expect(result.error.issues[0]?.message).toBe(
      'Groups must not contain duplicates'
    )
  })
})
