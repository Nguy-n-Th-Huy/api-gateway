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
  AlertCircle,
  Building2,
  CheckCircle2,
  Clock,
  Copy,
  Check,
} from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

import { useSePayOrderPolling } from '../hooks/use-sepay-order-polling'
import type { SePayOrder, SePayOrderStatus } from '../types'

function formatVND(amount: number | null | undefined): string {
  if (!Number.isFinite(amount as number)) return '-'
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits: 0,
  }).format(amount as number)
}

function formatCountdown(totalSeconds: number): string {
  const safe = totalSeconds > 0 ? totalSeconds : 0
  const m = Math.floor(safe / 60)
  const s = Math.floor(safe % 60)
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

interface CopyRowProps {
  label: string
  value: string
  copyLabel: string
}

function CopyRow({ label, value, copyLabel }: CopyRowProps) {
  const { t } = useTranslation()
  const { copyToClipboard, copiedText } = useCopyToClipboard({ notify: false })
  const copied = copiedText === value

  return (
    <div className='flex items-center justify-between gap-2 rounded-md border p-3'>
      <div className='min-w-0'>
        <div className='text-muted-foreground text-xs'>{label}</div>
        <code className='block max-w-full truncate font-mono text-sm font-medium'>
          {value}
        </code>
      </div>
      <Button
        type='button'
        variant='outline'
        size='sm'
        className='shrink-0'
        aria-label={copyLabel}
        onClick={() => {
          void copyToClipboard(value)
        }}
      >
        {copied ? <Check className='h-4 w-4' /> : <Copy className='h-4 w-4' />}
        <span className='ml-1'>{copied ? t('Copied') : t('Copy')}</span>
      </Button>
    </div>
  )
}

interface SePayPaymentPanelProps {
  order: SePayOrder
  /** Called once when polling reports the order became successful. */
  onSuccess?: () => void
  /** Called when the user dismisses a settled (successful) order. */
  onDone?: () => void
  /** Called when the user wants to start a fresh order after expiry. */
  onRetry?: () => void
  /** Optional override for tests to advance the countdown deterministically. */
  nowProvider?: () => number
}

/**
 * SePay payment panel: renders the per-order VietQR image (amount + memo
 * encoded), the payable Dong amount, the destination bank account, bank code,
 * and account holder, a copyable memo, a copyable account number, the
 * memo-required instruction, and a countdown to expiry.
 *
 * It polls the order status while open and stops on success, expiry, or
 * unmount. When the countdown reaches zero the panel treats the order as
 * expired immediately, so the UI does not wait for the background sweep.
 */
export function SePayPaymentPanel({
  order,
  onSuccess,
  onDone,
  onRetry,
  nowProvider,
}: SePayPaymentPanelProps) {
  const { t } = useTranslation()

  const { status: polledStatus } = useSePayOrderPolling({
    tradeNo: order.trade_no,
    onSuccess: () => {
      onSuccess?.()
    },
  })

  const [now, setNow] = useState(() =>
    nowProvider ? nowProvider() : Math.floor(Date.now() / 1000)
  )

  useEffect(() => {
    if (nowProvider) {
      setNow(nowProvider())
      return
    }
    const id = window.setInterval(
      () => setNow(Math.floor(Date.now() / 1000)),
      1000
    )
    return () => window.clearInterval(id)
  }, [nowProvider])

  const remainingSec = useMemo(() => {
    if (!Number.isFinite(order.expire_time)) return 0
    return order.expire_time - now
  }, [order.expire_time, now])

  const status = useMemo<SePayOrderStatus>(() => {
    if (polledStatus === 'success') return 'success'
    if (polledStatus === 'expired') return 'expired'
    if (remainingSec <= 0) return 'expired'
    return 'pending'
  }, [polledStatus, remainingSec])

  const isSuccess = status === 'success'
  const isExpired = status === 'expired'

  let badge: { label: string; variant: 'success' | 'destructive' | 'default' }
  if (isSuccess) {
    badge = { label: t('Success'), variant: 'success' }
  } else if (isExpired) {
    badge = { label: t('Expired'), variant: 'destructive' }
  } else {
    badge = { label: t('Pending'), variant: 'default' }
  }

  return (
    <div className='space-y-4'>
      <div className='flex items-center justify-between gap-3'>
        <div className='flex items-center gap-2'>
          <Building2 className='text-muted-foreground h-4 w-4' />
          <span className='text-sm font-medium'>{t('SePay Bank Transfer')}</span>
          <Badge variant={badge.variant}>{badge.label}</Badge>
        </div>
        {!isSuccess && (
          <span className='flex items-center gap-1.5 font-mono text-sm'>
            <Clock className='h-4 w-4' aria-hidden='true' />
            <span aria-live='polite'>
              {isExpired ? '00:00' : formatCountdown(remainingSec)}
            </span>
          </span>
        )}
      </div>

      {!isSuccess && !isExpired && (
        <Alert>
          <AlertCircle className='h-4 w-4' />
          <AlertDescription>
            {t(
              'You must use the transfer memo below as the transfer description. The order cannot be credited without it.'
            )}
          </AlertDescription>
        </Alert>
      )}

      {isSuccess && (
        <Alert>
          <CheckCircle2 className='h-4 w-4' />
          <AlertDescription>
            {t(
              'Payment confirmed. Your balance has been updated automatically.'
            )}
          </AlertDescription>
        </Alert>
      )}

      {isExpired && (
        <Alert variant='destructive'>
          <AlertCircle className='h-4 w-4' />
          <AlertDescription>
            {t('This order has expired. Create a new order to continue.')}
          </AlertDescription>
        </Alert>
      )}

      <div className='grid gap-4 md:grid-cols-[minmax(0,240px)_1fr]'>
        <div className='space-y-2'>
          <img
            src={order.vietqr_url}
            alt={t('VietQR payment code')}
            className='mx-auto h-auto w-full max-w-[280px] rounded-lg border bg-white'
            loading='lazy'
            decoding='async'
            referrerPolicy='no-referrer'
          />
          <p className='text-muted-foreground text-center text-xs'>
            {t(
              'Scan this code in your banking app. The amount and transfer memo are pre-filled.'
            )}
          </p>
        </div>

        <div className='space-y-3'>
          <div className='rounded-lg border p-3'>
            <div className='text-muted-foreground text-xs'>
              {t('Amount to transfer')}
            </div>
            <div className='text-xl font-semibold tabular-nums'>
              {formatVND(order.payable_vnd)}{' '}
              <span className='text-muted-foreground text-sm font-normal'>
                VND
              </span>
            </div>
          </div>

          <CopyRow
            label={t('Transfer memo (required)')}
            value={order.memo}
            copyLabel={t('Copy transfer memo')}
          />

          <CopyRow
            label={t('Account number')}
            value={order.bank_account}
            copyLabel={t('Copy account number')}
          />

          <div className='grid grid-cols-1 gap-2 text-sm sm:grid-cols-2'>
            <div className='rounded-md border px-3 py-2'>
              <div className='text-muted-foreground text-xs'>{t('Bank code')}</div>
              <div className='truncate font-medium'>{order.bank_code}</div>
            </div>
            <div className='rounded-md border px-3 py-2'>
              <div className='text-muted-foreground text-xs'>
                {t('Account holder')}
              </div>
              <div className='truncate font-medium'>{order.account_holder}</div>
            </div>
          </div>

          {isSuccess && onDone && (
            <Button type='button' className='w-full' onClick={onDone}>
              {t('Done')}
            </Button>
          )}
          {isExpired && onRetry && (
            <Button
              type='button'
              variant='outline'
              className='w-full'
              onClick={onRetry}
            >
              {t('Create new order')}
            </Button>
          )}
        </div>
      </div>
    </div>
  )
}
