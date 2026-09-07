package model

import "strings"

const (
	// TelegramHandleMinLength and TelegramHandleMaxLength bound a normalized
	// handle, matching Telegram's own username length rules.
	TelegramHandleMinLength = 5
	TelegramHandleMaxLength = 32
)

// NormalizeTelegramHandle trims surrounding whitespace, removes a leading
// "@", and lower-cases the result. It never fails: an input that normalizes
// to the empty string means no handle was declared.
func NormalizeTelegramHandle(raw string) string {
	normalized := strings.TrimSpace(raw)
	normalized = strings.TrimPrefix(normalized, "@")
	return strings.ToLower(normalized)
}

// ValidateTelegramHandle accepts an already-normalized handle only when it is
// 5 to 32 characters long, contains only letters, digits, and underscores,
// and begins with a letter. It returns a distinct error per failed rule so
// the caller can report which rule failed.
func ValidateTelegramHandle(normalized string) error {
	if len(normalized) < TelegramHandleMinLength {
		return ErrTelegramHandleTooShort
	}
	if len(normalized) > TelegramHandleMaxLength {
		return ErrTelegramHandleTooLong
	}
	first := normalized[0]
	if first < 'a' || first > 'z' {
		return ErrTelegramHandleMustStartWithLetter
	}
	for _, r := range normalized {
		isLower := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if !isLower && !isDigit && r != '_' {
			return ErrTelegramHandleInvalidChars
		}
	}
	return nil
}
