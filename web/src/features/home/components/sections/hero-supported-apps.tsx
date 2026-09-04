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
import { CherryStudio } from '@lobehub/icons'
import { useTranslation } from 'react-i18next'

// Stylized three-dots indicator representing "More"
function MoreIcon() {
  return (
    <svg
      className='text-muted-foreground/60 group-hover:text-foreground size-6 shrink-0 transition-colors'
      viewBox='0 0 24 24'
      fill='none'
      xmlns='http://www.w3.org/2000/svg'
      aria-hidden='true'
    >
      <circle cx='6' cy='12' r='2' fill='currentColor' />
      <circle cx='12' cy='12' r='2' fill='currentColor' />
      <circle cx='18' cy='12' r='2' fill='currentColor' />
    </svg>
  )
}

/**
 * "Supported Applications" pill row shown under the hero actions — split out
 * of hero.tsx to keep that file under the ~200-line ceiling (web/AGENTS.md 3.3).
 */
export function HeroSupportedApps() {
  const { t } = useTranslation()

  return (
    <div
      className='landing-animate-fade-up mt-10 w-full max-w-xl opacity-0'
      style={{ animationDelay: '240ms' }}
    >
      <div className='mb-4 flex flex-col gap-1'>
        <span className='text-muted-foreground text-[10px] font-bold tracking-[0.15em] uppercase'>
          {t('Supported Applications')}
        </span>
        <p className='text-muted-foreground text-xs leading-relaxed'>
          {t(
            'Supports one-click configuration and perfectly adapts to NewAPI multi-protocol configuration.'
          )}
        </p>
      </div>
      <div className='flex flex-wrap items-center gap-3'>
        {/* Cherry Studio */}
        <a
          href='https://cherry-ai.com'
          target='_blank'
          rel='noopener noreferrer'
          className='group border-border bg-card text-foreground hover:border-primary/40 pointer-coarse:touch-target-floor flex items-center gap-3 rounded-full border px-5 py-2.5 text-sm font-medium shadow-xs backdrop-blur-xs transition-all duration-300 hover:scale-[1.02]'
        >
          <CherryStudio.Color size={24} className='shrink-0' />
          <span>Cherry Studio</span>
        </a>

        {/* CC Switch */}
        <a
          href='https://ccswitch.io'
          target='_blank'
          rel='noopener noreferrer'
          className='group border-border bg-card text-foreground hover:border-primary/40 pointer-coarse:touch-target-floor flex items-center gap-3 rounded-full border px-5 py-2.5 text-sm font-medium shadow-xs backdrop-blur-xs transition-all duration-300 hover:scale-[1.02]'
        >
          <img
            src='https://ccswitch.io/favicon.png'
            alt='CC Switch'
            className='size-6 shrink-0 rounded-md object-contain'
            onError={(e) => {
              // Fallback to a styled text avatar if the remote favicon fails to load in sandbox or local environments
              e.currentTarget.style.display = 'none'
              const fallback = e.currentTarget.nextSibling as HTMLElement
              if (fallback) fallback.style.display = 'flex'
            }}
          />
          <span
            style={{ display: 'none' }}
            className='bg-primary/10 text-primary size-6 shrink-0 items-center justify-center rounded-md text-[10px] font-bold'
          >
            CC
          </span>
          <span>CC Switch</span>
        </a>

        {/* "More Apps" */}
        <div className='group border-border bg-card text-muted-foreground hover:border-primary/40 hover:text-foreground flex cursor-default items-center gap-2.5 rounded-full border px-5 py-2.5 text-sm font-medium shadow-xs backdrop-blur-xs transition-all duration-300 hover:scale-[1.02]'>
          <MoreIcon />
          <span>{t('More Apps')}</span>
        </div>
      </div>
    </div>
  )
}
