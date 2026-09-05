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
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'

import {
  getKeyCheckFormSchema,
  KEY_CHECK_FORM_DEFAULT_VALUES,
  type KeyCheckFormValues,
} from '../lib/validation'

export interface KeyCheckFormProps {
  isLoading: boolean
  onSubmit: (key: string) => void
}

/** The key input and submit control. Submits on Enter or the button, per
 * specs/public-key-check/spec.md — "Key check page validates input before
 * calling the API". */
export function KeyCheckForm(props: KeyCheckFormProps) {
  const { t } = useTranslation()
  const form = useForm<KeyCheckFormValues>({
    resolver: zodResolver(getKeyCheckFormSchema(t)),
    defaultValues: KEY_CHECK_FORM_DEFAULT_VALUES,
  })

  function handleSubmit(values: KeyCheckFormValues) {
    props.onSubmit(values.key)
  }

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(handleSubmit)}
        className='flex flex-col items-start gap-3 sm:flex-row'
        noValidate
      >
        <FormField
          control={form.control}
          name='key'
          render={({ field }) => (
            <FormItem className='w-full flex-1'>
              <FormLabel>{t('API key')}</FormLabel>
              <FormControl>
                <Input
                  placeholder={t('Paste your API key, e.g. sk-...')}
                  autoComplete='off'
                  autoCorrect='off'
                  autoCapitalize='off'
                  spellCheck={false}
                  disabled={props.isLoading}
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <Button
          type='submit'
          disabled={props.isLoading}
          className='sm:mt-6'
        >
          {props.isLoading && <Spinner />}
          {t('Check key')}
        </Button>
      </form>
    </Form>
  )
}
