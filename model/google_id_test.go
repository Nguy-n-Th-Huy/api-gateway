package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoogleIdColumn_AgreesAcrossHelpers proves that the google_id column
// used by GORM's schema (what oauth.GoogleProvider.ProviderUserIDColumn
// reports), the taken-check helper, and the fill helper all operate on the
// same column and agree on the same data.
func TestGoogleIdColumn_AgreesAcrossHelpers(t *testing.T) {
	truncateTables(t)

	user := &User{
		Username: "google_user_1",
		Password: "irrelevant-hash",
		GoogleId: "google-sub-123456",
	}
	require.NoError(t, user.Insert(0))

	t.Run("taken check reports true for a bound id", func(t *testing.T) {
		assert.True(t, IsGoogleIdAlreadyTaken("google-sub-123456"))
	})

	t.Run("taken check reports false for an unbound id", func(t *testing.T) {
		assert.False(t, IsGoogleIdAlreadyTaken("google-sub-does-not-exist"))
	})

	t.Run("fill helper finds the same user by google_id", func(t *testing.T) {
		lookup := &User{GoogleId: "google-sub-123456"}
		require.NoError(t, lookup.FillUserByGoogleId())
		assert.Equal(t, user.Id, lookup.Id)
		assert.Equal(t, "google_user_1", lookup.Username)
	})

	t.Run("fill helper rejects an empty google_id", func(t *testing.T) {
		lookup := &User{}
		err := lookup.FillUserByGoogleId()
		require.Error(t, err)
	})

	t.Run("bind column whitelist accepts google_id", func(t *testing.T) {
		other := &User{Username: "google_user_2", Password: "irrelevant-hash"}
		require.NoError(t, other.Insert(0))
		require.NoError(t, UpdateUserBindColumn(other.Id, "google_id", "google-sub-999"))

		reloaded := &User{Id: other.Id}
		require.NoError(t, reloaded.FillUserById())
		assert.Equal(t, "google-sub-999", reloaded.GoogleId)
	})

	t.Run("ClearBinding clears the google_id column", func(t *testing.T) {
		bound := &User{Username: "google_user_3", Password: "irrelevant-hash", GoogleId: "google-sub-clear-me"}
		require.NoError(t, bound.Insert(0))
		require.NoError(t, bound.ClearBinding("google"))
		assert.Empty(t, bound.GoogleId)

		reloaded := &User{Id: bound.Id}
		require.NoError(t, reloaded.FillUserById())
		assert.Empty(t, reloaded.GoogleId)
	})
}
