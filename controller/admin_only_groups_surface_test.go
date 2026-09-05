package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// The admin-only surface fixtures mark "secret" and give it an exclusive model, so
// a leaked name, description, ratio or model is always detectable by string.

type groupListResponse struct {
	Success bool                      `json:"success"`
	Message string                    `json:"message"`
	Data    map[string]map[string]any `json:"data"`
}

type pricingCatalogueResponse struct {
	Success     bool               `json:"success"`
	Data        []model.Pricing    `json:"data"`
	GroupRatio  map[string]float64 `json:"group_ratio"`
	UsableGroup map[string]string  `json:"usable_group"`
	AutoGroups  []string           `json:"auto_groups"`
}

func configureAdminOnlyGroupSurfaces(t *testing.T, adminOnlyJSON string) *gorm.DB {
	t.Helper()

	originalUsable := setting.UserUsableGroups2JSONString()
	originalAdminOnly := setting.AdminOnlyGroups2JsonString()
	originalRatios := ratio_setting.GroupRatio2JSONString()
	originalAuto := setting.AutoGroups2JsonString()
	originalMax := setting.GetMaxTokenAutoGroups()
	special := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup
	originalSpecial := special.ReadAll()

	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Token{}))

	// Clear the per-user-group overrides at setup so the fixture does not depend on
	// another test having cleaned up after itself.
	special.Clear()
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"auto":"Auto","default":"Default","vip":"VIP","secret":"Secret"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"auto":1,"default":1,"vip":1,"secret":1}`))
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["secret","vip","default"]`))
	require.NoError(t, setting.UpdateMaxTokenAutoGroups("5"))
	require.NoError(t, setting.UpdateAdminOnlyGroupsByJsonString(adminOnlyJSON))

	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "secret", Model: "zz-secret-only-model", ChannelId: 1, Enabled: true},
		{Group: "secret", Model: "zz-shared-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-default-model", ChannelId: 2, Enabled: true},
		{Group: "default", Model: "zz-shared-model", ChannelId: 2, Enabled: true},
	}).Error)

	originalMemoryCache := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	model.InvalidatePricingCache()

	t.Cleanup(func() {
		model.InvalidatePricingCache()
		common.MemoryCacheEnabled = originalMemoryCache
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsable))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios))
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(originalAuto))
		require.NoError(t, setting.UpdateMaxTokenAutoGroups(fmt.Sprintf("%d", originalMax)))
		require.NoError(t, setting.UpdateAdminOnlyGroupsByJsonString(originalAdminOnly))
		special.Clear()
		special.AddAll(originalSpecial)
	})

	return db
}

func createAdminOnlySurfaceUser(t *testing.T, db *gorm.DB, id int, username string, group string, role int) {
	t.Helper()
	require.NoError(t, db.Create(&model.User{
		Id:       id,
		Username: username,
		Password: "password",
		Group:    group,
		Role:     role,
		Status:   common.UserStatusEnabled,
		// aff_code carries a unique index, so each fixture user needs its own value.
		AffCode: "admin-only-surface-" + username,
	}).Error)
}

func decodeGroupListResponse(t *testing.T, recorder *httptest.ResponseRecorder) groupListResponse {
	t.Helper()

	var payload groupListResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success, payload.Message)
	return payload
}

func decodePricingCatalogue(t *testing.T, recorder *httptest.ResponseRecorder) pricingCatalogueResponse {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload pricingCatalogueResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	return payload
}

func newAdminOnlyGroupListContext(t *testing.T, userID int, userGroup string, role int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/groups", nil)
	if userID != 0 {
		ctx.Set("id", userID)
	}
	if userGroup != "" {
		common.SetContextKey(ctx, constant.ContextKeyUserGroup, userGroup)
	}
	if role >= 0 {
		common.SetContextKey(ctx, constant.ContextKeyUserRole, role)
	}
	return ctx, recorder
}

func newAdminOnlyPricingContext(t *testing.T, userID int, role int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/pricing", nil)
	if userID != 0 {
		ctx.Set("id", userID)
	}
	if role >= 0 {
		common.SetContextKey(ctx, constant.ContextKeyUserRole, role)
	}
	return ctx, recorder
}

// TestGetUserGroupsAdminOnlySurface covers the group list behind the API-key form
// for an ordinary user, an administrator and an unauthenticated request, plus the
// no-op case where nothing is marked.
func TestGetUserGroupsAdminOnlySurface(t *testing.T) {
	cases := []struct {
		name          string
		adminOnlyJSON string
		// secretVisibleToNonAdmin is false only when the group is marked.
		secretVisibleToNonAdmin bool
	}{
		{name: "marked admin-only", adminOnlyJSON: `["secret"]`, secretVisibleToNonAdmin: false},
		{name: "empty marking is a no-op", adminOnlyJSON: `[]`, secretVisibleToNonAdmin: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := configureAdminOnlyGroupSurfaces(t, tc.adminOnlyJSON)
			createAdminOnlySurfaceUser(t, db, 2001, "groups-common-user", "default", common.RoleCommonUser)
			createAdminOnlySurfaceUser(t, db, 2002, "groups-admin-user", "default", common.RoleAdminUser)
			expectedGroups := map[string]struct{}{"auto": {}, "default": {}, "vip": {}, "secret": {}}
			if !tc.secretVisibleToNonAdmin {
				delete(expectedGroups, "secret")
			}

			commonCtx, commonRecorder := newAdminOnlyGroupListContext(t, 2001, "default", common.RoleCommonUser)
			GetUserGroups(commonCtx)
			commonPayload := decodeGroupListResponse(t, commonRecorder)
			assert.Equal(t, expectedGroups, groupNames(commonPayload.Data))
			assert.Equal(t, tc.secretVisibleToNonAdmin, strings.Contains(strings.ToLower(commonRecorder.Body.String()), "secret"),
				"no name, description or ratio of an admin-only group may appear")

			anonymousCtx, anonymousRecorder := newAdminOnlyGroupListContext(t, 0, "", -1)
			GetUserGroups(anonymousCtx)
			anonymousPayload := decodeGroupListResponse(t, anonymousRecorder)
			assert.Equal(t, expectedGroups, groupNames(anonymousPayload.Data))
			assert.Equal(t, tc.secretVisibleToNonAdmin, strings.Contains(strings.ToLower(anonymousRecorder.Body.String()), "secret"))

			adminCtx, adminRecorder := newAdminOnlyGroupListContext(t, 2002, "default", common.RoleAdminUser)
			GetUserGroups(adminCtx)
			adminPayload := decodeGroupListResponse(t, adminRecorder)
			assert.Contains(t, groupNames(adminPayload.Data), "secret")
			assert.Equal(t, "Secret", adminPayload.Data["secret"]["desc"])
			assert.Equal(t, float64(1), adminPayload.Data["secret"]["ratio"])

			if tc.secretVisibleToNonAdmin {
				// With nothing marked, every requester resolves the same list, which is
				// the behaviour this endpoint had before the capability existed.
				assert.Equal(t, adminPayload.Data, commonPayload.Data)
				assert.Equal(t, adminPayload.Data, anonymousPayload.Data)
			}
		})
	}
}

func groupNames(groups map[string]map[string]any) map[string]struct{} {
	names := make(map[string]struct{}, len(groups))
	for name := range groups {
		names[name] = struct{}{}
	}
	return names
}

// TestGetPricingAdminOnlySurfaces covers the model and pricing catalogue: usable
// groups, group ratios, the automatic-selection list, exclusive models, a model
// shared with a usable group, and the enable_groups names.
func TestGetPricingAdminOnlySurfaces(t *testing.T) {
	cases := []struct {
		name                    string
		adminOnlyJSON           string
		secretVisibleToNonAdmin bool
	}{
		{name: "marked admin-only", adminOnlyJSON: `["secret"]`, secretVisibleToNonAdmin: false},
		{name: "empty marking is a no-op", adminOnlyJSON: `[]`, secretVisibleToNonAdmin: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := configureAdminOnlyGroupSurfaces(t, tc.adminOnlyJSON)
			createAdminOnlySurfaceUser(t, db, 2011, "pricing-common-user", "default", common.RoleCommonUser)
			createAdminOnlySurfaceUser(t, db, 2012, "pricing-admin-user", "default", common.RoleAdminUser)

			commonCtx, commonRecorder := newAdminOnlyPricingContext(t, 2011, common.RoleCommonUser)
			GetPricing(commonCtx)
			commonPayload := decodePricingCatalogue(t, commonRecorder)
			if tc.secretVisibleToNonAdmin {
				assert.Contains(t, commonPayload.UsableGroup, "secret")
				assert.Contains(t, commonPayload.GroupRatio, "secret")
				assert.Contains(t, commonPayload.AutoGroups, "secret")
			} else {
				assert.NotContains(t, commonPayload.UsableGroup, "secret")
				assert.NotContains(t, commonPayload.GroupRatio, "secret")
				assert.NotContains(t, commonPayload.AutoGroups, "secret")
			}

			commonModels := pricingByModelName(commonPayload.Data)
			if tc.secretVisibleToNonAdmin {
				assert.Contains(t, commonModels, "zz-secret-only-model")
			} else {
				assert.NotContains(t, commonModels, "zz-secret-only-model")
			}
			// A model shared with a usable group stays visible for the requester.
			shared, ok := commonModels["zz-shared-model"]
			require.True(t, ok)
			assert.Contains(t, shared.EnableGroup, "default")
			if tc.secretVisibleToNonAdmin {
				assert.Contains(t, shared.EnableGroup, "secret")
			} else {
				assert.NotContains(t, shared.EnableGroup, "secret")
			}

			anonymousCtx, anonymousRecorder := newAdminOnlyPricingContext(t, 0, -1)
			GetPricing(anonymousCtx)
			anonymousBody := strings.ToLower(anonymousRecorder.Body.String())
			anonymousPayload := decodePricingCatalogue(t, anonymousRecorder)
			assert.Equal(t, tc.secretVisibleToNonAdmin, strings.Contains(anonymousBody, "secret"))
			// The shared model must still be served, so the absence of the name above
			// cannot be explained by the row having been dropped.
			assert.Contains(t, pricingByModelName(anonymousPayload.Data), "zz-shared-model")

			adminCtx, adminRecorder := newAdminOnlyPricingContext(t, 2012, common.RoleAdminUser)
			GetPricing(adminCtx)
			adminPayload := decodePricingCatalogue(t, adminRecorder)
			assert.Contains(t, adminPayload.UsableGroup, "secret")
			assert.Contains(t, adminPayload.GroupRatio, "secret")
			assert.Contains(t, adminPayload.AutoGroups, "secret")
			adminModels := pricingByModelName(adminPayload.Data)
			assert.Contains(t, adminModels, "zz-secret-only-model")
			assert.Contains(t, adminModels["zz-shared-model"].EnableGroup, "secret")

			if tc.secretVisibleToNonAdmin {
				assert.Equal(t, adminPayload.UsableGroup, commonPayload.UsableGroup)
				assert.Equal(t, adminPayload.GroupRatio, commonPayload.GroupRatio)
				assert.Equal(t, adminPayload.AutoGroups, commonPayload.AutoGroups)
				assert.Equal(t, adminPayload.UsableGroup, anonymousPayload.UsableGroup)
				assert.ElementsMatch(t, pricingModelNames(adminPayload.Data), pricingModelNames(commonPayload.Data))
			}
		})
	}
}

func pricingModelNames(pricings []model.Pricing) []string {
	names := make([]string, 0, len(pricings))
	for _, pricing := range pricings {
		names = append(names, pricing.ModelName)
	}
	return names
}

// TestGetUserModelsAdminOnlySurface covers the per-user model list, including the
// explicit ?group=<admin-only> request that must yield no models.
func TestGetUserModelsAdminOnlySurface(t *testing.T) {
	cases := []struct {
		name                    string
		adminOnlyJSON           string
		secretVisibleToNonAdmin bool
	}{
		{name: "marked admin-only", adminOnlyJSON: `["secret"]`, secretVisibleToNonAdmin: false},
		{name: "empty marking is a no-op", adminOnlyJSON: `[]`, secretVisibleToNonAdmin: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := configureAdminOnlyGroupSurfaces(t, tc.adminOnlyJSON)
			createAdminOnlySurfaceUser(t, db, 2021, "models-common-user", "default", common.RoleCommonUser)
			createAdminOnlySurfaceUser(t, db, 2022, "models-admin-user", "default", common.RoleAdminUser)

			commonCtx, commonRecorder := newAuthenticatedContext(t, http.MethodGet, "/api/user/models?group=secret", nil, 2021)
			GetUserModels(commonCtx)
			commonModels := decodeUserModelsResponse(t, commonRecorder)
			if tc.secretVisibleToNonAdmin {
				assert.ElementsMatch(t, []string{"zz-secret-only-model", "zz-shared-model"}, commonModels)
			} else {
				assert.Empty(t, commonModels)
			}

			adminCtx, adminRecorder := newAuthenticatedContext(t, http.MethodGet, "/api/user/models?group=secret", nil, 2022)
			GetUserModels(adminCtx)
			assert.ElementsMatch(t, []string{"zz-secret-only-model", "zz-shared-model"}, decodeUserModelsResponse(t, adminRecorder))

			// Unfiltered list: a model reachable only through the admin-only group is
			// absent for a non-administrator and priced models they may use stay.
			allCtx, allRecorder := newAuthenticatedContext(t, http.MethodGet, "/api/user/models", nil, 2021)
			GetUserModels(allCtx)
			allModels := decodeUserModelsResponse(t, allRecorder)
			assert.Contains(t, allModels, "zz-default-model")
			assert.Contains(t, allModels, "zz-shared-model")
			if tc.secretVisibleToNonAdmin {
				assert.Contains(t, allModels, "zz-secret-only-model")
			} else {
				assert.NotContains(t, allModels, "zz-secret-only-model")
			}
		})
	}
}

// TestListModelsAdminOnlyOwnGroup covers /v1/models, which resolves the group from
// the token or the requester's own user group and never reads the effective group
// set by token authentication.
func TestListModelsAdminOnlyOwnGroup(t *testing.T) {
	cases := []struct {
		name                    string
		adminOnlyJSON           string
		secretVisibleToNonAdmin bool
	}{
		{name: "marked admin-only", adminOnlyJSON: `["secret"]`, secretVisibleToNonAdmin: false},
		{name: "empty marking is a no-op", adminOnlyJSON: `[]`, secretVisibleToNonAdmin: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withSelfUseModeEnabled(t)
			configureAdminOnlyGroupSurfaces(t, tc.adminOnlyJSON)

			// Ungrouped token: ContextKeyTokenGroup is absent, so the requester's own
			// group becomes the effective group for the model list.
			commonCtx, commonRecorder := newModelListContext(t, "secret", common.RoleCommonUser)
			ListModels(commonCtx, constant.ChannelTypeOpenAI)
			commonIds := decodeListModelsResponse(t, commonRecorder)
			if tc.secretVisibleToNonAdmin {
				assert.Contains(t, commonIds, "zz-secret-only-model")
			} else {
				assert.Empty(t, commonIds)
			}

			adminCtx, adminRecorder := newModelListContext(t, "secret", common.RoleAdminUser)
			ListModels(adminCtx, constant.ChannelTypeOpenAI)
			adminIds := decodeListModelsResponse(t, adminRecorder)
			assert.Contains(t, adminIds, "zz-secret-only-model")
		})
	}
}

func newModelListContext(t *testing.T, userGroup string, role int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, userGroup)
	common.SetContextKey(ctx, constant.ContextKeyUserRole, role)
	return ctx, recorder
}

// TestTokenAutoGroupsAdminOnlySurface covers the automatic-selection order editor
// and saving an admin-only group into a key's order.
func TestTokenAutoGroupsAdminOnlySurface(t *testing.T) {
	cases := []struct {
		name                       string
		adminOnlyJSON              string
		secretSelectableByNonAdmin bool
	}{
		{name: "marked admin-only", adminOnlyJSON: `["secret"]`, secretSelectableByNonAdmin: false},
		{name: "empty marking is a no-op", adminOnlyJSON: `[]`, secretSelectableByNonAdmin: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := configureAdminOnlyGroupSurfaces(t, tc.adminOnlyJSON)
			createAdminOnlySurfaceUser(t, db, 2031, "auto-common-user", "default", common.RoleCommonUser)
			createAdminOnlySurfaceUser(t, db, 2032, "auto-admin-user", "default", common.RoleAdminUser)

			// The order editor offers only groups the requester may use.
			commonCtx, commonRecorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodGet, "/api/token/auto-groups", nil, 2031)
			common.SetContextKey(commonCtx, constant.ContextKeyUserRole, common.RoleCommonUser)
			GetTokenAutoGroups(commonCtx)
			var commonAutoGroups struct {
				Groups   []string `json:"groups"`
				MaxCount int      `json:"max_count"`
			}
			commonResponse := decodeAPIResponse(t, commonRecorder)
			require.True(t, commonResponse.Success, commonResponse.Message)
			require.NoError(t, common.Unmarshal(commonResponse.Data, &commonAutoGroups))
			if tc.secretSelectableByNonAdmin {
				assert.Equal(t, []string{"secret", "vip", "default"}, commonAutoGroups.Groups)
			} else {
				assert.Equal(t, []string{"vip", "default"}, commonAutoGroups.Groups)
			}

			adminCtx, adminRecorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodGet, "/api/token/auto-groups", nil, 2032)
			common.SetContextKey(adminCtx, constant.ContextKeyUserRole, common.RoleAdminUser)
			GetTokenAutoGroups(adminCtx)
			adminResponse := decodeAPIResponse(t, adminRecorder)
			require.True(t, adminResponse.Success, adminResponse.Message)
			var adminAutoGroups struct {
				Groups []string `json:"groups"`
			}
			require.NoError(t, common.Unmarshal(adminResponse.Data, &adminAutoGroups))
			assert.Equal(t, []string{"secret", "vip", "default"}, adminAutoGroups.Groups)

			// Saving an admin-only group into the order must be refused for an
			// ordinary user and must store nothing.
			addRequest := baseAutoTokenRequest("auto-group-admin-only")
			addRequest["auto_groups"] = []string{"secret"}
			addCtx, addRecorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPost, "/api/token/", addRequest, 2031)
			common.SetContextKey(addCtx, constant.ContextKeyUserRole, common.RoleCommonUser)
			AddToken(addCtx)
			added := decodeAPIResponse(t, addRecorder)

			var tokenCount int64
			require.NoError(t, model.DB.Model(&model.Token{}).Count(&tokenCount).Error)

			if tc.secretSelectableByNonAdmin {
				require.True(t, added.Success, added.Message)
				require.Equal(t, int64(1), tokenCount)
				var stored model.Token
				require.NoError(t, model.DB.Where("name = ?", "auto-group-admin-only").First(&stored).Error)
				assert.JSONEq(t, `["secret"]`, stored.AutoGroups)
				return
			}

			require.False(t, added.Success)
			assert.Zero(t, tokenCount)

			// The refusal is the same invalid-group error any other unusable group
			// produces: with the marking removed and the group absent from the
			// registry, the identical request yields the identical message.
			require.NoError(t, setting.UpdateAdminOnlyGroupsByJsonString(`[]`))
			require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"auto":"Auto","default":"Default","vip":"VIP"}`))
			controlRequest := baseAutoTokenRequest("auto-group-control")
			controlRequest["auto_groups"] = []string{"secret"}
			controlCtx, controlRecorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPost, "/api/token/", controlRequest, 2031)
			common.SetContextKey(controlCtx, constant.ContextKeyUserRole, common.RoleCommonUser)
			AddToken(controlCtx)
			control := decodeAPIResponse(t, controlRecorder)
			require.False(t, control.Success)
			assert.Equal(t, control.Message, added.Message)
		})
	}
}
