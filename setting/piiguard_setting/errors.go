package piiguard_setting

import "errors"

var (
	// errNothingToDo rejects an enabled guard that neither masks the request
	// nor restores the response, which would only cost CPU.
	errNothingToDo = errors.New("piiguard: enabled with neither request masking nor response unmasking")
	// errRedactCannotUnmask rejects the contradictory combination of one-way
	// redaction with response restoration.
	errRedactCannotUnmask = errors.New("piiguard: redact mode cannot be combined with unmask_response")
	// errEmptyKeyword rejects a blank custom keyword.
	errEmptyKeyword = errors.New("piiguard: custom keywords must not be empty")
	// errNotDecodable rejects an option whose value cannot be read as the guard
	// field it belongs to.
	errNotDecodable = errors.New("piiguard: option value cannot be decoded")
	// errUnknownField rejects an option key that is not part of the guard
	// configuration.
	errUnknownField = errors.New("piiguard: unknown configuration field")
)
