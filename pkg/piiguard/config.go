// Package piiguard masks personally identifiable information in text that
// leaves this gateway towards an upstream AI provider, and reverses the
// substitution on the way back so the client still receives real data.
//
// The design follows two references:
//
//   - https://github.com/daslabhq/pii-proxy — bijective, deterministic
//     replacement plus a round-trip unmask, so a request never carries real
//     PII upstream while the caller still sees real values in the response.
//   - https://github.com/microsoft/presidio — layered detection: cheap
//     deterministic analyzers first (regex), each span resolving to one entity
//     type, with format-validating checks (Luhn) to keep precision high.
//
// Detection quality depends entirely on the enabled entity types, so the
// operator-facing defaults are conservative: high-precision pattern entities
// are on, broad patterns (national IDs of any country) are off.
package piiguard

// MaskMode selects what a detected span is replaced with.
const (
	// ModeRedact replaces the value with a stable typed placeholder such as
	// [EMAIL_1]. The mapping is not reversible and the model sees an opaque
	// token instead of a plausible value.
	ModeRedact = "redact"
	// ModePseudonym replaces the value with a deterministic fake of the same
	// shape, such as pii-6f3a2b91@example.com. The mapping is bijective, so the
	// same real value always maps to the same fake and the gateway can restore
	// the real value in the response.
	ModePseudonym = "pseudonym"
)

// PlaceholderStyle selects the placeholder shape used by ModeRedact.
const (
	// PlaceholderTyped renders [EMAIL_1] — short and stable for the model.
	PlaceholderTyped = "typed"
	// PlaceholderTemplate renders the TokenTemplate pattern, for example
	// «EMAIL_1», which is easier to spot in an audit trail.
	PlaceholderTemplate = "template"
)

// DefaultTokenTemplate is the placeholder shape of PlaceholderTemplate mode.
const DefaultTokenTemplate = "«{{type}}_{{index}}»"

// Entity types. The string value is also the placeholder type name, so these
// identifiers are part of the operator-visible contract.
const (
	EntityEmail        = "EMAIL"
	EntityPhone        = "PHONE"
	EntityCreditCard   = "CREDIT_CARD"
	EntityIPAddress    = "IP_ADDRESS"
	EntityURL          = "URL"
	EntityUUID         = "UUID"
	EntityIBAN         = "IBAN"
	EntitySSN          = "US_SSN"
	EntityPassport     = "PASSPORT"
	EntityDriverLic    = "DRIVER_LICENSE"
	EntityVNID         = "VN_ID"
	EntityVNTaxCode    = "VN_TAX_CODE"
	EntityDateOfBirth  = "DATE_OF_BIRTH"
	EntityPersonName   = "PERSON"
	EntityLocation     = "LOCATION"
	EntityOrganization = "ORGANIZATION"
	EntityCustom       = "CUSTOM"
)

// Config is the operator-facing configuration. It is registered as the
// "piiguard" module of setting/config, so every field is persisted as a
// "piiguard.<json key>" option and edited through the system settings API.
type Config struct {
	// Enabled turns the whole guard on. Disabled means outbound bodies are
	// forwarded byte for byte, exactly as before this feature existed.
	Enabled bool `json:"enabled"`
	// MaskRequest masks message text, prompts and instructions in the body
	// that is sent to the upstream provider.
	MaskRequest bool `json:"mask_request"`
	// UnmaskResponse restores masked values inside the upstream response, so
	// the client keeps seeing real data. It only applies to ModePseudonym,
	// because redaction is one-way.
	UnmaskResponse bool `json:"unmask_response"`
	// Mode is ModePseudonym or ModeRedact.
	Mode string `json:"mode"`
	// PlaceholderStyle is PlaceholderTyped or PlaceholderTemplate.
	PlaceholderStyle string `json:"placeholder_style"`
	// TokenTemplate is the placeholder pattern used by PlaceholderTemplate.
	TokenTemplate string `json:"token_template"`
	// Secret seeds the deterministic generator. Changing it changes every
	// generated fake. When empty a per-process random seed is used, which still
	// keeps a single request internally consistent.
	Secret string `json:"secret"`
	// MaxBodyBytes skips masking for bodies larger than this, bounding the
	// unmarshal cost of large multimodal payloads. 0 falls back to
	// DefaultMaxBodyBytes.
	MaxBodyBytes int `json:"max_body_bytes"`
	// EnabledEntityTypes limits detection to the listed entity types. Empty
	// means every type whose default is on.
	EnabledEntityTypes []string `json:"enabled_entity_types"`
	// DisabledEntityTypes removes entity types from the effective set, and wins
	// over EnabledEntityTypes.
	DisabledEntityTypes []string `json:"disabled_entity_types"`
	// CustomKeywords are admin-defined literals that are always masked as
	// EntityCustom. They cover regulated identifiers no general pattern knows.
	CustomKeywords []string `json:"custom_keywords"`
	// MinKeywordLength is the shortest custom keyword that is masked, to keep a
	// one-character keyword from shredding every message. 0 falls back to
	// DefaultMinKeywordLength.
	MinKeywordLength int `json:"min_keyword_length"`
	// Fakes overrides the generated fake for a specific real value, keyed by
	// entity type, for operators who need a compliant stand-in.
	Fakes map[string]map[string]string `json:"fakes"`
	// RequireMaskReject rejects a request whose masking could not be applied,
	// for example because the body exceeded MaxBodyBytes. It turns a silent
	// leak into a visible failure.
	RequireMaskReject bool `json:"require_mask_reject"`
}

// Defaults.
const (
	// DefaultMaxBodyBytes bounds the body the guard is willing to rewrite. Chat
	// requests are far smaller; the limit exists so a large base64 image or
	// audio payload is forwarded untouched instead of being unmarshalled into a
	// generic map.
	DefaultMaxBodyBytes = 1 << 20
	// DefaultMinKeywordLength keeps short custom keywords from matching inside
	// unrelated words.
	DefaultMinKeywordLength = 3
)

// DefaultConfig returns the shipped defaults: the guard exists but is off, and
// once switched on it pseudonymises request text and restores the response.
func DefaultConfig() Config {
	return Config{
		Enabled:          false,
		MaskRequest:      true,
		UnmaskResponse:   true,
		Mode:             ModePseudonym,
		PlaceholderStyle: PlaceholderTyped,
		TokenTemplate:    DefaultTokenTemplate,
		MaxBodyBytes:     DefaultMaxBodyBytes,
		MinKeywordLength: DefaultMinKeywordLength,
	}
}

// Normalize fills defaults and clamps invalid values so a stored option that
// predates a change, or an admin typo, cannot produce a half-configured guard.
func (c Config) Normalize() Config {
	if c.Mode != ModePseudonym && c.Mode != ModeRedact {
		c.Mode = ModePseudonym
	}
	if c.PlaceholderStyle != PlaceholderTyped && c.PlaceholderStyle != PlaceholderTemplate {
		c.PlaceholderStyle = PlaceholderTyped
	}
	if c.TokenTemplate == "" {
		c.TokenTemplate = DefaultTokenTemplate
	}
	if c.MaxBodyBytes <= 0 {
		c.MaxBodyBytes = DefaultMaxBodyBytes
	}
	if c.MinKeywordLength <= 0 {
		c.MinKeywordLength = DefaultMinKeywordLength
	}
	// Redaction is one-way, so a response cannot be restored.
	if c.Mode == ModeRedact {
		c.UnmaskResponse = false
	}
	if !c.Enabled {
		c.MaskRequest = false
	}
	return c
}

// Active reports whether the guard should rewrite an outbound body.
func (c Config) Active() bool {
	return c.Normalize().Enabled && c.MaskRequest
}

// entityEnabled resolves the three-way entity toggle (default, explicit enable,
// explicit disable) for one entity type.
func (c Config) entityEnabled(entityType string) bool {
	for _, disabled := range c.DisabledEntityTypes {
		if disabled == entityType {
			return false
		}
	}
	if len(c.EnabledEntityTypes) == 0 {
		return defaultEntityEnabled(entityType)
	}
	for _, enabled := range c.EnabledEntityTypes {
		if enabled == entityType {
			return true
		}
	}
	return false
}
