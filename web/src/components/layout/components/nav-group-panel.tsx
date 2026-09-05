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
  ArrowRight01Icon,
  BookOpen01Icon,
  Coins01Icon,
  InformationCircleIcon,
  Medal01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'

import { NavigationMenuLink } from '@/components/ui/navigation-menu'
import { cn } from '@/lib/utils'

import type { GroupedNavEntry, NavGroupChild, TopNavLink } from '../types'

type NavGroupEntry = Extract<GroupedNavEntry, { kind: 'group' }>

interface NavGroupPanelProps {
  group: NavGroupEntry
  pathname: string
  onLinkClick?: (
    event: React.MouseEvent<HTMLAnchorElement>,
    link: TopNavLink
  ) => void
}

const CHILD_ICONS: Record<string, typeof Coins01Icon> = {
  '/pricing': Coins01Icon,
  '/rankings': Medal01Icon,
  '/docs': BookOpen01Icon,
  '/about': InformationCircleIcon,
}

function resolveChildIcon(child: NavGroupChild): typeof Coins01Icon {
  if (child.external) return BookOpen01Icon
  return CHILD_ICONS[child.href] ?? BookOpen01Icon
}

/**
 * The desktop mega-menu panel for a group entry: a grid of child rows (icon,
 * title, description) plus an optional highlight cell. Built entirely on the
 * shared `NavigationMenuLink` so the primitive keeps owning focus, Escape
 * and `closeOnClick` — no hand-rolled open/close here.
 */
export function NavGroupPanel(props: NavGroupPanelProps) {
  const cellCount = props.group.children.length + (props.group.highlight ? 1 : 0)

  return (
    <div
      className={cn(
        'grid w-[min(880px,calc(100vw-2rem))] gap-5 p-5',
        cellCount >= 3 ? 'grid-cols-3' : 'grid-cols-2'
      )}
    >
      {props.group.children.map((child) => {
        const isActive = props.pathname === child.href
        const icon = resolveChildIcon(child)
        const rowClassName = cn(
          'flex min-h-11 items-start gap-3 rounded-xl p-2.5 no-underline transition-colors',
          isActive ? 'bg-accent' : 'hover:bg-muted',
          child.disabled && 'pointer-events-none opacity-40'
        )
        const content = (
          <>
            <span className='bg-muted text-accent-foreground flex size-9 shrink-0 items-center justify-center rounded-lg'>
              <HugeiconsIcon
                icon={icon}
                strokeWidth={1.8}
                className='size-4'
                aria-hidden='true'
              />
            </span>
            <span className='flex flex-col gap-0.5'>
              <span className='text-foreground text-sm font-semibold leading-tight'>
                {child.title}
              </span>
              <span className='text-muted-foreground text-xs leading-snug'>
                {child.description}
              </span>
            </span>
          </>
        )

        if (child.external) {
          return (
            <NavigationMenuLink
              key={child.href}
              href={child.href}
              target='_blank'
              rel='noopener noreferrer'
              closeOnClick
              aria-current={isActive ? 'page' : undefined}
              aria-disabled={child.disabled}
              onClick={(event) => props.onLinkClick?.(event, child)}
              className={rowClassName}
            >
              {content}
            </NavigationMenuLink>
          )
        }

        return (
          <NavigationMenuLink
            key={child.href}
            render={<Link to={child.href} disabled={child.disabled} />}
            closeOnClick
            aria-current={isActive ? 'page' : undefined}
            onClick={(event) => props.onLinkClick?.(event, child)}
            className={rowClassName}
          >
            {content}
          </NavigationMenuLink>
        )
      })}

      {props.group.highlight && (
        <NavigationMenuLink
          render={<Link to={props.group.highlight.href} />}
          closeOnClick
          className='bg-accent flex min-h-11 flex-col justify-between gap-3 rounded-xl p-4 no-underline'
        >
          <span className='flex flex-col gap-1.5'>
            <span className='font-display text-accent-foreground text-lg leading-tight font-semibold'>
              {props.group.highlight.title}
            </span>
            <span className='text-accent-foreground/80 text-xs leading-snug'>
              {props.group.highlight.description}
            </span>
          </span>
          <span className='text-accent-foreground inline-flex items-center gap-1.5 text-sm font-semibold'>
            {props.group.highlight.linkLabel}
            <HugeiconsIcon
              icon={ArrowRight01Icon}
              strokeWidth={2}
              className='size-3.5'
              aria-hidden='true'
            />
          </span>
        </NavigationMenuLink>
      )}
    </div>
  )
}
