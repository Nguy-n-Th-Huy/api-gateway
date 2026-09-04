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
/**
 * Home page constants
 * All hardcoded data for home page sections
 */
import type { TFunction } from 'i18next'

// Layout - Main base classes
export const MAIN_BASE_CLASSES = 'bg-background text-foreground w-full'

// Hero section - AI Applications (Left side)
export const AI_APPLICATIONS = [
  'LobeHub.Color',
  'Dify.Color',
  'OpenWebUI',
  'Cline',
] as const

// Hero section - AI Models (Right side)
export const AI_MODELS = [
  'Qwen.Color',
  'DeepSeek.Color',
  'Doubao.Color',
  'OpenAI',
  'Claude.Color',
  'Gemini.Color',
] as const

// Hero section - Gateway Features
export const GATEWAY_FEATURES = [
  'Cost Tracking',
  'Model Access',
  'Guardrails',
  'Observability',
  'Budgets',
  'Load Balancing',
  'Rate Limiting',
  'Token Mgmt',
  'Prompt Caching',
  'Pass-Through',
] as const

// Stats section - Default statistics
export const DEFAULT_STATS = [
  {
    value: '50',
    suffix: '+',
    description: 'upstream services integrated',
  },
  {
    value: '100',
    suffix: '+',
    description: 'model billing support',
  },
  {
    value: '50',
    suffix: '+',
    description: 'compatible API routes',
  },
  {
    value: '10',
    suffix: '+',
    description: 'scheduling controls',
  },
] as const

// Features section - Default features
export const DEFAULT_FEATURES = [
  {
    title: 'Lightning Fast',
    description:
      'Optimized network architecture ensures millisecond response times',
    iconName: 'Zap',
  },
  {
    title: 'Secure & Reliable',
    description:
      'Enterprise-grade security with comprehensive permission management',
    iconName: 'Shield',
  },
  {
    title: 'Global Coverage',
    description: 'Multi-region deployment for stable global access',
    iconName: 'Globe',
  },
  {
    title: 'Developer Friendly',
    description: 'Compatible API routes for common AI application workflows',
    iconName: 'Code',
  },
  {
    title: 'High Performance',
    description: 'Support for high concurrency with automatic load balancing',
    iconName: 'Gauge',
  },
  {
    title: 'Transparent Billing',
    description: 'Pay-as-you-go with real-time usage monitoring',
    iconName: 'DollarSign',
  },
  {
    title: 'Team Collaboration',
    description: 'Multi-user management with flexible permission allocation',
    iconName: 'Users',
  },
  {
    title: 'Open Source',
    description: 'Community driven, self-hosted, and extensible',
    iconName: 'HeartHandshake',
  },
] as const

export function getGatewayFeatures(t: TFunction) {
  return GATEWAY_FEATURES.map((feature) => t(feature))
}

export function getDefaultStats(t: TFunction) {
  return DEFAULT_STATS.map((stat) => ({
    ...stat,
    description: stat.description ? t(stat.description) : undefined,
  }))
}

export function getDefaultFeatures(t: TFunction) {
  return DEFAULT_FEATURES.map((feature) => ({
    ...feature,
    title: t(feature.title),
    description: t(feature.description),
  }))
}

// Hero section - trust bullets shown under the primary actions
export const HERO_TRUST_BULLETS = [
  'No maintenance fee',
  'No international card required',
  'Per-request usage logs',
] as const

export function getHeroTrustBullets(t: TFunction) {
  return HERO_TRUST_BULLETS.map((bullet) => t(bullet))
}

// Providers section - upstream model families the gateway routes to.
// Brand names are left untranslated per the existing hero app-pill precedent.
export const PROVIDER_NAMES = [
  'OpenAI',
  'Anthropic',
  'Gemini',
  'Azure OpenAI',
  'AWS Bedrock',
  'DeepSeek',
  'Qwen',
  'Mistral',
  'xAI Grok',
  'Cohere',
] as const

// Additional providers beyond the named strip, matching the "40+ upstream
// providers" figure already documented in the project's own AGENTS.md.
export const PROVIDER_STRIP_MORE_COUNT = 30

// SePay top-up section - ordered transfer-flow steps (no amounts, account
// numbers or transfer codes: those are only available to authenticated users)
export const SEPAY_TOPUP_STEPS = [
  {
    title: 'Scan VietQR from any banking app',
    description:
      'Choose a top-up amount, then scan the VietQR code shown on screen with any banking app.',
  },
  {
    title: 'SePay detects the transfer automatically',
    description:
      'SePay watches for the incoming bank transaction and notifies the gateway the moment it clears.',
  },
  {
    title: 'Quota lands in your account',
    description:
      'The balance is credited automatically — no receipt to send, no manual review.',
  },
] as const

export function getSepayTopupSteps(t: TFunction) {
  return SEPAY_TOPUP_STEPS.map((step) => ({
    title: t(step.title),
    description: t(step.description),
  }))
}

// Integrations section - client applications the gateway plugs into
export const INTEGRATION_APPS = [
  {
    name: 'Cherry Studio',
    description: 'Pick the "OpenAI" provider and paste in the base URL',
  },
  {
    name: 'Cursor / VS Code',
    description: 'Override the OpenAI base URL in Settings',
  },
  {
    name: 'CC Switch',
    description: 'Add a profile using the Claude Messages format',
  },
  {
    name: 'n8n / Dify / LobeChat',
    description: 'Use the built-in OpenAI node and change only the endpoint',
  },
] as const

export function getIntegrationApps(t: TFunction) {
  return INTEGRATION_APPS.map((app) => ({
    name: app.name,
    description: t(app.description),
  }))
}

// Comparison section - direct provider access vs. going through the gateway
export const COMPARISON_ROWS = [
  {
    criterion: 'Payment',
    direct: 'International Visa / Mastercard',
    gateway: 'Domestic bank transfer via VietQR',
  },
  {
    criterion: 'Providers',
    direct: 'One provider, one account each',
    gateway: '40+ providers behind a single key',
  },
  {
    criterion: 'When a provider fails',
    direct: 'Handled manually in application code',
    gateway: 'Automatic failover to a backup channel',
  },
  {
    criterion: 'Cost per project',
    direct: 'One combined invoice',
    gateway: 'Split by API key and by group',
  },
  {
    criterion: 'Where data is stored',
    direct: "On the provider's servers",
    gateway: 'On a server you host yourself',
  },
] as const

export function getComparisonRows(t: TFunction) {
  return COMPARISON_ROWS.map((row) => ({
    criterion: t(row.criterion),
    direct: t(row.direct),
    gateway: t(row.gateway),
  }))
}

// Features section - core bento cards
export const CORE_FEATURES = [
  {
    id: 'failover',
    title: 'Automatic failover when a provider errors',
    description:
      'Multiple weighted channels per model. A channel returning 429 or a 5xx is skipped automatically and the request moves to the next channel — callers never see the interruption.',
  },
  {
    id: 'tokens',
    title: 'Tokens & limits',
    description:
      'A separate key per project, each with its own quota and expiry, revocable instantly.',
  },
  {
    id: 'logs',
    title: 'Transparent request logs',
    description:
      'Every request records the model, tokens in and out, the channel that served it, and the amount charged.',
  },
  {
    id: 'protocols',
    title: 'One gateway, several client protocols',
    description:
      'Clients written for the OpenAI, Claude Messages, or Gemini format all work unchanged. Switching providers does not mean rewriting application code.',
  },
] as const

export function getCoreFeatures(t: TFunction) {
  return CORE_FEATURES.map((feature) => ({
    id: feature.id,
    title: t(feature.title),
    description: t(feature.description),
  }))
}

// Features section - supporting feature tiles
export const ADDITIONAL_GATEWAY_FEATURES = [
  {
    id: 'load',
    title: 'Handles high load',
    description: 'Redis-backed caching with per-IP and per-key rate limiting',
    iconName: 'Zap',
  },
  {
    id: 'billing',
    title: 'Precise billing',
    description:
      'Quota is pre-consumed, then settled once real usage is known',
    iconName: 'Gauge',
  },
  {
    id: 'groups',
    title: 'Group-based management',
    description: 'Split users into groups, each with its own price ratio',
    iconName: 'Users',
  },
  {
    id: 'opensource',
    title: 'Open source',
    description: 'Self-host on your own server — your data stays with you',
    iconName: 'HeartHandshake',
  },
] as const

export function getAdditionalGatewayFeatures(t: TFunction) {
  return ADDITIONAL_GATEWAY_FEATURES.map((feature) => ({
    ...feature,
    title: t(feature.title),
    description: t(feature.description),
  }))
}
