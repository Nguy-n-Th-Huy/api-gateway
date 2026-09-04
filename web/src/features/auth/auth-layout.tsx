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
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import { useSystemConfig } from '@/hooks/use-system-config'

type AuthLayoutProps = {
  children: React.ReactNode
}

export function AuthLayout({ children }: AuthLayoutProps) {
  const { t } = useTranslation()
  const { systemName, logo, loading } = useSystemConfig()

  return (
    <div className='bg-background min-h-svh'>
      {/* Desktop: split brand panel / form card; mobile: stacked, card-only. */}
      <div className='flex min-h-svh flex-col lg:grid lg:grid-cols-2'>
        {/* Brand panel — decorative gradient lockup, no invented copy. */}
        <div className='from-primary via-chart-3 to-chart-2 relative hidden flex-col overflow-hidden bg-linear-to-br lg:flex'>
          <div
            aria-hidden='true'
            className='absolute -top-36 -left-28 size-[420px] rounded-full bg-white/10'
          />
          <div
            aria-hidden='true'
            className='absolute -right-28 -bottom-44 size-[460px] rounded-full bg-white/10'
          />
          <div className='relative flex flex-1 items-center justify-center p-12'>
            <div className='flex flex-col items-center gap-6 text-center'>
              <div className='flex size-16 items-center justify-center rounded-2xl bg-white/20'>
                {loading ? (
                  <Skeleton className='size-16 rounded-2xl bg-white/40' />
                ) : (
                  <img
                    src={logo}
                    alt={t('Logo')}
                    className='size-14 rounded-xl object-cover'
                  />
                )}
              </div>
              {loading ? (
                <Skeleton className='h-8 w-40 bg-white/40' />
              ) : (
                <span className='font-display text-2xl font-bold text-white'>
                  {systemName}
                </span>
              )}
            </div>
          </div>
        </div>

        {/* Form side — the card the mock centers in its right column. */}
        <div className='flex flex-1 flex-col items-center justify-center px-4 py-10 sm:px-8 lg:px-12'>
          {/* Small-viewport brand header */}
          <Link to='/' className='mb-8 flex items-center gap-2 lg:hidden'>
            <div className='relative size-8'>
              {loading ? (
                <Skeleton className='absolute inset-0 rounded-full' />
              ) : (
                <img
                  src={logo}
                  alt={t('Logo')}
                  className='size-8 rounded-full object-cover'
                />
              )}
            </div>
            {loading ? (
              <Skeleton className='h-6 w-24' />
            ) : (
              <span className='font-display text-lg font-bold'>
                {systemName}
              </span>
            )}
          </Link>

          <div className='border-border bg-card w-full max-w-md rounded-2xl border p-6 shadow-[0_28px_60px_-30px_rgba(58,42,140,0.25)] sm:p-10'>
            {children}
          </div>
        </div>
      </div>
    </div>
  )
}
