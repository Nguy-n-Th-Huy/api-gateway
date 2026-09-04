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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Code2, Eye, ShieldAlert } from 'lucide-react'
import * as React from 'react'
import { useForm, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { JsonCodeEditor } from '@/components/json-code-editor'
import { RiskAcknowledgementDialog } from '@/components/risk-acknowledgement-dialog'
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
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
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { cn } from '@/lib/utils'

import { confirmPaymentCompliance } from '../api'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'
import { AmountDiscountVisualEditor } from './amount-discount-visual-editor'
import { AmountOptionsVisualEditor } from './amount-options-visual-editor'
import {
  formatJsonForEditor,
  getJsonError,
  normalizeJsonForComparison,
} from './utils'

const SEPAY_EXPIRY_MIN = 1
const SEPAY_EXPIRY_MAX = 7 * 24 * 60

const paymentSchema = z.object({
  Price: z.coerce.number().min(0),
  MinTopUp: z.coerce.number().min(0),
  AmountOptions: z.string().superRefine((value, ctx) => {
    const error = getJsonError(value, (parsed) => Array.isArray(parsed))
    if (error) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: error })
    }
  }),
  AmountDiscount: z.string().superRefine((value, ctx) => {
    const error = getJsonError(
      value,
      (parsed) => !!parsed && typeof parsed === 'object' && !Array.isArray(parsed)
    )
    if (error) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: error })
    }
  }),
  SePayEnabled: z.boolean(),
  SePayBankAccountNumber: z.string(),
  SePayBankCode: z.string(),
  SePayAccountHolder: z.string(),
  SePayWebhookApiKey: z.string(),
  SePayMinTopUp: z.coerce.number().min(0),
  SePayOrderExpiryMinutes: z.coerce.number().int().min(SEPAY_EXPIRY_MIN).max(SEPAY_EXPIRY_MAX),
})

type PaymentFormValues = z.infer<typeof paymentSchema>

const CURRENT_COMPLIANCE_TERMS_VERSION = 'v1'
const paymentTabContentClassName = 'mt-6 min-w-0'

type PaymentComplianceDefaults = {
  confirmed: boolean
  termsVersion: string
  confirmedAt: number
  confirmedBy: number
}

export type GeneralDefaults = {
  Price: number
  MinTopUp: number
  AmountOptions: string
  AmountDiscount: string
}

export type SePayDefaults = {
  SePayEnabled: boolean
  SePayBankAccountNumber: string
  SePayBankCode: string
  SePayAccountHolder: string
  SePayMinTopUp: number
  SePayOrderExpiryMinutes: number
}

type PaymentSettingsSectionProps = {
  generalDefaults: GeneralDefaults
  sepayDefaults: SePayDefaults
  complianceDefaults: PaymentComplianceDefaults
}

export function PaymentSettingsSection({
  generalDefaults,
  sepayDefaults,
  complianceDefaults,
}: PaymentSettingsSectionProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const updateOption = useUpdateOption()

  const initialFormValues = React.useMemo<PaymentFormValues>(
    () => ({
      Price: generalDefaults.Price,
      MinTopUp: generalDefaults.MinTopUp,
      AmountOptions: generalDefaults.AmountOptions,
      AmountDiscount: generalDefaults.AmountDiscount,
      SePayEnabled: sepayDefaults.SePayEnabled,
      SePayBankAccountNumber: sepayDefaults.SePayBankAccountNumber,
      SePayBankCode: sepayDefaults.SePayBankCode,
      SePayAccountHolder: sepayDefaults.SePayAccountHolder,
      SePayWebhookApiKey: '',
      SePayMinTopUp: sepayDefaults.SePayMinTopUp,
      SePayOrderExpiryMinutes: sepayDefaults.SePayOrderExpiryMinutes,
    }),
    [generalDefaults, sepayDefaults]
  )
  const initialRef = React.useRef(initialFormValues)
  const defaultsSignature = React.useMemo(
    () => JSON.stringify(initialFormValues),
    [initialFormValues]
  )

  const [amountOptionsVisualMode, setAmountOptionsVisualMode] = React.useState(true)
  const [amountDiscountVisualMode, setAmountDiscountVisualMode] = React.useState(true)
  const [showComplianceDialog, setShowComplianceDialog] = React.useState(false)

  const complianceStatements = React.useMemo(
    () => [
      t('You have legally obtained authorization for the connected model APIs, accounts, keys, and quotas.'),
      t('You commit to using upstream APIs, accounts, keys, quotas, and service capabilities only within the scope of lawful authorization obtained from upstream service providers, model service providers, or relevant rights holders, and will not conduct unauthorized resale, trafficking, distribution, or other non-compliant commercialization.'),
      t('If you provide generative AI services to the public in mainland China, you will fulfill legal obligations including filing, security assessment, content safety, complaint handling, generated content labeling, log retention, and personal information protection.'),
      t('You commit not to use this system to implement, assist with, or indirectly implement acts that violate applicable laws and regulations, regulatory requirements, platform rules, public interests, or the lawful rights and interests of third parties.'),
      t('You understand and independently bear legal responsibility arising from deployment, operation, and charging behavior.'),
      t('You understand this compliance reminder is only for risk notice and does not constitute legal advice, a compliance review conclusion, or a guarantee of the legality of your use of this system; you should consult professional legal or compliance advisors based on your actual business scenario.'),
    ],
    [t]
  )

  const complianceRequiredText = t(
    'I have read and understood the above compliance reminder, acknowledge the related legal risks, and confirm that I bear legal responsibility arising from deployment, operation, and charging behavior.'
  )
  const complianceRequiredTextParts = React.useMemo(
    () => [
      { type: 'input' as const, text: t('I have read and understood the above compliance reminder') },
      { type: 'static' as const, text: t('，') },
      { type: 'input' as const, text: t('acknowledge the related legal risks') },
      { type: 'static' as const, text: t('，and ') },
      { type: 'input' as const, text: t('confirm that I bear legal responsibility arising from deployment') },
      { type: 'static' as const, text: t('、') },
      { type: 'input' as const, text: t('operation and charging behavior') },
    ],
    [t]
  )

  const complianceConfirmed =
    complianceDefaults.confirmed && complianceDefaults.termsVersion === CURRENT_COMPLIANCE_TERMS_VERSION

  const confirmComplianceMutation = useMutation({
    mutationFn: confirmPaymentCompliance,
    onSuccess: (data) => {
      if (data.success) {
        toast.success(t('Compliance confirmed successfully'))
        setShowComplianceDialog(false)
        queryClient.invalidateQueries({ queryKey: ['system-options'] })
      } else {
        toast.error(data.message || t('Failed to confirm compliance'))
      }
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Failed to confirm compliance'))
    },
  })

  const form = useForm<PaymentFormValues>({
    resolver: zodResolver(paymentSchema) as Resolver<PaymentFormValues>,
    mode: 'onChange',
    defaultValues: {
      ...initialFormValues,
      AmountOptions: formatJsonForEditor(initialFormValues.AmountOptions),
      AmountDiscount: formatJsonForEditor(initialFormValues.AmountDiscount),
    },
  })

  const { isSubmitting } = form.formState

  React.useEffect(() => {
    const parsedDefaults = JSON.parse(defaultsSignature) as PaymentFormValues
    initialRef.current = parsedDefaults
    form.reset({
      ...parsedDefaults,
      AmountOptions: formatJsonForEditor(parsedDefaults.AmountOptions),
      AmountDiscount: formatJsonForEditor(parsedDefaults.AmountDiscount),
      SePayWebhookApiKey: '',
    })
  }, [defaultsSignature, form])

  const onSubmit = async (values: PaymentFormValues) => {
    const sanitized = {
      Price: values.Price,
      MinTopUp: values.MinTopUp,
      AmountOptions: values.AmountOptions.trim(),
      AmountDiscount: values.AmountDiscount.trim(),
      SePayEnabled: values.SePayEnabled,
      SePayBankAccountNumber: values.SePayBankAccountNumber.trim(),
      SePayBankCode: values.SePayBankCode.trim(),
      SePayAccountHolder: values.SePayAccountHolder.trim(),
      SePayWebhookApiKey: values.SePayWebhookApiKey.trim(),
      SePayMinTopUp: values.SePayMinTopUp,
      SePayOrderExpiryMinutes: values.SePayOrderExpiryMinutes,
    }

    const initial = {
      Price: initialRef.current.Price,
      MinTopUp: initialRef.current.MinTopUp,
      AmountOptions: initialRef.current.AmountOptions.trim(),
      AmountDiscount: initialRef.current.AmountDiscount.trim(),
      SePayEnabled: initialRef.current.SePayEnabled,
      SePayBankAccountNumber: initialRef.current.SePayBankAccountNumber.trim(),
      SePayBankCode: initialRef.current.SePayBankCode.trim(),
      SePayAccountHolder: initialRef.current.SePayAccountHolder.trim(),
      SePayMinTopUp: initialRef.current.SePayMinTopUp,
      SePayOrderExpiryMinutes: initialRef.current.SePayOrderExpiryMinutes,
    }

    const updates: Array<{ key: string; value: string | number | boolean }> = []

    if (sanitized.Price !== initial.Price) {
      updates.push({ key: 'Price', value: sanitized.Price })
    }
    if (sanitized.MinTopUp !== initial.MinTopUp) {
      updates.push({ key: 'MinTopUp', value: sanitized.MinTopUp })
    }
    if (
      normalizeJsonForComparison(sanitized.AmountOptions) !==
      normalizeJsonForComparison(initial.AmountOptions)
    ) {
      updates.push({ key: 'payment_setting.amount_options', value: sanitized.AmountOptions })
    }
    if (
      normalizeJsonForComparison(sanitized.AmountDiscount) !==
      normalizeJsonForComparison(initial.AmountDiscount)
    ) {
      updates.push({ key: 'payment_setting.amount_discount', value: sanitized.AmountDiscount })
    }
    if (sanitized.SePayEnabled !== initial.SePayEnabled) {
      updates.push({ key: 'SePayEnabled', value: sanitized.SePayEnabled })
    }
    if (sanitized.SePayBankAccountNumber !== initial.SePayBankAccountNumber) {
      updates.push({ key: 'SePayBankAccountNumber', value: sanitized.SePayBankAccountNumber })
    }
    if (sanitized.SePayBankCode !== initial.SePayBankCode) {
      updates.push({ key: 'SePayBankCode', value: sanitized.SePayBankCode })
    }
    if (sanitized.SePayAccountHolder !== initial.SePayAccountHolder) {
      updates.push({ key: 'SePayAccountHolder', value: sanitized.SePayAccountHolder })
    }
    if (sanitized.SePayMinTopUp !== initial.SePayMinTopUp) {
      updates.push({ key: 'SePayMinTopUp', value: sanitized.SePayMinTopUp })
    }
    if (sanitized.SePayOrderExpiryMinutes !== initial.SePayOrderExpiryMinutes) {
      updates.push({ key: 'SePayOrderExpiryMinutes', value: sanitized.SePayOrderExpiryMinutes })
    }
    if (sanitized.SePayWebhookApiKey) {
      updates.push({ key: 'SePayWebhookApiKey', value: sanitized.SePayWebhookApiKey })
    }

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }
  }

  return (
    <SettingsSection title={t('Payment Gateway')}>
      {!complianceConfirmed ? (
        <Alert variant='destructive' className='mb-6'>
          <ShieldAlert className='h-4 w-4' />
          <AlertTitle>{t('Compliance confirmation required')}</AlertTitle>
          <AlertDescription>
            <div className='space-y-3'>
              <p>
                {t(
                  'Payment, redemption codes, subscription plans, and invitation rewards are locked until the root administrator confirms the compliance terms.'
                )}
              </p>
              <ol className='list-decimal space-y-1 pl-5'>
                {complianceStatements.map((statement) => (
                  <li key={statement}>{statement}</li>
                ))}
              </ol>
            </div>
          </AlertDescription>
          <AlertAction>
            <Button
              type='button'
              size='sm'
              variant='destructive'
              onClick={() => setShowComplianceDialog(true)}
            >
              {t('Confirm compliance')}
            </Button>
          </AlertAction>
        </Alert>
      ) : (
        <Alert className='mb-6'>
          <AlertTitle>{t('Compliance confirmed')}</AlertTitle>
          <AlertDescription>
            {t('Confirmed at {{time}} by user #{{userId}}', {
              time: complianceDefaults.confirmedAt
                ? new Date(complianceDefaults.confirmedAt * 1000).toLocaleString()
                : '-',
              userId: complianceDefaults.confirmedBy || '-',
            })}
          </AlertDescription>
        </Alert>
      )}

      <RiskAcknowledgementDialog
        open={showComplianceDialog}
        onOpenChange={setShowComplianceDialog}
        title={t('Confirm compliance terms')}
        description={t(
          'This confirmation unlocks payment, redemption code, subscription plan, and invitation reward features. Please read the statements carefully.'
        )}
        items={complianceStatements}
        requiredText={complianceRequiredText}
        requiredTextParts={complianceRequiredTextParts}
        inputPrompt={t('Please type the following text to confirm:')}
        inputPlaceholder={t('Type the confirmation text here')}
        mismatchHint={t('The entered text does not match the required text.')}
        confirmText={t('Confirm and enable')}
        isLoading={confirmComplianceMutation.isPending}
        onConfirm={() => confirmComplianceMutation.mutate()}
      />

      <Form {...form}>
        <SettingsForm
          onSubmit={form.handleSubmit(onSubmit)}
          className={cn('gap-y-8', !complianceConfirmed && 'pointer-events-none opacity-40')}
          data-no-autosubmit='true'
        >
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending || isSubmitting}
            saveLabel='Save all settings'
          />
          <Tabs defaultValue='general' className='min-w-0'>
            <div className='overflow-x-auto pb-1'>
              <TabsList className='grid min-w-[28rem] grid-cols-2'>
                <TabsTrigger value='general'>{t('General')}</TabsTrigger>
                <TabsTrigger value='sepay'>SePay</TabsTrigger>
              </TabsList>
            </div>

            <TabsContent value='general' className={paymentTabContentClassName}>
              <div className='space-y-4'>
                <div>
                  <h3 className='text-lg font-medium'>{t('General Settings')}</h3>
                  <p className='text-muted-foreground text-sm'>
                    {t('Shared configuration for SePay orders')}
                  </p>
                </div>

                <div className='grid gap-6 md:grid-cols-2'>
                  <FormField
                    control={form.control}
                    name='Price'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Price (VND per USD)')}</FormLabel>
                        <FormControl>
                          <Input
                            type='number'
                            step='0.01'
                            min={0}
                            {...safeNumberFieldProps(field)}
                          />
                        </FormControl>
                        <FormDescription>
                          {t('How much Vietnamese Dong to charge for each USD of credited balance')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name='MinTopUp'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Minimum top-up (USD)')}</FormLabel>
                        <FormControl>
                          <Input
                            type='number'
                            step='0.01'
                            min={0}
                            {...safeNumberFieldProps(field)}
                          />
                        </FormControl>
                        <FormDescription>
                          {t('Smallest USD amount users can recharge')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>

                <div className='grid gap-6 md:grid-cols-2 md:items-start'>
                  <FormField
                    control={form.control}
                    name='AmountOptions'
                    render={({ field }) => (
                      <FormItem>
                        <div className='mb-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
                          <FormLabel>{t('Top-up amount options')}</FormLabel>
                          <Button
                            type='button'
                            variant='outline'
                            size='sm'
                            onClick={() => setAmountOptionsVisualMode(!amountOptionsVisualMode)}
                            className='w-full sm:w-auto'
                          >
                            {amountOptionsVisualMode ? (
                              <>
                                <Code2 className='mr-2 h-3 w-3' />
                                {t('JSON Editor')}
                              </>
                            ) : (
                              <>
                                <Eye className='mr-2 h-3 w-3' />
                                {t('Visual Editor')}
                              </>
                            )}
                          </Button>
                        </div>
                        <FormControl>
                          {amountOptionsVisualMode ? (
                            <AmountOptionsVisualEditor
                              value={field.value}
                              onChange={field.onChange}
                            />
                          ) : (
                            <JsonCodeEditor
                              value={field.value}
                              onChange={field.onChange}
                              name={field.name}
                              onBlur={field.onBlur}
                              textareaRef={field.ref}
                              placeholder='[10, 20, 50, 100]'
                              heightClassName='h-40 min-h-40 max-h-40'
                              aria-invalid={Boolean(form.formState.errors.AmountOptions)}
                            />
                          )}
                        </FormControl>
                        <FormDescription>
                          {t('Preset recharge amounts (JSON array)')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name='AmountDiscount'
                    render={({ field }) => (
                      <FormItem>
                        <div className='mb-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
                          <FormLabel>{t('Amount discount')}</FormLabel>
                          <Button
                            type='button'
                            variant='outline'
                            size='sm'
                            onClick={() => setAmountDiscountVisualMode(!amountDiscountVisualMode)}
                            className='w-full sm:w-auto'
                          >
                            {amountDiscountVisualMode ? (
                              <>
                                <Code2 className='mr-2 h-3 w-3' />
                                {t('JSON Editor')}
                              </>
                            ) : (
                              <>
                                <Eye className='mr-2 h-3 w-3' />
                                {t('Visual Editor')}
                              </>
                            )}
                          </Button>
                        </div>
                        <FormControl>
                          {amountDiscountVisualMode ? (
                            <AmountDiscountVisualEditor
                              value={field.value}
                              onChange={field.onChange}
                            />
                          ) : (
                            <JsonCodeEditor
                              value={field.value}
                              onChange={field.onChange}
                              name={field.name}
                              onBlur={field.onBlur}
                              textareaRef={field.ref}
                              placeholder='{"100":0.95,"200":0.9}'
                              heightClassName='h-40 min-h-40 max-h-40'
                              aria-invalid={Boolean(form.formState.errors.AmountDiscount)}
                            />
                          )}
                        </FormControl>
                        <FormDescription>
                          {t('Discount map by recharge amount (JSON object)')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </div>
            </TabsContent>

            <TabsContent value='sepay' className={paymentTabContentClassName}>
              <div className='space-y-4'>
                <div>
                  <h3 className='text-lg font-medium'>{t('SePay Gateway')}</h3>
                  <p className='text-muted-foreground text-sm'>
                    {t('Vietnamese domestic bank transfer with VietQR payment codes')}
                  </p>
                </div>

                <FormField
                  control={form.control}
                  name='SePayEnabled'
                  render={({ field }) => (
                    <SettingsSwitchItem>
                      <SettingsSwitchContent>
                        <FormLabel>{t('Enable SePay')}</FormLabel>
                        <FormDescription>
                          {t('Enable bank-transfer top-up and subscription payments via SePay')}
                        </FormDescription>
                      </SettingsSwitchContent>
                      <FormControl>
                        <Switch checked={field.value} onCheckedChange={field.onChange} />
                      </FormControl>
                    </SettingsSwitchItem>
                  )}
                />

                <div className='grid gap-6 md:grid-cols-2'>
                  <FormField
                    control={form.control}
                    name='SePayBankAccountNumber'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Bank account number')}</FormLabel>
                        <FormControl>
                          <Input placeholder='1234567890' {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name='SePayBankCode'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Bank code')}</FormLabel>
                        <FormControl>
                          <Input placeholder='VCB' {...field} />
                        </FormControl>
                        <FormDescription>
                          {t('Used to build the VietQR image')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>

                <FormField
                  control={form.control}
                  name='SePayAccountHolder'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Account holder')}</FormLabel>
                      <FormControl>
                        <Input placeholder={t('Company or account name')} {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <div className='grid gap-6 md:grid-cols-2'>
                  <FormField
                    control={form.control}
                    name='SePayWebhookApiKey'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Webhook API key')}</FormLabel>
                        <FormControl>
                          <Input
                            type='password'
                            autoComplete='new-password'
                            placeholder={t('Enter new key to update')}
                            {...field}
                            onChange={(event) => field.onChange(event.target.value)}
                          />
                        </FormControl>
                        <FormDescription>
                          {t('Set this value in your SePay dashboard. Leave blank unless rotating the key.')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name='SePayMinTopUp'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('SePay minimum top-up (USD)')}</FormLabel>
                        <FormControl>
                          <Input
                            type='number'
                            step='0.01'
                            min={0}
                            {...safeNumberFieldProps(field)}
                          />
                        </FormControl>
                        <FormDescription>{t('Leave 0 to fall back to global minimum')}</FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>

                <FormField
                  control={form.control}
                  name='SePayOrderExpiryMinutes'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Order expiry (minutes)')}</FormLabel>
                      <FormControl>
                        <Input
                          type='number'
                          step='1'
                          min={SEPAY_EXPIRY_MIN}
                          max={SEPAY_EXPIRY_MAX}
                          {...safeNumberFieldProps(field)}
                        />
                      </FormControl>
                      <FormDescription>
                        {t('1–10080 minutes (up to 7 days)')}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <Alert>
                  <ShieldAlert className='h-4 w-4' />
                  <AlertTitle>{t('Webhook configuration')}</AlertTitle>
                  <AlertDescription>
                    <ul className='list-inside list-disc space-y-1'>
                      <li>
                        {t('Webhook URL:')}{' '}
                        <code className='rounded bg-muted px-1 py-0.5 text-xs'>{'<ServerAddress>/api/sepay/webhook'}</code>
                      </li>
                      <li>
                        {t('Header:')}{' '}
                        <code className='rounded bg-muted px-1 py-0.5 text-xs'>Authorization: Apikey {'<key>'}</code>
                      </li>
                    </ul>
                  </AlertDescription>
                </Alert>
              </div>
            </TabsContent>
          </Tabs>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
