package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupTelegramLinkTestDB gives TelegramLinkRedeem a fresh sqlite fixture and
// the settings state it depends on, independent of any other test in this
// package.
func setupTelegramLinkTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)
	require.NoError(t, i18n.Init())

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// A bare ":memory:" sqlite DSN gives every new pooled connection its own
	// private database; concurrent redemption tests need every goroutine to
	// see the same one, so the pool is pinned to a single connection.
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.UserSession{},
		&model.AuthFlow{},
		&model.ExternalIdentityClaim{},
	))

	previousDB := model.DB
	previousRedis := common.RedisEnabled
	previousSecret := common.SessionSecret
	t.Cleanup(func() {
		model.DB = previousDB
		common.RedisEnabled = previousRedis
		common.SessionSecret = previousSecret
	})
	model.DB = db
	common.RedisEnabled = false
	common.SessionSecret = "telegram-link-test-secret"

	return db
}

type telegramLinkTestActor struct {
	user    *model.User
	session *model.UserSession
}

func createTelegramLinkTestActor(t *testing.T, db *gorm.DB, name string) telegramLinkTestActor {
	t.Helper()
	now := time.Now()
	user := &model.User{
		Username: name, Password: "password-placeholder", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1, AffCode: name,
	}
	require.NoError(t, db.Create(user).Error)
	session := &model.UserSession{
		SID: name + "-session", UserID: user.Id, Version: 1, UserAuthVersion: user.AuthVersion,
		Status: model.UserSessionStatusActive, RefreshHash: name + "-refresh-hash", LoginMethod: "password",
		CreatedAt: now.Unix(), LastActiveAt: now.Unix(), ExpiresAt: now.Add(time.Hour).Unix(),
	}
	require.NoError(t, model.CreateUserSession(session))
	return telegramLinkTestActor{user: user, session: session}
}

func telegramLinkRedeemContext(t *testing.T, actor telegramLinkTestActor, code string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	body, err := common.Marshal(map[string]any{"code": code})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/user/telegram/link/confirm", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", actor.user.Id)
	ctx.Set("session_id", actor.session.SID)
	ctx.Set("auth_version", actor.user.AuthVersion)
	ctx.Set("session_version", actor.session.Version)
	return ctx, recorder
}

func decodeTelegramLinkResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var response map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestTelegramLinkRedeemValidCodeBindsAccount(t *testing.T) {
	db := setupTelegramLinkTestDB(t)
	actor := createTelegramLinkTestActor(t, db, "link-valid-user")

	code, _, err := model.IssueTelegramLinkCode("telegram-valid-1", "  @LinkedHandle ")
	require.NoError(t, err)

	ctx, recorder := telegramLinkRedeemContext(t, actor, code)
	TelegramLinkRedeem(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	response := decodeTelegramLinkResponse(t, recorder)
	require.True(t, response["success"].(bool), response["message"])

	var stored model.User
	require.NoError(t, db.First(&stored, actor.user.Id).Error)
	assert.Equal(t, "telegram-valid-1", stored.TelegramId)
	assert.Equal(t, "linkedhandle", stored.TelegramUsername)

	var claim model.ExternalIdentityClaim
	require.NoError(t, db.Where("provider = ? AND subject = ?", model.ExternalIdentityProviderTelegram, "telegram-valid-1").First(&claim).Error)
	assert.Equal(t, actor.user.Id, claim.UserId)
}

func TestTelegramLinkRedeemCodeReuseRejected(t *testing.T) {
	db := setupTelegramLinkTestDB(t)
	first := createTelegramLinkTestActor(t, db, "link-reuse-first")
	second := createTelegramLinkTestActor(t, db, "link-reuse-second")

	code, _, err := model.IssueTelegramLinkCode("telegram-reuse-1", "")
	require.NoError(t, err)

	ctx, recorder := telegramLinkRedeemContext(t, first, code)
	TelegramLinkRedeem(ctx)
	require.True(t, decodeTelegramLinkResponse(t, recorder)["success"].(bool))

	ctx2, recorder2 := telegramLinkRedeemContext(t, second, code)
	TelegramLinkRedeem(ctx2)
	response2 := decodeTelegramLinkResponse(t, recorder2)
	assert.False(t, response2["success"].(bool))

	var stored model.User
	require.NoError(t, db.First(&stored, second.user.Id).Error)
	assert.Empty(t, stored.TelegramId, "reused code must not create a second binding")
}

func TestTelegramLinkRedeemExpiredCodeRejected(t *testing.T) {
	db := setupTelegramLinkTestDB(t)
	actor := createTelegramLinkTestActor(t, db, "link-expired-user")

	code, _, err := model.IssueTelegramLinkCode("telegram-expired-1", "")
	require.NoError(t, err)
	require.NoError(t, db.Model(&model.AuthFlow{}).
		Where("purpose = ?", model.AuthFlowPurposeTelegramLink).
		Update("expires_at", time.Now().Add(-time.Second)).Error)

	ctx, recorder := telegramLinkRedeemContext(t, actor, code)
	TelegramLinkRedeem(ctx)

	response := decodeTelegramLinkResponse(t, recorder)
	assert.False(t, response["success"].(bool))
	var stored model.User
	require.NoError(t, db.First(&stored, actor.user.Id).Error)
	assert.Empty(t, stored.TelegramId)
}

func TestTelegramLinkRedeemUnknownCodeRejected(t *testing.T) {
	db := setupTelegramLinkTestDB(t)
	actor := createTelegramLinkTestActor(t, db, "link-unknown-user")

	ctx, recorder := telegramLinkRedeemContext(t, actor, "NOSUCHCODE")
	TelegramLinkRedeem(ctx)

	response := decodeTelegramLinkResponse(t, recorder)
	assert.False(t, response["success"].(bool))
	var stored model.User
	require.NoError(t, db.First(&stored, actor.user.Id).Error)
	assert.Empty(t, stored.TelegramId)
}

// TestTelegramLinkRedeemIndistinguishableFailureResponses proves the three
// code-validity failure modes — unknown, expired, already-consumed — cannot
// be told apart by the caller (specs/telegram/account-link/spec.md, "Link
// codes are single-use and expire").
func TestTelegramLinkRedeemIndistinguishableFailureResponses(t *testing.T) {
	db := setupTelegramLinkTestDB(t)

	unknownActor := createTelegramLinkTestActor(t, db, "indist-unknown")
	ctxUnknown, recorderUnknown := telegramLinkRedeemContext(t, unknownActor, "AAAAAAAA")
	TelegramLinkRedeem(ctxUnknown)

	expiredCode, _, err := model.IssueTelegramLinkCode("telegram-indist-expired", "")
	require.NoError(t, err)
	require.NoError(t, db.Model(&model.AuthFlow{}).
		Where("purpose = ?", model.AuthFlowPurposeTelegramLink).
		Update("expires_at", time.Now().Add(-time.Second)).Error)
	expiredActor := createTelegramLinkTestActor(t, db, "indist-expired")
	ctxExpired, recorderExpired := telegramLinkRedeemContext(t, expiredActor, expiredCode)
	TelegramLinkRedeem(ctxExpired)

	consumedCode, _, err := model.IssueTelegramLinkCode("telegram-indist-consumed", "")
	require.NoError(t, err)
	consumedFirstActor := createTelegramLinkTestActor(t, db, "indist-consumed-first")
	ctxConsumeFirst, recorderConsumeFirst := telegramLinkRedeemContext(t, consumedFirstActor, consumedCode)
	TelegramLinkRedeem(ctxConsumeFirst)
	require.True(t, decodeTelegramLinkResponse(t, recorderConsumeFirst)["success"].(bool))
	consumedSecondActor := createTelegramLinkTestActor(t, db, "indist-consumed-second")
	ctxConsumeSecond, recorderConsumeSecond := telegramLinkRedeemContext(t, consumedSecondActor, consumedCode)
	TelegramLinkRedeem(ctxConsumeSecond)

	unknownResponse := decodeTelegramLinkResponse(t, recorderUnknown)
	expiredResponse := decodeTelegramLinkResponse(t, recorderExpired)
	consumedResponse := decodeTelegramLinkResponse(t, recorderConsumeSecond)

	assert.False(t, unknownResponse["success"].(bool))
	assert.False(t, expiredResponse["success"].(bool))
	assert.False(t, consumedResponse["success"].(bool))
	assert.Equal(t, unknownResponse["message"], expiredResponse["message"])
	assert.Equal(t, expiredResponse["message"], consumedResponse["message"])
}

func TestTelegramLinkRedeemConcurrentRedemptionBindsExactlyOnce(t *testing.T) {
	db := setupTelegramLinkTestDB(t)

	code, _, err := model.IssueTelegramLinkCode("telegram-concurrent-1", "")
	require.NoError(t, err)

	const contenders = 5
	actors := make([]telegramLinkTestActor, contenders)
	for i := 0; i < contenders; i++ {
		actors[i] = createTelegramLinkTestActor(t, db, "link-concurrent-"+string(rune('a'+i)))
	}

	successes := make([]bool, contenders)
	var wg sync.WaitGroup
	wg.Add(contenders)
	for i := 0; i < contenders; i++ {
		go func(idx int) {
			defer wg.Done()
			ctx, recorder := telegramLinkRedeemContext(t, actors[idx], code)
			TelegramLinkRedeem(ctx)
			successes[idx] = decodeTelegramLinkResponse(t, recorder)["success"].(bool)
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, ok := range successes {
		if ok {
			successCount++
		}
	}
	assert.Equal(t, 1, successCount, "exactly one concurrent redemption must succeed")

	var claims []model.ExternalIdentityClaim
	require.NoError(t, db.Where("provider = ? AND subject = ?", model.ExternalIdentityProviderTelegram, "telegram-concurrent-1").Find(&claims).Error)
	require.Len(t, claims, 1, "the telegram identity must be bound exactly once")
}

// --- Group 5.3 / 10.5: redemption guards --------------------------------

func TestTelegramLinkRedeemGuardIdentityAlreadyBoundElsewhere(t *testing.T) {
	db := setupTelegramLinkTestDB(t)
	owner := createTelegramLinkTestActor(t, db, "guard-identity-owner")
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return model.ClaimExternalIdentityWithTx(tx, model.ExternalIdentityProviderTelegram, "telegram-guard-identity", owner.user.Id)
	}))
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", owner.user.Id).Update("telegram_id", "telegram-guard-identity").Error)

	// A code is crafted directly (bypassing issuance's already-bound guard) to
	// simulate the race where the identity is bound between issuance and
	// redemption; redemption must still refuse it.
	_, _, err := model.CreateAuthFlow(model.AuthFlowCreate{
		Purpose:   model.AuthFlowPurposeTelegramLink,
		Provider:  model.ExternalIdentityProviderTelegram,
		Payload:   `{"telegram_user_id":"telegram-guard-identity"}`,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Token:     "GUARDCODE",
	})
	require.NoError(t, err)

	challenger := createTelegramLinkTestActor(t, db, "guard-identity-challenger")
	ctx, recorder := telegramLinkRedeemContext(t, challenger, "GUARDCODE")
	TelegramLinkRedeem(ctx)

	response := decodeTelegramLinkResponse(t, recorder)
	assert.False(t, response["success"].(bool))

	var storedOwner, storedChallenger model.User
	require.NoError(t, db.First(&storedOwner, owner.user.Id).Error)
	require.NoError(t, db.First(&storedChallenger, challenger.user.Id).Error)
	assert.Equal(t, "telegram-guard-identity", storedOwner.TelegramId, "the original binding must be untouched")
	assert.Empty(t, storedChallenger.TelegramId, "the challenger must gain no binding")
}

func TestTelegramLinkRedeemGuardAccountAlreadyLinked(t *testing.T) {
	db := setupTelegramLinkTestDB(t)
	actor := createTelegramLinkTestActor(t, db, "guard-account-user")
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return model.ClaimExternalIdentityWithTx(tx, model.ExternalIdentityProviderTelegram, "telegram-existing-binding", actor.user.Id)
	}))
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", actor.user.Id).Update("telegram_id", "telegram-existing-binding").Error)

	code, _, err := model.IssueTelegramLinkCode("telegram-new-binding-attempt", "")
	require.NoError(t, err)

	ctx, recorder := telegramLinkRedeemContext(t, actor, code)
	TelegramLinkRedeem(ctx)

	response := decodeTelegramLinkResponse(t, recorder)
	assert.False(t, response["success"].(bool))
	var stored model.User
	require.NoError(t, db.First(&stored, actor.user.Id).Error)
	assert.Equal(t, "telegram-existing-binding", stored.TelegramId, "the existing binding must be retained")
}

func TestTelegramLinkRedeemGuardDisabledAccount(t *testing.T) {
	db := setupTelegramLinkTestDB(t)
	actor := createTelegramLinkTestActor(t, db, "guard-disabled-user")
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", actor.user.Id).Update("status", common.UserStatusDisabled).Error)

	code, _, err := model.IssueTelegramLinkCode("telegram-disabled-account", "")
	require.NoError(t, err)

	ctx, recorder := telegramLinkRedeemContext(t, actor, code)
	TelegramLinkRedeem(ctx)

	response := decodeTelegramLinkResponse(t, recorder)
	assert.False(t, response["success"].(bool))
	var stored model.User
	require.NoError(t, db.First(&stored, actor.user.Id).Error)
	assert.Empty(t, stored.TelegramId)
}

func TestTelegramLinkRedeemGuardRevokedSession(t *testing.T) {
	db := setupTelegramLinkTestDB(t)
	actor := createTelegramLinkTestActor(t, db, "guard-revoked-user")
	require.NoError(t, db.Model(&model.UserSession{}).Where("sid = ?", actor.session.SID).Update("revoked_at", time.Now().Unix()).Error)

	code, _, err := model.IssueTelegramLinkCode("telegram-revoked-session", "")
	require.NoError(t, err)

	ctx, recorder := telegramLinkRedeemContext(t, actor, code)
	TelegramLinkRedeem(ctx)

	response := decodeTelegramLinkResponse(t, recorder)
	assert.False(t, response["success"].(bool))
	var stored model.User
	require.NoError(t, db.First(&stored, actor.user.Id).Error)
	assert.Empty(t, stored.TelegramId)
}
