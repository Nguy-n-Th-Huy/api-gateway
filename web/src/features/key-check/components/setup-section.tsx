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
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useStatus } from '@/hooks/use-status'

import {
  APPLICATIONS,
  DEFAULT_APPLICATION_ID,
  DEFAULT_OS_TARGET,
  type OsTarget,
} from '../lib/applications'
import {
  buildSetupCommand,
  canBuildSetupCommand,
  reconcileModelSelections,
} from '../lib/setup-command'
import type { KeyCheckReport } from '../types'
import { CopyableCommand } from './copyable-command'
import { SetupApplicationPicker } from './setup-application-picker'
import { SetupModelSelectors } from './setup-model-selectors'

export interface SetupSectionProps {
  report: KeyCheckReport | null
  /** The raw key from the last successful check. Only ever embedded in the
   * generated command (masked on screen); never rendered on its own. */
  checkedKey: string | null
}

function resolveBaseUrl(status: unknown): string {
  const candidate = (status as Record<string, unknown> | null)?.[
    'server_address'
  ]
  if (typeof candidate === 'string' && candidate.length > 0) {
    return candidate.replace(/\/+$/, '')
  }
  if (typeof window !== 'undefined') return window.location.origin
  return ''
}

/** The setup section: application/OS pickers, the install prerequisite, the
 * per-slot model selectors, and the generated one-line command. Gated on a
 * successful key check. See specs/public-setup-script/spec.md. */
export function SetupSection(props: SetupSectionProps) {
  const { t } = useTranslation()
  const { status: systemStatus } = useStatus()
  const [appId, setAppId] = useState(DEFAULT_APPLICATION_ID)
  const [os, setOs] = useState<OsTarget>(DEFAULT_OS_TARGET)
  const [modelSelections, setModelSelections] = useState<
    Record<string, string>
  >({})

  const app = useMemo(
    () =>
      APPLICATIONS.find((candidate) => candidate.id === appId) ??
      APPLICATIONS[0],
    [appId]
  )
  const availableModels = props.report?.available_models ?? []
  const availableModelsKey = availableModels.join(' ')
  const baseUrl = useMemo(() => resolveBaseUrl(systemStatus), [systemStatus])

  useEffect(() => {
    setModelSelections((previous) =>
      reconcileModelSelections(app, availableModels, previous)
    )
    // availableModelsKey is a stable, order-sensitive proxy for availableModels
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [app, availableModelsKey])

  if (!props.report || !props.checkedKey) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t('Setup')}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className='text-muted-foreground text-sm'>
            {t('Check a key above to generate a setup command.')}
          </p>
        </CardContent>
      </Card>
    )
  }

  const installStep = app.install[os]
  const commandIsAvailable = canBuildSetupCommand(availableModels)
  const command = commandIsAvailable
    ? buildSetupCommand({
        app,
        os,
        key: props.checkedKey,
        baseUrl,
        modelSelections,
      })
    : null

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('Setup')}</CardTitle>
      </CardHeader>
      <CardContent className='flex flex-col gap-4'>
        <SetupApplicationPicker
          appId={appId}
          onAppIdChange={setAppId}
          os={os}
          onOsChange={setOs}
        />

        <div className='flex flex-col gap-1'>
          <span className='text-muted-foreground text-xs'>
            {t('1. Install the CLI')}
          </span>
          <CopyableCommand
            displayText={installStep.command}
            label={t('install command')}
          />
          {installStep.runtimeNoteKey && (
            <span className='text-muted-foreground text-xs'>
              {t(installStep.runtimeNoteKey)}
            </span>
          )}
        </div>

        <SetupModelSelectors
          app={app}
          availableModels={availableModels}
          modelSelections={modelSelections}
          disabled={!commandIsAvailable}
          onSelectionChange={(slotId, model) =>
            setModelSelections((previous) => ({
              ...previous,
              [slotId]: model,
            }))
          }
        />

        {!commandIsAvailable || !command ? (
          <p className='text-muted-foreground text-sm'>
            {t("This key's group has no available model.")}
          </p>
        ) : (
          <div className='flex flex-col gap-1'>
            <span className='text-muted-foreground text-xs'>
              {t('2. Configure the CLI')}
            </span>
            <CopyableCommand
              displayText={command.display}
              copyText={command.copyText}
              label={t('setup command')}
            />
          </div>
        )}
      </CardContent>
    </Card>
  )
}
