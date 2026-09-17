package piiguard

import (
	"encoding/json"
	"strings"
)

// BodyResult reports what happened to one outbound or inbound body.
type BodyResult struct {
	// Body is the rewritten payload, or the original when nothing changed.
	Body []byte
	// Outcome explains an unchanged body: "masked", "skipped_disabled",
	// "skipped_too_large" or "skipped_unparsable".
	Outcome string
	// Stats counts the spans that were replaced.
	Stats Stats
	// Fields names the JSON fields that carried masked text, as paths with
	// array indices, for example "messages[0].content".
	Fields []string
	// Masked reports whether the body differs from the input.
	Masked bool
}

// Body outcomes.
const (
	OutcomeMasked           = "masked"
	OutcomeSkippedDisabled  = "skipped_disabled"
	OutcomeSkippedTooLarge  = "skipped_too_large"
	OutcomeSkippedMalformed = "skipped_unparsable"
)

// maxReportedFields caps the field list so one huge conversation cannot grow
// the audit record without bound.
const maxReportedFields = 24

// textFields are the JSON fields whose value is human text. This is the
// allowlist the reference implementations apply through a schema; using an
// allowlist keeps identifiers, model names and media payloads out of the
// rewrite.
var textFields = map[string]struct{}{
	"content":      {},
	"text":         {},
	"prompt":       {},
	"instructions": {},
	"input":        {},
	"system":       {},
	"query":        {},
	"completion":   {},
	"message":      {},
	"description":  {},
	"transcript":   {},
	"detail":       {},
	"refusal":      {},
	"thinking":     {},
	"reasoning":    {},
}

// structuralFields never carry PII in a relay body and must stay byte
// identical, either because a provider validates them or because the gateway
// itself routes on them.
var structuralFields = map[string]struct{}{
	"id": {}, "model": {}, "role": {}, "type": {}, "name": {}, "object": {},
	"tool_call_id": {}, "tool_use_id": {}, "tool_calls": {}, "tool_choice": {},
	"function": {}, "finish_reason": {}, "stop_reason": {}, "stop_sequence": {},
	"status": {}, "index": {}, "created": {}, "created_at": {}, "usage": {},
	"logprobs": {}, "encoding_format": {}, "response_format": {}, "stream": {},
	"signature": {}, "thinking_signature": {}, "system_fingerprint": {},
	"previous_response_id": {}, "response_id": {}, "request_id": {},
	"safety_identifier": {}, "user": {}, "metadata": {}, "citation": {}, "citations": {},
}

// mediaFields carry a URL or an inline payload. Their textual parts are handled
// separately, so the value itself is never walked as free text.
var mediaFields = map[string]struct{}{
	"image_url": {}, "video_url": {}, "audio_url": {}, "file_url": {}, "input_audio": {},
	"file": {}, "file_data": {}, "b64_json": {}, "base64": {}, "source": {},
	"tools": {}, "tool_resources": {}, "logit_bias": {}, "stop": {},
}

// payloadKeys are the object keys that make a value a media payload rather than
// prose, regardless of the field name it sits under. They are skipped wherever
// they appear, so an inline base64 attachment can never be rewritten.
var payloadKeys = map[string]struct{}{
	"b64_json": {}, "base64": {}, "data": {}, "image_url": {}, "video_url": {},
	"audio_url": {}, "file_url": {}, "file_data": {}, "file_id": {}, "input_audio": {},
	"url": {}, "source": {},
}

// MaskOutboundBody rewrites the JSON body that is about to be sent to an
// upstream provider, replacing every detected PII value.
//
// It returns the original slice when the guard is disabled, when the body is
// larger than the configured bound, or when the body is not a JSON object; in
// every one of those cases Outcome says why.
func (e *Engine) MaskOutboundBody(body []byte) BodyResult {
	if !e.config.Enabled || !e.config.MaskRequest {
		return BodyResult{Body: body, Outcome: OutcomeSkippedDisabled}
	}
	if len(body) > e.config.MaxBodyBytes {
		return BodyResult{Body: body, Outcome: OutcomeSkippedTooLarge}
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return BodyResult{Body: body, Outcome: OutcomeSkippedMalformed}
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return BodyResult{Body: body, Outcome: OutcomeSkippedMalformed}
	}

	context := &walkContext{engine: e, masked: e.MaskText}
	context.walkObject(root, "")
	if len(context.fields) == 0 {
		return BodyResult{Body: body, Outcome: OutcomeMasked, Stats: e.Stats()}
	}
	rewritten, err := json.Marshal(payload)
	if err != nil {
		return BodyResult{Body: body, Outcome: OutcomeSkippedMalformed}
	}
	return BodyResult{
		Body:    rewritten,
		Outcome: OutcomeMasked,
		Stats:   e.Stats(),
		Fields:  context.fields,
		Masked:  true,
	}
}

// UnmaskKnownFields restores masked values inside an upstream response, using
// the field paths recorded while the matching request was masked. It is the
// whole-body counterpart of the streaming reader: use it when the response is
// read as one document, and the reader when it is forwarded chunk by chunk.
func (e *Engine) UnmaskKnownFields(body []byte, paths []string) BodyResult {
	if !e.config.Enabled || !e.config.UnmaskResponse || len(paths) == 0 {
		return BodyResult{Body: body, Outcome: OutcomeSkippedDisabled}
	}
	if len(body) > e.config.MaxBodyBytes {
		return BodyResult{Body: body, Outcome: OutcomeSkippedTooLarge}
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return BodyResult{Body: body, Outcome: OutcomeSkippedMalformed}
	}
	context := &walkContext{engine: e, masked: e.UnmaskText}
	for _, path := range paths {
		context.maskPath(payload, path)
	}
	if len(context.fields) == 0 {
		return BodyResult{Body: body, Outcome: OutcomeMasked}
	}
	rewritten, err := json.Marshal(payload)
	if err != nil {
		return BodyResult{Body: body, Outcome: OutcomeSkippedMalformed}
	}
	return BodyResult{Body: rewritten, Outcome: OutcomeMasked, Fields: context.fields, Masked: true}
}

// walkContext carries the rewrite state through one body.
type walkContext struct {
	engine *Engine
	// masked transforms one text value.
	masked  func(string) string
	fields  []string
	changed bool
}

// walkObject rewrites the text values of one JSON object. The object's own path
// decides whether a plain string value is text.
func (w *walkContext) walkObject(object map[string]any, path string) {
	isMessage := isMessageObject(object)
	for key, value := range object {
		childPath := joinPath(path, key)
		if _, skip := structuralFields[key]; skip {
			continue
		}
		if _, media := mediaFields[key]; media {
			// Media values are never free text; only their nested captions and
			// transcripts, if any, are worth walking.
			if _, isText := textFields[key]; !isText {
				w.walkMediaValue(value, childPath)
				continue
			}
		}
		if text, ok := value.(string); ok {
			if !carriesText(key, isMessage) {
				continue
			}
			if isOpaqueValue(text) {
				continue
			}
			rewritten := w.masked(text)
			if rewritten != text {
				object[key] = rewritten
				w.remember(childPath)
			}
			continue
		}
		w.walkValue(value, key, childPath)
	}
}

// walkMediaValue descends into a media descriptor, which is either one object
// or an array of them.
func (w *walkContext) walkMediaValue(value any, path string) {
	switch typed := value.(type) {
	case map[string]any:
		w.walkMediaObject(typed, path)
	case []any:
		for index, item := range typed {
			if nested, ok := item.(map[string]any); ok {
				w.walkMediaObject(nested, joinIndex(path, index))
			}
		}
	}
}

// walkMediaObject walks the meaningful text of a media descriptor: a URL or an
// inline payload is left alone, but a caption or transcript beside it is not.
func (w *walkContext) walkMediaObject(object map[string]any, path string) {
	for key, value := range object {
		childPath := joinPath(path, key)
		if _, payload := payloadKeys[key]; payload {
			continue
		}
		if _, skip := structuralFields[key]; skip {
			continue
		}
		if text, ok := value.(string); ok {
			if _, isText := textFields[key]; !isText {
				continue
			}
			if isOpaqueValue(text) {
				continue
			}
			rewritten := w.masked(text)
			if rewritten != text {
				object[key] = rewritten
				w.remember(childPath)
			}
			continue
		}
		w.walkValue(value, key, childPath)
	}
}

// walkValue descends into arrays and nested objects. Iteration order matters:
// a map key that names a text field is masked first, because only the caller
// knows whether the field holds prose (a caption) or a pointer (a URL), and the
// media walk that follows treats every remaining string as a pointer.
func (w *walkContext) walkValue(value any, key, path string) {
	switch typed := value.(type) {
	case map[string]any:
		w.walkObject(typed, path)
	case []any:
		for index, item := range typed {
			itemPath := joinIndex(path, index)
			if nested, ok := item.(map[string]any); ok {
				w.walkObject(nested, itemPath)
				continue
			}
			if text, ok := item.(string); ok {
				if !carriesText(key, false) || isOpaqueValue(text) {
					continue
				}
				rewritten := w.masked(text)
				if rewritten != text {
					typed[index] = rewritten
					w.remember(itemPath)
				}
				continue
			}
			w.walkValue(item, key, itemPath)
		}
	}
}

// maskPath applies the rewrite to one dotted path such as
// "messages[0].content", creating no values and ignoring paths that do not
// resolve. root is the decoded body, which is a map or a slice at the top
// level, so it is taken as any.
func (w *walkContext) maskPath(root any, path string) {
	segments := splitPath(path)
	if len(segments) == 0 {
		return
	}
	current := root
	for index, segment := range segments {
		last := index == len(segments)-1
		object, ok := current.(map[string]any)
		if !ok {
			return
		}
		value, exists := object[segment.key]
		if !exists {
			return
		}
		if !segment.indexed {
			if last {
				w.applyLeaf(object, segment.key, value, path)
				return
			}
			current = value
			continue
		}
		list, ok := value.([]any)
		if !ok || segment.index >= len(list) {
			return
		}
		if last {
			text, ok := list[segment.index].(string)
			if !ok || isOpaqueValue(text) {
				return
			}
			rewritten := w.masked(text)
			if rewritten != text {
				list[segment.index] = rewritten
				w.remember(path)
			}
			return
		}
		current = list[segment.index]
	}
}

// applyLeaf rewrites one string leaf, or walks a nested value when the leaf is
// an object or an array of blocks.
func (w *walkContext) applyLeaf(object map[string]any, key string, value any, path string) {
	if text, ok := value.(string); ok {
		if isOpaqueValue(text) {
			return
		}
		rewritten := w.masked(text)
		if rewritten != text {
			object[key] = rewritten
			w.remember(path)
		}
		return
	}
	w.walkValue(value, key, path)
}

func (w *walkContext) remember(path string) {
	w.changed = true
	for _, existing := range w.fields {
		if existing == path {
			return
		}
	}
	if len(w.fields) >= maxReportedFields {
		return
	}
	w.fields = append(w.fields, path)
}

// pathSegment is one step of a dotted JSON path: a key, optionally followed by
// an array index.
type pathSegment struct {
	key     string
	index   int
	indexed bool
}

// splitPath parses "messages[0].content" into its segments.
func splitPath(path string) []pathSegment {
	var segments []pathSegment
	for _, part := range strings.Split(path, ".") {
		if part == "" {
			continue
		}
		key := part
		index := 0
		indexed := false
		if bracket := strings.IndexByte(part, '['); bracket >= 0 && strings.HasSuffix(part, "]") {
			key = part[:bracket]
			digits := part[bracket+1 : len(part)-1]
			parsed := 0
			valid := digits != ""
			for _, character := range digits {
				if character < '0' || character > '9' {
					valid = false
					break
				}
				parsed = parsed*10 + int(character-'0')
			}
			if !valid {
				continue
			}
			index = parsed
			indexed = true
		}
		segments = append(segments, pathSegment{key: key, index: index, indexed: indexed})
	}
	return segments
}

func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func joinIndex(path string, index int) string {
	return path + "[" + itoa(index) + "]"
}

// itoa keeps the package free of the strconv import in the hot walk path.
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	position := len(digits)
	for value > 0 {
		position--
		digits[position] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[position:])
}

// carriesText reports whether a field holds prose that must be masked. Inside a
// message object every remaining string field other than the structural ones is
// treated as text, because providers keep inventing new prose fields; outside
// one, only the known text fields qualify.
func carriesText(key string, isMessage bool) bool {
	if _, ok := textFields[key]; ok {
		return true
	}
	return isMessage
}

// isMessageObject reports whether an object describes one chat message, which
// is what widens the field allowlist for its children.
func isMessageObject(object map[string]any) bool {
	_, hasRole := object["role"]
	_, hasContent := object["content"]
	return hasRole && hasContent
}

// isOpaqueValue reports whether a string cannot be prose: an inline data URI, a
// base64 blob, or a long unbroken token. Rewriting those would corrupt binary
// payloads and cost CPU for no privacy gain.
func isOpaqueValue(value string) bool {
	if strings.HasPrefix(value, "data:") {
		return true
	}
	if len(value) > 8192 && !strings.ContainsAny(value, " \n") {
		return true
	}
	return false
}
