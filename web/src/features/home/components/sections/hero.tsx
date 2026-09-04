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
import { ArrowRight, BookOpen, CheckCircle2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { useStatus } from '@/hooks/use-status'

import { getHeroTrustBullets } from '../../constants'
import { HeroTerminalDemo } from '../hero-terminal-demo'
import { HeroSupportedApps } from './hero-supported-apps'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const docsUrl =
    (status?.docs_link as string | undefined) || 'https://docs.newapi.pro'
  const trustBullets = getHeroTrustBullets(t)

  const renderPricingButton = () => (
    <Button
      variant='outline'
      size='lg'
      className='h-11 rounded-full px-6 text-sm'
      render={<a href='#pricing' />}
    >
      {t('View Pricing')}
    </Button>
  )

  const renderDocsButton = () => {
    const isExternal = docsUrl.startsWith('http')
    const docsLink = isExternal ? (
      <a href={docsUrl} target='_blank' rel='noopener noreferrer' />
    ) : (
      <Link to={docsUrl} />
    )
    return (
      <Button
        variant='outline'
        className='group border-border hover:border-primary/40 inline-flex h-11 items-center gap-1.5 rounded-full px-6 text-sm'
        render={docsLink}
      >
        <BookOpen
          aria-hidden='true'
          className='text-muted-foreground group-hover:text-foreground size-4 transition-colors duration-200'
        />
        <span>{t('Docs')}</span>
      </Button>
    )
  }

  return (
    <section className='relative z-10 overflow-hidden px-6 pt-24 pb-16 md:pt-32 md:pb-24 lg:pt-36 lg:pb-28'>
      {/* Brand-tinted radial wash per giaodienmau/main.html (violet top-left,
          warm top-right), expressed from theme tokens instead of literals. */}
      <div
        aria-hidden
        className='pointer-events-none absolute inset-0 -z-10 bg-[radial-gradient(1100px_460px_at_12%_0%,var(--accent),transparent_62%),radial-gradient(900px_420px_at_92%_8%,color-mix(in_oklch,var(--overview-accent-2)_18%,transparent),transparent_60%)] opacity-60 dark:opacity-25'
      />
      {/* Grid pattern */}
      <div
        aria-hidden
        className='absolute inset-0 -z-10 bg-[linear-gradient(to_right,var(--border)_1px,transparent_1px),linear-gradient(to_bottom,var(--border)_1px,transparent_1px)] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_30%,black_20%,transparent_100%)] bg-[size:4rem_4rem] opacity-[0.08]'
      />

      <div className='mx-auto grid max-w-6xl grid-cols-1 items-start gap-12 lg:grid-cols-12 lg:gap-8'>
        {/* Left Column: Title, description, action buttons and application support */}
        <div className='flex flex-col items-start text-left lg:col-span-6'>
          {/* Top Pill Badge */}
          <div
            className='landing-animate-fade-up border-border bg-card text-accent-foreground mb-5 inline-flex items-center gap-1.5 rounded-full border px-3.5 py-1.5 text-[11px] font-semibold opacity-0 shadow-xs'
            style={{ animationDelay: '0ms' }}
          >
            <span className='relative flex size-1.5'>
              <span className='bg-success absolute inline-flex h-full w-full animate-ping rounded-full opacity-75' />
              <span className='bg-success relative inline-flex size-1.5 rounded-full' />
            </span>
            <span>{t('AI infrastructure for Vietnamese product teams')}</span>
          </div>

          <h1
            className='landing-animate-fade-up font-display text-[clamp(2.25rem,4.5vw,3.25rem)] leading-[1.1] font-extrabold tracking-tight'
            style={{ animationDelay: '60ms' }}
          >
            {t('One API key,')}
            <br />
            <span className='from-primary via-chart-3 to-overview-accent-2 bg-linear-to-r bg-clip-text text-transparent'>
              {t('for every AI model')}
            </span>
          </h1>
          <p
            className='landing-animate-fade-up text-muted-foreground mt-5 max-w-xl text-base leading-relaxed opacity-0 md:text-[15px]'
            style={{ animationDelay: '120ms' }}
          >
            {t(
              'GPT, Claude, Gemini, DeepSeek, Qwen and 40+ other providers — all through one OpenAI-compatible standard. Top up via domestic bank transfer through SePay; quota is added automatically.'
            )}
          </p>

          <div
            className='landing-animate-fade-up mt-8 flex flex-wrap items-center gap-3 opacity-0'
            style={{ animationDelay: '180ms' }}
          >
            {props.isAuthenticated ? (
              <>
                <Button
                  size='lg'
                  className='group h-11 rounded-full px-6 text-sm'
                  render={<Link to='/dashboard' />}
                >
                  {t('Go to Dashboard')}
                  <ArrowRight
                    aria-hidden='true'
                    className='ml-1.5 size-4 transition-transform duration-200 group-hover:translate-x-0.5'
                  />
                </Button>
                {renderPricingButton()}
                {renderDocsButton()}
              </>
            ) : (
              <>
                <Button
                  size='lg'
                  className='group h-11 rounded-full px-6 text-sm'
                  render={<Link to='/sign-up' />}
                >
                  {t('Get Started')}
                  <ArrowRight
                    aria-hidden='true'
                    className='ml-1.5 size-4 transition-transform duration-200 group-hover:translate-x-0.5'
                  />
                </Button>
                {renderPricingButton()}
                {renderDocsButton()}
              </>
            )}
          </div>

          {/* Trust bullets */}
          <ul
            className='landing-animate-fade-up mt-6 flex flex-wrap items-center gap-x-6 gap-y-2 opacity-0'
            style={{ animationDelay: '210ms' }}
          >
            {trustBullets.map((bullet) => (
              <li key={bullet} className='flex items-center gap-2'>
                <CheckCircle2
                  aria-hidden='true'
                  className='text-success size-4 shrink-0'
                />
                <span className='text-muted-foreground text-[13.5px]'>
                  {bullet}
                </span>
              </li>
            ))}
          </ul>

          {/* Supported Apps */}
          <HeroSupportedApps />
        </div>

        {/* Right Column: Hero Terminal API Demo */}
        <div
          className='landing-animate-fade-up flex w-full justify-center opacity-0 lg:col-span-6'
          style={{ animationDelay: '320ms' }}
        >
          <HeroTerminalDemo className='mt-8 lg:mt-0' />
        </div>
      </div>
    </section>
  )
}
