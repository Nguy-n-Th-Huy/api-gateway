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
import { useQuery } from '@tanstack/react-query'
import { useMemo, useState } from 'react'

import { fetchTokenLogs } from '../api'
import { USAGE_LOG_PAGE_SIZE } from '../constants'
import { getUsageLogPageCount } from '../lib/log-display'
import type { KeyUsageLogEntry } from '../types'

export interface UseTokenLogsResult {
  entries: KeyUsageLogEntry[]
  page: number
  pageCount: number
  total: number
  isLoading: boolean
  isError: boolean
  goToPage: (page: number) => void
  refetch: () => void
}

/**
 * Loads the checked key's usage log for the current page, independently of
 * the check itself so a log failure never blocks the report — see
 * specs/public-key-check/spec.md, "Key check page presents the checked key's
 * usage log".
 *
 * The query key carries the key and the page, so paging back to a visited
 * page is served from cache. It deliberately carries no `placeholderData`:
 * keeping the previous result would present the previous key's entries as the
 * current ones while a newly checked key loads. A different key starts again
 * at page one, decided during render rather than in an effect so the new key
 * is never requested with the old key's page number.
 */
export function useTokenLogs(checkedKey: string | null): UseTokenLogsResult {
  const [paging, setPaging] = useState<{ key: string | null; page: number }>({
    key: checkedKey,
    page: 1,
  })

  if (paging.key !== checkedKey) {
    setPaging({ key: checkedKey, page: 1 })
  }
  const page = paging.key === checkedKey ? paging.page : 1

  const key = checkedKey ?? ''
  const query = useQuery({
    queryKey: ['key-check', 'token-logs', key, page],
    queryFn: () => fetchTokenLogs(key, page),
    enabled: key !== '',
  })

  const entries = useMemo(() => query.data?.items ?? [], [query.data])
  const total = query.data?.total ?? 0

  return {
    entries,
    page,
    pageCount: getUsageLogPageCount(total, USAGE_LOG_PAGE_SIZE),
    total,
    isLoading: query.isLoading,
    isError: query.isError,
    goToPage: (nextPage: number) => {
      setPaging({ key: checkedKey, page: nextPage })
    },
    refetch: () => {
      void query.refetch()
    },
  }
}