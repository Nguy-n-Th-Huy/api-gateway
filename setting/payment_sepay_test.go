package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSePayOrderExpiryMinutes(t *testing.T) {
	assert.False(t, ValidateSePayOrderExpiryMinutes(0))
	assert.False(t, ValidateSePayOrderExpiryMinutes(-1))
	assert.False(t, ValidateSePayOrderExpiryMinutes(SePayOrderExpiryMinutesMax+1))
	assert.True(t, ValidateSePayOrderExpiryMinutes(1))
	assert.True(t, ValidateSePayOrderExpiryMinutes(30))
	assert.True(t, ValidateSePayOrderExpiryMinutes(SePayOrderExpiryMinutesMax))
}

func TestIsSePayConfiguredRequiresAllFields(t *testing.T) {
	originalEnabled := SePayEnabled
	originalAccount := SePayBankAccountNumber
	originalBankCode := SePayBankCode
	originalHolder := SePayAccountHolder
	originalKey := SePayWebhookApiKey
	t.Cleanup(func() {
		SePayEnabled = originalEnabled
		SePayBankAccountNumber = originalAccount
		SePayBankCode = originalBankCode
		SePayAccountHolder = originalHolder
		SePayWebhookApiKey = originalKey
	})

	set := func(enabled bool, account, bankCode, holder, key string) {
		SePayEnabled = enabled
		SePayBankAccountNumber = account
		SePayBankCode = bankCode
		SePayAccountHolder = holder
		SePayWebhookApiKey = key
	}

	// Disabled integration is never configured.
	set(false, "123", "niclop", "Alice", "key123")
	require.False(t, IsSePayConfigured())

	// Missing each field individually keeps IsSePayConfigured false.
	set(true, "", "niclop", "Alice", "key123")
	require.False(t, IsSePayConfigured())
	set(true, "123", "", "Alice", "key123")
	require.False(t, IsSePayConfigured())
	set(true, "123", "niclop", "", "key123")
	require.False(t, IsSePayConfigured())
	set(true, "123", "niclop", "Alice", "")
	require.False(t, IsSePayConfigured())

	// Whitespace-only fields count as empty.
	set(true, "  ", "niclop", "Alice", "key123")
	require.False(t, IsSePayConfigured())

	// All fields present → configured.
	set(true, "123", "niclop", "Alice", "key123")
	require.True(t, IsSePayConfigured())
}
