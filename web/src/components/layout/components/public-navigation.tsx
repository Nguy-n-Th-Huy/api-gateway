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
import { useRouterState } from '@tanstack/react-router'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { useTopNavLinks } from '@/hooks/use-top-nav-links'
import { cn } from '@/lib/utils'

import { defaultTopNavLinks } from '../config/top-nav.config'
import { groupTopNavLinks } from '../lib/group-top-nav-links'
import type { TopNavLink } from '../types'
import { NavGroupMenu } from './nav-group-menu'

interface PublicNavigationProps {
  /**
   * Custom navigation links
   * If not provided, will use dynamic links from backend or defaults
   */
  links?: TopNavLink[]
  /**
   * Additional className
   */
  className?: string
}

/**
 * Public navigation component that matches Launch UI template styling.
 * Consumes the same grouping function as `PublicHeader` so the two
 * renderers cannot drift apart.
 */
export function PublicNavigation({
  links: providedLinks,
  className,
}: PublicNavigationProps = {}) {
  const { t } = useTranslation()
  // Use the same logic as AppHeader: prioritize dynamic links from backend
  const dynamicLinks = useTopNavLinks()
  const defaultLinks = providedLinks || defaultTopNavLinks
  const links = dynamicLinks.length > 0 ? dynamicLinks : defaultLinks

  // Derive active state from the current location; dynamic links carry no
  // isActive flag, so path matching is the single source of truth.
  const routerState = useRouterState()
  const pathname = routerState.location.pathname

  const entries = useMemo(() => groupTopNavLinks(links, t), [links, t])

  return (
    <NavGroupMenu
      entries={entries}
      pathname={pathname}
      className={cn('hidden items-center gap-1 md:flex', className)}
    />
  )
}
