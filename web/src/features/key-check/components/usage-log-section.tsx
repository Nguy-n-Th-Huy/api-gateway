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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import { useTokenLogs } from '../hooks/use-token-logs'
import { getUsageLogRows } from '../lib/log-display'

export interface UsageLogSectionProps {
  /** The raw key from the last successful check; `null` before any check.
   * Only ever sent to the log endpoint, never rendered. */
  checkedKey: string | null
}

/** The checked key's own usage log. Gated on a successful check and loaded
 * independently of the report, so a log failure never blocks it. See
 * specs/public-key-check/spec.md, "Key check page presents the checked key's
 * usage log". */
export function UsageLogSection(props: UsageLogSectionProps) {
  const { t } = useTranslation()
  const logs = useTokenLogs(props.checkedKey)
  const rows = useMemo(() => getUsageLogRows(logs.entries), [logs.entries])

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('Usage logs')}</CardTitle>
        <CardDescription>
          {t('The latest requests made with this key.')}
        </CardDescription>
      </CardHeader>
      <CardContent className='flex flex-col gap-4'>
        {props.checkedKey === null && (
          <p className='text-muted-foreground text-sm'>
            {t('Check a key above to see its usage log.')}
          </p>
        )}

        {props.checkedKey !== null && logs.isLoading && (
          <div className='flex flex-col gap-2'>
            <Skeleton className='h-6 w-full' />
            <Skeleton className='h-6 w-full' />
            <Skeleton className='h-6 w-full' />
          </div>
        )}

        {props.checkedKey !== null && !logs.isLoading && logs.isError && (
          <Alert variant='destructive'>
            <AlertDescription className='flex items-center justify-between gap-3'>
              <span>{t('Failed to load the usage log')}</span>
              <Button variant='outline' size='sm' onClick={logs.refetch}>
                {t('Retry')}
              </Button>
            </AlertDescription>
          </Alert>
        )}

        {props.checkedKey !== null &&
          !logs.isLoading &&
          !logs.isError &&
          rows.length === 0 && (
            <Empty>
              <EmptyTitle>{t('No usage recorded yet')}</EmptyTitle>
              <EmptyDescription>
                {t('Requests made with this key will appear here.')}
              </EmptyDescription>
            </Empty>
          )}

        {props.checkedKey !== null &&
          !logs.isLoading &&
          !logs.isError &&
          rows.length > 0 && (
            <div className='flex flex-col gap-3'>
              <div className='overflow-x-auto'>
                <Table aria-label={t('Usage logs')}>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('Time')}</TableHead>
                      <TableHead>{t('Model')}</TableHead>
                      <TableHead>{t('Type')}</TableHead>
                      <TableHead>{t('Status')}</TableHead>
                      <TableHead className='text-right'>{t('Input')}</TableHead>
                      <TableHead className='text-right'>{t('Cache')}</TableHead>
                      <TableHead className='text-right'>{t('Output')}</TableHead>
                      <TableHead className='text-right'>{t('Cost')}</TableHead>
                      <TableHead className='text-right'>
                        {t('Duration')}
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {rows.map((row) => (
                      <TableRow key={row.id}>
                        <TableCell className='whitespace-nowrap'>
                          {row.time}
                        </TableCell>
                        <TableCell
                          className='max-w-[12rem] truncate'
                          title={row.modelName}
                        >
                          {row.modelName}
                        </TableCell>
                        <TableCell>
                          <StatusBadge
                            variant='neutral'
                            label={t(row.typeLabel)}
                            copyable={false}
                          />
                        </TableCell>
                        <TableCell>
                          {row.outcome ? (
                            <StatusBadge
                              variant={row.outcome.variant}
                              label={t(row.outcome.label)}
                              copyable={false}
                            />
                          ) : (
                            <span className='text-muted-foreground'>-</span>
                          )}
                        </TableCell>
                        <TableCell className='text-right tabular-nums'>
                          {row.inputTokens}
                        </TableCell>
                        <TableCell className='text-right tabular-nums'>
                          {row.cacheTokens}
                        </TableCell>
                        <TableCell className='text-right tabular-nums'>
                          {row.outputTokens}
                        </TableCell>
                        <TableCell className='text-right tabular-nums'>
                          {row.cost}
                        </TableCell>
                        <TableCell className='text-right tabular-nums'>
                          {row.duration}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              <div className='flex items-center justify-between gap-3'>
                <span className='text-muted-foreground text-xs'>
                  {t('Page {{current}} of {{total}}', {
                    current: logs.page,
                    total: logs.pageCount,
                  })}
                </span>
                <div className='flex gap-2'>
                  <Button
                    variant='outline'
                    size='sm'
                    disabled={logs.page <= 1}
                    onClick={() => logs.goToPage(logs.page - 1)}
                  >
                    {t('Previous')}
                  </Button>
                  <Button
                    variant='outline'
                    size='sm'
                    disabled={logs.page >= logs.pageCount}
                    onClick={() => logs.goToPage(logs.page + 1)}
                  >
                    {t('Next')}
                  </Button>
                </div>
              </div>
            </div>
          )}
      </CardContent>
    </Card>
  )
}