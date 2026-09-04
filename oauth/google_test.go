package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withGoogleEndpoints(t *testing.T, tokenURL, userInfoURL string) {
	t.Helper()
	originalToken := googleTokenEndpoint
	originalUserInfo := googleUserInfoEndpoint
	if tokenURL != "" {
		googleTokenEndpoint = tokenURL
	}
	if userInfoURL != "" {
		googleUserInfoEndpoint = userInfoURL
	}
	t.Cleanup(func() {
		googleTokenEndpoint = originalToken
		googleUserInfoEndpoint = originalUserInfo
	})
}

func TestGoogleProvider_GetUserInfo_NoEmailStillValid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":"1234567890","name":"Jane Doe"}`))
	}))
	defer server.Close()
	withGoogleEndpoints(t, "", server.URL)

	p := &GoogleProvider{}
	user, err := p.GetUserInfo(context.Background(), &OAuthToken{AccessToken: "test-token"})
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "1234567890", user.ProviderUserID)
	assert.Equal(t, "Jane Doe", user.DisplayName)
	assert.Empty(t, user.Email)
}

func TestGoogleProvider_GetUserInfo_EmptySubRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":"","name":"Jane Doe","email":"jane@example.com"}`))
	}))
	defer server.Close()
	withGoogleEndpoints(t, "", server.URL)

	p := &GoogleProvider{}
	user, err := p.GetUserInfo(context.Background(), &OAuthToken{AccessToken: "test-token"})
	require.Error(t, err)
	assert.Nil(t, user)

	oauthErr, ok := err.(*OAuthError)
	require.True(t, ok, "expected *OAuthError, got %T", err)
	assert.Equal(t, "oauth.user_info_empty", oauthErr.MsgKey)
	assert.Equal(t, "Google", oauthErr.Params["Provider"])
}

func TestGoogleProvider_GetUserInfo_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	withGoogleEndpoints(t, "", server.URL)

	p := &GoogleProvider{}
	user, err := p.GetUserInfo(context.Background(), &OAuthToken{AccessToken: "test-token"})
	require.Error(t, err)
	assert.Nil(t, user)
}

func TestGoogleProvider_ExchangeToken_EmptyCode(t *testing.T) {
	p := &GoogleProvider{}
	token, err := p.ExchangeToken(context.Background(), "", nil)
	require.Error(t, err)
	assert.Nil(t, token)

	oauthErr, ok := err.(*OAuthError)
	require.True(t, ok)
	assert.Equal(t, "oauth.invalid_code", oauthErr.MsgKey)
}

func TestGoogleProvider_ExchangeToken_ConnectFailureLeaksNoSecret(t *testing.T) {
	originalSecret := common.GoogleClientSecret
	common.GoogleClientSecret = "super-secret-value"
	defer func() { common.GoogleClientSecret = originalSecret }()

	// Point at a server that is closed before use, so the connection is
	// refused deterministically without depending on real network access.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()
	withGoogleEndpoints(t, server.URL, "")

	p := &GoogleProvider{}
	token, err := p.ExchangeToken(context.Background(), "some-code", nil)
	require.Error(t, err)
	assert.Nil(t, token)

	oauthErr, ok := err.(*OAuthError)
	require.True(t, ok, "expected *OAuthError, got %T", err)
	assert.Equal(t, "oauth.connect_failed", oauthErr.MsgKey)
	assert.Equal(t, "Google", oauthErr.Params["Provider"])
	assert.NotContains(t, oauthErr.Error(), "super-secret-value")
	assert.False(t, strings.Contains(oauthErr.RawError, "super-secret-value"))
}

func TestGoogleProvider_ProviderIdentity(t *testing.T) {
	p := &GoogleProvider{}
	assert.Equal(t, "google_", p.GetProviderPrefix())
	assert.Equal(t, "google_id", p.ProviderUserIDColumn())
	assert.Equal(t, "Google", p.GetName())
}
