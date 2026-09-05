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

import { StatusBadge } from '@/components/status-badge'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import type { ModelBarColor, ModelStatusEntry } from '../lib/model-status'

const STATE_LABEL_KEY: Record<ModelStatusEntry['state'], string> = {
  operational: 'Operational',
  down: 'Down',
  'no-data': 'No Data',
}

const STATE_VARIANT: Record<
  ModelStatusEntry['state'],
  'success' | 'danger' | 'neutral'
> = {
  operational: 'success',
  down: 'danger',
  'no-data': 'neutral',
}

const BAR_COLOR_CLASS: Record<ModelBarColor, string> = {
  healthy: 'bg-success',
  degraded: 'bg-warning',
  failing: 'bg-destructive',
  neutral: 'bg-muted',
}

export interface ModelStatusRowProps {
  entry: ModelStatusEntry
}

export function ModelStatusRow(props: ModelStatusRowProps) {
  const { t } = useTranslation()
  const { entry } = props

  return (
    <li className='flex items-center justify-between gap-3'>
      <div className='flex min-w-0 items-center gap-2'>
        <span className='truncate text-sm font-medium'>
          {entry.modelName}
        </span>
        <StatusBadge
          variant={STATE_VARIANT[entry.state]}
          label={t(STATE_LABEL_KEY[entry.state])}
          copyable={false}
        />
      </div>
      {/*
        role="group" (not "img") keeps this a normal grouping container: an
        "img" role would flatten every descendant — including each bar's own
        accessible name — into a single opaque node, hiding the per-bar
        detail from assistive technology. See
        specs/public-model-status/spec.md — "Each model shows a bar strip of
        recent intervals": colour must never be the only carrier of meaning,
        and each bar must expose its own interval and success rate.
      */}
      <div
        className='flex shrink-0 items-end gap-0.5'
        role='group'
        aria-label={t('Recent interval health for {{model}}', {
          model: entry.modelName,
        })}
      >
        {entry.bars.map((bar, index) => (
          <ModelStatusBar
            // eslint-disable-next-line react/no-array-index-key
            key={index}
            index={index}
            total={entry.bars.length}
            successRate={bar.successRate}
            color={bar.color}
          />
        ))}
      </div>
    </li>
  )
}

function ModelStatusBar(props: {
  index: number
  total: number
  successRate: number | null
  color: ModelBarColor
}) {
  const { t } = useTranslation()
  const agoCount = props.total - 1 - props.index
  const interval =
    agoCount === 0
      ? t('Most recent interval')
      : t('{{count}} intervals ago', { count: agoCount })
  const rate =
    props.successRate == null
      ? t('No recorded traffic')
      : t('{{rate}}% success', { rate: props.successRate })
  const detail = t('{{interval}} · {{rate}}', { interval, rate })

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <button
            type='button'
            tabIndex={0}
            aria-label={detail}
            className={cn(
              'h-5 w-1.5 rounded-sm',
              BAR_COLOR_CLASS[props.color]
            )}
          />
        }
      />
      <TooltipContent>{detail}</TooltipContent>
    </Tooltip>
  )
}
