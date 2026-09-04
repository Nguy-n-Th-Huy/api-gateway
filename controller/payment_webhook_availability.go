package controller

import (
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func isPaymentComplianceConfirmed() bool {
	return operation_setting.IsPaymentComplianceConfirmed()
}

// isSePayTopUpEnabled reports whether the SePay integration is available to
// users: compliance confirmed AND every required SePay setting populated.
func isSePayTopUpEnabled() bool {
	if !isPaymentComplianceConfirmed() {
		return false
	}
	return setting.IsSePayConfigured()
}

// isSePayWebhookConfigured reports whether SePay can receive and verify
// webhooks: a configured non-empty API key.
func isSePayWebhookConfigured() bool {
	return setting.SePayWebhookApiKey != ""
}

// isSePayWebhookEnabled reports whether the webhook endpoint should accept
// deliveries: SePay enabled AND the API key configured AND compliance
// confirmed.
func isSePayWebhookEnabled() bool {
	return isSePayTopUpEnabled() && isSePayWebhookConfigured()
}
