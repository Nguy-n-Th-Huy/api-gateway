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
import { useTranslation } from 'react-i18next'

import { Label } from '@/components/ui/label'
import {
  NativeSelect,
  NativeSelectOption,
} from '@/components/ui/native-select'

import type { AppConfig } from '../lib/applications'

export interface SetupModelSelectorsProps {
  app: AppConfig
  availableModels: string[]
  modelSelections: Record<string, string>
  onSelectionChange: (slotId: string, model: string) => void
  disabled: boolean
}

/**
 * One labelled model selector per slot the application defines, populated
 * exclusively from `available_models`. Renders a note instead for an
 * application with no documented model-role configuration (Pi, Oh My Pi).
 * See specs/public-setup-script/spec.md — "Setup section states which slots
 * map to which CLI setting" and "Model choices come only from the key's
 * group".
 */
export function SetupModelSelectors(props: SetupModelSelectorsProps) {
  const { t } = useTranslation()

  if (props.app.slots.length === 0) {
    return (
      <p className='text-muted-foreground text-sm'>
        {t(
          'This CLI has no model configuration - choose the model inside it.'
        )}
      </p>
    )
  }

  return (
    <div className='grid gap-3 sm:grid-cols-2'>
      {props.app.slots.map((slot) => (
        <div key={slot.id} className='flex flex-col gap-1'>
          <Label>{t(slot.labelKey)}</Label>
          <NativeSelect
            value={props.modelSelections[slot.id] ?? ''}
            disabled={props.disabled}
            onChange={(event) =>
              props.onSelectionChange(slot.id, event.target.value)
            }
          >
            {props.availableModels.map((model) => (
              <NativeSelectOption key={model} value={model}>
                {model}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </div>
      ))}
    </div>
  )
}
