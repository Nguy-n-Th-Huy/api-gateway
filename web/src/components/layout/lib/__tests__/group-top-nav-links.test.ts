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
import type { TFunction } from 'i18next'
import { describe, expect, test } from 'vitest'

import type { TopNavLink } from '../../types'
import { groupTopNavLinks } from '../group-top-nav-links'

// Identity translator: the module only needs a callable `t`, never a real
// i18next instance, which keeps this test free of React and the DOM.
const t = ((key: string) => key) as TFunction

const HOME: TopNavLink = { title: 'Home', href: '/' }
const CONSOLE: TopNavLink = { title: 'Console', href: '/dashboard' }
const PRICING: TopNavLink = { title: 'Model Square', href: '/pricing' }
const RANKINGS: TopNavLink = { title: 'Rankings', href: '/rankings' }
const DOCS: TopNavLink = { title: 'Docs', href: '/docs' }
const ABOUT: TopNavLink = { title: 'About', href: '/about' }

describe('groupTopNavLinks', () => {
  test('all modules enabled arranges four entries: home, models group, resources group, console', () => {
    const entries = groupTopNavLinks(
      [HOME, CONSOLE, PRICING, RANKINGS, DOCS, ABOUT],
      t
    )

    expect(entries).toEqual([
      { kind: 'link', ...HOME },
      {
        kind: 'group',
        id: 'models',
        label: 'Models',
        highlight: {
          title: 'Explore the full catalogue',
          description: 'One endpoint, one key. Prices shown in your currency.',
          linkLabel: 'View Pricing',
          href: '/pricing',
        },
        children: [
          { ...PRICING, description: 'The full catalogue with prices' },
          { ...RANKINGS, description: 'Which models are used most' },
        ],
      },
      {
        kind: 'group',
        id: 'resources',
        label: 'Resources',
        highlight: undefined,
        children: [
          { ...DOCS, description: 'Guides and API reference' },
          { ...ABOUT, description: 'Who we are and what we offer' },
        ],
      },
      { kind: 'link', ...CONSOLE },
    ])
  })

  test('both models children disabled drops the models group entirely, leaving no empty panel', () => {
    const entries = groupTopNavLinks([HOME, CONSOLE, DOCS, ABOUT], t)

    expect(entries.some((entry) => entry.kind === 'group' && entry.id === 'models')).toBe(
      false
    )
    expect(entries).toEqual([
      { kind: 'link', ...HOME },
      {
        kind: 'group',
        id: 'resources',
        label: 'Resources',
        highlight: undefined,
        children: [
          { ...DOCS, description: 'Guides and API reference' },
          { ...ABOUT, description: 'Who we are and what we offer' },
        ],
      },
      { kind: 'link', ...CONSOLE },
    ])
  })

  test('only one models child enabled renders a plain link, not a group', () => {
    const entries = groupTopNavLinks([HOME, CONSOLE, RANKINGS, DOCS, ABOUT], t)

    const modelsEntry = entries[1]
    expect(modelsEntry).toEqual({ kind: 'link', ...RANKINGS })
    expect(modelsEntry.kind).toBe('link')
  })

  test('both resources children disabled drops the resources group entirely', () => {
    const entries = groupTopNavLinks([HOME, CONSOLE, PRICING, RANKINGS], t)

    expect(
      entries.some((entry) => entry.kind === 'group' && entry.id === 'resources')
    ).toBe(false)
    expect(entries).toEqual([
      { kind: 'link', ...HOME },
      {
        kind: 'group',
        id: 'models',
        label: 'Models',
        highlight: {
          title: 'Explore the full catalogue',
          description: 'One endpoint, one key. Prices shown in your currency.',
          linkLabel: 'View Pricing',
          href: '/pricing',
        },
        children: [
          { ...PRICING, description: 'The full catalogue with prices' },
          { ...RANKINGS, description: 'Which models are used most' },
        ],
      },
      { kind: 'link', ...CONSOLE },
    ])
  })

  test('only one resources child enabled renders a plain link, not a group', () => {
    const entries = groupTopNavLinks([HOME, CONSOLE, PRICING, RANKINGS, ABOUT], t)

    const resourcesEntry = entries[2]
    expect(resourcesEntry).toEqual({ kind: 'link', ...ABOUT })
    expect(resourcesEntry.kind).toBe('link')
  })

  test('an external docs link is still matched into the resources group by its external flag, not by href', () => {
    const externalDocs: TopNavLink = {
      title: 'Docs',
      href: 'https://docs.example.com',
      external: true,
    }
    const entries = groupTopNavLinks(
      [HOME, CONSOLE, PRICING, RANKINGS, externalDocs, ABOUT],
      t
    )

    const resourcesEntry = entries.find(
      (entry): entry is Extract<typeof entry, { kind: 'group' }> =>
        entry.kind === 'group' && entry.id === 'resources'
    )
    expect(resourcesEntry).toBeDefined()
    expect(resourcesEntry?.children).toEqual([
      { ...externalDocs, description: 'Guides and API reference' },
      { ...ABOUT, description: 'Who we are and what we offer' },
    ])
  })
})
