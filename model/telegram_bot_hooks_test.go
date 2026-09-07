package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpireSePayTopUpsBulkDetailedReturnsExpiredRows(t *testing.T) {
	truncateTables(t)

	user := insertUserForPaymentGuardTest(t, 710, 0)
	now := common.GetTimestamp()

	expired := createSePayOrderWithAmount(t, user.Id, "SEPAYDETAILEXPIRE", PaymentProviderSePay, common.TopUpStatusPending, 10.0, now-3600)
	createSePayOrderWithAmount(t, user.Id, "SEPAYDETAILALIVE", PaymentProviderSePay, common.TopUpStatusPending, 10.0, now)

	rows, err := ExpireSePayTopUpsBulkDetailed(0)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, expired.TradeNo, rows[0].TradeNo)
	assert.Equal(t, user.Id, rows[0].UserId)

	assert.Equal(t, common.TopUpStatusExpired, getTopUpStatusForPaymentGuardTest(t, "SEPAYDETAILEXPIRE"))
	assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, "SEPAYDETAILALIVE"))
}

// TestRechargeSePayInvokesCreditedEventHookAfterCommit proves the hook fires
// only once the credit has committed and carries the settled order's own
// user id, trade number, and amount (specs/telegram/bot-events/spec.md,
// "Top-up credited").
func TestRechargeSePayInvokesCreditedEventHookAfterCommit(t *testing.T) {
	truncateTables(t)
	previousHook := TelegramCreditedEventHook
	t.Cleanup(func() { TelegramCreditedEventHook = previousHook })

	user := insertUserForPaymentGuardTest(t, 711, 0)
	topUp := createSePayOrderWithAmount(t, user.Id, "SEPAYHOOKCREDIT", PaymentProviderSePay, common.TopUpStatusPending, 10.0, common.GetTimestamp())

	type hookCall struct {
		userId  int
		tradeNo string
		amount  int64
	}
	var captured *hookCall
	TelegramCreditedEventHook = func(userId int, tradeNo string, creditedAmount int64) {
		// The order must already be committed as settled by the time the
		// hook runs, proving it fires strictly after the credit — never
		// before or in place of it.
		var reloaded TopUp
		require.NoError(t, DB.Where("trade_no = ?", tradeNo).First(&reloaded).Error)
		assert.Equal(t, common.TopUpStatusSuccess, reloaded.Status)
		captured = &hookCall{userId: userId, tradeNo: tradeNo, amount: creditedAmount}
	}

	alreadyDone, err := RechargeSePay(topUp.TradeNo, 10, "127.0.0.1")
	require.NoError(t, err)
	assert.False(t, alreadyDone)

	require.NotNil(t, captured)
	assert.Equal(t, user.Id, captured.userId)
	assert.Equal(t, topUp.TradeNo, captured.tradeNo)
	assert.Equal(t, topUp.Amount, captured.amount)
}
