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
import { describe, expect, test, vi } from 'vitest'

import { Hero } from '../hero'

// @lobehub/icons transitively imports @lobehub/fluent-emoji, which ships a
// directory import Node's ESM resolver (used by Vitest) cannot follow.
// Stub the package so the "Supported Applications" pill row can mount —
// mirrors the pattern in features/task-plugins/__tests__/plugin-card.test.tsx.
vi.mock('@lobehub/icons', () => ({
  CherryStudio: { Color: () => null },
}))

// Hero renders TanStack <Link>; provide a minimal stand-in so it can mount
// without a live router context (pattern established in
// components/layout/__tests__/shell-responsive.test.tsx).
vi.mock('@tanstack/react-router', () => ({
  Link: ({
    children,
    to,
    ...props
  }: {
    children?: React.ReactNode
    to: string
  } & React.AnchorHTMLAttributes<HTMLAnchorElement>) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
}))

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({ status: null, loading: false, error: null }),
}))

// web-home-page spec — "Hero states the offer and the primary actions": a
// sign-up path is offered when signed out, and it is replaced by a console
// path when signed in, while docs and in-page pricing stay available either way.

describe('home page hero actions', () => {
  test('offers a sign-up action for a signed-out visitor', () => {
    render(<Hero isAuthenticated={false} />)
    expect(
      screen.getByRole('button', { name: /Get Started/i })
    ).toHaveAttribute('href', '/sign-up')
    expect(
      screen.queryByRole('button', { name: /Go to Dashboard/i })
    ).not.toBeInTheDocument()
  })

  test('replaces the sign-up action with a console action for a signed-in visitor', () => {
    render(<Hero isAuthenticated />)
    expect(
      screen.getByRole('button', { name: /Go to Dashboard/i })
    ).toHaveAttribute('href', '/dashboard')
    expect(
      screen.queryByRole('button', { name: /Get Started/i })
    ).not.toBeInTheDocument()
  })

  test('offers an in-page path to pricing regardless of auth state', () => {
    const { rerender } = render(<Hero isAuthenticated={false} />)
    expect(
      screen.getByRole('button', { name: 'View Pricing' })
    ).toHaveAttribute('href', '#pricing')

    rerender(<Hero isAuthenticated />)
    expect(
      screen.getByRole('button', { name: 'View Pricing' })
    ).toHaveAttribute('href', '#pricing')
  })
})
