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
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'

import {
  getAccessedTimeDisplay,
  getCreatedTimeDisplay,
  getExpiryDisplay,
  getModelRestrictionDisplay,
  getQuotaDisplay,
  getStatusEntry,
} from '../lib/report-display'
import type { KeyCheckReport } from '../types'

export interface ResultPanelProps {
  report: KeyCheckReport
}

/**
 * The full key report. Never receives or renders the submitted key — see
 * specs/public-key-check/spec.md — "Key check page does not re-display the
 * full key".
 */
export function ResultPanel(props: ResultPanelProps) {
  const { t } = useTranslation()
  const { report } = props

  const statusEntry = getStatusEntry(report.status)
  const quota = getQuotaDisplay(report)
  const expiry = getExpiryDisplay(report.expires_at)
  const modelRestriction = getModelRestrictionDisplay(report)

  return (
    <Card>
      <CardHeader>
        <CardTitle className='truncate'>{report.name}</CardTitle>
      </CardHeader>
      <CardContent className='grid gap-4 sm:grid-cols-2'>
        <Field label={t('Group')}>{report.group}</Field>
        <Field label={t('Status')}>
          <StatusBadge
            variant={statusEntry.variant}
            label={t(statusEntry.label)}
            copyable={false}
          />
        </Field>

        <Field label={t('Quota')} className='sm:col-span-2'>
          {quota.unlimited ? (
            <span>{t('Unlimited')}</span>
          ) : (
            <div className='flex flex-col gap-1.5'>
              <span className='text-muted-foreground text-xs'>
                {t('{{used}} used of {{granted}} ({{remaining}} remaining)', {
                  used: quota.totalUsed,
                  granted: quota.totalGranted,
                  remaining: quota.totalAvailable,
                })}
              </span>
              <Progress value={quota.usedSharePercent ?? 0} />
            </div>
          )}
        </Field>

        <Field label={t('Expires')}>
          {expiry ?? t('Never expires')}
        </Field>
        <Field label={t('Created')}>{getCreatedTimeDisplay(report.created_time)}</Field>
        <Field label={t('Last used')}>
          {getAccessedTimeDisplay(report.accessed_time)}
        </Field>

        <Field label={t('Allowed models')} className='sm:col-span-2'>
          {modelRestriction.allAllowed ? (
            <span>{t('All models allowed')}</span>
          ) : (
            <span className='break-words'>
              {modelRestriction.models.join(', ')}
            </span>
          )}
        </Field>
      </CardContent>
    </Card>
  )
}

function Field(props: {
  label: string
  className?: string
  children: React.ReactNode
}) {
  return (
    <div className={props.className}>
      <div className='text-muted-foreground text-xs'>{props.label}</div>
      <div className='text-sm'>{props.children}</div>
    </div>
  )
}
