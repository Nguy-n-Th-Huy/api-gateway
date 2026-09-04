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
import { beforeAll, describe, expect, test, vi } from 'vitest'

import { Faq } from '../faq'

// AnimateInView (used inside Faq) observes intersection on mount; the shared
// test-setup.ts only stubs ResizeObserver, so provide a minimal
// IntersectionObserver stand-in local to this suite.
class IntersectionObserverMock {
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
}

beforeAll(() => {
  Object.defineProperty(globalThis, 'IntersectionObserver', {
    configurable: true,
    value: IntersectionObserverMock,
  })
})

const useFAQMock = vi.fn()

vi.mock('@/features/dashboard/hooks/use-status-data', () => ({
  useFAQ: () => useFAQMock(),
}))

// Home page FAQ block (web-home-page spec — "FAQ block reflects
// administrator configuration"): the block must render nothing at all
// (no heading, no placeholder, no empty container) whenever it is disabled,
// still loading, or enabled with zero entries — and must render the
// configured entries, each expandable, once populated.

describe('home page FAQ section', () => {
  test('renders nothing while the FAQ status is still loading', () => {
    useFAQMock.mockReturnValue({ items: [], loading: true })
    const { container } = render(<Faq />)
    expect(container).toBeEmptyDOMElement()
  })

  test('renders nothing when the FAQ feature is disabled (no items, not loading)', () => {
    useFAQMock.mockReturnValue({ items: [], loading: false })
    const { container } = render(<Faq />)
    expect(container).toBeEmptyDOMElement()
    expect(screen.queryByText('FAQ')).not.toBeInTheDocument()
  })

  test('renders the configured entries when the FAQ feature is enabled with content', () => {
    useFAQMock.mockReturnValue({
      items: [
        { id: 1, question: 'How fast is top-up?', answer: 'Within seconds.' },
        { id: 2, question: 'Do I need a card?', answer: 'No card needed.' },
      ],
      loading: false,
    })
    render(<Faq />)

    expect(screen.getByText('How fast is top-up?')).toBeInTheDocument()
    expect(screen.getByText('Do I need a card?')).toBeInTheDocument()
    // Each entry is an expandable trigger (accessible via its accordion role).
    expect(screen.getAllByRole('button')).toHaveLength(2)
  })
})
