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
import { act, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { getSePayOrderStatus } from '../api'
import { useSePayOrderPolling } from '../hooks/use-sepay-order-polling'
import type { ApiResponse, SePayOrder, SePayOrderStatus } from '../types'

// Mock the network boundary only; the hook's own polling logic runs for real.
vi.mock('../api', () => ({
  getSePayOrderStatus: vi.fn(),
  isApiSuccess: (response: ApiResponse) =>
    response.success === true || response.message === 'success',
}))

const mockGetStatus = vi.mocked(getSePayOrderStatus)

const baseOrder: SePayOrder = {
  trade_no: 'SPTESTABCDEF1234',
  memo: 'SPTESTABCDEF1234',
  payable_vnd: 10000,
  bank_account: '987654321',
  bank_code: 'niclop',
  account_holder: 'NGUYEN VAN A',
  vietqr_url: 'https://img.vietqr.io/image/niclop-987654321-compact2.png',
  create_time: 1700000000,
  expire_time: 1700000000 + 600,
  status: 'pending',
}

function statusResponse(status: SePayOrderStatus): ApiResponse<SePayOrder> {
  return { success: true, data: { ...baseOrder, status } }
}

function HookProbe(props: {
  tradeNo: string | null
  onSuccess?: (status: SePayOrderStatus) => void
}) {
  const { status, polling } = useSePayOrderPolling({
    tradeNo: props.tradeNo,
    onSuccess: props.onSuccess,
  })

  return (
    <>
      <span data-testid='poll-status'>{String(status)}</span>
      <span data-testid='polling'>{String(polling)}</span>
    </>
  )
}

// Advance fake timers just enough to flush the immediate first tick's awaited
// promise chain (the hook fires an immediate poll on mount, then a 3s interval).
async function flush() {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(0)
  })
}

describe('useSePayOrderPolling', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockGetStatus.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  test('reports the pending status and keeps polling', async () => {
    mockGetStatus.mockResolvedValue(statusResponse('pending'))

    render(<HookProbe tradeNo={baseOrder.trade_no} />)
    await flush()

    expect(mockGetStatus).toHaveBeenCalledWith(baseOrder.trade_no)
    expect(screen.getByTestId('poll-status').textContent).toBe('pending')
    expect(screen.getByTestId('polling').textContent).toBe('true')
  })

  test('stops polling on success and fires onSuccess exactly once', async () => {
    mockGetStatus
      .mockResolvedValueOnce(statusResponse('pending'))
      .mockResolvedValueOnce(statusResponse('success'))
    const onSuccess = vi.fn()

    render(<HookProbe tradeNo={baseOrder.trade_no} onSuccess={onSuccess} />)
    await flush() // immediate first tick -> pending
    await act(async () => {
      await vi.advanceTimersByTimeAsync(3000) // second tick -> success
    })

    expect(onSuccess).toHaveBeenCalledTimes(1)
    expect(onSuccess).toHaveBeenCalledWith('success')
    expect(screen.getByTestId('polling').textContent).toBe('false')

    // Terminal state stops the interval: further time adds no calls.
    const callsAtStop = mockGetStatus.mock.calls.length
    await act(async () => {
      await vi.advanceTimersByTimeAsync(12000)
    })
    expect(mockGetStatus).toHaveBeenCalledTimes(callsAtStop)
  })

  test('stops polling on expired and does not fire onSuccess', async () => {
    mockGetStatus.mockResolvedValue(statusResponse('expired'))
    const onSuccess = vi.fn()

    render(<HookProbe tradeNo={baseOrder.trade_no} onSuccess={onSuccess} />)
    await flush()

    expect(screen.getByTestId('poll-status').textContent).toBe('expired')
    expect(screen.getByTestId('polling').textContent).toBe('false')
    expect(onSuccess).not.toHaveBeenCalled()

    const callsAtStop = mockGetStatus.mock.calls.length
    await act(async () => {
      await vi.advanceTimersByTimeAsync(12000)
    })
    expect(mockGetStatus).toHaveBeenCalledTimes(callsAtStop)
  })

  test('a transient poll error is retried on the next tick', async () => {
    mockGetStatus
      .mockRejectedValueOnce(new Error('flaky network'))
      .mockResolvedValue(statusResponse('pending'))

    render(<HookProbe tradeNo={baseOrder.trade_no} />)
    await flush() // first tick rejects

    expect(screen.getByTestId('poll-status').textContent).toBe('null')
    expect(screen.getByTestId('polling').textContent).toBe('true')

    await act(async () => {
      await vi.advanceTimersByTimeAsync(3000) // retry succeeds
    })
    expect(screen.getByTestId('poll-status').textContent).toBe('pending')
  })

  test('unmount stops the interval', async () => {
    mockGetStatus.mockResolvedValue(statusResponse('pending'))

    const view = render(<HookProbe tradeNo={baseOrder.trade_no} />)
    await flush()
    view.unmount()

    const callsAtUnmount = mockGetStatus.mock.calls.length
    await act(async () => {
      await vi.advanceTimersByTimeAsync(12000)
    })
    expect(mockGetStatus).toHaveBeenCalledTimes(callsAtUnmount)
  })
})
