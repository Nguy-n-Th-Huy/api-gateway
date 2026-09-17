package service_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/pkg/piiguard"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/piiguard_setting"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// enablePIIGuard turns the guard on for one test and restores the shipped
// default afterwards, so the package level configuration never leaks between
// tests.
func enablePIIGuard(t *testing.T, configure func(*piiguard.Config)) {
	t.Helper()
	original := piiguard_setting.GetSettings()
	t.Cleanup(func() {
		require.NoError(t, piiguard_setting.UpdateSettings(original))
	})
	enabled := original
	enabled.Enabled = true
	enabled.MaskRequest = true
	enabled.UnmaskResponse = true
	enabled.Secret = "service-test-secret"
	if configure != nil {
		configure(&enabled)
	}
	require.NoError(t, piiguard_setting.UpdateSettings(enabled))
}

// TestPIIGuardKeepsRealValuesOutOfTheUpstreamRequestAndRestoresThemInTheResponse
// drives the real outbound path: the guard rewrites the request body, the HTTP
// layer sends it, and the response body wrapper turns the fake back into the
// real value before the client reads it.
func TestPIIGuardKeepsRealValuesOutOfTheUpstreamRequestAndRestoresThemInTheResponse(t *testing.T) {
	enablePIIGuard(t, nil)
	service.InitHttpClient()
	gin.SetMode(gin.TestMode)

	const prompt = "Email alex@example.com about invoice 4111 1111 1111 1111"

	received := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		received <- string(body)

		// The upstream only ever sees fakes, so its answer quotes a fake.
		var payload struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(body, &payload); err != nil || len(payload.Messages) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"I wrote to `+
			extractAddress(t, payload.Messages[0].Content)+`"}}]}`)
	}))
	defer upstream.Close()

	filter := service.NewPIIFilter()
	require.NotNil(t, filter, "the guard must be active while its settings are enabled")

	maskedBody, err := filter.MaskOutboundBody(newRelayTestContext(), chatRequestBody(t, prompt))
	require.NoError(t, err)
	require.NotContains(t, string(maskedBody), "alex@example.com")

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: upstream.URL},
		PIIFilter:   filter,
	}
	request, err := http.NewRequest(http.MethodPost, upstream.URL+"/v1/chat/completions", bytes.NewReader(maskedBody))
	require.NoError(t, err)

	response, err := channel.DoRequest(newRelayTestContext(), request, info)
	require.NoError(t, err)
	defer response.Body.Close()

	upstreamBody := <-received
	assert.NotContains(t, upstreamBody, "alex@example.com", "the real address must never leave the gateway")
	assert.NotContains(t, upstreamBody, "4111 1111 1111 1111", "the real card number must never leave the gateway")

	clientBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Contains(t, string(clientBody), "alex@example.com", "the client must still receive the real address")
	assert.NotContains(t, string(clientBody), "pii-", "no fake may reach the client")
}

func TestPIIGuardLeavesTheUpstreamBodyAloneWhenDisabled(t *testing.T) {
	// The shipped default is off, so nothing opts in here.
	require.False(t, piiguard_setting.GetSettings().Active())

	assert.Nil(t, service.NewPIIFilter(), "a disabled guard must not allocate a filter")
}

func newRelayTestContext() *gin.Context {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	return ctx
}

func chatRequestBody(t *testing.T, prompt string) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"model":    "gpt-4o",
		"messages": []map[string]any{{"role": "user", "content": prompt}},
	})
	require.NoError(t, err)
	return body
}

// extractAddress reads back the address the upstream was given, which is the
// only part of the prompt the fake replacement kept recognisable.
func extractAddress(t *testing.T, text string) string {
	t.Helper()
	start := strings.IndexByte(text, '@')
	require.GreaterOrEqual(t, start, 0, "the masked prompt must still carry an address: %q", text)
	begin, end := start, start
	for begin > 0 && text[begin-1] != ' ' && text[begin-1] != '"' {
		begin--
	}
	for end < len(text) && text[end] != ' ' && text[end] != '"' {
		end++
	}
	return text[begin:end]
}
