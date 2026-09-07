package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

// Stable, machine-readable error codes for /api/bot/v1. These travel in the
// JSON body of an ordinary HTTP 200 business-logic refusal — distinct from
// the transport-level 401 (bad credential), 404 (unconfigured), and 429
// (rate limited) responses the bot auth/throttle layer writes directly — so
// the bot can tell every outcome apart (specs/telegram/bot-api/spec.md,
// "Endpoints that need an account report an unlinked Telegram user
// distinguishably").
const (
	telegramBotErrorInvalidRequest   = "TELEGRAM_BOT_INVALID_REQUEST"
	telegramBotErrorNotLinked        = "TELEGRAM_BOT_NOT_LINKED"
	telegramBotErrorAccountUnusable  = "TELEGRAM_BOT_ACCOUNT_UNUSABLE"
	telegramBotErrorAlreadyLinked    = "TELEGRAM_BOT_ALREADY_LINKED"
	telegramBotErrorKeyNotFound      = "TELEGRAM_BOT_KEY_NOT_FOUND"
	telegramBotErrorOrderNotFound    = "TELEGRAM_BOT_ORDER_NOT_FOUND"
	telegramBotErrorTopUpUnavailable = "TELEGRAM_BOT_TOPUP_UNAVAILABLE"
	telegramBotErrorInternal         = "TELEGRAM_BOT_INTERNAL_ERROR"
)

// telegramBotLinkRedemptionAddress is the website endpoint a bot-issued link
// code is redeemed at. It is never reachable from this router
// (specs/telegram/account-link/spec.md, "Redemption requires an
// authenticated session").
const telegramBotLinkRedemptionAddress = "/api/user/telegram/link/confirm"

func telegramBotSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func telegramBotError(c *gin.Context, statusCode int, errorCode, message string) {
	c.JSON(statusCode, gin.H{"success": false, "error_code": errorCode, "message": message})
}

func telegramBotTopUpAvailable() bool {
	return operation_setting.IsPaymentComplianceConfirmed() && setting.IsSePayConfigured()
}

// telegramBotResolveUser resolves telegram_user_id to its linked, usable
// account for every account-scoped handler below: it enforces the
// per-Telegram-user rate limit, records the audit entry for a resolution
// failure, and writes the appropriate response itself. Callers get ok=false
// once the response has already been written and must return immediately.
func telegramBotResolveUser(c *gin.Context, telegramUserID, endpoint string) (*model.User, bool) {
	telegramUserID = strings.TrimSpace(telegramUserID)
	if telegramUserID == "" {
		telegramBotError(c, http.StatusBadRequest, telegramBotErrorInvalidRequest, "telegram_user_id is required")
		return nil, false
	}
	if !middleware.TelegramUserRateLimit(c, telegramUserID) {
		return nil, false
	}
	user, err := service.ResolveTelegramBotUser(telegramUserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTelegramBotUserNotLinked):
			service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, 0, endpoint, service.TelegramBotAuditOutcomeNotLinked)
			telegramBotError(c, http.StatusOK, telegramBotErrorNotLinked, "telegram user is not linked to any account")
		case errors.Is(err, service.ErrTelegramBotAccountUnusable):
			service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, 0, endpoint, service.TelegramBotAuditOutcomeAccountUnusable)
			telegramBotError(c, http.StatusOK, telegramBotErrorAccountUnusable, "linked account is disabled or deleted")
		default:
			common.SysError("telegram bot identity resolution failed: " + err.Error())
			service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, 0, endpoint, service.TelegramBotAuditOutcomeError)
			telegramBotError(c, http.StatusInternalServerError, telegramBotErrorInternal, "internal error")
		}
		return nil, false
	}
	return user, true
}

// --- 7.1 Health -------------------------------------------------------

type telegramBotHealthResponse struct {
	Version                string `json:"version"`
	TopUpAvailable         bool   `json:"topup_available"`
	TopUpMinAmount         int    `json:"topup_min_amount"`
	TopUpMaxAmount         int64  `json:"topup_max_amount"`
	TelegramHandleRequired bool   `json:"telegram_handle_required"`
}

// TelegramBotHealth is GET /api/bot/v1/health. It takes no telegram_user_id
// and is throttled by the ordinary global limit at the router, not the
// per-Telegram-user limiter (specs/telegram/bot-api/spec.md, "Integration
// health report").
func TelegramBotHealth(c *gin.Context) {
	telegramBotSuccess(c, telegramBotHealthResponse{
		Version:                common.Version,
		TopUpAvailable:         telegramBotTopUpAvailable(),
		TopUpMinAmount:         service.SePayEffectiveMinTopUp(),
		TopUpMaxAmount:         service.SePayMaxTopUpAmount,
		TelegramHandleRequired: setting.TelegramHandleRequired,
	})
}

// --- 7.2 Identity resolution -------------------------------------------

type telegramBotIdentityResponse struct {
	Linked    bool   `json:"linked"`
	AccountId int    `json:"account_id,omitempty"`
	Username  string `json:"username,omitempty"`
	Status    int    `json:"status,omitempty"`
}

type telegramBotIdentityResolveRequest struct {
	TelegramUserID string `json:"telegram_user_id"`
}

// TelegramBotIdentity is POST /api/bot/v1/identity/resolve. The Telegram
// identifier is read only from the JSON body — never a query parameter — so
// it never reaches access logs, the same reason keys on this surface are
// body-only (specs/telegram/bot-api/spec.md, "API keys are never accepted in
// a URL or query string"). Unlike every other account-scoped handler, an
// unbound telegram_user_id is a normal, successful outcome here — never the
// not-linked error code — because reporting bound vs. unbound is the
// endpoint's entire purpose (specs/telegram/bot-api/spec.md, "Telegram
// identity resolution"). It never reports balance, keys, orders, or logs.
func TelegramBotIdentity(c *gin.Context) {
	var req telegramBotIdentityResolveRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		telegramBotError(c, http.StatusBadRequest, telegramBotErrorInvalidRequest, "invalid request body")
		return
	}
	telegramUserID := strings.TrimSpace(req.TelegramUserID)
	if telegramUserID == "" {
		telegramBotError(c, http.StatusBadRequest, telegramBotErrorInvalidRequest, "telegram_user_id is required")
		return
	}
	if !middleware.TelegramUserRateLimit(c, telegramUserID) {
		return
	}
	var user model.User
	err := model.DB.Where("telegram_id = ?", telegramUserID).First(&user).Error
	if err != nil {
		service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, 0, "identity", service.TelegramBotAuditOutcomeSuccess)
		telegramBotSuccess(c, telegramBotIdentityResponse{Linked: false})
		return
	}
	service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, user.Id, "identity", service.TelegramBotAuditOutcomeSuccess)
	telegramBotSuccess(c, telegramBotIdentityResponse{
		Linked:    true,
		AccountId: user.Id,
		Username:  user.Username,
		Status:    user.Status,
	})
}

// --- 7.3 Link code issuance ---------------------------------------------

type telegramBotLinkStartRequest struct {
	TelegramUserID   string `json:"telegram_user_id"`
	TelegramUsername string `json:"telegram_username"`
}

type telegramBotLinkStartResponse struct {
	Code              string `json:"code"`
	ExpiresAt         int64  `json:"expires_at"`
	RedemptionAddress string `json:"redemption_address"`
}

// TelegramBotLinkStart is POST /api/bot/v1/identity/link/start, delegating to
// the issuance logic in model.IssueTelegramLinkCode. This router has no
// corresponding redeem route: redemption lives only on the
// user-authenticated website endpoint (controller.TelegramLinkRedeem).
func TelegramBotLinkStart(c *gin.Context) {
	var req telegramBotLinkStartRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		telegramBotError(c, http.StatusBadRequest, telegramBotErrorInvalidRequest, "invalid request body")
		return
	}
	telegramUserID := strings.TrimSpace(req.TelegramUserID)
	if telegramUserID == "" {
		telegramBotError(c, http.StatusBadRequest, telegramBotErrorInvalidRequest, "telegram_user_id is required")
		return
	}
	if !middleware.TelegramUserRateLimit(c, telegramUserID) {
		return
	}
	code, expiresAt, err := model.IssueTelegramLinkCode(telegramUserID, req.TelegramUsername)
	if err != nil {
		if errors.Is(err, model.ErrTelegramLinkAlreadyBound) {
			service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, 0, "identity.link.start", service.TelegramBotAuditOutcomeRejected)
			telegramBotError(c, http.StatusOK, telegramBotErrorAlreadyLinked, "telegram account is already linked")
			return
		}
		common.SysError("telegram bot link code issuance failed: " + err.Error())
		service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, 0, "identity.link.start", service.TelegramBotAuditOutcomeError)
		telegramBotError(c, http.StatusInternalServerError, telegramBotErrorInternal, "internal error")
		return
	}
	service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, 0, "identity.link.start", service.TelegramBotAuditOutcomeSuccess)
	telegramBotSuccess(c, telegramBotLinkStartResponse{
		Code:              code,
		ExpiresAt:         expiresAt.Unix(),
		RedemptionAddress: telegramBotLinkRedemptionAddress,
	})
}

// --- 7.4 Account summary -------------------------------------------------

type telegramBotAccountResponse struct {
	Quota            int    `json:"quota"`
	UsedQuota        int    `json:"used_quota"`
	Group            string `json:"group"`
	Status           int    `json:"status"`
	TelegramUsername string `json:"telegram_username"`
}

func TelegramBotAccount(c *gin.Context) {
	telegramUserID := c.Query("telegram_user_id")
	user, ok := telegramBotResolveUser(c, telegramUserID, "account")
	if !ok {
		return
	}
	service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, user.Id, "account", service.TelegramBotAuditOutcomeSuccess)
	telegramBotSuccess(c, telegramBotAccountResponse{
		Quota:            user.Quota,
		UsedQuota:        user.UsedQuota,
		Group:            user.Group,
		Status:           user.Status,
		TelegramUsername: user.TelegramUsername,
	})
}

// --- 7.5 Top-up configuration --------------------------------------------

type telegramBotTopUpPreset struct {
	Amount   int     `json:"amount"`
	Discount float64 `json:"discount,omitempty"`
}

type telegramBotTopUpConfigResponse struct {
	Available            bool                     `json:"available"`
	MinAmount            int                      `json:"min_amount"`
	MaxAmount            int64                    `json:"max_amount"`
	Presets              []telegramBotTopUpPreset `json:"presets"`
	Price                float64                  `json:"price"`
	OrderLifetimeMinutes int                      `json:"order_lifetime_minutes"`
}

// TelegramBotTopUpConfig is GET /api/bot/v1/topup/config. telegram_user_id is
// optional here: when present it is used only to scope the rate limit.
func TelegramBotTopUpConfig(c *gin.Context) {
	if telegramUserID := strings.TrimSpace(c.Query("telegram_user_id")); telegramUserID != "" {
		if !middleware.TelegramUserRateLimit(c, telegramUserID) {
			return
		}
	}
	paymentSetting := operation_setting.GetPaymentSetting()
	presets := make([]telegramBotTopUpPreset, 0, len(paymentSetting.AmountOptions))
	for _, amount := range paymentSetting.AmountOptions {
		presets = append(presets, telegramBotTopUpPreset{
			Amount:   amount,
			Discount: paymentSetting.AmountDiscount[amount],
		})
	}
	telegramBotSuccess(c, telegramBotTopUpConfigResponse{
		Available:            telegramBotTopUpAvailable(),
		MinAmount:            service.SePayEffectiveMinTopUp(),
		MaxAmount:            service.SePayMaxTopUpAmount,
		Presets:              presets,
		Price:                operation_setting.Price,
		OrderLifetimeMinutes: setting.SePayOrderExpiryMinutes,
	})
}

// --- 7.6 Top-up order creation --------------------------------------------

type telegramBotCreateOrderRequest struct {
	TelegramUserID string `json:"telegram_user_id"`
	Amount         int64  `json:"amount"`
}

// TelegramBotCreateOrder is POST /api/bot/v1/topup/orders, delegating the full
// validation-through-insertion chain to service.CreateSePayTopUpOrder — the
// same producer the console web handler calls — so a bot-created order is an
// ordinary SePay order in every respect (specs/telegram/bot-api/spec.md,
// "Bot-initiated top-up order creation").
func TelegramBotCreateOrder(c *gin.Context) {
	var req telegramBotCreateOrderRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		telegramBotError(c, http.StatusBadRequest, telegramBotErrorInvalidRequest, "invalid request body")
		return
	}
	user, ok := telegramBotResolveUser(c, req.TelegramUserID, "topup.order.create")
	if !ok {
		return
	}
	telegramUserID := strings.TrimSpace(req.TelegramUserID)

	topUp, payMoney, payableVND, err := service.CreateSePayTopUpOrder(c.Request.Context(), user.Id, req.Amount)
	if err != nil {
		if errors.Is(err, service.ErrPaymentComplianceRequired) || errors.Is(err, service.ErrSePayNotConfigured) {
			service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, user.Id, "topup.order.create", service.TelegramBotAuditOutcomeRejected)
			telegramBotError(c, http.StatusOK, telegramBotErrorTopUpUnavailable, "top-up is unavailable")
			return
		}
		service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, user.Id, "topup.order.create", service.TelegramBotAuditOutcomeRejected)
		telegramBotError(c, http.StatusOK, telegramBotErrorInvalidRequest, err.Error())
		return
	}
	service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, user.Id, "topup.order.create", service.TelegramBotAuditOutcomeSuccess)
	telegramBotSuccess(c, buildSePayOrderResponse(topUp.TradeNo, topUp.CreateTime, payMoney, payableVND, topUp.Status))
}

// --- 7.7 Order lookup ------------------------------------------------------

// TelegramBotOrderStatus is GET /api/bot/v1/topup/orders/:trade_no. A trade
// number owned by another account and one that does not exist resolve to the
// identical not-found response (specs/telegram/bot-api/spec.md, "Order
// lookup is scoped to the order's owner").
func TelegramBotOrderStatus(c *gin.Context) {
	telegramUserID := c.Query("telegram_user_id")
	user, ok := telegramBotResolveUser(c, telegramUserID, "topup.order.status")
	if !ok {
		return
	}
	tradeNo := strings.TrimSpace(c.Param("trade_no"))
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if tradeNo == "" || topUp == nil || topUp.UserId != user.Id || topUp.PaymentProvider != model.PaymentProviderSePay {
		service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, user.Id, "topup.order.status", service.TelegramBotAuditOutcomeNotFound)
		telegramBotError(c, http.StatusOK, telegramBotErrorOrderNotFound, "order not found")
		return
	}
	service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, user.Id, "topup.order.status", service.TelegramBotAuditOutcomeSuccess)
	telegramBotSuccess(c, buildSePayOrderResponse(topUp.TradeNo, topUp.CreateTime, topUp.Money, sePayPayableVNDIntFromMoney(topUp.Money), topUp.Status))
}

// --- 7.8 Top-up history ------------------------------------------------

func TelegramBotTopUpHistory(c *gin.Context) {
	telegramUserID := c.Query("telegram_user_id")
	user, ok := telegramBotResolveUser(c, telegramUserID, "topup.history")
	if !ok {
		return
	}
	pageInfo := common.GetPageQuery(c)
	topups, total, err := model.GetUserTopUps(user.Id, pageInfo)
	if err != nil {
		telegramBotError(c, http.StatusInternalServerError, telegramBotErrorInternal, "internal error")
		return
	}
	items := make([]sePayOrderResponse, 0, len(topups))
	for _, t := range topups {
		if t.PaymentProvider != model.PaymentProviderSePay {
			continue
		}
		items = append(items, buildSePayOrderResponse(t.TradeNo, t.CreateTime, t.Money, sePayPayableVNDIntFromMoney(t.Money), t.Status))
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, user.Id, "topup.history", service.TelegramBotAuditOutcomeSuccess)
	telegramBotSuccess(c, pageInfo)
}

// --- 7.9 Key inspection --------------------------------------------------

type telegramBotKeyInspectRequest struct {
	TelegramUserID string `json:"telegram_user_id"`
	Key            string `json:"key"`
}

// TelegramBotKeyInspect is POST /api/bot/v1/keys/check. The key is read only
// from the JSON body (specs/telegram/bot-api/spec.md, "API keys are never
// accepted in a URL or query string"), normalized the same way the public
// key-check endpoint normalizes it, and reported through the same shared
// producer (service.BuildTokenCheckReport) so the field set can never drift
// between the two surfaces.
func TelegramBotKeyInspect(c *gin.Context) {
	var req telegramBotKeyInspectRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		telegramBotError(c, http.StatusBadRequest, telegramBotErrorInvalidRequest, "invalid request body")
		return
	}
	telegramUserID := strings.TrimSpace(req.TelegramUserID)
	if telegramUserID == "" {
		telegramBotError(c, http.StatusBadRequest, telegramBotErrorInvalidRequest, "telegram_user_id is required")
		return
	}
	if !middleware.TelegramUserRateLimit(c, telegramUserID) {
		return
	}

	key := service.NormalizeTokenKey(req.Key)
	if key == "" {
		service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, 0, "key.inspect", service.TelegramBotAuditOutcomeNotFound)
		telegramBotError(c, http.StatusOK, telegramBotErrorKeyNotFound, "key not found")
		return
	}
	token, err := model.GetTokenByKey(key, false)
	if err != nil {
		service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, 0, "key.inspect", service.TelegramBotAuditOutcomeNotFound)
		telegramBotError(c, http.StatusOK, telegramBotErrorKeyNotFound, "key not found")
		return
	}
	report, err := service.BuildTokenCheckReport(token)
	if err != nil {
		common.SysError("telegram bot key inspection report failed: " + err.Error())
		telegramBotError(c, http.StatusInternalServerError, telegramBotErrorInternal, "internal error")
		return
	}
	owner := false
	if resolvedUser, resolveErr := service.ResolveTelegramBotUser(telegramUserID); resolveErr == nil {
		owner = resolvedUser.Id == token.UserId
	}
	// Audited without the submitted key or any prefix of it
	// (specs/telegram/bot-api/spec.md, "Key inspection is audited without the
	// key").
	service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, 0, "key.inspect", service.TelegramBotAuditOutcomeSuccess)
	telegramBotSuccess(c, gin.H{"report": report, "is_owner": owner})
}

// --- 7.10 Key listing ------------------------------------------------------

// telegramBotKeyListingEntry deliberately carries no key value or key
// fragment field of any kind (specs/telegram/bot-api/spec.md, "Key listing
// never discloses key values"); it must never gain a Key or masked-key field.
type telegramBotKeyListingEntry struct {
	Id             int    `json:"id"`
	Name           string `json:"name"`
	Group          string `json:"group"`
	Status         int    `json:"status"`
	RemainQuota    int    `json:"remain_quota"`
	UsedQuota      int    `json:"used_quota"`
	UnlimitedQuota bool   `json:"unlimited_quota"`
	ExpiredTime    int64  `json:"expired_time"`
}

func TelegramBotKeyList(c *gin.Context) {
	telegramUserID := c.Query("telegram_user_id")
	user, ok := telegramBotResolveUser(c, telegramUserID, "keys.list")
	if !ok {
		return
	}
	pageInfo := common.GetPageQuery(c)
	tokens, err := model.GetAllUserTokens(user.Id, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		telegramBotError(c, http.StatusInternalServerError, telegramBotErrorInternal, "internal error")
		return
	}
	total, err := model.CountUserTokens(user.Id)
	if err != nil {
		telegramBotError(c, http.StatusInternalServerError, telegramBotErrorInternal, "internal error")
		return
	}
	entries := make([]telegramBotKeyListingEntry, 0, len(tokens))
	for _, token := range tokens {
		entries = append(entries, telegramBotKeyListingEntry{
			Id:             token.Id,
			Name:           token.Name,
			Group:          token.Group,
			Status:         token.Status,
			RemainQuota:    token.RemainQuota,
			UsedQuota:      token.UsedQuota,
			UnlimitedQuota: token.UnlimitedQuota,
			ExpiredTime:    token.ExpiredTime,
		})
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(entries)
	service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, user.Id, "keys.list", service.TelegramBotAuditOutcomeSuccess)
	telegramBotSuccess(c, pageInfo)
}

// --- 7.11 Usage log listing --------------------------------------------

func TelegramBotUsageLogs(c *gin.Context) {
	telegramUserID := c.Query("telegram_user_id")
	user, ok := telegramBotResolveUser(c, telegramUserID, "logs.list")
	if !ok {
		return
	}
	pageInfo := common.GetPageQuery(c)
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	modelName := c.Query("model_name")
	keyName := c.Query("key_name")
	logs, total, err := model.GetUserLogs(
		user.Id, logType, startTimestamp, endTimestamp, modelName, keyName,
		pageInfo.GetStartIdx(), pageInfo.GetPageSize(), "", "", "",
	)
	if err != nil {
		telegramBotError(c, http.StatusInternalServerError, telegramBotErrorInternal, "internal error")
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, user.Id, "logs.list", service.TelegramBotAuditOutcomeSuccess)
	telegramBotSuccess(c, pageInfo)
}

// --- 7.12 Usage statistics -----------------------------------------------

func TelegramBotUsageStats(c *gin.Context) {
	telegramUserID := c.Query("telegram_user_id")
	user, ok := telegramBotResolveUser(c, telegramUserID, "usage.stats")
	if !ok {
		return
	}
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	summary, err := model.GetUserUsageSummary(user.Id, startTimestamp, endTimestamp)
	if err != nil {
		telegramBotError(c, http.StatusInternalServerError, telegramBotErrorInternal, "internal error")
		return
	}
	service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, user.Id, "usage.stats", service.TelegramBotAuditOutcomeSuccess)
	telegramBotSuccess(c, summary)
}

// --- 7.13 Key-scoped usage log --------------------------------------------

type telegramBotKeyLogsRequest struct {
	TelegramUserID string `json:"telegram_user_id"`
	Key            string `json:"key"`
}

// TelegramBotKeyLogs is POST /api/bot/v1/keys/logs. The key is read only from
// the JSON body, matching TelegramBotKeyInspect.
func TelegramBotKeyLogs(c *gin.Context) {
	var req telegramBotKeyLogsRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		telegramBotError(c, http.StatusBadRequest, telegramBotErrorInvalidRequest, "invalid request body")
		return
	}
	telegramUserID := strings.TrimSpace(req.TelegramUserID)
	if telegramUserID == "" {
		telegramBotError(c, http.StatusBadRequest, telegramBotErrorInvalidRequest, "telegram_user_id is required")
		return
	}
	if !middleware.TelegramUserRateLimit(c, telegramUserID) {
		return
	}

	key := service.NormalizeTokenKey(req.Key)
	if key == "" {
		service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, 0, "key.logs", service.TelegramBotAuditOutcomeNotFound)
		telegramBotError(c, http.StatusOK, telegramBotErrorKeyNotFound, "key not found")
		return
	}
	token, err := model.GetTokenByKey(key, false)
	if err != nil {
		service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, 0, "key.logs", service.TelegramBotAuditOutcomeNotFound)
		telegramBotError(c, http.StatusOK, telegramBotErrorKeyNotFound, "key not found")
		return
	}
	pageInfo := common.GetPageQuery(c)
	logs, total, err := model.GetLogsByTokenIdPaginated(token.Id, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		telegramBotError(c, http.StatusInternalServerError, telegramBotErrorInternal, "internal error")
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	service.RecordTelegramBotAudit(c.Request.Context(), telegramUserID, 0, "key.logs", service.TelegramBotAuditOutcomeSuccess)
	telegramBotSuccess(c, pageInfo)
}
