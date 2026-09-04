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
import {
  Gift,
  ExternalLink,
  Loader2,
  Receipt,
  WalletCards,
  Building2,
} from 'lucide-react'
import { useMemo, useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { IconBadge } from '@/components/ui/icon-badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { TitledCard } from '@/components/ui/titled-card'
import { cn } from '@/lib/utils'

import {
  calculatePayableVNDPreview,
  formatVND,
  getDiscountLabel,
  getMaxTopupAmount,
  getMinTopupAmount,
} from '../lib'
import type { PresetAmount, TopupInfo } from '../types'

interface RechargeFormCardProps {
  topupInfo: TopupInfo | null
  presetAmounts: PresetAmount[]
  selectedPreset: number | null
  onSelectPreset: (preset: PresetAmount) => void
  topupAmount: number
  onTopupAmountChange: (amount: number) => void
  sepayAvailable: boolean
  onCreateOrder: () => void
  creating: boolean
  redemptionCode: string
  onRedemptionCodeChange: (code: string) => void
  onRedeem: () => void
  redeeming: boolean
  topupLink?: string
  loading?: boolean
  onOpenBilling?: () => void
}

export function RechargeFormCard(props: RechargeFormCardProps) {
  const { t } = useTranslation()
  const [localAmount, setLocalAmount] = useState(
    props.topupAmount.toString()
  )

  useEffect(() => {
    // Empty string must survive, otherwise the field can never be cleared
    setLocalAmount((prev) =>
      prev === '' && props.topupAmount === 0 ? prev : props.topupAmount.toString()
    )
  }, [props.topupAmount])

  const handleAmountChange = (value: string) => {
    setLocalAmount(value)
    const numValue = Number.parseInt(value, 10)
    if (Number.isNaN(numValue)) {
      props.onTopupAmountChange(0)
      return
    }
    if (numValue >= 0) {
      props.onTopupAmountChange(numValue)
    }
  }

  const price = props.topupInfo?.price ?? 0
  const discount =
    props.topupInfo?.discount?.[props.topupAmount] ?? 1.0
  const minTopup = getMinTopupAmount(props.topupInfo)
  const maxTopup = getMaxTopupAmount()
  const redemptionEnabled = props.topupInfo?.enable_redemption !== false

  const payablePreview = useMemo(
    () =>
      calculatePayableVNDPreview({
        amount: props.topupAmount,
        price,
        discount,
      }),
    [props.topupAmount, price, discount]
  )

  const amountBelowMin = props.topupAmount > 0 && props.topupAmount < minTopup
  const amountAboveMax = props.topupAmount > maxTopup
  const canPay =
    props.sepayAvailable &&
    Number.isInteger(props.topupAmount) &&
    props.topupAmount >= minTopup &&
    props.topupAmount <= maxTopup &&
    !props.creating

  if (props.loading) {
    return (
      <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
        <CardHeader className='border-b p-3 !pb-3 sm:p-5 sm:!pb-5'>
          <Skeleton className='h-6 w-32' />
          <Skeleton className='mt-2 h-4 w-48' />
        </CardHeader>
        <CardContent className='space-y-4 p-3 sm:space-y-6 sm:p-5'>
          <div className='space-y-4 sm:space-y-6'>
            {/* Preset Amounts Skeleton */}
            <div className='space-y-3'>
              <Skeleton className='h-3 w-16' />
              <div className='grid grid-cols-2 gap-3 sm:grid-cols-4'>
                {Array.from({ length: 8 }, (_, index) => `preset-${index}`).map(
                  (key) => (
                    <Skeleton key={key} className='h-[72px] rounded-lg' />
                  )
                )}
              </div>
            </div>

            {/* Custom Amount Input Skeleton */}
            <div className='space-y-3'>
              <Skeleton className='h-3 w-28' />
              <Skeleton className='h-[42px] w-full' />
            </div>
          </div>

          {/* Redemption Code Section Skeleton */}
          <div className='space-y-3 border-t pt-8'>
            <Skeleton className='h-3 w-24' />
            <div className='flex gap-2'>
              <Skeleton className='h-10 flex-1' />
              <Skeleton className='h-10 w-20' />
            </div>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <TitledCard
      title={t('Add Funds')}
      description={t('Choose an amount and pay by bank transfer')}
      icon={<WalletCards className='h-4 w-4' />}
      iconTone='success'
      disableHoverEffect
      action={
        props.onOpenBilling ? (
          <Button
            variant='outline'
            size='sm'
            onClick={props.onOpenBilling}
            className='w-full gap-2 sm:w-auto'
          >
            <Receipt className='h-4 w-4' />
            {t('Order History')}
          </Button>
        ) : null
      }
      contentClassName='space-y-4 sm:space-y-6'
    >
      {props.sepayAvailable ? (
        <div className='space-y-4 sm:space-y-6'>
          {props.presetAmounts.length > 0 && (
            <div className='space-y-2.5 sm:space-y-3'>
              <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                {t('Amount')}
              </Label>
              <div className='grid grid-cols-2 gap-1.5 sm:gap-3 md:grid-cols-4'>
                {props.presetAmounts.map((preset) => {
                  const presetDiscount =
                    preset.discount || props.topupInfo?.discount?.[preset.value] || 1.0
                  const presetPayable = calculatePayableVNDPreview({
                    amount: preset.value,
                    price,
                    discount: presetDiscount,
                  })
                  const hasDiscount = presetDiscount < 1.0
                  return (
                    <Button
                      key={preset.value}
                      type='button'
                      variant='outline'
                      className={cn(
                        'flex min-h-16 flex-col items-start rounded-lg px-3 py-2.5 text-left whitespace-normal sm:min-h-[72px] sm:p-4',
                        props.selectedPreset === preset.value
                          ? 'border-foreground bg-foreground/5 dark:border-foreground dark:bg-foreground/10'
                          : 'border-muted'
                      )}
                      onClick={() => props.onSelectPreset(preset)}
                    >
                      <div className='flex w-full items-center justify-between'>
                        <div className='text-base font-semibold sm:text-lg'>
                          ${preset.value}
                        </div>
                        {hasDiscount && (
                          <div className='text-success text-xs font-medium'>
                            {getDiscountLabel(presetDiscount)}
                          </div>
                        )}
                      </div>
                      <div className='text-muted-foreground mt-1.5 w-full text-xs sm:mt-2'>
                        {t('Pay')} {presetPayable === null ? '-' : formatVND(presetPayable)} ₫
                      </div>
                    </Button>
                  )
                })}
              </div>
            </div>
          )}

          <div className='space-y-2.5 sm:space-y-3'>
            <Label
              htmlFor='topup-amount'
              className='text-muted-foreground text-xs font-medium tracking-wider uppercase'
            >
              {t('Custom Amount')}
            </Label>
            <div className='grid grid-cols-[minmax(0,1fr)_minmax(120px,0.55fr)] gap-2 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center'>
              <Input
                id='topup-amount'
                type='number'
                inputMode='numeric'
                value={localAmount}
                onChange={(e) => handleAmountChange(e.target.value)}
                min={minTopup}
                max={maxTopup}
                step={1}
                placeholder={`${t('Minimum')}: ${minTopup}`}
                className='h-9 text-base sm:h-10 sm:text-lg'
              />
              <div className='bg-muted/30 flex min-h-9 items-center justify-between gap-2 rounded-md border px-3 lg:min-w-52'>
                <span className='text-muted-foreground truncate text-xs'>
                  {t('Amount to transfer:')}
                </span>
                <span className='text-sm font-semibold tabular-nums'>
                  {payablePreview === null ? '-' : `${formatVND(payablePreview)} ₫`}
                </span>
              </div>
            </div>
            {amountBelowMin && (
              <p className='text-destructive text-xs'>
                {t('Minimum topup amount: {{amount}}', { amount: minTopup })}
              </p>
            )}
            {amountAboveMax && (
              <p className='text-destructive text-xs'>
                {t('Maximum topup amount: {{amount}}', { amount: maxTopup })}
              </p>
            )}
          </div>

          <Button
            type='button'
            className='w-full gap-2'
            onClick={props.onCreateOrder}
            disabled={!canPay}
          >
            {props.creating ? (
              <Loader2 className='h-4 w-4 animate-spin' />
            ) : (
              <Building2 className='h-4 w-4' />
            )}
            {t('Pay with SePay')}
          </Button>
        </div>
      ) : (
        <Alert>
          <AlertDescription>
            {t(
              'Online top-up is not enabled. Please use a redemption code or contact the administrator.'
            )}
          </AlertDescription>
        </Alert>
      )}

      {/* Redemption Code Section */}
      {redemptionEnabled ? (
        <div className='space-y-2.5 border-t pt-4 sm:space-y-3 sm:pt-6'>
          <div className='flex items-center gap-2'>
            <IconBadge tone='warning' size='xs'>
              <Gift />
            </IconBadge>
            <Label
              htmlFor='redemption-code'
              className='text-muted-foreground text-xs font-medium tracking-wider uppercase'
            >
              {t('Have a Code?')}
            </Label>
          </div>
          <div className='grid grid-cols-[minmax(0,1fr)_auto] gap-2'>
            <Input
              id='redemption-code'
              value={props.redemptionCode}
              onChange={(e) => props.onRedemptionCodeChange(e.target.value)}
              placeholder={t('Enter your redemption code')}
              className='h-9 min-w-0'
            />
            <Button
              onClick={props.onRedeem}
              disabled={props.redeeming}
              variant='outline'
              className='h-9 px-4'
            >
              {props.redeeming && (
                <Loader2 className='mr-2 h-4 w-4 animate-spin' />
              )}
              {t('Redeem')}
            </Button>
          </div>
          {props.topupLink && (
            <p className='text-muted-foreground text-xs'>
              {t('Need a redemption code?')}{' '}
              <a
                href={props.topupLink}
                target='_blank'
                rel='noopener noreferrer'
                className='inline-flex items-center gap-1 underline-offset-4 hover:underline'
              >
                {t('Get one here')}
                <ExternalLink className='h-3 w-3' />
              </a>
            </p>
          )}
        </div>
      ) : (
        <Alert className='border-t'>
          <AlertDescription>
            {t(
              'Redemption codes are disabled until the administrator confirms compliance terms.'
            )}
          </AlertDescription>
        </Alert>
      )}
    </TitledCard>
  )
}
