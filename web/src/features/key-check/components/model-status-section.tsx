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

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Empty, EmptyDescription, EmptyTitle } from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'

import { useModelStatus } from '../hooks/use-model-status'
import { ModelStatusRow } from './model-status-row'

export interface ModelStatusSectionProps {
  /** `null` before any key is checked; the checked key's `available_models`
   * afterwards. */
  availableModels: string[] | null
}

/** Model health section — independent of the key-check form's own state, so
 * a metrics failure never blocks it. See specs/public-model-status/spec.md. */
export function ModelStatusSection(props: ModelStatusSectionProps) {
  const { t } = useTranslation()
  const { entries, isLoading, isError, refetch } = useModelStatus(
    props.availableModels
  )

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('Model status')}</CardTitle>
        {props.availableModels != null && (
          <CardDescription>
            {t("Scoped to this key's group")}
          </CardDescription>
        )}
      </CardHeader>
      <CardContent>
        {isLoading && (
          <div className='flex flex-col gap-2'>
            <Skeleton className='h-6 w-full' />
            <Skeleton className='h-6 w-full' />
            <Skeleton className='h-6 w-full' />
          </div>
        )}

        {!isLoading && isError && (
          <Alert variant='destructive'>
            <AlertDescription className='flex items-center justify-between gap-3'>
              <span>{t('Failed to load model status')}</span>
              <Button variant='outline' size='sm' onClick={refetch}>
                {t('Retry')}
              </Button>
            </AlertDescription>
          </Alert>
        )}

        {!isLoading && !isError && entries.length === 0 && (
          <Empty>
            <EmptyTitle>{t('No model data yet')}</EmptyTitle>
            <EmptyDescription>
              {t('No model has recorded traffic in the last 24 hours.')}
            </EmptyDescription>
          </Empty>
        )}

        {!isLoading && !isError && entries.length > 0 && (
          <ul
            className='flex flex-col gap-3'
            aria-label={t('Model status')}
          >
            {entries.map((entry) => (
              <ModelStatusRow key={entry.modelName} entry={entry} />
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}
