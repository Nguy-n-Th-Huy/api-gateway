package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupOAuthLegacyIdentityTest points model.DB/model.LOG_DB at a fresh
// in-memory SQLite database for the duration of the test and restores the
// previous globals afterwards. It uses the real "github" provider registered
// by oauth/github.go's init(), since findOrCreateOAuthUser is written against
// the generic oauth.Provider interface and never touches network calls.
func setupOAuthLegacyIdentityTest(t *testing.T) *gin.Context {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	previousRedisEnabled := common.RedisEnabled

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}))

	model.DB, model.LOG_DB = db, db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false

	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		common.RedisEnabled = previousRedisEnabled
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	return c
}

func insertLegacyGitHubUser(t *testing.T, username, legacyLogin, email string) *model.User {
	t.Helper()
	user := &model.User{
		Username: username,
		Password: "irrelevant-hash",
		Email:    email,
		GitHubId: legacyLogin,
	}
	require.NoError(t, user.Insert(0))
	return user
}

func TestFindOrCreateOAuthUser_LegacyMatch_EmailEqual_MigratesAndSignsIn(t *testing.T) {
	c := setupOAuthLegacyIdentityTest(t)
	existing := insertLegacyGitHubUser(t, "legacy-match-user", "alice", "user@example.com")

	provider := oauth.GetProvider("github")
	require.NotNil(t, provider)

	oauthUser := &oauth.OAuthUser{
		ProviderUserID: "1000123",
		Username:       "alice",
		Email:          "User@Example.com", // different case: must still match after normalization
		Extra:          map[string]any{"legacy_id": "alice"},
	}

	user, err := findOrCreateOAuthUser(c, provider, oauthUser, "")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, existing.Id, user.Id)

	reloaded := &model.User{Id: existing.Id}
	require.NoError(t, reloaded.FillUserById())
	assert.Equal(t, "1000123", reloaded.GitHubId, "the account must be migrated to the stable identifier")
}

// TestFindOrCreateOAuthUser_LegacyMatch_EmailMismatch_Refused is the takeover
// regression test: a stranger holding a released GitHub login must not be
// able to sign in as the original account holder.
func TestFindOrCreateOAuthUser_LegacyMatch_EmailMismatch_Refused(t *testing.T) {
	c := setupOAuthLegacyIdentityTest(t)
	existing := insertLegacyGitHubUser(t, "legacy-mismatch-user", "alice", "owner@example.com")

	var countBefore int64
	require.NoError(t, model.DB.Model(&model.User{}).Count(&countBefore).Error)

	provider := oauth.GetProvider("github")
	require.NotNil(t, provider)

	oauthUser := &oauth.OAuthUser{
		ProviderUserID: "999888777",
		Username:       "alice",
		Email:          "attacker@example.com",
		Extra:          map[string]any{"legacy_id": "alice"},
	}

	user, err := findOrCreateOAuthUser(c, provider, oauthUser, "")
	require.Error(t, err)
	assert.Nil(t, user)
	var mismatchErr *OAuthLegacyIdentityMismatchError
	assert.ErrorAs(t, err, &mismatchErr)

	var countAfter int64
	require.NoError(t, model.DB.Model(&model.User{}).Count(&countAfter).Error)
	assert.Equal(t, countBefore, countAfter, "no new account must be created for the refused visitor")

	reloaded := &model.User{Id: existing.Id}
	require.NoError(t, reloaded.FillUserById())
	assert.Equal(t, "alice", reloaded.GitHubId, "the matched account's identifier must be unchanged")
	assert.Equal(t, "owner@example.com", reloaded.Email, "the matched account's email must be unchanged")
}

func TestFindOrCreateOAuthUser_LegacyMatch_MissingEmail_Refused(t *testing.T) {
	tests := []struct {
		name          string
		storedEmail   string
		providerEmail string
	}{
		{name: "no stored email on the matched account", storedEmail: "", providerEmail: "someone@example.com"},
		{name: "no email reported by the provider", storedEmail: "owner@example.com", providerEmail: ""},
		{name: "neither side has an email", storedEmail: "", providerEmail: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := setupOAuthLegacyIdentityTest(t)
			existing := insertLegacyGitHubUser(t, "legacy-missing-email-user", "bob", tc.storedEmail)

			provider := oauth.GetProvider("github")
			require.NotNil(t, provider)

			oauthUser := &oauth.OAuthUser{
				ProviderUserID: "555444333",
				Username:       "bob",
				Email:          tc.providerEmail,
				Extra:          map[string]any{"legacy_id": "bob"},
			}

			user, err := findOrCreateOAuthUser(c, provider, oauthUser, "")
			require.Error(t, err)
			assert.Nil(t, user)
			var unverifiableErr *OAuthLegacyIdentityUnverifiableError
			assert.ErrorAs(t, err, &unverifiableErr)

			reloaded := &model.User{Id: existing.Id}
			require.NoError(t, reloaded.FillUserById())
			assert.Equal(t, "bob", reloaded.GitHubId, "the matched account's identifier must be unchanged")
		})
	}
}

func TestFindOrCreateOAuthUser_StableIdentifier_NeverReachesLegacyBranch(t *testing.T) {
	c := setupOAuthLegacyIdentityTest(t)
	// Already migrated: the account is stored under the stable numeric ID, and
	// its email deliberately differs from what the provider now reports, so a
	// pass through the legacy branch would incorrectly refuse this sign-in.
	existing := &model.User{
		Username: "already-migrated-user",
		Password: "irrelevant-hash",
		Email:    "old-address@example.com",
		GitHubId: "1000123",
	}
	require.NoError(t, existing.Insert(0))

	provider := oauth.GetProvider("github")
	require.NotNil(t, provider)

	oauthUser := &oauth.OAuthUser{
		ProviderUserID: "1000123",
		Username:       "alice",
		Email:          "new-address@example.com",
		Extra:          map[string]any{"legacy_id": "alice"},
	}

	user, err := findOrCreateOAuthUser(c, provider, oauthUser, "")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, existing.Id, user.Id)
}
