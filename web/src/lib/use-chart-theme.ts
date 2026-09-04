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
import { useEffect, useRef, useState } from 'react'

import { useTheme } from '@/context/theme-provider'

/**
 * Literal color values VChart needs. VChart cannot consume CSS custom
 * properties, so the chart theme hook resolves the design tokens from the
 * computed style of the document root and hands charts plain strings. The
 * values are re-resolved whenever the resolved light/dark mode changes so a
 * mode switch never leaves light-mode series colors on a dark surface.
 */
export interface ChartThemeColors {
  /** Categorical series palette, ordered --chart-1 through --chart-5. */
  series: string[]
  /** Axis labels and legend text. */
  text: string
  /** Gridlines and axis strokes. */
  grid: string
  /** Chart surface / tooltip background. */
  surface: string
  /** Semantic success color for health/uptime series. */
  success: string
}

const SERIES_TOKENS = [
  '--chart-1',
  '--chart-2',
  '--chart-3',
  '--chart-4',
  '--chart-5',
] as const

export function resolveChartThemeColors(): ChartThemeColors {
  if (typeof document === 'undefined') {
    return { series: [], text: '', grid: '', surface: '', success: '' }
  }
  const style = getComputedStyle(document.documentElement)
  const read = (token: string) => style.getPropertyValue(token).trim()
  return {
    series: SERIES_TOKENS.map(read),
    text: read('--muted-foreground'),
    grid: read('--border'),
    surface: read('--card'),
    success: read('--success'),
  }
}

/**
 * Lazy-load VChart's `ThemeManager` and switch its theme to follow the
 * resolved app theme (light / dark). Returns flags consumers can use to
 * defer chart rendering until the theme is ready, plus the resolved token
 * colors for specs that need literal values.
 */
let themeManagerPromise: Promise<
  (typeof import('@visactor/vchart'))['ThemeManager']
> | null = null

export function useChartTheme() {
  const { resolvedTheme } = useTheme()
  const [themeReady, setThemeReady] = useState(false)
  const [chartColors, setChartColors] = useState<ChartThemeColors>(() =>
    resolveChartThemeColors()
  )
  const themeRef = useRef<
    (typeof import('@visactor/vchart'))['ThemeManager'] | null
  >(null)

  useEffect(() => {
    let cancelled = false
    const updateTheme = async () => {
      setThemeReady(false)
      if (!themeManagerPromise) {
        themeManagerPromise = import('@visactor/vchart').then(
          (m) => m.ThemeManager
        )
      }
      const ThemeManager = await themeManagerPromise
      if (cancelled) return
      themeRef.current = ThemeManager
      ThemeManager.setCurrentTheme(resolvedTheme === 'dark' ? 'dark' : 'light')
      setChartColors(resolveChartThemeColors())
      setThemeReady(true)
    }
    updateTheme()
    return () => {
      cancelled = true
    }
  }, [resolvedTheme])

  return { resolvedTheme, themeReady, chartColors }
}
