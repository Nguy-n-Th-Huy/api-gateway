package system_setting

import (
	"net"
	"net/url"
	"strings"
)

// NormalizeServerAddress completes an operator-configured server address into
// an absolute base URL.
//
// The address is free text in the admin settings form, so a bare host
// ("example.com", "192.168.1.5:3000") is a common entry. Every consumer builds
// absolute URLs from this value — the generated CLI setup scripts, OAuth
// callbacks, password-reset links, Midjourney image URLs, and the server
// address the frontend shows — so a scheme-less address silently produces
// unusable ones (a scheme-less curl target fails outright, and an
// ANTHROPIC_BASE_URL without a scheme is rejected by the CLI).
//
// A value that already names a scheme is only trimmed; a bare host is
// completed with https://, or http:// when it is loopback, where https is not
// reachable.
func NormalizeServerAddress(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if !strings.Contains(trimmed, "://") {
		host := trimmed
		if parsed, err := url.Parse("//" + trimmed); err == nil && parsed.Hostname() != "" {
			host = parsed.Hostname()
		}
		scheme := "https"
		if strings.EqualFold(host, "localhost") || host == "0.0.0.0" {
			scheme = "http"
		} else if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			scheme = "http"
		}
		trimmed = scheme + "://" + trimmed
	}
	return strings.TrimRight(trimmed, "/")
}