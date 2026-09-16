package service

import "github.com/QuantumNous/new-api/model"

// PublicKeyLogEntry is one usage-log entry as reported to a key holder by the
// public key-check endpoint (specs/public-key-check/spec.md, "Key usage log
// entries exclude account identity and infrastructure fields").
//
// It is an explicit whitelist, not the stored row: model.Log carries the
// owning account id, its username, the requesting client's IP, the upstream
// channel, the token's stored id and name, and the raw metadata blob. None of
// those belong on a page that anyone holding the key can open.
type PublicKeyLogEntry struct {
	CreatedAt        int64  `json:"created_at"`
	Type             int    `json:"type"`
	ModelName        string `json:"model_name"`
	Quota            int    `json:"quota"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	UseTime          int    `json:"use_time"`
	IsStream         bool   `json:"is_stream"`
	Group            string `json:"group"`
	RequestId        string `json:"request_id"`
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
		entries = append(entries, PublicKeyLogEntry{
			CreatedAt:        log.CreatedAt,
			Type:             log.Type,
			ModelName:        log.ModelName,
			Quota:            log.Quota,
			PromptTokens:     log.PromptTokens,
			CompletionTokens: log.CompletionTokens,
			UseTime:          log.UseTime,
			IsStream:         log.IsStream,
			Group:            log.Group,
			RequestId:        log.RequestId,
		})
	}
	return entries
}
