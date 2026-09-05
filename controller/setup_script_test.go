package controller

import (
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tomlLine mirrors how buildCodexScript embeds a TOML line: the whole line
// is later JSON-encoded as one element of a JS array literal, so a line
// containing quote characters (like `wire_api = "responses"`) appears
// escaped in the rendered script rather than verbatim.
func tomlLine(t *testing.T, line string) string {
	t.Helper()
	encoded, err := common.Marshal(line)
	require.NoError(t, err)
	return string(encoded)
}

// ---- 2b.5: rendered script content per application ----

func TestBuildClaudeCodeScriptDocumentsPathBaseURLAndNoEnvVar(t *testing.T) {
	baseURL := "https://gw.example.com"
	key := "testkey1234567890"
	models := map[string]string{
		"opus":     "claude-opus-4",
		"sonnet":   "claude-sonnet-4",
		"haiku":    "claude-haiku-4",
		"subagent": "claude-haiku-4",
	}

	js := buildClaudeCodeScript(baseURL, key, models)
	assert.Contains(t, js, `".claude"`)
	assert.Contains(t, js, `"settings.json"`)
	assert.Contains(t, js, "ANTHROPIC_BASE_URL")
	assert.Contains(t, js, "ANTHROPIC_AUTH_TOKEN")
	assert.Contains(t, js, baseURL)
	assert.Contains(t, js, key)
	assert.NotContains(t, renderPosixScript(js, nil), "export ")

	ps := buildClaudeCodeScriptPS(baseURL, key, models)
	assert.Contains(t, ps, ".claude")
	assert.Contains(t, ps, "settings.json")
	assert.Contains(t, ps, "ANTHROPIC_BASE_URL")
	assert.Contains(t, ps, "ANTHROPIC_AUTH_TOKEN")
	assert.Contains(t, ps, baseURL)
	assert.Contains(t, ps, key)
	assert.NotContains(t, renderPowerShellScript(ps, nil), "SetEnvironmentVariable")
}

func TestBuildCodexScriptDocumentsPathAndPersistsExactlyOneEnvVar(t *testing.T) {
	baseURL := "https://gw.example.com"
	key := "testkey1234567890"
	models := map[string]string{"small": "gpt-5-mini", "medium": "gpt-5", "large": "gpt-5-pro"}

	js := buildCodexScript(baseURL, key, models)
	assert.Contains(t, js, `".codex"`)
	assert.Contains(t, js, `"config.toml"`)
	assert.Contains(t, js, baseURL+"/v1", "Codex requires base_url to end in /v1")
	assert.Contains(t, js, tomlLine(t, `wire_api = "responses"`))
	assert.Contains(t, js, "NEWAPI_API_KEY")
	assert.Contains(t, js, "gpt-5-mini")
	assert.Contains(t, js, "gpt-5")
	assert.Contains(t, js, "gpt-5-pro")

	envVar := &setupScriptEnvVar{
		Name:    "NEWAPI_API_KEY",
		Value:   key,
		Comment: "Codex reads the API key only from this environment variable.",
	}

	ps := buildCodexScriptPS(baseURL, key, models)
	assert.Contains(t, ps, ".codex")
	assert.Contains(t, ps, "config.toml")
	assert.Contains(t, ps, baseURL+"/v1", "Codex requires base_url to end in /v1")
	assert.Contains(t, ps, "wire_api")
	assert.Contains(t, ps, "responses")
	assert.Contains(t, ps, "NEWAPI_API_KEY")
	assert.Contains(t, ps, "gpt-5-mini")
	assert.Contains(t, ps, "gpt-5")
	assert.Contains(t, ps, "gpt-5-pro")

	psScript := renderPowerShellScript(ps, envVar)
	assert.Equal(t, 1, strings.Count(psScript, "SetEnvironmentVariable"), "exactly one persisted environment variable")
	assert.Contains(t, psScript, "NEWAPI_API_KEY")

	posixScript := renderPosixScript(js, envVar)
	assert.Contains(t, posixScript, "export NEWAPI_API_KEY=")
}

func TestBuildOpenCodeScriptDocumentsPathBaseURLAndModelForm(t *testing.T) {
	baseURL := "https://gw.example.com"
	key := "testkey1234567890"
	models := map[string]string{"model": "gpt-4o", "small_model": "gpt-4o-mini"}
	available := []string{"gpt-4o", "gpt-4o-mini"}

	js := buildOpenCodeScript(baseURL, key, models, available)
	assert.Contains(t, js, `".config"`)
	assert.Contains(t, js, `"opencode"`)
	assert.Contains(t, js, `"opencode.json"`)
	assert.Contains(t, js, baseURL)
	assert.Contains(t, js, `"newapi/gpt-4o"`)
	assert.Contains(t, js, `"newapi/gpt-4o-mini"`)
	assert.Contains(t, js, "@ai-sdk/openai-compatible")
	assert.NotContains(t, renderPosixScript(js, nil), "export ")

	ps := buildOpenCodeScriptPS(baseURL, key, models, available)
	assert.Contains(t, ps, ".config")
	assert.Contains(t, ps, "opencode")
	assert.Contains(t, ps, "opencode.json")
	assert.Contains(t, ps, baseURL)
	assert.Contains(t, ps, "newapi/gpt-4o")
	assert.Contains(t, ps, "newapi/gpt-4o-mini")
	assert.Contains(t, ps, "@ai-sdk/openai-compatible")
	assert.NotContains(t, renderPowerShellScript(ps, nil), "SetEnvironmentVariable")
}

func TestBuildPiScriptDocumentsPathAndRegistersGroupModelsWithoutSelectors(t *testing.T) {
	baseURL := "https://gw.example.com"
	key := "testkey1234567890"
	available := []string{"gpt-4o", "gpt-4o-mini"}

	js := buildPiScript(baseURL, key, available)
	assert.Contains(t, js, `".pi"`)
	assert.Contains(t, js, `"agent"`)
	assert.Contains(t, js, `"models.json"`)
	assert.Contains(t, js, baseURL)
	assert.Contains(t, js, "gpt-4o")
	assert.Contains(t, js, "gpt-4o-mini")
	assert.NotContains(t, renderPosixScript(js, nil), "export ")

	ps := buildPiScriptPS(baseURL, key, available)
	assert.Contains(t, ps, ".pi")
	assert.Contains(t, ps, "agent")
	assert.Contains(t, ps, "models.json")
	assert.Contains(t, ps, baseURL)
	assert.Contains(t, ps, "gpt-4o")
	assert.Contains(t, ps, "gpt-4o-mini")
	assert.NotContains(t, renderPowerShellScript(ps, nil), "SetEnvironmentVariable")
}

func TestBuildOhMyPiScriptDocumentsPathAndOnlyTouchesProvidersRoot(t *testing.T) {
	baseURL := "https://gw.example.com"
	key := "testkey1234567890"
	available := []string{"gpt-4o"}

	js := buildOhMyPiScript(baseURL, key, available)
	assert.Contains(t, js, `".omp"`)
	assert.Contains(t, js, `"agent"`)
	assert.Contains(t, js, `"models.yml"`)
	assert.Contains(t, js, baseURL)
	assert.Contains(t, js, `"providers:"`, "the script must only ever touch the documented providers root key")
	assert.NotContains(t, renderPosixScript(js, nil), "export ")

	ps := buildOhMyPiScriptPS(baseURL, key, available)
	assert.Contains(t, ps, ".omp")
	assert.Contains(t, ps, "agent")
	assert.Contains(t, ps, "models.yml")
	assert.Contains(t, ps, baseURL)
	assert.Contains(t, ps, "providers:", "the script must only ever touch the documented providers root key")
	assert.NotContains(t, renderPowerShellScript(ps, nil), "SetEnvironmentVariable")
}

// ---- 2b.5: handler-level validation ----

func TestGetTokenSetupScriptRejectsUnknownApplication(t *testing.T) {
	setupTokenControllerTestDB(t)

	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/setup/script?app=nonexistent&os=unix&key=abc", nil, 0)
	GetTokenSetupScript(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestGetTokenSetupScriptRejectsUnknownOS(t *testing.T) {
	setupTokenControllerTestDB(t)

	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/setup/script?app=pi&os=plan9&key=abc", nil, 0)
	GetTokenSetupScript(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestGetTokenSetupScriptRejectsMissingSlot(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Ability{}))

	token := seedToken(t, db, 1, "slot-test", "slottest1234abcd")
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: "claude-opus-4", ChannelId: 1, Enabled: true}).Error)

	// model_haiku is deliberately omitted.
	url := "/api/setup/script?app=claude-code&os=unix&key=" + token.Key +
		"&model_opus=claude-opus-4&model_sonnet=claude-opus-4&model_subagent=claude-opus-4"
	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, url, nil, 0)
	GetTokenSetupScript(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "haiku")
}

func TestGetTokenSetupScriptRejectsModelOutsideGroup(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Ability{}))

	token := seedToken(t, db, 1, "model-test", "modeltest1234abc")
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: "claude-opus-4", ChannelId: 1, Enabled: true}).Error)

	url := "/api/setup/script?app=claude-code&os=unix&key=" + token.Key +
		"&model_opus=claude-opus-4&model_sonnet=not-enabled-model&model_haiku=claude-opus-4&model_subagent=claude-opus-4"
	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, url, nil, 0)
	GetTokenSetupScript(ctx)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestGetTokenSetupScriptRejectsUnknownKeyWithGenericMessage(t *testing.T) {
	setupTokenControllerTestDB(t)

	url := "/api/setup/script?app=pi&os=unix&key=doesnotexist12345"
	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, url, nil, 0)
	GetTokenSetupScript(ctx)

	// No script content should be returned for an unknown key.
	assert.NotContains(t, recorder.Body.String(), "require(\"fs\")")
}

func TestGetTokenSetupScriptSucceedsAndReturnsScriptForValidRequest(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Ability{}))

	token := seedToken(t, db, 1, "ok-test", "oktest12345678ab")
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: "claude-opus-4", ChannelId: 1, Enabled: true}).Error)

	url := "/api/setup/script?app=claude-code&os=unix&key=" + token.Key +
		"&model_opus=claude-opus-4&model_sonnet=claude-opus-4&model_haiku=claude-opus-4&model_subagent=claude-opus-4"
	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, url, nil, 0)
	GetTokenSetupScript(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), token.Key)
	assert.Contains(t, recorder.Body.String(), "claude-opus-4")
}

func TestGetTokenSetupScriptHasNoSlotsForPiAndOhMyPi(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Ability{}))

	token := seedToken(t, db, 1, "no-slot-test", "noslottest1234ab")
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: "claude-opus-4", ChannelId: 1, Enabled: true}).Error)

	for _, app := range []string{"pi", "oh-my-pi"} {
		url := "/api/setup/script?app=" + app + "&os=unix&key=" + token.Key
		ctx, recorder := newAuthenticatedContext(t, http.MethodGet, url, nil, 0)
		GetTokenSetupScript(ctx)
		require.Equal(t, http.StatusOK, recorder.Code, "app=%s should not require any model slot", app)
	}
}
