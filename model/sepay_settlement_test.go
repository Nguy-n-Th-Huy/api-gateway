package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createSePayOrderWithAmount inserts a SePay top-up with a frozen payable VND
// (whole Dong, per D5) and a given create_time, mirroring the order row the
// SePay creation endpoint produces.
func createSePayOrderWithAmount(t *testing.T, userId int, tradeNo string, provider string, status string, money float64, createTime int64) TopUp {
	t.Helper()
	topUp := TopUp{
		UserId:          userId,
		Amount:          2,
		Money:           money,
		TradeNo:         tradeNo,
		PaymentMethod:   PaymentMethodSePay,
		PaymentProvider: provider,
		CreateTime:      createTime,
		Status:          status,
	}
	require.NoError(t, DB.Create(&topUp).Error)
	return topUp
}

func TestGenerateSePayTradeNoFormat(t *testing.T) {
	for _, prefix := range []string{SePayMemoPrefixTopUp, SePayMemoPrefixSubscription} {
		memo := GenerateSePayTradeNo(prefix)
		assert.True(t, len(memo) == len(prefix)+sePayMemoRandomLen, "expected total length %d, got %d", len(prefix)+sePayMemoRandomLen, len(memo))
		assert.True(t, memo[:len(prefix)] == prefix, "expected prefix %s, got %s", prefix, memo[:len(prefix)])
		for _, r := range memo[len(prefix):] {
			assert.True(t, (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'),
				"memo segment must be uppercase A-Z0-9, found %q in %s", string(r), memo)
		}
	}
}

func TestRechargeSePayOverpaymentCreditsExactlyOrderedQuota(t *testing.T) {
	truncateTables(t)

	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })

	user := insertUserForPaymentGuardTest(t, 601, 0)
	order := createSePayTestOrder(t, user.Id, "SEPAYTESTOVERPAY", PaymentProviderSePay, common.TopUpStatusPending)

	// Transfer 10x the payable amount; exactly the ordered quota is credited.
	alreadyDone, err := RechargeSePay(order.TradeNo, 100, "127.0.0.1")
	require.NoError(t, err)
	assert.False(t, alreadyDone)
	assert.Equal(t, 2*500000, getUserQuotaForPaymentGuardTest(t, user.Id))

	reloaded := GetTopUpByTradeNo(order.TradeNo)
	require.NotNil(t, reloaded)
	assert.Equal(t, common.TopUpStatusSuccess, reloaded.Status)
}

func TestRechargeSePayRejectedByExpiredByCreateTime(t *testing.T) {
	truncateTables(t)

	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })

	user := insertUserForPaymentGuardTest(t, 602, 0)
	// CreateTime far in the past, beyond the configured expiry window.
	expiredCreateTime := common.GetTimestamp() - int64(sePayExpiryWindowMinutes())*60 - 10
	order := createSePayOrderWithAmount(t, user.Id, "SEPAYTESTEXPIREDBYTIME", PaymentProviderSePay, common.TopUpStatusPending, 10.0, expiredCreateTime)

	_, err := RechargeSePay(order.TradeNo, 10, "127.0.0.1")
	require.ErrorIs(t, err, ErrTopUpStatusInvalid)
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, user.Id))
	assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, order.TradeNo))
}

func TestRechargeSePaySubscriptionRejectsUnderpayment(t *testing.T) {
	truncateTables(t)

	user := insertUserForPaymentGuardTest(t, 604, 0)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 602)
	insertSubscriptionOrderForPaymentGuardTest(t, "SEPAYTESTSUBUNDERPAY", user.Id, plan.Id, PaymentProviderSePay)

	order := GetSubscriptionOrderByTradeNo("SEPAYTESTSUBUNDERPAY")
	require.NotNil(t, order)
	order.Money = 100.0
	require.NoError(t, DB.Save(order).Error)

	alreadyDone, err := RechargeSePaySubscription(order.TradeNo, 99)
	require.ErrorIs(t, err, ErrTopUpAmountInsufficient)
	assert.False(t, alreadyDone)

	reloaded := GetSubscriptionOrderByTradeNo(order.TradeNo)
	require.NotNil(t, reloaded)
	assert.Equal(t, common.TopUpStatusPending, reloaded.Status)
}

func TestRechargeSePaySubscriptionRejectsForeignProvider(t *testing.T) {
	truncateTables(t)

	user := insertUserForPaymentGuardTest(t, 605, 0)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 603)
	insertSubscriptionOrderForPaymentGuardTest(t, "SEPAYTESTSUBFOREIGN", user.Id, plan.Id, PaymentProviderStripe)

	_, err := RechargeSePaySubscription("SEPAYTESTSUBFOREIGN", 100)
	require.ErrorIs(t, err, ErrPaymentMethodMismatch)

	order := GetSubscriptionOrderByTradeNo("SEPAYTESTSUBFOREIGN")
	require.NotNil(t, order)
	assert.Equal(t, common.TopUpStatusPending, order.Status)
}

func TestExpireSePayTopUpsBulkOnlyAffectsSePayPending(t *testing.T) {
	truncateTables(t)

	user := insertUserForPaymentGuardTest(t, 606, 0)
	now := common.GetTimestamp()

	// SePay pending past deadline → expires.
	createSePayOrderWithAmount(t, user.Id, "SEPAYBULKEXPIRE", PaymentProviderSePay, common.TopUpStatusPending, 10.0, now-3600)
	// SePay pending within window → untouched.
	createSePayOrderWithAmount(t, user.Id, "SEPAYBULKALIVE", PaymentProviderSePay, common.TopUpStatusPending, 10.0, now)
	// Legacy pending past deadline → deliberately untouched.
	createSePayOrderWithAmount(t, user.Id, "LEGACYBULKALIVE", PaymentProviderEpay, common.TopUpStatusPending, 10.0, now-3600)

	count, err := ExpireSePayTopUpsBulk(0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	assert.Equal(t, common.TopUpStatusExpired, getTopUpStatusForPaymentGuardTest(t, "SEPAYBULKEXPIRE"))
	assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, "SEPAYBULKALIVE"))
	assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, "LEGACYBULKALIVE"))
}

func TestSePayOrderExpiryUnixUsesConfiguredWindow(t *testing.T) {
	oldMinutes := setting.SePayOrderExpiryMinutes
	setting.SePayOrderExpiryMinutes = 45
	t.Cleanup(func() { setting.SePayOrderExpiryMinutes = oldMinutes })

	createTime := int64(1700000000)
	expected := createTime + 45*60
	assert.Equal(t, expected, SePayOrderExpiryUnix(createTime))
}
