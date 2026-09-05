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

For commercial licensing, please contact support@quantumnous.com
*/
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'

import {
  formatAvgLatency,
  getAvgLatencyConfig,
  type ChannelHealthCellView,
} from '../lib'

/**
 * Success-rate cell for the channel health column. A channel absent from the
 * health lookup (`!view.hasData`) renders the no-data indicator instead of a
 * fabricated `0%`, so an idle channel is never mistaken for a failing one.
 *
 * Kept in its own module (rather than inline in `channels-columns.tsx`) so it
 * can be unit-tested without importing that file's much larger, icon-heavy
 * dependency graph.
 */
export function SuccessRateCell({ view }: { view: ChannelHealthCellView }) {
  if (!view.hasData) {
    return <span className='text-muted-foreground text-xs'>-</span>
  }
  return (
    <StatusBadge
      label={view.successRateLabel}
      variant={view.successRateVariant}
      size='sm'
      copyable={false}
      className='-ml-1.5'
    />
  )
}

/**
 * Average-latency cell for the channel health column. A channel absent from
 * the health lookup (`!view.hasData`) renders the no-data indicator; a
 * present channel with `avgLatencyMs === 0` renders "< 1s" with a healthy
 * badge, since that means every request completed in under one second (see
 * `formatAvgLatency`), not that the channel was never tested.
 */
export function AvgLatencyCell({ view }: { view: ChannelHealthCellView }) {
  const { t } = useTranslation()
  if (!view.hasData) {
    return <span className='text-muted-foreground text-xs'>-</span>
  }
  const config = getAvgLatencyConfig(view.avgLatencyMs)
  return (
    <StatusBadge
      label={formatAvgLatency(view.avgLatencyMs, t)}
      variant={config.variant}
      size='sm'
      copyable={false}
      className='-ml-1.5'
    />
  )
}
