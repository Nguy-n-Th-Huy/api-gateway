package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNormalizeTelegramHandle covers the normalization rules directly: trim
// surrounding whitespace, strip a leading "@", and lower-case
// (specs/telegram/account-link/spec.md, "Accounts carry a Telegram handle
// distinct from the Telegram identity").
func TestNormalizeTelegramHandle(t *testing.T) {
	testCases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "leading at sign and surrounding whitespace", raw: "  @JohnDoe  ", want: "johndoe"},
		{name: "mixed case without at sign", raw: "JohnDoe99", want: "johndoe99"},
		{name: "already normalized", raw: "john_doe", want: "john_doe"},
		{name: "empty string stays empty", raw: "", want: ""},
		{name: "whitespace only normalizes to empty", raw: "   ", want: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, NormalizeTelegramHandle(tc.raw))
		})
	}
}

// TestValidateTelegramHandle is a table test over every format rule: 5-32
// characters, letters/digits/underscore only, must start with a letter
// (specs/telegram/account-link/spec.md, "Telegram handle format
// validation"). Inputs are pre-normalized, matching how callers chain
// NormalizeTelegramHandle before validating.
func TestValidateTelegramHandle(t *testing.T) {
	testCases := []struct {
		name       string
		normalized string
		wantErr    error
	}{
		{name: "valid handle", normalized: "john_doe99", wantErr: nil},
		{name: "minimum length accepted", normalized: "abcde", wantErr: nil},
		{name: "maximum length accepted", normalized: "a0123456789012345678901234567890", wantErr: nil}, // 32 chars
		{name: "too short", normalized: "abcd", wantErr: ErrTelegramHandleTooShort},
		{name: "too long", normalized: "a01234567890123456789012345678901", wantErr: ErrTelegramHandleTooLong}, // 33 chars
		{name: "illegal hyphen", normalized: "john-doe", wantErr: ErrTelegramHandleInvalidChars},
		{name: "illegal dot", normalized: "john.doe", wantErr: ErrTelegramHandleInvalidChars},
		{name: "illegal space", normalized: "john doe", wantErr: ErrTelegramHandleInvalidChars},
		{name: "starts with a digit", normalized: "1johndoe", wantErr: ErrTelegramHandleMustStartWithLetter},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTelegramHandle(tc.normalized)
			if tc.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}
