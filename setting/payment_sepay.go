package setting

import (
	"strings"
)

// SePay is the single external payment provider: Vietnamese domestic bank
// transfer with VietQR payment codes and balance-change webhooks. There is no
// hosted checkout and no redirect; the user transfers from their own banking
// app and SePay reports the matching incoming transfer via webhook.
var (
	SePayEnabled           bool
	SePayBankAccountNumber string
	SePayBankCode          string
	SePayAccountHolder     string
	SePayWebhookApiKey     string
	SePayMinTopUp          int = 1
	// SePayOrderExpiryMinutes is how long a pending SePay order stays payable
	// before the expiry sweep marks it expired. The deadline of an order is
	// derived from create_time + this window at check time, so changing the
	// window moves the deadline of already-pending orders (the payable amount
	// stays frozen at creation).
	SePayOrderExpiryMinutes int = defaultSePayOrderExpiryMinutes
)

const (
	defaultSePayOrderExpiryMinutes = 30
	// SePayOrderExpiryMinutesMin / Max bound the configurable expiry window so
	// an operator cannot set a zero/negative deadline or an unreasonably long
	// one (D4).
	SePayOrderExpiryMinutesMin = 1
	SePayOrderExpiryMinutesMax = 7 * 24 * 60 // 7 days
)

// IsSePayConfigured reports whether SePay can actually accept orders: it must
// be enabled and every credential/bank field required to render a payment and
// verify a webhook must be non-empty. A partially configured gateway is
// treated as unavailable so it never accepts orders.
func IsSePayConfigured() bool {
	if !SePayEnabled {
		return false
	}
	return strings.TrimSpace(SePayBankAccountNumber) != "" &&
		strings.TrimSpace(SePayBankCode) != "" &&
		strings.TrimSpace(SePayAccountHolder) != "" &&
		strings.TrimSpace(SePayWebhookApiKey) != ""
}

// ValidateSePayOrderExpiryMinutes rejects a zero, negative, or above-maximum
// expiry window. Callers must keep the previously stored value on failure.
func ValidateSePayOrderExpiryMinutes(minutes int) bool {
	return minutes >= SePayOrderExpiryMinutesMin && minutes <= SePayOrderExpiryMinutesMax
}

// SePayMissingFields returns the names of required SePay settings that are
// empty. It is used to tell the administrator exactly what is missing when
// they enable an incompletely configured gateway.
func SePayMissingFields() []string {
	missing := make([]string, 0, 4)
	if strings.TrimSpace(SePayBankAccountNumber) == "" {
		missing = append(missing, "bank account number")
	}
	if strings.TrimSpace(SePayBankCode) == "" {
		missing = append(missing, "bank code")
	}
	if strings.TrimSpace(SePayAccountHolder) == "" {
		missing = append(missing, "account holder")
	}
	if strings.TrimSpace(SePayWebhookApiKey) == "" {
		missing = append(missing, "webhook API key")
	}
	return missing
}
