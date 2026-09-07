package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-gonic/gin"
)

// SetBotApiRouter mounts the service-to-service surface an externally hosted
// Telegram bot calls on behalf of a Telegram user
// (specs/telegram/bot-api/spec.md). Every route sits behind
// middleware.TelegramBotAuth(), which returns 404 while the integration is
// disabled or unkeyed and 401 for any credential that does not match — so an
// unconfigured deployment exposes no discoverable surface at all. There is
// deliberately no unlink route and no code-redemption route on this group
// (specs/telegram/account-link/spec.md, "Redemption requires an
// authenticated session"): those stay on the user-authenticated website
// surface in SetApiRouter.
func SetBotApiRouter(router *gin.Engine) {
	botRouter := router.Group("/api/bot/v1")
	botRouter.Use(middleware.RouteTag("bot-api"))
	botRouter.Use(middleware.TelegramBotAuth())
	{
		botRouter.GET("/health", middleware.CriticalRateLimit(), controller.TelegramBotHealth)
		botRouter.POST("/identity/resolve", controller.TelegramBotIdentity)
		botRouter.POST("/identity/link/start", controller.TelegramBotLinkStart)
		botRouter.GET("/account", controller.TelegramBotAccount)
		botRouter.GET("/topup/config", controller.TelegramBotTopUpConfig)
		botRouter.POST("/topup/orders", controller.TelegramBotCreateOrder)
		botRouter.GET("/topup/orders/:trade_no", controller.TelegramBotOrderStatus)
		botRouter.GET("/topup/orders", controller.TelegramBotTopUpHistory)
		botRouter.POST("/keys/check", controller.TelegramBotKeyInspect)
		botRouter.GET("/keys", controller.TelegramBotKeyList)
		botRouter.GET("/logs", controller.TelegramBotUsageLogs)
		botRouter.GET("/logs/stat", controller.TelegramBotUsageStats)
		botRouter.POST("/keys/logs", controller.TelegramBotKeyLogs)
	}
}
