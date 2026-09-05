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
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { APPLICATIONS, OS_TARGETS, type OsTarget } from '../lib/applications'

export interface SetupApplicationPickerProps {
  appId: string
  onAppIdChange: (appId: string) => void
  os: OsTarget
  onOsChange: (os: OsTarget) => void
}

/** The application selector and the Windows / macOS-Linux OS tabs. See
 * specs/public-setup-script/spec.md — "Setup section offers the supported
 * applications and operating systems". */
export function SetupApplicationPicker(props: SetupApplicationPickerProps) {
  const { t } = useTranslation()

  return (
    <div className='flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between'>
      <div className='flex flex-col gap-1'>
        <Label>{t('Application')}</Label>
        <NativeSelect
          value={props.appId}
          onChange={(event) => props.onAppIdChange(event.target.value)}
        >
          {APPLICATIONS.map((option) => (
            <NativeSelectOption key={option.id} value={option.id}>
              {t(option.labelKey)}
            </NativeSelectOption>
          ))}
        </NativeSelect>
      </div>

      <Tabs
        value={props.os}
        onValueChange={(value) => props.onOsChange(value as OsTarget)}
      >
        <TabsList>
          <TabsTrigger value={OS_TARGETS.WINDOWS}>{t('Windows')}</TabsTrigger>
          <TabsTrigger value={OS_TARGETS.UNIX}>{t('macOS/Linux')}</TabsTrigger>
        </TabsList>
      </Tabs>
    </div>
  )
}
