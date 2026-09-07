package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTelegramBotEndpointTest reuses setupSePayTestContext's compliance and
// SePay wiring (sepay_webhook_test.go, same package) and additionally
// migrates the Token table, so bot handlers that touch keys can run. Handlers
// are invoked directly, bypassing the router and its middleware.TelegramBotAuth
// / rate-limit layer, which are covered separately.
func setupTelegramBotEndpointTest(t *testing.T) *model.User {
	t.Helper()
	_, user := setupSePayTestContext(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Token{}))

	originalRateLimitEnable := common.CriticalRateLimitEnable
	originalRedisEnabled := common.RedisEnabled
	t.Cleanup(func() {
		common.CriticalRateLimitEnable = originalRateLimitEnable
		common.RedisEnabled = originalRedisEnabled
	})
	// These tests exercise business logic, not throttling (covered by
	// TestTelegramUserRateLimitIsolatesIdentifiers in the middleware package).
	common.CriticalRateLimitEnable = false
	// common.RedisEnabled defaults to true outside a real server start; this
	// sqlite-backed fixture has no Redis client for the token/session caches
	// that default flips on.
	common.RedisEnabled = false

	return user
}

func telegramBotTestContext(method, target string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, nil)
	return ctx, recorder
}

// telegramBotTestContextWithBody is telegramBotTestContext plus a JSON body,
// for the POST endpoints that read a key or an amount from it.
func telegramBotTestContextWithBody(t *testing.T, method, target string, body map[string]any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	payload, err := common.Marshal(body)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	return ctx, recorder
}

// --- 10.6 Order ownership -------------------------------------------------

func TestTelegramBotOrderOwnershipScoping(t *testing.T) {
	owner := setupTelegramBotEndpointTest(t)
	owner.TelegramId = "tg-order-owner"
	require.NoError(t, model.DB.Model(owner).Update("telegram_id", owner.TelegramId).Error)

	stranger := &model.User{Username: "order_stranger", Status: common.UserStatusEnabled, Group: "default", AffCode: "order_stranger", TelegramId: "tg-order-stranger"}
	require.NoError(t, model.DB.Create(stranger).Error)

	topUp := &model.TopUp{
		UserId: owner.Id, Amount: 10, Money: 100000, PaymentMethod: model.PaymentMethodSePay,
		PaymentProvider: model.PaymentProviderSePay, CreateTime: time.Now().Unix(), Status: common.TopUpStatusPending,
	}
	require.NoError(t, model.InsertSePayTopUp(topUp))

	// Owner reads their own order.
	ctxOwner, recOwner := telegramBotTestContext(http.MethodGet, "/api/bot/v1/topup/orders/"+topUp.TradeNo+"?telegram_user_id="+owner.TelegramId)
	ctxOwner.Params = gin.Params{{Key: "trade_no", Value: topUp.TradeNo}}
	TelegramBotOrderStatus(ctxOwner)
	assert.Equal(t, http.StatusOK, recOwner.Code)
	assert.Contains(t, recOwner.Body.String(), topUp.TradeNo)

	// A different account polling the same trade number is refused.
	ctxStranger, recStranger := telegramBotTestContext(http.MethodGet, "/api/bot/v1/topup/orders/"+topUp.TradeNo+"?telegram_user_id="+stranger.TelegramId)
	ctxStranger.Params = gin.Params{{Key: "trade_no", Value: topUp.TradeNo}}
	TelegramBotOrderStatus(ctxStranger)

	// An unknown trade number for the owner is refused identically.
	ctxUnknown, recUnknown := telegramBotTestContext(http.MethodGet, "/api/bot/v1/topup/orders/NOSUCHORDER?telegram_user_id="+owner.TelegramId)
	ctxUnknown.Params = gin.Params{{Key: "trade_no", Value: "NOSUCHORDER"}}
	TelegramBotOrderStatus(ctxUnknown)

	assert.Equal(t, recStranger.Code, recUnknown.Code)
	assert.JSONEq(t, recStranger.Body.String(), recUnknown.Body.String(),
		"a non-owner and an unknown trade number must be indistinguishable")
	assert.NotContains(t, recStranger.Body.String(), topUp.TradeNo)
}

// --- 10.7 Key listing leaks no key material --------------------------------

func TestTelegramBotKeyListingNeverContainsKeyValue(t *testing.T) {
	owner := setupTelegramBotEndpointTest(t)
	owner.TelegramId = "tg-key-listing-owner"
	require.NoError(t, model.DB.Model(owner).Update("telegram_id", owner.TelegramId).Error)

	const rawKey = "sk-supersecretlistingkey1234567890"
	token := &model.Token{
		UserId: owner.Id, Name: "listing-token", Key: rawKey, Status: common.TokenStatusEnabled,
		CreatedTime: time.Now().Unix(), AccessedTime: time.Now().Unix(), ExpiredTime: -1,
		RemainQuota: 100, Group: "default",
	}
	require.NoError(t, model.DB.Create(token).Error)

	ctx, recorder := telegramBotTestContext(http.MethodGet, "/api/bot/v1/keys?telegram_user_id="+owner.TelegramId)
	TelegramBotKeyList(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	assert.Contains(t, body, "listing-token", "the listing must still report the key's name")
	assert.NotContains(t, body, rawKey, "the raw key value must never appear")
	assert.NotContains(t, body, strings.TrimPrefix(rawKey, "sk-")[:8], "no fragment of the key may appear either")
	assert.NotContains(t, body, `"key"`, "no field carrying a key value may be present")
}

// --- 10.8 Key-accepting endpoints ignore a query-only key ------------------

func TestTelegramBotKeyEndpointsRejectQueryOnlyKey(t *testing.T) {
	owner := setupTelegramBotEndpointTest(t)
	owner.TelegramId = "tg-query-key-owner"
	require.NoError(t, model.DB.Model(owner).Update("telegram_id", owner.TelegramId).Error)

	const rawKey = "sk-queryonlykeyshouldneverresolve1"
	token := &model.Token{
		UserId: owner.Id, Name: "query-only-token", Key: rawKey, Status: common.TokenStatusEnabled,
		CreatedTime: time.Now().Unix(), AccessedTime: time.Now().Unix(), ExpiredTime: -1,
		RemainQuota: 100, Group: "default",
	}
	require.NoError(t, model.DB.Create(token).Error)

	requestBody := map[string]any{"telegram_user_id": owner.TelegramId}

	inspectCtx, inspectRec := telegramBotTestContextWithBody(t, http.MethodPost, "/api/bot/v1/keys/check?key="+rawKey, requestBody)
	TelegramBotKeyInspect(inspectCtx)
	assert.Contains(t, inspectRec.Body.String(), telegramBotErrorKeyNotFound,
		"a key supplied only in the query string must be treated as missing")

	logsCtx, logsRec := telegramBotTestContextWithBody(t, http.MethodPost, "/api/bot/v1/keys/logs?key="+rawKey, requestBody)
	TelegramBotKeyLogs(logsCtx)
	assert.Contains(t, logsRec.Body.String(), telegramBotErrorKeyNotFound,
		"a key supplied only in the query string must be treated as missing")
}

// --- 10.9 Bot order creation shares the console validation chain ----------

func TestTelegramBotCreateOrderValidationChain(t *testing.T) {
	owner := setupTelegramBotEndpointTest(t)
	owner.TelegramId = "tg-order-validation"
	require.NoError(t, model.DB.Model(owner).Update("telegram_id", owner.TelegramId).Error)

	countOrders := func() int64 {
		var count int64
		require.NoError(t, model.DB.Model(&model.TopUp{}).Where("user_id = ?", owner.Id).Count(&count).Error)
		return count
	}

	t.Run("below minimum", func(t *testing.T) {
		before := countOrders()
		ctx, recorder := telegramBotTestContextWithBody(t, http.MethodPost, "/api/bot/v1/topup/orders",
			map[string]any{"telegram_user_id": owner.TelegramId, "amount": 0})
		TelegramBotCreateOrder(ctx)

		var response map[string]any
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		assert.False(t, response["success"].(bool))
		assert.Equal(t, before, countOrders(), "no order may be created below the minimum")
	})

	t.Run("above maximum", func(t *testing.T) {
		before := countOrders()
		ctx, recorder := telegramBotTestContextWithBody(t, http.MethodPost, "/api/bot/v1/topup/orders",
			map[string]any{"telegram_user_id": owner.TelegramId, "amount": 100000})
		TelegramBotCreateOrder(ctx)

		var response map[string]any
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		assert.False(t, response["success"].(bool))
		assert.Equal(t, before, countOrders(), "no order may be created above the per-order maximum")
	})

	t.Run("beyond wallet capacity", func(t *testing.T) {
		require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", owner.Id).
			Update("quota", common.MaxWalletQuota-10).Error)
		before := countOrders()
		ctx, recorder := telegramBotTestContextWithBody(t, http.MethodPost, "/api/bot/v1/topup/orders",
			map[string]any{"telegram_user_id": owner.TelegramId, "amount": 100})
		TelegramBotCreateOrder(ctx)

		var response map[string]any
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		assert.False(t, response["success"].(bool))
		assert.Equal(t, before, countOrders(), "no order may be created beyond the wallet ceiling")
	})
}
