package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/piiguard_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitOptionMapExposesPIIGuardSettings(t *testing.T) {
	previous := common.OptionMap
	t.Cleanup(func() { common.OptionMap = previous })
	InitOptionMap()

	common.OptionMapRWMutex.RLock()
	value, exists := common.OptionMap["piiguard.enabled"]
	common.OptionMapRWMutex.RUnlock()
	require.True(t, exists, "the guard must be exposed as an option key")
	assert.Equal(t, "false", value)

	require.NoError(t, UpdateOption("piiguard.enabled", "true"))
	assert.True(t, piiguard_setting.GetSettings().Enabled)
	require.NoError(t, UpdateOption("piiguard.enabled", "false"))

	require.Error(t, UpdateOption("piiguard.max_body_bytes", "not-a-number"))
}
