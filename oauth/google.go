package oauth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

func init() {
	Register("google", &GoogleProvider{})
}

// googleTokenEndpoint and googleUserInfoEndpoint are package-level vars (not
// consts) so tests can point them at an httptest.Server instead of the real
// Google endpoints.
var (
	googleTokenEndpoint    = "https://oauth2.googleapis.com/token"
	googleUserInfoEndpoint = "https://openidconnect.googleapis.com/v1/userinfo"
)

// GoogleProvider implements OAuth for Google
type GoogleProvider struct{}

type googleOAuthResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

// googleUser mirrors the fields returned by Google's userinfo endpoint.
// Sub is the stable, unique subject identifier and is the only field used
// to identify the account; email and name are display data only.
type googleUser struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (p *GoogleProvider) GetName() string {
	return "Google"
}

func (p *GoogleProvider) IsEnabled() bool {
	return common.GoogleOAuthEnabled
}

func (p *GoogleProvider) ExchangeToken(ctx context.Context, code string, c *gin.Context) (*OAuthToken, error) {
	if code == "" {
		return nil, NewOAuthError(i18n.MsgOAuthInvalidCode, nil)
	}

	logger.LogDebug(ctx, "[OAuth-Google] ExchangeToken: code=%s...", code[:min(len(code), 10)])

	redirectUri := fmt.Sprintf("%s/oauth/google", system_setting.ServerAddress)
	values := url.Values{}
	values.Set("client_id", common.GoogleClientId)
	values.Set("client_secret", common.GoogleClientSecret)
	values.Set("code", code)
	values.Set("grant_type", "authorization_code")
	values.Set("redirect_uri", redirectUri)

	logger.LogDebug(ctx, "[OAuth-Google] ExchangeToken: redirect_uri=%s", redirectUri)

	req, err := http.NewRequestWithContext(ctx, "POST", googleTokenEndpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := http.Client{
		Timeout: 20 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-Google] ExchangeToken error: %s", err.Error()))
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthConnectFailed, map[string]any{"Provider": "Google"}, err.Error())
	}
	defer res.Body.Close()

	logger.LogDebug(ctx, "[OAuth-Google] ExchangeToken response status: %d", res.StatusCode)

	var oAuthResponse googleOAuthResponse
	err = common.DecodeJson(res.Body, &oAuthResponse)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-Google] ExchangeToken decode error: %s", err.Error()))
		return nil, err
	}

	if oAuthResponse.AccessToken == "" {
		logger.LogError(ctx, "[OAuth-Google] ExchangeToken failed: empty access token")
		return nil, NewOAuthError(i18n.MsgOAuthTokenFailed, map[string]any{"Provider": "Google"})
	}

	logger.LogDebug(ctx, "[OAuth-Google] ExchangeToken success: scope=%s", oAuthResponse.Scope)

	return &OAuthToken{
		AccessToken:  oAuthResponse.AccessToken,
		TokenType:    oAuthResponse.TokenType,
		RefreshToken: oAuthResponse.RefreshToken,
		ExpiresIn:    oAuthResponse.ExpiresIn,
		Scope:        oAuthResponse.Scope,
		IDToken:      oAuthResponse.IDToken,
	}, nil
}

// GetUserInfo fetches the profile from Google's userinfo endpoint rather than
// decoding the ID token locally, since verifying an ID token correctly would
// require fetching and caching Google's JWKS and checking signature, issuer,
// audience and expiry — infrastructure this codebase does not have.
func (p *GoogleProvider) GetUserInfo(ctx context.Context, token *OAuthToken) (*OAuthUser, error) {
	logger.LogDebug(ctx, "[OAuth-Google] GetUserInfo: fetching user info")

	req, err := http.NewRequestWithContext(ctx, "GET", googleUserInfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := http.Client{
		Timeout: 20 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-Google] GetUserInfo error: %s", err.Error()))
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthConnectFailed, map[string]any{"Provider": "Google"}, err.Error())
	}
	defer res.Body.Close()

	logger.LogDebug(ctx, "[OAuth-Google] GetUserInfo response status: %d", res.StatusCode)

	if res.StatusCode != http.StatusOK {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-Google] GetUserInfo failed: status=%d", res.StatusCode))
		return nil, NewOAuthError(i18n.MsgOAuthGetUserErr, map[string]any{"Provider": "Google"})
	}

	var googleUser googleUser
	err = common.DecodeJson(res.Body, &googleUser)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-Google] GetUserInfo decode error: %s", err.Error()))
		return nil, err
	}

	if googleUser.Sub == "" {
		logger.LogError(ctx, "[OAuth-Google] GetUserInfo failed: empty sub field")
		return nil, NewOAuthError(i18n.MsgOAuthUserInfoEmpty, map[string]any{"Provider": "Google"})
	}

	logger.LogDebug(ctx, "[OAuth-Google] GetUserInfo success: sub=%s, name=%s, email=%s",
		googleUser.Sub, googleUser.Name, googleUser.Email)

	return &OAuthUser{
		ProviderUserID: googleUser.Sub,
		DisplayName:    googleUser.Name,
		Email:          googleUser.Email,
	}, nil
}

func (p *GoogleProvider) IsUserIDTaken(providerUserID string) bool {
	return model.IsGoogleIdAlreadyTaken(providerUserID)
}

func (p *GoogleProvider) FillUserByProviderID(user *model.User, providerUserID string) error {
	user.GoogleId = providerUserID
	return user.FillUserByGoogleId()
}

func (p *GoogleProvider) SetProviderUserID(user *model.User, providerUserID string) {
	user.GoogleId = providerUserID
}

func (p *GoogleProvider) GetProviderPrefix() string {
	return "google_"
}

// ProviderUserIDColumn returns the users-table column storing this provider's user ID.
func (p *GoogleProvider) ProviderUserIDColumn() string {
	return "google_id"
}
