package piiguard

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pseudonymConfig returns an engine configuration that masks deterministically
// without depending on randomness in the test.
func pseudonymConfig() Config {
	config := DefaultConfig()
	config.Enabled = true
	config.Secret = "test-secret"
	return config
}

func TestMaskTextReplacesEmailWithDeterministicPseudonym(t *testing.T) {
	engine := NewEngine(pseudonymConfig())

	masked := engine.MaskText("Contact alex@example.com about the order")

	require.NotContains(t, masked, "alex@example.com")
	assert.Contains(t, masked, "@example.com", "the domain carries meaning and must survive")
	assert.True(t, strings.HasPrefix(masked, "Contact pii-"), "got %q", masked)

	// The same value must map to the same fake, in this text and in the next.
	assert.Equal(t, masked, engine.MaskText("Contact alex@example.com about the order"))
	assert.Contains(t, engine.MaskText("Second message for alex@example.com"), masked[len("Contact "):len(masked)-len(" about the order")])
}

func TestMaskTextKeepsDistinctValuesDistinct(t *testing.T) {
	engine := NewEngine(pseudonymConfig())

	first := engine.MaskText("mail alex@example.com")
	second := engine.MaskText("mail sam@example.com")

	assert.NotEqual(t, first, second)
}

func TestMaskTextRedactionUsesOneStablePlaceholderPerType(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.Mode = ModeRedact
	engine := NewEngine(config)

	masked := engine.MaskText("mail alex@example.com then sam@example.com")

	assert.Equal(t, "mail [EMAIL_1] then [EMAIL_2]", masked)
	assert.Equal(t, "mail alex@example.com then sam@example.com", engine.UnmaskText(masked),
		"distinct placeholders keep redaction reversible")
}

func TestMaskTextRedactionTemplateRendersTypeAndIndex(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.Mode = ModeRedact
	config.PlaceholderStyle = PlaceholderTemplate
	engine := NewEngine(config)

	assert.Equal(t, "mail «EMAIL_1»", engine.MaskText("mail alex@example.com"))
}

func TestMaskTextGeneratesFormatValidReplacements(t *testing.T) {
	engine := NewEngine(pseudonymConfig())

	tests := []struct {
		name      string
		input     string
		entity    string
		assertion func(t *testing.T, fake string)
	}{
		{
			name:   "credit card keeps the Luhn checksum",
			input:  "card 4111 1111 1111 1111 on file",
			entity: EntityCreditCard,
			assertion: func(t *testing.T, fake string) {
				assert.True(t, luhnValid(creditCardSeparators.Replace(fake)), "fake %q must satisfy Luhn", fake)
			},
		},
		{
			name:   "ipv4 uses the benchmarking range",
			input:  "client 203.0.113.7 connected",
			entity: EntityIPAddress,
			assertion: func(t *testing.T, fake string) {
				assert.True(t, strings.HasPrefix(fake, "198.18."), "fake %q must use RFC 2544 space", fake)
			},
		},
		{
			name:   "uuid keeps its shape",
			input:  "session 3f2504e0-4f89-11d3-9a0c-0305e82c3301 ended",
			entity: EntityUUID,
			assertion: func(t *testing.T, fake string) {
				require.Len(t, fake, 36)
				assert.Equal(t, "-", string(fake[8]))
				assert.Equal(t, "4", string(fake[14]))
			},
		},
		{
			name:   "international phone keeps a dialable shape",
			input:  "call +1 415 555 2671 now",
			entity: EntityPhone,
			assertion: func(t *testing.T, fake string) {
				assert.True(t, strings.HasPrefix(fake, "+1 555 01"), "fake %q must use the reserved 555-01 range", fake)
			},
		},
		{
			name:   "passport keeps its shape",
			input:  "passport B1234567 issued",
			entity: EntityPassport,
			assertion: func(t *testing.T, fake string) {
				require.Len(t, fake, 8)
				assert.Equal(t, "P", string(fake[0]))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			masked := engine.MaskText(test.input)
			require.NotEqual(t, test.input, masked)
			replacements := replacementsOfType(engine, test.entity)
			require.Len(t, replacements, 1)
			test.assertion(t, replacements[0])
		})
	}
}

// replacementsOfType returns the fakes the engine generated for one type.
func replacementsOfType(engine *Engine, entityType string) []string {
	var fakes []string
	for _, mapping := range engine.Mappings() {
		if mapping.EntityType == entityType {
			fakes = append(fakes, mapping.Fake)
		}
	}
	return fakes
}

func TestMaskTextReportsEntityCountsPerOccurrence(t *testing.T) {
	engine := NewEngine(pseudonymConfig())

	engine.MaskText("mail alex@example.com, card 4111 1111 1111 1111, backup alex@example.com")

	stats := engine.Stats()
	assert.Equal(t, 3, stats.Spans, "both email occurrences plus the card")
	assert.Equal(t, 2, stats.ByEntityType[EntityEmail])
	assert.Equal(t, 1, stats.ByEntityType[EntityCreditCard])
}

func TestMaskTextLeavesOrdinaryProseAlone(t *testing.T) {
	engine := NewEngine(pseudonymConfig())

	const prose = "Summarise the quarterly report and list the three biggest risks."

	assert.Equal(t, prose, engine.MaskText(prose))
	assert.Zero(t, engine.Stats().Spans)
}

func TestMaskTextSkipsContextGatedIdentifiersWithoutContext(t *testing.T) {
	engine := NewEngine(pseudonymConfig())

	// A bare nine digit number is an order number, not a national ID.
	assert.Equal(t, "order 123456789 shipped", engine.MaskText("order 123456789 shipped"))
	assert.Zero(t, engine.Stats().Spans)
}

func TestMaskTextDetectsContextGatedIdentifiersWithContext(t *testing.T) {
	engine := NewEngine(pseudonymConfig())

	masked := engine.MaskText("CCCD 012345678901 was verified")

	assert.NotContains(t, masked, "012345678901")
	assert.Equal(t, 1, engine.Stats().ByEntityType[EntityVNID])
}

func TestMaskTextDetectsPersonNameOnlyWhenEnabled(t *testing.T) {
	off := NewEngine(pseudonymConfig())
	assert.Equal(t, "Mr. Nguyen Van An arrived", off.MaskText("Mr. Nguyen Van An arrived"))

	config := pseudonymConfig()
	config.EnabledEntityTypes = []string{EntityPersonName}
	on := NewEngine(config)
	masked := on.MaskText("Mr. Nguyen Van An arrived")

	assert.NotContains(t, masked, "Nguyen Van An")
	assert.Equal(t, 1, on.Stats().ByEntityType[EntityPersonName])
}

func TestMaskTextMasksCustomKeywords(t *testing.T) {
	config := pseudonymConfig()
	config.CustomKeywords = []string{"HOSP-9911"}
	engine := NewEngine(config)

	masked := engine.MaskText("record HOSP-9911 closed")

	assert.NotContains(t, masked, "HOSP-9911")
	assert.Equal(t, 1, engine.Stats().ByEntityType[EntityCustom])
}

func TestMaskTextDetectsIPv6WithoutEatingClockReadings(t *testing.T) {
	engine := NewEngine(pseudonymConfig())

	masked := engine.MaskText("peer 2001:db8::1 answered at 12:30:45")

	assert.NotContains(t, masked, "2001:db8::1")
	assert.Contains(t, masked, "12:30:45", "a clock reading is not an address")
	assert.Contains(t, masked, "2001:db8::", "the fake stays in the documentation range")
}

func TestMaskTextIgnoresCustomKeywordsBelowMinimumLength(t *testing.T) {
	config := pseudonymConfig()
	config.CustomKeywords = []string{"ab"}
	engine := NewEngine(config)

	assert.Equal(t, "ab is short", engine.MaskText("ab is short"))
	assert.Zero(t, engine.Stats().Spans)
}

func TestMaskTextPrefersTheLongestOverlappingDetection(t *testing.T) {
	config := pseudonymConfig()
	config.EnabledEntityTypes = []string{EntityEmail, EntityURL}
	engine := NewEngine(config)

	// The URL detector would otherwise claim the whole address including the
	// scheme, and the email detector the address inside it.
	masked := engine.MaskText("see https://portal.example.com/reset?user=alex@example.com now")

	assert.NotContains(t, masked, "alex@example.com")
	assert.Equal(t, 1, engine.Stats().Spans, "one span must win, not two overlapping ones")
}

func TestUnmaskTextRestoresEveryFake(t *testing.T) {
	engine := NewEngine(pseudonymConfig())
	const original = "mail alex@example.com, card 4111 1111 1111 1111, ip 203.0.113.7"

	masked := engine.MaskText(original)
	require.NotEqual(t, original, masked)

	assert.Equal(t, original, engine.UnmaskText(masked))
}

func TestUnmaskTextIsAnIdentityWithoutAssignments(t *testing.T) {
	engine := NewEngine(pseudonymConfig())

	assert.Equal(t, "nothing was masked", engine.UnmaskText("nothing was masked"))
	assert.Empty(t, engine.UnmaskTable())
}

func TestConfigNormalizeDisablesRequestMaskingWhenGuardIsOff(t *testing.T) {
	config := Config{Enabled: false, MaskRequest: true, Mode: "nonsense", MaxBodyBytes: -5}

	normalized := config.Normalize()

	assert.False(t, normalized.Active())
	assert.Equal(t, ModePseudonym, normalized.Mode)
	assert.Equal(t, DefaultMaxBodyBytes, normalized.MaxBodyBytes)
	assert.Equal(t, DefaultMinKeywordLength, normalized.MinKeywordLength)
}

func TestConfigNormalizeForcesRedactionToBeOneWay(t *testing.T) {
	config := Config{Enabled: true, MaskRequest: true, UnmaskResponse: true, Mode: ModeRedact}

	normalized := config.Normalize()

	assert.False(t, normalized.UnmaskResponse)
	assert.True(t, normalized.Active())
}
