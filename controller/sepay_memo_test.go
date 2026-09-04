package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Memo generation format (bank-safe prefix + uppercase A-Z0-9 tail) is covered
// in model/sepay_settlement_test.go against GenerateSePayTradeNo. The
// controller-level tests here exercise the webhook's content normalization and
// memo extraction (D2) and the Apikey header parser (D6).

func TestNormalizeSePayContentHandlesNoisyBankContent(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"lowercase uppercased", "hello", "HELLO"},
		{"punctuation becomes space", "hello,world!", "HELLO WORLD "},
		{"digits preserved", "abc 123", "ABC 123"},
		{"non-ASCII diacritics become space", "chuyển khoản SPABCDEF12345678", "CHUY N KHO N SPABCDEF12345678"},
		{"memo packed against text", "ctnd SPABCDEF12345678 chuyentien", "CTND SPABCDEF12345678 CHUYENTIEN"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, normalizeSePayContent(tc.content))
		})
	}
}

func TestExtractSePayMemosFromNoisyContent(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{"lowercase memo normalized", "spabcdef12345678", []string{"SPABCDEF12345678"}},
		{"punctuation around the memo", "chuyen khoan! SPABCDEF12345678.", []string{"SPABCDEF12345678"}},
		{"memo packed against adjacent text", "ctndSPABCDEF12345678chuyentien", []string{"SPABCDEF12345678"}},
		{"subscription prefix extracted", "ssabcdef12345678", []string{"SSABCDEF12345678"}},
		{"two distinct memos", "SPABCDEF12345678 SS12345678901234", []string{"SPABCDEF12345678", "SS12345678901234"}},
		{"duplicate candidates deduped", "SPABCDEF12345678 SPABCDEF12345678", []string{"SPABCDEF12345678"}},
		{"no memo shape", "hello world", nil},
		{"too-short candidate ignored", "SPABC SS12", nil},
		// A memo fragmented by punctuation does not rejoin: normalization maps
		// non-alphanumerics to spaces, so "sp-abcdef..." never matches. System
		// memos are pure A-Z0-9 and banks strip punctuation without inserting
		// spaces, so fragmented memos cannot occur for valid orders.
		{"punctuation inside memo does not match", "sp-abcdef-12345678", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractSePayMemos(tc.content)
			if tc.want == nil {
				assert.Empty(t, got, "from %q", tc.content)
			} else {
				assert.Equal(t, tc.want, got, "from %q", tc.content)
			}
		})
	}
}

func TestParseSePayApiKeyAuthRequiresCaseSensitiveApikeyScheme(t *testing.T) {
	cases := []struct {
		name   string
		header string
		ok     bool
		key    string
	}{
		{"valid Apikey scheme", "Apikey correct-key_123", true, "correct-key_123"},
		{"lowercase apikey scheme rejected", "apikey correct-key_123", false, ""},
		{"uppercase APIKEY rejected", "APIKEY correct-key_123", false, ""},
		{"Bearer scheme rejected", "Bearer correct-key_123", false, ""},
		{"scheme only no key rejected", "Apikey", false, ""},
		{"empty header rejected", "", false, ""},
		{"surrounding whitespace tolerated", "  Apikey   correct-key_123  ", true, "correct-key_123"},
		{"three fields rejected", "Apikey a b", false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key, ok := parseSePayApiKeyAuth(tc.header)
			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.key, key)
			}
		})
	}
}
