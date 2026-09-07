/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupUpdateSelfTestDB gives UpdateSelf a fresh in-memory users table,
// independent of whatever any other test in this package left model.DB set
// to, and disables Redis so the cache-write side effects inside
// model.User.Update are no-ops.
func setupUpdateSelfTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	require.NoError(t, i18n.Init())

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	model.DB = db

	originalRedisEnabled := common.RedisEnabled
	t.Cleanup(func() {
		common.RedisEnabled = originalRedisEnabled
	})
	common.RedisEnabled = false

	return db
}

func createUpdateSelfTestUser(t *testing.T, db *gorm.DB, telegramUsername string) model.User {
	t.Helper()
	user := model.User{
		Username:         "update-self-user",
		Password:         "hashed-password",
		DisplayName:      "Update Self User",
		Status:           common.UserStatusEnabled,
		Role:             common.RoleCommonUser,
		Group:            "default",
		AuthVersion:      1,
		TelegramUsername: telegramUsername,
	}
	require.NoError(t, db.Create(&user).Error)
	return user
}

// TestUpdateSelfTelegramHandleClearing covers the self-service profile
// update path's handling of the Telegram handle
// (openspec task 4.3, "let a signed-in user change their own handle through
// the existing self-update path"). model.User.Update writes through GORM's
// Updates(struct), which treats an empty string the same as "field not
// submitted" and skips it — so submitting an empty handle to actually clear
// it must not collapse into the same no-op as never mentioning the field.
func TestUpdateSelfTelegramHandleClearing(t *testing.T) {
	testCases := []struct {
		name          string
		requestBody   map[string]any
		initialHandle string
		wantHandle    string
	}{
		{
			name: "submitting an empty handle clears the stored value",
			requestBody: map[string]any{
				"display_name":      "Updated Name",
				"telegram_username": "",
			},
			initialHandle: "existing_handle",
			wantHandle:    "",
		},
		{
			name: "a profile update that never mentions the handle leaves it untouched",
			requestBody: map[string]any{
				"display_name": "Updated Name Only",
			},
			initialHandle: "existing_handle",
			wantHandle:    "existing_handle",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := setupUpdateSelfTestDB(t)
			user := createUpdateSelfTestUser(t, db, tc.initialHandle)

			ctx, recorder := newAuthenticatedContext(t, http.MethodPut, "/api/user/self", tc.requestBody, user.Id)
			UpdateSelf(ctx)

			require.Equal(t, http.StatusOK, recorder.Code)
			response := decodeAPIResponse(t, recorder)
			require.True(t, response.Success, response.Message)

			var stored model.User
			require.NoError(t, db.First(&stored, user.Id).Error)
			assert.Equal(t, tc.wantHandle, stored.TelegramUsername)
			assert.Equal(t, tc.requestBody["display_name"], stored.DisplayName, "the unrelated display_name field must still update")
		})
	}
}
