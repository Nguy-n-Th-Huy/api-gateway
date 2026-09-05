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

// ============================================================================
// Setup section — supported applications and operating systems
// ============================================================================

export const OS_TARGETS = {
  WINDOWS: 'windows',
  UNIX: 'unix',
} as const

export type OsTarget = (typeof OS_TARGETS)[keyof typeof OS_TARGETS]

export interface AppModelSlot {
  /** Also the setup-script query parameter suffix (`model_<id>`) and the key
   * used to track the visitor's per-slot model selection. */
  id: string
  labelKey: string
}

export interface AppInstallStep {
  command: string
  runtimeNoteKey?: string
}

export interface AppConfig {
  id: string
  labelKey: string
  /** Empty for an application with no documented model-role configuration
   * (Pi, Oh My Pi) — its script registers every model the group enables. */
  slots: AppModelSlot[]
  install: Record<OsTarget, AppInstallStep>
}

/**
 * NOTE on install commands: verified against each project's official source
 * during this change's research phase (2026-09-05):
 * - Claude Code — code.claude.com/docs/en/setup. Native installer needs no
 *   runtime; only its npm alternative (not shown here) needs Node.js 22+.
 * - Codex CLI — github.com/openai/codex. The native installers ship a
 *   standalone Rust binary; no Node.js needed. (Its npm alternative needs
 *   Node.js 22+, a community-sourced figure, not shown here.)
 * - OpenCode — opencode.ai/docs (repo moved from sst/opencode to
 *   anomalyco/opencode). No documented minimum Node version for the
 *   Windows npm path.
 * - Pi — github.com/earendil-works/pi, package
 *   `@earendil-works/pi-coding-agent`, binary `pi`. `--ignore-scripts` is
 *   part of the documented command; no runtime version documented.
 * - Oh My Pi — github.com/can1357/oh-my-pi, package
 *   `@oh-my-pi/pi-coding-agent`, binary `omp`. The native installer needs no
 *   Node.js (a separate, un-shown Bun install path needs Bun >= 1.3.14).
 * None of the five documents a runtime requirement for the CLI's own install
 * command shown here.
 *
 * `runtimeNoteKey` on every macOS/Linux (`unix`) step below is unrelated to
 * that: it discloses that step 2, "Configure the CLI" (the generated
 * one-liner rendered by `setup-command.ts`), still runs the config-file
 * merge as a small embedded Node.js program on that platform and therefore
 * needs Node.js on PATH — see design.md — Decisions. The Windows step never
 * carries this note: the Windows configure command is native PowerShell with
 * no runtime dependency.
 *
 * Kept in sync with the backend's own `setupScriptApps` registry
 * (`controller/setup_script.go`), which records the same commands for
 * completeness even though it does not serve them — the frontend renders
 * its own install step independently.
 */
export const APPLICATIONS: AppConfig[] = [
  {
    id: 'claude-code',
    labelKey: 'Claude Code',
    slots: [
      { id: 'opus', labelKey: 'Opus model' },
      { id: 'sonnet', labelKey: 'Sonnet model' },
      { id: 'haiku', labelKey: 'Haiku model' },
      { id: 'subagent', labelKey: 'Subagent model' },
    ],
    install: {
      [OS_TARGETS.WINDOWS]: {
        command: 'irm https://claude.ai/install.ps1 | iex',
      },
      [OS_TARGETS.UNIX]: {
        command: 'curl -fsSL https://claude.ai/install.sh | bash',
        runtimeNoteKey: 'The configure command below requires Node.js',
      },
    },
  },
  {
    id: 'codex',
    labelKey: 'Codex',
    slots: [
      { id: 'small', labelKey: 'Small model' },
      { id: 'medium', labelKey: 'Medium model' },
      { id: 'large', labelKey: 'Large model' },
    ],
    install: {
      [OS_TARGETS.WINDOWS]: {
        command:
          'powershell -ExecutionPolicy ByPass -c "irm https://chatgpt.com/codex/install.ps1 | iex"',
      },
      [OS_TARGETS.UNIX]: {
        command: 'curl -fsSL https://chatgpt.com/codex/install.sh | sh',
        runtimeNoteKey: 'The configure command below requires Node.js',
      },
    },
  },
  {
    id: 'opencode',
    labelKey: 'OpenCode',
    slots: [
      { id: 'model', labelKey: 'Main model' },
      { id: 'small_model', labelKey: 'Small model' },
    ],
    install: {
      [OS_TARGETS.WINDOWS]: {
        command: 'npm install -g opencode-ai',
      },
      [OS_TARGETS.UNIX]: {
        command: 'curl -fsSL https://opencode.ai/install | bash',
        runtimeNoteKey: 'The configure command below requires Node.js',
      },
    },
  },
  {
    id: 'pi',
    labelKey: 'Pi',
    slots: [],
    install: {
      [OS_TARGETS.WINDOWS]: {
        command: 'npm install -g --ignore-scripts @earendil-works/pi-coding-agent',
      },
      [OS_TARGETS.UNIX]: {
        command: 'npm install -g --ignore-scripts @earendil-works/pi-coding-agent',
        runtimeNoteKey: 'The configure command below requires Node.js',
      },
    },
  },
  {
    id: 'oh-my-pi',
    labelKey: 'Oh My Pi',
    slots: [],
    install: {
      [OS_TARGETS.WINDOWS]: {
        command: 'irm https://omp.sh/install.ps1 | iex',
      },
      [OS_TARGETS.UNIX]: {
        command: 'curl -fsSL https://omp.sh/install | sh',
        runtimeNoteKey: 'The configure command below requires Node.js',
      },
    },
  },
]

export const DEFAULT_APPLICATION_ID = APPLICATIONS[0].id
export const DEFAULT_OS_TARGET: OsTarget = OS_TARGETS.UNIX
