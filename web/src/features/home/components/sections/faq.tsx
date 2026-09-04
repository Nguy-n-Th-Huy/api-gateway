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
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Markdown } from '@/components/ui/markdown'
import { useFAQ } from '@/features/dashboard/hooks/use-status-data'
import type { FAQItem } from '@/features/dashboard/types'

interface FaqProps {
  className?: string
}

/**
 * Renders only the administrator-configured FAQ entries. Returns null while
 * still loading, when the feature is disabled, and when it is enabled with
 * no entries — the block never renders an empty heading or placeholder.
 */
export function Faq(_props: FaqProps) {
  const { t } = useTranslation()
  const { items, loading } = useFAQ()

  if (loading || items.length === 0) {
    return null
  }

  return (
    <section className='relative z-10 px-6 py-24 md:py-32'>
      <div className='mx-auto grid max-w-6xl gap-10 lg:grid-cols-[0.85fr_1.4fr] lg:gap-14'>
        <AnimateInView className='flex flex-col gap-3'>
          <p className='text-muted-foreground text-xs font-medium tracking-widest uppercase'>
            {t('FAQ')}
          </p>
          <h2 className='text-2xl leading-tight font-bold tracking-tight md:text-3xl'>
            {t('Questions people ask first')}
          </h2>
        </AnimateInView>

        <AnimateInView delay={100}>
          <Accordion className='w-full'>
            {items.map((item: FAQItem, idx: number) => {
              const key = item.id ?? `home-faq-${idx}`
              return (
                <AccordionItem
                  key={key}
                  value={`home-faq-${key}`}
                  className='border-border/60'
                >
                  <AccordionTrigger className='text-start hover:no-underline'>
                    <Markdown className='text-sm leading-relaxed font-semibold'>
                      {item.question}
                    </Markdown>
                  </AccordionTrigger>
                  <AccordionContent>
                    <Markdown className='text-muted-foreground text-sm'>
                      {item.answer}
                    </Markdown>
                  </AccordionContent>
                </AccordionItem>
              )
            })}
          </Accordion>
        </AnimateInView>
      </div>
    </section>
  )
}
