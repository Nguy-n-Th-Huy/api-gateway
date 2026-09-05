package model

import "time"

// channelHealthWindow is the fixed trailing window over which channel health
// metrics are derived. Not user-configurable (see design.md — Non-Goals).
const channelHealthWindow = 24 * time.Hour

// ChannelHealthStat is the per-channel health summary derived from request
// logs over the trailing channelHealthWindow. SuccessRate is a ratio in
// [0, 1]; percentage formatting is a frontend concern. AvgLatencyMs is a
// lower bound because logs.use_time is recorded with one-second granularity
// (see design.md — Known measurement limitation).
type ChannelHealthStat struct {
	ChannelId       int     `json:"channel_id"`
	TotalRequests   int     `json:"total_requests"`
	SuccessRequests int     `json:"success_requests"`
	FailedRequests  int     `json:"failed_requests"`
	SuccessRate     float64 `json:"success_rate"`
	AvgLatencyMs    int     `json:"avg_latency_ms"`
}

// channelHealthAggRow is the raw scan target for the standalone LOG_DB
// aggregation. It intentionally holds only summed integers so the query
// stays valid across SQLite, MySQL, PostgreSQL, and ClickHouse — no AVG(),
// no join to channels (see design.md — Decisions).
type channelHealthAggRow struct {
	ChannelId           int   `gorm:"column:channel_id"`
	TotalRequests       int   `gorm:"column:total_requests"`
	FailedRequests      int   `gorm:"column:failed_requests"`
	TotalUseTimeSeconds int64 `gorm:"column:total_use_time_seconds"`
}

// GetChannelHealthStats aggregates logs on LOG_DB, standalone (no join to
// channels), over the trailing 24h window. Only consume and error log types
// are counted; entries with a non-positive channel_id are excluded. A
// channel with no matching rows in the window is simply absent from the
// result, which is the caller's signal to render "no data" rather than 0%.
func GetChannelHealthStats() ([]ChannelHealthStat, error) {
	windowStart := time.Now().Add(-channelHealthWindow).Unix()

	var rows []channelHealthAggRow
	err := LOG_DB.Table("logs").
		Select("channel_id, COUNT(*) AS total_requests, SUM(CASE WHEN type = ? THEN 1 ELSE 0 END) AS failed_requests, SUM(use_time) AS total_use_time_seconds", LogTypeError).
		Where("created_at >= ?", windowStart).
		Where("type IN (?, ?)", LogTypeConsume, LogTypeError).
		Where("channel_id > 0").
		Group("channel_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	stats := make([]ChannelHealthStat, 0, len(rows))
	for _, row := range rows {
		stats = append(stats, channelHealthStatFromRow(row))
	}
	return stats, nil
}

// channelHealthStatFromRow converts a raw aggregation row into the reported
// stat. Pure and DB-free so it is directly unit-testable.
func channelHealthStatFromRow(row channelHealthAggRow) ChannelHealthStat {
	stat := ChannelHealthStat{
		ChannelId:      row.ChannelId,
		TotalRequests:  row.TotalRequests,
		FailedRequests: row.FailedRequests,
	}
	stat.SuccessRequests = stat.TotalRequests - stat.FailedRequests
	if stat.TotalRequests <= 0 {
		return stat
	}
	stat.SuccessRate = float64(stat.SuccessRequests) / float64(stat.TotalRequests)
	stat.AvgLatencyMs = int(row.TotalUseTimeSeconds * 1000 / int64(stat.TotalRequests))
	return stat
}
