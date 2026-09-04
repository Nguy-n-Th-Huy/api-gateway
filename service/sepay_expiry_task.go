package service

import (
	"context"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

const sePayExpirySweepInterval = 1 * time.Minute

// SePayExpiryHandler is a scheduled system-task handler that expires pending
// SePay wallet top-up and subscription orders whose derived deadline
// (create_time + configured expiry window) is in the past. It deliberately
// scopes to payment_provider = sepay so legacy pending orders from removed
// gateways stay visible for manual settlement (D4).
// The webhook path already checks the derived deadline directly so a transfer
// arriving one second after expiry is never credited even if the sweep hasn't
// run yet; this task materializes the expired status so the UI and admin views
// agree with the webhook.
type SePayExpiryHandler struct{}

func (h SePayExpiryHandler) Type() string { return "sepay_expiry" }

func (h SePayExpiryHandler) Enabled() bool { return true }

func (h SePayExpiryHandler) Interval() time.Duration { return sePayExpirySweepInterval }

// NewPayload is nil: the expiry sweep uses the SePayOrderExpiryMinutes
// setting at run time rather than baking parameters into the task payload.
func (h SePayExpiryHandler) NewPayload() any { return nil }

func (h SePayExpiryHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	for {
		select {
		case <-ctx.Done():
			_ = model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusSucceeded, nil, "")
			return
		default:
		}

		var topUps, subOrders int64
		var topUpErr, subErr error
		topUps, topUpErr = model.ExpireSePayTopUpsBulk(300)
		subOrders, subErr = model.ExpireSePaySubscriptionOrdersBulk(300)
		if topUpErr != nil || subErr != nil {
			errMsg := ""
			if topUpErr != nil {
				errMsg += fmt.Sprintf("topup: %v", topUpErr)
			}
			if subErr != nil {
				if errMsg != "" {
					errMsg += "; "
				}
				errMsg += fmt.Sprintf("subscription: %v", subErr)
			}
			logger.LogWarn(ctx, fmt.Sprintf("sepay expiry sweep failed: %s", errMsg))
			_ = model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusFailed, nil, errMsg)
			return
		}
		if common.DebugEnabled && (topUps > 0 || subOrders > 0) {
			logger.LogDebug(ctx, "sepay expiry sweep: topups=%d subscription_orders=%d", topUps, subOrders)
		}
		if topUps == 0 && subOrders == 0 {
			break
		}
		if topUps < 300 && subOrders < 300 {
			break
		}
		select {
		case <-ctx.Done():
			_ = model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusSucceeded, nil, "")
			return
		case <-time.After(200 * time.Millisecond):
		}
	}
	_ = model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusSucceeded, nil, "")
}

func init() {
	RegisterSystemTaskHandler(SePayExpiryHandler{})
}
