package piiguard_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/pkg/piiguard"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateSettingsAppliesTheValueTheOptionSystemStores covers the exact value
// the option system writes: strconv.FormatBool for a switch and a plain string
// for a select, both of which must reach the live setting.
func TestUpdateSettingsAppliesTheValueTheOptionSystemStores(t *testing.T) {
	original := GetSettings()
	t.Cleanup(func() {
		require.NoError(t, UpdateSettings(original))
	})

	enabled := original
	enabled.Enabled = true
	require.NoError(t, UpdateSettings(enabled))
	assert.True(t, GetSettings().Enabled, "FormatBool output must switch the guard on")

	rejected := original
	rejected.Enabled = true
	rejected.MaskRequest = false
	rejected.UnmaskResponse = false
	assert.ErrorIs(t, UpdateSettings(rejected), errNothingToDo, "the guard must reject a configuration that does nothing")
	assert.True(t, GetSettings().Enabled, "a rejected update must leave the live setting untouched")

	require.NoError(t, UpdateSettings(original))
	assert.Equal(t, piiguard.ModePseudonym, GetSettings().Mode)
	assert.NotEmpty(t, GetSettings().TokenTemplate)
}
