package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateOptionMapNormalizesServerAddress covers the wiring the setup
// scripts, the status endpoint and the admin form all read: whatever an
// operator types, the in-memory address and the option map hold an absolute
// base URL.
func TestUpdateOptionMapNormalizesServerAddress(t *testing.T) {
	previousAddress := system_setting.ServerAddress
	previousMap := common.OptionMap
	common.OptionMap = map[string]string{}
	t.Cleanup(func() {
		system_setting.ServerAddress = previousAddress
		common.OptionMap = previousMap
	})

	require.NoError(t, updateOptionMap("ServerAddress", "browzyapi.augmentdev.online"))

	assert.Equal(t, "https://browzyapi.augmentdev.online", system_setting.ServerAddress)
	assert.Equal(t, "https://browzyapi.augmentdev.online", common.OptionMap["ServerAddress"],
		"the option map is what the settings form and the public status endpoint read")

	require.NoError(t, updateOptionMap("ServerAddress", "http://localhost:3000/"))

	assert.Equal(t, "http://localhost:3000", system_setting.ServerAddress)
	assert.Equal(t, "http://localhost:3000", common.OptionMap["ServerAddress"])
}
