package service

import (
	"context"
	"fmt"

	"github.com/QuantumNous/new-api/logger"
)

// Outcome labels recorded by RecordTelegramBotAudit. These are stable audit
// vocabulary, not HTTP status codes: several map to the same HTTP response
// (for example TelegramBotAuditOutcomeNotLinked and
// TelegramBotAuditOutcomeAccountUnusable both surface as an ordinary 200
// business-logic refusal) so the operator log stays precise even where the
// wire response is deliberately coarse.
const (
	TelegramBotAuditOutcomeSuccess         = "success"
	TelegramBotAuditOutcomeNotLinked       = "not_linked"
	TelegramBotAuditOutcomeAccountUnusable = "account_unusable"
	TelegramBotAuditOutcomeRejected        = "rejected"
	TelegramBotAuditOutcomeNotFound        = "not_found"
	TelegramBotAuditOutcomeRateLimited     = "rate_limited"
	TelegramBotAuditOutcomeError           = "error"
)

// RecordTelegramBotAudit records an operator-visible entry for a
// /api/bot/v1 request that reads or changes account-scoped data
// (specs/telegram/bot-api/spec.md, "User-scoped bot requests are
// auditable"). Callers pass only the Telegram identifier, the resolved
// account id (0 when none was resolved), the endpoint name, and the
// outcome — never the submitted key, the bot service key, or any other
// credential, so an audit entry can never carry key material.
func RecordTelegramBotAudit(ctx context.Context, telegramUserID string, userId int, endpoint, outcome string) {
	logger.LogInfo(ctx, fmt.Sprintf(
		"telegram bot request telegram_user_id=%s user_id=%d endpoint=%s outcome=%s",
		telegramUserID, userId, endpoint, outcome,
	))
}
