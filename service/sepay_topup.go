package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/shopspring/decimal"
)

// SePayMaxTopUpAmount is the per-order upper bound on a SePay top-up amount,
// independent of the wallet-capacity ceiling: it caps a single order
// regardless of how much headroom the wallet still has.
const SePayMaxTopUpAmount = int64(9999)

// ErrPaymentComplianceRequired and ErrSePayNotConfigured are returned by
// CreateSePayTopUpOrder so a caller can translate them into its own
// user-facing message; every other rejection's Error() text is already the
// message shown to the caller, matching the pre-extraction handler.
var (
	ErrPaymentComplianceRequired = errors.New("payment compliance confirmation required")
	ErrSePayNotConfigured        = errors.New("SePay 未配置")
)

// SePayEffectiveMinTopUp is the minimum SePay top-up amount: the
// SePay-specific override when set, otherwise the general top-up minimum.
func SePayEffectiveMinTopUp() int {
	if setting.SePayMinTopUp > 0 {
		return setting.SePayMinTopUp
	}
	return operation_setting.MinTopUp
}

// SePayPayMoneyFromDecimal applies the payable-amount formula shared by
// every SePay-priced order: payable_vnd = round(amount x price x
// topup_group_ratio x discount). Discount tiers are keyed by preset integer
// amounts; only an exact match applies, so plans and non-preset amounts skip
// the discount.
func SePayPayMoneyFromDecimal(amount decimal.Decimal, group string) (float64, int64, error) {
	topupGroupRatio := common.GetTopupGroupRatio(group)
	if topupGroupRatio == 0 {
		topupGroupRatio = 1
	}
	dPrice := decimal.NewFromFloat(operation_setting.Price)
	dRatio := decimal.NewFromFloat(topupGroupRatio)
	discount := 1.0
	if amtInt, exact := amount.Float64(); exact {
		if ds, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(amtInt)]; ok && ds > 0 {
			discount = ds
		}
	}
	dDiscount := decimal.NewFromFloat(discount)
	payMoneyDec := amount.Mul(dPrice).Mul(dRatio).Mul(dDiscount)
	payableVND := payMoneyDec.Round(0).IntPart()
	if payableVND <= 0 {
		return 0, 0, errors.New("充值金额过低")
	}
	payMoneyFloat, _ := payMoneyDec.Round(0).Float64()
	return payMoneyFloat, payableVND, nil
}

// SePayPayMoneyFromAmount is the top-up path of the payable-amount
// conversion; the subscription-plan price path uses SePayPayMoneyFromDecimal
// directly so both share one formula.
func SePayPayMoneyFromAmount(amount int64, group string) (float64, int64, error) {
	dAmount := decimal.NewFromInt(amount)
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount = dAmount.Div(decimal.NewFromFloat(common.QuotaPerUnit))
	}
	return SePayPayMoneyFromDecimal(dAmount, group)
}

// CreateSePayTopUpOrder is the single producer for SePay top-up order
// creation: the console web handler and the Telegram bot integration
// surface both call this function so a bot-initiated order is indistinguishable
// from a console-created order in every respect that affects money. It
// preserves, in order, every step the original handler applied: the payment
// compliance gate, the SePay configuration gate, the minimum-amount check,
// the per-order maximum, quota conversion via ValidateTopUpQuota, the
// wallet-capacity check, the user group lookup, currency conversion, the
// tokens-display stored-amount handling, and order insertion with a unique
// trade number.
func CreateSePayTopUpOrder(ctx context.Context, userId int, amount int64) (*model.TopUp, float64, int64, error) {
	if !operation_setting.IsPaymentComplianceConfirmed() {
		return nil, 0, 0, ErrPaymentComplianceRequired
	}
	if !setting.IsSePayConfigured() {
		return nil, 0, 0, ErrSePayNotConfigured
	}

	minTopUp := SePayEffectiveMinTopUp()
	if amount < int64(minTopUp) {
		return nil, 0, 0, fmt.Errorf("充值数量不能小于 %d", minTopUp)
	}
	if amount > SePayMaxTopUpAmount {
		return nil, 0, 0, fmt.Errorf("单笔充值数量不能大于 %d", SePayMaxTopUpAmount)
	}

	creditedQuota, err := ValidateTopUpQuota(amount)
	if err != nil {
		return nil, 0, 0, err
	}
	if err := model.ValidateTopUpQuotaCapacity(userId, creditedQuota); err != nil {
		return nil, 0, 0, err
	}

	group, err := model.GetUserGroup(userId, true)
	if err != nil {
		return nil, 0, 0, errors.New("获取用户分组失败")
	}
	payMoney, payableVND, err := SePayPayMoneyFromAmount(amount, group)
	if err != nil {
		return nil, 0, 0, err
	}

	// Amount column stores the display amount: tokens mode stores
	// tokens/QuotaPerUnit, matching settlement's expectation.
	storedAmount := amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		storedAmount = decimal.NewFromInt(amount).
			Div(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
	}

	topUp := &model.TopUp{
		UserId:          userId,
		Amount:          storedAmount,
		Money:           payMoney,
		PaymentMethod:   model.PaymentMethodSePay,
		PaymentProvider: model.PaymentProviderSePay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := model.InsertSePayTopUp(topUp); err != nil {
		logger.LogError(ctx, fmt.Sprintf("SePay 创建充值订单失败 user_id=%d amount=%d error=%q", userId, amount, err.Error()))
		return nil, 0, 0, errors.New("创建订单失败")
	}
	return topUp, payMoney, payableVND, nil
}
