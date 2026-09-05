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

import { KEY_CHECK_MIN_LENGTH } from '../constants'

/**
 * Key-check form schema: the key is required, trimmed, and must be at least
 * `KEY_CHECK_MIN_LENGTH` characters after trimming. Empty and too-short
 * inputs get distinct localized messages, per
 * specs/public-key-check/spec.md — "Key check page validates input before
 * calling the API".
 */
export function getKeyCheckFormSchema(t: TFunction) {
  return z.object({
    key: z
      .string()
      .transform((value) => value.trim())
      .superRefine((value, ctx) => {
        if (value.length === 0) {
          ctx.addIssue({
            code: 'custom',
            message: t('Please enter an API key'),
          })
          return
        }
        if (value.length < KEY_CHECK_MIN_LENGTH) {
          ctx.addIssue({
            code: 'custom',
            message: t('The key must be at least {{min}} characters', {
              min: KEY_CHECK_MIN_LENGTH,
            }),
          })
        }
      }),
  })
}

export type KeyCheckFormValues = z.infer<ReturnType<typeof getKeyCheckFormSchema>>

export const KEY_CHECK_FORM_DEFAULT_VALUES: { key: string } = { key: '' }
