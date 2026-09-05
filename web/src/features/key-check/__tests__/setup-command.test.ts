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
import { describe, expect, test } from 'vitest'

import { APPLICATIONS, OS_TARGETS, type AppConfig } from '../lib/applications'
import {
  buildSetupCommand,
  canBuildSetupCommand,
  maskKey,
  reconcileModelSelections,
} from '../lib/setup-command'

function mustFindApp(id: string): AppConfig {
  const app = APPLICATIONS.find((candidate) => candidate.id === id)
  if (!app) throw new Error(`Fixture error: no application registered as "${id}"`)
  return app
}

const claudeCode = mustFindApp('claude-code')

describe('setup command — model selectors', () => {
  test('every slot is populated exclusively from available_models', () => {
    const availableModels = ['gpt-4o', 'gpt-4o-mini', 'claude-3', 'gemini-pro']
    const selections = reconcileModelSelections(
      claudeCode,
      availableModels,
      {}
    )

    expect(Object.keys(selections)).toEqual(
      claudeCode.slots.map((slot) => slot.id)
    )
    for (const value of Object.values(selections)) {
      expect(availableModels).toContain(value)
    }
  })

  test('an empty available_models list yields no selections and no command', () => {
    expect(canBuildSetupCommand([])).toBe(false)
    expect(reconcileModelSelections(claudeCode, [], { opus: 'gpt-4o' })).toEqual(
      {}
    )
  })

  test('re-checking a different group replaces a selection the new group does not enable', () => {
    const previous = { opus: 'old-model', sonnet: 'old-model' }
    const newAvailableModels = ['new-model']

    const reconciled = reconcileModelSelections(
      claudeCode,
      newAvailableModels,
      previous
    )

    expect(reconciled.opus).toBe('new-model')
    expect(reconciled.sonnet).toBe('new-model')
  })

  test('a selection the new group still enables is kept', () => {
    const previous = { opus: 'kept-model' }
    const reconciled = reconcileModelSelections(
      claudeCode,
      ['kept-model', 'other-model'],
      previous
    )

    expect(reconciled.opus).toBe('kept-model')
  })

  test('Pi and Oh My Pi have no documented model slots', () => {
    const pi = mustFindApp('pi')
    const ohMyPi = mustFindApp('oh-my-pi')

    expect(pi.slots).toEqual([])
    expect(ohMyPi.slots).toEqual([])
  })
})

describe('setup command — masking', () => {
  test('the displayed command masks the key while the copied command carries it', () => {
    const key = 'sk-abcdef1234567890'
    const command = buildSetupCommand({
      app: claudeCode,
      os: OS_TARGETS.UNIX,
      key,
      baseUrl: 'https://gateway.example.com',
      modelSelections: { opus: 'gpt-4o', sonnet: 'gpt-4o', haiku: 'gpt-4o', subagent: 'gpt-4o' },
    })

    expect(command.display).not.toContain(key)
    expect(command.copyText).toContain(key)
  })

  test('maskKey never returns the full key', () => {
    const key = 'sk-abcdef1234567890'
    const masked = maskKey(key)

    expect(masked).not.toBe(key)
    expect(masked.length).toBeLessThanOrEqual(key.length)
  })

  test('the Windows command uses the PowerShell one-liner form', () => {
    const command = buildSetupCommand({
      app: claudeCode,
      os: OS_TARGETS.WINDOWS,
      key: 'sk-abcdef1234567890',
      baseUrl: 'https://gateway.example.com',
      modelSelections: {},
    })

    expect(command.copyText.startsWith('irm "')).toBe(true)
    expect(command.copyText.endsWith('| iex')).toBe(true)
  })

  test('a Bearer-prefixed key with internal whitespace is never shown unmasked', () => {
    // The raw checked key can carry a "Bearer " prefix and a space (the
    // backend's normalizeTokenKey accepts that form). The space gets
    // percent-encoded once the key travels through URLSearchParams, so a
    // masking approach that string-replaces the *raw* key against the
    // already-encoded command would find no match and leave the real key
    // fully visible on screen.
    const key = 'Bearer sk-abcdef1234567890'
    const command = buildSetupCommand({
      app: claudeCode,
      os: OS_TARGETS.UNIX,
      key,
      baseUrl: 'https://gateway.example.com',
      modelSelections: {
        opus: 'gpt-4o',
        sonnet: 'gpt-4o',
        haiku: 'gpt-4o',
        subagent: 'gpt-4o',
      },
    })

    expect(command.display).not.toContain(key)
    expect(command.display).not.toContain('abcdef1234567890')

    const copyUrlMatch = command.copyText.match(/curl -fsSL "([^"]+)"/)
    if (!copyUrlMatch) {
      throw new Error('Fixture error: copyText did not contain a curl URL')
    }
    const copyUrl = new URL(copyUrlMatch[1])
    expect(copyUrl.searchParams.get('key')).toBe(key)
  })

  test('the macOS/Linux command uses the curl one-liner form', () => {
    const command = buildSetupCommand({
      app: claudeCode,
      os: OS_TARGETS.UNIX,
      key: 'sk-abcdef1234567890',
      baseUrl: 'https://gateway.example.com',
      modelSelections: {},
    })

    expect(command.copyText.startsWith('curl -fsSL "')).toBe(true)
    expect(command.copyText.endsWith('| sh')).toBe(true)
  })
})
