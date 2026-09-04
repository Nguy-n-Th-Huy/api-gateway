package controller

import (
	"bytes"
	_ "unsafe"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

//go:linkname modelCommonGroupCol github.com/QuantumNous/new-api/model.commonGroupCol
var modelCommonGroupCol string

// initModelGroupColumn sets the model package's unexported column name used by
// GetUserGroup. Controller tests run in a separate package binary whose
// TestMain is not the model package's one, so this column stays zero-valued
// unless explicitly set here. [Finding: no exported test helper exists for this.]
func initModelGroupColumn(t *testing.T) {
	t.Helper()
	modelCommonGroupCol = "`group`"
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
}

func initSePayControllerTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.TopUp{}, &model.User{}, &model.SubscriptionPlan{}, &model.SubscriptionOrder{},
		&model.Log{}, &model.Option{},
	))
	model.DB = db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	initModelGroupColumn(t)
}

// setupSePayTestContext configures a gin test context with a SePay controller
// test DB, compliance confirmed, and a fully configured SePay integration.
func setupSePayTestContext(t *testing.T) (*gin.Engine, *model.User) {
	t.Helper()
	initSePayControllerTestDB(t)

	// Compliance confirmed.
	ps := operation_setting.GetPaymentSetting()
	originalConfirmed := ps.ComplianceConfirmed
	originalVersion := ps.ComplianceTermsVersion
	t.Cleanup(func() {
		ps.ComplianceConfirmed = originalConfirmed
		ps.ComplianceTermsVersion = originalVersion
	})
	ps.ComplianceConfirmed = true
	ps.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion

	// SePay fully configured.
	originalEnabled := setting.SePayEnabled
	originalAccount := setting.SePayBankAccountNumber
	originalBankCode := setting.SePayBankCode
	originalHolder := setting.SePayAccountHolder
	originalKey := setting.SePayWebhookApiKey
	originalMinTopUp := setting.SePayMinTopUp
	originalPrice := operation_setting.Price
	originalQuotaPerUnit := common.QuotaPerUnit
	originalDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	t.Cleanup(func() {
		setting.SePayEnabled = originalEnabled
		setting.SePayBankAccountNumber = originalAccount
		setting.SePayBankCode = originalBankCode
		setting.SePayAccountHolder = originalHolder
		setting.SePayWebhookApiKey = originalKey
		setting.SePayMinTopUp = originalMinTopUp
		operation_setting.Price = originalPrice
		common.QuotaPerUnit = originalQuotaPerUnit
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalDisplayType
	})
	setting.SePayEnabled = true
	setting.SePayBankAccountNumber = "987654321"
	setting.SePayBankCode = "niclop"
	setting.SePayAccountHolder = "NGUYEN VAN A"
	setting.SePayWebhookApiKey = "sepay_test_key"
	setting.SePayMinTopUp = 1
	operation_setting.Price = 1000.0
	common.QuotaPerUnit = 500000
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD

	user := &model.User{Id: 42, Username: "sepay_test_user", Status: common.UserStatusEnabled, Quota: 0, Group: "default"}
	require.NoError(t, model.DB.Create(user).Error)

	router := gin.New()
	router.POST("/api/user/sepay/pay", func(c *gin.Context) {
		c.Set("id", user.Id)
		SePayRequestTopUp(c)
	})
	router.GET("/api/user/sepay/order/:trade_no", func(c *gin.Context) {
		c.Set("id", user.Id)
		SePayGetOrder(c)
	})
	return router, user
}

func TestSePayWebhookAuthenticationHeaderVariants(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{name: "valid Apikey header", header: "Apikey sepay_test_key", wantStatus: http.StatusOK},
		{name: "wrong scheme Bearer", header: "Bearer sepay_test_key", wantStatus: http.StatusUnauthorized},
		{name: "lowercase apikey scheme", header: "apikey sepay_test_key", wantStatus: http.StatusUnauthorized},
		{name: "wrong key", header: "Apikey wrong_value", wantStatus: http.StatusUnauthorized},
		{name: "empty key", header: "Apikey ", wantStatus: http.StatusUnauthorized},
		{name: "missing header", header: "", wantStatus: http.StatusUnauthorized},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initSePayControllerTestDB(t)

			ps := operation_setting.GetPaymentSetting()
			originalConfirmed := ps.ComplianceConfirmed
			originalVersion := ps.ComplianceTermsVersion
			t.Cleanup(func() {
				ps.ComplianceConfirmed = originalConfirmed
				ps.ComplianceTermsVersion = originalVersion
			})
			ps.ComplianceConfirmed = true
			ps.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion

			origKey := setting.SePayWebhookApiKey
			origEnabled := setting.SePayEnabled
			origAccount := setting.SePayBankAccountNumber
			origBankCode := setting.SePayBankCode
			origHolder := setting.SePayAccountHolder
			t.Cleanup(func() {
				setting.SePayWebhookApiKey = origKey
				setting.SePayEnabled = origEnabled
				setting.SePayBankAccountNumber = origAccount
				setting.SePayBankCode = origBankCode
				setting.SePayAccountHolder = origHolder
			})
			setting.SePayWebhookApiKey = "sepay_test_key"
			setting.SePayEnabled = true
			setting.SePayBankAccountNumber = "987654321"
			setting.SePayBankCode = "niclop"
			setting.SePayAccountHolder = "NGUYEN VAN A"

			router := gin.New()
			router.POST("/api/sepay/webhook", SePayWebhook)

			body := `{"id":1,"content":"hello","transferType":"in","transferAmount":100,"referenceCode":"REF001"}`
			req := httptest.NewRequest(http.MethodPost, "/api/sepay/webhook", bytes.NewReader([]byte(body)))
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestSePayWebhookRejectedWhenIntegrationDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	initSePayControllerTestDB(t)

	ps := operation_setting.GetPaymentSetting()
	originalConfirmed := ps.ComplianceConfirmed
	originalVersion := ps.ComplianceTermsVersion
	t.Cleanup(func() {
		ps.ComplianceConfirmed = originalConfirmed
		ps.ComplianceTermsVersion = originalVersion
	})
	ps.ComplianceConfirmed = true
	ps.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion

	originalEnabled := setting.SePayEnabled
	originalKey := setting.SePayWebhookApiKey
	originalAccount := setting.SePayBankAccountNumber
	originalBankCode := setting.SePayBankCode
	originalHolder := setting.SePayAccountHolder
	t.Cleanup(func() {
		setting.SePayEnabled = originalEnabled
		setting.SePayWebhookApiKey = originalKey
		setting.SePayBankAccountNumber = originalAccount
		setting.SePayBankCode = originalBankCode
		setting.SePayAccountHolder = originalHolder
	})
	setting.SePayEnabled = false
	setting.SePayWebhookApiKey = "sepay_test_key"
	setting.SePayBankAccountNumber = "987654321"
	setting.SePayBankCode = "niclop"
	setting.SePayAccountHolder = "NGUYEN VAN A"

	router := gin.New()
	router.POST("/api/sepay/webhook", SePayWebhook)

	body := `{"id":1,"content":"hello","transferType":"in","transferAmount":100,"referenceCode":"REF001"}`
	req := httptest.NewRequest(http.MethodPost, "/api/sepay/webhook", bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Apikey sepay_test_key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSePayOrderCreationBoundsRejectsAndDoesNotCreateOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("below minimum", func(t *testing.T) {
		router, _ := setupSePayTestContext(t)
		setting.SePayMinTopUp = 5

		body := fmt.Sprintf(`{"amount":%d}`, 2)
		req := httptest.NewRequest(http.MethodPost, "/api/user/sepay/pay", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		var count int64
		require.NoError(t, model.DB.Model(&model.TopUp{}).Count(&count).Error)
		assert.Equal(t, int64(0), count)
		assert.Contains(t, rec.Body.String(), `"success":false`)
	})

	t.Run("above sePayMaxTopUpAmount returns not-created and error", func(t *testing.T) {
		router, _ := setupSePayTestContext(t)

		body := fmt.Sprintf(`{"amount":%d}`, sePayMaxTopUpAmount+1)
		req := httptest.NewRequest(http.MethodPost, "/api/user/sepay/pay", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		var count int64
		require.NoError(t, model.DB.Model(&model.TopUp{}).Count(&count).Error)
		assert.Equal(t, int64(0), count)
		assert.Contains(t, rec.Body.String(), `"success":false`)
	})

	t.Run("zero Dong payable rejected (price=0)", func(t *testing.T) {
		router, _ := setupSePayTestContext(t)
		operation_setting.Price = 0

		body := `{"amount":1}`
		req := httptest.NewRequest(http.MethodPost, "/api/user/sepay/pay", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		var count int64
		require.NoError(t, model.DB.Model(&model.TopUp{}).Count(&count).Error)
		assert.Equal(t, int64(0), count)
		assert.Contains(t, rec.Body.String(), `"success":false`)
	})

	t.Run("wallet capacity exceeded", func(t *testing.T) {
		router, user := setupSePayTestContext(t)
		require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", user.Id).
			Update("quota", common.MaxWalletQuota-10).Error)

		body := `{"amount":1}`
		req := httptest.NewRequest(http.MethodPost, "/api/user/sepay/pay", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		var count int64
		require.NoError(t, model.DB.Model(&model.TopUp{}).Count(&count).Error)
		assert.Equal(t, int64(0), count)
		assert.Contains(t, rec.Body.String(), `"success":false`)
	})

	t.Run("non-integer amount 10.5 rejected by int64 field", func(t *testing.T) {
		router, _ := setupSePayTestContext(t)
		// sePayTopUpRequest uses `json:"amount"` on int64; ShouldBindJSON returns
		// an error for a decimal like 10.5, so the handler replies with
		// {"success":false,"message":"参数错误"} and creates no order — satisfies
		// the spec's "Malformed amount submitted" scenario.
		body := `{"amount":10.5}`
		req := httptest.NewRequest(http.MethodPost, "/api/user/sepay/pay", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		var count int64
		require.NoError(t, model.DB.Model(&model.TopUp{}).Count(&count).Error)
		assert.Equal(t, int64(0), count)
		assert.Contains(t, rec.Body.String(), `"success":false`)
	})
}

func TestSePayOrderCreationArbitraryNonPresetAmountAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, _ := setupSePayTestContext(t)
	body := `{"amount":17}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/sepay/pay", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var topUps []*model.TopUp
	require.NoError(t, model.DB.Find(&topUps).Error)
	require.Len(t, topUps, 1)
	assert.Equal(t, int64(17), topUps[0].Amount)
	assert.Contains(t, rec.Body.String(), "success")
}

func TestSePayGetOrderScopesToOwnUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, _ := setupSePayTestContext(t)

	other := &model.User{Id: 99, Username: "other", Status: common.UserStatusEnabled, Quota: 0, Group: "default", AffCode: "aff_other_99"}
	require.NoError(t, model.DB.Create(other).Error)

	topUp := &model.TopUp{
		UserId:          other.Id,
		Amount:          10,
		Money:           10000,
		TradeNo:         "SPSCOPEOWNERS1213456",
		PaymentMethod:   model.PaymentMethodSePay,
		PaymentProvider: model.PaymentProviderSePay,
		CreateTime:      common.GetTimestamp(),
		Status:          common.TopUpStatusPending,
	}
	require.NoError(t, model.DB.Create(topUp).Error)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/user/sepay/order/%s", topUp.TradeNo), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Contains(t, rec.Body.String(), `"success":false`, "order owned by other user must resolve as not-found, body=%s", rec.Body.String())
}

func TestSePayWebhookAmbiguousContentResolvesAtMatchLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	initSePayControllerTestDB(t)

	// Two pending SePay orders whose memos both appear in one webhook content.
	ps := operation_setting.GetPaymentSetting()
	originalConfirmed := ps.ComplianceConfirmed
	originalVersion := ps.ComplianceTermsVersion
	t.Cleanup(func() {
		ps.ComplianceConfirmed = originalConfirmed
		ps.ComplianceTermsVersion = originalVersion
	})
	ps.ComplianceConfirmed = true
	ps.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion

	origKey := setting.SePayWebhookApiKey
	origEnabled := setting.SePayEnabled
	origAccount := setting.SePayBankAccountNumber
	origBankCode := setting.SePayBankCode
	origHolder := setting.SePayAccountHolder
	t.Cleanup(func() {
		setting.SePayWebhookApiKey = origKey
		setting.SePayEnabled = origEnabled
		setting.SePayBankAccountNumber = origAccount
		setting.SePayBankCode = origBankCode
		setting.SePayAccountHolder = origHolder
	})
	setting.SePayWebhookApiKey = "sepay_test_key"
	setting.SePayEnabled = true
	setting.SePayBankAccountNumber = "987654321"
	setting.SePayBankCode = "niclop"
	setting.SePayAccountHolder = "NGUYEN VAN A"

	memo1 := "SPABCDEFG1234567"
	memo2 := "SPABCDEFG9999999"
	for _, memo := range []string{memo1, memo2} {
		topUp := &model.TopUp{
			UserId:          42,
			Amount:          10,
			Money:           10000,
			TradeNo:         memo,
			PaymentMethod:   model.PaymentMethodSePay,
			PaymentProvider: model.PaymentProviderSePay,
			CreateTime:      common.GetTimestamp(),
			Status:          common.TopUpStatusPending,
		}
		if memo == memo2 {
			topUp.UserId = 43
			topUp.Amount = 5
		}
		other := &model.User{Id: topUp.UserId, Username: fmt.Sprintf("u%d", topUp.UserId), Status: common.UserStatusEnabled, Quota: 0, Group: "default"}
		_ = model.DB.FirstOrCreate(other, model.User{Id: topUp.UserId}).Error
		require.NoError(t, model.DB.Create(topUp).Error)
	}

	// Webhook with both memos in content → ambiguous, settles none.
	router := gin.New()
	router.POST("/api/sepay/webhook", SePayWebhook)
	body := fmt.Sprintf(`{"id":1,"content":"%s %s","transferType":"in","transferAmount":20000,"referenceCode":"REFAMBIG001"}`, memo1, memo2)
	req := httptest.NewRequest(http.MethodPost, "/api/sepay/webhook", bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Apikey sepay_test_key")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "ambiguous webhook returns 200 (settles nothing)")
	first := model.GetTopUpByTradeNo(memo1)
	require.NotNil(t, first)
	assert.Equal(t, common.TopUpStatusPending, first.Status)
	second := model.GetTopUpByTradeNo(memo2)
	require.NotNil(t, second)
	assert.Equal(t, common.TopUpStatusPending, second.Status)
}
