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
import { Link } from '@tanstack/react-router'
import { RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { Alert, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Empty, EmptyTitle } from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { DEFAULT_TOKEN_UNIT } from '@/features/pricing/constants'
import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'
import { formatPrice } from '@/features/pricing/lib/price'

interface PricingPreviewProps {
  className?: string
}

// Small, fixed preview of the live pricing table; the full catalog lives on
// the dedicated pricing page (same `['pricing']` query cache, per design.md).
const PREVIEW_MODEL_LIMIT = 6

/**
 * Bounded preview of the public pricing data. Owns its own loading, error
 * and empty treatments per the web-home-page spec — the block never hides
 * itself away, unlike the FAQ block.
 */
export function PricingPreview(_props: PricingPreviewProps) {
  const { t } = useTranslation()
  const { models, isLoading, error, refetch, priceRate, usdExchangeRate } =
    usePricingData()
  const previewModels = models.slice(0, PREVIEW_MODEL_LIMIT)

  return (
    <section
      id='pricing'
      className='relative z-10 scroll-mt-20 px-6 py-24 md:py-32'
    >
      <div className='mx-auto max-w-6xl'>
        <AnimateInView className='mb-12 max-w-2xl'>
          <p className='text-muted-foreground mb-3 text-xs font-medium tracking-widest uppercase'>
            {t('Pricing')}
          </p>
          <h2 className='text-2xl leading-tight font-bold tracking-tight md:text-3xl'>
            {t('Real prices, pulled live')}
          </h2>
          <p className='text-muted-foreground mt-4 text-sm leading-relaxed md:text-base'>
            {t(
              'Rates and per-group multipliers are set by the administrator. Every request logs the exact tokens and amount charged.'
            )}
          </p>
        </AnimateInView>

        <AnimateInView className='border-border/40 bg-card overflow-hidden rounded-xl border'>
          {isLoading && (
            <div className='space-y-2 p-6'>
              {[0, 1, 2, 3].map((i) => (
                <Skeleton key={i} className='h-10 w-full' />
              ))}
            </div>
          )}

          {!isLoading && error && (
            <div className='space-y-4 p-6'>
              <Alert variant='destructive'>
                <AlertTitle>{t('Unable to load pricing right now.')}</AlertTitle>
              </Alert>
              <Button
                variant='outline'
                size='sm'
                onClick={() => refetch()}
              >
                <RefreshCw aria-hidden='true' className='size-3.5' />
                {t('Retry')}
              </Button>
            </div>
          )}

          {!isLoading && !error && previewModels.length === 0 && (
            <Empty className='border-none py-12'>
              <EmptyTitle>{t('No models are configured yet.')}</EmptyTitle>
            </Empty>
          )}

          {!isLoading && !error && previewModels.length > 0 && (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('Model')}</TableHead>
                    <TableHead className='text-right'>{t('Input')}</TableHead>
                    <TableHead className='text-right'>
                      {t('Output')}
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {previewModels.map((model) => (
                    <TableRow key={model.id}>
                      <TableCell className='font-medium'>
                        {model.model_name}
                      </TableCell>
                      <TableCell className='text-muted-foreground text-right font-mono text-sm'>
                        {formatPrice(
                          model,
                          'input',
                          DEFAULT_TOKEN_UNIT,
                          false,
                          priceRate,
                          usdExchangeRate
                        )}
                      </TableCell>
                      <TableCell className='text-muted-foreground text-right font-mono text-sm'>
                        {formatPrice(
                          model,
                          'output',
                          DEFAULT_TOKEN_UNIT,
                          false,
                          priceRate,
                          usdExchangeRate
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              <p className='text-muted-foreground border-border/40 border-t px-4 py-3 text-xs'>
                {t('Prices shown per 1M tokens')}
              </p>
            </>
          )}
        </AnimateInView>

        <div className='mt-6 flex justify-center'>
          <Button
            variant='outline'
            className='rounded-full'
            render={<Link to='/pricing' />}
          >
            {t('View Pricing')}
          </Button>
        </div>
      </div>
    </section>
  )
}
