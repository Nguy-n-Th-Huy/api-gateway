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
import { CheckCircle2, KeyRound, Loader2, XCircle } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import { redeemTelegramLinkCode } from '../../api'

// ============================================================================
// Telegram Link Code Section
// ============================================================================
//
// Redeems a short-lived link code the bot integration issued in a chat
// (specs/telegram/account-link, "Link codes carry a linking intent from a
// chat to a browser session"). This sits alongside the existing Telegram
// Login Widget binding above — it is an alternative entry point into the
// same binding, not a replacement for it. The backend already returns a
// distinct, localized message for every refusal reason (unknown/expired/
// consumed code, already-bound identity or account, disabled account,
// revoked session), so this component simply surfaces that message.

type RedeemStatus = 'idle' | 'pending' | 'success' | 'error'

interface TelegramLinkCodeSectionProps {
  onSuccess: () => void
}

export function TelegramLinkCodeSection(props: TelegramLinkCodeSectionProps) {
  const { t } = useTranslation()
  const [code, setCode] = useState('')
  const [status, setStatus] = useState<RedeemStatus>('idle')
  const [message, setMessage] = useState<string | null>(null)

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    const trimmed = code.trim()
    if (!trimmed) {
      setStatus('error')
      setMessage(t('Please enter the link code'))
      return
    }

    setStatus('pending')
    setMessage(null)
    try {
      const response = await redeemTelegramLinkCode(trimmed)
      if (response.success) {
        setStatus('success')
        setMessage(response.message || t('Telegram account linked'))
        setCode('')
        props.onSuccess()
      } else {
        setStatus('error')
        setMessage(response.message || t('Failed to link Telegram account'))
      }
    } catch {
      setStatus('error')
      setMessage(t('Failed to link Telegram account'))
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      className='space-y-2 rounded-lg border p-2.5 sm:p-3'
    >
      <div className='flex items-center gap-2.5 sm:gap-3'>
        <div className='bg-muted shrink-0 rounded-md p-1.5 sm:p-2'>
          <KeyRound className='h-4 w-4' />
        </div>
        <div className='min-w-0'>
          <Label htmlFor='telegram-link-code' className='text-sm font-medium'>
            {t('Have a link code from the bot?')}
          </Label>
          <p className='text-muted-foreground truncate text-xs'>
            {t('Enter the code the Telegram bot gave you to bind your account')}
          </p>
        </div>
      </div>

      <div className='flex gap-2'>
        <Input
          id='telegram-link-code'
          value={code}
          onChange={(event) => {
            setCode(event.target.value.toUpperCase())
            if (status !== 'idle') {
              setStatus('idle')
              setMessage(null)
            }
          }}
          placeholder={t('e.g. AB3D9F2K')}
          maxLength={8}
          autoComplete='off'
          disabled={status === 'pending'}
          className='font-mono uppercase'
        />
        <Button type='submit' disabled={status === 'pending'}>
          {status === 'pending' && (
            <Loader2 className='mr-1 h-3.5 w-3.5 animate-spin' />
          )}
          {t('Link')}
        </Button>
      </div>

      {status === 'success' && message && (
        <Alert variant='success'>
          <CheckCircle2 className='h-4 w-4' />
          <AlertDescription>{message}</AlertDescription>
        </Alert>
      )}
      {status === 'error' && message && (
        <Alert variant='destructive'>
          <XCircle className='h-4 w-4' />
          <AlertDescription>{message}</AlertDescription>
        </Alert>
      )}
    </form>
  )
}
