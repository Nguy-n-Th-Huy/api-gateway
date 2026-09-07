package model

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// TelegramLinkCodeTTL is how long an issued link code stays redeemable
// (specs/telegram/account-link/spec.md, "Link codes carry a linking intent
// from a chat to a browser session": "expire within a short window").
const TelegramLinkCodeTTL = 5 * time.Minute

// ErrTelegramLinkAlreadyBound is returned by IssueTelegramLinkCode when the
// requested Telegram account identifier is already bound to an account.
var ErrTelegramLinkAlreadyBound = errors.New("telegram account is already linked")

// TelegramLinkPayload is the JSON stored in the issuing AuthFlow's Payload
// column. It carries the Telegram account identifier the code was issued for
// and, when the bot reported one, that account's Telegram handle, so
// redemption can overwrite the stored (self-declared, unverified) handle with
// the verified one Telegram reports, or clear it when Telegram reports none
// (specs/telegram/account-link/spec.md, "A stored handle is untrusted until
// the account is linked").
type TelegramLinkPayload struct {
	TelegramUserID   string `json:"telegram_user_id"`
	TelegramUsername string `json:"telegram_username,omitempty"`
}

// IssueTelegramLinkCode issues a single-use, short-lived link code for a
// Telegram account identifier that is not already bound to any account.
// Issuing a code never creates, changes, or removes a binding by itself — the
// binding is written only on redemption (controller.TelegramLinkRedeem),
// reachable at POST /api/user/telegram/link/confirm.
func IssueTelegramLinkCode(telegramUserID, telegramUsername string) (code string, expiresAt time.Time, err error) {
	telegramUserID = strings.TrimSpace(telegramUserID)
	if telegramUserID == "" {
		return "", time.Time{}, ErrAuthFlowInvalid
	}
	if IsTelegramIdAlreadyTaken(telegramUserID) {
		return "", time.Time{}, ErrTelegramLinkAlreadyBound
	}

	payload := TelegramLinkPayload{
		TelegramUserID:   telegramUserID,
		TelegramUsername: NormalizeTelegramHandle(telegramUsername),
	}
	payloadBytes, err := common.Marshal(payload)
	if err != nil {
		return "", time.Time{}, err
	}

	rawCode, err := GenerateTelegramLinkCode()
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt = time.Now().Add(TelegramLinkCodeTTL)
	if _, _, err := CreateAuthFlow(AuthFlowCreate{
		Purpose:   AuthFlowPurposeTelegramLink,
		Provider:  ExternalIdentityProviderTelegram,
		Payload:   string(payloadBytes),
		ExpiresAt: expiresAt,
		Token:     rawCode,
	}); err != nil {
		return "", time.Time{}, err
	}
	return rawCode, expiresAt, nil
}
