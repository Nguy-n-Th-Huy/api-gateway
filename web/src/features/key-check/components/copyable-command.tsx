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
import { Copy } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

export interface CopyableCommandProps {
  /** Text shown on screen — may be masked. */
  displayText: string
  /** Text placed on the clipboard. Defaults to `displayText` when the
   * command carries no secret (e.g. an install step). */
  copyText?: string
  label: string
}

/** A read-only, copyable one-line command. Copying always uses `copyText`
 * (the real, working command), never the on-screen (possibly masked) text. */
export function CopyableCommand(props: CopyableCommandProps) {
  const { t } = useTranslation()
  const { copyToClipboard } = useCopyToClipboard()

  return (
    <div className='flex items-center gap-2'>
      <code className='bg-muted min-w-0 flex-1 truncate rounded-md px-3 py-2 font-mono text-xs'>
        {props.displayText}
      </code>
      <Button
        type='button'
        variant='outline'
        size='icon-sm'
        aria-label={t('Copy {{label}}', { label: props.label })}
        onClick={() => {
          void copyToClipboard(props.copyText ?? props.displayText)
        }}
      >
        <Copy />
      </Button>
    </div>
  )
}
