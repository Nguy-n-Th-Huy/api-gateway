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
import { z } from 'zod'

import { parseQuotaFromDollars, quotaUnitsToDollars } from '@/lib/format'

import type { ApiKey, ApiKeyFormData } from '../types'

// ============================================================================
// Form Schema
// ============================================================================

export function getApiKeyFormSchema(t: TFunction, maxAutoGroups = 5) {
  const groupLimit =
    Number.isInteger(maxAutoGroups) && maxAutoGroups > 0 ? maxAutoGroups : 5

  return z
    .object({
      name: z.string().min(1, t('Please enter a name')),
      remain_quota_dollars: z.number().optional(),
      expired_time: z.date().optional(),
      unlimited_quota: z.boolean(),
      model_limits: z.array(z.string()),
      allow_ips: z.string().optional(),
      groups: z.array(z.string()),
      use_global_auto: z.boolean(),
      cross_group_retry: z.boolean().optional(),
      tokenCount: z.number().min(1).optional(),
    })
    .superRefine((data, ctx) => {
      // The global Auto toggle owns the order: its list is empty by
      // construction, so the list checks only apply to an explicit selection.
      // An empty selection is legal — the key then follows its owner's group.
      if (!data.use_global_auto) {
        if (data.groups.length > groupLimit) {
          ctx.addIssue({
            code: 'custom',
            path: ['groups'],
            message: t('Select at most {{max}} groups', {
              max: groupLimit,
            }),
          })
        }

        if (new Set(data.groups).size !== data.groups.length) {
          ctx.addIssue({
            code: 'custom',
            path: ['groups'],
            message: t('Groups must not contain duplicates'),
          })
        }
      }

      if (data.unlimited_quota) {
        return
      }

      if (
        data.remain_quota_dollars === undefined ||
        data.remain_quota_dollars < 0
      ) {
        ctx.addIssue({
          code: 'custom',
          path: ['remain_quota_dollars'],
          message: t('Quota must be zero or greater'),
        })
      }
    })
}

export type ApiKeyFormValues = z.infer<ReturnType<typeof getApiKeyFormSchema>>

// ============================================================================
// Form Defaults
// ============================================================================

export const API_KEY_FORM_DEFAULT_VALUES: ApiKeyFormValues = {
  name: '',
  remain_quota_dollars: 10,
  expired_time: undefined,
  unlimited_quota: true,
  model_limits: [],
  allow_ips: '',
  groups: [],
  use_global_auto: false,
  cross_group_retry: false,
  tokenCount: 1,
}

export function getApiKeyFormDefaultValues(
  defaultUseAutoGroup: boolean
): ApiKeyFormValues {
  return {
    ...API_KEY_FORM_DEFAULT_VALUES,
    use_global_auto: defaultUseAutoGroup,
    cross_group_retry: defaultUseAutoGroup,
  }
}

// ============================================================================
// Form Data Transformation
// ============================================================================

/**
 * Map the ordered selection onto the stored key columns.
 *
 * One group stays a plain group so a single-group key keeps its today's
 * behaviour; two or more become an ordered snapshot behind the `auto`
 * placeholder, which is what the relay path already walks; the global Auto
 * toggle stores the placeholder alone; and an empty selection clears the group so
 * the key follows its owner's. The cross-group switch is only meaningful when the
 * key can span more than one group.
 */
function getStoredGroupSelection(data: ApiKeyFormValues): {
  group: string
  autoGroups: string[]
  crossGroupRetry: boolean
} {
  if (data.use_global_auto) {
    return {
      group: 'auto',
      autoGroups: [],
      crossGroupRetry: !!data.cross_group_retry,
    }
  }

  if (data.groups.length === 0) {
    return { group: '', autoGroups: [], crossGroupRetry: false }
  }

  const [onlyGroup] = data.groups
  if (data.groups.length === 1) {
    return { group: onlyGroup ?? '', autoGroups: [], crossGroupRetry: false }
  }

  return {
    group: 'auto',
    autoGroups: [...data.groups],
    crossGroupRetry: !!data.cross_group_retry,
  }
}

/**
 * Transform form data to API payload
 */
export function transformFormDataToPayload(
  data: ApiKeyFormValues
): ApiKeyFormData {
  const selection = getStoredGroupSelection(data)

  return {
    name: data.name,
    remain_quota: data.unlimited_quota
      ? 0
      : parseQuotaFromDollars(data.remain_quota_dollars || 0),
    expired_time: data.expired_time
      ? Math.floor(data.expired_time.getTime() / 1000)
      : -1,
    unlimited_quota: data.unlimited_quota,
    model_limits_enabled: data.model_limits.length > 0,
    model_limits: data.model_limits.join(','),
    allow_ips: data.allow_ips || '',
    group: selection.group,
    auto_groups: selection.autoGroups,
    cross_group_retry: selection.crossGroupRetry,
  }
}

/**
 * Transform API key data to form defaults
 *
 * The stored shape is what distinguishes the two `auto` keys: `auto` with a
 * snapshot is an explicit ordered list, `auto` alone is the global Auto order.
 * Every group the requester may no longer select is dropped, so the form can
 * never save a group the key would not be allowed to use.
 */
export function transformApiKeyToFormDefaults(
  apiKey: ApiKey,
  availableGroups: string[] = [],
  maxAutoGroups = 5
): ApiKeyFormValues {
  const availableSet = new Set(availableGroups)
  const limit = Math.max(0, maxAutoGroups)
  const storedSnapshot = apiKey.auto_groups ?? []
  const storedGroup = apiKey.group ?? ''

  let groups: string[] = []
  let useGlobalAuto = false
  if (storedGroup === 'auto') {
    if (storedSnapshot.length > 0) {
      groups = storedSnapshot
        .filter((group) => availableSet.has(group))
        .slice(0, limit)
    } else {
      useGlobalAuto = true
    }
  } else if (storedGroup !== '' && availableSet.has(storedGroup)) {
    groups = [storedGroup]
  }

  return {
    name: apiKey.name,
    remain_quota_dollars: apiKey.unlimited_quota
      ? 0
      : quotaUnitsToDollars(apiKey.remain_quota),
    expired_time:
      apiKey.expired_time > 0
        ? new Date(apiKey.expired_time * 1000)
        : undefined,
    unlimited_quota: apiKey.unlimited_quota,
    model_limits: apiKey.model_limits
      ? apiKey.model_limits.split(',').filter(Boolean)
      : [],
    allow_ips: apiKey.allow_ips || '',
    groups,
    use_global_auto: useGlobalAuto,
    cross_group_retry: !!apiKey.cross_group_retry,
    tokenCount: 1,
  }
}
