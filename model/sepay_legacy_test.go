package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLegacyTopUpsRemainListableAndSearchable(t *testing.T) {
	truncateTables(t)

	user := insertUserForPaymentGuardTest(t, 701, 0)
	// Seed legacy orders with the retired provider values that historical rows
	// carry, plus one SePay order, and confirm both listing and searching still
	// return them unchanged (legacy-gateway-removal spec).
	legacyOrders := []struct {
		tradeNo  string
		provider string
		method   string
	}{
		{"EPAYLEGACY000001", PaymentProviderEpay, PaymentProviderEpay},
		{"STRIPELEGACY00002", PaymentProviderStripe, PaymentMethodStripe},
		{"WAFFOLEGACY0000003", PaymentProviderWaffo, PaymentMethodWaffo},
		{"CREEMLEGACY0000004", PaymentProviderCreem, PaymentMethodCreem},
	}
	legacyTradeNos := make([]string, 0, len(legacyOrders))
	for _, lo := range legacyOrders {
		topUp := &TopUp{
			UserId:          user.Id,
			Amount:          10,
			Money:           10.0,
			TradeNo:         lo.tradeNo,
			PaymentMethod:   lo.method,
			PaymentProvider: lo.provider,
			CreateTime:      common.GetTimestamp(),
			Status:          common.TopUpStatusSuccess,
		}
		require.NoError(t, DB.Create(topUp).Error)
		legacyTradeNos = append(legacyTradeNos, lo.tradeNo)
	}
	sepayOrder := createSePayTestOrder(t, user.Id, "SEPAYLEGACYLIST0001", PaymentProviderSePay, common.TopUpStatusSuccess)

	// GetAllTopUps (admin) returns all rows including legacy.
	pageInfo := &common.PageInfo{Page: 1, PageSize: 100}
	topups, total, err := GetAllTopUps(pageInfo)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(len(legacyTradeNos)+1))
	tradeNos := make(map[string]bool)
	for _, tp := range topups {
		tradeNos[tp.TradeNo] = true
	}
	for _, tn := range legacyTradeNos {
		assert.True(t, tradeNos[tn], "legacy trade_no %s not found in GetAllTopUps", tn)
	}
	assert.True(t, tradeNos[sepayOrder.TradeNo])

	// SearchAllTopUps by legacy prefix returns legacy rows.
	for _, tn := range legacyTradeNos {
		results, _, err := SearchAllTopUps(tn, pageInfo)
		require.NoError(t, err)
		found := false
		for _, r := range results {
			if r.TradeNo == tn {
				found = true
				assert.Equal(t, tn, r.TradeNo)
				assert.NotEmpty(t, r.PaymentProvider)
				break
			}
		}
		assert.True(t, found, "SearchAllTopUps did not return legacy order %s", tn)
	}
}

func TestManualCompleteTopUpStillCreditsLegacyPendingOrderExactlyOnce(t *testing.T) {
	truncateTables(t)

	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })

	user := insertUserForPaymentGuardTest(t, 702, 0)
	tradeNo := "EPAYLEGACYMANUAL0001"
	topUp := &TopUp{
		UserId:          user.Id,
		Amount:          2,
		Money:           10.0,
		TradeNo:         tradeNo,
		PaymentMethod:   PaymentProviderEpay,
		PaymentProvider: PaymentProviderEpay,
		CreateTime:      common.GetTimestamp(),
		Status:          common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(topUp).Error)

	require.NoError(t, ManualCompleteTopUp(tradeNo, "127.0.0.1"))
	assert.Equal(t, 2*500000, getUserQuotaForPaymentGuardTest(t, user.Id))
	reloaded := GetTopUpByTradeNo(tradeNo)
	require.NotNil(t, reloaded)
	assert.Equal(t, common.TopUpStatusSuccess, reloaded.Status)

	// Idempotent: second call leaves quota and status unchanged.
	require.NoError(t, ManualCompleteTopUp(tradeNo, "127.0.0.1"))
	assert.Equal(t, 2*500000, getUserQuotaForPaymentGuardTest(t, user.Id))
	assert.Equal(t, common.TopUpStatusSuccess, GetTopUpByTradeNo(tradeNo).Status)
}

func TestLegacySubscriptionOrdersRemainSearchable(t *testing.T) {
	truncateTables(t)

	user := insertUserForPaymentGuardTest(t, 703, 0)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 701)
	tradeNo := "SUBCREEMLEGACY00001"
	insertSubscriptionOrderForPaymentGuardTest(t, tradeNo, user.Id, plan.Id, PaymentProviderCreem)

	order := GetSubscriptionOrderByTradeNo(tradeNo)
	require.NotNil(t, order)
	assert.Equal(t, PaymentProviderCreem, order.PaymentProvider)
	assert.Equal(t, common.TopUpStatusPending, order.Status)
}
