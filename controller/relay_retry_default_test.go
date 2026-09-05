package controller

import (
	"errors"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
)

// TestShouldRetryHasBudgetByDefault protects the channel-failover fix: the
// out-of-the-box RetryTimes default must leave retry budget available for a
// generic upstream error on the first attempt, where the previous disabled
// default (0) made shouldRetry return false via its retryTimes<=0 guard and
// left a dead channel permanently in the request path.
func TestShouldRetryHasBudgetByDefault(t *testing.T) {
	assert.Greater(t, common.RetryTimes, 0, "RetryTimes must default to a positive value so failover works out of the box")

	openaiErr := types.NewOpenAIError(errors.New("upstream"), types.ErrorCodeBadResponseStatusCode, http.StatusInternalServerError)
	c := newPinRetryContext()

	// Mirrors controller/relay.go: on the first attempt retryParam.GetRetry()
	// is 0, so the remaining budget passed to shouldRetry is common.RetryTimes-0.
	remainingRetries := common.RetryTimes
	assert.True(t, shouldRetry(c, openaiErr, remainingRetries), "a channel error must be retryable when the default retry budget is available")

	// Same error, exhausted budget: must not retry. This is the behavior the
	// old RetryTimes=0 default produced unconditionally.
	assert.False(t, shouldRetry(c, openaiErr, 0), "a channel error must not be retried once the retry budget is exhausted")
}
