package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestTelegramUserRateLimitIsolatesIdentifiers covers 10.10: exhausting the
// limit for one Telegram identifier must not impede requests for a different
// one (specs/telegram/bot-api/spec.md, "Rate limiting is scoped to the
// Telegram user, not the caller IP"). Every request in this test arrives
// from the same client address, which is the whole point being proven.
func TestTelegramUserRateLimitIsolatesIdentifiers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	useRateLimitMiniRedis(t)

	originalEnable := common.CriticalRateLimitEnable
	originalNum := common.CriticalRateLimitNum
	originalDuration := common.CriticalRateLimitDuration
	t.Cleanup(func() {
		common.CriticalRateLimitEnable = originalEnable
		common.CriticalRateLimitNum = originalNum
		common.CriticalRateLimitDuration = originalDuration
	})
	common.CriticalRateLimitEnable = true
	common.CriticalRateLimitNum = 1
	common.CriticalRateLimitDuration = 60

	router := gin.New()
	router.GET("/bot", func(c *gin.Context) {
		telegramUserID := c.Query("telegram_user_id")
		if !TelegramUserRateLimit(c, telegramUserID) {
			return
		}
		c.Status(http.StatusOK)
	})

	request := func(telegramUserID string) int {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/bot?telegram_user_id="+telegramUserID, nil)
		req.RemoteAddr = "203.0.113.5:12345" // same client address for every request
		router.ServeHTTP(recorder, req)
		return recorder.Code
	}

	assert.Equal(t, http.StatusOK, request("tg-alice"), "alice's first request is allowed")
	assert.Equal(t, http.StatusTooManyRequests, request("tg-alice"), "alice's second request exhausts her limit")
	assert.Equal(t, http.StatusOK, request("tg-bob"), "bob is unaffected by alice's exhausted limit")
	assert.Equal(t, http.StatusOK, request("tg-carol"), "carol is unaffected too")
	assert.Equal(t, http.StatusTooManyRequests, request("tg-alice"), "alice remains limited")
}

// TestTelegramUserRateLimitNoOpWithoutIdentifier reports that a blank
// telegram_user_id (the health endpoint's shape, which never resolves one)
// is let through rather than limited, matching "apply the ordinary global
// limit to the endpoint that takes no Telegram user".
func TestTelegramUserRateLimitNoOpWithoutIdentifier(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalEnable := common.CriticalRateLimitEnable
	t.Cleanup(func() { common.CriticalRateLimitEnable = originalEnable })
	common.CriticalRateLimitEnable = true

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/bot", nil)

	assert.True(t, TelegramUserRateLimit(ctx, ""))
}
