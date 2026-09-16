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
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, test } from 'vitest'

// Dynamic imports match the sibling tests in this directory: the drawer is loaded
// after this file's own i18next instance exists.
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { api } = await import('@/lib/api')
const { ApiKeysProvider } = await import('../api-keys-provider')
const { ApiKeysMutateDrawer } = await import('../api-keys-mutate-drawer')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

type ApiMethod = (url: string, data?: unknown) => Promise<{ data: unknown }>
type MockableApi = {
  get: ApiMethod
  post: ApiMethod
  put: ApiMethod
}
type RenderedDrawer = {
  queryClient: InstanceType<typeof QueryClient>
}
type DrawerFixtures = {
  created: Array<Record<string, unknown>>
  updated: Array<Record<string, unknown>>
}
type FixtureOptions = {
  defaultUseAutoGroup?: boolean
  offerAuto?: boolean
  storedKey?: Record<string, unknown>
}

const apiClient = api as unknown as MockableApi
const originalGet = apiClient.get
const originalPost = apiClient.post
const originalPut = apiClient.put
let renderedDrawer: RenderedDrawer | null = null

function buildGroupsData(offerAuto: boolean) {
  const data: Record<string, { desc: string; ratio: number | string }> = {
    default: { desc: 'Standard access', ratio: 1 },
    vip: { desc: 'Priority access', ratio: 2 },
  }
  if (offerAuto) {
    data.auto = { desc: 'Automatic routing', ratio: 'auto' }
  }
  return data
}

function installApiFixtures(
  fixtures: DrawerFixtures,
  options: FixtureOptions = {}
) {
  const { defaultUseAutoGroup = true, offerAuto = true, storedKey } = options
  const groupsData = buildGroupsData(offerAuto)

  apiClient.get = async (url) => {
    if (url === '/api/status') {
      return { data: { data: { default_use_auto_group: defaultUseAutoGroup } } }
    }
    if (url === '/api/user/models') {
      return { data: { success: true, data: [] } }
    }
    if (url === '/api/user/self/groups') {
      return { data: { success: true, data: groupsData } }
    }
    if (url === '/api/token/auto-groups') {
      return {
        data: {
          success: true,
          data: { groups: ['vip', 'default'], max_count: 3 },
        },
      }
    }
    if (storedKey && url === `/api/token/${storedKey.id}`) {
      return { data: { success: true, data: storedKey } }
    }
    throw new Error(`Unexpected GET ${url}`)
  }
  apiClient.post = async (url, data) => {
    expect(url).toBe('/api/token/')
    expect(data && typeof data === 'object').toBeTruthy()
    fixtures.created.push(data as Record<string, unknown>)
    return { data: { success: true, data: {} } }
  }
  apiClient.put = async (url, data) => {
    expect(url).toBe('/api/token/')
    expect(data && typeof data === 'object').toBeTruthy()
    fixtures.updated.push(data as Record<string, unknown>)
    return { data: { success: true, data: {} } }
  }
}

async function renderDrawer(options: {
  currentRow?: Record<string, unknown>
  defaultUseAutoGroup?: boolean
  offerAuto?: boolean
}): Promise<void> {
  const { currentRow, defaultUseAutoGroup = true, offerAuto = true } = options
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const freshAt = Date.now() + 60_000
  queryClient.setQueryData(
    ['status'],
    { default_use_auto_group: defaultUseAutoGroup },
    { updatedAt: freshAt }
  )
  queryClient.setQueryData(
    ['user-models'],
    { success: true, data: [] },
    { updatedAt: freshAt }
  )
  queryClient.setQueryData(
    ['user-groups'],
    { success: true, data: buildGroupsData(offerAuto) },
    { updatedAt: freshAt }
  )
  queryClient.setQueryData(
    ['token-auto-groups'],
    {
      success: true,
      data: { groups: ['vip', 'default'], max_count: 3 },
    },
    { updatedAt: freshAt }
  )
  if (currentRow) {
    queryClient.setQueryData(
      ['api-key', currentRow.id],
      { success: true, data: currentRow },
      { updatedAt: freshAt }
    )
  }
  renderedDrawer = { queryClient }

  render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <ApiKeysProvider>
          <ApiKeysMutateDrawer
            open
            onOpenChange={() => undefined}
            currentRow={currentRow as never}
          />
        </ApiKeysProvider>
      </I18nextProvider>
    </QueryClientProvider>
  )
  await waitFor(
    () => {
      const saveButton = findButton('Save changes', false)
      expect(saveButton).toBeEnabled()
    },
    { timeout: 1500 }
  )
}

function findButton(text: string, required: true): HTMLButtonElement
function findButton(text: string, required: false): HTMLButtonElement | null
function findButton(text: string, required = true): HTMLButtonElement | null {
  const button = screen
    .queryAllByRole<HTMLButtonElement>('button')
    .find((candidate) => candidate.textContent?.includes(text))
  if (required && !button) {
    throw new Error(`Expected button containing "${text}"`)
  }
  return button ?? null
}

function findAriaButton(label: string): HTMLButtonElement {
  const button = document.querySelector<HTMLButtonElement>(
    `button[aria-label="${label}"]`
  )
  if (!button) {
    throw new Error(`Expected button labelled "${label}"`)
  }
  return button
}

function findLabel(labelText: string): HTMLLabelElement | null {
  return (
    [...document.querySelectorAll<HTMLLabelElement>('label')].find(
      (candidate) => candidate.textContent?.trim() === labelText
    ) ?? null
  )
}

function getControlByLabel(labelText: string): HTMLElement {
  const label = findLabel(labelText)
  if (!label) {
    throw new Error(`Expected label "${labelText}"`)
  }

  const control =
    label.control ??
    label
      .closest('[data-slot="form-item"]')
      ?.querySelector<HTMLElement>(
        '[data-slot="form-control"], input, textarea, button[role="switch"], button[role="combobox"], [role="group"]'
      )
  if (!control) {
    throw new Error(`Expected control for label "${labelText}"`)
  }
  return control
}

function getPickerCombobox(): HTMLButtonElement {
  const combobox = getControlByLabel('Group').querySelector<HTMLButtonElement>(
    'button[role="combobox"]'
  )
  if (!combobox) {
    throw new Error('Expected the group picker combobox')
  }
  return combobox
}

// The switch primitive renders its interactive element beside the hidden input
// the form label points at, so the state is read from the switch role itself.
function getSwitch(labelText: string): HTMLElement {
  const formItem = findLabel(labelText)?.closest('[data-slot="form-item"]')
  const control = formItem?.querySelector<HTMLElement>('[role="switch"]')
  if (!control) {
    throw new Error(`Expected a switch for label "${labelText}"`)
  }
  return control
}

function isSwitchChecked(labelText: string): boolean {
  return getSwitch(labelText).getAttribute('aria-checked') === 'true'
}

function changeInput(input: HTMLInputElement, value: string): void {
  fireEvent.input(input, { target: { value } })
}

function selectComboboxOption(
  trigger: HTMLButtonElement,
  optionDescription: string
): void {
  fireEvent.click(trigger)
  const option = [
    ...document.querySelectorAll<HTMLElement>('[data-slot="command-item"]'),
  ].find((candidate) => candidate.textContent?.includes(optionDescription))
  if (!option) {
    throw new Error(`Expected option containing "${optionDescription}"`)
  }
  fireEvent.click(option)
}

function addGroup(optionDescription: string): void {
  selectComboboxOption(getPickerCombobox(), optionDescription)
}

function getPickerOrder(): string[] {
  return [
    ...document.querySelectorAll<HTMLButtonElement>(
      'button[aria-label^="Remove "]'
    ),
  ].map((button) =>
    (button.getAttribute('aria-label') ?? '').replace(/^Remove /, '')
  )
}

function getPreviewOrder(): (string | null)[] {
  return [
    ...document.querySelectorAll<HTMLElement>(
      '[data-slot="global-auto-order-name"]'
    ),
  ].map((item) => item.textContent)
}

afterEach(() => {
  apiClient.get = originalGet
  apiClient.post = originalPost
  apiClient.put = originalPut
  localStorage.clear()
  if (renderedDrawer) {
    renderedDrawer.queryClient.clear()
    renderedDrawer = null
  }
})

describe('API keys mutate drawer group integration', () => {
  test('inherits the root Auto order and sends an empty override for every batch-created key', async () => {
    const fixtures: DrawerFixtures = { created: [], updated: [] }
    installApiFixtures(fixtures)
    await renderDrawer({})

    expect(findLabel('Use the global Auto order')).not.toBeNull()
    expect(isSwitchChecked('Use the global Auto order')).toBe(true)
    expect(
      document.body.textContent?.includes(
        'Using the complete global Auto order (2 groups)'
      )
    ).toBe(true)
    expect(getPreviewOrder()).toEqual(['vip', 'default'])

    changeInput(
      document.querySelector<HTMLInputElement>(
        'input[name="name"]'
      ) as HTMLInputElement,
      'batch'
    )
    changeInput(
      document.querySelector<HTMLInputElement>(
        'input[name="tokenCount"]'
      ) as HTMLInputElement,
      '2'
    )
    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(fixtures.created).toHaveLength(2))

    expect(fixtures.created.length).toBe(2)
    expect(fixtures.created[0]?.name).toBe('batch')
    for (const payload of fixtures.created) {
      expect(payload.group).toBe('auto')
      expect(payload.auto_groups).toEqual([])
      expect(payload.cross_group_retry).toBe(true)
    }
  })

  test('creates a single-group key from the picker without a cross-group switch', async () => {
    const fixtures: DrawerFixtures = { created: [], updated: [] }
    installApiFixtures(fixtures, { defaultUseAutoGroup: false })
    await renderDrawer({ defaultUseAutoGroup: false })

    expect(getPickerOrder()).toEqual([])
    expect(document.body.textContent?.includes('0 / 3 groups selected')).toBe(
      true
    )
    expect(findLabel('Cross-group retry')).toBeNull()

    addGroup('Priority access')
    expect(getPickerOrder()).toEqual(['vip'])
    expect(document.body.textContent?.includes('1 / 3 groups selected')).toBe(
      true
    )
    expect(findLabel('Cross-group retry')).toBeNull()

    changeInput(
      document.querySelector<HTMLInputElement>(
        'input[name="name"]'
      ) as HTMLInputElement,
      'single'
    )
    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(fixtures.created).toHaveLength(1))

    expect(fixtures.created[0]?.group).toBe('vip')
    expect(fixtures.created[0]?.auto_groups).toEqual([])
    expect(fixtures.created[0]?.cross_group_retry).toBe(false)
  })

  test('creates a multi-group key that keeps the chosen order and enables cross-group retry', async () => {
    const fixtures: DrawerFixtures = { created: [], updated: [] }
    installApiFixtures(fixtures, { defaultUseAutoGroup: false })
    await renderDrawer({ defaultUseAutoGroup: false })

    addGroup('Priority access')
    expect(getPickerOrder()).toEqual(['vip'])
    expect(findLabel('Cross-group retry')).toBeNull()

    addGroup('Standard access')
    expect(getPickerOrder()).toEqual(['vip', 'default'])
    expect(document.body.textContent?.includes('2 / 3 groups selected')).toBe(
      true
    )
    expect(isSwitchChecked('Cross-group retry')).toBe(true)

    changeInput(
      document.querySelector<HTMLInputElement>(
        'input[name="name"]'
      ) as HTMLInputElement,
      'multi'
    )
    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(fixtures.created).toHaveLength(1))

    expect(fixtures.created[0]?.group).toBe('auto')
    expect(fixtures.created[0]?.auto_groups).toEqual(['vip', 'default'])
    expect(fixtures.created[0]?.cross_group_retry).toBe(true)
  })

  test('offers the picker but no global Auto toggle when the deployment does not offer auto', async () => {
    const fixtures: DrawerFixtures = { created: [], updated: [] }
    installApiFixtures(fixtures, {
      defaultUseAutoGroup: true,
      offerAuto: false,
    })
    await renderDrawer({ defaultUseAutoGroup: true, offerAuto: false })

    expect(findLabel('Use the global Auto order')).toBeNull()
    expect(getPickerCombobox()).toBeTruthy()

    addGroup('Priority access')
    addGroup('Standard access')

    changeInput(
      document.querySelector<HTMLInputElement>(
        'input[name="name"]'
      ) as HTMLInputElement,
      'without-auto'
    )
    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(fixtures.created).toHaveLength(1))

    expect(fixtures.created[0]?.group).toBe('auto')
    expect(fixtures.created[0]?.auto_groups).toEqual(['vip', 'default'])
    expect(fixtures.created[0]?.cross_group_retry).toBe(true)
  })

  test('saves an empty selection as a key without a group and says so', async () => {
    const fixtures: DrawerFixtures = { created: [], updated: [] }
    installApiFixtures(fixtures, { defaultUseAutoGroup: false })
    await renderDrawer({ defaultUseAutoGroup: false })

    expect(document.body.textContent?.includes('No groups selected')).toBe(true)
    expect(
      document.body.textContent?.includes(
        "Saving with no groups leaves the key on its owner's group."
      )
    ).toBe(true)

    changeInput(
      document.querySelector<HTMLInputElement>(
        'input[name="name"]'
      ) as HTMLInputElement,
      'ungrouped'
    )
    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(fixtures.created).toHaveLength(1))

    expect(fixtures.created[0]?.group).toBe('')
    expect(fixtures.created[0]?.auto_groups).toEqual([])
    expect(fixtures.created[0]?.cross_group_retry).toBe(false)
  })

  test('clears the explicit list when the global Auto toggle is turned on and returns to the picker when it is turned off', async () => {
    const fixtures: DrawerFixtures = { created: [], updated: [] }
    installApiFixtures(fixtures, { defaultUseAutoGroup: false })
    await renderDrawer({ defaultUseAutoGroup: false })

    addGroup('Priority access')
    expect(getPickerOrder()).toEqual(['vip'])

    fireEvent.click(getSwitch('Use the global Auto order'))
    expect(getPreviewOrder()).toEqual(['vip', 'default'])
    expect(document.querySelector('button[aria-label^="Remove "]')).toBeNull()

    fireEvent.click(getSwitch('Use the global Auto order'))
    expect(getPickerOrder()).toEqual([])
    expect(document.body.textContent?.includes('0 / 3 groups selected')).toBe(
      true
    )

    changeInput(
      document.querySelector<HTMLInputElement>(
        'input[name="name"]'
      ) as HTMLInputElement,
      'cleared'
    )
    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(fixtures.created).toHaveLength(1))

    expect(fixtures.created[0]?.group).toBe('')
    expect(fixtures.created[0]?.auto_groups).toEqual([])
  })

  test('round-trips a stored snapshot key unchanged on save', async () => {
    const storedKey = {
      id: 7,
      name: 'stored-multi',
      key: 'sk-stored',
      status: 1,
      remain_quota: 0,
      used_quota: 0,
      unlimited_quota: true,
      expired_time: -1,
      created_time: 1,
      accessed_time: 0,
      group: 'auto',
      auto_groups: ['vip', 'default'],
      cross_group_retry: true,
      model_limits_enabled: false,
      model_limits: '',
      allow_ips: '',
    }
    const fixtures: DrawerFixtures = { created: [], updated: [] }
    installApiFixtures(fixtures, { storedKey })
    await renderDrawer({ currentRow: storedKey })

    expect(getPickerOrder()).toEqual(['vip', 'default'])
    expect(isSwitchChecked('Cross-group retry')).toBe(true)

    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(fixtures.updated).toHaveLength(1))

    expect(fixtures.updated[0]?.id).toBe(7)
    expect(fixtures.updated[0]?.group).toBe('auto')
    expect(fixtures.updated[0]?.auto_groups).toEqual(['vip', 'default'])
    expect(fixtures.updated[0]?.cross_group_retry).toBe(true)
  })

  test('keeps a stored cross-group preference off when a snapshot key is saved unchanged', async () => {
    const storedKey = {
      id: 8,
      name: 'stored-no-retry',
      key: 'sk-stored-2',
      status: 1,
      remain_quota: 0,
      used_quota: 0,
      unlimited_quota: true,
      expired_time: -1,
      created_time: 1,
      accessed_time: 0,
      group: 'auto',
      auto_groups: ['vip', 'default'],
      cross_group_retry: false,
      model_limits_enabled: false,
      model_limits: '',
      allow_ips: '',
    }
    const fixtures: DrawerFixtures = { created: [], updated: [] }
    installApiFixtures(fixtures, { storedKey })
    await renderDrawer({ currentRow: storedKey })

    expect(isSwitchChecked('Cross-group retry')).toBe(false)

    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(fixtures.updated).toHaveLength(1))

    expect(fixtures.updated[0]?.group).toBe('auto')
    expect(fixtures.updated[0]?.auto_groups).toEqual(['vip', 'default'])
    expect(fixtures.updated[0]?.cross_group_retry).toBe(false)
  })

  test('keeps a stored cross-group preference off when a snapshot key is reordered', async () => {
    const storedKey = {
      id: 10,
      name: 'stored-reordered',
      key: 'sk-stored-4',
      status: 1,
      remain_quota: 0,
      used_quota: 0,
      unlimited_quota: true,
      expired_time: -1,
      created_time: 1,
      accessed_time: 0,
      group: 'auto',
      auto_groups: ['vip', 'default'],
      cross_group_retry: false,
      model_limits_enabled: false,
      model_limits: '',
      allow_ips: '',
    }
    const fixtures: DrawerFixtures = { created: [], updated: [] }
    installApiFixtures(fixtures, { storedKey })
    await renderDrawer({ currentRow: storedKey })

    expect(getPickerOrder()).toEqual(['vip', 'default'])
    expect(isSwitchChecked('Cross-group retry')).toBe(false)

    fireEvent.click(findAriaButton('Move default up'))
    expect(getPickerOrder()).toEqual(['default', 'vip'])
    expect(isSwitchChecked('Cross-group retry')).toBe(false)

    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(fixtures.updated).toHaveLength(1))

    expect(fixtures.updated[0]?.id).toBe(10)
    expect(fixtures.updated[0]?.group).toBe('auto')
    expect(fixtures.updated[0]?.auto_groups).toEqual(['default', 'vip'])
    expect(fixtures.updated[0]?.cross_group_retry).toBe(false)
  })

  test('turns the cross-group default on when a stored single-group key gains a second group', async () => {
    const storedKey = {
      id: 11,
      name: 'stored-single',
      key: 'sk-stored-5',
      status: 1,
      remain_quota: 0,
      used_quota: 0,
      unlimited_quota: true,
      expired_time: -1,
      created_time: 1,
      accessed_time: 0,
      group: 'vip',
      auto_groups: [],
      cross_group_retry: false,
      model_limits_enabled: false,
      model_limits: '',
      allow_ips: '',
    }
    const fixtures: DrawerFixtures = { created: [], updated: [] }
    installApiFixtures(fixtures, { storedKey })
    await renderDrawer({ currentRow: storedKey })

    expect(getPickerOrder()).toEqual(['vip'])
    expect(findLabel('Cross-group retry')).toBeNull()

    addGroup('Standard access')
    expect(getPickerOrder()).toEqual(['vip', 'default'])
    expect(isSwitchChecked('Cross-group retry')).toBe(true)

    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(fixtures.updated).toHaveLength(1))

    expect(fixtures.updated[0]?.group).toBe('auto')
    expect(fixtures.updated[0]?.auto_groups).toEqual(['vip', 'default'])
    expect(fixtures.updated[0]?.cross_group_retry).toBe(true)
  })

  test('round-trips a stored global Auto key unchanged on save', async () => {
    const storedKey = {
      id: 9,
      name: 'stored-global-auto',
      key: 'sk-stored-3',
      status: 1,
      remain_quota: 0,
      used_quota: 0,
      unlimited_quota: true,
      expired_time: -1,
      created_time: 1,
      accessed_time: 0,
      group: 'auto',
      auto_groups: [],
      cross_group_retry: true,
      model_limits_enabled: false,
      model_limits: '',
      allow_ips: '',
    }
    const fixtures: DrawerFixtures = { created: [], updated: [] }
    installApiFixtures(fixtures, { storedKey })
    await renderDrawer({ currentRow: storedKey })

    expect(isSwitchChecked('Use the global Auto order')).toBe(true)
    expect(getPreviewOrder()).toEqual(['vip', 'default'])

    fireEvent.click(findButton('Save changes', true))
    await waitFor(() => expect(fixtures.updated).toHaveLength(1))

    expect(fixtures.updated[0]?.group).toBe('auto')
    expect(fixtures.updated[0]?.auto_groups).toEqual([])
    expect(fixtures.updated[0]?.cross_group_retry).toBe(true)
  })
})
