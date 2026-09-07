package middleware

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
)

const telegramBotAuthScheme = "Bot "

// TelegramBotAuth gates every /api/bot/v1 request. While the integration is
// disabled or unkeyed, every request is rejected with 404 so an unconfigured
// deployment exposes no discoverable surface (specs/telegram/bot-api/spec.md,
// "The bot API is unavailable until it is configured and enabled"). Once
// configured, the presented "Authorization: Bot <key>" credential is compared
// against the configured service key with crypto/subtle, so a missing,
// malformed, or wrong credential is rejected through the same comparison and
// cannot be distinguished by the caller ("Bot service key authentication").
// A session token or personal access token is rejected identically: neither
// carries the "Bot " scheme this surface requires.
func TelegramBotAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		if !setting.IsTelegramBotConfigured() {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		if !telegramBotCredentialMatches(c.GetHeader("Authorization")) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized"})
			return
		}
		c.Next()
	}
}

// telegramBotCredentialMatches reports whether header carries
// "Authorization: Bot <key>" for the currently configured service key.
func telegramBotCredentialMatches(header string) bool {
	if !strings.HasPrefix(header, telegramBotAuthScheme) {
		return false
	}
	provided := strings.TrimPrefix(header, telegramBotAuthScheme)
	return subtle.ConstantTimeCompare([]byte(provided), []byte(setting.TelegramBotServiceKey)) == 1
}

// TelegramUserRateLimit enforces the ordinary critical rate limit keyed by
// the Telegram account identifier rather than the caller's IP: every bot
// request arrives from the same host, so an IP-scoped limit would let one
// abusive Telegram user exhaust the budget for every other Telegram user
// (design.md, "Rate limiting keyed by Telegram user"; specs/telegram/bot-api/
// spec.md, "Rate limiting is scoped to the Telegram user, not the caller
// IP"). It delegates to the same fixed-window Redis limiter (and in-memory
// fallback) every other limiter in this package uses, keyed by the supplied
// identifier through userRedisRateLimiter's pre-built-key seam rather than by
// a numeric user id.
//
// Returns false, having already written the 429 response, when the caller
// must stop handling the request. A blank identifier is let through: the
// caller is responsible for resolving telegram_user_id before this is
// reached, and the health endpoint (which never resolves one) uses the
// ordinary global limiter instead.
func TelegramUserRateLimit(c *gin.Context, telegramUserID string) bool {
	if !common.CriticalRateLimitEnable || telegramUserID == "" {
		return true
	}
	if common.RedisEnabled {
		key := fmt.Sprintf("%s:tguser:%s", redisRateLimitNamespace, telegramUserID)
		userRedisRateLimiter(c, common.CriticalRateLimitNum, common.CriticalRateLimitDuration, key)
		return !c.IsAborted()
	}
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
	key := fmt.Sprintf("TGUSER:%s", telegramUserID)
	if !inMemoryRateLimiter.Request(key, common.CriticalRateLimitNum, common.CriticalRateLimitDuration) {
		writeRateLimited(c, common.CriticalRateLimitDuration)
		return false
	}
	return true
}
