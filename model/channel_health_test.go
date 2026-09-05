package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChannelHealthStatFromRow(t *testing.T) {
	cases := []struct {
		name string
		row  channelHealthAggRow
		want ChannelHealthStat
	}{
		{
			name: "zero total requests guards divide-by-zero",
			row:  channelHealthAggRow{ChannelId: 1, TotalRequests: 0, FailedRequests: 0, TotalUseTimeSeconds: 0},
			want: ChannelHealthStat{ChannelId: 1, TotalRequests: 0, SuccessRequests: 0, FailedRequests: 0, SuccessRate: 0, AvgLatencyMs: 0},
		},
		{
			name: "all failures reports zero success rate",
			row:  channelHealthAggRow{ChannelId: 2, TotalRequests: 5, FailedRequests: 5, TotalUseTimeSeconds: 15},
			want: ChannelHealthStat{ChannelId: 2, TotalRequests: 5, SuccessRequests: 0, FailedRequests: 5, SuccessRate: 0, AvgLatencyMs: 3000},
		},
		{
			name: "all successes reports success rate of one",
			row:  channelHealthAggRow{ChannelId: 3, TotalRequests: 4, FailedRequests: 0, TotalUseTimeSeconds: 8},
			want: ChannelHealthStat{ChannelId: 3, TotalRequests: 4, SuccessRequests: 4, FailedRequests: 0, SuccessRate: 1, AvgLatencyMs: 2000},
		},
		{
			name: "mixed successes and failures matches the spec scenario",
			row:  channelHealthAggRow{ChannelId: 4, TotalRequests: 10, FailedRequests: 2, TotalUseTimeSeconds: 20},
			want: ChannelHealthStat{ChannelId: 4, TotalRequests: 10, SuccessRequests: 8, FailedRequests: 2, SuccessRate: 0.8, AvgLatencyMs: 2000},
		},
		{
			name: "sub-second durations recorded as zero yield zero average latency",
			row:  channelHealthAggRow{ChannelId: 5, TotalRequests: 3, FailedRequests: 0, TotalUseTimeSeconds: 0},
			want: ChannelHealthStat{ChannelId: 5, TotalRequests: 3, SuccessRequests: 3, FailedRequests: 0, SuccessRate: 1, AvgLatencyMs: 0},
		},
		{
			name: "durations of 1 2 3 2 seconds convert to 2000ms average",
			row:  channelHealthAggRow{ChannelId: 6, TotalRequests: 4, FailedRequests: 0, TotalUseTimeSeconds: 8},
			want: ChannelHealthStat{ChannelId: 6, TotalRequests: 4, SuccessRequests: 4, FailedRequests: 0, SuccessRate: 1, AvgLatencyMs: 2000},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := channelHealthStatFromRow(tc.row)
			assert.Equal(t, tc.want, got)
			assert.GreaterOrEqual(t, got.SuccessRate, 0.0, "success rate must never go negative")
			assert.LessOrEqual(t, got.SuccessRate, 1.0, "success rate must never exceed 1")
		})
	}
}
