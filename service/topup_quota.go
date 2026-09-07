package service

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/shopspring/decimal"
)

// TopUpQuotaFromAmount converts a requested top-up amount (in the unit the
// current quota display type uses) into the quota it credits, truncating a
// tokens-display amount to a whole QuotaPerUnit multiple exactly as
// settlement does, and applying the same overflow-safe wallet conversion
// used everywhere else quota is derived from money.
func TopUpQuotaFromAmount(amount int64) (int, error) {
	quota := decimal.NewFromInt(amount)
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		quotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		quota = decimal.NewFromInt(quota.Div(quotaPerUnit).IntPart()).Mul(quotaPerUnit)
	} else {
		quota = quota.Mul(decimal.NewFromFloat(common.QuotaPerUnit))
	}
	return common.WalletQuotaFromDecimalStrict(quota)
}

// MaxTopUpAmount is the largest single top-up amount that can still be
// represented as a wallet quota without saturating, in the unit the current
// quota display type uses.
func MaxTopUpAmount() int64 {
	if common.QuotaPerUnit <= 0 {
		return 0
	}
	quotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
	maxStoredAmount := decimal.NewFromInt(common.MaxWalletQuota).
		Div(quotaPerUnit).
		Floor()
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		return maxStoredAmount.Add(decimal.NewFromInt(1)).
			Mul(quotaPerUnit).
			Ceil().
			Sub(decimal.NewFromInt(1)).
			IntPart()
	}
	return maxStoredAmount.IntPart()
}

// ValidateTopUpQuota is the single producer for "is this top-up amount
// representable" across every top-up surface (the console top-up handler and
// the SePay order-creation chain): it converts the amount to a credited
// quota and, on failure, reports the per-order maximum so the caller can
// show it to the requester.
func ValidateTopUpQuota(amount int64) (int, error) {
	quota, err := TopUpQuotaFromAmount(amount)
	if err == nil && quota > 0 {
		return quota, nil
	}
	maxAmount := MaxTopUpAmount()
	if maxAmount > 0 && amount > maxAmount {
		return 0, fmt.Errorf("单笔充值数量不能大于 %d", maxAmount)
	}
	return 0, errors.New("充值数量无效")
}
