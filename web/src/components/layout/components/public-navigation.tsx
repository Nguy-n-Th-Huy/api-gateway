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
import { Link, useRouterState } from '@tanstack/react-router'

import { useTopNavLinks } from '@/hooks/use-top-nav-links'
import { cn } from '@/lib/utils'

import { defaultTopNavLinks } from '../config/top-nav.config'
import type { TopNavLink } from '../types'

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
 * Public navigation component that matches Launch UI template styling
 * Used in PublicHeader for desktop navigation
 */
export function PublicNavigation({
  links: providedLinks,
  className,
}: PublicNavigationProps = {}) {
  // Use the same logic as AppHeader: prioritize dynamic links from backend
  const dynamicLinks = useTopNavLinks()
  const defaultLinks = providedLinks || defaultTopNavLinks
  const links = dynamicLinks.length > 0 ? dynamicLinks : defaultLinks

  // Derive active state from the current location; dynamic links carry no
  // isActive flag, so path matching is the single source of truth.
  const routerState = useRouterState()
  const pathname = routerState.location.pathname

  return (
    <nav className={cn('hidden items-center gap-1 md:flex', className)}>
      {links.map((link) => {
        const isActive = pathname === link.href
        const linkClassName = cn(
          'rounded-full px-3.5 py-1.5 text-sm font-medium transition-colors focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-hidden',
          isActive
            ? 'bg-accent text-accent-foreground font-semibold'
            : 'text-muted-foreground hover:bg-accent/55 hover:text-foreground',
          link.disabled && 'pointer-events-none opacity-40'
        )
        // Handle external links
        if (link.external) {
          return (
            <a
              key={`${link.title}-${link.href}`}
              href={link.href}
              target='_blank'
              rel='noopener noreferrer'
              aria-disabled={link.disabled}
              className={linkClassName}
            >
              {link.title}
            </a>
          )
        }
        // Handle internal links
        return (
          <Link
            key={`${link.title}-${link.href}`}
            to={link.href}
            disabled={link.disabled}
            aria-current={isActive ? 'page' : undefined}
            className={linkClassName}
          >
            {link.title}
          </Link>
        )
      })}
    </nav>
  )
}
