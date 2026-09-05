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
import { Link } from '@tanstack/react-router'

import {
  NavigationMenu,
  NavigationMenuContent,
  NavigationMenuItem,
  NavigationMenuLink,
  NavigationMenuList,
  NavigationMenuTrigger,
} from '@/components/ui/navigation-menu'
import { cn } from '@/lib/utils'

import type { GroupedNavEntry, TopNavLink } from '../types'
import { NavGroupPanel } from './nav-group-panel'

interface NavGroupMenuProps {
  entries: GroupedNavEntry[]
  pathname: string
  onLinkClick?: (
    event: React.MouseEvent<HTMLAnchorElement>,
    link: TopNavLink
  ) => void
  className?: string
}

function topLevelLinkClassName(isActive: boolean, disabled?: boolean) {
  return cn(
    'rounded-full px-3.5 py-1.5 text-sm font-medium transition-colors',
    isActive
      ? 'bg-accent text-accent-foreground font-semibold'
      : 'text-muted-foreground hover:bg-accent/55 hover:text-foreground',
    disabled && 'pointer-events-none opacity-40'
  )
}

function triggerClassName(isActive: boolean) {
  return cn(
    'h-auto gap-1 rounded-full px-3.5 py-1.5 text-sm font-medium hover:bg-accent/55',
    isActive
      ? 'bg-accent text-accent-foreground data-open:bg-accent data-popup-open:bg-accent font-semibold'
      : 'text-muted-foreground hover:text-foreground'
  )
}

/**
 * The public header's desktop navigation: plain top-level links and group
 * triggers share one `NavigationMenu`, so arrow-key movement, Escape and
 * focus handling stay coordinated across the whole bar. Shared by
 * `PublicHeader` and `PublicNavigation` so the two renderers cannot drift.
 */
export function NavGroupMenu(props: NavGroupMenuProps) {
  return (
    <NavigationMenu
      align='center'
      className={cn('max-w-none flex-none justify-start', props.className)}
    >
      <NavigationMenuList className='gap-0.5'>
        {props.entries.map((entry) => {
          if (entry.kind === 'link') {
            const isActive = props.pathname === entry.href
            return (
              <NavigationMenuItem key={entry.href}>
                {entry.external ? (
                  <NavigationMenuLink
                    href={entry.href}
                    target='_blank'
                    rel='noopener noreferrer'
                    aria-current={isActive ? 'page' : undefined}
                    aria-disabled={entry.disabled}
                    onClick={(event) => props.onLinkClick?.(event, entry)}
                    className={topLevelLinkClassName(isActive, entry.disabled)}
                  >
                    {entry.title}
                  </NavigationMenuLink>
                ) : (
                  <NavigationMenuLink
                    render={<Link to={entry.href} disabled={entry.disabled} />}
                    aria-current={isActive ? 'page' : undefined}
                    onClick={(event) => props.onLinkClick?.(event, entry)}
                    className={topLevelLinkClassName(isActive, entry.disabled)}
                  >
                    {entry.title}
                  </NavigationMenuLink>
                )}
              </NavigationMenuItem>
            )
          }

          const isGroupActive = entry.children.some(
            (child) => child.href === props.pathname
          )

          return (
            <NavigationMenuItem key={entry.id}>
              <NavigationMenuTrigger className={triggerClassName(isGroupActive)}>
                {entry.label}
              </NavigationMenuTrigger>
              <NavigationMenuContent>
                <NavGroupPanel
                  group={entry}
                  pathname={props.pathname}
                  onLinkClick={props.onLinkClick}
                />
              </NavigationMenuContent>
            </NavigationMenuItem>
          )
        })}
      </NavigationMenuList>
    </NavigationMenu>
  )
}
