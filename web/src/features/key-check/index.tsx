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
import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { Alert, AlertDescription } from '@/components/ui/alert'

import { KeyCheckForm } from './components/key-check-form'
import { ModelStatusSection } from './components/model-status-section'
import { ResultPanel } from './components/result-panel'
import { SetupSection } from './components/setup-section'
import { useKeyCheck } from './hooks/use-key-check'

/**
 * Public, unauthenticated `/key` page: look up an API key's own usage and
 * configuration, see recent model health, and get a one-line setup command.
 * See openspec/changes/add-public-key-check-page.
 */
export function KeyCheck() {
  const { t } = useTranslation()
  const keyCheck = useKeyCheck()
  const resultRegionRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (keyCheck.status === 'success') {
      resultRegionRef.current?.focus()
    }
  }, [keyCheck.status])

  return (
    <PublicLayout>
      <div className='mx-auto flex max-w-3xl flex-col gap-6 py-8'>
        <div className='space-y-1'>
          <h1 className='text-2xl font-semibold'>
            {t('Check your API key')}
          </h1>
          <p className='text-muted-foreground text-sm'>
            {t(
              'Paste an API key to see its usage, status, and setup command. No account needed.'
            )}
          </p>
        </div>

        <KeyCheckForm
          isLoading={keyCheck.status === 'loading'}
          onSubmit={(key) => {
            void keyCheck.submit(key)
          }}
        />

        <div ref={resultRegionRef} tabIndex={-1} aria-live='polite' className='outline-none'>
          {keyCheck.status === 'idle' && (
            <p className='text-muted-foreground text-sm'>
              {t('Enter your key above to see its usage report.')}
            </p>
          )}

          {keyCheck.status === 'error' && keyCheck.errorMessage && (
            <Alert variant='destructive'>
              <AlertDescription>{keyCheck.errorMessage}</AlertDescription>
            </Alert>
          )}

          {keyCheck.status === 'success' && keyCheck.report && (
            <ResultPanel report={keyCheck.report} />
          )}
        </div>

        <ModelStatusSection
          availableModels={keyCheck.report?.available_models ?? null}
        />

        <SetupSection
          report={keyCheck.report}
          checkedKey={keyCheck.checkedKey}
        />
      </div>
    </PublicLayout>
  )
}
