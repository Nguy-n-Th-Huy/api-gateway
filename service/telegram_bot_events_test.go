package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// allowLocalCallbacksForTest disables SSRF filtering for the duration of a
// test so a loopback httptest.Server can stand in for the bot's callback
// address. Production traffic is unaffected: this only touches the
// process-wide fetch setting for the life of the test.
func allowLocalCallbacksForTest(t *testing.T) {
	t.Helper()
	fetchSetting := system_setting.GetFetchSetting()
	original := *fetchSetting
	t.Cleanup(func() { *fetchSetting = original })
	fetchSetting.EnableSSRFProtection = false
	InitHttpClient()
}

func TestDeliverTelegramBotEventOnceSignsAndSucceeds(t *testing.T) {
	allowLocalCallbacksForTest(t)

	var receivedSignature string
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSignature = r.Header.Get("X-Telegram-Bot-Signature")
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	body := []byte(`{"event_id":"evt-1","type":"topup.succeeded"}`)
	err := deliverTelegramBotEventOnce(server.URL, "shared-secret", body)
	require.NoError(t, err)

	assert.Equal(t, body, receivedBody)
	mac := hmac.New(sha256.New, []byte("shared-secret"))
	mac.Write(body)
	assert.Equal(t, hex.EncodeToString(mac.Sum(nil)), receivedSignature)
}

func TestDeliverTelegramBotEventOnceFailsOnServerError(t *testing.T) {
	allowLocalCallbacksForTest(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := deliverTelegramBotEventOnce(server.URL, "shared-secret", []byte(`{}`))
	assert.Error(t, err)
}

// TestDeliverTelegramBotEventWithRetryAbandonsAfterBoundedAttempts proves
// retries stop after a bounded number of attempts rather than continuing
// indefinitely (specs/telegram/bot-events/spec.md, "Delivery is retried,
// bounded, and observable"). It exercises the real backoff, so it takes a
// few seconds.
func TestDeliverTelegramBotEventWithRetryAbandonsAfterBoundedAttempts(t *testing.T) {
	allowLocalCallbacksForTest(t)

	var attempts atomic.Int64
	var mu sync.Mutex
	var receivedEventIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		body, _ := io.ReadAll(r.Body)
		var decoded map[string]any
		_ = common.Unmarshal(body, &decoded)
		mu.Lock()
		receivedEventIDs = append(receivedEventIDs, decoded["event_id"].(string))
		mu.Unlock()
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	deliverTelegramBotEventWithRetry(telegramBotEventPayload{
		EventID: "evt-bounded", Type: TelegramBotEventCredited, Timestamp: common.GetTimestamp(),
		Data: map[string]any{"trade_no": "T1"},
	}, server.URL, "shared-secret")

	assert.Equal(t, int64(telegramBotEventMaxAttempts), attempts.Load())
	require.Len(t, receivedEventIDs, telegramBotEventMaxAttempts)
	for _, id := range receivedEventIDs {
		assert.Equal(t, "evt-bounded", id, "every retry of one delivery must carry the same event identifier")
	}
}

// setupTelegramBotEventsSettlementTest gives model.RechargeSePay a fresh
// sqlite fixture and configures the bot integration with a callback address
// that is guaranteed to refuse the connection immediately.
func setupTelegramBotEventsSettlementTest(t *testing.T) (unreachableCallback string) {
	t.Helper()
	allowLocalCallbacksForTest(t)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.TopUp{}))

	previousDB := model.DB
	previousRedis := common.RedisEnabled
	t.Cleanup(func() {
		model.DB = previousDB
		common.RedisEnabled = previousRedis
	})
	model.DB = db
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)

	previousEnabled := setting.TelegramBotIntegrationEnabled
	previousKey := setting.TelegramBotServiceKey
	previousCallback := setting.TelegramBotCallbackURL
	previousSecret := setting.TelegramBotCallbackSecret
	t.Cleanup(func() {
		setting.TelegramBotIntegrationEnabled = previousEnabled
		setting.TelegramBotServiceKey = previousKey
		setting.TelegramBotCallbackURL = previousCallback
		setting.TelegramBotCallbackSecret = previousSecret
	})
	setting.TelegramBotIntegrationEnabled = true
	setting.TelegramBotServiceKey = "settlement-test-service-key"
	setting.TelegramBotCallbackSecret = "settlement-test-signing-secret"

	// A server that is opened and immediately closed hands back a port no
	// one is listening on, so every connection attempt is refused at once —
	// deterministic and fast, unlike relying on ambient network behavior.
	deadServer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	unreachableCallback = deadServer.URL
	deadServer.Close()
	setting.TelegramBotCallbackURL = unreachableCallback

	return unreachableCallback
}

// TestRechargeSePayCreditSettlesDespiteUnreachableEventCallback is the core
// 10.11 regression: a wallet credit must commit, and RechargeSePay must
// return promptly, even though the Telegram bot event it triggers can never
// be delivered (specs/telegram/bot-events/spec.md, "Delivery failure never
// damages the originating operation").
func TestRechargeSePayCreditSettlesDespiteUnreachableEventCallback(t *testing.T) {
	setupTelegramBotEventsSettlementTest(t)

	user := &model.User{Username: "settlement-user", Status: common.UserStatusEnabled, Group: "default", TelegramId: "tg-settlement-user"}
	require.NoError(t, model.DB.Create(user).Error)

	topUp := &model.TopUp{
		UserId: user.Id, Amount: 10, Money: 100000, PaymentMethod: model.PaymentMethodSePay,
		PaymentProvider: model.PaymentProviderSePay, CreateTime: time.Now().Unix(), Status: common.TopUpStatusPending,
	}
	require.NoError(t, model.InsertSePayTopUp(topUp))

	started := time.Now()
	alreadyDone, err := model.RechargeSePay(topUp.TradeNo, 100000, "203.0.113.9")
	elapsed := time.Since(started)

	require.NoError(t, err)
	assert.False(t, alreadyDone)
	assert.Less(t, elapsed, 2*time.Second,
		"an unreachable bot callback must never delay the settlement path")

	var reloadedOrder model.TopUp
	require.NoError(t, model.DB.Where("trade_no = ?", topUp.TradeNo).First(&reloadedOrder).Error)
	assert.Equal(t, common.TopUpStatusSuccess, reloadedOrder.Status, "the order must still be marked settled")

	var reloadedUser model.User
	require.NoError(t, model.DB.First(&reloadedUser, user.Id).Error)
	assert.Positive(t, reloadedUser.Quota, "the wallet must still be credited")
}
