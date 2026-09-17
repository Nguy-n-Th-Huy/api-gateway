package piiguard

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaskOutboundBodyMasksMessageTextOnly(t *testing.T) {
	engine := NewEngine(pseudonymConfig())
	body := []byte(`{"model":"gpt-4o","messages":[{"role":"system","content":"Reply in Vietnamese"},` +
		`{"role":"user","content":"Email alex@example.com about invoice 4111 1111 1111 1111"}],` +
		`"temperature":0.2}`)

	result := engine.MaskOutboundBody(body)

	require.True(t, result.Masked)
	assert.Equal(t, OutcomeMasked, result.Outcome)
	assert.Equal(t, []string{"messages[1].content"}, result.Fields)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(result.Body, &payload))
	assert.Equal(t, "gpt-4o", payload["model"], "the model name must survive byte for byte")
	assert.InDelta(t, 0.2, payload["temperature"], 0.0001)
	messages := payload["messages"].([]any)
	assert.Equal(t, "Reply in Vietnamese", messages[0].(map[string]any)["content"], "text without PII is unchanged")
	maskedText := messages[1].(map[string]any)["content"].(string)
	assert.NotContains(t, maskedText, "alex@example.com")
	assert.NotContains(t, maskedText, "4111 1111 1111 1111")
	assert.Contains(t, maskedText, "about invoice")
}

func TestMaskOutboundBodyLeavesMediaAndMetadataAlone(t *testing.T) {
	engine := NewEngine(pseudonymConfig())
	body := []byte(`{"messages":[{"role":"user","content":[{"type":"text","text":"describe alex@example.com"},` +
		`{"type":"image_url","image_url":{"url":"data:image/png;base64,aGVsbG8=","detail":"high"}}]}],` +
		`"user":"tenant-42","tool_choice":"auto","metadata":{"trace":"abc"}}`)

	result := engine.MaskOutboundBody(body)

	require.True(t, result.Masked)
	rewritten := string(result.Body)
	assert.NotContains(t, rewritten, "alex@example.com")
	assert.Contains(t, rewritten, "aGVsbG8=", "an inline payload must survive")
	assert.Contains(t, rewritten, "tenant-42", "a tenant identifier is not prose")
	assert.Contains(t, rewritten, `"trace":"abc"`)
}

func TestMaskOutboundBodyNeverRewritesAMediaPayload(t *testing.T) {
	engine := NewEngine(pseudonymConfig())
	// The same key that carries a caption in one API carries an inline payload
	// in another, so the payload shape has to win over the field name.
	body := []byte(`{"messages":[{"role":"user","content":` +
		`[{"type":"image_url","image_url":{"data":"data:image/png;base64,alex@example.com"}}]}]}`)

	result := engine.MaskOutboundBody(body)

	assert.False(t, result.Masked, "an inline payload is not prose: %s", result.Body)
	assert.Contains(t, string(result.Body), "base64,alex@example.com")
}

func TestMaskOutboundBodyIsInertWhenNothingMatches(t *testing.T) {
	engine := NewEngine(pseudonymConfig())
	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"Summarise the report"}]}`)

	result := engine.MaskOutboundBody(body)

	assert.False(t, result.Masked, "an untouched body must not be re-marshalled")
	assert.Equal(t, body, result.Body)
	assert.Equal(t, OutcomeMasked, result.Outcome)
	assert.Zero(t, result.Stats.Spans)
}

func TestMaskOutboundBodySkipsWhenGuardIsDisabled(t *testing.T) {
	config := DefaultConfig()
	engine := NewEngine(config)
	body := []byte(`{"messages":[{"role":"user","content":"mail alex@example.com"}]}`)

	result := engine.MaskOutboundBody(body)

	assert.Equal(t, OutcomeSkippedDisabled, result.Outcome)
	assert.Equal(t, body, result.Body)
}

func TestMaskOutboundBodySkipsBodiesAboveTheLimit(t *testing.T) {
	config := pseudonymConfig()
	config.MaxBodyBytes = 64
	engine := NewEngine(config)
	body := []byte(`{"messages":[{"role":"user","content":"mail alex@example.com and more text to exceed the limit"}]}`)

	result := engine.MaskOutboundBody(body)

	assert.Equal(t, OutcomeSkippedTooLarge, result.Outcome)
	assert.Equal(t, body, result.Body, "a body above the limit is forwarded untouched, never half rewritten")
}

func TestMaskOutboundBodySkipsUnparsableBodies(t *testing.T) {
	engine := NewEngine(pseudonymConfig())

	result := engine.MaskOutboundBody([]byte(`{"messages":`))

	assert.Equal(t, OutcomeSkippedMalformed, result.Outcome)
	assert.Equal(t, []byte(`{"messages":`), result.Body)
}

func TestMaskOutboundBodyCoversResponsesAPIInput(t *testing.T) {
	engine := NewEngine(pseudonymConfig())
	body := []byte(`{"model":"gpt-4o","instructions":"Contact alex@example.com",` +
		`"input":[{"role":"user","content":[{"type":"input_text","text":"from sam@example.com"}]}]}`)

	result := engine.MaskOutboundBody(body)

	require.True(t, result.Masked)
	rewritten := string(result.Body)
	assert.NotContains(t, rewritten, "alex@example.com")
	assert.NotContains(t, rewritten, "sam@example.com")
	assert.Contains(t, result.Fields, "instructions")
}

func TestMaskOutboundBodyCoversEmbeddingInput(t *testing.T) {
	engine := NewEngine(pseudonymConfig())

	single := engine.MaskOutboundBody([]byte(`{"model":"text-embedding-3-small","input":"alex@example.com"}`))
	require.True(t, single.Masked)
	assert.NotContains(t, string(single.Body), "alex@example.com")
	assert.Contains(t, string(single.Body), "text-embedding-3-small")

	batch := engine.MaskOutboundBody([]byte(`{"model":"text-embedding-3-small","input":["alex@example.com","plain text"]}`))
	require.True(t, batch.Masked)
	assert.NotContains(t, string(batch.Body), "alex@example.com")
	assert.Contains(t, string(batch.Body), "plain text")
}

func TestUnmaskKnownFieldsRestoresResponseText(t *testing.T) {
	engine := NewEngine(pseudonymConfig())
	request := []byte(`{"messages":[{"role":"user","content":"mail alex@example.com"}]}`)
	maskedRequest := engine.MaskOutboundBody(request)
	require.True(t, maskedRequest.Masked)

	// A realistic response echoes the fake the model saw.
	var requestPayload map[string]any
	require.NoError(t, json.Unmarshal(maskedRequest.Body, &requestPayload))
	fake := requestPayload["messages"].([]any)[0].(map[string]any)["content"].(string)
	require.NotContains(t, fake, "alex@example.com")

	responseBody := []byte(`{"choices":[{"message":{"role":"assistant","content":"Sure, I will write to ` + fake + `"}}]}`)
	result := engine.UnmaskKnownFields(responseBody, []string{"choices[0].message.content"})

	require.True(t, result.Masked)
	assert.Contains(t, string(result.Body), "alex@example.com")
	assert.NotContains(t, string(result.Body), fake)
}

func TestSplitPathParsesNestedIndices(t *testing.T) {
	segments := splitPath("choices[0].message.content")

	require.Len(t, segments, 3)
	assert.Equal(t, "choices", segments[0].key)
	assert.True(t, segments[0].indexed)
	assert.Equal(t, 0, segments[0].index)
	assert.Equal(t, "content", segments[2].key)
	assert.False(t, segments[2].indexed)
}
