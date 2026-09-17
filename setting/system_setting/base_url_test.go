package system_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeServerAddressCompletesMissingScheme(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected string
	}{
		{"bare public host gets https", "browzyapi.augmentdev.online", "https://browzyapi.augmentdev.online"},
		{"bare host with port gets https", "gateway.example.com:8443", "https://gateway.example.com:8443"},
		{"bare private address gets https", "192.168.1.5:3000", "https://192.168.1.5:3000"},
		{"localhost gets http", "localhost:3000", "http://localhost:3000"},
		{"loopback address gets http", "127.0.0.1:3000", "http://127.0.0.1:3000"},
		{"ipv6 loopback gets http", "[::1]:3000", "http://[::1]:3000"},
		{"unspecified address gets http", "0.0.0.0:3000", "http://0.0.0.0:3000"},
		{"trailing slashes are trimmed", "https://example.com/api/", "https://example.com/api"},
		{"surrounding whitespace is trimmed", "  example.com  ", "https://example.com"},
		{"empty stays empty", "", ""},
		{"whitespace only stays empty", "   ", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, NormalizeServerAddress(test.raw))
		})
	}
}

func TestNormalizeServerAddressKeepsAnExistingScheme(t *testing.T) {
	for _, raw := range []string{
		"https://example.com",
		"http://example.com",
		"https://example.com:8443/gateway",
		"HTTPS://example.com",
	} {
		t.Run(raw, func(t *testing.T) {
			assert.Equal(t, raw, NormalizeServerAddress(raw))
		})
	}
}
