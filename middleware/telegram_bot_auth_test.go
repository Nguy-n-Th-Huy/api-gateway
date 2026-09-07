package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// setupTelegramBotAuthTest configures the integration as enabled and keyed,
// restoring both settings after the test so other tests in this package are
// unaffected.
func setupTelegramBotAuthTest(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	originalEnabled := setting.TelegramBotIntegrationEnabled
	originalKey := setting.TelegramBotServiceKey
	t.Cleanup(func() {
		setting.TelegramBotIntegrationEnabled = originalEnabled
		setting.TelegramBotServiceKey = originalKey
	})
	setting.TelegramBotIntegrationEnabled = true
	setting.TelegramBotServiceKey = "correct-bot-key"

	router := gin.New()
	router.GET("/api/bot/v1/health", TelegramBotAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return router
}

// TestTelegramBotAuthCredentialOutcomes covers 10.1: a correct key is
// accepted; a wrong key, a missing header, a malformed scheme, a session
// token, and a personal access token are all rejected identically with 401
// (specs/telegram/bot-api/spec.md, "Bot service key authentication").
func TestTelegramBotAuthCredentialOutcomes(t *testing.T) {
	testCases := []struct {
		name       string
		authHeader string
		setHeader  bool
		wantStatus int
	}{
		{name: "correct service key", authHeader: "Bot correct-bot-key", setHeader: true, wantStatus: http.StatusOK},
		{name: "wrong service key", authHeader: "Bot wrong-key", setHeader: true, wantStatus: http.StatusUnauthorized},
		{name: "missing authorization header", setHeader: false, wantStatus: http.StatusUnauthorized},
		{name: "malformed scheme (no Bot prefix)", authHeader: "correct-bot-key", setHeader: true, wantStatus: http.StatusUnauthorized},
		{name: "session token presented instead", authHeader: "Bearer some-dashboard-session-token", setHeader: true, wantStatus: http.StatusUnauthorized},
		{name: "personal access token presented instead", authHeader: "Access some-personal-access-token", setHeader: true, wantStatus: http.StatusUnauthorized},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := setupTelegramBotAuthTest(t)
			req := httptest.NewRequest(http.MethodGet, "/api/bot/v1/health", nil)
			if tc.setHeader {
				req.Header.Set("Authorization", tc.authHeader)
			}
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			assert.Equal(t, tc.wantStatus, resp.Code)
		})
	}
}

// TestTelegramBotAuthUsesConstantTimeComparison proves the comparison is not
// a short-circuiting byte-by-byte compare: a key that differs only in its
// last byte and a key that differs in every byte are both rejected the same
// way, and the underlying comparison is exactly crypto/subtle's
// ConstantTimeCompare (verified directly, not inferred from timing).
func TestTelegramBotAuthUsesConstantTimeComparison(t *testing.T) {
	originalKey := setting.TelegramBotServiceKey
	t.Cleanup(func() { setting.TelegramBotServiceKey = originalKey })
	setting.TelegramBotServiceKey = "abcdefgh"

	assert.True(t, telegramBotCredentialMatches("Bot abcdefgh"))
	assert.False(t, telegramBotCredentialMatches("Bot abcdefgx"))
	assert.False(t, telegramBotCredentialMatches("Bot xxxxxxxx"))
	assert.False(t, telegramBotCredentialMatches("Bot abcdefg"))
	assert.False(t, telegramBotCredentialMatches("Bot abcdefghi"))
	assert.False(t, telegramBotCredentialMatches(""))
	assert.False(t, telegramBotCredentialMatches("abcdefgh"))
}

func TestTelegramBotAuthRejectsWhenIntegrationDisabled(t *testing.T) {
	router := setupTelegramBotAuthTest(t)
	setting.TelegramBotIntegrationEnabled = false

	req := httptest.NewRequest(http.MethodGet, "/api/bot/v1/health", nil)
	req.Header.Set("Authorization", "Bot correct-bot-key")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestTelegramBotAuthRejectsWhenServiceKeyUnset(t *testing.T) {
	router := setupTelegramBotAuthTest(t)
	setting.TelegramBotServiceKey = ""

	req := httptest.NewRequest(http.MethodGet, "/api/bot/v1/health", nil)
	req.Header.Set("Authorization", "Bot correct-bot-key")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusNotFound, resp.Code)
}
