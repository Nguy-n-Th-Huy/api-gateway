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
import * as React from 'react'

import { cn } from '@/lib/utils'

function Textarea({ className, ...props }: React.ComponentProps<'textarea'>) {
  return (
    <textarea
      data-slot='textarea'
      className={cn(
        'border-input bg-muted placeholder:text-muted-foreground/70 flex field-sizing-content max-h-96 min-h-16 w-full overflow-y-auto rounded-lg border px-3 py-2 text-base transition-colors outline-none focus-visible:border-primary focus-visible:bg-card focus-visible:ring-4 focus-visible:ring-primary/15 disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-60 aria-invalid:border-destructive aria-invalid:ring-4 aria-invalid:ring-destructive/15 md:text-sm',
        className
      )}
      {...props}
    />
  )
}

export { Textarea }
