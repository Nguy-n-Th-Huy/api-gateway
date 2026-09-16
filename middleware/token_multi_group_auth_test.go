package middleware

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// createMultiGroupMiddlewareToken stores a token with an explicit auto_groups
// snapshot: the middleware fixture has no helper for the snapshot column because
// every pre-existing case stores a single group or none at all.
func createMultiGroupMiddlewareToken(t *testing.T, db *gorm.DB, tokenID int, userID int, tokenKey string, group string, autoGroups string) {
	t.Helper()
	require.NoError(t, db.Create(&model.Token{
		Id:             tokenID,
		UserId:         userID,
		Key:            tokenKey,
		Name:           "multi-group-token-" + tokenKey,
		Status:         common.TokenStatusEnabled,
		CreatedTime:    1,
		AccessedTime:   1,
		ExpiredTime:    -1,
		RemainQuota:    100000,
		UnlimitedQuota: true,
		Group:          group,
		AutoGroups:     autoGroups,
	}).Error)
}

// setMultiGroupMiddlewareFixtures pins the shared configuration of this file's
// cases: "auto" is deliberately absent from the usable groups of every requester
// group, which is exactly the owner the multi-group console produces a key for.
func setMultiGroupMiddlewareFixtures(t *testing.T) {
	t.Helper()

	setupAdminOnlyMiddlewareDB(t)
	configureAdminOnlyMiddlewareSettings(t, `[]`)

	require.NotContains(t,
		service.GetUserUsableGroups("default", common.RoleCommonUser),
		"auto",
		"fixture invariant: the requester must not have auto in its usable groups",
	)
}

// TestTokenAuthAcceptsSnapshotBackedAutoWithoutUsableAuto covers the relaxation
// itself: a key stored as group == "auto" with an ordered snapshot authenticates
// for an owner whose usable groups exclude auto, and the request context carries
// the snapshot so the relay path can route on it.
func TestTokenAuthAcceptsSnapshotBackedAutoWithoutUsableAuto(t *testing.T) {
	setMultiGroupMiddlewareFixtures(t)

	createAdminOnlyMiddlewareUser(t, model.DB, 6001, "multi-group-owner", "default", common.RoleCommonUser)
	createMultiGroupMiddlewareToken(t, model.DB, 7001, 6001, "k7001multigroupsnap0", "auto", `["vip","default"]`)

	ctx, recorder := newAdminOnlyTokenAuthContext("k7001multigroupsnap0")
	TokenAuth()(ctx)

	require.NotEqual(t, http.StatusForbidden, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "无权访问")

	usingGroup, ok := common.GetContextKey(ctx, constant.ContextKeyUsingGroup)
	require.True(t, ok)
	assert.Equal(t, "auto", usingGroup)

	storedGroup, ok := common.GetContextKey(ctx, constant.ContextKeyTokenGroup)
	require.True(t, ok)
	assert.Equal(t, "auto", storedGroup)

	autoGroups, ok := common.GetContextKey(ctx, constant.ContextKeyTokenAutoGroups)
	require.True(t, ok)
	assert.Equal(t, []string{"vip", "default"}, autoGroups)
}

// TestTokenAuthRefusesAutoWithoutSnapshotWhenAutoUnusable pins the boundary of
// the relaxation: the same key shape without a snapshot is a global-Auto key and
// stays refused for an owner who may not select auto.
func TestTokenAuthRefusesAutoWithoutSnapshotWhenAutoUnusable(t *testing.T) {
	setMultiGroupMiddlewareFixtures(t)

	createAdminOnlyMiddlewareUser(t, model.DB, 6011, "global-auto-owner", "default", common.RoleCommonUser)
	createMultiGroupMiddlewareToken(t, model.DB, 7011, 6011, "k7011globalautonone0", "auto", "")

	ctx, recorder := newAdminOnlyTokenAuthContext("k7011globalautonone0")
	TokenAuth()(ctx)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	message := decodeOpenAIErrorMessage(t, recorder)
	assert.Contains(t, message, "无权访问")
	assert.Contains(t, message, "auto")
}

// TestTokenAuthRefusesAutoWithUnusableSnapshot covers the fail-closed cases: an
// empty array and an unparsable value are both "no snapshot", so neither may
// authenticate a key whose owner cannot select auto.
func TestTokenAuthRefusesAutoWithUnusableSnapshot(t *testing.T) {
	setMultiGroupMiddlewareFixtures(t)

	createAdminOnlyMiddlewareUser(t, model.DB, 6021, "empty-snapshot-owner", "default", common.RoleCommonUser)
	createMultiGroupMiddlewareToken(t, model.DB, 7021, 6021, "k7021emptysnapshot0", "auto", `[]`)
	createAdminOnlyMiddlewareUser(t, model.DB, 6022, "malformed-snapshot-owner", "default", common.RoleCommonUser)
	createMultiGroupMiddlewareToken(t, model.DB, 7022, 6022, "k7022malformedsnap0", "auto", `not-json`)

	for _, key := range []string{"k7021emptysnapshot0", "k7022malformedsnap0"} {
		ctx, recorder := newAdminOnlyTokenAuthContext(key)
		TokenAuth()(ctx)
		require.Equal(t, http.StatusForbidden, recorder.Code, "key %s must stay refused", key)
		assert.Contains(t, recorder.Body.String(), "无权访问")
	}
}

// TestTokenAuthKeepsSingleGroupAndUngroupedKeysUnchanged proves the relaxation
// reaches no other shape: a usable single group still passes, an unusable one is
// still refused, the same stored snapshot on a non-auto group grants nothing, and
// an ungrouped key keeps inheriting its owner's group.
func TestTokenAuthKeepsSingleGroupAndUngroupedKeysUnchanged(t *testing.T) {
	setMultiGroupMiddlewareFixtures(t)

	createAdminOnlyMiddlewareUser(t, model.DB, 6031, "single-group-owner", "default", common.RoleCommonUser)
	createMultiGroupMiddlewareToken(t, model.DB, 7031, 6031, "k7031singleusablevg0", "vip", "")
	createMultiGroupMiddlewareToken(t, model.DB, 7032, 6031, "k7032singleunusable0", "revoked", "")
	createMultiGroupMiddlewareToken(t, model.DB, 7033, 6031, "k7033snapshotnona0", "revoked", `["vip","default"]`)
	createMultiGroupMiddlewareToken(t, model.DB, 7034, 6031, "k7034ungroupedkey0", "", "")

	usableCtx, usableRecorder := newAdminOnlyTokenAuthContext("k7031singleusablevg0")
	TokenAuth()(usableCtx)
	require.NotEqual(t, http.StatusForbidden, usableRecorder.Code)
	usingGroup, ok := common.GetContextKey(usableCtx, constant.ContextKeyUsingGroup)
	require.True(t, ok)
	assert.Equal(t, "vip", usingGroup)

	ungroupedCtx, ungroupedRecorder := newAdminOnlyTokenAuthContext("k7034ungroupedkey0")
	TokenAuth()(ungroupedCtx)
	require.NotEqual(t, http.StatusForbidden, ungroupedRecorder.Code)
	usingGroup, ok = common.GetContextKey(ungroupedCtx, constant.ContextKeyUsingGroup)
	require.True(t, ok)
	assert.Equal(t, "default", usingGroup)

	for _, key := range []string{"k7032singleunusable0", "k7033snapshotnona0"} {
		ctx, recorder := newAdminOnlyTokenAuthContext(key)
		TokenAuth()(ctx)
		require.Equal(t, http.StatusForbidden, recorder.Code, "key %s must stay refused", key)
		assert.Contains(t, recorder.Body.String(), "无权访问")
	}
}

// TestTokenAuthStillRefusesAdminOnlyEffectiveGroupForSnapshotBackedAuto pins the
// gate order: the relaxation replaces only the usable-group lookup for auto, so
// the admin-only refusal on the effective group still fires for a non-admin.
func TestTokenAuthStillRefusesAdminOnlyEffectiveGroupForSnapshotBackedAuto(t *testing.T) {
	setupAdminOnlyMiddlewareDB(t)
	configureAdminOnlyMiddlewareSettings(t, `["auto"]`)

	createAdminOnlyMiddlewareUser(t, model.DB, 6041, "admin-only-auto-common", "default", common.RoleCommonUser)
	createMultiGroupMiddlewareToken(t, model.DB, 7041, 6041, "k7041adminonlyauto0", "auto", `["vip","default"]`)
	createAdminOnlyMiddlewareUser(t, model.DB, 6042, "admin-only-auto-admin", "default", common.RoleAdminUser)
	createMultiGroupMiddlewareToken(t, model.DB, 7042, 6042, "k7042adminonlyautoa0", "auto", `["vip","default"]`)

	commonCtx, commonRecorder := newAdminOnlyTokenAuthContext("k7041adminonlyauto0")
	TokenAuth()(commonCtx)
	require.Equal(t, http.StatusForbidden, commonRecorder.Code)

	adminCtx, adminRecorder := newAdminOnlyTokenAuthContext("k7042adminonlyautoa0")
	TokenAuth()(adminCtx)
	assert.NotEqual(t, http.StatusForbidden, adminRecorder.Code)
}
