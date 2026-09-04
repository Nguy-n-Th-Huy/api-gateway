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
import { CheckCircle2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { Button } from '@/components/ui/button'

import { getSepayTopupSteps } from '../../constants'

interface SepayTopupProps {
  className?: string
  isAuthenticated?: boolean
}

/**
 * Describes the SePay bank-transfer top-up flow as an ordered, numberless
 * sequence of steps. No amount, bank account number or transfer code is
 * shown here — those are only available to an authenticated user.
 */
export function SepayTopup(props: SepayTopupProps) {
  const { t } = useTranslation()
  const steps = getSepayTopupSteps(t)

  return (
    <section className='relative z-10 px-6 py-24 md:py-32'>
      <div className='mx-auto max-w-6xl'>
        <AnimateInView className='mb-12 max-w-2xl'>
          <p className='text-muted-foreground mb-3 text-xs font-medium tracking-widest uppercase'>
            {t('Domestic payments')}
          </p>
          <h2 className='text-2xl leading-tight font-bold tracking-tight md:text-3xl'>
            {t('Top up by bank transfer.')}
            <br />
            {t('Pay for exactly the tokens you use.')}
          </h2>
          <p className='text-muted-foreground mt-4 max-w-xl text-sm leading-relaxed md:text-base'>
            {t(
              'No Visa or Mastercard, no foreign currency wallet needed. Scan VietQR from any banking app — SePay reconciles the transfer and adds quota automatically.'
            )}
          </p>
        </AnimateInView>

        <ol className='border-border/40 bg-border/40 grid gap-px overflow-hidden rounded-xl border md:grid-cols-3'>
          {steps.map((step, i) => (
            <AnimateInView
              key={step.title}
              delay={i * 120}
              animation='fade-up'
              as='li'
              className='bg-background p-7 md:p-8'
            >
              <span className='border-border/40 bg-muted text-muted-foreground mb-4 flex size-8 items-center justify-center rounded-full border text-xs font-bold'>
                {i + 1}
              </span>
              <h3 className='mb-2 text-sm font-semibold'>{step.title}</h3>
              <p className='text-muted-foreground text-sm leading-relaxed'>
                {step.description}
              </p>
            </AnimateInView>
          ))}
        </ol>

        <div className='mt-8 flex flex-col items-start gap-4 sm:flex-row sm:items-center sm:justify-between'>
          <div className='border-success/25 bg-success/10 text-success inline-flex items-center gap-2 rounded-full border px-4 py-2 text-xs font-semibold'>
            <CheckCircle2 aria-hidden='true' className='size-4 shrink-0' />
            {t('Reconciled automatically — no receipt to send')}
          </div>
          {props.isAuthenticated ? (
            <Button size='lg' className='rounded-full' render={<Link to='/wallet' />}>
              {t('Go to Wallet')}
            </Button>
          ) : (
            <Button size='lg' className='rounded-full' render={<Link to='/sign-up' />}>
              {t('Get Started')}
            </Button>
          )}
        </div>
      </div>
    </section>
  )
}
