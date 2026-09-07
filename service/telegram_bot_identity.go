package service

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"gorm.io/gorm"
)

// ErrTelegramBotUserNotLinked and ErrTelegramBotAccountUnusable are the two
// distinct outcomes ResolveTelegramBotUser reports for an account-scoped
// /api/bot/v1 request, per specs/telegram/bot-api/spec.md ("Endpoints that
// need an account report an unlinked Telegram user distinguishably"): a
// Telegram identifier bound to nothing, and one bound to an account that is
// disabled or deleted, are never confused with each other.
var (
	ErrTelegramBotUserNotLinked    = errors.New("telegram user is not linked to any account")
	ErrTelegramBotAccountUnusable  = errors.New("telegram-linked account is disabled or deleted")
	ErrTelegramBotIdentityRequired = errors.New("telegram_user_id is required")
)

// ResolveTelegramBotUser resolves a telegram_user_id to the account it is
// bound to, distinguishing "bound to nothing" from "bound to an account that
// can no longer be served" so account-scoped handlers can report each
// outcome distinctly (see the errors above) without ever leaking balance,
// key, order, or log data for either.
func ResolveTelegramBotUser(telegramUserID string) (*model.User, error) {
	telegramUserID = strings.TrimSpace(telegramUserID)
	if telegramUserID == "" {
		return nil, ErrTelegramBotIdentityRequired
	}

	var user model.User
	err := model.DB.Unscoped().Where("telegram_id = ?", telegramUserID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTelegramBotUserNotLinked
		}
		return nil, err
	}
	if user.DeletedAt.Valid || user.Status != common.UserStatusEnabled {
		return nil, ErrTelegramBotAccountUnusable
	}
	return &user, nil
}
