package service

import (
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/pkg/piiguard"
	"github.com/QuantumNous/new-api/setting/piiguard_setting"

	"github.com/gin-gonic/gin"
)

// PIIFilter is the PII guard state of one relay request: the configuration it
// runs under, the engine holding the request-scoped real-to-fake mapping, and
// the JSON field paths that carried masked text.
//
// The mapping is sensitive by definition — it is the only thing that can turn a
// fake back into the real value — so it is never persisted and never written to
// a log.
type PIIFilter struct {
	config *piiguard.Config
	engine *piiguard.Engine
	fields []string
}

// NewPIIFilter returns the guard for the current settings, or nil when the
// guard is off, so the relay path stays free of the extra bookkeeping.
func NewPIIFilter() *PIIFilter {
	settings := piiguard_setting.GetSettings()
	if !settings.Active() {
		return nil
	}
	return &PIIFilter{
		config: &settings,
		engine: piiguard.NewEngine(settings),
	}
}

// MaskOutboundBody replaces every detected PII value in an outbound request
// body. It is a no-op outside the text relay formats, where the body is not the
// conversation itself.
func (f *PIIFilter) MaskOutboundBody(c *gin.Context, body []byte) ([]byte, error) {
	if f == nil || f.engine == nil || len(body) == 0 {
		return body, nil
	}
	result := f.engine.MaskOutboundBody(body)
	f.recordFields(result.Fields)
	switch result.Outcome {
	case piiguard.OutcomeSkippedTooLarge:
		if f.config.RequireMaskReject {
			return nil, errPIIBodyTooLarge
		}
		logger.LogWarn(c, "piiguard: outbound body exceeds the mask limit; it was forwarded unmasked")
	case piiguard.OutcomeSkippedMalformed:
		if f.config.RequireMaskReject {
			return nil, errPIIBodyUnparsable
		}
		logger.LogWarn(c, "piiguard: outbound body is not a JSON object; it was forwarded unmasked")
	}
	if result.Masked {
		if summary := f.summary(result.Stats); summary != "" {
			logger.LogInfo(c, "piiguard masked "+summary)
		}
	}
	return result.Body, nil
}

// NewUnmaskReader restores real values in the upstream response. It returns the
// source reader unchanged when there is nothing to restore.
func (f *PIIFilter) NewUnmaskReader(source io.Reader) io.Reader {
	if f == nil || f.engine == nil || source == nil || !f.config.UnmaskResponse {
		return source
	}
	return f.engine.NewUnmaskReader(source)
}

// MaskedFieldPaths returns the recorded field paths, for the consume log's
// admin info.
func (f *PIIFilter) MaskedFieldPaths() []string {
	if f == nil {
		return nil
	}
	return append([]string(nil), f.fields...)
}

// MaskedSpanCount returns how many values were replaced for this request.
func (f *PIIFilter) MaskedSpanCount() int {
	if f == nil || f.engine == nil {
		return 0
	}
	return f.engine.Stats().Spans
}

// MaskedEntityCounts returns the per-entity-type counts of this request, for
// the consume log's admin info. The counts are safe to persist; the values they
// replaced are not.
func (f *PIIFilter) MaskedEntityCounts() map[string]int {
	if f == nil || f.engine == nil {
		return nil
	}
	stats := f.engine.Stats()
	if len(stats.ByEntityType) == 0 {
		return nil
	}
	counts := make(map[string]int, len(stats.ByEntityType))
	for entityType, count := range stats.ByEntityType {
		counts[entityType] = count
	}
	return counts
}

func (f *PIIFilter) recordFields(fields []string) {
	if f == nil || len(fields) == 0 {
		return
	}
	for _, field := range fields {
		known := false
		for _, existing := range f.fields {
			if existing == field {
				known = true
				break
			}
		}
		if !known {
			f.fields = append(f.fields, field)
		}
	}
}

// summary renders the per-entity counts of one mask pass. It reports types and
// counts only, never a matched value.
func (f *PIIFilter) summary(stats piiguard.Stats) string {
	if stats.Spans == 0 {
		return ""
	}
	parts := make([]string, 0, len(stats.ByEntityType))
	for _, entityType := range sortedEntityTypes(stats.ByEntityType) {
		parts = append(parts, entityType+"="+strconv.Itoa(stats.ByEntityType[entityType]))
	}
	return strconv.Itoa(stats.Spans) + " value(s) across " + strings.Join(parts, ", ")
}

// sortedEntityTypes orders entity names so the log line is stable.
func sortedEntityTypes(counts map[string]int) []string {
	types := make([]string, 0, len(counts))
	for entityType := range counts {
		types = append(types, entityType)
	}
	sort.Strings(types)
	return types
}
