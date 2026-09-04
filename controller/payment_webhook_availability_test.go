package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/require"
)

func confirmPaymentComplianceForTest(t *testing.T) {
	t.Helper()
	paymentSetting := operation_setting.GetPaymentSetting()
	originalConfirmed := paymentSetting.ComplianceConfirmed
	originalTermsVersion := paymentSetting.ComplianceTermsVersion
	t.Cleanup(func() {
		paymentSetting.ComplianceConfirmed = originalConfirmed
		paymentSetting.ComplianceTermsVersion = originalTermsVersion
	})
	paymentSetting.ComplianceConfirmed = true
	paymentSetting.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
}

func setSePaySettingsForTest(t *testing.T) {
	t.Helper()
	originalEnabled := setting.SePayEnabled
	originalAccount := setting.SePayBankAccountNumber
	originalBankCode := setting.SePayBankCode
	originalHolder := setting.SePayAccountHolder
	originalKey := setting.SePayWebhookApiKey
	t.Cleanup(func() {
		setting.SePayEnabled = originalEnabled
		setting.SePayBankAccountNumber = originalAccount
		setting.SePayBankCode = originalBankCode
		setting.SePayAccountHolder = originalHolder
		setting.SePayWebhookApiKey = originalKey
	})
	setting.SePayEnabled = true
	setting.SePayBankAccountNumber = "987654321"
	setting.SePayBankCode = "niclop"
	setting.SePayAccountHolder = "NGUYEN VAN A"
	setting.SePayWebhookApiKey = "sepay_test_key"
}

func TestSePayTopUpEnabledRequiresComplianceAndConfig(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	setSePaySettingsForTest(t)

	require.True(t, isSePayTopUpEnabled())

	// Incomplete configuration (missing account holder) disables the gateway.
	setting.SePayAccountHolder = "  "
	require.False(t, isSePayTopUpEnabled())

	setting.SePayAccountHolder = "NGUYEN VAN A"
	setting.SePayEnabled = false
	require.False(t, isSePayTopUpEnabled())
}

func TestSePayTopUpDisabledWithoutCompliance(t *testing.T) {
	paymentSetting := operation_setting.GetPaymentSetting()
	originalConfirmed := paymentSetting.ComplianceConfirmed
	originalTermsVersion := paymentSetting.ComplianceTermsVersion
	t.Cleanup(func() {
		paymentSetting.ComplianceConfirmed = originalConfirmed
		paymentSetting.ComplianceTermsVersion = originalTermsVersion
	})
	paymentSetting.ComplianceConfirmed = false
	paymentSetting.ComplianceTermsVersion = ""

	setSePaySettingsForTest(t)
	require.False(t, isSePayTopUpEnabled())
}

func TestSePayWebhookEnabledRequiresTopUpAndApiKey(t *testing.T) {
	confirmPaymentComplianceForTest(t)
	setSePaySettingsForTest(t)

	require.True(t, isSePayWebhookEnabled())

	setting.SePayWebhookApiKey = ""
	require.False(t, isSePayWebhookEnabled())
}
