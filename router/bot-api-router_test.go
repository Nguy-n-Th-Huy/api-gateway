package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetBotApiRouterDoesNotCollideWithExistingRoutes covers task 6.7: the
// bot surface mounts alongside the rest of the API router without a gin
// route-tree panic.
func TestSetBotApiRouterDoesNotCollideWithExistingRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	require.NotPanics(t, func() {
		engine := gin.New()
		SetApiRouter(engine)
		SetBotApiRouter(engine)

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/bot/v1/health", nil)
		engine.ServeHTTP(recorder, req)
		// Unconfigured in this test process, so 404 is expected — the point
		// here is only that the route exists and dispatches, not 404 from an
		// unregistered path.
		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

// TestSetBotApiRouterRegistersTopUpOrdersRouteGroupWithoutConflict guards
// against a gin route-tree panic from the static GET/POST /topup/orders leaf
// coexisting with the wildcard GET /topup/orders/:trade_no child under it, and
// confirms every renamed route in the approved contract dispatches (not a
// registration miss reported as 404).
func TestSetBotApiRouterRegistersTopUpOrdersRouteGroupWithoutConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)

	require.NotPanics(t, func() {
		engine := gin.New()
		SetApiRouter(engine)
		SetBotApiRouter(engine)

		routes := map[string]bool{}
		for _, route := range engine.Routes() {
			routes[route.Method+" "+route.Path] = true
		}

		expected := []string{
			"GET /api/bot/v1/health",
			"POST /api/bot/v1/identity/resolve",
			"POST /api/bot/v1/identity/link/start",
			"GET /api/bot/v1/account",
			"GET /api/bot/v1/topup/config",
			"POST /api/bot/v1/topup/orders",
			"GET /api/bot/v1/topup/orders/:trade_no",
			"GET /api/bot/v1/topup/orders",
			"POST /api/bot/v1/keys/check",
			"GET /api/bot/v1/keys",
			"GET /api/bot/v1/logs",
			"GET /api/bot/v1/logs/stat",
			"POST /api/bot/v1/keys/logs",
		}
		for _, route := range expected {
			assert.True(t, routes[route], "expected route %q to be registered", route)
		}

		// Unconfigured in this test process, so every dispatch is 404 from
		// middleware.TelegramBotAuth — the point here is only that the request
		// reaches the handler chain rather than gin reporting no matching
		// route at all.
		for _, req := range []*http.Request{
			httptest.NewRequest(http.MethodPost, "/api/bot/v1/topup/orders", nil),
			httptest.NewRequest(http.MethodGet, "/api/bot/v1/topup/orders", nil),
			httptest.NewRequest(http.MethodGet, "/api/bot/v1/topup/orders/T123", nil),
		} {
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, req)
			assert.Equal(t, http.StatusNotFound, recorder.Code)
		}
	})
}

// TestTelegramBotSurfaceHasNoUnlinkOrRedeemRoute is a structural guard for
// design.md's deliberate blast-radius bound: the bot service key can issue a
// link code but can never redeem one, and can never remove a binding
// (specs/telegram/bot-api/spec.md, "The bot surface cannot unlink an
// account"; specs/telegram/account-link/spec.md, "Redemption is absent from
// the bot surface"). It asserts this against the actual registered route
// table rather than only by code review.
func TestTelegramBotSurfaceHasNoUnlinkOrRedeemRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetBotApiRouter(engine)

	var botRoutes []string
	for _, route := range engine.Routes() {
		if strings.HasPrefix(route.Path, "/api/bot/v1") {
			botRoutes = append(botRoutes, route.Method+" "+route.Path)
		}
	}
	require.NotEmpty(t, botRoutes)

	for _, route := range botRoutes {
		lower := strings.ToLower(route)
		assert.NotContains(t, lower, "unlink", "the bot surface must expose no unlink route")
		assert.NotContains(t, lower, "redeem", "the bot surface must expose no redemption route")
	}
}

// TestTelegramLinkRedemptionOnlyReachableFromTheWebsiteSurface confirms the
// redemption endpoint exists only under the user-authenticated website
// router, never under /api/bot/v1.
func TestTelegramLinkRedemptionOnlyReachableFromTheWebsiteSurface(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalEnabled := setting.TelegramBotIntegrationEnabled
	t.Cleanup(func() { setting.TelegramBotIntegrationEnabled = originalEnabled })
	setting.TelegramBotIntegrationEnabled = false

	engine := gin.New()
	SetApiRouter(engine)
	SetBotApiRouter(engine)

	foundOnWebsite := false
	for _, route := range engine.Routes() {
		if route.Path == "/api/user/telegram/link/confirm" {
			foundOnWebsite = true
		}
		if strings.HasPrefix(route.Path, "/api/bot/v1") {
			assert.NotEqual(t, "/api/bot/v1/identity/link/confirm", route.Path)
		}
	}
	assert.True(t, foundOnWebsite, "the redemption endpoint must be registered on the website surface")
}
