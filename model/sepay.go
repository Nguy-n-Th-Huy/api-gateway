package model

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	// SePayMemoPrefixTopUp marks a bank-safe transfer memo as belonging to a
	// wallet top-up (table top_ups). SePayMemoPrefixSubscription marks one
	// belonging to a subscription order (table subscription_orders). The
	// webhook routes by this prefix so at most one order type is ever consulted
	// per candidate (D3).
	SePayMemoPrefixTopUp        = "SP"
	SePayMemoPrefixSubscription = "SS"
)

const (
	sePayMemoRandomLen       = 14
	sePayTradeNoInsertRetries = 5
	sePayOrderExpiryFallbackMinutes = 30
)

// sePayMemoAlphabet is exactly uppercase ASCII letters + digits. Banks strip
// punctuation, diacritics, and sometimes lowercase from the transfer
// description, so the memo must survive transit as only these characters (D1).
const sePayMemoAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomSePayMemoSegment(length int) string {
	alphabetLen := big.NewInt(int64(len(sePayMemoAlphabet)))
	out := make([]byte, length)
	for i := range out {
		idx, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			// Crypto rand failure is vanishingly rare; fall back to '0' so the
			// generated memo stays well-formed and retryable.
			out[i] = '0'
			continue
		}
		out[i] = sePayMemoAlphabet[idx.Int64()]
	}
	return string(out)
}

// GenerateSePayTradeNo returns a bank-safe trade number that doubles as the
// transfer memo. prefix selects the order table (SP top-up, SS subscription).
// The trade_no's unique constraint is what enforces memo uniqueness; the
// callers that persist orders retry on the (astronomically unlikely) insert
// conflict (D1).
func GenerateSePayTradeNo(prefix string) string {
	return prefix + randomSePayMemoSegment(sePayMemoRandomLen)
}

// InsertSePayTopUp generates a unique trade number for topUp, inserts it, and
// retries on the (astronomically unlikely) unique-constraint collision so no
// two orders can ever share a memo (D1).
func InsertSePayTopUp(topUp *TopUp) error {
	if topUp == nil {
		return errors.New("top-up is nil")
	}
	for attempt := 0; attempt < sePayTradeNoInsertRetries; attempt++ {
		topUp.TradeNo = GenerateSePayTradeNo(SePayMemoPrefixTopUp)
		if err := DB.Create(topUp).Error; err == nil {
			return nil
		} else if isDuplicateTradeNoError(err) {
			common.SysError(fmt.Sprintf("SePay trade_no collision trade_no=%s attempt=%d", topUp.TradeNo, attempt))
			continue
		} else {
			return err
		}
	}
	return errors.New("创建订单失败，请稍后重试")
}

// InsertSePaySubscriptionOrder is the subscription-order counterpart of
// InsertSePayTopUp.
func InsertSePaySubscriptionOrder(order *SubscriptionOrder) error {
	if order == nil {
		return errors.New("subscription order is nil")
	}
	for attempt := 0; attempt < sePayTradeNoInsertRetries; attempt++ {
		order.TradeNo = GenerateSePayTradeNo(SePayMemoPrefixSubscription)
		if err := DB.Create(order).Error; err == nil {
			return nil
		} else if isDuplicateTradeNoError(err) {
			common.SysError(fmt.Sprintf("SePay subscription trade_no collision trade_no=%s attempt=%d", order.TradeNo, attempt))
			continue
		} else {
			return err
		}
	}
	return errors.New("创建订单失败，请稍后重试")
}

func isDuplicateTradeNoError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	lower := strings.ToLower(err.Error())
	// Cover SQLite (glebarez/sqlite), MySQL, and PostgreSQL messages without
	// requiring GORM's TranslateError to be enabled (it is not in new-api):
	// - SQLite:       "UNIQUE constraint failed: ...trade_no"
	// - MySQL:        "Error 1062 (23000): Duplicate entry '...' for key 'trade_no'"
	// - PostgreSQL:   "duplicate key value violates unique constraint"
	return strings.Contains(lower, "duplicate") || strings.Contains(lower, "unique constraint")
}

// ErrTopUpAmountInsufficient is returned by RechargeSePay when the reported
// SePay transfer is smaller than the order's frozen payable amount. The order
// stays pending (D8); the sweep will expire it normally.
var ErrTopUpAmountInsufficient = errors.New("sepay transfer amount insufficient")

// RechargeSePay atomically completes a SePay wallet top-up when the matching
// bank transfer lands: the order row is locked, the provider and status are
// verified, the reported transferAmountVND is checked against the order's
// frozen payable VND, then the order is marked success and the wallet is
// credited in the same transaction (D5–D7). Idempotency is enforced against
// the DB (replayed or concurrent deliveries for the same order), so it holds
// across multiple application instances.
//
// Comparison of VND amounts is done on integers derived from decimal so the
// stored float64 money (whole Dong, D5) can be compared exactly to the
// integer transferAmount reported by SePay.
func RechargeSePay(tradeNo string, transferAmountVND int64, callerIp string) (alreadyDone bool, err error) {
	if tradeNo == "" {
		return false, errors.New("未提供支付单号")
	}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	var quotaToAdd int
	var payableVND int64
	var transferVND int64
	topUp := &TopUp{}
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if topUp.PaymentProvider != PaymentProviderSePay {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status == common.TopUpStatusSuccess {
			alreadyDone = true
			return nil
		}
		if topUp.Status != common.TopUpStatusPending {
			return ErrTopUpStatusInvalid
		}
		// SePay transfer must not be credited if the order is already past its
		// derived deadline, even before the sweep runs (D4).
		if isSePayOrderExpiredByCreateTime(topUp.CreateTime) {
			return ErrTopUpStatusInvalid
		}
		payableVND = sePayPayableVNDInt(topUp.Money)
		transferVND = transferAmountVND
		if transferAmountVND < payableVND {
			return ErrTopUpAmountInsufficient
		}

		var quotaErr error
		quotaToAdd, quotaErr = common.WalletQuotaFromDecimalStrict(
			decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)),
		)
		if quotaErr != nil || quotaToAdd <= 0 {
			return ErrInvalidTopUpQuota
		}
		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}
		return creditTopUpQuota(tx, topUp.UserId, quotaToAdd, nil)
	})
	if err != nil {
		if !errors.Is(err, ErrTopUpNotFound) && !errors.Is(err, ErrPaymentMethodMismatch) && !errors.Is(err, ErrTopUpStatusInvalid) && !errors.Is(err, ErrTopUpAmountInsufficient) {
			common.SysError("sepay topup failed: " + err.Error())
		}
		return false, err
	}
	if alreadyDone {
		return true, nil
	}
	syncCreditUserQuotaCache(topUp.UserId, quotaToAdd, "sepay topup")

	if transferVND > payableVND {
		common.SysError(fmt.Sprintf("SePay overpaid order trade_no=%s payable_vnd=%d transfer_vnd=%d surplus=%d", tradeNo, payableVND, transferVND, transferVND-payableVND))
	}
	common.SysLog(fmt.Sprintf("SePay充值成功 trade_no=%s user_id=%d quota_to_add=%d money=%.0f", topUp.TradeNo, topUp.UserId, quotaToAdd, topUp.Money))
	RecordTopupLog(topUp.UserId, fmt.Sprintf("使用在线充值成功，充值金额: %v，支付金额：%.0f", logger.LogQuota(quotaToAdd), topUp.Money), callerIp, PaymentMethodSePay, PaymentProviderSePay)
	return false, nil
}

// RechargeSePaySubscription is the SePay subscription-settlement counterpart
// of RechargeSePay. It verifies provider, pending status, derived expiry, and
// amount sufficiency, then delegates to CompleteSubscriptionOrder which itself
// holds the row lock and is idempotent (D5–D8, D3).
//
// The pre-checks here are advisory — the settlement inside
// CompleteSubscriptionOrder remains the source of truth for idempotency
// across concurrent deliveries. A transfer for an expired-but-not-yet-swept
// order is rejected here even though the sweep hasn't marked it yet.
func RechargeSePaySubscription(tradeNo string, transferAmountVND int64) (alreadyDone bool, err error) {
	if tradeNo == "" {
		return false, errors.New("未提供支付单号")
	}
	order := GetSubscriptionOrderByTradeNo(tradeNo)
	if order == nil {
		return false, ErrSubscriptionOrderNotFound
	}
	if order.PaymentProvider != PaymentProviderSePay {
		return false, ErrPaymentMethodMismatch
	}
	if order.Status == common.TopUpStatusSuccess {
		return true, nil
	}
	if order.Status != common.TopUpStatusPending {
		return false, ErrSubscriptionOrderStatusInvalid
	}
	if isSePayOrderExpiredByCreateTime(order.CreateTime) {
		return false, ErrSubscriptionOrderStatusInvalid
	}
	payableVND := sePayPayableVNDInt(order.Money)
	if transferAmountVND < payableVND {
		return false, ErrTopUpAmountInsufficient
	}
	if transferAmountVND > payableVND {
		common.SysError(fmt.Sprintf("SePay overpaid subscription order trade_no=%s payable_vnd=%d transfer_vnd=%d surplus=%d", tradeNo, payableVND, transferAmountVND, transferAmountVND-payableVND))
	}
	if err := CompleteSubscriptionOrder(tradeNo, "", PaymentProviderSePay, PaymentMethodSePay); err != nil {
		return false, err
	}
	return false, nil
}

// sePayPayableVNDInt coerces the stored float64 Money (whole Dong, D5) to
// int64 so it can be compared exactly against the integer transferAmount
// reported by SePay. Money is always whole Dong for SePay orders.
func sePayPayableVNDInt(money float64) int64 {
	return decimal.NewFromFloat(money).Round(0).IntPart()
}

// isSePayOrderExpiredByCreateTime reports whether a pending SePay order whose
// create_time is the given Unix timestamp is already past its derived deadline
// (create_time + configured window). Used by the webhook path so a transfer
// arriving one second after expiry is never credited even if the sweep has not
// run yet (D4).
func isSePayOrderExpiredByCreateTime(createTime int64) bool {
	return common.GetTimestamp() >= sePayOrderExpiryUnix(createTime)
}

// sePayOrderExpiryUnix returns the derived expiry instant (Unix seconds) for a
// pending SePay order whose create_time is the given Unix timestamp.
func sePayOrderExpiryUnix(createTime int64) int64 {
	return createTime + int64(sePayExpiryWindowMinutes())*60
}

func sePayExpiryWindowMinutes() int {
	minutes := setting.SePayOrderExpiryMinutes
	if minutes <= 0 {
		return sePayOrderExpiryFallbackMinutes
	}
	return minutes
}

// SePayOrderExpiryUnix is exported so the payment controller can surface the
// derived deadline on the create response and on the order-status query.
func SePayOrderExpiryUnix(createTime int64) int64 {
	return sePayOrderExpiryUnix(createTime)
}

// ExpireSePayTopUpsBulk marks pending SePay top-up orders as expired in bulk
// when their derived deadline (create_time + configured expiry window) is in
// the past. It deliberately scopes to payment_provider = SePay so legacy
// pending orders from removed gateways stay visible for manual settlement
// (D4, legacy-gateway-removal). Returns the number of rows updated.
// The caller may pass limit <= 0 for an unbounded run.
func ExpireSePayTopUpsBulk(limit int) (int64, error) {
	cutoff := common.GetTimestamp() - int64(sePayExpiryWindowMinutes())*60
	return expireSePayRowsByProvider(&TopUp{}, limit, cutoff)
}

// ExpireSePaySubscriptionOrdersBulk is the subscription-order counterpart of
// ExpireSePayTopUpsBulk. It expires only SePay subscription orders.
func ExpireSePaySubscriptionOrdersBulk(limit int) (int64, error) {
	cutoff := common.GetTimestamp() - int64(sePayExpiryWindowMinutes())*60
	return expireSePayRowsByProvider(&SubscriptionOrder{}, limit, cutoff)
}

// expireSePayRowsByProvider runs the bulk expiry UPDATE for either the TopUp
// or SubscriptionOrder table (both share the status/payment_provider/
// create_time/complete_time columns that this sweep touches). Cross-DB
// compatible: PostgreSQL and SQLite do not support LIMIT on UPDATE, so when
// limit > 0 we first Pluck matching ids and then update by primary-key set.
func expireSePayRowsByProvider(model interface{}, limit int, cutoff int64) (int64, error) {
	now := common.GetTimestamp()
	updates := map[string]interface{}{
		"status":        common.TopUpStatusExpired,
		"complete_time": now,
	}
	whereSQL := "status = ? AND payment_provider = ? AND create_time > 0 AND create_time <= ?"
	whereArgs := []interface{}{common.TopUpStatusPending, PaymentProviderSePay, cutoff}
	if limit > 0 {
		var ids []int
		if err := DB.Model(model).
			Select("id").
			Where(whereSQL, whereArgs...).
			Order("id asc").
			Limit(limit).
			Pluck("id", &ids).Error; err != nil {
			return 0, err
		}
		if len(ids) == 0 {
			return 0, nil
		}
		res := DB.Model(model).Where("id IN ?", ids).Updates(updates)
		return res.RowsAffected, res.Error
	}
	res := DB.Model(model).Where(whereSQL, whereArgs...).Updates(updates)
	return res.RowsAffected, res.Error
}
