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
import { useMemo } from 'react'

import { getChannelHealth } from '../api'
import { channelsQueryKeys } from '../lib/channel-actions'
import type { ChannelHealthLookup } from '../lib/channel-utils'

export type { ChannelHealthLookup }

const EMPTY_LOOKUP: ChannelHealthLookup = new Map()

/**
 * Fetches channel health metrics once (backed by the server's short-lived
 * cache) and exposes them as a lookup keyed by channel id. Call this once
 * inside `useChannelsColumns` and close over the lookup in the cell
 * renderers, rather than calling it once per cell.
 */
export function useChannelHealth() {
  const query = useQuery({
    queryKey: channelsQueryKeys.health(),
    queryFn: getChannelHealth,
    retry: false,
    staleTime: 60 * 1000,
  })

  const lookup = useMemo<ChannelHealthLookup>(() => {
    const stats = query.data?.data
    if (!stats || stats.length === 0) {
      return EMPTY_LOOKUP
    }
    const map: ChannelHealthLookup = new Map()
    for (const stat of stats) {
      map.set(stat.channel_id, stat)
    }
    return map
  }, [query.data])

  // A rejected request (network/HTTP failure) throws and sets isError; a
  // business-level failure (success: false) resolves normally, so it must be
  // checked explicitly. Either way, the caller renders the no-data state.
  const isUnavailable = query.isError || query.data?.success === false

  return { lookup, isUnavailable, isLoading: query.isLoading }
}
