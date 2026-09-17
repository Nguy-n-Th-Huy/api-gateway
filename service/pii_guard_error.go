package service

import "errors"

var (
	// errPIIBodyTooLarge is returned when the operator asked the guard to fail
	// closed and the outbound body exceeded the configured mask limit. Failing
	// closed is what turns a silent leak into a visible error.
	errPIIBodyTooLarge = errors.New("piiguard: request body exceeds the configured PII mask limit")
	// errPIIBodyUnparsable is returned when the operator asked the guard to fail
	// closed and the outbound body could not be parsed for masking.
	errPIIBodyUnparsable = errors.New("piiguard: request body could not be parsed for PII masking")
)
