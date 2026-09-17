package piiguard_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/pkg/piiguard"
	"github.com/QuantumNous/new-api/setting/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigModuleRoundTripsThroughTheOptionSystem(t *testing.T) {
	original := GetSettings()
	t.Cleanup(func() {
		require.NoError(t, UpdateSettings(original))
	})

	enabled := original
	enabled.Enabled = true
	enabled.Mode = piiguard.ModePseudonym
	enabled.CustomKeywords = []string{"HOSP-9911"}
	require.NoError(t, UpdateSettings(enabled))

	// Persist exactly the way the option system does, then reload.
	saved := make(map[string]string)
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	require.Contains(t, saved, SettingsKeyPrefix+"enabled")

	require.NoError(t, config.GlobalConfig.LoadFromDB(saved))

	reloaded := GetSettings()
	assert.True(t, reloaded.Enabled)
	assert.Equal(t, piiguard.ModePseudonym, reloaded.Mode)
	assert.Equal(t, []string{"HOSP-9911"}, reloaded.CustomKeywords)
}

func TestValidateRejectsAnEnabledGuardWithNothingToDo(t *testing.T) {
	candidate := piiguard.DefaultConfig()
	candidate.Enabled = true
	candidate.MaskRequest = false
	candidate.UnmaskResponse = false

	assert.ErrorIs(t, Validate(candidate), errNothingToDo)
}

func TestValidateRejectsRedactionWithResponseUnmasking(t *testing.T) {
	candidate := piiguard.DefaultConfig()
	candidate.Enabled = true
	candidate.Mode = piiguard.ModeRedact
	candidate.UnmaskResponse = true

	assert.ErrorIs(t, Validate(candidate), errRedactCannotUnmask)
}

func TestValidateAcceptsTheShippedDefault(t *testing.T) {
	assert.NoError(t, Validate(piiguard.DefaultConfig()))
}

func TestValidateOptionValueRejectsAnUnreadableValue(t *testing.T) {
	assert.ErrorIs(t, ValidateOptionValue(SettingsKeyPrefix+"enabled", "{not json"), errNotDecodable)
}

func TestValidateOptionValueReadsAPlainStringField(t *testing.T) {
	assert.NoError(t, ValidateOptionValue(SettingsKeyPrefix+"mode", piiguard.ModePseudonym))
}

func TestValidateOptionValueRejectsAnEnabledGuardWithNoWork(t *testing.T) {
	err := ValidateOptionValue(SettingsKeyPrefix+"enabled", "true")

	require.NoError(t, err, "enabling alone keeps the default request masking on")
}
