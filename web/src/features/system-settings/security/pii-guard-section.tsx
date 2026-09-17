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
import { useId, useMemo, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'
import type { PiiGuardMode, PiiGuardPlaceholderStyle } from '../types'
import { safeNumberFieldProps } from '../utils/numeric-field'

/**
 * Entity identifiers the backend detector accepts. The identifier is what the
 * guard stores and what the replacement-override JSON is keyed by, so it is
 * shown next to the operator-facing label.
 */
const ENTITY_TYPES = [
  { value: 'EMAIL', labelKey: 'Email addresses' },
  { value: 'PHONE', labelKey: 'Phone numbers' },
  { value: 'CREDIT_CARD', labelKey: 'Credit card numbers' },
  { value: 'IP_ADDRESS', labelKey: 'IP addresses' },
  { value: 'URL', labelKey: 'URLs' },
  { value: 'UUID', labelKey: 'UUIDs' },
  { value: 'IBAN', labelKey: 'IBAN account numbers' },
  { value: 'US_SSN', labelKey: 'US Social Security numbers' },
  { value: 'PASSPORT', labelKey: 'Passport numbers' },
  { value: 'DRIVER_LICENSE', labelKey: 'Driver license numbers' },
  { value: 'VN_ID', labelKey: 'Vietnamese citizen identity numbers' },
  { value: 'VN_TAX_CODE', labelKey: 'Vietnamese tax codes' },
  { value: 'DATE_OF_BIRTH', labelKey: 'Dates of birth' },
  { value: 'PERSON', labelKey: 'Person names' },
  { value: 'LOCATION', labelKey: 'Locations' },
  { value: 'ORGANIZATION', labelKey: 'Organizations' },
  { value: 'CUSTOM', labelKey: 'Custom keywords' },
] as const

const ENTITY_TYPE_VALUES: string[] = ENTITY_TYPES.map((entity) => entity.value)

/** Shipped default of the template placeholder style, from the Go config. */
const DEFAULT_TOKEN_TEMPLATE = '«{{type}}_{{index}}»'

type PiiGuardDefaults = {
  'piiguard.enabled': boolean
  'piiguard.mask_request': boolean
  'piiguard.unmask_response': boolean
  'piiguard.mode': PiiGuardMode
  'piiguard.placeholder_style': PiiGuardPlaceholderStyle
  'piiguard.token_template': string
  'piiguard.secret': string
  'piiguard.max_body_bytes': number
  'piiguard.enabled_entity_types': string[]
  'piiguard.disabled_entity_types': string[]
  'piiguard.custom_keywords': string[]
  'piiguard.min_keyword_length': number
  'piiguard.fakes': string
  'piiguard.require_mask_reject': boolean
}

type PiiGuardSectionProps = {
  defaultValues: PiiGuardDefaults
}

const nonNegativeInteger = (t: (key: string) => string) =>
  z
    .number()
    .int(t('Enter a whole number'))
    .nonnegative(t('Enter a number of zero or more'))

const createPiiGuardSchema = (t: (key: string) => string) =>
  z
    .object({
      piiguard: z.object({
        enabled: z.boolean(),
        mask_request: z.boolean(),
        unmask_response: z.boolean(),
        mode: z.enum(['pseudonym', 'redact']),
        placeholder_style: z.enum(['typed', 'template']),
        token_template: z.string(),
        secret: z.string(),
        max_body_bytes: nonNegativeInteger(t),
        enabled_entity_types: z.array(z.string()),
        disabled_entity_types: z.array(z.string()),
        custom_keywords: z.string(),
        min_keyword_length: nonNegativeInteger(t),
        fakes: z.string(),
        require_mask_reject: z.boolean(),
      }),
    })
    .superRefine((values, ctx) => {
      const guard = values.piiguard

      if (guard.enabled && !guard.mask_request && !guard.unmask_response) {
        ctx.addIssue({
          code: 'custom',
          path: ['piiguard', 'enabled'],
          message: t(
            'An enabled guard must mask requests or restore responses'
          ),
        })
      }

      if (guard.mode === 'redact' && guard.unmask_response) {
        ctx.addIssue({
          code: 'custom',
          path: ['piiguard', 'unmask_response'],
          message: t('Redact mode cannot restore responses'),
        })
      }

      if (!isFakesJson(guard.fakes)) {
        ctx.addIssue({
          code: 'custom',
          path: ['piiguard', 'fakes'],
          message: t(
            'Enter a JSON object of entity types mapped to real values and their replacements'
          ),
        })
      }
    })

type PiiGuardSchema = ReturnType<typeof createPiiGuardSchema>
type PiiGuardFormInput = z.input<PiiGuardSchema>
type PiiGuardFormValues = z.output<PiiGuardSchema>

const splitKeywords = (value: string): string[] =>
  value
    .split('\n')
    .map((keyword) => keyword.trim())
    .filter(Boolean)

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

/**
 * The option stores the override map as JSON text. A guard that was never
 * configured hands back the JSON literal `null`, and an emptied field means
 * "no override", so both are shown as an empty textarea.
 */
const normalizeFakesText = (value: string): string => {
  const trimmed = value.trim()
  if (trimmed === '' || trimmed === 'null') return ''
  return trimmed
}

const isFakesJson = (value: string): boolean => {
  const trimmed = normalizeFakesText(value)
  if (trimmed === '') return true

  let parsed: unknown
  try {
    parsed = JSON.parse(trimmed)
  } catch {
    return false
  }

  if (!isPlainObject(parsed)) return false

  return Object.values(parsed).every(
    (overrides) =>
      isPlainObject(overrides) &&
      Object.values(overrides).every(
        (replacement) => typeof replacement === 'string'
      )
  )
}

type EntityTypeCheckboxGroupProps = {
  label: string
  value: string[]
  disabled: boolean
  onChange: (value: string[]) => void
}

function EntityTypeCheckboxGroup(props: EntityTypeCheckboxGroupProps) {
  const { t } = useTranslation()
  const groupId = useId()

  const handleToggle = (entityType: string, checked: boolean) => {
    const selected = new Set(props.value)
    if (checked) {
      selected.add(entityType)
    } else {
      selected.delete(entityType)
    }

    // Keep the canonical identifier order so a saved list is comparable to the
    // stored one, and keep identifiers this build does not know about.
    const known = ENTITY_TYPE_VALUES.filter((value) => selected.has(value))
    const other = props.value.filter(
      (value) => !ENTITY_TYPE_VALUES.includes(value)
    )
    props.onChange([...known, ...other])
  }

  return (
    <div
      role='group'
      aria-label={props.label}
      className='grid gap-2 sm:grid-cols-2 xl:grid-cols-3'
    >
      {ENTITY_TYPES.map((entity) => {
        const inputId = `${groupId}-${entity.value}`
        return (
          <div key={entity.value} className='flex min-w-0 items-center gap-2'>
            <Checkbox
              id={inputId}
              checked={props.value.includes(entity.value)}
              onCheckedChange={(checked) =>
                handleToggle(entity.value, checked === true)
              }
              disabled={props.disabled}
            />
            <Label htmlFor={inputId} className='min-w-0 text-sm font-normal'>
              <span>{t(entity.labelKey)}</span>
              <code className='text-muted-foreground text-xs'>
                {entity.value}
              </code>
            </Label>
          </div>
        )
      })}
    </div>
  )
}

const buildFormDefaults = (
  defaults: PiiGuardDefaults
): PiiGuardFormInput => ({
  piiguard: {
    enabled: defaults['piiguard.enabled'],
    mask_request: defaults['piiguard.mask_request'],
    unmask_response: defaults['piiguard.unmask_response'],
    mode: defaults['piiguard.mode'],
    placeholder_style: defaults['piiguard.placeholder_style'],
    token_template: defaults['piiguard.token_template'],
    // The stored seed is never returned by the options API, so the field
    // always starts empty and only a typed value is written back.
    secret: '',
    max_body_bytes: defaults['piiguard.max_body_bytes'],
    enabled_entity_types: defaults['piiguard.enabled_entity_types'],
    disabled_entity_types: defaults['piiguard.disabled_entity_types'],
    custom_keywords: defaults['piiguard.custom_keywords'].join('\n'),
    min_keyword_length: defaults['piiguard.min_keyword_length'],
    fakes: normalizeFakesText(defaults['piiguard.fakes']),
    require_mask_reject: defaults['piiguard.require_mask_reject'],
  },
})

const normalizeDefaults = (defaults: PiiGuardDefaults): PiiGuardDefaults => ({
  ...defaults,
  'piiguard.token_template': defaults['piiguard.token_template'].trim(),
  // The stored seed is never returned by the options API.
  'piiguard.secret': '',
  'piiguard.fakes': normalizeFakesText(defaults['piiguard.fakes']),
})

const normalizeFormValues = (
  values: PiiGuardFormValues
): PiiGuardDefaults => ({
  'piiguard.enabled': values.piiguard.enabled,
  'piiguard.mask_request': values.piiguard.mask_request,
  'piiguard.unmask_response': values.piiguard.unmask_response,
  'piiguard.mode': values.piiguard.mode,
  'piiguard.placeholder_style': values.piiguard.placeholder_style,
  'piiguard.token_template': values.piiguard.token_template.trim(),
  'piiguard.secret': values.piiguard.secret.trim(),
  'piiguard.max_body_bytes': values.piiguard.max_body_bytes,
  'piiguard.enabled_entity_types': values.piiguard.enabled_entity_types,
  'piiguard.disabled_entity_types': values.piiguard.disabled_entity_types,
  'piiguard.custom_keywords': splitKeywords(values.piiguard.custom_keywords),
  'piiguard.min_keyword_length': values.piiguard.min_keyword_length,
  'piiguard.fakes': normalizeFakesText(values.piiguard.fakes),
  'piiguard.require_mask_reject': values.piiguard.require_mask_reject,
})

const isEqual = (a: unknown, b: unknown) => {
  if (Array.isArray(a) && Array.isArray(b)) {
    return JSON.stringify(a) === JSON.stringify(b)
  }
  return a === b
}

export function PiiGuardSection({ defaultValues }: PiiGuardSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const baselineRef = useRef<PiiGuardDefaults>(normalizeDefaults(defaultValues))

  const formDefaults = useMemo(
    () => buildFormDefaults(defaultValues),
    [defaultValues]
  )

  const form = useForm<PiiGuardFormInput, unknown, PiiGuardFormValues>({
    resolver: zodResolver(createPiiGuardSchema(t)),
    defaultValues: formDefaults,
  })

  useResetForm(form, formDefaults)

  const enabled = form.watch('piiguard.enabled')
  const mode = form.watch('piiguard.mode')
  const placeholderStyle = form.watch('piiguard.placeholder_style')
  const redact = mode === 'redact'

  const handleModeChange = (next: PiiGuardMode) => {
    form.setValue('piiguard.mode', next)
    // Redaction is one-way, so the backend rejects it together with response
    // restoration; clearing it here keeps the two controls consistent.
    if (next === 'redact') {
      form.setValue('piiguard.unmask_response', false)
    }
  }

  const onSubmit = async (values: PiiGuardFormValues) => {
    const normalized = normalizeFormValues(values)

    // Mode and response restoration depend on each other: entering redact mode
    // must clear restoration before the mode is stored, while leaving redact
    // mode must store the mode before restoration is switched back on, or the
    // backend rejects the intermediate state.
    const modeOrder: Array<keyof PiiGuardDefaults> = redact
      ? ['piiguard.unmask_response', 'piiguard.mode']
      : ['piiguard.mode', 'piiguard.unmask_response']

    const ordered: Array<keyof PiiGuardDefaults> = [
      ...modeOrder,
      'piiguard.mask_request',
      'piiguard.placeholder_style',
      'piiguard.token_template',
      'piiguard.max_body_bytes',
      'piiguard.enabled_entity_types',
      'piiguard.disabled_entity_types',
      'piiguard.custom_keywords',
      'piiguard.min_keyword_length',
      'piiguard.fakes',
      'piiguard.require_mask_reject',
      // Enabling the guard last, so the masking flags are already stored.
      'piiguard.enabled',
    ]

    const updates = ordered.filter(
      (key) => !isEqual(normalized[key], baselineRef.current[key])
    )

    // A stored empty seed means "use a per-process random seed", and saving an
    // empty value would clear the seed that is already in place, so the secret
    // is only written when the admin typed a new one.
    const secret = normalized['piiguard.secret']
    if (secret !== '') {
      updates.push('piiguard.secret')
    }

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const key of updates) {
      const value = normalized[key]
      if (Array.isArray(value)) {
        await updateOption.mutateAsync({ key, value: JSON.stringify(value) })
        continue
      }
      if (key === 'piiguard.fakes') {
        // An empty override map must be written as valid JSON: the backend
        // skips a value it cannot decode, which would leave the stored option
        // and the live setting disagreeing.
        await updateOption.mutateAsync({
          key,
          value: value === '' ? '{}' : value,
        })
        continue
      }
      await updateOption.mutateAsync({ key, value })
    }

    baselineRef.current = { ...normalized, 'piiguard.secret': '' }
  }

  return (
    <SettingsSection title={t('PII Guard')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save PII Guard settings'
          />

          <Alert>
            <AlertTitle>{t('How the PII guard works')}</AlertTitle>
            <AlertDescription>
              <p>
                {t(
                  'Detected values in the request body are replaced before the body leaves the gateway, and the real values are restored in the response, so the upstream provider only ever sees the stand-ins.'
                )}
              </p>
              <p>
                {t(
                  'Everything runs in-process. The real-to-stand-in mapping is sensitive: it is never written to a log, a metric or the database, and it lives only for the lifetime of one request.'
                )}
              </p>
              <p>
                {t(
                  'An enabled guard must mask requests or restore responses; saving it with both off is rejected.'
                )}
              </p>
              <p>
                {t(
                  'Redact mode is one-way, so it cannot be combined with restoring responses.'
                )}
              </p>
            </AlertDescription>
          </Alert>

          <FormField
            control={form.control}
            name='piiguard.enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable PII Guard')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Master switch. While it is off, request bodies are forwarded upstream byte for byte.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <div className='grid min-w-0 gap-4 sm:grid-cols-2'>
            <FormField
              control={form.control}
              name='piiguard.mode'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Masking mode')}</FormLabel>
                  <Select
                    items={[
                      {
                        value: 'pseudonym',
                        label: t('Pseudonym (deterministic stand-in)'),
                      },
                      {
                        value: 'redact',
                        label: t('Redact (typed placeholder)'),
                      },
                    ]}
                    value={field.value}
                    onValueChange={(value) =>
                      handleModeChange(value as PiiGuardMode)
                    }
                    disabled={!enabled}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent alignItemWithTrigger={false}>
                      <SelectGroup>
                        <SelectItem value='pseudonym'>
                          {t('Pseudonym (deterministic stand-in)')}
                        </SelectItem>
                        <SelectItem value='redact'>
                          {t('Redact (typed placeholder)')}
                        </SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    {t(
                      'Pseudonym keeps a stand-in of the same shape that is stable within one request, while redact replaces each value with a typed placeholder.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='piiguard.placeholder_style'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Placeholder style')}</FormLabel>
                  <Select
                    items={[
                      { value: 'typed', label: t('Typed placeholder') },
                      { value: 'template', label: t('Custom template') },
                    ]}
                    value={field.value}
                    onValueChange={(value) =>
                      field.onChange(value as PiiGuardPlaceholderStyle)
                    }
                    disabled={!enabled || !redact}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent alignItemWithTrigger={false}>
                      <SelectGroup>
                        <SelectItem value='typed'>
                          {t('Typed placeholder')}
                        </SelectItem>
                        <SelectItem value='template'>
                          {t('Custom template')}
                        </SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    {t('Shape of the placeholder written in redact mode.')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <FormField
            control={form.control}
            name='piiguard.mask_request'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Mask request bodies')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Replace detected values in the body that is sent to the upstream provider.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={!enabled}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='piiguard.unmask_response'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Restore values in responses')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Put the real values back into the upstream response. Unavailable in redact mode, which cannot be reversed.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={!enabled || redact}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='piiguard.require_mask_reject'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>
                    {t('Reject bodies that cannot be masked')}
                  </FormLabel>
                  <FormDescription>
                    {t(
                      'Fail the request instead of forwarding a body the guard could not mask, for example one above the size limit.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={!enabled}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <div className='grid min-w-0 gap-4 sm:grid-cols-2'>
            <FormField
              control={form.control}
              name='piiguard.max_body_bytes'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Maximum body size (bytes)')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      step={1}
                      {...safeNumberFieldProps(field)}
                      disabled={!enabled}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Bodies larger than this limit are not masked. The shipped default is 1048576 bytes (1 MiB).'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='piiguard.min_keyword_length'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Minimum keyword length')}</FormLabel>
                  <FormControl>
                    <Input
                      type='number'
                      min={0}
                      step={1}
                      {...safeNumberFieldProps(field)}
                      disabled={!enabled}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Shortest custom keyword that is masked, so a one-character keyword cannot shred every message. The shipped default is 3.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='piiguard.token_template'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Placeholder template')}</FormLabel>
                  <FormControl>
                    <Input
                      type='text'
                      placeholder={DEFAULT_TOKEN_TEMPLATE}
                      autoComplete='off'
                      {...field}
                      disabled={
                        !enabled || !redact || placeholderStyle !== 'template'
                      }
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Pattern used by the custom template style. It receives the entity type and the per-type index of the value being replaced.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='piiguard.secret'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Seed secret')}</FormLabel>
                  <FormControl>
                    <Input
                      type='password'
                      placeholder={t('Leave blank to keep the current seed')}
                      autoComplete='new-password'
                      {...field}
                      disabled={!enabled}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Seeds the deterministic stand-in generator, so the same real value always maps to the same stand-in. Without a seed a per-process random seed is used and stand-ins change on every restart.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <FormField
            control={form.control}
            name='piiguard.enabled_entity_types'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Allowed entity types')}</FormLabel>
                <EntityTypeCheckboxGroup
                  label={t('Allowed entity types')}
                  value={field.value}
                  onChange={field.onChange}
                  disabled={!enabled}
                />
                <FormDescription>
                  {t(
                    'Restrict detection to the entity types checked here. With none checked, every entity type that is on by default is detected.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='piiguard.disabled_entity_types'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Disabled entity types')}</FormLabel>
                <EntityTypeCheckboxGroup
                  label={t('Disabled entity types')}
                  value={field.value}
                  onChange={field.onChange}
                  disabled={!enabled}
                />
                <FormDescription>
                  {t(
                    'Entity types checked here are never detected. This list wins over the allowed list.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='piiguard.custom_keywords'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Custom keywords')}</FormLabel>
                <FormControl>
                  <Textarea
                    rows={6}
                    placeholder={t('Enter one keyword per line')}
                    {...field}
                    disabled={!enabled}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Literals that are always masked as CUSTOM, for identifiers no general pattern knows, for example a regulated number of your own. One keyword per line; empty lines are ignored.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='piiguard.fakes'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Replacement overrides (JSON)')}</FormLabel>
                <FormControl>
                  <Textarea
                    rows={6}
                    placeholder={t(
                      '{"EMAIL": {"alex@example.com": "fake@example.com"}}'
                    )}
                    spellCheck={false}
                    className='font-mono text-xs'
                    {...field}
                    disabled={!enabled}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Override the generated stand-in per entity type and real value. Values that are not listed here keep a generated stand-in.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
