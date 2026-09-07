package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueTelegramLinkCodeForUnlinkedIdentifier(t *testing.T) {
	truncateTables(t)

	before := time.Now()
	code, expiresAt, err := IssueTelegramLinkCode("telegram-issue-unlinked", "  @JohnDoe  ")
	require.NoError(t, err)
	assert.Len(t, code, TelegramLinkCodeLength)
	assert.WithinDuration(t, before.Add(TelegramLinkCodeTTL), expiresAt, time.Second)

	// Issuing a code must not, by itself, create any binding.
	var count int64
	require.NoError(t, DB.Model(&User{}).Where("telegram_id = ?", "telegram-issue-unlinked").Count(&count).Error)
	assert.Zero(t, count)

	flow, err := GetAuthFlow(code, AuthFlowMatch{Purpose: AuthFlowPurposeTelegramLink})
	require.NoError(t, err)
	var payload TelegramLinkPayload
	require.NoError(t, common.UnmarshalJsonStr(flow.Payload, &payload))
	assert.Equal(t, "telegram-issue-unlinked", payload.TelegramUserID)
	assert.Equal(t, "johndoe", payload.TelegramUsername)
}

func TestIssueTelegramLinkCodeRefusesAlreadyBoundIdentifier(t *testing.T) {
	truncateTables(t)

	user := User{Username: "already-bound-user", Password: "password", TelegramId: "telegram-already-bound", AffCode: "already-bound-user"}
	require.NoError(t, DB.Create(&user).Error)

	_, _, err := IssueTelegramLinkCode("telegram-already-bound", "")
	assert.ErrorIs(t, err, ErrTelegramLinkAlreadyBound)
}
