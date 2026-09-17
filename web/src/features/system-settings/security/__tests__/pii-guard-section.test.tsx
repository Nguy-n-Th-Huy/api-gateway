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
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ComponentProps } from 'react'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { PiiGuardSection } from '../pii-guard-section'

// PiiGuardSection persists through a react-query mutation; that is not the
// behavior under test, so the hook is stubbed at the same module boundary the
// OAuth section test uses.
const mockMutateAsync = vi.fn()
vi.mock('../../hooks/use-update-option', () => ({
  useUpdateOption: () => ({
    mutateAsync: (...args: unknown[]) => mockMutateAsync(...args),
    isPending: false,
  }),
}))

type PiiGuardDefaults = ComponentProps<typeof PiiGuardSection>['defaultValues']

function createDefaults(
  overrides: Partial<PiiGuardDefaults> = {}
): PiiGuardDefaults {
  return {
    'piiguard.enabled': true,
    'piiguard.mask_request': true,
    'piiguard.unmask_response': true,
    'piiguard.mode': 'pseudonym',
    'piiguard.placeholder_style': 'typed',
    'piiguard.token_template': '«{{type}}_{{index}}»',
    'piiguard.secret': '',
    'piiguard.max_body_bytes': 1048576,
    'piiguard.enabled_entity_types': [],
    'piiguard.disabled_entity_types': [],
    'piiguard.custom_keywords': [],
    'piiguard.min_keyword_length': 3,
    'piiguard.fakes': '',
    'piiguard.require_mask_reject': false,
    ...overrides,
  }
}

function submitForm(container: HTMLElement) {
  const form = container.querySelector('form')
  if (!form) throw new Error('Expected the settings form to be present')
  fireEvent.submit(form)
}

function getSwitch(name: string): HTMLElement {
  return screen.getByRole('switch', { name })
}

function getEntityCheckbox(groupName: string, label: string): HTMLElement {
  const group = screen.getByRole('group', { name: groupName })
  return within(group).getByRole('checkbox', { name: new RegExp(label) })
}

async function selectOption(triggerName: string, optionName: string) {
  const user = userEvent.setup()
  await user.click(screen.getByRole('combobox', { name: triggerName }))
  await user.click(await screen.findByRole('option', { name: optionName }))
}

function sentRequests(): Array<{ key: string; value: unknown }> {
  return mockMutateAsync.mock.calls.map(
    ([request]) => request as { key: string; value: unknown }
  )
}

describe('PiiGuardSection', () => {
  afterEach(() => {
    mockMutateAsync.mockReset()
  })

  test('disables every dependent control while the guard is off', () => {
    render(<PiiGuardSection defaultValues={createDefaults({
      'piiguard.enabled': false,
    })} />)

    for (const name of [
      'Mask request bodies',
      'Restore values in responses',
      'Reject bodies that cannot be masked',
    ]) {
      expect(getSwitch(name)).toHaveAttribute('aria-disabled', 'true')
    }

    expect(
      screen.getByRole('combobox', { name: 'Masking mode' })
    ).toBeDisabled()
    expect(screen.getByLabelText('Maximum body size (bytes)')).toBeDisabled()
    expect(screen.getByLabelText('Minimum keyword length')).toBeDisabled()
    expect(screen.getByLabelText('Seed secret')).toBeDisabled()
    expect(screen.getByLabelText('Custom keywords')).toBeDisabled()
    expect(
      screen.getByLabelText('Replacement overrides (JSON)')
    ).toBeDisabled()

    for (const groupName of ['Allowed entity types', 'Disabled entity types']) {
      expect(
        getEntityCheckbox(groupName, 'Email addresses')
      ).toHaveAttribute('aria-disabled', 'true')
    }

    // The master switch itself stays editable, so the guard can be turned on.
    expect(getSwitch('Enable PII Guard')).not.toHaveAttribute(
      'aria-disabled',
      'true'
    )
  })

  test('disables the placeholder style and template outside redact mode', () => {
    render(<PiiGuardSection defaultValues={createDefaults()} />)

    expect(
      screen.getByRole('combobox', { name: 'Placeholder style' })
    ).toBeDisabled()
    expect(screen.getByLabelText('Placeholder template')).toBeDisabled()
  })

  test('clears and disables response restoration when redact mode is selected', async () => {
    mockMutateAsync.mockResolvedValue({ success: true })
    const { container } = render(
      <PiiGuardSection defaultValues={createDefaults()} />
    )

    expect(getSwitch('Restore values in responses')).not.toHaveAttribute(
      'aria-disabled',
      'true'
    )

    await selectOption('Masking mode', 'Redact (typed placeholder)')

    const unmaskSwitch = getSwitch('Restore values in responses')
    expect(unmaskSwitch).toHaveAttribute('aria-checked', 'false')
    expect(unmaskSwitch).toHaveAttribute('aria-disabled', 'true')

    submitForm(container)

    await waitFor(() => {
      expect(mockMutateAsync).toHaveBeenCalledWith({
        key: 'piiguard.mode',
        value: 'redact',
      })
    })

    // The backend rejects redact mode while restoration is still on, so
    // clearing restoration has to be stored first.
    const keys = sentRequests().map((request) => request.key)
    expect(keys.indexOf('piiguard.unmask_response')).toBeLessThan(
      keys.indexOf('piiguard.mode')
    )
  })

  test('saves entity lists in detection order, keywords per line and overrides as JSON', async () => {
    mockMutateAsync.mockResolvedValue({ success: true })
    const { container } = render(
      <PiiGuardSection defaultValues={createDefaults()} />
    )

    // Checked out of order on purpose: the stored list keeps the detector's
    // identifier order so an unchanged form never looks modified.
    fireEvent.click(getEntityCheckbox('Allowed entity types', 'URLs'))
    fireEvent.click(
      getEntityCheckbox('Allowed entity types', 'Email addresses')
    )
    fireEvent.input(screen.getByLabelText('Custom keywords'), {
      target: { value: 'tax-123\n\n  acme-corp  ' },
    })
    fireEvent.input(screen.getByLabelText('Replacement overrides (JSON)'), {
      target: { value: '{"EMAIL": {"alex@example.com": "fake@example.com"}}' },
    })

    submitForm(container)

    await waitFor(() => {
      expect(mockMutateAsync).toHaveBeenCalledWith({
        key: 'piiguard.enabled_entity_types',
        value: '["EMAIL","URL"]',
      })
    })
    expect(mockMutateAsync).toHaveBeenCalledWith({
      key: 'piiguard.custom_keywords',
      value: '["tax-123","acme-corp"]',
    })
    expect(mockMutateAsync).toHaveBeenCalledWith({
      key: 'piiguard.fakes',
      value: '{"EMAIL": {"alex@example.com": "fake@example.com"}}',
    })
    expect(sentRequests().map((request) => request.key)).not.toContain(
      'piiguard.max_body_bytes'
    )
  })

  test('saves the seed secret only when the admin types a new one', async () => {
    mockMutateAsync.mockResolvedValue({ success: true })
    const { container } = render(
      <PiiGuardSection defaultValues={createDefaults()} />
    )

    fireEvent.input(screen.getByLabelText('Seed secret'), {
      target: { value: 'new-seed' },
    })

    submitForm(container)

    await waitFor(() => {
      expect(mockMutateAsync).toHaveBeenCalledWith({
        key: 'piiguard.secret',
        value: 'new-seed',
      })
    })
  })

  test('leaves the stored seed untouched while the seed field stays empty', async () => {
    mockMutateAsync.mockResolvedValue({ success: true })
    const { container } = render(
      <PiiGuardSection defaultValues={createDefaults()} />
    )

    // Save an unrelated change: the write-only seed must not be sent, because
    // an empty value would reset the seed the guard already uses.
    fireEvent.input(screen.getByLabelText('Maximum body size (bytes)'), {
      target: { value: '2048' },
    })

    submitForm(container)

    await waitFor(() => {
      expect(mockMutateAsync).toHaveBeenCalledWith({
        key: 'piiguard.max_body_bytes',
        value: 2048,
      })
    })
    expect(sentRequests().map((request) => request.key)).not.toContain(
      'piiguard.secret'
    )
  })

  test('blocks enabling the guard with masking and restoration both off', async () => {
    const { container } = render(
      <PiiGuardSection defaultValues={createDefaults()} />
    )

    fireEvent.click(getSwitch('Mask request bodies'))
    fireEvent.click(getSwitch('Restore values in responses'))

    submitForm(container)

    expect(
      await screen.findByText(
        'An enabled guard must mask requests or restore responses'
      )
    ).toBeInTheDocument()
    expect(mockMutateAsync).not.toHaveBeenCalled()
  })
})
