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
import { beforeAll, describe, expect, test, vi } from 'vitest'

import { PricingPreview } from '../pricing-preview'

// AnimateInView (used inside PricingPreview) observes intersection on mount;
// the shared test-setup.ts only stubs ResizeObserver, so provide a minimal
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

// PricingPreview renders a TanStack <Link> for "View Pricing"; provide a
// minimal stand-in so it can mount without a live router context.
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

const usePricingDataMock = vi.fn()

vi.mock('@/features/pricing/hooks/use-pricing-data', () => ({
  usePricingData: () => usePricingDataMock(),
}))

const sampleModel = {
  id: 1,
  model_name: 'gpt-5',
  quota_type: 0,
  model_ratio: 1,
  completion_ratio: 2,
  enable_groups: ['default'],
  key: 'gpt-5',
}

// web-home-page spec — "Pricing preview is sourced from live pricing data":
// loading, error-with-retry and empty treatments must each keep the page
// usable, and no price is ever hardcoded.

describe('home page pricing preview', () => {
  test('shows a loading treatment while pricing data is being fetched', () => {
    usePricingDataMock.mockReturnValue({
      models: [],
      isLoading: true,
      error: null,
      refetch: vi.fn(),
      priceRate: 1,
      usdExchangeRate: 1,
    })
    render(<PricingPreview />)
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  test('shows an error message with a retry action when the fetch fails, and refetch is called on retry', async () => {
    const refetch = vi.fn()
    usePricingDataMock.mockReturnValue({
      models: [],
      isLoading: false,
      error: new Error('network error'),
      refetch,
      priceRate: 1,
      usdExchangeRate: 1,
    })
    const user = userEvent.setup()
    render(<PricingPreview />)

    expect(
      screen.getByText('Unable to load pricing right now.')
    ).toBeInTheDocument()
    const retryButton = screen.getByRole('button', { name: 'Retry' })
    await user.click(retryButton)
    expect(refetch).toHaveBeenCalledTimes(1)
  })

  test('shows an empty-state message when pricing data returns no models', () => {
    usePricingDataMock.mockReturnValue({
      models: [],
      isLoading: false,
      error: null,
      refetch: vi.fn(),
      priceRate: 1,
      usdExchangeRate: 1,
    })
    render(<PricingPreview />)
    expect(
      screen.getByText('No models are configured yet.')
    ).toBeInTheDocument()
  })

  test('renders a bounded preview table of models with prices when data is available', () => {
    usePricingDataMock.mockReturnValue({
      models: [sampleModel],
      isLoading: false,
      error: null,
      refetch: vi.fn(),
      priceRate: 1,
      usdExchangeRate: 1,
    })
    render(<PricingPreview />)
    expect(screen.getByRole('table')).toBeInTheDocument()
    expect(screen.getByText('gpt-5')).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'View Pricing' })
    ).toHaveAttribute('href', '/pricing')
  })
})
