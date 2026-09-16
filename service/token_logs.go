package service

import (
	"math"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// PublicKeyLogEntry is one usage-log entry as reported to a key holder by the
// public key-check endpoint (specs/public-key-check/spec.md, "Key usage log
// entries exclude account identity and infrastructure fields").
//
// It is an explicit whitelist, not the stored row: model.Log carries the
// owning account id, its username, the requesting client's IP, the upstream
// channel, the token's stored id and name, and the raw metadata blob. None of
// those belong on a page that anyone holding the key can open. The three
// fields the page shows that live inside that metadata — the endpoint, the
// cache-read and the cache-write token counts — are lifted out individually,
// so a key added by a future log writer cannot reach this surface by accident.
type PublicKeyLogEntry struct {
	CreatedAt           int64  `json:"created_at"`
	Type                int    `json:"type"`
	ModelName           string `json:"model_name"`
	Quota               int    `json:"quota"`
	PromptTokens        int    `json:"prompt_tokens"`
	CompletionTokens    int    `json:"completion_tokens"`
	CacheTokens         int    `json:"cache_tokens"`
	CacheCreationTokens int    `json:"cache_creation_tokens"`
	RequestPath         string `json:"request_path"`
	UseTime             int    `json:"use_time"`
	IsStream            bool   `json:"is_stream"`
	Group               string `json:"group"`
	RequestId           string `json:"request_id"`
}

// publicLogMetadata is the part of a log's user-visible metadata the page
// shows. Every field is optional, and a blob that cannot be read yields the
// zero value, so a single corrupt row never costs the caller its entry.
type publicLogMetadata struct {
	RequestPath         string   `json:"request_path"`
	CacheTokens         *float64 `json:"cache_tokens"`
	CacheCreationTokens *float64 `json:"cache_creation_tokens"`
}

func parsePublicLogMetadata(raw string) publicLogMetadata {
	if raw == "" {
		return publicLogMetadata{}
	}
	var metadata publicLogMetadata
	if err := common.UnmarshalJsonStr(raw, &metadata); err != nil {
		return publicLogMetadata{}
	}
	return metadata
}

// clampTokenCount keeps an upstream-reported count inside the range the page
// can display: never negative, never NaN, never beyond the 32-bit boundary.
func clampTokenCount(value *float64) int {
	if value == nil || math.IsNaN(*value) || *value <= 0 {
		return 0
	}
	if *value >= math.MaxInt32 {
		return math.MaxInt32
	}
	return int(*value)
}

// BuildPublicKeyLogEntries is the single producer of the public entry shape.
// Every surface that reports a key's log entries maps its rows through here,
// so the field set cannot drift and no identity or channel field can be added
// by accident.
func BuildPublicKeyLogEntries(logs []*model.Log) []PublicKeyLogEntry {
	entries := make([]PublicKeyLogEntry, 0, len(logs))
	for _, log := range logs {
		if log == nil {
			continue
		}
		metadata := parsePublicLogMetadata(log.Other)
		entries = append(entries, PublicKeyLogEntry{
			CreatedAt:           log.CreatedAt,
			Type:                log.Type,
			ModelName:           log.ModelName,
			Quota:               log.Quota,
			PromptTokens:        log.PromptTokens,
			CompletionTokens:    log.CompletionTokens,
			CacheTokens:         clampTokenCount(metadata.CacheTokens),
			CacheCreationTokens: clampTokenCount(metadata.CacheCreationTokens),
			RequestPath:         metadata.RequestPath,
			UseTime:             log.UseTime,
			IsStream:            log.IsStream,
			Group:               log.Group,
			RequestId:           log.RequestId,
		})
	}
	return entries
}
