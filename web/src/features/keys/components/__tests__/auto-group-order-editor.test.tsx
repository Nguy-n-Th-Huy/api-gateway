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
import { fireEvent, render, within } from '@testing-library/react'
import { describe, expect, test } from 'vitest'

// The imports stay dynamic to match the sibling tests in this directory: the
// component is loaded after this file's own i18next instance is created, so the
// rendered copy comes from the resources declared below.
const { useState } = await import('react')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { AutoGroupOrderEditor, GlobalAutoOrderPreview } =
  await import('../auto-group-order-editor')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        '{{count}} / {{max}} groups selected':
          '{{count}} / {{max}} groups selected',
        'Add group': 'Add group',
        'Drag {{group}} to reorder': 'Drag {{group}} to reorder',
        Groups: 'Groups',
        'Inherit global Auto order': 'Inherit global Auto order',
        'Maximum {{max}} groups selected': 'Maximum {{max}} groups selected',
        'Move {{group}} down': 'Move {{group}} down',
        'Move {{group}} up': 'Move {{group}} up',
        'No group found.': 'No group found.',
        'No groups selected': 'No groups selected',
        'No available groups in the global Auto order.':
          'No available groups in the global Auto order.',
        'Remove {{group}}': 'Remove {{group}}',
        Ratio: 'Ratio',
        "Saving with no groups leaves the key on its owner's group.":
          "Saving with no groups leaves the key on its owner's group.",
        'Search...': 'Search...',
        'Select a group': 'Select a group',
        'Using the complete global Auto order ({{count}} groups)':
          'Using the complete global Auto order ({{count}} groups)',
      },
    },
  },
})

const globalOptions = [
  { value: 'vip', label: 'VIP', desc: 'Priority access', ratio: 3 },
  { value: 'default', label: 'Default', desc: 'Standard access', ratio: 1 },
  { value: 'team', label: 'Team', desc: 'Shared access', ratio: 2 },
]

function Harness(props: { initialGroups?: string[] }) {
  const [groups, setGroups] = useState(
    props.initialGroups ?? ['default', 'vip']
  )

  return (
    <I18nextProvider i18n={i18n}>
      <AutoGroupOrderEditor
        value={groups}
        options={[
          { value: 'auto', label: 'auto' },
          { value: 'default', label: 'default', ratio: 1 },
          { value: 'vip', label: 'vip', ratio: 2 },
          { value: 'team', label: 'team', ratio: 3 },
        ]}
        maxCount={2}
        onChange={setGroups}
      />
      <output data-testid='order'>{groups.join(',')}</output>
    </I18nextProvider>
  )
}

function PreviewHarness(props: { globalOptions?: typeof globalOptions }) {
  return (
    <I18nextProvider i18n={i18n}>
      <GlobalAutoOrderPreview
        globalOptions={props.globalOptions ?? globalOptions}
      />
    </I18nextProvider>
  )
}

function findButton(container: HTMLElement, label: string): HTMLButtonElement {
  return within(container).getByRole('button', { name: label })
}

function getCommandItem(label: string): HTMLElement {
  const item = [
    ...document.querySelectorAll<HTMLElement>('[data-slot="command-item"]'),
  ].find((candidate) => candidate.textContent?.includes(label))
  if (!item) {
    throw new Error(`Expected command item containing "${label}"`)
  }
  return item
}

function getCommandItems(): string[] {
  return [
    ...document.querySelectorAll<HTMLElement>('[data-slot="command-item"]'),
  ].map((item) => item.textContent ?? '')
}

describe('Auto group order editor', () => {
  test('enforces the limit and exposes accessible reorder controls', () => {
    const { container } = render(<Harness />)

    const addButton = within(container).getByRole('combobox')
    expect(addButton).toBeDisabled()
    expect(container).toHaveTextContent('2 / 2 groups selected')
    expect(
      within(container).getByRole('group', { name: 'Groups' })
    ).toBeInTheDocument()
    expect(findButton(container, 'Drag default to reorder').type).toBe('button')

    fireEvent.click(findButton(container, 'Move default down'))
    expect(within(container).getByTestId('order')).toHaveTextContent(
      'vip,default'
    )

    fireEvent.keyDown(findButton(container, 'Drag vip to reorder'), {
      key: 'ArrowDown',
    })
    expect(within(container).getByTestId('order')).toHaveTextContent(
      'default,vip'
    )
  })

  test('adds a group and offers only the candidates that are not selected yet', () => {
    const { container } = render(<Harness initialGroups={['default']} />)

    const addButton = within(container).getByRole('combobox')
    expect(addButton).toBeEnabled()
    expect(container).toHaveTextContent('1 / 2 groups selected')

    fireEvent.click(addButton)
    const candidates = getCommandItems()
    expect(candidates.some((item) => item.includes('team'))).toBe(true)
    expect(candidates.some((item) => item.includes('default'))).toBe(false)
    expect(candidates.some((item) => item.includes('auto'))).toBe(false)

    fireEvent.click(getCommandItem('team'))
    expect(within(container).getByTestId('order')).toHaveTextContent(
      'default,team'
    )
    expect(addButton).toBeDisabled()
  })

  test('removes a group and appends the new candidate in the chosen order', () => {
    const { container } = render(<Harness />)

    fireEvent.click(findButton(container, 'Remove vip'))
    expect(within(container).getByTestId('order')).toHaveTextContent('default')

    const addButton = within(container).getByRole('combobox')
    fireEvent.click(addButton)
    fireEvent.click(getCommandItem('team'))
    expect(within(container).getByTestId('order')).toHaveTextContent(
      'default,team'
    )
  })

  test('states what an empty selection means instead of falling back to the global order', () => {
    const { container } = render(<Harness initialGroups={['default']} />)

    fireEvent.click(findButton(container, 'Remove default'))

    expect(within(container).getByTestId('order')).toBeEmptyDOMElement()
    expect(container).toHaveTextContent('0 / 2 groups selected')
    expect(container).toHaveTextContent('No groups selected')
    expect(container).toHaveTextContent(
      "Saving with no groups leaves the key on its owner's group."
    )
    expect(container.querySelector('[data-slot="global-auto-order"]')).toBe(
      null
    )
    expect(container.querySelector('[aria-label^="Remove "]')).toBe(null)
  })
})

describe('Global Auto order preview', () => {
  test('renders the complete global order with metadata and without reorder controls', () => {
    const { container } = render(<PreviewHarness />)

    expect(container).toHaveTextContent(
      'Using the complete global Auto order (3 groups)'
    )
    expect(container).not.toHaveTextContent('groups selected')

    const order = container.querySelector<HTMLOListElement>(
      '[data-slot="global-auto-order"]'
    )
    if (!order) {
      throw new Error('Expected the global Auto group order')
    }
    expect(order).toHaveClass('overflow-y-auto', 'flex-wrap')

    const items = [...order.querySelectorAll('li')]
    expect(items.length).toBe(3)
    expect(
      order.querySelectorAll('[data-slot="global-auto-order-connector"]').length
    ).toBe(2)
    expect(
      items.map((item) => ({
        index: item.querySelector('[data-slot="global-auto-order-index"]')
          ?.textContent,
        name: item.querySelector('[data-slot="global-auto-order-name"]')
          ?.textContent,
        title: item
          .querySelector('[data-slot="global-auto-order-chip"]')
          ?.getAttribute('title'),
        description: item.querySelector(
          '[data-slot="global-auto-order-description"]'
        )?.textContent,
        ratio: item.querySelector('[data-slot="badge"]')?.textContent,
      }))
    ).toEqual([
      {
        index: '1',
        name: 'VIP',
        title: 'Priority access',
        description: 'Priority access',
        ratio: '3x Ratio',
      },
      {
        index: '2',
        name: 'Default',
        title: 'Standard access',
        description: 'Standard access',
        ratio: '1x Ratio',
      },
      {
        index: '3',
        name: 'Team',
        title: 'Shared access',
        description: 'Shared access',
        ratio: '2x Ratio',
      },
    ])

    for (const item of items) {
      const chip = item.querySelector('[data-slot="global-auto-order-chip"]')
      expect(chip).toBeInTheDocument()
      const description = item.querySelector(
        '[data-slot="global-auto-order-description"]'
      )
      expect(description).toHaveClass('sr-only')
    }

    expect(
      items[0]?.querySelector('[data-slot="global-auto-order-connector"]')
    ).toBe(null)
    for (const item of items.slice(1)) {
      const connector = item.querySelector(
        '[data-slot="global-auto-order-connector"]'
      )
      expect(connector).toHaveAttribute('aria-hidden', 'true')
    }

    expect(container.querySelector('[aria-label^="Drag "]')).toBe(null)
    expect(container.querySelector('[aria-label^="Move "]')).toBe(null)
    expect(container.querySelector('[aria-label^="Remove "]')).toBe(null)
  })

  test('shows an explicit empty state when the global order has no selectable group', () => {
    const { container } = render(<PreviewHarness globalOptions={[]} />)

    expect(container).toHaveTextContent(
      'Using the complete global Auto order (0 groups)'
    )
    expect(container).toHaveTextContent(
      'No available groups in the global Auto order.'
    )
    expect(container.querySelector('[data-slot="global-auto-order"]')).toBe(
      null
    )
    expect(
      [
        ...container.querySelectorAll<HTMLElement>(
          '[data-slot="global-auto-order-name"]'
        ),
      ].map((chip) => chip.textContent)
    ).toEqual([])
  })
})
