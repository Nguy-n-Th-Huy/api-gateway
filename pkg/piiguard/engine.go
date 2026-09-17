package piiguard

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"sort"
	"strings"
	"sync"
)

// Engine masks outbound text and reverses the substitutions in the response.
//
// One Engine belongs to one relay request. The mapping it builds is
// request-scoped on purpose: the reference implementation treats the mapping as
// sensitive data, so it is never written to disk, never logged as payload, and
// dropped together with the request.
type Engine struct {
	config    Config
	detectors []detector
	seed      string

	mu           sync.Mutex
	replacements map[string][]Mapping
	stats        Stats
}

// Mapping is one real value and the fake that replaced it.
type Mapping struct {
	EntityType string
	Real       string
	Fake       string
}

// Stats reports what the engine did for one request.
type Stats struct {
	Spans        int
	ByEntityType map[string]int
}

// NewEngine builds an engine for one request.
func NewEngine(config Config) *Engine {
	config = config.Normalize()
	seed := config.Secret
	if seed == "" {
		seed = randomSeed()
	}
	return &Engine{
		config:       config,
		detectors:    defaultDetectors(),
		seed:         seed,
		replacements: make(map[string][]Mapping),
	}
}

// Config returns the normalized configuration the engine runs with.
func (e *Engine) Config() Config {
	return e.config
}

// Stats returns the detection counters gathered so far.
func (e *Engine) Stats() Stats {
	e.mu.Lock()
	defer e.mu.Unlock()
	counts := make(map[string]int, len(e.stats.ByEntityType))
	for entityType, count := range e.stats.ByEntityType {
		counts[entityType] = count
	}
	return Stats{Spans: e.stats.Spans, ByEntityType: counts}
}

func (e *Engine) record(entityType string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stats.Spans++
	if e.stats.ByEntityType == nil {
		e.stats.ByEntityType = make(map[string]int)
	}
	e.stats.ByEntityType[entityType]++
}

// MaskText replaces every detected value in text with its replacement. It is
// exported so a caller can mask a single string without a request body.
func (e *Engine) MaskText(text string) string {
	spans := e.detect(text)
	if len(spans) == 0 {
		return text
	}
	var builder strings.Builder
	builder.Grow(len(text))
	cursor := 0
	for _, found := range spans {
		if found.start < cursor {
			continue
		}
		builder.WriteString(text[cursor:found.start])
		builder.WriteString(e.replacementFor(found.kind, text[found.start:found.end]))
		cursor = found.end
		e.record(found.kind)
	}
	builder.WriteString(text[cursor:])
	return builder.String()
}

// UnmaskText restores every fake value that appears in text.
func (e *Engine) UnmaskText(text string) string {
	table := e.UnmaskTable()
	if len(table) == 0 {
		return text
	}
	return applyUnmaskTable(text, table)
}

// UnmaskTable returns the real value behind every fake. Entries whose fake
// equals the real value are dropped, because reversing them is a no-op.
func (e *Engine) UnmaskTable() map[string]string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.replacements) == 0 {
		return nil
	}
	table := make(map[string]string, len(e.replacements))
	for _, replacements := range e.replacements {
		for _, replacement := range replacements {
			if replacement.Fake == replacement.Real {
				continue
			}
			table[replacement.Fake] = replacement.Real
		}
	}
	return table
}

// Mappings returns every substitution this request performed, for an audit
// trail. The package itself never writes them to a log.
func (e *Engine) Mappings() []Mapping {
	e.mu.Lock()
	defer e.mu.Unlock()
	mappings := make([]Mapping, 0, len(e.replacements))
	for _, replacements := range e.replacements {
		mappings = append(mappings, replacements...)
	}
	sort.Slice(mappings, func(left, right int) bool {
		if mappings[left].EntityType != mappings[right].EntityType {
			return mappings[left].EntityType < mappings[right].EntityType
		}
		return mappings[left].Real < mappings[right].Real
	})
	return mappings
}

// MaxReplacementLength returns the length of the longest replacement produced
// so far, which is how far the streaming unmask has to hold bytes back.
func (e *Engine) MaxReplacementLength() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	longest := 0
	for _, replacements := range e.replacements {
		for _, replacement := range replacements {
			if len(replacement.Fake) > longest {
				longest = len(replacement.Fake)
			}
		}
	}
	return longest
}

// NewUnmaskReader wraps an upstream response body so fakes are restored to the
// real values while the bytes stream to the client. A fake that is split across
// two SSE chunks is held back until it is complete.
func (e *Engine) NewUnmaskReader(source io.Reader) io.Reader {
	table := e.UnmaskTable()
	if len(table) == 0 {
		return source
	}
	return newUnmaskReader(source, table, e.MaxReplacementLength())
}

// replacementFor returns the replacement of one real value, creating it on
// first use so the mapping stays bijective across the whole request.
func (e *Engine) replacementFor(entityType, real string) string {
	e.mu.Lock()
	for _, existing := range e.replacements[entityType] {
		if existing.Real == real {
			fake := existing.Fake
			e.mu.Unlock()
			return fake
		}
	}
	fake := e.computeReplacement(entityType, real, len(e.replacements[entityType])+1)
	e.replacements[entityType] = append(e.replacements[entityType], Mapping{
		EntityType: entityType,
		Real:       real,
		Fake:       fake,
	})
	e.mu.Unlock()
	return fake
}

// computeReplacement decides between an operator supplied value, the redaction
// placeholder, and the deterministic generator. ordinal is the position of the
// value within its entity type, which is what keeps redaction placeholders
// distinct from each other.
func (e *Engine) computeReplacement(entityType, real string, ordinal int) string {
	if override, ok := e.config.Fakes[entityType][real]; ok && override != "" {
		return override
	}
	if e.config.Mode == ModeRedact {
		return redactionPlaceholder(e.config, entityType, ordinal)
	}
	return generate(e.config, entityType, real, hashValue(e.seed, entityType, real))
}

// detect returns the non-overlapping spans selected for one text, sorted by
// position. Overlap resolution is longest-match-first with the detector
// priority as the tie-break, which is the rule that keeps an email address from
// being split into a URL and its local part.
func (e *Engine) detect(text string) []span {
	var candidates []span
	for _, detector := range e.detectors {
		if !e.config.entityEnabled(detector.entityType()) {
			continue
		}
		for _, candidate := range detector.match(text) {
			if len(candidate.requiredContext) > 0 && !hasContextKeyword(text, candidate.start, candidate.requiredContext) {
				continue
			}
			candidates = append(candidates, candidate)
		}
	}
	candidates = append(candidates, e.customKeywordSpans(text)...)
	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(left, right int) bool {
		if candidates[left].start != candidates[right].start {
			return candidates[left].start < candidates[right].start
		}
		leftLength := candidates[left].end - candidates[left].start
		rightLength := candidates[right].end - candidates[right].start
		if leftLength != rightLength {
			return leftLength > rightLength
		}
		return candidates[left].priority > candidates[right].priority
	})

	selected := make([]span, 0, len(candidates))
	cursor := 0
	for _, candidate := range candidates {
		if candidate.start < cursor {
			continue
		}
		selected = append(selected, candidate)
		cursor = candidate.end
	}
	return selected
}

// customKeywordSpans reports the admin-defined literals. They are matched
// case-insensitively but keep their real extent in the original text.
func (e *Engine) customKeywordSpans(text string) []span {
	keywords := e.config.CustomKeywords
	if len(keywords) == 0 {
		return nil
	}
	lower := strings.ToLower(text)
	var spans []span
	for _, keyword := range keywords {
		trimmed := strings.TrimSpace(keyword)
		if len(trimmed) < e.config.MinKeywordLength {
			continue
		}
		needle := strings.ToLower(trimmed)
		for offset := 0; ; {
			found := strings.Index(lower[offset:], needle)
			if found < 0 {
				break
			}
			start := offset + found
			spans = append(spans, span{
				start:    start,
				end:      start + len(needle),
				kind:     EntityCustom,
				priority: priorityHigh,
			})
			offset = start + len(needle)
		}
	}
	return spans
}

// redactionPlaceholder returns the placeholder of redaction mode for one value.
//
// Each distinct real value gets its own ordinal, which is what keeps the
// mapping bijective: two different values never share a token, so the model can
// still tell two entities apart, and the round trip stays exact.
func redactionPlaceholder(config Config, entityType string, ordinal int) string {
	if config.PlaceholderStyle == PlaceholderTemplate {
		return indexTemplate(config.TokenTemplate, entityType, ordinal)
	}
	return "[" + entityType + "_" + itoa(ordinal) + "]"
}

// randomSeed returns a per-process seed when the operator configured none.
// A random seed still keeps one request internally consistent, which is what
// round-trip unmasking requires.
func randomSeed() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "piiguard-fallback-seed"
	}
	return hex.EncodeToString(buffer)
}
