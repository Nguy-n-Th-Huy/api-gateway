package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type tokenLogsPage struct {
	Page     int                         `json:"page"`
	PageSize int                         `json:"page_size"`
	Total    int                         `json:"total"`
	Items    []service.PublicKeyLogEntry `json:"items"`
}

// setupTokenLogsTestDB opens the in-memory SQLite harness the token controller
// tests already use and migrates the three tables the log endpoint touches:
// the token being looked up, its owning user, and the log rows themselves.
func setupTokenLogsTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := openTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Token{}, &model.User{}, &model.Log{}))
	return db
}

func seedTokenLog(t *testing.T, db *gorm.DB, tokenId int, createdAt int64, modelName string, quota int, other string) *model.Log {
	t.Helper()

	log := &model.Log{
		UserId:           1,
		Username:         "key-owner",
		CreatedAt:        createdAt,
		Type:             model.LogTypeConsume,
		ModelName:        modelName,
		Quota:            quota,
		PromptTokens:     10,
		CompletionTokens: 5,
		UseTime:          2,
		IsStream:         true,
		ChannelId:        77,
		TokenId:          tokenId,
		Group:            "default",
		Ip:               "203.0.113.9",
		Other:            other,
	}
	require.NoError(t, db.Create(log).Error)
	return log
}

// seedTwoKeysWithLogs creates two keys whose log rows interleave, so a query
// that forgot its token_id filter would visibly return the other key's rows.
func seedTwoKeysWithLogs(t *testing.T, db *gorm.DB) (checked *model.Token, other *model.Token) {
	t.Helper()

	checked = seedToken(t, db, 1, "my-key", "chk1234chk12345678")
	other = seedToken(t, db, 1, "other-key", "oth1234oth12345678")

	seedTokenLog(t, db, other.Id, 150, "other-model", 999, `{"model_ratio":1.5}`)
	seedTokenLog(t, db, checked.Id, 100, "old-model", 300, `{"model_ratio":1.5}`)
	seedTokenLog(t, db, checked.Id, 200, "new-model", 500, `{"model_ratio":1.5}`)
	return checked, other
}

func decodeTokenLogsPage(t *testing.T, recorder *httptest.ResponseRecorder) tokenLogsPage {
	t.Helper()

	response := decodeAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)

	var page tokenLogsPage
	require.NoError(t, common.Unmarshal(response.Data, &page))
	return page
}

func TestCheckTokenLogsReturnsOnlyTheCheckedKeysEntries(t *testing.T) {
	db := setupTokenLogsTestDB(t)
	seedTwoKeysWithLogs(t, db)

	// The key is pasted in the same forms the report endpoint already accepts:
	// a Bearer prefix, the sk- prefix, surrounding whitespace, and a trailing
	// segment.
	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/token/logs",
		map[string]any{"key": "  Bearer sk-chk1234chk12345678-extra  "}, 0)
	CheckTokenLogs(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	page := decodeTokenLogsPage(t, recorder)

	assert.Equal(t, 2, page.Total)
	require.Len(t, page.Items, 2)
	assert.Equal(t, "new-model", page.Items[0].ModelName, "entries are newest first")
	assert.Equal(t, "old-model", page.Items[1].ModelName)
	assert.Equal(t, 500, page.Items[0].Quota)
	assert.Equal(t, 10, page.Items[0].PromptTokens)
	assert.Equal(t, 5, page.Items[0].CompletionTokens)
	assert.Equal(t, "default", page.Items[0].Group)
	assert.True(t, page.Items[0].IsStream)
	for _, item := range page.Items {
		assert.NotEqual(t, "other-model", item.ModelName, "another key's rows must not be returned")
	}
}

func TestCheckTokenLogsOmitsIdentityAndInfrastructureFields(t *testing.T) {
	db := setupTokenLogsTestDB(t)
	seedTwoKeysWithLogs(t, db)

	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/token/logs",
		map[string]any{"key": "chk1234chk12345678"}, 0)
	CheckTokenLogs(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()

	for _, forbidden := range []string{
		`"user_id"`, `"username"`, `"email"`, `"ip"`, `"channel"`, `"channel_name"`,
		`"token_id"`, `"token_name"`, `"other"`,
	} {
		assert.NotContains(t, body, forbidden, "the public entry must not carry %s", forbidden)
	}
	assert.NotContains(t, body, "key-owner", "the owning account's username must not reach the response")
	assert.NotContains(t, body, "203.0.113.9", "the client IP must not reach the response")

	page := decodeTokenLogsPage(t, recorder)
	require.Len(t, page.Items, 2)
	for _, item := range page.Items {
		assert.NotEmpty(t, item.ModelName)
		assert.NotZero(t, item.CreatedAt)
	}
}

func TestCheckTokenLogsCarriesCacheCountsAndRequestPath(t *testing.T) {
	db := setupTokenLogsTestDB(t)
	token := seedToken(t, db, 1, "rich-key", "ric1234ric12345678")

	// Seeded oldest first: the endpoint orders by row id, which follows
	// insertion order, so this makes the newest entry first in the response.
	seedTokenLog(t, db, token.Id, 100, "gpt-4o", 600, "")
	seedTokenLog(t, db, token.Id, 200, "gpt-4o", 700, `{not json`)
	seedTokenLog(t, db, token.Id, 300, "gpt-4o", 800,
		`{"model_ratio":1.5,"request_path":"/v1/chat/completions","cache_tokens":86272,"cache_creation_tokens":1024}`)
	seedTokenLog(t, db, token.Id, 400, "clamped-model", 900,
		`{"cache_tokens":5000000000,"cache_creation_tokens":-3}`)

	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/token/logs",
		map[string]any{"key": token.Key}, 0)
	CheckTokenLogs(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	page := decodeTokenLogsPage(t, recorder)
	require.Len(t, page.Items, 4)

	clamped := page.Items[0]
	assert.Equal(t, 2147483647, clamped.CacheTokens, "an oversized upstream count is clamped, never wrapped")
	assert.Zero(t, clamped.CacheCreationTokens, "a negative upstream count is reported as zero")

	withMetadata := page.Items[1]
	assert.Equal(t, "/v1/chat/completions", withMetadata.RequestPath)
	assert.Equal(t, 86272, withMetadata.CacheTokens)
	assert.Equal(t, 1024, withMetadata.CacheCreationTokens)

	unreadable := page.Items[2]
	assert.Empty(t, unreadable.RequestPath, "an unreadable metadata blob must not cost the caller its entry")
	assert.Zero(t, unreadable.CacheTokens)
	assert.Equal(t, "gpt-4o", unreadable.ModelName, "the rest of the entry survives unreadable metadata")
	assert.Equal(t, 700, unreadable.Quota)

	missing := page.Items[3]
	assert.Empty(t, missing.RequestPath)
	assert.Zero(t, missing.CacheTokens)
	assert.Zero(t, missing.CacheCreationTokens)

	body := recorder.Body.String()
	assert.NotContains(t, body, `"other"`)
	assert.NotContains(t, body, "model_ratio", "the metadata blob stays out even though three of its values are lifted out")
}

func TestCheckTokenLogsPaginatesAndReportsEveryEntryInTotal(t *testing.T) {
	db := setupTokenLogsTestDB(t)
	checked, _ := seedTwoKeysWithLogs(t, db)
	seedTokenLog(t, db, checked.Id, 300, "newest-model", 700, `{"model_ratio":1.5}`)

	firstCtx, firstRecorder := newAuthenticatedContext(t, http.MethodPost, "/api/token/logs?p=1&page_size=2",
		map[string]any{"key": checked.Key}, 0)
	CheckTokenLogs(firstCtx)

	require.Equal(t, http.StatusOK, firstRecorder.Code)
	firstPage := decodeTokenLogsPage(t, firstRecorder)
	assert.Equal(t, 3, firstPage.Total, "total counts every entry for the key")
	assert.Equal(t, 2, firstPage.PageSize)
	require.Len(t, firstPage.Items, 2)
	assert.Equal(t, "newest-model", firstPage.Items[0].ModelName)

	secondCtx, secondRecorder := newAuthenticatedContext(t, http.MethodPost, "/api/token/logs?p=2&page_size=2",
		map[string]any{"key": checked.Key}, 0)
	CheckTokenLogs(secondCtx)

	require.Equal(t, http.StatusOK, secondRecorder.Code)
	secondPage := decodeTokenLogsPage(t, secondRecorder)
	assert.Equal(t, 3, secondPage.Total)
	require.Len(t, secondPage.Items, 1)
	assert.Equal(t, "old-model", secondPage.Items[0].ModelName)
}

func TestCheckTokenLogsReturnsEmptyItemsWhenKeyHasNoEntries(t *testing.T) {
	db := setupTokenLogsTestDB(t)
	token := seedToken(t, db, 1, "fresh-key", "fre1234fre12345678")

	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/token/logs",
		map[string]any{"key": token.Key}, 0)
	CheckTokenLogs(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	page := decodeTokenLogsPage(t, recorder)
	assert.Zero(t, page.Total)
	assert.NotNil(t, page.Items, "items must serialize as an empty list, not null")
	assert.Empty(t, page.Items)
	assert.Contains(t, recorder.Body.String(), `"items":[]`)
}

func TestCheckTokenLogsUnknownKeyGetsGenericRejection(t *testing.T) {
	setupTokenLogsTestDB(t)

	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/token/logs",
		map[string]any{"key": "nosuchkey12345678"}, 0)
	CheckTokenLogs(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	response := decodeAPIResponse(t, recorder)
	assert.False(t, response.Success)
	assert.NotEmpty(t, response.Message)
	assert.NotContains(t, recorder.Body.String(), "model_name", "an unknown key must not return entries")
}

func TestCheckTokenLogsEmptyKeyReturns400(t *testing.T) {
	setupTokenLogsTestDB(t)

	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/token/logs",
		map[string]any{"key": "   "}, 0)
	CheckTokenLogs(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	response := decodeAPIResponse(t, recorder)
	assert.False(t, response.Success)
}
