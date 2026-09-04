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
  Gauge,
  HeartHandshake,
  KeyRound,
  RefreshCw,
  ScrollText,
  Users,
  Waypoints,
  Zap,
  type LucideIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'

import { getAdditionalGatewayFeatures, getCoreFeatures } from '../../constants'

interface FeaturesProps {
  className?: string
}

const CORE_FEATURE_STYLE: Record<
  string,
  { icon: LucideIcon; iconClass: string; badgeClass: string; span: string }
> = {
  failover: {
    icon: RefreshCw,
    iconClass: 'text-primary',
    badgeClass: 'bg-primary/10',
    span: 'md:col-span-2',
  },
  tokens: {
    icon: KeyRound,
    iconClass: 'text-success',
    badgeClass: 'bg-success/10',
    span: 'md:col-span-1',
  },
  logs: {
    icon: ScrollText,
    iconClass: 'text-chart-3',
    badgeClass: 'bg-chart-3/10',
    span: 'md:col-span-1',
  },
  protocols: {
    icon: Waypoints,
    iconClass: 'text-warning',
    badgeClass: 'bg-warning/10',
    span: 'md:col-span-2',
  },
}

const ADDITIONAL_FEATURE_ICONS: Record<string, LucideIcon> = {
  Zap,
  Gauge,
  Users,
  HeartHandshake,
}

export function Features(_props: FeaturesProps) {
  const { t } = useTranslation()
  const coreFeatures = getCoreFeatures(t)
  const additionalFeatures = getAdditionalGatewayFeatures(t)

  return (
    <section className='relative z-10 px-6 py-24 md:py-32'>
      <div className='mx-auto max-w-6xl'>
        <AnimateInView className='mb-16 max-w-lg'>
          <p className='text-muted-foreground mb-3 text-xs font-medium tracking-widest uppercase'>
            {t('Core Features')}
          </p>
          <h2 className='text-2xl leading-tight font-bold tracking-tight md:text-3xl'>
            {t('Enough for a team running AI in a real product')}
          </h2>
        </AnimateInView>

        {/* Bento grid */}
        <div className='border-border/40 bg-border/40 grid gap-px overflow-hidden rounded-xl border md:grid-cols-3'>
          {coreFeatures.map((f, i) => {
            const style = CORE_FEATURE_STYLE[f.id]
            const Icon = style.icon
            return (
              <AnimateInView
                key={f.id}
                delay={i * 100}
                animation='scale-in'
                className={`bg-background group hover:bg-muted/20 p-7 transition-colors duration-300 md:p-8 ${style.span}`}
              >
                <div className='mb-3 flex items-center gap-3'>
                  <span
                    className={`flex size-9 items-center justify-center rounded-lg ${style.badgeClass}`}
                  >
                    <Icon
                      aria-hidden='true'
                      className={`size-4 ${style.iconClass}`}
                    />
                  </span>
                  <h3 className='text-sm font-semibold'>{f.title}</h3>
                </div>
                <p className='text-muted-foreground text-sm leading-relaxed'>
                  {f.description}
                </p>
              </AnimateInView>
            )
          })}
        </div>

        {/* Additional features row */}
        <div className='mt-12 grid grid-cols-2 gap-8 md:grid-cols-4 md:gap-12'>
          {additionalFeatures.map((f, i) => {
            const Icon = ADDITIONAL_FEATURE_ICONS[f.iconName]
            return (
              <AnimateInView
                key={f.id}
                delay={i * 100}
                animation='fade-up'
                className='flex flex-col items-center text-center'
              >
                <div className='text-muted-foreground border-border/50 bg-muted/30 group-hover:text-foreground mb-3 flex size-12 items-center justify-center rounded-xl border transition-colors'>
                  <Icon aria-hidden='true' className='size-5' strokeWidth={1.5} />
                </div>
                <h3 className='mb-1.5 text-sm font-semibold'>{f.title}</h3>
                <p className='text-muted-foreground max-w-[200px] text-xs leading-relaxed'>
                  {f.description}
                </p>
              </AnimateInView>
            )
          })}
        </div>
      </div>
    </section>
  )
}
