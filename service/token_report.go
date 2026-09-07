package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// TokenReportBase holds the quota/config fields shared between the
// authenticated GET /api/usage/token report and the public POST
// /api/token/check report. This is the single producer for these fields:
// every surface that reports on a key — including the Telegram bot
// integration surface — derives them from here so they can never drift
// apart (see specs/public-key-check/spec.md, "Key check reports the key's
// full usage and configuration").
type TokenReportBase struct {
	Name               string
	TotalGranted       int
	TotalUsed          int
	TotalAvailable     int
	UnlimitedQuota     bool
	ModelLimitsEnabled bool
	ModelLimits        map[string]bool
}

func BuildTokenReportBase(token *model.Token) TokenReportBase {
	return TokenReportBase{
		Name:               token.Name,
		TotalGranted:       token.RemainQuota + token.UsedQuota,
		TotalUsed:          token.UsedQuota,
		TotalAvailable:     token.RemainQuota,
		UnlimitedQuota:     token.UnlimitedQuota,
		ModelLimitsEnabled: token.ModelLimitsEnabled,
		ModelLimits:        token.GetModelLimitsMap(),
	}
}

// EffectiveTokenStatus derives a token's status as of now without persisting
// anything, applying the fixed precedence disabled -> expired -> exhausted ->
// enabled. It matches the checks model.ValidateUserToken applies on the relay
// path, so every report surface agrees with the console.
func EffectiveTokenStatus(token *model.Token, now int64) int {
	if token.Status == common.TokenStatusDisabled {
		return common.TokenStatusDisabled
	}
	if token.ExpiredTime != -1 && token.ExpiredTime < now {
		return common.TokenStatusExpired
	}
	if !token.UnlimitedQuota && token.RemainQuota <= 0 {
		return common.TokenStatusExhausted
	}
	return common.TokenStatusEnabled
}

// NormalizeTokenKey applies the same normalization as relay and read-only
// token authentication (middleware.TokenAuthReadOnly): trim whitespace, strip
// a leading Bearer/bearer prefix, strip a leading sk- prefix, and keep only
// the segment before the first remaining "-". This lets a key pasted in any
// of those forms resolve to the same token, on every surface that accepts a
// key for inspection.
func NormalizeTokenKey(key string) string {
	key = strings.TrimSpace(key)
	if strings.HasPrefix(key, "Bearer ") || strings.HasPrefix(key, "bearer ") {
		key = strings.TrimSpace(key[len("Bearer "):])
	}
	key = strings.TrimPrefix(key, "sk-")
	parts := strings.Split(key, "-")
	return parts[0]
}

// ResolveEffectiveTokenGroup reports the group that actually applies to a
// token: the token's own group when set, otherwise the owning user's group.
func ResolveEffectiveTokenGroup(token *model.Token) (string, error) {
	if token.Group != "" {
		return token.Group, nil
	}
	return model.GetUserGroup(token.UserId, false)
}

// TokenCheckReport is the report returned by every surface that inspects a
// key: the public key-check endpoint, the setup script's key lookup, and the
// Telegram bot integration surface. It deliberately omits every account
// identity field (user id, username, email): the person holding a key is not
// necessarily the account owner.
type TokenCheckReport struct {
	Name               string          `json:"name"`
	Group              string          `json:"group"`
	Status             int             `json:"status"`
	UnlimitedQuota     bool            `json:"unlimited_quota"`
	TotalGranted       int             `json:"total_granted"`
	TotalUsed          int             `json:"total_used"`
	TotalAvailable     int             `json:"total_available"`
	ExpiresAt          int64           `json:"expires_at"`
	CreatedTime        int64           `json:"created_time"`
	AccessedTime       int64           `json:"accessed_time"`
	ModelLimitsEnabled bool            `json:"model_limits_enabled"`
	ModelLimits        map[string]bool `json:"model_limits"`
	AvailableModels    []string        `json:"available_models"`
}

// BuildTokenCheckReport resolves the token's effective group and status and
// assembles the report documented in specs/public-key-check/spec.md. It
// performs no database write. This is the shared producer: every surface
// that reports on a key calls this function so the field set and its
// semantics cannot drift between surfaces.
func BuildTokenCheckReport(token *model.Token) (*TokenCheckReport, error) {
	group, err := ResolveEffectiveTokenGroup(token)
	if err != nil {
		return nil, err
	}
	base := BuildTokenReportBase(token)
	availableModels := model.GetGroupEnabledModels(group)
	if availableModels == nil {
		availableModels = []string{}
	}
	return &TokenCheckReport{
		Name:               base.Name,
		Group:              group,
		Status:             EffectiveTokenStatus(token, common.GetTimestamp()),
		UnlimitedQuota:     base.UnlimitedQuota,
		TotalGranted:       base.TotalGranted,
		TotalUsed:          base.TotalUsed,
		TotalAvailable:     base.TotalAvailable,
		ExpiresAt:          token.ExpiredTime,
		CreatedTime:        token.CreatedTime,
		AccessedTime:       token.AccessedTime,
		ModelLimitsEnabled: base.ModelLimitsEnabled,
		ModelLimits:        base.ModelLimits,
		AvailableModels:    availableModels,
	}, nil
}
