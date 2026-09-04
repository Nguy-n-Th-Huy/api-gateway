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
import { describe, expect, test } from 'vitest'

import { Button, buttonVariants } from '../button'

/*
 * Contract: icon-only buttons are drawn at 24–32px but must expose a
 * 44x44 CSS-px activatable area on coarse-pointer (touch) viewports.
 * The visible size stays unchanged on pointer:fine (desktop). In code
 * this is realized by the `touch-target-floor` utility applied under
 * the `pointer-coarse:` Tailwind variant (→ `@media (pointer: coarse)`).
 *
 * jsdom limitation: jsdom does not evaluate `@media (pointer: coarse)`,
 * so computed-style checks for the 44px floor would be meaningless.
 * We instead assert the project-observable invariant — that the Button
 * base class declares `pointer-coarse:touch-target-floor` — which the
 * stylesheet expands to the ::after hit-area covering max(44px, 100%).
 * See web/src/styles/index.css (`@utility touch-target-floor`).
 */

function hasTouchTargetFloor(className: string): boolean {
  return className.includes('pointer-coarse:touch-target-floor')
}

describe('Button touch-target floor', () => {
  test('base buttonVariants includes the coarse-pointer touch-target utility', () => {
    // Arrange
    const className = buttonVariants({ size: 'default', variant: 'default' })

    // Assert — the floor is on the base, so every size inherits it
    expect(hasTouchTargetFloor(className)).toBe(true)
  })

  test('icon size maps to the touch-target floor utility', () => {
    const className = buttonVariants({ size: 'icon' })
    expect(hasTouchTargetFloor(className)).toBe(true)
  })

  test('icon-sm size maps to the touch-target floor utility', () => {
    const className = buttonVariants({ size: 'icon-sm' })
    expect(hasTouchTargetFloor(className)).toBe(true)
  })

  test('icon-xs size maps to the touch-target floor utility', () => {
    const className = buttonVariants({ size: 'icon-xs' })
    expect(hasTouchTargetFloor(className)).toBe(true)
  })

  test('rendered icon button exposes the touch-target class in the DOM', () => {
    // Arrange & Act
    render(
      <Button size='icon' aria-label='add'>
        +
      </Button>
    )
    const button = screen.getByRole('button', { name: 'add' })

    // Assert
    expect(button.className).toContain('pointer-coarse:touch-target-floor')
  })

  test('rendered default button also exposes the touch-target class (not icon-only)', () => {
    render(<Button>Save</Button>)
    const button = screen.getByRole('button', { name: 'Save' })
    expect(button.className).toContain('pointer-coarse:touch-target-floor')
  })
})
