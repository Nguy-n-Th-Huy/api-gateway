package setting

import (
	"strings"
)

// The Telegram bot integration lets a separate, first-party bot act on
// behalf of linked accounts through a small, audited surface: reading
// balance/usage, creating SePay top-up orders on the existing rails, and
// issuing (never redeeming) link codes. It authenticates with a single
// service key rather than per-user tokens or mTLS: per-user tokens would
// still be minted on demand from this same key, so the added caching,
// refresh, and invalidation machinery would buy little, and mTLS or an IP
// allowlist would fix the bot's deployment topology to a static address
// before that topology is known. Power is instead removed by subtraction —
// the surface can never unlink, redeem, or mutate a binding — with every
// user-scoped call audited by Telegram identifier and outcome.
var (
	TelegramBotIntegrationEnabled bool
	// TelegramBotServiceKey is the single credential the bot presents as
	// "Authorization: Bot <key>". It is compared with crypto/subtle and is
	// write-only in the admin settings UI, exactly like SePayWebhookApiKey:
	// saving an empty value keeps the stored key.
	TelegramBotServiceKey string
	// TelegramBotCallbackURL and TelegramBotCallbackSecret configure the
	// outbound event channel (credited/expired/linked). Both are optional:
	// while either is empty no event is emitted, but the bot surface itself
	// still serves requests as long as TelegramBotServiceKey is set.
	TelegramBotCallbackURL    string
	TelegramBotCallbackSecret string
	// TelegramHandleRequired makes the Telegram handle field mandatory at
	// registration. It defaults to true through this Go variable
	// initialization rather than a GORM boolean default tag, because MySQL
	// and PostgreSQL normalize `default:true` differently and that makes
	// AutoMigrate re-issue ALTER TABLE on every restart.
	TelegramHandleRequired = true
)

// IsTelegramBotConfigured reports whether the bot integration surface can
// actually serve requests: it must be enabled and hold a non-empty service
// key. The outbound callback address and secret are not required here
// because they only gate emission of best-effort events, not the surface
// itself.
func IsTelegramBotConfigured() bool {
	if !TelegramBotIntegrationEnabled {
		return false
	}
	return strings.TrimSpace(TelegramBotServiceKey) != ""
}

// TelegramBotMissingFields returns the names of the fields required to serve
// bot requests that are currently empty, so an administrator enabling the
// integration without a key is told exactly what is missing.
func TelegramBotMissingFields() []string {
	missing := make([]string, 0, 1)
	if strings.TrimSpace(TelegramBotServiceKey) == "" {
		missing = append(missing, "service key")
	}
	return missing
}

// IsTelegramBotEventDeliveryConfigured reports whether outbound events
// (credited/expired/linked) can be emitted: the integration must be
// configured to serve requests, and both the callback address and its
// signing secret must be set.
func IsTelegramBotEventDeliveryConfigured() bool {
	if !IsTelegramBotConfigured() {
		return false
	}
	return strings.TrimSpace(TelegramBotCallbackURL) != "" && strings.TrimSpace(TelegramBotCallbackSecret) != ""
}
