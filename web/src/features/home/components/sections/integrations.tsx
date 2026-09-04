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

import { AnimateInView } from '@/components/animate-in-view'

import { getIntegrationApps } from '../../constants'

interface IntegrationsProps {
  className?: string
}

/** OpenAI-SDK style Python snippet — base_url uses the real deployment
 * origin (never a bracketed placeholder), the API key is a masked example. */
function buildCodeSample(baseUrl: string) {
  return [
    { key: 'import', node: (
      <>
        <span className='text-chart-3'>from</span>{' '}
        <span className='text-muted-foreground'>openai</span>{' '}
        <span className='text-chart-3'>import</span>{' '}
        <span className='text-primary font-medium'>OpenAI</span>
      </>
    ) },
    { key: 'blank-1', node: <>&nbsp;</> },
    { key: 'client-open', node: (
      <>
        <span className='text-muted-foreground'>client = </span>
        <span className='text-primary font-medium'>OpenAI</span>
        <span className='text-muted-foreground'>(</span>
      </>
    ) },
    { key: 'base-url', node: (
      <>
        {'  '}
        <span className='text-muted-foreground'>base_url=</span>
        <span className='text-success'>&quot;{baseUrl}&quot;</span>
        <span className='text-muted-foreground'>,</span>
      </>
    ) },
    { key: 'api-key', node: (
      <>
        {'  '}
        <span className='text-muted-foreground'>api_key=</span>
        <span className='text-success'>&quot;sk-••••••••&quot;</span>
        <span className='text-muted-foreground'>,</span>
      </>
    ) },
    { key: 'client-close', node: <span className='text-muted-foreground'>)</span> },
  ]
}

export function Integrations(_props: IntegrationsProps) {
  const { t } = useTranslation()
  const apps = getIntegrationApps(t)
  const baseUrl =
    typeof window !== 'undefined' ? `${window.location.origin}/v1` : '/v1'
  const codeLines = buildCodeSample(baseUrl)

  return (
    <section className='relative z-10 px-6 py-24 md:py-32'>
      <div className='mx-auto max-w-6xl'>
        <AnimateInView className='mb-12 max-w-2xl'>
          <p className='text-muted-foreground mb-3 text-xs font-medium tracking-widest uppercase'>
            {t('Start in 3 minutes')}
          </p>
          <h2 className='text-2xl leading-tight font-bold tracking-tight md:text-3xl'>
            {t('Change two lines of config and your existing app keeps running')}
          </h2>
          <p className='text-muted-foreground mt-4 max-w-xl text-sm leading-relaxed md:text-base'>
            {t(
              'No proprietary SDK, no new API to learn. Swap the base URL and the API key and you are done.'
            )}
          </p>
        </AnimateInView>

        <div className='grid gap-4 lg:grid-cols-[1fr_1.25fr] lg:items-start'>
          <ul className='flex flex-col gap-3'>
            {apps.map((app) => (
              <li
                key={app.name}
                className='border-border bg-card flex items-center gap-4 rounded-2xl border px-5 py-4'
              >
                <span className='bg-primary/10 text-primary flex size-10 shrink-0 items-center justify-center rounded-xl text-xs font-bold'>
                  {app.name.slice(0, 2).toUpperCase()}
                </span>
                <div className='flex flex-col gap-0.5'>
                  <span className='text-sm font-semibold'>{app.name}</span>
                  <span className='text-muted-foreground text-xs'>
                    {app.description}
                  </span>
                </div>
              </li>
            ))}
          </ul>

          <div className='border-border bg-card overflow-hidden rounded-2xl border'>
            <div className='border-border bg-muted/40 border-b px-5 py-3'>
              <span className='text-muted-foreground font-mono text-[11px] tracking-wider uppercase'>
                Python
              </span>
            </div>
            <pre className='overflow-x-auto px-6 py-5 font-mono text-[12.5px] leading-[1.9]'>
              <code>
                {codeLines.map((line) => (
                  <span key={line.key} className='block'>
                    {line.node}
                  </span>
                ))}
              </code>
            </pre>
            <p className='border-border text-muted-foreground border-t px-6 py-4 text-xs'>
              {t(
                'Change model to any name in the list — nothing else needs updating.'
              )}
            </p>
          </div>
        </div>
      </div>
    </section>
  )
}
