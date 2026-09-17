// Package piiguard_setting exposes the PII guard configuration to the option
// system, the admin API and the settings UI. The guard itself lives in
// pkg/piiguard; this package only owns the persisted options.
package piiguard_setting

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/pkg/piiguard"
	"github.com/QuantumNous/new-api/setting/config"
)

// SettingsKeyPrefix is the option prefix of every guard field. The prefix is
// what InitOptionMap and SaveToDB use to persist the configuration.
const SettingsKeyPrefix = "piiguard."

// defaultSettings holds the live configuration. It is registered with the
// shared config manager, so an admin update through the system settings API
// mutates this value in place.
var defaultSettings = piiguard.DefaultConfig()

func init() {
	config.GlobalConfig.Register("piiguard", &defaultSettings)
}

// GetSettings returns the live configuration. Callers must not mutate it; use
// UpdateSettings for a change that has to go through validation.
func GetSettings() piiguard.Config {
	return defaultSettings.Normalize()
}

// UpdateSettings validates and applies a new configuration, which the config
// manager then persists.
func UpdateSettings(next piiguard.Config) error {
	if err := Validate(next); err != nil {
		return err
	}
	defaultSettings = next.Normalize()
	return nil
}

// Validate rejects a configuration the guard cannot honour, so an admin cannot
// store a combination that silently disables masking.
func Validate(candidate piiguard.Config) error {
	normalized := candidate.Normalize()
	if normalized.Enabled && !normalized.MaskRequest && !normalized.UnmaskResponse {
		return errNothingToDo
	}
	if normalized.Mode == piiguard.ModeRedact && candidate.UnmaskResponse {
		return errRedactCannotUnmask
	}
	for _, keyword := range normalized.CustomKeywords {
		if strings.TrimSpace(keyword) == "" {
			return errEmptyKeyword
		}
	}
	return nil
}

// ValidateOptionValue checks one stored option before it is written, so a bad
// value is rejected at the API boundary instead of silently ignored while the
// database keeps it. The config manager skips a field it cannot decode, which
// would otherwise leave the persisted value and the live setting disagreeing.
//
// The candidate is built with the same decoder the manager uses, so validation
// and application can never disagree about how a value is read.
func ValidateOptionValue(key string, value string) error {
	field := strings.TrimPrefix(key, SettingsKeyPrefix)
	kind, known := configFieldKinds()[field]
	if !known {
		return errUnknownField
	}
	if err := checkOptionSyntax(kind, value); err != nil {
		return err
	}
	candidate := defaultSettings
	if err := config.UpdateConfigFromMap(&candidate, map[string]string{field: value}); err != nil {
		return errNotDecodable
	}
	return Validate(candidate)
}

// checkOptionSyntax rejects a value the shared decoder would silently skip,
// which is the case that would leave the database and the live setting
// disagreeing.
func checkOptionSyntax(kind reflect.Kind, value string) error {
	switch kind {
	case reflect.Bool:
		if _, err := strconv.ParseBool(value); err != nil {
			return errNotDecodable
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return errNotDecodable
		}
	}
	return nil
}

// configFieldKinds maps the JSON name of every guard option to its field kind,
// which is the same mapping the config manager builds by reflection.
func configFieldKinds() map[string]reflect.Kind {
	kinds := make(map[string]reflect.Kind)
	valueType := reflect.TypeOf(piiguard.Config{})
	for index := 0; index < valueType.NumField(); index++ {
		field := valueType.Field(index)
		if !field.IsExported() {
			continue
		}
		name := field.Tag.Get("json")
		if name == "" || name == "-" {
			name = field.Name
		}
		kinds[name] = field.Type.Kind()
	}
	return kinds
}
