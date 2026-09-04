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
import { useState } from 'react'
import { describe, expect, test, vi } from 'vitest'

import { Button } from '../button'
import { Checkbox } from '../checkbox'
import { Input } from '../input'
import { Switch } from '../switch'

function ClickLatch() {
  const [count, setCount] = useState(0)
  const [loading, setLoading] = useState(false)
  const onClick = () => {
    if (loading) return
    setLoading(true)
    setCount((c) => c + 1)
    // keep disabled so double-click cannot fire again in this tick
  }
  return (
    <>
      <Button onClick={onClick} disabled={loading}>
        Save
      </Button>
      <output data-testid='save-count'>{count}</output>
    </>
  )
}

describe('Button primitive', () => {
  test('renders as an accessible button and fires onClick', async () => {
    const onClick = vi.fn()
    render(<Button onClick={onClick}>Save</Button>)
    const button = screen.getByRole('button', { name: 'Save' })
    await userEvent.click(button)
    expect(onClick).toHaveBeenCalledTimes(1)
  })

  test('disabled button is inert to pointer interaction', async () => {
    const onClick = vi.fn()
    render(
      <Button disabled onClick={onClick}>
        Save
      </Button>
    )
    const button = screen.getByRole('button', { name: 'Save' })
    expect(button).toBeDisabled()
    await userEvent.click(button).catch(() => {})
    expect(onClick).not.toHaveBeenCalled()
  })

  test('disabled button is inert to keyboard activation', async () => {
    const user = userEvent.setup()
    const onClick = vi.fn()
    render(
      <div>
        <Button disabled onClick={onClick} aria-label='save'>
          Save
        </Button>
        <button type='button'>other</button>
      </div>
    )
    await user.tab()
    const active = document.activeElement
    // the disabled Save button must not hold keyboard focus
    expect(active?.getAttribute('aria-label')).not.toBe('save')
  })

  test('focus is visible when the enabled button receives keyboard focus', async () => {
    const user = userEvent.setup()
    render(<Button>Save</Button>)
    await user.tab()
    const button = screen.getByRole('button', { name: 'Save' })
    expect(button).toHaveFocus()
  })

  test('loading state prevents double submission', async () => {
    render(<ClickLatch />)
    const button = screen.getByRole('button', { name: 'Save' })
    await userEvent.click(button)
    const buttonAfter = screen.getByRole('button', { name: 'Save' })
    // after first click the button is disabled, so a second pointer event is inert
    expect(buttonAfter).toBeDisabled()
    expect(screen.getByTestId('save-count').textContent).toBe('1')
    await userEvent.click(buttonAfter).catch(() => {})
    expect(screen.getByTestId('save-count').textContent).toBe('1')
  })
})

describe('Switch primitive', () => {
  test('toggles checked state on click', async () => {
    render(<Switch data-testid='s' />)
    const el = screen.getByTestId('s')
    expect(el.getAttribute('aria-checked')).toBe('false')
    await userEvent.click(el)
    expect(el.getAttribute('aria-checked')).toBe('true')
  })

  test('disabled switch is inert', async () => {
    render(<Switch disabled data-testid='s' />)
    const el = screen.getByTestId('s')
    expect(
      el.hasAttribute('disabled') ||
        el.getAttribute('aria-disabled') === 'true' ||
        el.hasAttribute('data-disabled')
    ).toBe(true)
    const before = el.getAttribute('aria-checked')
    await userEvent.click(el).catch(() => {})
    expect(el.getAttribute('aria-checked')).toBe(before)
  })
})

describe('Checkbox primitive', () => {
  test('exposes a checkbox role and toggles', async () => {
    render(<Checkbox />)
    const box = screen.getByRole('checkbox')
    expect(box.getAttribute('aria-checked')).toBe('false')
    await userEvent.click(box)
    expect(box.getAttribute('aria-checked')).toBe('true')
  })

  test('disabled checkbox is inert', async () => {
    render(<Checkbox disabled />)
    const box = screen.getByRole('checkbox')
    expect(
      box.hasAttribute('data-disabled') || box.hasAttribute('disabled')
    ).toBe(true)
    const before = box.getAttribute('aria-checked')
    await userEvent.click(box).catch(() => {})
    expect(box.getAttribute('aria-checked')).toBe(before)
  })
})

describe('Input primitive', () => {
  test('aria-invalid state is exposed to AT when invalid', () => {
    render(<Input aria-label='email' aria-invalid />)
    expect(screen.getByLabelText('email')).toHaveAttribute(
      'aria-invalid',
      'true'
    )
  })

  test('disabled input is not interactable', async () => {
    render(<Input aria-label='email' disabled />)
    const input = screen.getByLabelText('email') as HTMLInputElement
    expect(input).toBeDisabled()
    await userEvent.type(input, 'x').catch(() => {})
    expect(input.value).toBe('')
  })

  test('disabled input exposes disabled state to AT', () => {
    render(<Input aria-label='email' disabled />)
    expect(screen.getByLabelText('email')).toBeDisabled()
  })
})
