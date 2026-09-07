package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupRegisterTestDB gives Register a fresh in-memory users table and
// enables the two feature flags it gates on, independent of whatever any
// other test in this package left them set to.
func setupRegisterTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	require.NoError(t, i18n.Init())

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	model.DB = db

	originalRegisterEnabled := common.RegisterEnabled
	originalPasswordRegisterEnabled := common.PasswordRegisterEnabled
	originalEmailVerificationEnabled := common.EmailVerificationEnabled
	originalRedisEnabled := common.RedisEnabled
	t.Cleanup(func() {
		common.RegisterEnabled = originalRegisterEnabled
		common.PasswordRegisterEnabled = originalPasswordRegisterEnabled
		common.EmailVerificationEnabled = originalEmailVerificationEnabled
		common.RedisEnabled = originalRedisEnabled
	})
	common.RegisterEnabled = true
	common.PasswordRegisterEnabled = true
	common.EmailVerificationEnabled = false
	// Registration inserts the account then re-reads it to set up sidebar
	// defaults; that re-read path writes through to Redis when enabled, which
	// this test's bare sqlite fixture has no client for.
	common.RedisEnabled = false

	return db
}

func registerRequestBody(username, telegramUsername string) map[string]any {
	body := map[string]any{
		"username": username,
		"password": "password123",
	}
	if telegramUsername != "" {
		body["telegram_username"] = telegramUsername
	}
	return body
}

func userCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&model.User{}).Count(&count).Error)
	return count
}

// TestRegisterTelegramHandleRequirement covers both states of the
// registration-handle requirement option: while enabled a missing or
// invalid handle must create no account, and a valid one must be stored
// normalized; while disabled the handle is optional
// (specs/telegram/account-link/spec.md, "Administrators can require a
// Telegram handle at registration").
func TestRegisterTelegramHandleRequirement(t *testing.T) {
	originalRequired := setting.TelegramHandleRequired
	t.Cleanup(func() { setting.TelegramHandleRequired = originalRequired })

	testCases := []struct {
		name               string
		requirementOn      bool
		telegramUsername   string
		wantAccountCreated bool
		wantStoredHandle   string
	}{
		{
			name:               "requirement enabled and handle omitted is rejected",
			requirementOn:      true,
			telegramUsername:   "",
			wantAccountCreated: false,
		},
		{
			name:               "requirement enabled and handle too short is rejected",
			requirementOn:      true,
			telegramUsername:   "abcd",
			wantAccountCreated: false,
		},
		{
			name:               "requirement enabled and handle starting with a digit is rejected",
			requirementOn:      true,
			telegramUsername:   "1johndoe",
			wantAccountCreated: false,
		},
		{
			name:               "requirement enabled and valid handle is accepted and normalized",
			requirementOn:      true,
			telegramUsername:   "  @JohnDoe  ",
			wantAccountCreated: true,
			wantStoredHandle:   "johndoe",
		},
		{
			name:               "requirement disabled and handle omitted still creates the account",
			requirementOn:      false,
			telegramUsername:   "",
			wantAccountCreated: true,
			wantStoredHandle:   "",
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := setupRegisterTestDB(t)
			setting.TelegramHandleRequired = tc.requirementOn

			username := "reg_user_" + string(rune('a'+i))
			ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/user/register", registerRequestBody(username, tc.telegramUsername), 0)

			Register(ctx)

			assert.Equal(t, http.StatusOK, recorder.Code)
			response := decodeAPIResponse(t, recorder)

			if !tc.wantAccountCreated {
				assert.False(t, response.Success, "expected registration to be rejected")
				assert.Equal(t, int64(0), userCount(t, db), "no account must be created on rejection")
				return
			}

			require.True(t, response.Success, response.Message)
			assert.Equal(t, int64(1), userCount(t, db))

			var stored model.User
			require.NoError(t, db.Where("username = ?", username).First(&stored).Error)
			assert.Equal(t, tc.wantStoredHandle, stored.TelegramUsername)
		})
	}
}
