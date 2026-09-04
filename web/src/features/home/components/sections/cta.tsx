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
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { Button } from '@/components/ui/button'

interface CTAProps {
  className?: string
  isAuthenticated?: boolean
}

export function CTA(props: CTAProps) {
  const { t } = useTranslation()

  if (props.isAuthenticated) {
    return null
  }

  return (
    <section className='relative z-10 overflow-hidden px-6 py-24 md:py-32'>
      {/* Brand-tinted wash expressed from theme tokens. */}
      <div
        aria-hidden
        className='absolute inset-0 -z-10 bg-[radial-gradient(50%_50%_at_30%_50%,var(--accent),transparent_70%),radial-gradient(40%_40%_at_70%_40%,color-mix(in_oklch,var(--overview-accent-2)_14%,transparent),transparent_70%)] opacity-70 dark:opacity-20'
      />

      <AnimateInView
        className='mx-auto max-w-2xl text-center'
        animation='scale-in'
      >
        <h2 className='font-display text-2xl leading-tight font-bold tracking-tight md:text-4xl'>
          {t('Ready to top up and')}
          <br />
          <span className='from-primary via-chart-3 to-overview-accent-2 bg-linear-to-r bg-clip-text text-transparent'>
            {t('make your first request?')}
          </span>
        </h2>
        <p className='text-muted-foreground mx-auto mt-5 max-w-md text-sm leading-relaxed md:text-base'>
          {t(
            'Top up in seconds by bank transfer, then start calling the gateway with your existing code.'
          )}
        </p>
        <div className='mt-8 flex items-center justify-center gap-3'>
          <Button
            className='group rounded-full'
            render={<Link to='/sign-up' />}
          >
            {t('Get Started')}
            <ArrowRight
              aria-hidden='true'
              className='ml-1 size-3.5 transition-transform duration-200 group-hover:translate-x-0.5'
            />
          </Button>
          <Button
            variant='outline'
            className='border-border hover:border-primary/40 rounded-full'
            render={<Link to='/pricing' />}
          >
            {t('View Pricing')}
          </Button>
        </div>
      </AnimateInView>
    </section>
  )
}
