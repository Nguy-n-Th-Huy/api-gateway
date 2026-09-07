package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
)

// Event types delivered to the Telegram bot's callback address
// (specs/telegram/bot-events/spec.md, "The delivered event set").
const (
	TelegramBotEventCredited = "topup.succeeded"
	TelegramBotEventExpired  = "topup.expired"
	TelegramBotEventLinked   = "account.linked"
)

const (
	// telegramBotEventTimeout bounds a single delivery attempt so a callback
	// address that accepts a connection and never responds cannot hang the
	// background delivery goroutine indefinitely.
	telegramBotEventTimeout = 5 * time.Second
	// telegramBotEventMaxAttempts bounds total delivery attempts so a
	// permanently unreachable callback is abandoned rather than retried
	// forever (specs/telegram/bot-events/spec.md, "Delivery is retried,
	// bounded, and observable").
	telegramBotEventMaxAttempts = 4
	telegramBotEventBaseDelay   = 1 * time.Second
)

// telegramBotEventPayload is the JSON body delivered to the bot's callback
// address. EventID is stable across every retry of one delivery and distinct
// between different events, satisfying "Every event carries a stable
// identifier for idempotency".
type telegramBotEventPayload struct {
	EventID   string         `json:"event_id"`
	Type      string         `json:"type"`
	Timestamp int64          `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

func init() {
	// Wires model.RechargeSePay's post-commit hook to this package's sender
	// without model importing service (which already imports model and would
	// create a cycle). See model.TelegramCreditedEventHook.
	model.TelegramCreditedEventHook = EmitTelegramCreditedEvent
}

// EmitTelegramCreditedEvent emits the "credited" event for a top-up order
// after its wallet credit has already committed. It is a no-op when event
// delivery is not configured or the owning account has no Telegram binding.
func EmitTelegramCreditedEvent(userId int, tradeNo string, creditedAmount int64) {
	telegramID, ok := telegramIdentityForEvent(userId)
	if !ok {
		return
	}
	balance, err := model.GetUserQuota(userId, true)
	if err != nil {
		common.SysError("telegram bot credited event balance lookup failed: " + err.Error())
		return
	}
	emitTelegramBotEvent(TelegramBotEventCredited, map[string]any{
		"telegram_user_id": telegramID,
		"trade_no":         tradeNo,
		"credited_amount":  creditedAmount,
		"balance":          balance,
	})
}

// EmitTelegramExpiredEvent emits the "expired" event for a top-up order the
// expiry sweep has just marked expired. It is a no-op when event delivery is
// not configured or the owning account has no Telegram binding.
func EmitTelegramExpiredEvent(userId int, tradeNo string) {
	telegramID, ok := telegramIdentityForEvent(userId)
	if !ok {
		return
	}
	emitTelegramBotEvent(TelegramBotEventExpired, map[string]any{
		"telegram_user_id": telegramID,
		"trade_no":         tradeNo,
	})
}

// EmitTelegramLinkedEvent emits the "linked" event after a link-code
// redemption has committed a new Telegram binding.
func EmitTelegramLinkedEvent(telegramID, consoleUsername string) {
	telegramID = strings.TrimSpace(telegramID)
	if telegramID == "" || !setting.IsTelegramBotEventDeliveryConfigured() {
		return
	}
	emitTelegramBotEvent(TelegramBotEventLinked, map[string]any{
		"telegram_user_id": telegramID,
		"username":         consoleUsername,
	})
}

// telegramIdentityForEvent resolves the Telegram identifier events should be
// addressed to for userId, reporting ok=false when delivery is unconfigured
// or the account has no Telegram binding — in either case there is no chat to
// notify (specs/telegram/bot-events/spec.md, "Event concerns an unlinked
// account").
func telegramIdentityForEvent(userId int) (telegramID string, ok bool) {
	if !setting.IsTelegramBotEventDeliveryConfigured() {
		return "", false
	}
	var user model.User
	if err := model.DB.Select("telegram_id").Where("id = ?", userId).First(&user).Error; err != nil {
		return "", false
	}
	if user.TelegramId == "" {
		return "", false
	}
	return user.TelegramId, true
}

// emitTelegramBotEvent hands one event to a background goroutine so delivery
// can never delay, roll back, or fail the operation that produced it
// (specs/telegram/bot-events/spec.md, "Delivery failure never damages the
// originating operation").
func emitTelegramBotEvent(eventType string, data map[string]any) {
	if !setting.IsTelegramBotEventDeliveryConfigured() {
		return
	}
	payload := telegramBotEventPayload{
		EventID:   common.NewRequestId(),
		Type:      eventType,
		Timestamp: common.GetTimestamp(),
		Data:      data,
	}
	callbackURL := setting.TelegramBotCallbackURL
	secret := setting.TelegramBotCallbackSecret
	go deliverTelegramBotEventWithRetry(payload, callbackURL, secret)
}

func deliverTelegramBotEventWithRetry(payload telegramBotEventPayload, callbackURL, secret string) {
	body, err := common.Marshal(payload)
	if err != nil {
		common.SysError("telegram bot event marshal failed type=" + payload.Type + " error=" + err.Error())
		return
	}

	delay := telegramBotEventBaseDelay
	var lastErr error
	for attempt := 1; attempt <= telegramBotEventMaxAttempts; attempt++ {
		if err := deliverTelegramBotEventOnce(callbackURL, secret, body); err != nil {
			lastErr = err
			if attempt < telegramBotEventMaxAttempts {
				time.Sleep(delay)
				delay *= 2
			}
			continue
		}
		return
	}
	common.SysError(fmt.Sprintf(
		"telegram bot event abandoned type=%s event_id=%s attempts=%d error=%q",
		payload.Type, payload.EventID, telegramBotEventMaxAttempts, lastErr,
	))
}

func deliverTelegramBotEventOnce(callbackURL, secret string, body []byte) error {
	if err := ValidateSSRFProtectedFetchURL(callbackURL); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), telegramBotEventTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Telegram-Bot-Signature", generateSignature(secret, body))

	client := GetSSRFProtectedHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram bot event delivery failed with status %d", resp.StatusCode)
	}
	return nil
}
