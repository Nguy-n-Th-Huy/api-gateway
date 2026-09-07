package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
)

// TelegramBotUsageSummary is the aggregate the Telegram bot integration's
// usage-statistics endpoint reports for a linked account over a requested
// time range (specs/telegram/bot-api/spec.md, "Usage statistics"). It draws
// on the same logs table and the same LogTypeConsume filter as the console's
// own aggregates, so the two can never disagree.
type TelegramBotUsageSummary struct {
	Quota            int `json:"quota"`
	RequestCount     int `json:"request_count"`
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// GetUserUsageSummary aggregates one account's consume-type log entries over
// an optional time range (0 means unbounded on that side).
func GetUserUsageSummary(userId int, startTimestamp int64, endTimestamp int64) (TelegramBotUsageSummary, error) {
	var summary TelegramBotUsageSummary
	tx := LOG_DB.Table("logs").
		Select("COALESCE(sum(quota), 0) quota, count(*) request_count, "+
			"COALESCE(sum(prompt_tokens), 0) prompt_tokens, COALESCE(sum(completion_tokens), 0) completion_tokens").
		Where("user_id = ? AND type = ?", userId, LogTypeConsume)
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if err := tx.Scan(&summary).Error; err != nil {
		common.SysError("failed to query telegram bot usage summary: " + err.Error())
		return summary, errors.New("查询统计数据失败")
	}
	return summary, nil
}

// GetLogsByTokenIdPaginated is the paginated counterpart of GetLogByTokenId,
// used by the Telegram bot integration's key-scoped usage log endpoint
// (specs/telegram/bot-api/spec.md, "Key-scoped usage log").
func GetLogsByTokenIdPaginated(tokenId int, startIdx int, num int) (logs []*Log, total int64, err error) {
	if err = LOG_DB.Model(&Log{}).Where("token_id = ?", tokenId).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "id desc"
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		order = clickHouseLogOrder("")
	}
	if err = LOG_DB.Model(&Log{}).Where("token_id = ?", tokenId).Order(order).
		Limit(num).Offset(startIdx).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	formatUserLogs(logs, startIdx)
	return logs, total, nil
}
