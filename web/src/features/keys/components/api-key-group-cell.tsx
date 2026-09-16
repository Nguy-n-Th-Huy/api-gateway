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

import { BadgeCell, TruncatedCell } from '@/components/data-table'
import { GroupBadge } from '@/components/group-badge'
import { StatusBadge } from '@/components/status-badge'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import {
  // AutoGroupBadge,
  GroupRatioBadge,
  type GroupRatio,
} from './auto-group-visuals'

type ApiKeyGroupCellProps = {
  group: string
  groups: string[]
  ratio?: GroupRatio
  groupRatios?: Record<string, GroupRatio>
  shouldReduceMotion: boolean
}

export function ApiKeyGroupCell(props: ApiKeyGroupCellProps) {
  const { t } = useTranslation()

  if (props.group !== 'auto') {
    const ratio = typeof props.ratio === 'number' ? props.ratio : undefined
    return (
      <TruncatedCell
        className='-ml-1.5'
        tooltipContent={props.group || '-'}
        tooltipClassName='break-all'
      >
        <GroupBadge group={props.group} ratio={ratio} />
      </TruncatedCell>
    )
  }

  // A key that names its own groups shows them: the stored order is the order the
  // relay path walks, so the chips and the tooltip both read it, and each chip
  // carries its own group's ratio. The Auto ratio badge would be a lie here — the
  // request is priced by the group that serves it, not by the placeholder.
  if (props.groups.length > 0) {
    return (
      <Tooltip>
        <TooltipTrigger
          render={
            <BadgeCell
              data-api-key-group-cell='auto'
              className='gap-1.5 overflow-visible text-xs'
            />
          }
        >
          <StatusBadge
            label={t('Cross-group')}
            variant='info'
            copyable={false}
          />
          {props.groups.map((group) => {
            const ratio = props.groupRatios?.[group]
            return (
              <GroupBadge
                key={group}
                group={group}
                ratio={typeof ratio === 'number' ? ratio : undefined}
              />
            )
          })}
        </TooltipTrigger>
        <TooltipContent>
          <span className='text-xs'>
            {t('Tries these groups in order: {{groups}}', {
              groups: props.groups.join(' → '),
            })}
          </span>
        </TooltipContent>
      </Tooltip>
    )
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <BadgeCell
            data-api-key-group-cell='auto'
            className='gap-1.5 overflow-visible text-xs'
          />
        }
      >
        <StatusBadge label={t('Cross-group')} variant='info' copyable={false} />
        {/*<AutoGroupBadge shouldReduceMotion={props.shouldReduceMotion} />*/}
        <GroupRatioBadge
          ratio={props.ratio}
          isAuto
          shouldReduceMotion={props.shouldReduceMotion}
        />
      </TooltipTrigger>
      <TooltipContent>
        <span className='text-xs'>
          {t(
            'Automatically selects the best available group with circuit breaker mechanism'
          )}
        </span>
      </TooltipContent>
    </Tooltip>
  )
}
