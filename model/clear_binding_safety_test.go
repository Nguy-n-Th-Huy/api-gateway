package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClearBindingGuardsAgainstStrandingAccount protects the last-sign-in-method
// invariant added to ClearBinding: an administrator can never blank the account's
// only remaining way to sign in, including the email column.
func TestClearBindingGuardsAgainstStrandingAccount(t *testing.T) {
	truncateTables(t)

	t.Run("succeeds while a password remains", func(t *testing.T) {
		user := &User{Username: "guard-password-remains", Password: "irrelevant-hash", GitHubId: "github-guard-1"}
		require.NoError(t, user.Insert(0))

		require.NoError(t, user.ClearBinding("github"))
		assert.Empty(t, user.GitHubId)

		reloaded := &User{Id: user.Id}
		require.NoError(t, reloaded.FillUserById())
		assert.Empty(t, reloaded.GitHubId)
	})

	t.Run("succeeds while another binding remains", func(t *testing.T) {
		user := &User{Username: "guard-other-binding-remains", Password: "", GitHubId: "github-guard-2", GoogleId: "google-guard-2"}
		require.NoError(t, user.Insert(0))

		require.NoError(t, user.ClearBinding("github"))
		assert.Empty(t, user.GitHubId)
		assert.Equal(t, "google-guard-2", user.GoogleId)

		reloaded := &User{Id: user.Id}
		require.NoError(t, reloaded.FillUserById())
		assert.Empty(t, reloaded.GitHubId)
		assert.Equal(t, "google-guard-2", reloaded.GoogleId)
	})

	t.Run("refused when neither a password nor another binding remains", func(t *testing.T) {
		user := &User{Username: "guard-last-method", Password: "", GitHubId: "github-guard-3"}
		require.NoError(t, user.Insert(0))

		err := user.ClearBinding("github")
		require.ErrorIs(t, err, ErrNoRemainingSignInMethod)

		reloaded := &User{Id: user.Id}
		require.NoError(t, reloaded.FillUserById())
		assert.Equal(t, "github-guard-3", reloaded.GitHubId, "the column must be left untouched on refusal")
	})

	t.Run("clearing email is refused when it is the only remaining method", func(t *testing.T) {
		user := &User{Username: "guard-email-only", Password: "", Email: "only-method@example.com"}
		require.NoError(t, user.Insert(0))

		err := user.ClearBinding("email")
		require.ErrorIs(t, err, ErrNoRemainingSignInMethod)

		reloaded := &User{Id: user.Id}
		require.NoError(t, reloaded.FillUserById())
		assert.Equal(t, "only-method@example.com", reloaded.Email, "email must be left untouched on refusal")
	})

	t.Run("clearing email succeeds while a github binding remains", func(t *testing.T) {
		user := &User{Username: "guard-email-with-github", Password: "", Email: "clearable@example.com", GitHubId: "github-guard-4"}
		require.NoError(t, user.Insert(0))

		require.NoError(t, user.ClearBinding("email"))
		assert.Empty(t, user.Email)

		reloaded := &User{Id: user.Id}
		require.NoError(t, reloaded.FillUserById())
		assert.Empty(t, reloaded.Email)
		assert.Equal(t, "github-guard-4", reloaded.GitHubId)
	})
}
