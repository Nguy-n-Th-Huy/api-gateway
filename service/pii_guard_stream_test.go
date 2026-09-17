package service_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPIIGuardRestoresAStreamedResponse drives the streaming path: the upstream
// answers with text/event-stream, so the response body is forwarded chunk by
// chunk and the guard has to restore the real value inside the streamed frames.
func TestPIIGuardRestoresAStreamedResponse(t *testing.T) {
	enablePIIGuard(t, nil)
	service.InitHttpClient()
	gin.SetMode(gin.TestMode)

	const prompt = "Email alex@example.com about the invoice"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Sure\"}}]}\n\n")
		flusher.Flush()
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"alex@example.com\"}}]}\n\n")
		flusher.Flush()
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer upstream.Close()

	filter := service.NewPIIFilter()
	require.NotNil(t, filter)

	// The fake the guard produced for the prompt is exactly what the upstream
	// would be able to hand back, so the stream carries that value.
	masked, err := filter.MaskOutboundBody(newRelayTestContext(), chatRequestBody(t, prompt))
	require.NoError(t, err)
	fake := fakeAddressIn(t, masked)
	require.NotContains(t, fake, "alex@example.com")

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: upstream.URL},
		PIIFilter:   filter,
		IsStream:    true,
	}
	request, err := http.NewRequest(http.MethodPost, upstream.URL+"/v1/chat/completions",
		strings.NewReader(`{"model":"gpt-4o","stream":true,"messages":[{"role":"user","content":"`+fake+`"}]}`))
	require.NoError(t, err)

	response, err := channel.DoRequest(newRelayTestContext(), request, info)
	require.NoError(t, err)
	defer response.Body.Close()

	require.Equal(t, "text/event-stream", response.Header.Get("Content-Type"))
	streamed, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	body := string(streamed)
	assert.Contains(t, body, "alex@example.com", "the streamed frame must carry the real address")
	assert.NotContains(t, body, fake, "the fake must never reach the client")
	assert.Contains(t, body, "data: [DONE]", "the stream framing must survive the rewrite")
	assert.Contains(t, body, "\"content\":\"Sure\"", "a frame without PII must pass through unchanged")
}

// fakeAddressIn reads the stand-in the guard put into a masked chat body.
func fakeAddressIn(t *testing.T, maskedBody []byte) string {
	t.Helper()
	var payload struct {
		Messages []struct {
			Content string `json:"content"`
		} `json:"messages"`
	}
	require.NoError(t, json.Unmarshal(maskedBody, &payload))
	require.NotEmpty(t, payload.Messages)
	return extractAddress(t, payload.Messages[0].Content)
}
