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

import { PROVIDER_NAMES, PROVIDER_STRIP_MORE_COUNT } from '../../constants'

interface ProvidersProps {
  className?: string
}

/**
 * Strip naming the upstream model families the gateway routes to. Provider
 * names are proper nouns and stay untranslated, matching the app-pill
 * precedent already established in the hero (Cherry Studio, CC Switch).
 */
export function Providers(_props: ProvidersProps) {
  const { t } = useTranslation()

  return (
    <section className='border-border/40 bg-muted/10 relative z-10 border-y'>
      <div className='mx-auto flex max-w-6xl flex-col gap-4 px-6 py-6 sm:flex-row sm:items-center sm:gap-7'>
        <span className='text-muted-foreground shrink-0 text-xs font-semibold sm:max-w-40'>
          {t('Routes through 40+ providers')}
        </span>
        <div className='flex flex-wrap gap-2'>
          {PROVIDER_NAMES.map((name) => (
            <span
              key={name}
              className='border-border bg-card text-muted-foreground rounded-full border px-3.5 py-1.5 text-[13px] font-medium'
            >
              {name}
            </span>
          ))}
          <span className='text-muted-foreground rounded-full px-3.5 py-1.5 text-[13px] font-medium'>
            {t('+{{count}} more', { count: PROVIDER_STRIP_MORE_COUNT })}
          </span>
        </div>
      </div>
    </section>
  )
}
