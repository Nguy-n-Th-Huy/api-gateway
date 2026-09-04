package controller

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// performAdminClearUserBindingRequest invokes AdminClearUserBinding directly,
// as the administrator-only route in router/api-router.go would.
func performAdminClearUserBindingRequest(t *testing.T, userId int, bindingType string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/user/"+strconv.Itoa(userId)+"/bindings/"+bindingType, nil)
	c.Params = gin.Params{
		{Key: "id", Value: strconv.Itoa(userId)},
		{Key: "binding_type", Value: bindingType},
	}
	c.Set("id", 1)
	c.Set("role", common.RoleRootUser)
	c.Set("username", "root-operator")

	AdminClearUserBinding(c)
	return recorder
}

// TestAdminClearUserBindingSurfacesLastSignInMethodRefusal covers the
// "Refusals are reported to the operator" requirement: an administrator who
// tries to clear an account's last way in must see why, not a generic error.
func TestAdminClearUserBindingSurfacesLastSignInMethodRefusal(t *testing.T) {
	db := setupManageUserTestDB(t)

	user := model.User{
		Username: "clear-binding-last-method", Password: "", GitHubId: "github-clear-guard",
		Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default",
	}
	require.NoError(t, db.Create(&user).Error)

	recorder := performAdminClearUserBindingRequest(t, user.Id, "github")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	assert.NotContains(t, recorder.Body.String(), model.ErrNoRemainingSignInMethod.Error(),
		"the response must carry the i18n message, not the raw Go error text")
	assert.Contains(t, recorder.Body.String(), "user.binding_removal_would_lock_account",
		"the response must carry the dedicated i18n key for this refusal")

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, user.Id).Error)
	assert.Equal(t, "github-clear-guard", reloaded.GitHubId, "the binding must be left untouched on refusal")
}

func TestAdminClearUserBindingSucceedsWhenAnotherMethodRemains(t *testing.T) {
	db := setupManageUserTestDB(t)

	user := model.User{
		Username: "clear-binding-other-remains", Password: "irrelevant-hash", GitHubId: "github-clear-ok",
		Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default",
	}
	require.NoError(t, db.Create(&user).Error)

	recorder := performAdminClearUserBindingRequest(t, user.Id, "github")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, user.Id).Error)
	assert.Empty(t, reloaded.GitHubId)
}
