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
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { NavLinkItem } from '../components/nav-link-item'
import { TopNav } from '../components/top-nav'
import type { TopNavLink } from '../types'

const link = (overrides?: Partial<TopNavLink>): TopNavLink => ({
  title: 'Pricing',
  href: '/pricing',
  ...overrides,
})

describe('NavLinkItem', () => {
  test('renders an anchor element that is reachable by keyboard', async () => {
    const user = userEvent.setup()
    render(<NavLinkItem link={link({ external: true })} />)
    await user.tab()
    expect(screen.getByRole('link', { name: 'Pricing' })).toHaveFocus()
  })

  test('disabled link exposes aria-disabled to assistive technology', () => {
    render(<NavLinkItem link={link({ external: true, disabled: true })} />)
    expect(screen.getByRole('link', { name: 'Pricing' })).toHaveAttribute(
      'aria-disabled',
      'true'
    )
  })

  test('active link announces itself via aria-current', () => {
    render(<NavLinkItem link={link({ external: true, isActive: true })} />)
    expect(screen.getByRole('link', { name: 'Pricing' })).toHaveAttribute(
      'aria-current',
      'page'
    )
  })
})

describe('TopNav', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  test('active link announces itself via aria-current in desktop nav', () => {
    render(
      <nav>
        <TopNav links={[link({ isActive: true, external: true })]} />
      </nav>
    )
    const active = screen
      .getAllByRole('link')
      .find((l) => l.textContent === 'Pricing')
    expect(active).toHaveAttribute('aria-current', 'page')
  })
})
