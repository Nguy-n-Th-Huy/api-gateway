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
// Backend endpoints
//
// Both endpoints are being built in parallel on the Go side
// (openspec/changes/add-public-key-check-page). Their exact route and query
// parameter names are centralized here so a rename on the backend is a
// one-line fix on the frontend.
// ============================================================================

/** `POST /api/token/check` — body `{ key }`. See specs/public-key-check. */
export const KEY_CHECK_ENDPOINT = '/api/token/check'

/**
 * Public `GET` endpoint that renders the setup script as plain text. See
 * specs/public-setup-script/spec.md and design.md's "Decisions" table.
 */
export const SETUP_SCRIPT_ENDPOINT = '/api/setup/script'

/** Query parameter names for the setup-script URL. */
export const SETUP_SCRIPT_QUERY_PARAMS = {
  key: 'key',
  application: 'app',
  os: 'os',
} as const

// Setup section application/OS registry lives in `./lib/applications` (kept
// separate to keep this file focused and under the project's ~200-line
// split guideline): `OS_TARGETS`, `OsTarget`, `AppConfig`, `APPLICATIONS`,
// `DEFAULT_APPLICATION_ID`, `DEFAULT_OS_TARGET`.

// ============================================================================
// Model status section
// ============================================================================

/** Presentation thresholds (percent). Not an operator setting yet — see
 * design.md's "Decisions" for why they live in one constant. */
export const MODEL_STATUS_THRESHOLDS = {
  /** At/above this success rate (with recorded traffic) a model is
   * `Operational`; below it, `Down`. Also the bar strip's healthy boundary. */
  OPERATIONAL: 90,
  /** Bar strip's degraded/failing boundary. */
  DEGRADED: 50,
} as const

/** Number of recent-interval bars rendered per model, matching the backend's
 * `recentSuccessRates(..., 3)` window (pkg/perf_metrics/metrics.go). */
export const MODEL_STATUS_BAR_WINDOW = 3

// ============================================================================
// Form validation
// ============================================================================

/** Minimum key length (after trimming) accepted by the lookup form. */
export const KEY_CHECK_MIN_LENGTH = 8

// ============================================================================
// Key masking
// ============================================================================

/** Leading/trailing characters kept visible when masking a key on screen. */
export const KEY_MASK_VISIBLE_CHARS = 4
