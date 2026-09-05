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
import { useCallback, useState } from 'react'
import { toast } from 'sonner'

import { checkToken, KeyCheckRequestError } from '../api'
import type { KeyCheckReport } from '../types'

export type KeyCheckStatus = 'idle' | 'loading' | 'error' | 'success'

export interface UseKeyCheckResult {
  status: KeyCheckStatus
  /** `null` until a lookup succeeds, and cleared again on the next failure
   * so a stale result is never presented as current. */
  report: KeyCheckReport | null
  /** The raw key from the last successful lookup — needed to build the
   * setup section's command. Never rendered back to the visitor in full. */
  checkedKey: string | null
  errorMessage: string | null
  submit: (key: string) => Promise<void>
}

/** Drives the key-check form's four states and holds the last successful
 * report. See specs/public-key-check/spec.md — "Key check page presents
 * every documented state". */
export function useKeyCheck(): UseKeyCheckResult {
  const [status, setStatus] = useState<KeyCheckStatus>('idle')
  const [report, setReport] = useState<KeyCheckReport | null>(null)
  const [checkedKey, setCheckedKey] = useState<string | null>(null)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const submit = useCallback(async (key: string) => {
    setStatus('loading')
    setErrorMessage(null)

    try {
      const result = await checkToken(key)
      setReport(result)
      setCheckedKey(key)
      setStatus('success')
    } catch (error) {
      setReport(null)
      setCheckedKey(null)
      setStatus('error')
      const message =
        error instanceof KeyCheckRequestError
          ? error.message
          : (error as Error)?.message
      if (message) {
        setErrorMessage(message)
        toast.error(message)
      }
    }
  }, [])

  return { status, report, checkedKey, errorMessage, submit }
}
