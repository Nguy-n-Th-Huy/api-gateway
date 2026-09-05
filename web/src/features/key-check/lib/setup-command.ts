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
import {
  KEY_MASK_VISIBLE_CHARS,
  SETUP_SCRIPT_ENDPOINT,
  SETUP_SCRIPT_QUERY_PARAMS,
} from '../constants'
import { OS_TARGETS, type AppConfig, type OsTarget } from './applications'

/** Masks a key to only its leading/trailing characters, e.g. `sk-a...f456`.
 * Never returns the full key. */
export function maskKey(key: string): string {
  if (key.length <= KEY_MASK_VISIBLE_CHARS * 2) {
    if (key.length <= 2) return '*'.repeat(key.length)
    return `${key.at(0)}${'*'.repeat(key.length - 2)}${key.at(-1)}`
  }
  return `${key.slice(0, KEY_MASK_VISIBLE_CHARS)}...${key.slice(-KEY_MASK_VISIBLE_CHARS)}`
}

export interface SetupCommandParams {
  app: AppConfig
  os: OsTarget
  /** The real, unmasked key. */
  key: string
  /** Gateway base URL (no trailing slash required). */
  baseUrl: string
  /** Slot id -> selected model name. Ignored for an app with no slots. */
  modelSelections: Record<string, string>
}

/** Whether the setup section can offer a command at all — it cannot when
 * the key's group enables no model. */
export function canBuildSetupCommand(availableModels: string[]): boolean {
  return availableModels.length > 0
}

/**
 * Keeps an existing per-slot selection when the (possibly new) group still
 * enables it, and otherwise falls back to the group's first available model.
 * Used both for the initial default selection (`previousSelections: {}`) and
 * when a newly checked key's group replaces the current one.
 */
export function reconcileModelSelections(
  app: AppConfig,
  availableModels: string[],
  previousSelections: Record<string, string>
): Record<string, string> {
  if (!canBuildSetupCommand(availableModels)) return {}

  const availableSet = new Set(availableModels)
  const fallback = availableModels[0]

  return Object.fromEntries(
    app.slots.map((slot) => {
      const previous = previousSelections[slot.id]
      const value =
        previous && availableSet.has(previous) ? previous : fallback
      return [slot.id, value]
    })
  )
}

export function buildSetupScriptUrl(params: SetupCommandParams): string {
  const query = new URLSearchParams()
  query.set(SETUP_SCRIPT_QUERY_PARAMS.key, params.key)
  query.set(SETUP_SCRIPT_QUERY_PARAMS.application, params.app.id)
  query.set(SETUP_SCRIPT_QUERY_PARAMS.os, params.os)

  for (const slot of params.app.slots) {
    const model = params.modelSelections[slot.id]
    if (model) query.set(`model_${slot.id}`, model)
  }

  const base = params.baseUrl.replace(/\/+$/, '')
  return `${base}${SETUP_SCRIPT_ENDPOINT}?${query.toString()}`
}

/** The copy-paste one-liner: `irm ... | iex` on Windows, `curl ... | sh`
 * elsewhere. The frontend never fetches the script itself. */
export function buildSetupOneLiner(url: string, os: OsTarget): string {
  return os === OS_TARGETS.WINDOWS
    ? `irm "${url}" | iex`
    : `curl -fsSL "${url}" | sh`
}

export interface SetupCommandResult {
  /** Safe to render on screen — the key is masked. */
  display: string
  /** The real, working command. Only ever placed on the clipboard. */
  copyText: string
}

export function buildSetupCommand(
  params: SetupCommandParams
): SetupCommandResult {
  const copyUrl = buildSetupScriptUrl(params)
  const copyText = buildSetupOneLiner(copyUrl, params.os)

  if (!params.key) {
    return { display: copyText, copyText }
  }

  // Building a masked variant of the *URL* — rather than string-replacing
  // the raw key inside the already-built copyText — keeps the mask correct
  // regardless of how the key gets percent-encoded. The key travels through
  // URLSearchParams, so a key containing a character it encodes (e.g. the
  // space in a pasted "Bearer sk-..." value, which the backend's
  // normalizeTokenKey accepts) would never literal-match inside the encoded
  // copyText, leaving the real key on screen.
  const displayUrl = buildSetupScriptUrl({
    ...params,
    key: maskKey(params.key),
  })
  const display = buildSetupOneLiner(displayUrl, params.os)

  return { display, copyText }
}
