package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// seedChannelHealthLog inserts one log row for the channel health aggregation
// test below, via the shared in-memory SQLite fixture set up by TestMain.
func seedChannelHealthLog(t *testing.T, channelID int, logType int, createdAt int64, useTime int) {
	t.Helper()
	require.NoError(t, DB.Create(&Log{
		ChannelId: channelID,
		Type:      logType,
		CreatedAt: createdAt,
		UseTime:   useTime,
	}).Error)
}

// TestGetChannelHealthStatsAggregatesOverSQLite exercises the standalone
// LOG_DB aggregation query (COUNT(*), SUM(CASE WHEN ...), SUM(use_time); no
// join to channels, no AVG()) against a real SQLite database, proving the
// SQL is valid on this dialect and that the window, log-type, and
// non-positive-channel-id filters behave per spec.md.
func TestGetChannelHealthStatsAggregatesOverSQLite(t *testing.T) {
	truncateTables(t)
	now := time.Now().Unix()

	// Channel 501: 8 consume + 2 error within the window (spec.md — mixed
	// successes and failures scenario: total 10, success 8, failed 2, rate 0.8).
	for i := 0; i < 8; i++ {
		seedChannelHealthLog(t, 501, LogTypeConsume, now-10, 2)
	}
	for i := 0; i < 2; i++ {
		seedChannelHealthLog(t, 501, LogTypeError, now-10, 1)
	}

	// Channel 502: only failures within the window (spec.md — only-failures scenario).
	for i := 0; i < 5; i++ {
		seedChannelHealthLog(t, 502, LogTypeError, now-10, 3)
	}

	// Channel 503: only a log older than the 24h window -> excluded entirely.
	seedChannelHealthLog(t, 503, LogTypeConsume, now-25*3600, 5)

	// Channel 504: only non-consume/error log types within the window -> excluded.
	seedChannelHealthLog(t, 504, LogTypeTopup, now-10, 5)
	seedChannelHealthLog(t, 504, LogTypeManage, now-10, 5)
	seedChannelHealthLog(t, 504, LogTypeSystem, now-10, 5)
	seedChannelHealthLog(t, 504, LogTypeRefund, now-10, 5)
	seedChannelHealthLog(t, 504, LogTypeLogin, now-10, 5)

	// A log with a non-positive channel id must never surface as a channel.
	seedChannelHealthLog(t, 0, LogTypeConsume, now-10, 5)

	stats, err := GetChannelHealthStats()
	require.NoError(t, err)

	byChannel := make(map[int]ChannelHealthStat, len(stats))
	for _, stat := range stats {
		byChannel[stat.ChannelId] = stat
	}

	stat501, ok := byChannel[501]
	require.True(t, ok, "channel 501 must be present")
	require.Equal(t, 10, stat501.TotalRequests)
	require.Equal(t, 8, stat501.SuccessRequests)
	require.Equal(t, 2, stat501.FailedRequests)
	require.InDelta(t, 0.8, stat501.SuccessRate, 1e-9)
	require.Equal(t, 1800, stat501.AvgLatencyMs) // (8*2 + 2*1) = 18s / 10 * 1000

	stat502, ok := byChannel[502]
	require.True(t, ok, "channel 502 must be present")
	require.Equal(t, 5, stat502.TotalRequests)
	require.Equal(t, 0, stat502.SuccessRequests)
	require.Equal(t, 5, stat502.FailedRequests)
	require.Equal(t, float64(0), stat502.SuccessRate)

	_, ok = byChannel[503]
	require.False(t, ok, "a channel with only logs older than the window must be absent")

	_, ok = byChannel[504]
	require.False(t, ok, "a channel with only non-consume/error log types must be absent")

	_, ok = byChannel[0]
	require.False(t, ok, "logs with a non-positive channel id must never surface as a channel")
}
