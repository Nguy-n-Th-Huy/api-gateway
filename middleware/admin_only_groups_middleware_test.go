package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// initAdminOnlyMiddlewareColumnNames mirrors controller/model_list_test.go's
// initModelListColumnNames: model.InitDB initialises the reserved-word column
// quoting that GetTokenByKey relies on.
func initAdminOnlyMiddlewareColumnNames(t *testing.T) {
	t.Helper()

	originalIsMasterNode := common.IsMasterNode
	originalSQLitePath := common.SQLitePath
	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()
	originalSQLDSN, hadSQLDSN := os.LookupEnv("SQL_DSN")
	defer func() {
		common.IsMasterNode = originalIsMasterNode
		common.SQLitePath = originalSQLitePath
		common.SetDatabaseTypes(originalMainDatabaseType, originalLogDatabaseType)
		if hadSQLDSN {
			require.NoError(t, os.Setenv("SQL_DSN", originalSQLDSN))
		} else {
			require.NoError(t, os.Unsetenv("SQL_DSN"))
		}
	}()

	common.IsMasterNode = false
	common.SQLitePath = fmt.Sprintf("file:%s_init?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, os.Setenv("SQL_DSN", "local"))

	require.NoError(t, model.InitDB())
	if model.DB != nil {
		sqlDB, err := model.DB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}
}

func setupAdminOnlyMiddlewareDB(t *testing.T) *gorm.DB {
	t.Helper()

	initAdminOnlyMiddlewareColumnNames(t)

	gin.SetMode(gin.TestMode)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}))
	require.NoError(t, i18n.Init())

	t.Cleanup(func() {
		sqlDB, closeErr := db.DB()
		if closeErr == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func configureAdminOnlyMiddlewareSettings(t *testing.T, adminOnlyJSON string) {
	t.Helper()

	originalUsable := setting.UserUsableGroups2JSONString()
	originalAdminOnly := setting.AdminOnlyGroups2JsonString()
	originalRatios := ratio_setting.GroupRatio2JSONString()
	special := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup
	originalSpecial := special.ReadAll()

	special.Clear()
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP","secret":"Secret"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1,"secret":1}`))
	require.NoError(t, setting.UpdateAdminOnlyGroupsByJsonString(adminOnlyJSON))

	t.Cleanup(func() {
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsable))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios))
		require.NoError(t, setting.UpdateAdminOnlyGroupsByJsonString(originalAdminOnly))
		special.Clear()
		special.AddAll(originalSpecial)
	})
}

func createAdminOnlyMiddlewareUser(t *testing.T, db *gorm.DB, id int, username string, group string, role int) {
	t.Helper()
	require.NoError(t, db.Create(&model.User{
		Id:          id,
		Username:    username,
		Password:    "password",
		Role:        role,
		Status:      common.UserStatusEnabled,
		Group:       group,
		AccessToken: nil,
		AffCode:     fmt.Sprintf("admin-only-middleware-%s-%d", username, id),
	}).Error)
}

func createAdminOnlyMiddlewareToken(t *testing.T, db *gorm.DB, tokenID int, userID int, tokenKey string, group string) {
	t.Helper()
	require.NoError(t, db.Create(&model.Token{
		Id:             tokenID,
		UserId:         userID,
		Key:            tokenKey,
		Name:           "admin-only-token-" + tokenKey,
		Status:         common.TokenStatusEnabled,
		CreatedTime:    1,
		AccessedTime:   1,
		ExpiredTime:    -1,
		RemainQuota:    100000,
		UnlimitedQuota: true,
		Group:          group,
	}).Error)
}

func newAdminOnlyTokenAuthContext(key string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	ctx.Request.Header.Set("Authorization", "Bearer "+key)
	return ctx, recorder
}

func decodeOpenAIErrorMessage(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body.Error.Message
}

// TestTokenAuthRefusesAdminOnlyTokenGroupForNonAdmin covers spec.md:150-153: a key
// created on a group that is marked admin-only afterwards is refused for a
// non-administrator with the existing forbidden-group response, and keeps working
// for an administrator (spec.md:155-158).
func TestTokenAuthRefusesAdminOnlyTokenGroupForNonAdmin(t *testing.T) {
	setupAdminOnlyMiddlewareDB(t)
	configureAdminOnlyMiddlewareSettings(t, `["secret"]`)

	createAdminOnlyMiddlewareUser(t, model.DB, 4001, "auth-common", "default", common.RoleCommonUser)
	createAdminOnlyMiddlewareUser(t, model.DB, 4002, "auth-admin", "default", common.RoleAdminUser)
	createAdminOnlyMiddlewareToken(t, model.DB, 5001, 4001, "k4001admincommon0000a", "secret")
	createAdminOnlyMiddlewareToken(t, model.DB, 5002, 4002, "k4002adminadmin0000a", "secret")

	commonCtx, commonRecorder := newAdminOnlyTokenAuthContext("k4001admincommon0000a")
	TokenAuth()(commonCtx)
	require.Equal(t, http.StatusForbidden, commonRecorder.Code)
	message := decodeOpenAIErrorMessage(t, commonRecorder)
	assert.Contains(t, message, "无权访问")
	assert.Contains(t, message, "secret")

	adminCtx, adminRecorder := newAdminOnlyTokenAuthContext("k4002adminadmin0000a")
	TokenAuth()(adminCtx)
	assert.NotEqual(t, http.StatusForbidden, adminRecorder.Code)
	assert.NotContains(t, adminRecorder.Body.String(), "无权访问")
}

// TestTokenAuthRefusesAdminOnlyOwnGroupForUngroupedToken covers the case where the
// requester's own user group is admin-only and the token has no group of its own
// (the default). Before this change the usable-group check ran only for named
// token groups, so the owner's group was adopted as the effective group unchecked.
func TestTokenAuthRefusesAdminOnlyOwnGroupForUngroupedToken(t *testing.T) {
	setupAdminOnlyMiddlewareDB(t)
	configureAdminOnlyMiddlewareSettings(t, `["secret"]`)

	createAdminOnlyMiddlewareUser(t, model.DB, 4011, "own-group-common", "secret", common.RoleCommonUser)
	createAdminOnlyMiddlewareToken(t, model.DB, 5011, 4011, "k4011owngrpungroupeda", "")
	createAdminOnlyMiddlewareUser(t, model.DB, 4012, "own-group-admin", "secret", common.RoleAdminUser)
	createAdminOnlyMiddlewareToken(t, model.DB, 5012, 4012, "k4012owngrpungroupeda", "")

	commonCtx, commonRecorder := newAdminOnlyTokenAuthContext("k4011owngrpungroupeda")
	TokenAuth()(commonCtx)
	require.Equal(t, http.StatusForbidden, commonRecorder.Code)
	message := decodeOpenAIErrorMessage(t, commonRecorder)
	assert.Contains(t, message, "无权访问")
	assert.Contains(t, message, "secret")

	adminCtx, adminRecorder := newAdminOnlyTokenAuthContext("k4012owngrpungroupeda")
	TokenAuth()(adminCtx)
	assert.NotEqual(t, http.StatusForbidden, adminRecorder.Code)
	assert.NotContains(t, adminRecorder.Body.String(), "无权访问")
}

func newAdminOnlyPlaygroundContext(t *testing.T, playgroundGroup string, userGroup string, role int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"model": "gpt-4",
		"group": playgroundGroup,
	})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/pg/chat/completions", strings.NewReader(string(body)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, userGroup)
	common.SetContextKey(ctx, constant.ContextKeyUsingGroup, userGroup)
	common.SetContextKey(ctx, constant.ContextKeyUserRole, role)
	return ctx, recorder
}

// TestDistributeRefusesAdminOnlyPlaygroundOverrideForNonAdmin covers spec.md:169-177.
// A refusal is distinguishable without a reachable upstream: 403 for the group
// access refusal, 503 (no channel for the model) for an allowed request.
func TestDistributeRefusesAdminOnlyPlaygroundOverrideForNonAdmin(t *testing.T) {
	setupAdminOnlyMiddlewareDB(t)
	configureAdminOnlyMiddlewareSettings(t, `["secret"]`)

	commonCtx, commonRecorder := newAdminOnlyPlaygroundContext(t, "secret", "default", common.RoleCommonUser)
	Distribute()(commonCtx)
	require.Equal(t, http.StatusForbidden, commonRecorder.Code)
	assert.NotEmpty(t, decodeOpenAIErrorMessage(t, commonRecorder))

	adminCtx, adminRecorder := newAdminOnlyPlaygroundContext(t, "secret", "default", common.RoleAdminUser)
	Distribute()(adminCtx)
	assert.NotEqual(t, http.StatusForbidden, adminRecorder.Code)
}

// TestDistributeRefusesAdminOnlyOwnPlaygroundGroupForNonAdmin pins leak point 2:
// naming one's own group skips the usability check (playgroundRequest.Group ==
// usingGroup), so the gate must sit upstream of that comparison.
func TestDistributeRefusesAdminOnlyOwnPlaygroundGroupForNonAdmin(t *testing.T) {
	setupAdminOnlyMiddlewareDB(t)
	configureAdminOnlyMiddlewareSettings(t, `["secret"]`)

	commonCtx, commonRecorder := newAdminOnlyPlaygroundContext(t, "secret", "secret", common.RoleCommonUser)
	Distribute()(commonCtx)
	require.Equal(t, http.StatusForbidden, commonRecorder.Code)

	adminCtx, adminRecorder := newAdminOnlyPlaygroundContext(t, "secret", "secret", common.RoleAdminUser)
	Distribute()(adminCtx)
	assert.NotEqual(t, http.StatusForbidden, adminRecorder.Code)
}

// TestTokenAuthEmptyMarkingKeepsBehaviourRoleBlind covers the no-op case for the
// enforcement path: with nothing marked, a key on "secret" must behave exactly as
// it did before the capability existed, for both roles.
func TestTokenAuthEmptyMarkingKeepsBehaviourRoleBlind(t *testing.T) {
	setupAdminOnlyMiddlewareDB(t)
	configureAdminOnlyMiddlewareSettings(t, `[]`)

	createAdminOnlyMiddlewareUser(t, model.DB, 4031, "no-op-common", "default", common.RoleCommonUser)
	createAdminOnlyMiddlewareUser(t, model.DB, 4032, "no-op-admin", "default", common.RoleAdminUser)
	createAdminOnlyMiddlewareToken(t, model.DB, 5031, 4031, "k4031nooptokcom0000a", "secret")
	createAdminOnlyMiddlewareToken(t, model.DB, 5032, 4032, "k4032nooptokadm0000a", "secret")

	for _, key := range []string{"k4031nooptokcom0000a", "k4032nooptokadm0000a"} {
		ctx, recorder := newAdminOnlyTokenAuthContext(key)
		TokenAuth()(ctx)
		assert.NotEqual(t, http.StatusForbidden, recorder.Code, "key %s must not be refused with nothing marked", key)
	}
}
