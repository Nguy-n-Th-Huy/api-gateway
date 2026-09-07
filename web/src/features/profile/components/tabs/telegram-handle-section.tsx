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
import { AtSign, Loader2, Pencil } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  isValidTelegramHandle,
  normalizeTelegramHandle,
} from '@/lib/telegram-handle'

import { updateUserProfile } from '../../api'

// ============================================================================
// Telegram Handle Section
// ============================================================================
//
// Displays and edits the self-declared/verified Telegram handle exposed on
// the self-user payload (specs/telegram/account-link, "Accounts carry a
// Telegram handle distinct from the Telegram identity"). This value is
// display-only: it is never a lookup key for authentication, binding, or
// authorization, so correcting it here never touches the account's Telegram
// binding.

interface TelegramHandleSectionProps {
  currentHandle?: string
  onUpdate: () => void
}

export function TelegramHandleSection(props: TelegramHandleSectionProps) {
  const { t } = useTranslation()
  const [editing, setEditing] = useState(false)
  const [value, setValue] = useState(props.currentHandle ?? '')
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const startEditing = () => {
    setValue(props.currentHandle ?? '')
    setError(null)
    setEditing(true)
  }

  const cancelEditing = () => {
    setEditing(false)
    setError(null)
  }

  const handleSave = async () => {
    const normalized = normalizeTelegramHandle(value)
    if (normalized !== '' && !isValidTelegramHandle(normalized)) {
      setError(
        t(
          'Telegram handle must be 5-32 characters, start with a letter, and contain only letters, digits, and underscores'
        )
      )
      return
    }

    setSaving(true)
    setError(null)
    try {
      const response = await updateUserProfile({
        telegram_username: normalized,
      })
      if (response.success) {
        toast.success(t('Telegram handle updated'))
        setEditing(false)
        props.onUpdate()
      } else {
        toast.error(response.message || t('Failed to update Telegram handle'))
      }
    } catch {
      toast.error(t('Failed to update Telegram handle'))
    } finally {
      setSaving(false)
    }
  }

  if (!editing) {
    return (
      <div className='flex items-center justify-between gap-2.5 rounded-lg border p-2.5 sm:gap-3 sm:p-3'>
        <div className='flex min-w-0 items-center gap-2.5 sm:gap-3'>
          <div className='bg-muted shrink-0 rounded-md p-1.5 sm:p-2'>
            <AtSign className='h-4 w-4' />
          </div>
          <div className='min-w-0'>
            <p className='text-sm font-medium'>{t('Telegram handle')}</p>
            <p className='text-muted-foreground truncate text-xs'>
              {props.currentHandle
                ? `@${props.currentHandle}`
                : t('Not set')}
            </p>
          </div>
        </div>
        <Button
          variant='outline'
          size='sm'
          className='h-7 shrink-0 gap-1 px-2.5 text-xs'
          onClick={startEditing}
        >
          <Pencil className='h-3 w-3' />
          {t('Edit')}
        </Button>
      </div>
    )
  }

  return (
    <div className='space-y-2 rounded-lg border p-2.5 sm:p-3'>
      <p className='text-sm font-medium'>{t('Telegram handle')}</p>
      <Input
        value={value}
        onChange={(event) => setValue(event.target.value)}
        placeholder={t('e.g. john_doe99')}
        disabled={saving}
        autoFocus
      />
      {error && <p className='text-destructive text-xs'>{error}</p>}
      <div className='flex justify-end gap-2'>
        <Button
          variant='outline'
          size='sm'
          className='h-7 px-2.5 text-xs'
          onClick={cancelEditing}
          disabled={saving}
        >
          {t('Cancel')}
        </Button>
        <Button
          size='sm'
          className='h-7 px-2.5 text-xs'
          onClick={handleSave}
          disabled={saving}
        >
          {saving && <Loader2 className='mr-1 h-3 w-3 animate-spin' />}
          {t('Save')}
        </Button>
      </div>
    </div>
  )
}
