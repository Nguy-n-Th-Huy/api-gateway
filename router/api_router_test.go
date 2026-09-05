package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestSetApiRouterRegistersPublicTokenRoutesWithoutConflict guards against a
// gin route-tree panic when the public POST /api/token/check and GET
// /api/setup/script routes are registered alongside the existing
// authenticated GET /api/token/:id route. Gin's router rejects some
// combinations of static and named-parameter segments at the same path
// depth, so this is a real regression risk whenever a new static segment is
// added under a path that already has a :param sibling (as is the case for
// /api/token/check, one level under the existing /api/token/:id).
func TestSetApiRouterRegistersPublicTokenRoutesWithoutConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)

	require.NotPanics(t, func() {
		engine := gin.New()
		SetApiRouter(engine)

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/token/check", nil)
		engine.ServeHTTP(recorder, req)
		require.NotEqual(t, http.StatusNotFound, recorder.Code, "POST /api/token/check should be routed, not 404")

		recorder2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodGet, "/api/setup/script", nil)
		engine.ServeHTTP(recorder2, req2)
		require.NotEqual(t, http.StatusNotFound, recorder2.Code, "GET /api/setup/script should be routed, not 404")
	})
}
