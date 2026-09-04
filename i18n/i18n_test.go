package i18n

import (
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// loadLocaleKeys reads a locale YAML file from disk and returns its message
// keys. Locale files are flat maps of dotted keys to translated strings.
func loadLocaleKeys(t *testing.T, path string) []string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoErrorf(t, err, "reading %s", path)

	var messages map[string]string
	require.NoErrorf(t, yaml.Unmarshal(data, &messages), "parsing %s", path)

	keys := make([]string, 0, len(messages))
	for key := range messages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// TestLocaleKeySetsMatch protects backend message coverage: every key in the
// English bundle must have a Vietnamese translation, so a merge that adds an
// English key fails here until the Vietnamese one is added.
func TestLocaleKeySetsMatch(t *testing.T) {
	enKeys := loadLocaleKeys(t, "locales/en.yaml")
	viKeys := loadLocaleKeys(t, "locales/vi.yaml")

	assert.NotEmpty(t, enKeys)
	assert.Equal(t, enKeys, viKeys, "vi.yaml must have exactly the same message keys as en.yaml")
}

func TestNormalizeLang(t *testing.T) {
	tests := []struct {
		name string
		lang string
		want string
	}{
		{name: "exact vi", lang: "vi", want: LangVi},
		{name: "vietnamese regional variant", lang: "vi-VN", want: LangVi},
		{name: "vietnamese uppercase", lang: "VI-VN", want: LangVi},
		{name: "exact en", lang: "en", want: LangEn},
		{name: "english regional variant", lang: "en-US", want: LangEn},
		{name: "unsupported language falls back to default", lang: "zh-CN", want: DefaultLang},
		{name: "unrelated language falls back to default", lang: "fr", want: DefaultLang},
		{name: "empty string falls back to default", lang: "", want: DefaultLang},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeLang(tt.lang))
		})
	}
}
