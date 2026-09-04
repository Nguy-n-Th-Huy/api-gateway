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

import { AnimateInView } from '@/components/animate-in-view'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import { getComparisonRows } from '../../constants'

interface ComparisonProps {
  className?: string
}

/**
 * Contrasts direct provider access with going through the gateway. Renders
 * as a table from md up, and as stacked cards below md so nothing forces
 * horizontal scrolling of the page body.
 */
export function Comparison(_props: ComparisonProps) {
  const { t } = useTranslation()
  const rows = getComparisonRows(t)

  return (
    <section className='relative z-10 px-6 py-24 md:py-32'>
      <div className='mx-auto max-w-6xl'>
        <AnimateInView className='mb-12 max-w-2xl'>
          <p className='text-muted-foreground mb-3 text-xs font-medium tracking-widest uppercase'>
            {t('Comparison')}
          </p>
          <h2 className='text-2xl leading-tight font-bold tracking-tight md:text-3xl'>
            {t('Why not buy directly from OpenAI or Anthropic?')}
          </h2>
          <p className='text-muted-foreground mt-4 max-w-xl text-sm leading-relaxed md:text-base'>
            {t(
              'Buying direct still makes sense if you only use one provider and hold an international card. Otherwise, here is the difference.'
            )}
          </p>
        </AnimateInView>

        {/* Table — md and up */}
        <div className='border-border/40 bg-card hidden overflow-hidden rounded-xl border md:block'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Criteria')}</TableHead>
                <TableHead>{t('Direct purchase')}</TableHead>
                <TableHead className='text-primary'>
                  {t('Via the gateway')}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row.criterion}>
                  <TableCell className='font-medium'>
                    {row.criterion}
                  </TableCell>
                  <TableCell className='text-muted-foreground'>
                    {row.direct}
                  </TableCell>
                  <TableCell className='font-medium'>{row.gateway}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>

        {/* Stacked cards — below md */}
        <ul className='flex flex-col gap-3 md:hidden'>
          {rows.map((row) => (
            <li
              key={row.criterion}
              className='border-border bg-card flex flex-col gap-3 rounded-2xl border p-5'
            >
              <span className='text-sm font-semibold'>{row.criterion}</span>
              <div className='flex flex-col gap-1.5 text-sm'>
                <span className='text-muted-foreground'>
                  {t('Direct purchase')}:{' '}
                  <span className='text-foreground'>{row.direct}</span>
                </span>
                <span className='text-muted-foreground'>
                  {t('Via the gateway')}:{' '}
                  <span className='text-primary font-medium'>
                    {row.gateway}
                  </span>
                </span>
              </div>
            </li>
          ))}
        </ul>
      </div>
    </section>
  )
}
