package service

import (
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
)

// channelHealthCacheTTL bounds how often the log store is aggregated.
// Operators watching a degrading channel expect the numbers to move, so this
// is shorter than the rankings cache TTL (see design.md — Decisions).
const channelHealthCacheTTL = 60 * time.Second

type channelHealthCacheItem struct {
	expiresAt time.Time
	data      []model.ChannelHealthStat
}

var (
	channelHealthCacheMu sync.Mutex
	channelHealthCache   *channelHealthCacheItem
)

// GetChannelHealthSnapshot returns the cached channel health aggregation,
// recomputing it when the cache is empty or has expired. The endpoint takes
// no parameters, so a single cache entry suffices.
func GetChannelHealthSnapshot() ([]model.ChannelHealthStat, error) {
	now := time.Now()

	channelHealthCacheMu.Lock()
	if channelHealthCache != nil && now.Before(channelHealthCache.expiresAt) {
		data := channelHealthCache.data
		channelHealthCacheMu.Unlock()
		return data, nil
	}
	channelHealthCacheMu.Unlock()

	data, err := model.GetChannelHealthStats()
	if err != nil {
		return nil, err
	}

	channelHealthCacheMu.Lock()
	channelHealthCache = &channelHealthCacheItem{
		expiresAt: now.Add(channelHealthCacheTTL),
		data:      data,
	}
	channelHealthCacheMu.Unlock()

	return data, nil
}
