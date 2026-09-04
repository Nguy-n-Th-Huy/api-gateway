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

import { Dialog } from '@/components/dialog'

import type { SePayOrder } from '../types'

import { SePayPaymentPanel } from './sepay-payment-panel'

interface SePayPaymentDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  order: SePayOrder | null
  onSuccess?: () => void
  onDone?: () => void
  onRetry?: () => void
}

/**
 * Wraps the SePay payment panel in a dialog so both the wallet top-up flow and
 * the subscription purchase flow can present the same bank-transfer payment UI.
 * The panel polls the order status while the dialog is mounted and stops on
 * success/expiry.
 */
export function SePayPaymentDialog({
  open,
  onOpenChange,
  order,
  onSuccess,
  onDone,
  onRetry,
}: SePayPaymentDialogProps) {
  const { t } = useTranslation()

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Complete Your Payment')}
      description={t(
        'Transfer the exact amount and use the memo shown below to top up.'
      )}
      contentClassName='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-2xl'
      contentHeight='auto'
      bodyClassName='space-y-4'
    >
      {order ? (
        <SePayPaymentPanel
          order={order}
          onSuccess={onSuccess}
          onDone={onDone}
          onRetry={onRetry}
        />
      ) : null}
    </Dialog>
  )
}
