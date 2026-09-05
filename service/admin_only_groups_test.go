package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configureAdminOnlyGroupsTest installs the group, ratio and admin-only
// configuration the admin-only resolution scenarios need, and restores every
// mutated global setting on cleanup so the fixtures stay order-independent, the
// way the neighbouring group tests do.
func configureAdminOnlyGroupsTest(t *testing.T, adminOnlyJSON string) {
	t.Helper()

	originalUsable := setting.UserUsableGroups2JSONString()
	originalAdminOnly := setting.AdminOnlyGroups2JsonString()
	originalRatios := ratio_setting.GroupRatio2JSONString()
	originalSpecial := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.ReadAll()

	// Clear the per-user-group overrides at setup, not just at teardown: a fixture
	// must not depend on the previous test having cleaned up after itself.
	special := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup
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

func TestGetUserUsableGroupsRoleThreshold(t *testing.T) {
	cases := []struct {
		name          string
		userGroup     string
		role          int
		wantSecret    bool
		wantSecretDes string
	}{
		{"administrator sees admin-only group", "default", common.RoleAdminUser, true, "Secret"},
		{"root sees admin-only group", "default", common.RoleRootUser, true, "Secret"},
		{"ordinary user does not see admin-only group", "default", common.RoleCommonUser, false, ""},
		{"anonymous role 0 does not see admin-only group", "default", common.RoleGuestUser, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configureAdminOnlyGroupsTest(t, `["secret"]`)

			groups := GetUserUsableGroups(tc.userGroup, tc.role)

			// Non-admin-only groups are always present for both roles.
			assert.Contains(t, groups, "default")
			assert.Contains(t, groups, "vip")
			if tc.wantSecret {
				assert.Equal(t, tc.wantSecretDes, groups["secret"])
			} else {
				assert.NotContains(t, groups, "secret")
			}
		})
	}
}

func TestGetUserUsableGroupsPlusOverrideCannotRestoreAdminOnlyGroup(t *testing.T) {
	configureAdminOnlyGroupsTest(t, `["secret"]`)
	ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.Set("default", map[string]string{"+:secret": "granted"})

	groups := GetUserUsableGroups("default", common.RoleCommonUser)

	assert.NotContains(t, groups, "secret")
	// The override still applies for an administrator, proving the filter runs
	// after the override rather than being bypassed entirely.
	adminGroups := GetUserUsableGroups("default", common.RoleAdminUser)
	assert.Contains(t, adminGroups, "secret")
}

func TestGetUserUsableGroupsOwnGroupCannotRestoreAdminOnlyGroup(t *testing.T) {
	configureAdminOnlyGroupsTest(t, `["secret"]`)

	// "secret" is the requester's own user group. The own-group rule would grant
	// it, but the last filter must remove it for a non-administrator.
	groups := GetUserUsableGroups("secret", common.RoleCommonUser)

	assert.NotContains(t, groups, "secret")
	// An administrator keeps their own admin-only group.
	assert.Contains(t, GetUserUsableGroups("secret", common.RoleAdminUser), "secret")
}

func TestGetUserUsableGroupsInertAdminOnlyNameChangesNothing(t *testing.T) {
	// "ghost" is marked admin-only but is not in the registry at all.
	configureAdminOnlyGroupsTest(t, `["ghost"]`)

	nonAdmin := GetUserUsableGroups("default", common.RoleCommonUser)
	admin := GetUserUsableGroups("default", common.RoleAdminUser)

	assert.NotContains(t, nonAdmin, "ghost")
	assert.NotContains(t, admin, "ghost")
	assert.Equal(t, admin, nonAdmin)
}

func TestGetUserUsableGroupsEmptyConfigIsRoleBlind(t *testing.T) {
	configureAdminOnlyGroupsTest(t, `[]`)

	// With nothing marked, every role resolves the identical map: byte-identical
	// to pre-feature behaviour.
	commonUser := GetUserUsableGroups("default", common.RoleCommonUser)
	admin := GetUserUsableGroups("default", common.RoleAdminUser)

	assert.Equal(t, map[string]string{"default": "Default", "vip": "VIP", "secret": "Secret"}, commonUser)
	assert.Equal(t, admin, commonUser)
}

func TestGetUserUsableGroupsMalformedConfigIsRoleBlind(t *testing.T) {
	// Malformed JSON leaves the admin-only list empty rather than erroring out of
	// option loading, so resolution stays role-blind. The fixture runs first so it
	// captures — and later restores — the real pre-test configuration.
	configureAdminOnlyGroupsTest(t, `[]`)

	require.Error(t, setting.UpdateAdminOnlyGroupsByJsonString(`{"not":"an array"`))
	require.NoError(t, setting.UpdateAdminOnlyGroupsByJsonString(`["secret"]`))
	require.Error(t, setting.UpdateAdminOnlyGroupsByJsonString(`not json at all`))
	assert.Empty(t, setting.GetAdminOnlyGroups())

	assert.Equal(t, GetUserUsableGroups("default", common.RoleAdminUser), GetUserUsableGroups("default", common.RoleCommonUser))
}

func newAdminOnlyGroupsContext(role int) *gin.Context {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(ctx, constant.ContextKeyUserRole, role)
	return ctx
}

func newAdminOnlyGroupsContextWithoutRole() *gin.Context {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	return ctx
}

func TestGetRequestAutoGroupsSkipsAdminOnlyGroupForNonAdmin(t *testing.T) {
	originalUsable := setting.UserUsableGroups2JSONString()
	originalAdminOnly := setting.AdminOnlyGroups2JsonString()
	originalRatios := ratio_setting.GroupRatio2JSONString()
	originalAuto := setting.AutoGroups2JsonString()
	originalSpecial := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.ReadAll()
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsable))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios))
		require.NoError(t, setting.UpdateAdminOnlyGroupsByJsonString(originalAdminOnly))
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(originalAuto))
		special := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup
		special.Clear()
		special.AddAll(originalSpecial)
	})

	special := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup
	special.Clear()
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP","secret":"Secret"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1,"secret":1}`))
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["secret","vip","default"]`))
	require.NoError(t, setting.UpdateAdminOnlyGroupsByJsonString(`["secret"]`))

	// A key on `auto` whose global order contains an admin-only group must skip it
	// for an ordinary user and include it for an administrator.
	assert.Equal(t, []string{"vip", "default"}, GetRequestAutoGroups(newAdminOnlyGroupsContext(common.RoleCommonUser), "default"))
	assert.Equal(t, []string{"secret", "vip", "default"}, GetRequestAutoGroups(newAdminOnlyGroupsContext(common.RoleAdminUser), "default"))
	// RoleGuestUser written explicitly onto the context behaves the same way.
	assert.Equal(t, []string{"vip", "default"}, GetRequestAutoGroups(newAdminOnlyGroupsContext(common.RoleGuestUser), "default"))
	// A request that carries no role key at all — what an anonymous request
	// actually looks like — resolves as the lowest role and is filtered too.
	assert.Equal(t, []string{"vip", "default"}, GetRequestAutoGroups(newAdminOnlyGroupsContextWithoutRole(), "default"))
}
