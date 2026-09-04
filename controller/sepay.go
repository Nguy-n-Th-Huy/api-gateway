package controller

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

const (
	sePayMaxTopUpAmount = int64(9999)
	sePayVietQRTemplate = "compact2"
	sePayMemoRegexLen   = 16
)

// sePayMemoRegex matches the fixed memo shape (S[PS] + 14 uppercase
// alphanumerics, total 16 chars) inside free-form bank content after
// normalization. No word-boundary anchors: bank content can pack the memo
// against surrounding text (D2).
var sePayMemoRegex = regexp.MustCompile(`S[PS][A-Z0-9]{14}`)

// sePayWebhookPayload matches SePay's flat JSON webhook body.
// https://developer.sepay.vn/en/sepay-webhooks/tich-hop-webhook
type sePayWebhookPayload struct {
	Id              int     `json:"id"`
	Gateway         string  `json:"gateway"`
	TransactionDate string  `json:"transactionDate"`
	AccountNumber   string  `json:"accountNumber"`
	SubAccount      *string `json:"subAccount"`
	Code            *string `json:"code"`
	Content         string  `json:"content"`
	TransferType    string  `json:"transferType"`
	Description     string  `json:"description"`
	TransferAmount  int64   `json:"transferAmount"`
	Accumulated     *int64  `json:"accumulated"`
	ReferenceCode   string  `json:"referenceCode"`
}

type sePayTopUpRequest struct {
	Amount int64 `json:"amount"`
}

type sePaySubscriptionPayRequest struct {
	PlanId int `json:"plan_id"`
}

type sePayOrderResponse struct {
	TradeNo      string  `json:"trade_no"`
	Memo         string  `json:"memo"`
	PayableVND   int64   `json:"payable_vnd"`
	BankAccount  string  `json:"bank_account"`
	BankCode     string  `json:"bank_code"`
	AccountName  string  `json:"account_holder"`
	VietQRURL    string  `json:"vietqr_url"`
	CreateTime   int64   `json:"create_time"`
	ExpireTime   int64   `json:"expire_time"`
	Status       string  `json:"status,omitempty"`
	Money        float64 `json:"money"`
}

type sePayMatch struct {
	tradeNo string
	isSub   bool
}

type sePayMatchList []sePayMatch

// SePayRequestTopUp is POST /api/user/sepay/pay. Compliance gate, config
// gate, minimum-amount check, ValidateTopUpQuotaCapacity, decimal VND
// conversion via common/quota_math.go helpers, pending order insert with a
// unique trade_no, and a response carrying memo / payable VND / bank details
// / VietQR image URL / trade_no / expiry (task 3.1).
func SePayRequestTopUp(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	if !setting.IsSePayConfigured() {
		common.ApiErrorMsg(c, "SePay 未配置")
		return
	}

	var req sePayTopUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	minTopUp := sePayEffectiveMinTopUp()
	if req.Amount < int64(minTopUp) {
		common.ApiErrorMsg(c, fmt.Sprintf("充值数量不能小于 %d", minTopUp))
		return
	}
	if req.Amount > sePayMaxTopUpAmount {
		common.ApiErrorMsg(c, fmt.Sprintf("单笔充值数量不能大于 %d", sePayMaxTopUpAmount))
		return
	}

	userId := c.GetInt("id")
	creditedQuota, err := validateTopUpQuota(req.Amount)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if err := model.ValidateTopUpQuotaCapacity(userId, creditedQuota); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	group, err := model.GetUserGroup(userId, true)
	if err != nil {
		common.ApiErrorMsg(c, "获取用户分组失败")
		return
	}
	payMoney, payableVND, err := sePayPayMoneyFromAmount(req.Amount, group)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	// Amount column stores the display amount, matching the pre-existing
	// RequestEpay behavior: tokens mode stores tokens/QuotaPerUnit.
	storedAmount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		storedAmount = decimal.NewFromInt(req.Amount).
			Div(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
	}

	topUp := &model.TopUp{
		UserId:          userId,
		Amount:          storedAmount,
		Money:           payMoney,
		PaymentMethod:   model.PaymentMethodSePay,
		PaymentProvider: model.PaymentProviderSePay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := model.InsertSePayTopUp(topUp); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("SePay 创建充值订单失败 user_id=%d amount=%d error=%q", userId, req.Amount, err.Error()))
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}
	common.ApiSuccess(c, buildSePayOrderResponse(topUp.TradeNo, topUp.CreateTime, payMoney, payableVND, topUp.Status))
}

// SePayRequestSubscriptionPay is POST /api/subscription/sepay/pay. Same
// response shape as SePayRequestTopUp but backed by SubscriptionOrder rows
// and using the plan's PriceAmount as the base amount (task 3.2).
func SePayRequestSubscriptionPay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	if !setting.IsSePayConfigured() {
		common.ApiErrorMsg(c, "SePay 未配置")
		return
	}

	var req sePaySubscriptionPayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	plan, err := model.GetSubscriptionPlanById(req.PlanId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !plan.Enabled {
		common.ApiErrorMsg(c, "套餐未启用")
		return
	}
	if plan.PriceAmount <= 0 {
		common.ApiErrorMsg(c, "套餐金额过低")
		return
	}

	userId := c.GetInt("id")
	if plan.MaxPurchasePerUser > 0 {
		count, err := model.CountUserSubscriptionsByPlan(userId, plan.Id)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if count >= int64(plan.MaxPurchasePerUser) {
			common.ApiErrorMsg(c, "已达到该套餐购买上限")
			return
		}
	}

	group, err := model.GetUserGroup(userId, true)
	if err != nil {
		common.ApiErrorMsg(c, "获取用户分组失败")
		return
	}
	payMoney, payableVND, err := sePayPayMoneyFromDecimal(decimal.NewFromFloat(plan.PriceAmount), group)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	order := &model.SubscriptionOrder{
		UserId:          userId,
		PlanId:          plan.Id,
		Money:           payMoney,
		PaymentMethod:   model.PaymentMethodSePay,
		PaymentProvider: model.PaymentProviderSePay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := model.InsertSePaySubscriptionOrder(order); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("SePay 创建订阅订单失败 user_id=%d plan_id=%d error=%q", userId, plan.Id, err.Error()))
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}
	common.ApiSuccess(c, buildSePayOrderResponse(order.TradeNo, order.CreateTime, payMoney, payableVND, order.Status))
}

// SePayGetOrder is GET /api/user/sepay/order/:trade_no, scoped to the
// authenticated user, returning status, payable amount, memo, and expiry
// (task 3.3). A trade number owned by another user or belonging to a
// retired-provider order resolves as not-found so no cross-user leak.
func SePayGetOrder(c *gin.Context) {
	userId := c.GetInt("id")
	tradeNo := strings.TrimSpace(c.Param("trade_no"))
	if tradeNo == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if t := model.GetTopUpByTradeNo(tradeNo); t != nil {
		if t.UserId != userId || t.PaymentProvider != model.PaymentProviderSePay {
			common.ApiError(c, model.ErrTopUpNotFound)
			return
		}
		common.ApiSuccess(c, buildSePayOrderResponse(t.TradeNo, t.CreateTime, t.Money, sePayPayableVNDIntFromMoney(t.Money), t.Status))
		return
	}
	if s := model.GetSubscriptionOrderByTradeNo(tradeNo); s != nil {
		if s.UserId != userId || s.PaymentProvider != model.PaymentProviderSePay {
			common.ApiError(c, model.ErrTopUpNotFound)
			return
		}
		common.ApiSuccess(c, buildSePayOrderResponse(s.TradeNo, s.CreateTime, s.Money, sePayPayableVNDIntFromMoney(s.Money), s.Status))
		return
	}
	common.ApiError(c, model.ErrTopUpNotFound)
}

// SePayWebhook is POST /api/sepay/webhook (anonymous, body-limited). It
// authenticates "Authorization: Apikey <key>" with crypto/subtle, extracts
// memo candidates from the free-form content, routes by prefix (SP topup /
// SS subscription), and settles at most one pending order per transfer. Every
// outcome branch returns HTTP 200 with body {"success": true}; only 401 (auth
// failure) and 500 (internal fault) are non-success (tasks 4.1–4.3, D6/D7).
func SePayWebhook(c *gin.Context) {
	if !isSePayWebhookEnabled() {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("SePay webhook 被拒绝 reason=webhook_disabled client_ip=%s", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized"})
		return
	}
	provided, ok := parseSePayApiKeyAuth(c.GetHeader("Authorization"))
	if !ok {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("SePay webhook 认证失败 reason=missing_or_malformed client_ip=%s", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized"})
		return
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(setting.SePayWebhookApiKey)) != 1 {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("SePay webhook 认证失败 reason=wrong_key client_ip=%s", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized"})
		return
	}

	payload := &sePayWebhookPayload{}
	if err := common.DecodeJson(c.Request.Body, payload); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("SePay webhook 解析失败 client_ip=%s error=%q", c.ClientIP(), err.Error()))
		sepayWebhookAck(c)
		return
	}

	if strings.EqualFold(payload.TransferType, "out") {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("SePay webhook 出账忽略 id=%d reference_code=%s", payload.Id, payload.ReferenceCode))
		sepayWebhookAck(c)
		return
	}

	candidates := extractSePayMemos(payload.Content)
	if len(candidates) == 0 {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("SePay webhook 未匹配 memo id=%d reference_code=%s amount=%d content=%q", payload.Id, payload.ReferenceCode, payload.TransferAmount, payload.Content))
		sepayWebhookAck(c)
		return
	}

	type match struct {
		tradeNo string
		isSub   bool
	}
	var pending, credited, expired sePayMatchList
	seen := map[string]bool{}
	for _, memo := range candidates {
		if seen[memo] {
			continue
		}
		seen[memo] = true
		if t := model.GetTopUpByTradeNo(memo); t != nil && t.PaymentProvider == model.PaymentProviderSePay {
			switch t.Status {
			case common.TopUpStatusPending:
				pending = append(pending, sePayMatch{tradeNo: t.TradeNo})
			case common.TopUpStatusSuccess:
				credited = append(credited, sePayMatch{tradeNo: t.TradeNo})
			case common.TopUpStatusExpired:
				expired = append(expired, sePayMatch{tradeNo: t.TradeNo})
			}
			continue
		}
		if s := model.GetSubscriptionOrderByTradeNo(memo); s != nil && s.PaymentProvider == model.PaymentProviderSePay {
			switch s.Status {
			case common.TopUpStatusPending:
				pending = append(pending, sePayMatch{tradeNo: s.TradeNo, isSub: true})
			case common.TopUpStatusSuccess:
				credited = append(credited, sePayMatch{tradeNo: s.TradeNo, isSub: true})
			case common.TopUpStatusExpired:
				expired = append(expired, sePayMatch{tradeNo: s.TradeNo, isSub: true})
			}
		}
	}

	if len(pending) > 1 {
		logger.LogError(c.Request.Context(), fmt.Sprintf("SePay webhook 歧义多单不结算 id=%d reference_code=%s pending=%v", payload.Id, payload.ReferenceCode, candidateTradeNoStrings(pending)))
		sepayWebhookAck(c)
		return
	}
	if len(pending) == 0 {
		if len(credited) > 0 {
			logger.LogInfo(c.Request.Context(), fmt.Sprintf("SePay webhook 重复幂等忽略 id=%d reference_code=%s credited=%v", payload.Id, payload.ReferenceCode, candidateTradeNoStrings(credited)))
			sepayWebhookAck(c)
			return
		}
		if len(expired) > 0 {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("SePay webhook 匹配订单已过期不结算 id=%d reference_code=%s amount=%d expired=%v", payload.Id, payload.ReferenceCode, payload.TransferAmount, candidateTradeNoStrings(expired)))
			sepayWebhookAck(c)
			return
		}
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("SePay webhook 未匹配待支付订单 id=%d reference_code=%s amount=%d candidates=%v", payload.Id, payload.ReferenceCode, payload.TransferAmount, candidates))
		sepayWebhookAck(c)
		return
	}

	target := pending[0]
	LockOrder(target.tradeNo)
	defer UnlockOrder(target.tradeNo)

	if target.isSub {
		order := model.GetSubscriptionOrderByTradeNo(target.tradeNo)
		if order == nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("SePay webhook 订阅订单查询失败 trade_no=%s id=%d", target.tradeNo, payload.Id))
			sepayWebhookAck(c)
			return
		}
		_, err := model.RechargeSePaySubscription(order.TradeNo, payload.TransferAmount)
		sepaySettlementOutcomeLog(c, payload, order.TradeNo, sePayPayableVNDIntFromMoney(order.Money), err)
		sepayWebhookAck(c)
		return
	}
	topUp := model.GetTopUpByTradeNo(target.tradeNo)
	if topUp == nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("SePay webhook 充值订单查询失败 trade_no=%s id=%d", target.tradeNo, payload.Id))
		sepayWebhookAck(c)
		return
	}
	_, err := model.RechargeSePay(topUp.TradeNo, payload.TransferAmount, c.ClientIP())
	sepaySettlementOutcomeLog(c, payload, topUp.TradeNo, sePayPayableVNDIntFromMoney(topUp.Money), err)
	sepayWebhookAck(c)
}

func sepaySettlementOutcomeLog(c *gin.Context, payload *sePayWebhookPayload, tradeNo string, payableVND int64, err error) {
	switch {
	case err == nil:
		if payload.TransferAmount > payableVND {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("SePay webhook 超额支付已按订单金额入账 trade_no=%s payable=%d transfer=%d id=%d reference_code=%s", tradeNo, payableVND, payload.TransferAmount, payload.Id, payload.ReferenceCode))
		} else {
			logger.LogInfo(c.Request.Context(), fmt.Sprintf("SePay webhook 充值成功 trade_no=%s id=%d reference_code=%s", tradeNo, payload.Id, payload.ReferenceCode))
		}
	case errors.Is(err, model.ErrTopUpAmountInsufficient):
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("SePay webhook 金额不足不结算 trade_no=%s payable=%d transfer=%d id=%d reference_code=%s", tradeNo, payableVND, payload.TransferAmount, payload.Id, payload.ReferenceCode))
	case errors.Is(err, model.ErrTopUpNotFound):
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("SePay webhook 订单不存在 trade_no=%s id=%d reference_code=%s", tradeNo, payload.Id, payload.ReferenceCode))
	case errors.Is(err, model.ErrPaymentMethodMismatch):
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("SePay webhook 订单支付网关不匹配 trade_no=%s id=%d reference_code=%s", tradeNo, payload.Id, payload.ReferenceCode))
	case errors.Is(err, model.ErrTopUpStatusInvalid):
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("SePay webhook 订单状态非法或已过期 trade_no=%s id=%d reference_code=%s", tradeNo, payload.Id, payload.ReferenceCode))
	default:
		logger.LogError(c.Request.Context(), fmt.Sprintf("SePay webhook 结算失败 trade_no=%s id=%d reference_code=%s error=%q", tradeNo, payload.Id, payload.ReferenceCode, err.Error()))
	}
}

func sepayWebhookAck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func candidateTradeNoStrings(matches sePayMatchList) []string {
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.tradeNo)
	}
	return out
}

// normalizeSePayContent converts free-form bank description into a
// A-Z0-9-and-space string (D2). Every non-[A-Z0-9] byte becomes a space so
// the memo regex can still match when the memo is packed against adjacent
// text.
func normalizeSePayContent(content string) string {
	upper := strings.ToUpper(content)
	var b strings.Builder
	b.Grow(len(upper))
	for _, r := range upper {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// extractSePayMemos returns every distinct fixed-shape memo candidate found
// in the free-form content, after normalization.
func extractSePayMemos(content string) []string {
	normalized := normalizeSePayContent(content)
	seen := map[string]bool{}
	var out []string
	for _, m := range sePayMemoRegex.FindAllString(normalized, -1) {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

// parseSePayApiKeyAuth parses "Authorization: Apikey <key>". Header scheme
// literal is case-sensitive ("Apikey", not "APIKEY" or "ApiKey") because that
// is what SePay sends. Returns the key and whether the header was well-formed.
func parseSePayApiKeyAuth(header string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(header))
	if len(fields) != 2 || fields[0] != "Apikey" || fields[1] == "" {
		return "", false
	}
	return fields[1], true
}

// sePayPayMoneyFromAmount is the top-up path of the D5 conversion; the plan
// price path uses sePayPayMoneyFromDecimal so both share one helper.
func sePayPayMoneyFromAmount(amount int64, group string) (float64, int64, error) {
	dAmount := decimal.NewFromInt(amount)
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount = dAmount.Div(decimal.NewFromFloat(common.QuotaPerUnit))
	}
	return sePayPayMoneyFromDecimal(dAmount, group)
}

// sePayPayMoneyFromDecimal applies the D5 formula:
// payable_vnd = round(amount × price × topup_group_ratio × discount).
// Discount tiers by preset are only meaningful for top-up amounts; the caller
// passes the same amount both as the input and for the discount lookup.
func sePayPayMoneyFromDecimal(amount decimal.Decimal, group string) (float64, int64, error) {
	topupGroupRatio := common.GetTopupGroupRatio(group)
	if topupGroupRatio == 0 {
		topupGroupRatio = 1
	}
	dPrice := decimal.NewFromFloat(operation_setting.Price)
	dRatio := decimal.NewFromFloat(topupGroupRatio)
	discount := 1.0
	// Discount tiers are keyed by preset integer amounts; only exact matches
	// apply. Plans and non-preset amounts skip.
	if amtInt, exact := amount.Float64(); exact {
		if ds, ok := operation_setting.GetPaymentSetting().AmountDiscount[int(amtInt)]; ok && ds > 0 {
			discount = ds
		}
	}
	dDiscount := decimal.NewFromFloat(discount)
	payMoneyDec := amount.Mul(dPrice).Mul(dRatio).Mul(dDiscount)
	payableVND := payMoneyDec.Round(0).IntPart()
	if payableVND <= 0 {
		return 0, 0, errors.New("充值金额过低")
	}
	payMoneyFloat, _ := payMoneyDec.Round(0).Float64()
	return payMoneyFloat, payableVND, nil
}

func sePayPayableVNDIntFromMoney(money float64) int64 {
	return decimal.NewFromFloat(money).Round(0).IntPart()
}

func sePayEffectiveMinTopUp() int {
	if setting.SePayMinTopUp > 0 {
		return setting.SePayMinTopUp
	}
	return operation_setting.MinTopUp
}

func buildSePayOrderResponse(tradeNo string, createTime int64, money float64, payableVND int64, status string) sePayOrderResponse {
	return sePayOrderResponse{
		TradeNo:     tradeNo,
		Memo:        tradeNo,
		PayableVND:  payableVND,
		BankAccount: setting.SePayBankAccountNumber,
		BankCode:    setting.SePayBankCode,
		AccountName: setting.SePayAccountHolder,
		VietQRURL:   buildSePayVietQRURL(tradeNo, payableVND),
		CreateTime:  createTime,
		ExpireTime:  model.SePayOrderExpiryUnix(createTime),
		Status:      status,
		Money:       money,
	}
}

// buildSePayVietQRURL returns a per-order image URL that carries the
// destination account, bank code, payable Dong amount (D5: frozen at
// creation), and the transfer memo (D11).
func buildSePayVietQRURL(memo string, amountVND int64) string {
	q := url.Values{}
	q.Set("amount", strconv.FormatInt(amountVND, 10))
	q.Set("addInfo", memo)
	q.Set("accountName", setting.SePayAccountHolder)
	return fmt.Sprintf(
		"https://img.vietqr.io/image/%s-%s-%s.png?%s",
		url.PathEscape(setting.SePayBankCode),
		url.PathEscape(setting.SePayBankAccountNumber),
		url.PathEscape(sePayVietQRTemplate),
		q.Encode(),
	)
}
