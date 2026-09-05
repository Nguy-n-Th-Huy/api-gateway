package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- 1.2: effectiveTokenStatus is a pure function of the token row and a
// timestamp. These cases encode the precedence rule itself (disabled >
// expired > exhausted > enabled), not implementation details, and none of
// them touch a database.

func TestEffectiveTokenStatusPrecedence(t *testing.T) {
	const now int64 = 1_700_000_000

	t.Run("stored disabled wins over a past expiry", func(t *testing.T) {
		token := &model.Token{
			Status:      common.TokenStatusDisabled,
			ExpiredTime: now - 1000,
			RemainQuota: 100,
		}
		assert.Equal(t, common.TokenStatusDisabled, effectiveTokenStatus(token, now))
	})

	t.Run("past expiry wins over zero remaining quota", func(t *testing.T) {
		token := &model.Token{
			Status:         common.TokenStatusEnabled,
			ExpiredTime:    now - 1,
			RemainQuota:    0,
			UnlimitedQuota: false,
		}
		assert.Equal(t, common.TokenStatusExpired, effectiveTokenStatus(token, now))
	})

	t.Run("zero remaining quota with limited quota gives exhausted", func(t *testing.T) {
		token := &model.Token{
			Status:         common.TokenStatusEnabled,
			ExpiredTime:    -1,
			RemainQuota:    0,
			UnlimitedQuota: false,
		}
		assert.Equal(t, common.TokenStatusExhausted, effectiveTokenStatus(token, now))
	})

	t.Run("zero remaining quota with unlimited quota gives enabled", func(t *testing.T) {
		token := &model.Token{
			Status:         common.TokenStatusEnabled,
			ExpiredTime:    -1,
			RemainQuota:    0,
			UnlimitedQuota: true,
		}
		assert.Equal(t, common.TokenStatusEnabled, effectiveTokenStatus(token, now))
	})

	t.Run("expired_time -1 never expires even with a large remaining balance", func(t *testing.T) {
		token := &model.Token{
			Status:         common.TokenStatusEnabled,
			ExpiredTime:    -1,
			RemainQuota:    100,
			UnlimitedQuota: false,
		}
		assert.Equal(t, common.TokenStatusEnabled, effectiveTokenStatus(token, now))
	})

	t.Run("boundary: an expiry equal to now is not yet in the past", func(t *testing.T) {
		token := &model.Token{
			Status:         common.TokenStatusEnabled,
			ExpiredTime:    now,
			RemainQuota:    100,
			UnlimitedQuota: false,
		}
		assert.Equal(t, common.TokenStatusEnabled, effectiveTokenStatus(token, now))
	})
}

func TestNormalizeTokenKey(t *testing.T) {
	assert.Equal(t, "abc123def456", normalizeTokenKey("sk-abc123def456"))
	assert.Equal(t, "abc123def456", normalizeTokenKey("  Bearer sk-abc123def456  "))
	assert.Equal(t, "abc123def456", normalizeTokenKey("sk-abc123def456-extra"))
	assert.Equal(t, "abc123def456", normalizeTokenKey("bearer sk-abc123def456"))
	assert.Equal(t, "", normalizeTokenKey("   "))
	assert.Equal(t, "", normalizeTokenKey(""))
}

// ---- 2: POST /api/token/check handler tests ----

func TestCheckTokenUsageReturnsFullReportForExistingKey(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Ability{}))

	token := seedToken(t, db, 1, "my-key", "abcd1234efgh5678")
	token.UsedQuota = 500000
	token.RemainQuota = 1500000
	token.Group = "vip"
	require.NoError(t, db.Save(token).Error)

	require.NoError(t, db.Create(&model.Ability{Group: "vip", Model: "gpt-4o", ChannelId: 1, Enabled: true}).Error)
	require.NoError(t, db.Create(&model.Ability{Group: "vip", Model: "gpt-4o-mini", ChannelId: 1, Enabled: true}).Error)

	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/token/check", map[string]any{"key": "sk-" + token.Key}, 0)
	CheckTokenUsage(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	response := decodeAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)

	var report tokenCheckResponse
	require.NoError(t, common.Unmarshal(response.Data, &report))
	assert.Equal(t, "my-key", report.Name)
	assert.Equal(t, "vip", report.Group)
	assert.Equal(t, common.TokenStatusEnabled, report.Status)
	assert.Equal(t, 500000, report.TotalUsed)
	assert.Equal(t, 1500000, report.TotalAvailable)
	assert.Equal(t, 2000000, report.TotalGranted)
	assert.ElementsMatch(t, []string{"gpt-4o", "gpt-4o-mini"}, report.AvailableModels)

	assert.NotContains(t, recorder.Body.String(), token.Key, "public report must never echo the raw key")
	assert.NotContains(t, recorder.Body.String(), "user_id")
	assert.NotContains(t, recorder.Body.String(), "username")
	assert.NotContains(t, recorder.Body.String(), "email")
}

func TestCheckTokenUsageUnknownKeyGetsGenericRejection(t *testing.T) {
	setupTokenControllerTestDB(t)

	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/token/check", map[string]any{"key": "sk-doesnotexist12345"}, 0)
	CheckTokenUsage(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	response := decodeAPIResponse(t, recorder)
	assert.False(t, response.Success)
	assert.NotEmpty(t, response.Message)
}

func TestCheckTokenUsageDisabledKeyStillReturnsReportWithoutMutatingState(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	token := seedToken(t, db, 1, "disabled-key", "disa1234bled5678")
	token.Status = common.TokenStatusDisabled
	require.NoError(t, db.Save(token).Error)

	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/token/check", map[string]any{"key": token.Key}, 0)
	CheckTokenUsage(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	response := decodeAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)

	var report tokenCheckResponse
	require.NoError(t, common.Unmarshal(response.Data, &report))
	assert.Equal(t, common.TokenStatusDisabled, report.Status)

	var reloaded model.Token
	require.NoError(t, db.First(&reloaded, "id = ?", token.Id).Error)
	assert.Equal(t, common.TokenStatusDisabled, reloaded.Status, "the check endpoint must not persist any recomputed status")
}

func TestCheckTokenUsageEmptyKeyReturns400(t *testing.T) {
	setupTokenControllerTestDB(t)

	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/token/check", map[string]any{"key": "   "}, 0)
	CheckTokenUsage(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	response := decodeAPIResponse(t, recorder)
	assert.False(t, response.Success)
}

func TestCheckTokenUsageMalformedBodyReturns400(t *testing.T) {
	setupTokenControllerTestDB(t)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/token/check", strings.NewReader("{not json"))
	ctx.Request.Header.Set("Content-Type", "application/json")

	CheckTokenUsage(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestCheckTokenUsageInheritsUserGroupWhenTokenGroupEmpty(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}))

	owner := &model.User{Username: "owner", Password: "hashed-password", Group: "owner-group"}
	require.NoError(t, db.Create(owner).Error)

	token := seedToken(t, db, owner.Id, "inherited-group-key", "ihgk1234ihgk5678")
	token.Group = ""
	require.NoError(t, db.Save(token).Error)

	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/token/check", map[string]any{"key": token.Key}, 0)
	CheckTokenUsage(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	response := decodeAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)

	var report tokenCheckResponse
	require.NoError(t, common.Unmarshal(response.Data, &report))
	assert.Equal(t, "owner-group", report.Group)
}
