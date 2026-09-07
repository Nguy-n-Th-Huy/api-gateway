package controller

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	errTelegramLinkAccountAlreadyBound = errors.New("account already holds a telegram binding")
	errTelegramLinkAccountUnusable     = errors.New("telegram link account is disabled or deleted")
)

type telegramLinkRedeemRequest struct {
	Code string `json:"code"`
}

// TelegramLinkRedeem is POST /api/user/telegram/link/confirm, reachable only
// to a signed-in user. It is the ONLY place a bot-issued link code can be
// redeemed: the /api/bot/v1 surface can issue a code but exposes no route
// that consumes one, so possession of the bot service key alone can never
// complete a link (specs/telegram/account-link/spec.md, "Redemption requires
// an authenticated session"). It applies every guard the existing Telegram
// Login Widget binding path (TelegramBind, controller/telegram.go) applies —
// already-bound identity, already-bound account, disabled/deleted account,
// revoked session — and binds inside the same transaction that consumes the
// code (model.ConsumeAuthFlowWithAction), so a guard failure rolls back the
// consumption along with the binding and the code stays redeemable.
func TelegramLinkRedeem(c *gin.Context) {
	var req telegramLinkRedeemRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgTelegramLinkCodeRequired)
		return
	}
	code := model.NormalizeTelegramLinkCode(req.Code)
	if code == "" {
		common.ApiErrorI18n(c, i18n.MsgTelegramLinkCodeRequired)
		return
	}

	identity, ok := middleware.GetSessionAuthIdentity(c)
	if !ok {
		common.ApiErrorI18n(c, i18n.MsgAuthNotLoggedIn)
		return
	}

	// Pre-check the redeeming session before touching the code at all, so a
	// revoked session or an unusable account never consumes it.
	if _, err := service.ValidateSessionReference(identity.UserID, identity.SessionID); err != nil {
		if !errors.Is(err, service.ErrLoginSessionInvalid) &&
			!errors.Is(err, service.ErrLoginSessionRevoked) &&
			!errors.Is(err, model.ErrUserSessionInactive) &&
			!errors.Is(err, gorm.ErrRecordNotFound) {
			common.SysError("TelegramLinkRedeem session validation failed: " + err.Error())
			common.ApiErrorI18n(c, i18n.MsgTelegramLinkInternalError)
			return
		}
		respondTelegramLinkSessionRefusal(c, identity.UserID)
		return
	}

	var linkedTelegramID, linkedUsername string
	_, err := model.ConsumeAuthFlowWithAction(code, model.AuthFlowMatch{
		Purpose: model.AuthFlowPurposeTelegramLink,
	}, func(tx *gorm.DB, flow *model.AuthFlow) error {
		var payload model.TelegramLinkPayload
		if unmarshalErr := common.UnmarshalJsonStr(flow.Payload, &payload); unmarshalErr != nil {
			return unmarshalErr
		}
		if payload.TelegramUserID == "" {
			return model.ErrAuthFlowInvalid
		}

		var user model.User
		if err := tx.First(&user, identity.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errTelegramLinkAccountUnusable
			}
			return err
		}
		if user.Status != common.UserStatusEnabled {
			return errTelegramLinkAccountUnusable
		}

		var session model.UserSession
		if err := tx.Where("sid = ? AND user_id = ?", identity.SessionID, identity.UserID).First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return service.ErrLoginSessionRevoked
			}
			return err
		}
		if session.Status != model.UserSessionStatusActive || session.RevokedAt != 0 || session.ExpiresAt <= common.GetTimestamp() {
			return service.ErrLoginSessionRevoked
		}
		if session.UserAuthVersion != user.AuthVersion {
			return service.ErrLoginSessionRevoked
		}
		if user.TelegramId != "" {
			return errTelegramLinkAccountAlreadyBound
		}

		if err := model.ClaimExternalIdentityWithTx(tx, model.ExternalIdentityProviderTelegram, payload.TelegramUserID, user.Id); err != nil {
			if errors.Is(err, model.ErrExternalIdentityAlreadyClaimed) {
				return model.ErrTelegramLinkAlreadyBound
			}
			return err
		}

		result := tx.Model(&model.User{}).
			Where("id = ? AND status = ? AND auth_version = ? AND telegram_id = ?", user.Id, common.UserStatusEnabled, user.AuthVersion, "").
			Updates(map[string]interface{}{
				"telegram_id":       payload.TelegramUserID,
				"telegram_username": payload.TelegramUsername,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errTelegramLinkAccountAlreadyBound
		}

		linkedTelegramID = payload.TelegramUserID
		linkedUsername = user.Username
		return nil
	})
	if err != nil {
		respondTelegramLinkRedemptionError(c, identity.UserID, err)
		return
	}

	// Emitted only after the binding transaction above has committed, so a
	// slow or unreachable bot callback can never delay or affect the bind.
	service.EmitTelegramLinkedEvent(linkedTelegramID, linkedUsername)
	common.ApiSuccessI18n(c, i18n.MsgTelegramLinkSuccess, nil)
}

// respondTelegramLinkSessionRefusal reports the pre-check session failure,
// distinguishing a disabled/deleted account from a merely revoked session,
// and records an operator-visible entry naming the reason and the account —
// never the submitted code.
func respondTelegramLinkSessionRefusal(c *gin.Context, userId int) {
	var user model.User
	userErr := model.DB.First(&user, userId).Error
	switch {
	case errors.Is(userErr, gorm.ErrRecordNotFound):
		logTelegramLinkRefusal("account_unusable", userId, "")
		common.ApiErrorI18n(c, i18n.MsgTelegramLinkAccountUnusable)
	case userErr != nil:
		common.SysError("TelegramLinkRedeem user status lookup failed: " + userErr.Error())
		common.ApiErrorI18n(c, i18n.MsgTelegramLinkInternalError)
	case user.Status != common.UserStatusEnabled:
		logTelegramLinkRefusal("account_unusable", userId, "")
		common.ApiErrorI18n(c, i18n.MsgTelegramLinkAccountUnusable)
	default:
		logTelegramLinkRefusal("session_invalid", userId, "")
		common.ApiErrorI18n(c, i18n.MsgTelegramLinkSessionInvalid)
	}
}

// respondTelegramLinkRedemptionError maps a ConsumeAuthFlowWithAction failure
// to its response. Unknown, expired, and already-consumed codes share one
// message so they cannot be told apart by the caller
// (specs/telegram/account-link/spec.md, "Link codes are single-use and
// expire"); every binding-safety guard keeps its own distinct message,
// mirroring TelegramBind.
func respondTelegramLinkRedemptionError(c *gin.Context, userId int, err error) {
	switch {
	case errors.Is(err, model.ErrTelegramLinkAlreadyBound):
		logTelegramLinkRefusal("identity_already_bound", userId, "")
		common.ApiErrorI18n(c, i18n.MsgTelegramLinkAlreadyBound)
	case errors.Is(err, errTelegramLinkAccountAlreadyBound):
		logTelegramLinkRefusal("account_already_bound", userId, "")
		common.ApiErrorI18n(c, i18n.MsgTelegramLinkAccountAlreadyBound)
	case errors.Is(err, errTelegramLinkAccountUnusable):
		logTelegramLinkRefusal("account_unusable", userId, "")
		common.ApiErrorI18n(c, i18n.MsgTelegramLinkAccountUnusable)
	case errors.Is(err, service.ErrLoginSessionRevoked):
		logTelegramLinkRefusal("session_invalid", userId, "")
		common.ApiErrorI18n(c, i18n.MsgTelegramLinkSessionInvalid)
	case errors.Is(err, model.ErrAuthFlowInvalid), errors.Is(err, model.ErrAuthFlowExpired), errors.Is(err, model.ErrAuthFlowConsumed):
		common.ApiErrorI18n(c, i18n.MsgTelegramLinkCodeInvalid)
	default:
		common.SysError("TelegramLinkRedeem failed: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgTelegramLinkInternalError)
	}
}

// logTelegramLinkRefusal records an operator-visible entry naming the
// refusal reason, the account, and (when known) the Telegram identity. It
// never receives a code, token, key, or password
// (specs/auth/account-binding-safety/spec.md, "Binding refusals are recorded
// for the operator").
func logTelegramLinkRefusal(reason string, userId int, telegramUserID string) {
	common.SysLog(fmt.Sprintf(
		"telegram link redemption refused reason=%s account=%d telegram_user_id=%s",
		reason, userId, telegramUserID,
	))
}
